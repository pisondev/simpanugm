"""The acceptance gates of evaluation.md section 8. A gate is never lowered within the semester."""

from __future__ import annotations

from typing import Any

# slices where Recall@5 may stay under the bar: generic items are the hard case by construction
EXEMPT_SLICES = {"distinctiveness=generic"}
SLICE_FLOOR = 0.60
TOLERANCE = 0.02


def gate(name: str, value: float | None, threshold: float, *, atleast: bool = True) -> dict[str, Any]:
    ok = value is not None and (value >= threshold if atleast else value <= threshold)
    return {"gate": name, "value": value, "threshold": threshold, "passed": bool(ok)}


def evaluate(metrics: dict[str, Any], active: dict[str, Any] | None = None) -> list[dict[str, Any]]:
    """Gate table for B3 on the test split. `active` is the metrics.json of the version in production, if any."""
    b3 = metrics["B3"]
    overall = b3["overall"]
    rows = [
        gate("recall@5", overall["recall@5"], 0.80),
        gate("recall@1", overall["recall@1"], 0.55),
        gate("mrr", overall["mrr"], 0.65),
        gate("high tier precision", overall["tier_precision"]["high"], 0.85),
    ]
    without = b3["slices"].get("lost_photo=without")
    rows.append(gate("recall@10 without lost photo", None if without is None else without["recall@10"], 0.60))
    weak = {
        k: v["recall@5"]
        for k, v in b3["slices"].items()
        if k not in EXEMPT_SLICES and not k.startswith("noise_any=") and v["n"] >= 1 and v["recall@5"] < SLICE_FLOOR
    }
    rows.append({"gate": "no slice below recall@5 0.60", "value": weak, "threshold": SLICE_FLOOR, "passed": not weak})
    if active:
        prev = active["B3"]["overall"]
        worse = {
            m: round(prev[m] - overall[m], 4)
            for m in ("recall@5", "recall@1", "mrr")
            if prev[m] - overall[m] > TOLERANCE
        }
        rows.append(
            {
                "gate": "no gate metric worse than active by more than 0.02",
                "value": worse,
                "threshold": TOLERANCE,
                "passed": not worse,
            }
        )
    c = metrics.get("components")
    if c:
        rows += [
            gate("T4 category top 3", c.get("t4_top3"), 0.85),
            gate("T5 colour", c.get("t5_colour"), 0.75),
            gate("T6 brand precision", c.get("t6_precision"), 0.90),
            gate("T6 brand recall", c.get("t6_recall"), 0.50),
            gate("T7 duplicate precision", c.get("t7_precision"), 0.80),
            gate("T8 semantic recall@5", c.get("t8_recall@5"), 0.80),
        ]
    return rows
