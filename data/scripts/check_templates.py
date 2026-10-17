#!/usr/bin/env python3
"""Validate blank private collection templates before data entry begins."""

from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

from create_eval_templates import (
    DISTRACTOR_FIELDS,
    DUPLICATE_FIELDS,
    ITEM_FIELDS,
    LABEL_FIELDS,
    QUERY_FIELDS,
)


PROFILE_FILES = {
    "trial": (("items.csv", ITEM_FIELDS), ("labels.csv", LABEL_FIELDS)),
    "full": (
        ("items.csv", ITEM_FIELDS),
        ("labels.csv", LABEL_FIELDS),
        ("distractors.csv", DISTRACTOR_FIELDS),
        ("duplicates.csv", DUPLICATE_FIELDS),
        ("queries.csv", QUERY_FIELDS),
    ),
}
PROFILE_IDS = {
    "items.csv": "I",
    "labels.csv": "I",
    "distractors.csv": "D",
    "duplicates.csv": "P",
    "queries.csv": "Q",
}


def expected_ids(filename: str, count: int) -> list[str]:
    return [f"{PROFILE_IDS[filename]}{index:03d}" for index in range(1, count + 1)]


def validate_csv_template(path: Path, fields: tuple[str, ...], count: int) -> list[str]:
    errors: list[str] = []
    if not path.is_file() or path.is_symlink():
        return [f"missing or invalid template file: {path.name}"]
    try:
        with path.open(newline="", encoding="utf-8") as handle:
            reader = csv.DictReader(handle)
            if tuple(reader.fieldnames or ()) != fields:
                return [f"{path.name}: expected columns {fields}, got {tuple(reader.fieldnames or ())}"]
            rows = list(reader)
    except (OSError, csv.Error) as exc:
        return [f"{path.name}: cannot read template: {exc}"]

    ids = [row.get(fields[0], "") for row in rows]
    if ids != expected_ids(path.name, count):
        errors.append(f"{path.name}: IDs must run from {expected_ids(path.name, count)[0]} through {expected_ids(path.name, count)[-1]} in order")
    for index, row in enumerate(rows, start=1):
        if any(row.get(field, "") for field in fields[1:]):
            errors.append(f"{path.name}: row {index} must be blank apart from its stable ID")
    return errors


def validate_template_dir(path: Path, profile: str, counts: dict[str, int] | None = None) -> list[str]:
    if profile not in PROFILE_FILES:
        return [f"unknown template profile: {profile}"]
    if path.is_symlink() or not path.is_dir():
        return [f"template directory does not exist or is a symlink: {path}"]

    default_counts = {
        "items.csv": 20 if profile == "trial" else 150,
        "labels.csv": 20 if profile == "trial" else 150,
        "distractors.csv": 400,
        "duplicates.csv": 40,
        "queries.csv": 50,
    }
    if counts:
        default_counts.update(counts)
    expected_files = {filename for filename, _ in PROFILE_FILES[profile]}
    errors: list[str] = []
    for filename, fields in PROFILE_FILES[profile]:
        errors.extend(validate_csv_template(path / filename, fields, default_counts[filename]))

    unexpected = sorted(
        child.name
        for child in path.iterdir()
        if child.name != "images" and child.name not in expected_files
    )
    errors.extend(f"unexpected template entry: {name}" for name in unexpected)

    images = path / "images"
    if images.is_symlink() or not images.is_dir():
        errors.append("images/: missing or symlink")
    else:
        for child in images.rglob("*"):
            relative = child.relative_to(path).as_posix()
            errors.append(f"template images/ must be empty before data entry: {relative}")
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, required=True, help="private raw template directory")
    parser.add_argument("--profile", choices=tuple(PROFILE_FILES), required=True)
    parser.add_argument("--items", type=int, help="override item count for a local fixture")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    counts = None
    if args.items is not None:
        counts = {"items.csv": args.items, "labels.csv": args.items}
    errors = validate_template_dir(args.input, args.profile, counts)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Template check passed: {args.input}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
