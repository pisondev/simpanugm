"""Metrics of evaluation.md section 5. A rank is 1 based, None means the true report was not retrieved in the top 50."""

from __future__ import annotations

import math
import random
from collections.abc import Callable, Sequence

CUTOFF = 50


def recall_at_k(ranks: Sequence[int | None], k: int) -> float:
    return sum(1 for r in ranks if r is not None and r <= k) / len(ranks) if ranks else 0.0


def mrr(ranks: Sequence[int | None]) -> float:
    return sum(1 / r for r in ranks if r is not None and r <= CUTOFF) / len(ranks) if ranks else 0.0


def ndcg_at_10(ranks: Sequence[int | None]) -> float:
    """With one relevant item the ideal gain is 1, so the value is 1 / log2(rank + 1) inside the top 10."""
    return sum(1 / math.log2(r + 1) for r in ranks if r is not None and r <= 10) / len(ranks) if ranks else 0.0


def tier_precision(tiers: Sequence[str], truth: Sequence[bool], tier: str) -> float | None:
    """Among candidates in a tier, the share that are true pairs. None when the tier is empty."""
    hits = [t for tr, t in zip(tiers, truth, strict=True) if tr == tier]
    return sum(hits) / len(hits) if hits else None


def tier_recall(tiers: Sequence[str], truth: Sequence[bool], tier: str, n_true: int) -> float:
    """Among all true pairs, the share that land in this tier or above."""
    order = {"high": 2, "medium": 1, "low": 0}
    at_least = [t for tr, t in zip(tiers, truth, strict=True) if tr in order and order[tr] >= order[tier]]
    return sum(at_least) / n_true if n_true else 0.0


def ece(confidence: Sequence[float], outcome: Sequence[bool], bins: int = 10) -> float:
    """Expected calibration error over equal width bins."""
    total = len(confidence)
    if not total:
        return 0.0
    err = 0.0
    for b in range(bins):
        lo, hi = b / bins, (b + 1) / bins
        members = [
            (c, o) for c, o in zip(confidence, outcome, strict=True) if lo <= c < hi or (b == bins - 1 and c == 1.0)
        ]
        if members:
            err += (
                len(members)
                / total
                * abs(sum(c for c, _ in members) / len(members) - sum(o for _, o in members) / len(members))
            )
    return err


def bootstrap_ci(
    ranks: Sequence[int | None],
    metric: Callable[[Sequence[int | None]], float],
    *,
    seed: int = 2026,
    resamples: int = 1000,
) -> tuple[float, float]:
    """95 percent interval over items, 1000 resamples, fixed seed."""
    if not ranks:
        return (0.0, 0.0)
    rng = random.Random(seed)  # noqa: S311  (resampling, not security)
    n = len(ranks)
    values = sorted(metric([ranks[rng.randrange(n)] for _ in range(n)]) for _ in range(resamples))
    return (values[int(0.025 * resamples)], values[int(0.975 * resamples) - 1])


def summary(ranks: Sequence[int | None]) -> dict[str, float]:
    return {
        "n": float(len(ranks)),
        "recall@1": recall_at_k(ranks, 1),
        "recall@5": recall_at_k(ranks, 5),
        "recall@10": recall_at_k(ranks, 10),
        "mrr": mrr(ranks),
        "ndcg@10": ndcg_at_10(ranks),
    }
