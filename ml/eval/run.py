"""The evaluation harness, `uv run python -m eval.run` (evaluation.md section 10).

A number in a report comes from here and from nowhere else. A run refuses a dirty tree, seeds everything with 2026 and
writes config, metrics, per query rows, slices and the environment to ml/eval/runs/<run-id>/.
"""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import platform
import random
import subprocess
import sys
from collections import defaultdict
from collections.abc import Callable
from dataclasses import dataclass
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import numpy as np

from app.backend import Backend
from app.registry import Registry
from eval import calibrate, components, scenario, text
from eval.dataset import Entry as DatasetEntry
from eval.dataset import leaf_parents, load
from eval.engine import Embedder, Engine, Query, Ranked, Variant, load_compat, rank, rank_of_true
from eval.fusion import Config
from eval.gates import evaluate
from eval.metrics import bootstrap_ci, mrr, recall_at_k, summary, tier_precision, tier_recall

ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
SEED = 2026
MIN_SLICE_ITEMS = 10


def variants(cfg: Config) -> dict[str, Variant]:
    loc = cfg.raw["location"]
    time = cfg.raw["time"]
    return {
        "B0": Variant("B0", cfg, random_order=True, floor=False),
        "B1": Variant("B1", cfg, visual=False, text=False, attributes=False),
        "B2": Variant("B2", cfg, visual=False, tfidf=True, attributes=False),
        "B3": Variant("B3", cfg),
        "A1": Variant("A1", cfg, visual=False),
        "A2": Variant("A2", cfg, text=False),
        "A3": Variant("A3", cfg, location=False),
        "A4": Variant("A4", cfg, time=False),
        "A5": Variant("A5", cfg, hard_category=True),
        "A6a": Variant("A6a", cfg.with_changes(location={**loc, "scale_m": 200})),
        "A6b": Variant("A6b", cfg.with_changes(location={**loc, "scale_m": 800})),
        "A6c": Variant("A6c", cfg.with_changes(time={**time, "half_life_h": 24})),
        "A6d": Variant("A6d", cfg.with_changes(time={**time, "half_life_h": 168})),
    }


@dataclass
class QueryResult:
    item: DatasetEntry
    flags: list[str]
    rank: int | None
    true_overall: float | None
    true_tier: str | None
    pool: int
    ranked: list[Ranked]
    lost_photo: bool

    def row(self) -> dict[str, Any]:
        return {
            "item_id": self.item.id,
            "rank": self.rank or "",
            "true_overall": "" if self.true_overall is None else self.true_overall,
            "true_tier": self.true_tier or "",
            "pool": self.pool,
            "lost_photo": self.lost_photo,
            "parent": self.item.parent,
            "distinctiveness": self.item.distinctiveness,
            "noise": "+".join(self.flags) or "clean",
        }


def evaluate_variant(
    variant: Variant, queries: list[Query], parents: dict[str, str], found_leaf: dict[str, str]
) -> list[QueryResult]:
    out = []
    for q in queries:
        ranked = rank(variant, q, parents.__getitem__, found_leaf)
        r = rank_of_true(ranked)
        true = next((x for x in ranked if x.true_pair), None)
        out.append(
            QueryResult(
                q.item,
                q.meta.flags,
                r,
                true.overall if true else None,
                true.tier if true else None,
                len(q.pool),
                ranked,
                q.item.has_owner_photo,
            )
        )
    return out


def slices(results: list[QueryResult]) -> dict[str, list[QueryResult]]:
    groups: dict[str, list[QueryResult]] = defaultdict(list)
    for r in results:
        groups[f"lost_photo={'with' if r.lost_photo else 'without'}"].append(r)
        groups[f"distinctiveness={r.item.distinctiveness}"].append(r)
        groups[f"noise={'+'.join(r.flags) or 'clean'}"].append(r)
        for flag in r.flags:
            groups[f"noise_any={flag}"].append(r)
        groups[f"parent={r.item.parent}"].append(r)
    counts: dict[str, int] = defaultdict(int)
    for r in results:
        counts[r.item.parent] += 1
    return {
        k: v
        for k, v in sorted(groups.items())
        if not k.startswith("parent=") or counts[k.split("=", 1)[1]] >= MIN_SLICE_ITEMS
    }


def metrics_of(results: list[QueryResult]) -> dict[str, Any]:
    ranks = [r.rank for r in results]
    out: dict[str, Any] = summary(ranks)
    out["recall@5_ci95"] = bootstrap_ci(ranks, lambda x: recall_at_k(x, 5))
    out["mrr_ci95"] = bootstrap_ci(ranks, mrr)
    top = [(c.tier, c.true_pair) for r in results for c in r.ranked[:10]]
    tiers, truth = [t for t, _ in top], [v for _, v in top]
    out["tier_precision"] = {t: tier_precision(tiers, truth, t) for t in ("high", "medium", "low")}
    out["tier_recall"] = {t: tier_recall(tiers, truth, t, len(results)) for t in ("high", "medium")}
    return out


def git_state() -> tuple[str, bool]:
    def run(*args: str) -> str:
        return subprocess.run(["git", *args], cwd=ROOT, capture_output=True, text=True, check=False).stdout.strip()  # noqa: S603, S607

    return run("rev-parse", "HEAD") or "unknown", bool(
        run("status", "--porcelain", "--", "ml", "data/seed", "api/internal/matching")
    )


def seed_everything() -> None:
    random.seed(SEED)
    np.random.seed(SEED)
    try:
        import torch

        torch.manual_seed(SEED)
        torch.use_deterministic_algorithms(True, warn_only=True)
    except ImportError:  # the harness also runs without PyTorch against a stand in backend
        pass


def write_csv(path: Path, rows: list[dict[str, Any]]) -> None:
    if not rows:
        path.write_text("", encoding="utf-8")
        return
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0]))
        writer.writeheader()
        writer.writerows(rows)


def run(
    dataset_dir: Path,
    split: str,
    out_root: Path,
    *,
    backend: Backend,
    lock: Path,
    config_path: Path | None = None,
    locations_csv: Path | None = None,
    cache_dir: Path | None = None,
    allow_dirty: bool = False,
    only: list[str] | None = None,
    run_id: str | None = None,
    components_too: bool = True,
    calibrate_as: str | None = None,
    config_dir: Path | None = None,
) -> Path:
    if calibrate_as and split != "val":
        raise SystemExit("calibration reads val only, the test split is touched at gates")
    commit, dirty = git_state()
    if dirty and not allow_dirty:
        raise SystemExit(
            "uncommitted changes in ml, data/seed or api/internal/matching: commit them or pass --allow-dirty"
        )
    seed_everything()
    cfg = Config.load(config_path) if config_path else Config.load()
    registry = Registry.from_lock(lock)
    dataset = load(dataset_dir, SEED_DIR)
    parents = leaf_parents(SEED_DIR)
    locations = scenario.load_locations(locations_csv or ROOT / "deploy" / "dev-seed" / "locations.csv")
    synonyms = text.load_synonyms(SEED_DIR / "synonyms.csv")
    compat_pairs = load_compat(SEED_DIR / "category_compatibility.csv")
    compatible: dict[str, list[str]] = defaultdict(list)
    for key in compat_pairs:
        a, b = key.split("|")
        compatible[a].append(b)
        compatible[b].append(a)
    items = {e.id: e.leaf for e in dataset.entries.values() if e.is_item}
    distractors = [e.id for e in dataset.entries.values() if not e.is_item]
    generated = scenario.Generator(locations, parents, compatible).generate(items, distractors)
    identity = "|".join(f"{n}={c.model_id}@{c.revision}" for n, c in sorted(registry.components.items()))
    embedder = Embedder(backend, identity, cache_dir)
    engine = Engine(dataset, generated, {loc.slug: loc for loc in locations}, embedder, parents, compat_pairs, synonyms)
    queries = engine.build(split, cfg)
    found_leaf = {e.id: e.leaf for e in dataset.split(split)}
    if calibrate_as:
        cfg = calibrate.calibrate(queries, cfg, parents.__getitem__, found_leaf, calibrate_as)
        written = calibrate.write(cfg, config_dir or Path(__file__).resolve().parent / "configs")
        print(f"calibrated fusion document written to {written}")
        queries = engine.build(split, cfg)
    chosen = variants(cfg)
    if only:
        chosen = {k: v for k, v in chosen.items() if k in only}
    # A7 needs the texts embedded again without the slang table
    if not only or "A7" in only:
        no_syn = Engine(dataset, generated, {loc.slug: loc for loc in locations}, embedder, parents, compat_pairs, None)
        chosen["A7"] = Variant("A7", cfg)
        alt_queries = no_syn.build(split, cfg)
    all_metrics: dict[str, Any] = {}
    slice_rows: list[dict[str, Any]] = []
    per_query: list[dict[str, Any]] = []
    for name, variant in chosen.items():
        results = evaluate_variant(variant, alt_queries if name == "A7" else queries, parents, found_leaf)
        all_metrics[name] = {"overall": metrics_of(results), "slices": {}}
        for label, members in slices(results).items():
            m = summary([r.rank for r in members])
            all_metrics[name]["slices"][label] = m
            slice_rows.append({"variant": name, "slice": label, **m})
        per_query += [{"variant": name, **r.row()} for r in results]
    if components_too:
        all_metrics["components"] = {
            **components.t4_t5_t6(dataset, split, backend, SEED_DIR),
            **components.t7(dataset, embedder, synonyms),
            **components.t8(dataset, embedder, synonyms),
        }
    all_metrics["gates"] = evaluate(all_metrics) if "B3" in all_metrics else []
    run_name = run_id or f"{datetime.now(UTC):%Y%m%dT%H%M%SZ}-{split}-{commit[:8]}"
    out = out_root / run_name
    out.mkdir(parents=True, exist_ok=False)
    (out / "metrics.json").write_text(json.dumps(all_metrics, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    write_csv(out / "per_query.csv", per_query)
    write_csv(out / "slices.csv", slice_rows)
    (out / "config.yaml").write_text(
        json.dumps(
            {
                "split": split,
                "fusion": cfg.raw,
                "fusion_sha256": hashlib.sha256(json.dumps(cfg.raw, sort_keys=True).encode()).hexdigest(),
                "scenario": scenario.manifest_entry(),
                "variants": sorted(chosen),
                "pool_sizes": {q.item.id: len(q.pool) for q in queries},
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    (out / "environment.json").write_text(
        json.dumps(
            {
                "git_commit": commit,
                "dirty": dirty,
                "dataset_version": dataset.version,
                "models": {
                    n: {"model_id": c.model_id, "revision": c.revision} for n, c in sorted(registry.components.items())
                },
                "python": sys.version.split()[0],
                "platform": platform.platform(),
                "seed": SEED,
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    return out


def main(argv: list[str] | None = None, backend_factory: Callable[[Registry, Path], Backend] | None = None) -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument("--split", choices=["val", "test"], default="val")
    parser.add_argument("--out", type=Path, default=Path(__file__).resolve().parent / "runs")
    parser.add_argument("--lock", type=Path, default=ROOT / "ml" / "models.lock.json")
    parser.add_argument("--config", type=Path)
    parser.add_argument("--locations", type=Path)
    parser.add_argument("--cache", type=Path)
    parser.add_argument("--hf-home", type=Path, default=Path.home() / ".cache" / "huggingface")
    parser.add_argument("--only", nargs="*")
    parser.add_argument("--allow-dirty", action="store_true")
    parser.add_argument("--calibrate", metavar="NEW_FUSION_ID", help="calibrate on val and write a new fusion document")
    args = parser.parse_args(argv)
    registry = Registry.from_lock(args.lock)
    if backend_factory:
        backend = backend_factory(registry, args.hf_home)
    else:
        from app.backend import TorchBackend

        backend = TorchBackend(registry, args.hf_home, threads=4)
    backend.load()
    out = run(
        args.dataset,
        args.split,
        args.out,
        backend=backend,
        lock=args.lock,
        config_path=args.config,
        locations_csv=args.locations,
        cache_dir=args.cache,
        allow_dirty=args.allow_dirty,
        only=args.only,
        calibrate_as=args.calibrate,
    )
    print(out)


if __name__ == "__main__":
    main()
