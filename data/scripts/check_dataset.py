#!/usr/bin/env python3
"""Run quality gates on a prepared private evaluation dataset."""

from __future__ import annotations

import argparse
import csv
import hashlib
import re
import sys
import unicodedata
from pathlib import Path

from check_manifest import check_manifest
from check_trial import validate_trial


IMAGE_RE = re.compile(r"^(finder|owner)_[0-9]+\.(jpg|jpeg|png|webp)$", re.IGNORECASE)
ITEM_FIELDS = ("item_id", "item_name", "owner_description", "finder_description")


def normalize_text(value: str) -> str:
    return " ".join(unicodedata.normalize("NFKC", value).casefold().split())


def text_digest(value: str) -> str:
    return hashlib.sha256(normalize_text(value).encode("utf-8")).hexdigest()


def read_items(path: Path) -> tuple[list[dict[str, str]], list[str]]:
    if not path.is_file():
        return [], [f"missing metadata file: {path.name}"]
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        fields = tuple(reader.fieldnames or ())
        missing = [field for field in ITEM_FIELDS if field not in fields]
        if missing:
            return [], [f"items.csv missing columns: {', '.join(missing)}"]
        rows = list(reader)
    return rows, []


def read_labels(path: Path) -> tuple[list[dict[str, str]], list[str]]:
    if not path.is_file():
        return [], [f"missing metadata file: {path.name}"]
    with path.open(newline="", encoding="utf-8") as handle:
        rows = list(csv.DictReader(handle))
    return rows, []


def image_files(dataset_dir: Path, item_id: str, kind: str) -> list[Path]:
    item_dir = dataset_dir / "images" / item_id
    if not item_dir.is_dir():
        return []
    return sorted(
        path
        for path in item_dir.iterdir()
        if path.is_file() and IMAGE_RE.fullmatch(path.name) and path.name.casefold().startswith(f"{kind}_")
    )


def validate_dataset(dataset_dir: Path, seed_dir: Path, expected_count: int = 20) -> list[str]:
    errors = check_manifest(dataset_dir)
    items, item_errors = read_items(dataset_dir / "items.csv")
    labels, label_errors = read_labels(dataset_dir / "labels.csv")
    errors.extend(item_errors)
    errors.extend(label_errors)
    if item_errors or label_errors:
        return errors

    expected_ids = {f"I{index:03d}" for index in range(1, expected_count + 1)}
    item_ids = [row.get("item_id", "") for row in items]
    label_ids = [row.get("item_id", "") for row in labels]
    if set(item_ids) != expected_ids or len(item_ids) != expected_count:
        errors.append("items.csv must contain each expected item exactly once")
    if len(set(item_ids)) != len(item_ids):
        errors.append("items.csv contains duplicate item_id values")
    if set(label_ids) != expected_ids or len(label_ids) != expected_count:
        errors.append("labels.csv must contain each expected item exactly once")
    if len(set(label_ids)) != len(label_ids):
        errors.append("labels.csv contains duplicate item_id values")

    errors.extend(validate_trial(dataset_dir / "labels.csv", seed_dir, expected_count))
    label_by_id = {row.get("item_id", ""): row for row in labels}
    description_sources: dict[str, tuple[str, str]] = {}
    item_by_id = {row.get("item_id", ""): row for row in items}
    for item_id in sorted(expected_ids):
        item = item_by_id.get(item_id)
        label = label_by_id.get(item_id)
        if item is None or label is None:
            continue
        item_name = item.get("item_name", "").strip()
        owner_description = item.get("owner_description", "").strip()
        finder_description = item.get("finder_description", "").strip()
        if not 3 <= len(item_name) <= 80:
            errors.append(f"{item_id}: item_name must be 3 to 80 characters")
        if not 20 <= len(owner_description) <= 1000:
            errors.append(f"{item_id}: owner_description must be 20 to 1000 characters")
        if not 10 <= len(finder_description) <= 500:
            errors.append(f"{item_id}: finder_description must be 10 to 500 characters")
        if owner_description and finder_description and normalize_text(owner_description) == normalize_text(finder_description):
            errors.append(f"{item_id}: owner and finder descriptions must not be identical")
        for field, description in (
            ("owner_description", owner_description),
            ("finder_description", finder_description),
        ):
            if not description:
                continue
            digest = text_digest(description)
            previous = description_sources.get(digest)
            if previous is not None and previous[0] != item_id:
                errors.append(f"{item_id}: {field} duplicates {previous[0]} ({previous[1]})")
            else:
                description_sources[digest] = (item_id, field)

        finder_images = image_files(dataset_dir, item_id, "finder")
        owner_images = image_files(dataset_dir, item_id, "owner")
        if len(finder_images) < 2:
            errors.append(f"{item_id}: at least two finder photos are required")
        owner_photo_expected = label.get("has_owner_photo") == "true"
        if owner_photo_expected and len(owner_images) != 1:
            errors.append(f"{item_id}: has_owner_photo=true requires exactly one owner photo")
        if not owner_photo_expected and owner_images:
            errors.append(f"{item_id}: has_owner_photo=false must not have an owner photo")

    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset-dir", type=Path, required=True)
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--expected-count", type=int, default=20)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_dataset(args.dataset_dir, args.seed_dir, args.expected_count)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Dataset quality check passed: {args.dataset_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
