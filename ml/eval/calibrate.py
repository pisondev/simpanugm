"""Calibration on `val`, evaluation.md section 6. The result is a new fusion document under ml/eval/configs/."""

from __future__ import annotations

import itertools
import json
import statistics
from collections.abc import Callable
from pathlib import Path
from typing import Any

from eval.engine import Query, Variant, rank, rank_of_true
from eval.fusion import Config
from eval.metrics import mrr

STEP = 0.05
HIGH_PRECISION = 0.85
MEDIUM_RECALL = 0.80
MIN_SUPPORT = 5
FLOOR_DEFAULT = 0.40
FLOOR_LOSS = 0.05


def _round2(x: float) -> float:
    return round(x * 100) / 100


def bands(queries: list[Query], cfg: Config) -> dict[str, list[float]]:
    """`lo` is the median cosine of retrieved negatives, `hi` the 95th percentile of true pairs, two decimals."""
    pools: dict[str, tuple[list[float], list[float]]] = {"image": ([], []), "cross_modal": ([], []), "e5": ([], [])}
    for q in queries:
        for s in q.pool:
            key = s.mode if s.mode in ("image", "cross_modal") else None
            if key and s.visual is not None:
                pools[key][0 if s.true_pair else 1].append(s.visual)
            pools["e5"][0 if s.true_pair else 1].append(s.semantic)
    out = {k: list(v) for k, v in cfg.raw["bands"].items()}
    for name, (pos, neg) in pools.items():
        if len(pos) >= 2 and neg:
            lo, hi = _round2(statistics.median(neg)), _round2(statistics.quantiles(pos, n=20)[-1])
            if hi > lo:
                out[name] = [lo, hi]
    return out


def _mrr(queries: list[Query], cfg: Config, parents: Callable[[str], str], leaf: dict[str, str]) -> float:
    return mrr([rank_of_true(rank(Variant("cal", cfg), q, parents, leaf)) for q in queries])


def _combos(lo_hi: dict[str, tuple[float, float]]) -> list[dict[str, float]]:
    axes = {k: [round(lo + STEP * i, 2) for i in range(round((hi - lo) / STEP) + 1)] for k, (lo, hi) in lo_hi.items()}
    out = []
    for values in itertools.product(*axes.values()):
        if abs(sum(values) - 1.0) < 1e-9:
            out.append(dict(zip(axes, values, strict=True)))
    return out


def _spread(w: dict[str, float]) -> float:
    """Distance from equal weights, the tie breaker."""
    mean = 1 / len(w)
    return sum((v - mean) ** 2 for v in w.values())


def weights(queries: list[Query], cfg: Config, parents: Callable[[str], str], leaf: dict[str, str]) -> dict[str, float]:
    """Grid search maximising MRR, ties broken toward the simplest configuration (closest to equal weights)."""
    base = dict(cfg.raw["weights"])
    best: tuple[float, float, dict[str, float]] | None = None
    ranges = {"visual_image": (0.20, 0.50), "text": (0.20, 0.50), "location": (0.05, 0.25), "time": (0.05, 0.25)}
    for combo in _combos(ranges):
        trial = cfg.with_changes(weights={**base, **combo})
        score = _mrr(queries, trial, parents, leaf)
        key = (score, -_spread(combo))
        if best is None or key > (best[0], -best[1]):
            best = (score, _spread(combo), combo)
    chosen = dict(base)
    if best:
        chosen.update(best[2])
    # the cross modal weight is searched on its own subset, the other weights fixed
    subset = [q for q in queries if not q.item.has_owner_photo]
    if subset:
        top: tuple[float, float, float] | None = None
        for wx in [round(0.15 + STEP * i, 2) for i in range(6)]:
            trial = cfg.with_changes(weights={**chosen, "visual_cross_modal": wx})
            key = (_mrr(subset, trial, parents, leaf), -abs(wx - chosen["visual_image"]))
            if top is None or key > (top[0], -top[1]):
                top = (key[0], abs(wx - chosen["visual_image"]), wx)
        if top:
            chosen["visual_cross_modal"] = top[2]
    return {k: round(v, 2) for k, v in chosen.items()}


def thresholds(
    queries: list[Query], cfg: Config, parents: Callable[[str], str], leaf: dict[str, str]
) -> dict[str, float]:
    ranked = [rank(Variant("cal", cfg, floor=False), q, parents, leaf) for q in queries]
    cands = [(c.overall, c.true_pair, i) for i, r in enumerate(ranked) for c in r[:10]]
    scores = sorted({s for s, _, _ in cands})
    tiers = dict(cfg.raw["tiers"])
    high = None
    for t in scores:
        kept = [tp for s, tp, _ in cands if s >= t]
        if len(kept) >= MIN_SUPPORT and sum(kept) / len(kept) >= HIGH_PRECISION:
            high = t
            break
    if high is not None:
        tiers["high"] = _round2(high)
    medium = None
    for t in reversed(scores):
        hits = sum(1 for r in ranked if any(c.true_pair and c.overall >= t for c in r[:5]))
        if ranked and hits / len(ranked) >= MEDIUM_RECALL:
            medium = t
            break
    if medium is not None and medium < tiers["high"]:
        tiers["medium"] = _round2(medium)
    below = sum(1 for r in ranked if any(c.true_pair and c.overall < FLOOR_DEFAULT for c in r))
    tiers["floor"] = (
        FLOOR_DEFAULT if not ranked or below / len(ranked) <= FLOOR_LOSS else _round2(max(0.2, FLOOR_DEFAULT - 0.1))
    )
    tiers["medium"] = min(tiers["medium"], _round2(tiers["high"] - 0.01))
    return {k: float(v) for k, v in tiers.items()}


def calibrate(
    queries: list[Query], cfg: Config, parents: Callable[[str], str], leaf: dict[str, str], new_id: str
) -> Config:
    """The four steps of section 6 in order, each seeing the result of the one before."""
    step1 = cfg.with_changes(bands=bands(queries, cfg))
    step2 = step1.with_changes(weights=weights(queries, step1, parents, leaf))
    step3 = step2.with_changes(tiers=thresholds(queries, step2, parents, leaf))
    return step3.with_changes(id=new_id)


def write(cfg: Config, directory: Path) -> Path:
    path = directory / f"{cfg.id}.json"
    if path.exists():
        raise FileExistsError(f"{path} already exists, a version id is never reused")
    doc: dict[str, Any] = cfg.raw
    path.write_text(json.dumps(doc, indent=2) + "\n", encoding="utf-8")
    return path
