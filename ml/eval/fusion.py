"""The Python reference of matching-engine sections 6 and 7.

It must agree with api/internal/matching/fusion on data/fixtures/fusion_cases.json to six decimals, and it reads the
same configuration document, so a calibrated version is one file that both sides load.
"""

from __future__ import annotations

import json
import math
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Any, Literal

CONFIGS = Path(__file__).resolve().parent / "configs"
DEFAULT = CONFIGS / "fusion-linear-2026.11.0.json"

NOT_STORED = "not_stored"
Mode = Literal["image", "cross_modal", "none"]

# the palette adjacency list of section 6.2, symmetric
_ADJACENT_PAIRS = [
    ("hitam", "abu-abu"),
    ("abu-abu", "perak"),
    ("perak", "putih"),
    ("putih", "krem"),
    ("krem", "cokelat"),
    ("cokelat", "oranye"),
    ("oranye", "merah"),
    ("merah", "merah muda"),
    ("oranye", "kuning"),
    ("kuning", "emas"),
    ("emas", "cokelat"),
    ("biru", "ungu"),
    ("biru", "hijau"),
]
ADJACENT = {p for pair in _ADJACENT_PAIRS for p in (pair, pair[::-1])}


@dataclass(frozen=True)
class Config:
    raw: dict[str, Any]

    @classmethod
    def load(cls, path: Path = DEFAULT) -> Config:
        raw = json.loads(path.read_text(encoding="utf-8"))
        tiers = raw["tiers"]
        if raw.get("kind") != "weighted_mean" or not tiers["floor"] < tiers["medium"] < tiers["high"]:
            raise ValueError(f"{path}: not a weighted mean fusion document with rising tiers")
        return cls(raw)

    @property
    def id(self) -> str:
        return str(self.raw["id"])

    def with_changes(self, **changes: Any) -> Config:
        """A copy with top level sections replaced, for ablations and calibration."""
        return Config({**self.raw, **changes})


def norm(x: float, band: tuple[float, float]) -> float:
    lo, hi = band
    if hi <= lo:
        return 0.0
    return min(1.0, max(0.0, (x - lo) / (hi - lo)))


def colour_relation(a: list[str], b: list[str]) -> str:
    if not a or not b:
        return "missing"
    relation = "different"
    for x in a:
        for y in b:
            x, y = x.strip().lower(), y.strip().lower()
            if x == y:
                return "shared"
            if (x, y) in ADJACENT:
                relation = "adjacent"
    return relation


def brand_relation(a: str, b: str) -> str:
    a, b = a.strip().lower(), b.strip().lower()
    if not a or not b:
        return "missing"
    return "same" if a == b else "different"


def category_factor(cfg: Config, *, same_leaf: bool, same_parent: bool, compatible: float | None) -> float:
    if same_leaf:
        return 1.0
    g = float(cfg.raw["category"]["same_parent" if same_parent else "other"])
    if compatible is not None and compatible > g:
        g = compatible
    return g


def location_term(cfg: Config, distance_m: float, *, zone_only: bool = False) -> float:
    loc = cfg.raw["location"]
    s = math.exp(-distance_m / loc["scale_m"])
    return min(s, loc["zone_cap"]) if zone_only else s


def time_score(cfg: Config, found_at: datetime, lost_from: datetime, lost_until: datetime) -> float:
    t = cfg.raw["time"]
    if (lost_from - found_at).total_seconds() / 3600 > t["tolerance_before_h"]:
        return 0.0
    if found_at <= lost_until:
        return 1.0
    return float(0.5 ** (((found_at - lost_until).total_seconds() / 3600) / t["half_life_h"]))


def time_score_hours(cfg: Config, *, relation: str, hours: float = 0.0) -> float:
    """The same rule from the relation names the shared fixture uses."""
    t = cfg.raw["time"]
    if relation == "before_window":
        return 0.0 if hours > t["tolerance_before_h"] else 1.0
    if relation == "in_window":
        return 1.0
    return float(0.5 ** (hours / t["half_life_h"]))


@dataclass
class Pair:
    """Raw signals of one pair, None is a missing signal that leaves the weighted mean."""

    mode: Mode = "none"
    visual_cosine: float | None = None
    semantic_cosine: float | None = None
    colour: str = "missing"
    brand: str = "missing"
    name_hit: bool = False
    location: float | None = None
    time: float = 1.0
    category: float = 1.0


@dataclass
class Scored:
    mode: Mode
    visual: float | None = None
    semantic: float | None = None
    attributes: float | None = None
    text: float | None = None
    location: float | None = None
    time: float = 0.0
    category: float = 1.0
    fused: float = 0.0
    overall: float = 0.0
    tier: str = NOT_STORED
    parts: dict[str, float] = field(default_factory=dict)


def tier_of(cfg: Config, overall: float) -> str:
    t = cfg.raw["tiers"]
    if overall >= t["high"]:
        return "high"
    if overall >= t["medium"]:
        return "medium"
    if overall >= t["floor"]:
        return "low"
    return NOT_STORED


def attributes(cfg: Config, colour: str, brand: str) -> float | None:
    total, n = 0.0, 0
    if colour == "shared":
        total, n = total + 1, n + 1
    elif colour == "adjacent":
        total, n = total + cfg.raw["colour_partial"], n + 1
    elif colour == "different":
        n += 1
    if brand == "same":
        total, n = total + 1, n + 1
    elif brand == "different":
        n += 1
    return None if n == 0 else total / n


def score(cfg: Config, pair: Pair) -> Scored:
    w = cfg.raw["weights"]
    r = Scored(mode=pair.mode, category=pair.category)
    weight_sum = weighted = 0.0

    def add(weight: float, value: float) -> None:
        nonlocal weight_sum, weighted
        weight_sum += weight
        weighted += weight * value

    if pair.visual_cosine is not None and pair.mode != "none":
        band_name, weight = (
            ("cross_modal", w["visual_cross_modal"]) if pair.mode == "cross_modal" else ("image", w["visual_image"])
        )
        r.visual = norm(pair.visual_cosine, tuple(cfg.raw["bands"][band_name]))
        add(weight, r.visual)
    else:
        r.mode = "none"
    r.attributes = attributes(cfg, pair.colour, pair.brand)
    mix = cfg.raw["text_mix"]
    if pair.semantic_cosine is not None:
        r.semantic = norm(pair.semantic_cosine, tuple(cfg.raw["bands"]["e5"]))
        r.text = r.semantic if r.attributes is None else mix["semantic"] * r.semantic + mix["attributes"] * r.attributes
    elif r.attributes is not None:
        r.text = r.attributes
    floor = cfg.raw["identity_name_hit"]["text_floor"]
    if pair.name_hit and (r.text is None or r.text < floor):
        r.text = floor
    if r.text is not None:
        add(w["text"], r.text)
    if pair.location is not None:
        r.location = pair.location
        add(w["location"], pair.location)
    r.time = pair.time
    add(w["time"], pair.time)
    if weight_sum > 0:
        r.fused = pair.category * weighted / weight_sum
    r.overall = math.floor(r.fused * 10000 + 0.5) / 10000
    r.tier = tier_of(cfg, r.overall)
    return r
