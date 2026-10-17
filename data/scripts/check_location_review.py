#!/usr/bin/env python3
"""Check that an unreviewed location CSV still matches its raw OSM response."""

from __future__ import annotations

import argparse
import csv
import json
import math
import sys
from pathlib import Path
from typing import Any

from osm_locations import REVIEW_FIELDS, parse_bbox, review_rows


def read_review(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        if tuple(reader.fieldnames or ()) != REVIEW_FIELDS:
            raise ValueError(f"review CSV must have columns {REVIEW_FIELDS}")
        return list(reader)


def validate_review(
    raw_path: Path,
    review_path: Path,
    bbox: tuple[float, float, float, float] | None = None,
    require_pending: bool = True,
) -> list[str]:
    try:
        with raw_path.open(encoding="utf-8") as handle:
            raw_document: Any = json.load(handle)
    except (OSError, json.JSONDecodeError) as exc:
        return [f"cannot read raw Overpass response: {exc}"]

    try:
        expected_rows = review_rows(raw_document, bbox)
        actual_rows = read_review(review_path)
    except (OSError, csv.Error, ValueError) as exc:
        return [f"cannot read location review: {exc}"]

    errors: list[str] = []
    expected_by_key = {(row["osm_type"], row["osm_id"]): row for row in expected_rows}
    actual_by_key: dict[tuple[str, str], dict[str, str]] = {}
    seen_slugs: set[str] = set()
    south, west, north, east = bbox if bbox is not None else (None, None, None, None)

    for row in actual_rows:
        key = (row["osm_type"], row["osm_id"])
        if key in actual_by_key:
            errors.append(f"duplicate review OSM element: {key[0]}/{key[1]}")
            continue
        actual_by_key[key] = row
        if row["slug"] in seen_slugs:
            errors.append(f"duplicate review slug: {row['slug']}")
        seen_slugs.add(row["slug"])
        if require_pending:
            if row["review_status"] != "needs_review":
                errors.append(f"{row['slug']}: review_status must remain needs_review")
            if row["zone_slug"] or row["aliases"]:
                errors.append(f"{row['slug']}: zone_slug and aliases must remain empty before review")
        expected = expected_by_key.get(key)
        if expected is None:
            errors.append(f"{row['slug']}: OSM element {key[0]}/{key[1]} is not in the raw response")
            continue
        for field in ("slug", "name", "kind", "lat", "lng"):
            if row[field] != expected[field]:
                errors.append(
                    f"{row['slug']}: {field} differs from raw extraction "
                    f"({row[field]!r} != {expected[field]!r})"
                )
        if bbox is not None:
            try:
                lat = float(row["lat"])
                lng = float(row["lng"])
            except ValueError:
                errors.append(f"{row['slug']}: coordinates must be numeric")
            else:
                if not math.isfinite(lat) or not math.isfinite(lng) or not (south <= lat <= north and west <= lng <= east):
                    errors.append(f"{row['slug']}: coordinates fall outside the review bbox")

    expected_keys = set(expected_by_key)
    actual_keys = set(actual_by_key)
    for key in sorted(expected_keys - actual_keys):
        errors.append(f"missing review OSM element: {key[0]}/{key[1]}")
    return errors


def validate_provenance(
    raw_path: Path,
    review_path: Path,
    bbox: tuple[float, float, float, float] | None = None,
) -> list[str]:
    """Validate OSM-derived fields while allowing human review fields to change."""
    return validate_review(raw_path, review_path, bbox, require_pending=False)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--raw", type=Path, required=True, help="saved raw Overpass JSON")
    parser.add_argument("--review", type=Path, required=True, help="generated review CSV")
    parser.add_argument("--bbox", type=parse_bbox, help="optional south,west,north,east check")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_review(args.raw, args.review, args.bbox)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Location review check passed: {args.review}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
