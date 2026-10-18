"""Embeds a dataset split once and ranks its pools under any fusion variant (evaluation.md sections 5 to 7)."""

from __future__ import annotations

import csv
import hashlib
import random
from collections.abc import Callable
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from pathlib import Path

import numpy as np
from numpy.typing import NDArray
from PIL import Image

from app.backend import Backend
from eval import fusion, text
from eval.dataset import Dataset, Entry
from eval.fusion import Config, Pair
from eval.scenario import FoundMeta, Location, LostMeta, Scenario, haversine_m

POOL_TOP = 50
Vec = NDArray[np.float32]


def load_compat(path: Path) -> dict[str, float]:
    """Pairs keyed by `a|b` with the smaller slug first."""
    with path.open(newline="", encoding="utf-8") as handle:
        return {
            "|".join(sorted((r["category_a"], r["category_b"]))): float(r["factor"]) for r in csv.DictReader(handle)
        }


@dataclass
class Signals:
    """Config independent raw signals of one lost and found pair, scored again under every variant."""

    found_id: str
    mode: str
    visual: float | None
    semantic: float
    colour: str
    brand: str
    distance_m: float | None
    compatible: float | None
    same_leaf: bool
    true_pair: bool
    found_at: datetime
    lost_from: datetime
    lost_until: datetime
    tfidf: float = 0.0


@dataclass
class Query:
    item: Entry
    meta: LostMeta
    pool: list[Signals] = field(default_factory=list)


class Embedder:
    """Caches vectors by the hash of their input and the model identity, so a rerun costs nothing."""

    def __init__(self, backend: Backend, identity: str, cache_dir: Path | None) -> None:
        self.backend = backend
        self.identity = identity
        self.cache_dir = cache_dir
        if cache_dir:
            cache_dir.mkdir(parents=True, exist_ok=True)

    def _key(self, kind: str, payload: bytes) -> str:
        return hashlib.sha256(kind.encode() + b"\0" + self.identity.encode() + b"\0" + payload).hexdigest()

    def _cached(self, kind: str, payload: bytes, make: Callable[[], Vec]) -> Vec:
        if self.cache_dir:
            path = self.cache_dir / f"{self._key(kind, payload)}.npy"
            if path.is_file():
                return np.load(path)
            vector = make()
            np.save(path, vector)
            return vector
        return make()

    def image(self, path: Path) -> Vec:
        return self._cached(
            "image", path.read_bytes(), lambda: self.backend.embed_images([Image.open(path).convert("RGB")])[0]
        )

    def e5(self, value: str) -> Vec:
        return self._cached("e5", value.encode(), lambda: self.backend.embed_e5([value])[0])

    def clip(self, value: str) -> Vec:
        return self._cached("clip", value.encode(), lambda: self.backend.embed_clip_text([value])[0])


def _text_input(e: Entry, owner: bool, category: str, parent: str, parents: dict[str, str]) -> text.TextInput:
    return text.TextInput(
        item_name=e.name,
        category=category,
        parent=parents.get(category, parent),
        colours=e.colours,
        brand=e.brand,
        description=e.owner_description if owner else e.finder_description,
    )


def _best(a: list[Vec], b: list[Vec]) -> float | None:
    if not a or not b:
        return None
    return float(max(np.dot(x, y) for x in a for y in b))


class Engine:
    def __init__(
        self,
        dataset: Dataset,
        scenario: Scenario,
        locations: dict[str, Location],
        embedder: Embedder,
        parents: dict[str, str],
        compat: dict[str, float],
        synonyms: dict[str, str] | None,
    ) -> None:
        self.dataset, self.scenario, self.locations = dataset, scenario, locations
        self.embedder, self.parents, self.compat, self.synonyms = embedder, parents, compat, synonyms

    def build(self, split: str, cfg: Config) -> list[Query]:
        """Signals for every lost report of the split against the found reports of the same split inside the pool."""
        found = self.dataset.split(split)
        found_vecs = {e.id: [self.embedder.image(p) for p in e.finder_images] for e in found}
        found_e5 = {
            e.id: self.embedder.e5(text.e5_text(_text_input(e, False, e.leaf, e.parent, self.parents), self.synonyms))
            for e in found
        }
        tfidf = _Tfidf([e.finder_description for e in found], [e.id for e in found])
        queries: list[Query] = []
        for item in self.dataset.items(split):
            meta = self.scenario.lost[item.id]
            owner_in = _text_input(
                item, True, meta.category, self.parents.get(meta.category, item.parent), self.parents
            )
            owner_vecs = [self.embedder.image(p) for p in item.owner_images] if item.has_owner_photo else []
            owner_e5 = self.embedder.e5(text.e5_text(owner_in, self.synonyms))
            owner_clip = None if owner_vecs else self.embedder.clip(text.clip_text(owner_in, self.synonyms))
            tf = tfidf.scores(item.owner_description)
            q = Query(item, meta)
            pool_start = meta.lost_from - timedelta(hours=cfg.raw["time"]["pool_before_h"])
            pool_end = meta.lost_until + timedelta(days=cfg.raw["time"]["pool_after_d"])
            for f in found:
                fm = self.scenario.found[f.id]
                if not pool_start <= fm.found_at <= pool_end:
                    continue
                q.pool.append(
                    self._signals(
                        item,
                        meta,
                        f,
                        fm,
                        owner_vecs,
                        owner_clip,
                        owner_e5,
                        found_vecs[f.id],
                        found_e5[f.id],
                        tf.get(f.id, 0.0),
                    )
                )
            queries.append(q)
        return queries

    def _signals(
        self,
        item: Entry,
        meta: LostMeta,
        f: Entry,
        fm: FoundMeta,
        owner_vecs: list[Vec],
        owner_clip: Vec | None,
        owner_e5: Vec,
        f_vecs: list[Vec],
        f_e5: Vec,
        tfidf: float,
    ) -> Signals:
        if owner_vecs:
            mode, visual = "image", _best(owner_vecs, f_vecs)
        elif owner_clip is not None:
            mode, visual = "cross_modal", _best([owner_clip], f_vecs)
        else:
            mode, visual = "none", None
        distance = None
        if not meta.unknown and meta.places:
            fp = self.locations[fm.place]
            distance = min(haversine_m(self.locations[p], fp) for p in meta.places)
        return Signals(
            found_id=f.id,
            mode=mode,
            visual=visual,
            semantic=float(np.dot(owner_e5, f_e5)),
            colour=fusion.colour_relation(item.colours, f.colours),
            brand=fusion.brand_relation(item.brand, f.brand),
            distance_m=distance,
            compatible=self.compat.get("|".join(sorted((meta.category, f.leaf)))),
            same_leaf=meta.category == f.leaf,
            true_pair=f.id == item.id,
            found_at=fm.found_at,
            lost_from=meta.lost_from,
            lost_until=meta.lost_until,
            tfidf=tfidf,
        )


class _Tfidf:
    def __init__(self, documents: list[str], ids: list[str]) -> None:
        from sklearn.feature_extraction.text import TfidfVectorizer

        self.ids = ids
        self.vectorizer = TfidfVectorizer(lowercase=True)
        self.matrix = self.vectorizer.fit_transform(documents)

    def scores(self, query: str) -> dict[str, float]:
        from sklearn.metrics.pairwise import cosine_similarity

        sims = cosine_similarity(self.vectorizer.transform([query]), self.matrix)[0]
        return {i: float(s) for i, s in zip(self.ids, sims, strict=True)}


@dataclass(frozen=True)
class Variant:
    """How a run turns signals into a ranking."""

    name: str
    config: Config
    visual: bool = True
    text: bool = True
    location: bool = True
    time: bool = True
    attributes: bool = True
    hard_category: bool = False
    tfidf: bool = False
    random_order: bool = False
    floor: bool = True


@dataclass
class Ranked:
    found_id: str
    overall: float
    tier: str
    true_pair: bool


def rank(
    variant: Variant, query: Query, parent_of: Callable[[str], str], found_leaf: dict[str, str], seed: int = 2026
) -> list[Ranked]:
    cfg = variant.config
    out: list[Ranked] = []
    for s in query.pool:
        if variant.hard_category and not s.same_leaf:
            continue
        category = (
            1.0
            if variant.hard_category
            else fusion.category_factor(
                cfg,
                same_leaf=s.same_leaf,
                same_parent=parent_of(query.meta.category) == parent_of(found_leaf[s.found_id]),
                compatible=s.compatible,
            )
        )
        pair = Pair(
            mode=s.mode if variant.visual else "none",  # type: ignore[arg-type]
            visual_cosine=s.visual if variant.visual else None,
            semantic_cosine=s.tfidf if variant.tfidf else (s.semantic if variant.text else None),
            colour=s.colour if variant.attributes and variant.text else "missing",
            brand=s.brand if variant.attributes and variant.text else "missing",
            location=fusion.location_term(cfg, s.distance_m) if variant.location and s.distance_m is not None else None,
            time=fusion.time_score(cfg, s.found_at, s.lost_from, s.lost_until) if variant.time else 1.0,
            category=category,
        )
        scored = fusion.score(cfg, pair)
        out.append(Ranked(s.found_id, scored.overall, scored.tier, s.true_pair))
    if variant.random_order:
        rng = random.Random(f"{seed}:{query.item.id}")  # noqa: S311
        rng.shuffle(out)
        return out[:POOL_TOP]
    kept = [r for r in out if not variant.floor or r.tier != fusion.NOT_STORED]
    kept.sort(key=lambda r: (-r.overall, r.found_id))
    return kept[:POOL_TOP]


def rank_of_true(ranked: list[Ranked]) -> int | None:
    for i, r in enumerate(ranked, start=1):
        if r.true_pair:
            return i
    return None


__all__ = ["Embedder", "Engine", "Variant", "load_compat", "rank", "rank_of_true"]
