#!/usr/bin/env python3
"""Validate the label sheet for the 20-item dataset trial."""

from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

from check import ValidationError, read_csv, validate_categories


FIELDS = (
    "item_id",
    "leaf_category",
    "primary_colour",
    "secondary_colour",
    "brand",
    "distinctiveness",
    "has_owner_photo",
    "flags",
)
ALLOWED_DISTINCTIVENESS = {"generic", "distinctive"}
ALLOWED_FLAGS = {"low_light", "small_in_frame", "damaged"}


def read_palette(seed_dir: Path) -> set[str]:
    with (seed_dir / "colours.csv").open(newline="", encoding="utf-8") as handle:
        return {row["name"] for row in csv.DictReader(handle)}


def validate_trial(path: Path, seed_dir: Path, expected_count: int = 20) -> list[str]:
    errors: list[str] = []
    try:
        rows = read_csv(path, FIELDS)
        leaves = validate_categories(seed_dir / "categories.yaml")
        colours = read_palette(seed_dir)
    except (OSError, KeyError, ValidationError) as exc:
        return [str(exc)]

    expected_ids = [f"I{index:03d}" for index in range(1, expected_count + 1)]
    ids = [row["item_id"] for row in rows]
    if ids != expected_ids:
        errors.append(f"{path}: item_id must contain {expected_ids[0]} through {expected_ids[-1]} in order")
    for row in rows:
        item_id = row["item_id"] or "<blank>"
        if row["leaf_category"] not in leaves:
            errors.append(f"{path}: {item_id} has an unknown leaf_category")
        if row["primary_colour"] not in colours:
            errors.append(f"{path}: {item_id} must have a valid primary_colour")
        if row["secondary_colour"] and row["secondary_colour"] not in colours:
            errors.append(f"{path}: {item_id} has an invalid secondary_colour")
        if row["distinctiveness"] not in ALLOWED_DISTINCTIVENESS:
            errors.append(f"{path}: {item_id} has an invalid distinctiveness value")
        if row["has_owner_photo"] not in {"true", "false"}:
            errors.append(f"{path}: {item_id} has_owner_photo must be true or false")
        flags = {flag for flag in row["flags"].split(";") if flag}
        if not flags <= ALLOWED_FLAGS:
            errors.append(f"{path}: {item_id} has an unknown flag")
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, required=True, help="completed trial label CSV")
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--expected-count", type=int, default=20)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_trial(args.input, args.seed_dir, expected_count=args.expected_count)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Trial labels passed: {args.input}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
