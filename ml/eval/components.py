"""Component tasks T4 to T8 of evaluation.md section 1, measured on the same dataset as the matching tasks."""

from __future__ import annotations

import csv
from pathlib import Path

import numpy as np
import yaml
from PIL import Image

from app.analysis.brand import BrandMatcher, load_brands
from app.analysis.category import CategoryPrompts, ZeroShotClassifier
from app.analysis.colour import ColourExtractor, load_palette
from app.backend import Backend
from eval import text
from eval.dataset import Dataset, Entry
from eval.engine import Embedder

DUP_FLAG = 0.80
DUP_IMAGE_WEIGHT = 0.6
DUP_PHASH_BITS = 6


def category_prompts(seed_dir: Path) -> list[CategoryPrompts]:
    document = yaml.safe_load((seed_dir / "categories.yaml").read_text(encoding="utf-8"))
    return [
        CategoryPrompts(c["slug"], list(c["clip_prompts"])) for p in document["parents"] for c in p.get("children", [])
    ]


def t4_t5_t6(dataset: Dataset, split: str, backend: Backend, seed_dir: Path) -> dict[str, float]:
    """Category top 3 accuracy, colour agreement, and brand precision and recall, on the first finder photo."""
    classifier = ZeroShotClassifier(backend.embed_prompts)
    cats = category_prompts(seed_dir)
    colours = ColourExtractor(load_palette(seed_dir / "colours.csv"))
    brands = BrandMatcher(load_brands(seed_dir / "brands.csv"))
    top3 = colour_hits = n = 0
    brand_tp = brand_fp = brand_fn = 0
    for e in dataset.items(split):
        if not e.finder_images:
            continue
        image = Image.open(e.finder_images[0]).convert("RGB")
        vector = backend.embed_images([image])[0]
        n += 1
        top3 += any(s.slug == e.leaf for s in classifier.classify(vector, "eval", cats))
        named = colours.extract(image)
        colour_hits += bool(named and e.colours and named[0].name == e.colours[0])
        found = {b.name.lower() for b in brands.match(" ".join(line.text for line in backend.read_text(image)))}
        truth = {e.brand.lower()} if e.brand else set()
        brand_tp += len(found & truth)
        brand_fp += len(found - truth)
        brand_fn += len(truth - found)
    return {
        "n": float(n),
        "t4_top3": top3 / n if n else 0.0,
        "t5_colour": colour_hits / n if n else 0.0,
        "t6_precision": brand_tp / (brand_tp + brand_fp) if brand_tp + brand_fp else 0.0,
        "t6_recall": brand_tp / (brand_tp + brand_fn) if brand_tp + brand_fn else 0.0,
    }


def _phash(path: Path) -> int:
    """An 8x8 average hash, the cheap perceptual hash the duplicate gate of section 8 compares by Hamming distance."""
    small = np.asarray(Image.open(path).convert("L").resize((8, 8), Image.Resampling.LANCZOS), dtype=np.float64)
    bits = (small > small.mean()).flatten()
    return int("".join("1" if b else "0" for b in bits), 2)


def s_dup(embedder: Embedder, a: Entry, b: Entry, synonyms: dict[str, str]) -> float:
    ha = [_phash(p) for p in a.finder_images]
    hb = [_phash(p) for p in b.finder_images]
    if any(bin(x ^ y).count("1") <= DUP_PHASH_BITS for x in ha for y in hb):
        return 1.0
    va = [embedder.image(p) for p in a.finder_images]
    vb = [embedder.image(p) for p in b.finder_images]
    cos = max(float(np.dot(x, y)) for x in va for y in vb)
    sa = embedder.e5(
        text.e5_text(text.TextInput(a.name, a.leaf, a.parent, a.colours, a.brand, a.finder_description), synonyms)
    )
    sb = embedder.e5(
        text.e5_text(text.TextInput(b.name, b.leaf, b.parent, b.colours, b.brand, b.finder_description), synonyms)
    )
    band_lo, band_hi = 0.60, 0.95  # the image band of the fusion document
    image = min(1.0, max(0.0, (cos - band_lo) / (band_hi - band_lo)))
    sem = min(1.0, max(0.0, (float(np.dot(sa, sb)) - 0.78) / (0.93 - 0.78)))
    return DUP_IMAGE_WEIGHT * image + (1 - DUP_IMAGE_WEIGHT) * sem


def t7(dataset: Dataset, embedder: Embedder, synonyms: dict[str, str]) -> dict[str, float]:
    """Precision and recall at the flag over duplicate pairs and same parent negatives."""
    positives = [
        (dataset.entries[a], dataset.entries[b])
        for a, b in dataset.duplicates
        if a in dataset.entries and b in dataset.entries
    ]
    flagged_true = sum(s_dup(embedder, a, b, synonyms) >= DUP_FLAG for a, b in positives)
    known = {frozenset(p) for p in dataset.duplicates}
    negatives: list[tuple[Entry, Entry]] = []
    pool = sorted(dataset.entries.values(), key=lambda e: e.id)
    for i, a in enumerate(pool):
        for b in pool[i + 1 : i + 6]:
            if a.parent == b.parent and frozenset((a.id, b.id)) not in known and a.finder_images and b.finder_images:
                negatives.append((a, b))
    flagged_false = sum(s_dup(embedder, a, b, synonyms) >= DUP_FLAG for a, b in negatives)
    flagged = flagged_true + flagged_false
    return {
        "pairs": float(len(positives)),
        "negatives": float(len(negatives)),
        "t7_precision": flagged_true / flagged if flagged else 0.0,
        "t7_recall": flagged_true / len(positives) if positives else 0.0,
    }


def t8(dataset: Dataset, embedder: Embedder, synonyms: dict[str, str]) -> dict[str, float]:
    """Recall@5 of semantic search: the paraphrase query against the found report vectors."""
    reports = sorted(dataset.entries.values(), key=lambda e: e.id)
    vectors = {
        e.id: embedder.e5(
            text.e5_text(text.TextInput(e.name, e.leaf, e.parent, e.colours, e.brand, e.finder_description), synonyms)
        )
        for e in reports
    }
    hits = n = 0
    for query, target in dataset.queries:
        if target not in vectors:
            continue
        q = embedder.e5("query: " + query)
        ranked = sorted(vectors, key=lambda i: (-float(np.dot(q, vectors[i])), i))[:5]
        hits += target in ranked
        n += 1
    return {"queries": float(n), "t8_recall@5": hits / n if n else 0.0}


def read_csv_rows(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        return list(csv.DictReader(handle))
