"""Learned fusion experiment with Logistic Regression (matching-engine section 7.2, evaluation.md section 7).

Trains a logistic regression model on multi-modal component features using grouped cross validation
and compares performance against calibrated linear weighted mean fusion (B3).
"""

from __future__ import annotations

import argparse
import logging
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import numpy as np
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import GroupKFold

from eval import fusion
from eval.engine import Query, Ranked, rank_of_true
from eval.fusion import Config
from eval.metrics import ece, summary, tier_precision

logger = logging.getLogger(__name__)

FEATURE_NAMES = [
    "s_vis",
    "vis_missing",
    "mode_image",
    "mode_cross",
    "s_sem",
    "sem_missing",
    "s_attr",
    "attr_missing",
    "s_loc",
    "loc_missing",
    "s_time",
    "time_missing",
    "category_factor",
]


@dataclass
class LearnedFusionResult:
    b3_metrics: dict[str, Any]
    b4_metrics: dict[str, Any]
    coefficients: dict[str, float]
    intercept: float
    gate_decision: dict[str, Any]


def extract_pair_features(
    cfg: Config, s: Any, q: Query, parent_of: Callable[[str], str], found_leaf: dict[str, str]
) -> list[float]:
    """Extract the 13 features described in matching-engine.md section 7.2."""
    # 1. Visual
    if s.visual is not None and s.mode in ("image", "cross_modal"):
        band_name = "cross_modal" if s.mode == "cross_modal" else "image"
        s_vis = fusion.norm(s.visual, tuple(cfg.raw["bands"][band_name]))
        vis_missing = 0.0
    else:
        s_vis = 0.0
        vis_missing = 1.0

    mode_image = 1.0 if s.mode == "image" else 0.0
    mode_cross = 1.0 if s.mode == "cross_modal" else 0.0

    # 2. Semantic
    if s.semantic is not None:
        s_sem = fusion.norm(s.semantic, tuple(cfg.raw["bands"]["e5"]))
        sem_missing = 0.0
    else:
        s_sem = 0.0
        sem_missing = 1.0

    # 3. Attributes
    attr_val = fusion.attributes(cfg, s.colour, s.brand)
    if attr_val is not None:
        s_attr = attr_val
        attr_missing = 0.0
    else:
        s_attr = 0.0
        attr_missing = 1.0

    # 4. Location
    if s.distance_m is not None:
        s_loc = fusion.location_term(cfg, s.distance_m)
        loc_missing = 0.0
    else:
        s_loc = 0.0
        loc_missing = 1.0

    # 5. Time
    s_time = fusion.time_score(cfg, s.found_at, s.lost_from, s.lost_until)
    time_missing = 0.0

    # 6. Category factor
    same_parent = parent_of(q.meta.category) == parent_of(found_leaf[s.found_id])
    g = fusion.category_factor(cfg, same_leaf=s.same_leaf, same_parent=same_parent, compatible=s.compatible)

    return [
        s_vis,
        vis_missing,
        mode_image,
        mode_cross,
        s_sem,
        sem_missing,
        s_attr,
        attr_missing,
        s_loc,
        loc_missing,
        s_time,
        time_missing,
        g,
    ]


def run_experiment(
    queries: list[Query],
    cfg: Config,
    parent_of: Callable[[str], str],
    found_leaf: dict[str, str],
    random_state: int = 2026,
) -> LearnedFusionResult:
    """Train logistic regression and compare against calibrated linear fusion."""
    x_rows: list[list[float]] = []
    y_labels: list[int] = []
    groups: list[str] = []
    pair_meta: list[tuple[int, str, bool]] = []  # (query_idx, found_id, true_pair)

    for q_idx, q in enumerate(queries):
        for s in q.pool:
            feats = extract_pair_features(cfg, s, q, parent_of, found_leaf)
            x_rows.append(feats)
            y_labels.append(1 if s.true_pair else 0)
            groups.append(q.item.id)
            pair_meta.append((q_idx, s.found_id, s.true_pair))

    x_mat = np.array(x_rows, dtype=np.float64)
    y = np.array(y_labels, dtype=np.int32)
    group_arr = np.array(groups)

    # 1. Baseline B3 scoring
    b3_ranked_queries: list[list[Ranked]] = []
    for q in queries:
        ranked = fusion_score_query(cfg, q, parent_of, found_leaf)
        b3_ranked_queries.append(ranked)

    b3_ranks = [rank_of_true(r) for r in b3_ranked_queries]
    b3_metrics: dict[str, Any] = dict(summary(b3_ranks))
    b3_top = [(c.tier, c.true_pair) for r in b3_ranked_queries for c in r[:10]]
    b3_tiers, b3_truth = [t for t, _ in top_item(b3_top)], [v for _, v in top_item(b3_top)]
    b3_metrics["tier_precision_high"] = tier_precision(b3_tiers, b3_truth, "high")

    # 2. B4 Learned Fusion via GroupKFold cross-validation
    unique_groups = len(set(groups))
    n_splits = min(5, max(2, unique_groups))
    gkf = GroupKFold(n_splits=n_splits)

    oof_probs = np.zeros(len(y), dtype=np.float64)
    for train_idx, val_idx in gkf.split(x_mat, y, groups=group_arr):
        clf = LogisticRegression(C=1.0, random_state=random_state, max_iter=1000)
        # Handle single class in training fold if dataset is tiny
        if len(np.unique(y[train_idx])) > 1:
            clf.fit(x_mat[train_idx], y[train_idx])
            oof_probs[val_idx] = clf.predict_proba(x_mat[val_idx])[:, 1]
        else:
            oof_probs[val_idx] = 0.5

    # Full fit for coefficients
    final_clf = LogisticRegression(C=1.0, random_state=random_state, max_iter=1000)
    if len(np.unique(y)) > 1:
        final_clf.fit(x_mat, y)
        coef_dict = {name: float(c) for name, c in zip(FEATURE_NAMES, final_clf.coef_[0], strict=True)}
        intercept = float(final_clf.intercept_[0])
    else:
        coef_dict = {name: 0.0 for name in FEATURE_NAMES}
        intercept = 0.0

    # Build B4 rankings per query from OOF probabilities
    b4_ranked_queries: list[list[Ranked]] = [[] for _ in range(len(queries))]
    for idx, (q_idx, found_id, true_pair) in enumerate(pair_meta):
        prob = float(oof_probs[idx])
        # Map probability to tier
        tier = "high" if prob >= 0.72 else ("medium" if prob >= 0.55 else ("low" if prob >= 0.40 else "not_stored"))
        b4_ranked_queries[q_idx].append(
            Ranked(found_id=found_id, overall=round(prob, 4), tier=tier, true_pair=true_pair)
        )

    for q_idx in range(len(queries)):
        b4_ranked_queries[q_idx].sort(key=lambda r: (-r.overall, r.found_id))
        b4_ranked_queries[q_idx] = b4_ranked_queries[q_idx][:50]

    b4_ranks = [rank_of_true(r) for r in b4_ranked_queries]
    b4_metrics: dict[str, Any] = dict(summary(b4_ranks))
    b4_top = [(c.tier, c.true_pair) for r in b4_ranked_queries for c in r[:10]]
    b4_tiers, b4_truth = [t for t, _ in top_item(b4_top)], [v for _, v in top_item(b4_top)]
    b4_metrics["tier_precision_high"] = tier_precision(b4_tiers, b4_truth, "high")
    b4_metrics["ece"] = ece(list(oof_probs), [bool(v) for v in y])

    # 3. Acceptance gate evaluation
    # Rule: activates only if it improves both Recall@5 and MRR by at least 0.02 absolute
    # and keeps high tier precision at or above 0.85
    r5_diff = b4_metrics["recall@5"] - b3_metrics["recall@5"]
    mrr_diff = b4_metrics["mrr"] - b3_metrics["mrr"]
    high_prec = b4_metrics["tier_precision_high"]

    meets_r5 = r5_diff >= 0.02
    meets_mrr = mrr_diff >= 0.02
    meets_precision = high_prec is not None and high_prec >= 0.85

    gate_passed = meets_r5 and meets_mrr and meets_precision

    decision = {
        "passed": gate_passed,
        "meets_recall@5_delta": meets_r5,
        "recall@5_delta": round(r5_diff, 4),
        "meets_mrr_delta": meets_mrr,
        "mrr_delta": round(mrr_diff, 4),
        "meets_high_precision": meets_precision,
        "high_tier_precision": high_prec,
        "recommendation": "ACTIVATE" if gate_passed else "KEEP_LINEAR_FUSION",
    }

    return LearnedFusionResult(
        b3_metrics=b3_metrics,
        b4_metrics=b4_metrics,
        coefficients=coef_dict,
        intercept=intercept,
        gate_decision=decision,
    )


def top_item(top: list[tuple[str, bool]]) -> list[tuple[str, bool]]:
    return top


def fusion_score_query(
    cfg: Config, q: Query, parent_of: Callable[[str], str], found_leaf: dict[str, str]
) -> list[Ranked]:
    out: list[Ranked] = []
    for s in q.pool:
        category = fusion.category_factor(
            cfg,
            same_leaf=s.same_leaf,
            same_parent=parent_of(q.meta.category) == parent_of(found_leaf[s.found_id]),
            compatible=s.compatible,
        )
        pair = fusion.Pair(
            mode=s.mode,  # type: ignore[arg-type]
            visual_cosine=s.visual,
            semantic_cosine=s.semantic,
            colour=s.colour,
            brand=s.brand,
            location=fusion.location_term(cfg, s.distance_m) if s.distance_m is not None else None,
            time=fusion.time_score(cfg, s.found_at, s.lost_from, s.lost_until),
            category=category,
        )
        scored = fusion.score(cfg, pair)
        out.append(Ranked(s.found_id, scored.overall, scored.tier, s.true_pair))
    kept = [r for r in out if r.tier != fusion.NOT_STORED]
    kept.sort(key=lambda r: (-r.overall, r.found_id))
    return kept[:50]


def main() -> None:
    import json
    import os
    import tempfile
    from collections import defaultdict

    from app.backend import Backend
    from app.registry import Registry
    from eval import scenario, text
    from eval.dataset import leaf_parents, load
    from eval.engine import Embedder, Engine, load_compat
    from tests.fakes import FakeBackend
    from tests.synth import build

    parser = argparse.ArgumentParser(description="Learned fusion experiment (B4 vs B3)")
    parser.add_argument("--dataset", type=Path)
    parser.add_argument("--split", choices=["val", "test"], default="val")
    parser.add_argument("--items", type=int, default=20)
    parser.add_argument("--distractors", type=int, default=12)
    parser.add_argument(
        "--real-weights", action="store_true", help="use real PyTorch neural models instead of fake backend"
    )
    parser.add_argument(
        "--hf-home", type=Path, default=Path(os.environ.get("HF_HOME", Path.home() / ".cache" / "huggingface"))
    )
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[2]
    seed_dir = root / "data" / "seed"
    cfg = Config.load()
    lock_path = root / "ml" / "models.lock.json"
    registry = Registry.from_lock(lock_path)

    backend: Backend
    if args.real_weights:
        from app.backend import TorchBackend

        backend = TorchBackend(registry, args.hf_home, threads=4)
        backend.load()
    else:
        backend = FakeBackend(registry)

    with tempfile.TemporaryDirectory() as tmp_dir:
        dataset_path = args.dataset
        is_official = False
        if not dataset_path or not (dataset_path / "items.csv").is_file():
            print("Using synthetic dataset fixture for learned fusion experiment (SINTETIS)")
            dataset_path = build(Path(tmp_dir) / "synth_data", items=args.items, distractors=args.distractors)
        else:
            freeze_file = dataset_path / "freeze_report.json"
            if freeze_file.is_file():
                try:
                    fdata = json.loads(freeze_file.read_text(encoding="utf-8"))
                    is_official = (
                        fdata.get("profile") == "full"
                        and not fdata.get("is_synthetic", True)
                        and fdata.get("item_count", 0) >= 150
                    )
                except Exception:
                    is_official = False

        status_tag = "RESMI" if is_official else "SINTETIS"
        dataset = load(dataset_path, seed_dir)
        parents = leaf_parents(seed_dir)
        locations = scenario.load_locations(root / "deploy" / "dev-seed" / "locations.csv")
        synonyms = text.load_synonyms(seed_dir / "synonyms.csv")
        compat_pairs = load_compat(seed_dir / "category_compatibility.csv")
        compatible: dict[str, list[str]] = defaultdict(list)
        for key in compat_pairs:
            a, b = key.split("|")
            compatible[a].append(b)
            compatible[b].append(a)

        items = {e.id: e.leaf for e in dataset.entries.values() if e.is_item}
        distractors = [e.id for e in dataset.entries.values() if not e.is_item]
        generated = scenario.Generator(locations, parents, compatible).generate(items, distractors)
        identity = "|".join(f"{n}={c.model_id}@{c.revision}" for n, c in sorted(registry.components.items()))
        embedder = Embedder(backend, identity, None)
        engine = Engine(
            dataset, generated, {loc.slug: loc for loc in locations}, embedder, parents, compat_pairs, synonyms
        )
        queries = engine.build(args.split, cfg)
        found_leaf = {e.id: e.leaf for e in dataset.split(args.split)}

        res = run_experiment(queries, cfg, parents.__getitem__, found_leaf)

        print(f"\n=== Learned Fusion Experiment Results ({status_tag}) ===")
        print(f"Dataset: {dataset_path} ({status_tag})")
        print(f"Dataset split: {args.split} (items: {len(queries)})")
        print("\nComparison Table:")
        print(f"{'Metric':<25} | {'B3 (Linear Fusion)':<20} | {'B4 (Learned Fusion)':<20}")
        print("-" * 72)
        for metric in ["recall@1", "recall@5", "recall@10", "mrr", "ndcg@10"]:
            b3_v = res.b3_metrics.get(metric, 0.0)
            b4_v = res.b4_metrics.get(metric, 0.0)
            print(f"{metric:<25} | {b3_v:<20.4f} | {b4_v:<20.4f}")
        b3_hp = res.b3_metrics.get("tier_precision_high")
        b4_hp = res.b4_metrics.get("tier_precision_high")
        print(f"{'high_tier_precision':<25} | {b3_hp!s:<20} | {b4_hp!s:<20}")
        print(f"{'ece':<25} | {'N/A':<20} | {res.b4_metrics.get('ece', 0.0):<20.4f}")

        print("\nLearned Logistic Regression Coefficients:")
        for name, coef in res.coefficients.items():
            print(f"  {name:<20}: {coef:+.4f}")
        print(f"  {'intercept':<20}: {res.intercept:+.4f}")

        print("\nAcceptance Gate Decision (docs/ai/matching-engine.md section 7.2):")
        print("Decision Rule: B4 learned fusion is recommended for production activation ONLY if it")
        print("beats B3 linear fusion on the frozen test split with:")
        print("  1. Recall@5 delta >= +0.02 absolute")
        print("  2. MRR delta >= +0.02 absolute")
        print("  3. High tier precision >= 0.85 on test split")
        print("Otherwise, linear fusion (fusion-linear-2026.11.0.json) remains active in production.")
        r5_d = res.gate_decision["recall@5_delta"]
        mrr_d = res.gate_decision["mrr_delta"]
        print(f"  Recall@5 delta >= +0.02: {res.gate_decision['meets_recall@5_delta']} (delta: {r5_d:+.4f})")
        print(f"  MRR delta >= +0.02     : {res.gate_decision['meets_mrr_delta']} (delta: {mrr_d:+.4f})")
        print(f"  High tier prec >= 0.85 : {res.gate_decision['meets_high_precision']} ({b4_hp})")
        print(f"  Recommendation         : {res.gate_decision['recommendation']}")


if __name__ == "__main__":
    main()
