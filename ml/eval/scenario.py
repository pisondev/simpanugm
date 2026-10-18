"""The scenario generator of evaluation.md section 3. Photos and texts are real, the metadata is drawn here with a fixed
seed so every run sees the same places and times. Changing a parameter is a new dataset version."""

from __future__ import annotations

import csv
import hashlib
import json
import math
import random
from dataclasses import asdict, dataclass, field
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

WIB = timedelta(hours=7)
SEED = 2026
KIND_WEIGHT = {"building": 3.0, "facility": 2.5, "outdoor": 1.5, "transport": 1.0, "parking": 1.0}
NEAR_M = 300.0

PARAMETERS: dict[str, Any] = {
    "seed": SEED,
    "window_start": "2026-09-07",
    "window_weeks": 10,
    "day_hours": [7, 21],
    "owner_places": {"true_only": 0.70, "true_plus_near": 0.15, "near_only": 0.10, "unknown": 0.05},
    "owner_window": {"before_h": 3.0, "after_h": 2.0, "wrong_day": 0.10},
    "found_delay_mean_h": 6.0,
    "found_place_true": 0.85,
    "owner_category_true": 0.85,
    "near_m": NEAR_M,
}
PARAMETERS_HASH = hashlib.sha256(json.dumps(PARAMETERS, sort_keys=True).encode()).hexdigest()


@dataclass(frozen=True)
class Location:
    slug: str
    name: str
    kind: str
    lat: float
    lng: float


def load_locations(path: Path) -> list[Location]:
    with path.open(newline="", encoding="utf-8") as handle:
        return [
            Location(r["slug"], r["name"], r["kind"], float(r["lat"]), float(r["lng"])) for r in csv.DictReader(handle)
        ]


def haversine_m(a: Location, b: Location) -> float:
    rad = math.pi / 180
    dlat, dlng = (b.lat - a.lat) * rad, (b.lng - a.lng) * rad
    h = math.sin(dlat / 2) ** 2 + math.cos(a.lat * rad) * math.cos(b.lat * rad) * math.sin(dlng / 2) ** 2
    return 2 * 6371000.0 * math.asin(min(1.0, math.sqrt(h)))


@dataclass
class LostMeta:
    places: list[str]
    unknown: bool
    lost_from: datetime
    lost_until: datetime
    category: str
    flags: list[str] = field(default_factory=list)


@dataclass
class FoundMeta:
    place: str
    found_at: datetime


@dataclass
class Scenario:
    lost: dict[str, LostMeta]
    found: dict[str, FoundMeta]


class Generator:
    def __init__(
        self, locations: list[Location], leaf_parent: dict[str, str], compatible: dict[str, list[str]]
    ) -> None:
        self.locations = sorted(locations, key=lambda loc: loc.slug)
        self.by_slug = {loc.slug: loc for loc in self.locations}
        self.leaf_parent = leaf_parent
        self.compatible = compatible

    def _place(self, rng: random.Random) -> Location:
        return rng.choices(self.locations, weights=[KIND_WEIGHT.get(loc.kind, 1.0) for loc in self.locations])[0]

    def _near(self, rng: random.Random, origin: Location) -> Location | None:
        close = [loc for loc in self.locations if loc.slug != origin.slug and haversine_m(origin, loc) <= NEAR_M]
        return rng.choice(close) if close else None

    def _time(self, rng: random.Random) -> datetime:
        start = datetime.fromisoformat(PARAMETERS["window_start"]).replace(tzinfo=UTC)
        days = rng.randrange(int(PARAMETERS["window_weeks"]) * 7)
        while (start + timedelta(days=days)).weekday() >= 5:
            days = rng.randrange(int(PARAMETERS["window_weeks"]) * 7)
        lo, hi = PARAMETERS["day_hours"]
        local = start + timedelta(days=days, hours=lo, minutes=rng.randrange((hi - lo) * 60))
        return local - WIB

    def _category(self, rng: random.Random, true_leaf: str, flags: list[str]) -> str:
        if rng.random() < float(PARAMETERS["owner_category_true"]):
            return true_leaf
        siblings = sorted(s for s, p in self.leaf_parent.items() if p == self.leaf_parent[true_leaf] and s != true_leaf)
        options = siblings + sorted(self.compatible.get(true_leaf, []))
        if not options:
            return true_leaf
        flags.append("wrong_category")
        return rng.choice(options)

    def generate(self, items: dict[str, str], distractors: list[str]) -> Scenario:
        """items maps item id to its true leaf category. Draws follow sorted ids, so the seed fixes everything."""
        rng = random.Random(SEED)  # noqa: S311
        lost: dict[str, LostMeta] = {}
        found: dict[str, FoundMeta] = {}
        for item_id in sorted(items):
            true_place = self._place(rng)
            t = self._time(rng)
            flags: list[str] = []
            draw = rng.random()
            op = PARAMETERS["owner_places"]
            places: list[str]
            unknown = False
            if draw < op["true_only"]:
                places = [true_place.slug]
            elif draw < op["true_only"] + op["true_plus_near"]:
                places = [true_place.slug]
                for _ in range(rng.choice([1, 2])):
                    near = self._near(rng, true_place)
                    if near and near.slug not in places:
                        places.append(near.slug)
            elif draw < op["true_only"] + op["true_plus_near"] + op["near_only"]:
                near = self._near(rng, true_place)
                places = [near.slug if near else true_place.slug]
                if near:
                    flags.append("wrong_place")
            else:
                places, unknown = [], True
            ow = PARAMETERS["owner_window"]
            lost_from = t - timedelta(hours=rng.uniform(0, ow["before_h"]))
            lost_until = t + timedelta(hours=rng.uniform(0, ow["after_h"]))
            if rng.random() < ow["wrong_day"]:
                lost_from, lost_until = lost_from + timedelta(days=1), lost_until + timedelta(days=1)
                flags.append("wrong_day")
            category = self._category(rng, items[item_id], flags)
            lost[item_id] = LostMeta(places, unknown, lost_from, lost_until, category, flags)
            found_at = t + timedelta(hours=rng.expovariate(1 / float(PARAMETERS["found_delay_mean_h"])))
            at = true_place
            if rng.random() >= float(PARAMETERS["found_place_true"]):
                at = self._near(rng, true_place) or true_place
            found[item_id] = FoundMeta(at.slug, found_at)
        for did in sorted(distractors):
            found[did] = FoundMeta(self._place(rng).slug, self._time(rng))
        return Scenario(lost, found)


def manifest_entry() -> dict[str, Any]:
    return {"parameters": asdict_safe(PARAMETERS), "parameters_sha256": PARAMETERS_HASH}


def asdict_safe(d: dict[str, Any]) -> dict[str, Any]:
    out: dict[str, Any] = json.loads(json.dumps(d))
    return out


__all__ = ["Generator", "Location", "Scenario", "asdict", "haversine_m", "load_locations", "manifest_entry"]
