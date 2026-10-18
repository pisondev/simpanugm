"""Reads the prepared simpanugm-eval directory of dataset-collection.md section 6."""

from __future__ import annotations

import csv
import hashlib
from dataclasses import dataclass, field
from pathlib import Path

import yaml

IMAGE_SUFFIXES = {".jpg", ".jpeg", ".png", ".webp"}


@dataclass
class Entry:
    """One physical item or one distractor. Distractors only exist as found reports."""

    id: str
    split: str
    leaf: str
    parent: str
    colours: list[str]
    brand: str
    distinctiveness: str
    name: str = ""
    owner_description: str = ""
    finder_description: str = ""
    has_owner_photo: bool = False
    finder_images: list[Path] = field(default_factory=list)
    owner_images: list[Path] = field(default_factory=list)

    @property
    def is_item(self) -> bool:
        return self.id.startswith("I")


@dataclass
class Dataset:
    root: Path
    version: str
    entries: dict[str, Entry]
    duplicates: list[tuple[str, str]]
    queries: list[tuple[str, str]]

    def split(self, name: str) -> list[Entry]:
        return sorted((e for e in self.entries.values() if e.split == name), key=lambda e: e.id)

    def items(self, name: str) -> list[Entry]:
        return [e for e in self.split(name) if e.is_item]


def _rows(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        return list(csv.DictReader(handle))


def leaf_parents(seed_dir: Path) -> dict[str, str]:
    document = yaml.safe_load((seed_dir / "categories.yaml").read_text(encoding="utf-8"))
    return {c["slug"]: p["slug"] for p in document["parents"] for c in p.get("children", [])}


def _images(root: Path, entry_id: str, kind: str) -> list[Path]:
    folder = root / "images" / entry_id
    return sorted(p for p in folder.glob(f"{kind}_*") if p.suffix.lower() in IMAGE_SUFFIXES) if folder.is_dir() else []


def _colours(row: dict[str, str]) -> list[str]:
    return [c for c in (row.get("primary_colour", ""), row.get("secondary_colour", "")) if c]


def load(root: Path, seed_dir: Path) -> Dataset:
    parents = leaf_parents(seed_dir)
    items = {r["item_id"]: r for r in _rows(root / "items.csv")}
    labels = {r["item_id"]: r for r in _rows(root / "labels.csv")}
    splits = {r["item_id"]: r["split"] for r in _rows(root / "splits.csv")}
    entries: dict[str, Entry] = {}
    for item_id, row in sorted(items.items()):
        label = labels[item_id]
        entries[item_id] = Entry(
            id=item_id,
            split=splits[item_id],
            leaf=label["leaf_category"],
            parent=parents[label["leaf_category"]],
            colours=_colours(label),
            brand=label.get("brand", ""),
            distinctiveness=label["distinctiveness"],
            name=row["item_name"],
            owner_description=row["owner_description"],
            finder_description=row["finder_description"],
            has_owner_photo=label.get("has_owner_photo", "").strip().lower() in {"1", "true", "yes", "ya"},
            finder_images=_images(root, item_id, "finder"),
            owner_images=_images(root, item_id, "owner"),
        )
    distractor_split = (
        {r["distractor_id"]: r["split"] for r in _rows(root / "distractor-splits.csv")}
        if (root / "distractor-splits.csv").is_file()
        else {}
    )
    distractors = root / "distractors.csv"
    for row in _rows(distractors) if distractors.is_file() else []:
        did = row["distractor_id"]
        entries[did] = Entry(
            id=did,
            split=distractor_split[did],
            leaf=row["leaf_category"],
            parent=parents[row["leaf_category"]],
            colours=[row["primary_colour"]] if row.get("primary_colour") else [],
            brand="",
            distinctiveness=row.get("distinctiveness", "generic"),
            name=row["leaf_category"],
            finder_description=row["description"],
            finder_images=_images(root, did, "finder"),
        )
    duplicates = (
        [(r["first_report_id"], r["second_report_id"]) for r in _rows(root / "duplicates.csv")]
        if (root / "duplicates.csv").is_file()
        else []
    )
    queries = (
        [(r["query_text"], r["target_report_id"]) for r in _rows(root / "queries.csv")]
        if (root / "queries.csv").is_file()
        else []
    )
    manifest = root / "manifest.json"
    version = hashlib.sha256(manifest.read_bytes()).hexdigest() if manifest.is_file() else "unversioned"
    return Dataset(root, version, entries, duplicates, queries)


def leakage(dataset: Dataset) -> list[str]:
    """NFR-REPRO-04: no item id, image hash or description hash may appear in two splits."""
    seen: dict[tuple[str, str], str] = {}
    problems: list[str] = []

    def note(kind: str, key: str, entry: Entry) -> None:
        other = seen.setdefault((kind, key), entry.split)
        if other != entry.split:
            problems.append(f"{kind} {key[:16]} of {entry.id} appears in {other} and {entry.split}")

    for e in dataset.entries.values():
        note("id", e.id, e)
        for path in [*e.finder_images, *e.owner_images]:
            note("image", hashlib.sha256(path.read_bytes()).hexdigest(), e)
        for description in (e.owner_description, e.finder_description):
            if description.strip():
                note("description", hashlib.sha256(" ".join(description.casefold().split()).encode()).hexdigest(), e)
    return problems
