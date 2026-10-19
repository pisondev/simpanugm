"""Real evaluation runner with freeze verification and run artifact packaging.

This script executes the official evaluation protocol (docs/ai/evaluation.md):
1. Verifies that the dataset has passed check_freeze validation.
2. Calibrates fusion weights and confidence tiers on the val split.
3. Evaluates all baseline, ablation, and component gates on the test split.
4. Generates a packaged run directory containing metrics, per-query logs, slices,
   configuration, environment metadata, and an authoritative REPORT.md labeled
   RESMI (for frozen real data) or SINTETIS (for synthetic/trial data).
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import sys
from collections import defaultdict
from collections.abc import Callable
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from app.backend import Backend
from app.registry import Registry
from eval import calibrate, components, scenario, text
from eval.dataset import leaf_parents, load
from eval.engine import Embedder, Engine, Variant, load_compat
from eval.fusion import Config
from eval.gates import evaluate
from eval.metrics import summary
from eval.run import (
    ROOT,
    SEED,
    SEED_DIR,
    evaluate_variant,
    git_state,
    metrics_of,
    seed_everything,
    slices,
    variants,
    write_csv,
)


def verify_freeze_report(report_path: Path) -> tuple[bool, bool, str, dict[str, Any]]:
    """Validate the freeze report and determine whether the run is official (RESMI) or synthetic (SINTETIS).

    Returns (is_valid, is_official, message, data).
    """
    if not report_path.is_file():
        msg = (
            f"missing check_freeze report at {report_path}. "
            "Dataset must pass check_freeze before running real evaluation."
        )
        return False, False, msg, {}

    try:
        data = json.loads(report_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        msg = f"cannot read freeze report at {report_path}: {exc}"
        return False, False, msg, {}

    passed = bool(data.get("passed", False) or data.get("status") == "passed")
    errors = data.get("errors", [])
    if not passed or errors:
        msg = f"dataset freeze verification failed: {', '.join(errors) if errors else 'passed=False'}"
        return False, False, msg, data

    profile = data.get("profile", "")
    is_synth = bool(data.get("is_synthetic", True))
    item_count = int(data.get("item_count", 0))

    is_official = profile == "full" and not is_synth and item_count >= 150
    return True, is_official, "Freeze verification passed", data


def build_markdown_report(
    run_id: str,
    timestamp: str,
    is_official: bool,
    git_commit: str,
    dirty: bool,
    dataset_version: str,
    freeze_data: dict[str, Any],
    calibrated_cfg: Config,
    all_metrics: dict[str, Any],
    gates: list[dict[str, Any]],
    registry: Registry,
) -> str:
    status_label = "RESMI" if is_official else "SINTETIS"
    b3 = all_metrics.get("B3", {})
    overall = b3.get("overall", {})

    lines = [
        f"# Evaluation Report | SimpanUGM (Run {run_id})",
        "",
    ]

    if is_official:
        lines += [
            "> [!IMPORTANT]",
            "> **STATUS: RESMI**",
            "> This evaluation was executed on the official frozen dataset (simpanugm-eval v1)",
            "> with verified freeze checks. All acceptance gate decisions are official.",
            "",
        ]
    else:
        lines += [
            "> [!WARNING]",
            "> **STATUS: SINTETIS**",
            "> This evaluation was executed on synthetic or trial data. These numbers are",
            "> illustrative and do not reflect model accuracy on real campus data.",
            "",
        ]

    lines += [
        "## 1. Run Metadata",
        "",
        "| Attribute | Value |",
        "|---|---|",
        f"| Run ID | `{run_id}` |",
        f"| Date (UTC) | `{timestamp}` |",
        f"| Evaluation Status | **{status_label}** |",
        f"| Git Commit | `{git_commit}`{' (dirty)' if dirty else ''} |",
        f"| Dataset Version | `{dataset_version}` |",
        f"| Freeze Profile | `{freeze_data.get('profile', 'unknown')}` |",
        f"| Item Count | {freeze_data.get('item_count', 'unknown')} |",
        f"| Random Seed | `{SEED}` |",
        f"| Fusion ID | `{calibrated_cfg.id}` |",
        "",
        "### Models",
        "",
        "| Component | Model ID | Revision |",
        "|---|---|---|",
    ]
    for comp_name, comp in sorted(registry.components.items()):
        lines.append(f"| `{comp_name}` | `{comp.model_id}` | `{comp.revision[:16]}...` |")

    lines += [
        "",
        "## 2. Calibrated Fusion Parameters (from val split)",
        "",
        "### Component Weights",
        "",
        "| Weight Component | Value |",
        "|---|---|",
    ]
    for k, v in sorted(calibrated_cfg.raw.get("weights", {}).items()):
        lines.append(f"| `{k}` | {v:.4f} |")

    tiers = calibrated_cfg.raw.get("tiers", {})
    lines += [
        "",
        "### Confidence Tiers",
        "",
        "| Tier | Score Range | Action |",
        "|---|---|---|",
        f"| `high` | >= {tiers.get('high', 0.72):.2f} | Owner notification and high priority queue |",
        f"| `medium` | {tiers.get('medium', 0.55):.2f} to {tiers.get('high', 0.72):.2f} | Digest listing |",
        f"| `low` | {tiers.get('floor', 0.40):.2f} to {tiers.get('medium', 0.55):.2f} | Manual search queue |",
        f"| `not_stored` | < {tiers.get('floor', 0.40):.2f} | Dropped |",
        "",
        "## 3. Acceptance Gate Results (test split)",
        "",
        "| Gate | Threshold | Test Value | Gate Status |",
        "|---|---|---|---|",
    ]

    all_gates_pass = True
    for g in gates:
        passed = g.get("passed", False)
        if not passed:
            all_gates_pass = False
        val_str = str(g.get("value")) if g.get("value") is not None else "null"
        th_str = str(g.get("threshold"))
        status_str = f"PASS ({status_label})" if passed else f"FAIL ({status_label})"
        lines.append(f"| {g.get('gate')} | {th_str} | {val_str} | {status_str} |")

    lines += [
        "",
        "## 4. Primary Variant Metrics (B3 Full Engine on test split)",
        "",
        "| Metric | Overall Score |",
        "|---|---|",
        f"| Recall@1 | {overall.get('recall@1', 0.0):.4f} |",
        f"| Recall@5 | {overall.get('recall@5', 0.0):.4f} |",
        f"| Recall@10 | {overall.get('recall@10', 0.0):.4f} |",
        f"| MRR | {overall.get('mrr', 0.0):.4f} |",
        f"| nDCG@10 | {overall.get('ndcg@10', 0.0):.4f} |",
    ]
    high_prec = overall.get("tier_precision", {}).get("high")
    high_prec_str = f"{high_prec:.4f}" if high_prec is not None else "null"
    lines.append(f"| High Tier Precision | {high_prec_str} |")

    lines += [
        "",
        "## 5. Variant Comparison on test split",
        "",
        "| Variant | Recall@1 | Recall@5 | Recall@10 | MRR | nDCG@10 |",
        "|---|---|---|---|---|---|",
    ]
    for v_name, v_data in sorted(all_metrics.items()):
        if v_name in {"components", "gates"}:
            continue
        v_over = v_data.get("overall", {})
        r1 = v_over.get("recall@1", 0.0)
        r5 = v_over.get("recall@5", 0.0)
        r10 = v_over.get("recall@10", 0.0)
        vmrr = v_over.get("mrr", 0.0)
        vndcg = v_over.get("ndcg@10", 0.0)
        lines.append(f"| {v_name} | {r1:.4f} | {r5:.4f} | {r10:.4f} | {vmrr:.4f} | {vndcg:.4f} |")

    lines += [
        "",
        "## 6. Slices Breakdown (B3 on test split)",
        "",
        "| Slice | Count (n) | Recall@1 | Recall@5 | Recall@10 | MRR |",
        "|---|---|---|---|---|---|",
    ]
    b3_slices = b3.get("slices", {})
    for s_name, s_vals in sorted(b3_slices.items()):
        n = s_vals.get("n", 0)
        r1 = s_vals.get("recall@1", 0.0)
        r5 = s_vals.get("recall@5", 0.0)
        r10 = s_vals.get("recall@10", 0.0)
        smrr = s_vals.get("mrr", 0.0)
        lines.append(f"| `{s_name}` | {n} | {r1:.4f} | {r5:.4f} | {r10:.4f} | {smrr:.4f} |")

    lines += [
        "",
        "## 7. Recommendation and Release Decision",
        "",
    ]
    if is_official:
        if all_gates_pass:
            lines.append("All official acceptance gates PASSED on test split. Approved for production activation.")
        else:
            lines.append("One or more official acceptance gates FAILED on test split. Production release blocked.")
    else:
        lines.append("Evaluation completed on synthetic/trial data. Official production gate status remains TERTUNDA.")

    lines.append("")
    return "\n".join(lines)


def run_real(
    dataset_dir: Path,
    out_root: Path,
    *,
    freeze_report_path: Path | None = None,
    backend: Backend,
    lock: Path,
    locations_csv: Path | None = None,
    cache_dir: Path | None = None,
    allow_dirty: bool = False,
    run_id: str | None = None,
    config_dir: Path | None = None,
) -> Path:
    # 1. Check freeze verification
    target_freeze_path = freeze_report_path or (dataset_dir / "freeze_report.json")
    valid_freeze, is_official, freeze_msg, freeze_data = verify_freeze_report(target_freeze_path)
    if not valid_freeze:
        print(f"ERROR: {freeze_msg}", file=sys.stderr)
        raise SystemExit(1)

    commit, dirty = git_state()
    if dirty and not allow_dirty:
        msg = "uncommitted changes in ml, data/seed or api/internal/matching: commit them or pass --allow-dirty"
        raise SystemExit(msg)

    seed_everything()
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

    now_utc = datetime.now(UTC)
    active_run_id = run_id or f"{now_utc:%Y%m%dT%H%M%SZ}-real-{commit[:8]}"

    # 2. Calibration on val split
    base_cfg = Config.load()
    val_queries = engine.build("val", base_cfg)
    val_found_leaf = {e.id: e.leaf for e in dataset.split("val")}
    calibrate_name = f"fusion-linear-{active_run_id}"
    calibrated_cfg = calibrate.calibrate(val_queries, base_cfg, parents.__getitem__, val_found_leaf, calibrate_name)

    configs_target_dir = config_dir or (Path(__file__).resolve().parent / "configs")
    written_cfg = calibrate.write(calibrated_cfg, configs_target_dir)
    print(f"Calibrated fusion configuration written to {written_cfg}")

    # 3. Gate run on test split
    test_queries = engine.build("test", calibrated_cfg)
    test_found_leaf = {e.id: e.leaf for e in dataset.split("test")}

    chosen = variants(calibrated_cfg)
    no_syn = Engine(dataset, generated, {loc.slug: loc for loc in locations}, embedder, parents, compat_pairs, None)
    chosen["A7"] = Variant("A7", calibrated_cfg)
    test_alt_queries = no_syn.build("test", calibrated_cfg)

    all_metrics: dict[str, Any] = {}
    slice_rows: list[dict[str, Any]] = []
    per_query: list[dict[str, Any]] = []

    for name, variant in chosen.items():
        q_set = test_alt_queries if name == "A7" else test_queries
        results = evaluate_variant(variant, q_set, parents, test_found_leaf)
        all_metrics[name] = {"overall": metrics_of(results), "slices": {}}
        for label, members in slices(results).items():
            m = summary([r.rank for r in members])
            all_metrics[name]["slices"][label] = m
            slice_rows.append({"variant": name, "slice": label, **m})
        per_query += [{"variant": name, **r.row()} for r in results]

    all_metrics["components"] = {
        **components.t4_t5_t6(dataset, "test", backend, SEED_DIR),
        **components.t7(dataset, embedder, synonyms),
        **components.t8(dataset, embedder, synonyms),
    }
    gate_results = evaluate(all_metrics)
    all_metrics["gates"] = gate_results

    # 4. Packaging run directory and REPORT.md
    out = out_root / active_run_id
    out.mkdir(parents=True, exist_ok=False)

    (out / "metrics.json").write_text(json.dumps(all_metrics, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    write_csv(out / "per_query.csv", per_query)
    write_csv(out / "slices.csv", slice_rows)
    (out / "config.yaml").write_text(
        json.dumps(
            {
                "split": "test",
                "calibration_split": "val",
                "fusion": calibrated_cfg.raw,
                "fusion_sha256": hashlib.sha256(json.dumps(calibrated_cfg.raw, sort_keys=True).encode()).hexdigest(),
                "scenario": scenario.manifest_entry(),
                "variants": sorted(chosen),
                "pool_sizes": {q.item.id: len(q.pool) for q in test_queries},
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
                "freeze_report": freeze_data,
                "is_official": is_official,
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

    report_md = build_markdown_report(
        run_id=active_run_id,
        timestamp=f"{now_utc:%Y-%m-%d %H:%M:%SZ}",
        is_official=is_official,
        git_commit=commit,
        dirty=dirty,
        dataset_version=dataset.version,
        freeze_data=freeze_data,
        calibrated_cfg=calibrated_cfg,
        all_metrics=all_metrics,
        gates=gate_results,
        registry=registry,
    )
    (out / "REPORT.md").write_text(report_md, encoding="utf-8")
    return out


def main(argv: list[str] | None = None, backend_factory: Callable[[Registry, Path], Backend] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset", type=Path, default=ROOT / "data" / "eval")
    parser.add_argument("--freeze-report", type=Path, help="path to freeze_report.json")
    parser.add_argument("--out", type=Path, default=Path(__file__).resolve().parent / "runs")
    parser.add_argument("--lock", type=Path, default=ROOT / "ml" / "models.lock.json")
    parser.add_argument("--locations", type=Path)
    parser.add_argument("--cache", type=Path)
    parser.add_argument(
        "--hf-home", type=Path, default=Path(os.environ.get("HF_HOME", Path.home() / ".cache" / "huggingface"))
    )
    parser.add_argument("--allow-dirty", action="store_true")
    parser.add_argument("--run-id", type=str)
    args = parser.parse_args(argv)

    registry = Registry.from_lock(args.lock)
    if backend_factory:
        backend = backend_factory(registry, args.hf_home)
    else:
        from app.backend import TorchBackend

        backend = TorchBackend(registry, args.hf_home, threads=4)

    backend.load()
    out = run_real(
        args.dataset,
        args.out,
        freeze_report_path=args.freeze_report,
        backend=backend,
        lock=args.lock,
        locations_csv=args.locations,
        cache_dir=args.cache,
        allow_dirty=args.allow_dirty,
        run_id=args.run_id,
    )
    print(f"\nReal evaluation finished successfully.\nRun directory: {out}\nReport: {out / 'REPORT.md'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
