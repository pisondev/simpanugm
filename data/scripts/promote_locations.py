#!/usr/bin/env python3
"""Promote approved OpenStreetMap review rows into the locations seed."""

from __future__ import annotations

import argparse
import csv
import math
import sys
from pathlib import Path

from check import SLUG_RE, ValidationError, read_csv
from osm_locations import REVIEW_FIELDS, parse_bbox


LOCATION_FIELDS = ("slug", "name", "zone_slug", "kind", "lat", "lng", "aliases")
ZONE_FIELDS = ("slug", "name", "centroid_lat", "centroid_lng")
KINDS = {"building", "outdoor", "facility", "transport", "parking"}


def read_review(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        if tuple(reader.fieldnames or ()) != REVIEW_FIELDS:
            raise ValidationError(f"{path}: expected columns {REVIEW_FIELDS}")
        rows = list(reader)
    if not rows:
        raise ValidationError(f"{path}: review file must contain at least one row")
    return rows


def read_zone_slugs(path: Path) -> set[str]:
    rows = read_csv(path, ZONE_FIELDS)
    slugs = {row["slug"].strip() for row in rows}
    if any(not slug or not SLUG_RE.fullmatch(slug) for slug in slugs):
        raise ValidationError(f"{path}: zone slugs must be lowercase hyphenated slugs")
    if len(slugs) != len(rows):
        raise ValidationError(f"{path}: zone slugs must be unique")
    return slugs


def parse_coordinate(value: str, field: str) -> float:
    try:
        result = float(value)
    except ValueError as exc:
        raise ValidationError(f"{field} must be numeric: {value!r}") from exc
    if not math.isfinite(result):
        raise ValidationError(f"{field} must be finite: {value!r}")
    return result


def normalize_aliases(value: str, name: str) -> str:
    aliases: list[str] = []
    seen: set[str] = {name.casefold()}
    for raw_alias in value.split(";"):
        alias = " ".join(raw_alias.split())
        if alias and alias.casefold() not in seen:
            aliases.append(alias)
            seen.add(alias.casefold())
    return ";".join(aliases)


def promote_rows(
    rows: list[dict[str, str]],
    zones: set[str],
    bbox: tuple[float, float, float, float],
) -> tuple[list[dict[str, str]], list[str]]:
    south, west, north, east = bbox
    errors: list[str] = []
    output: list[dict[str, str]] = []
    seen_slugs: set[str] = set()
    for row in rows:
        status = row["review_status"].strip().casefold()
        slug = row["slug"].strip()
        if status == "rejected":
            continue
        if status != "approved":
            errors.append(f"{slug or '<blank>'}: review_status must be approved or rejected")
            continue
        name = " ".join(row["name"].split())
        zone_slug = row["zone_slug"].strip()
        kind = row["kind"].strip()
        try:
            lat = parse_coordinate(row["lat"], f"{slug}.lat")
            lng = parse_coordinate(row["lng"], f"{slug}.lng")
        except ValidationError as exc:
            errors.append(str(exc))
            continue
        if not slug or not SLUG_RE.fullmatch(slug):
            errors.append(f"{slug or '<blank>'}: invalid slug")
        if slug in seen_slugs:
            errors.append(f"{slug}: duplicate slug")
        seen_slugs.add(slug)
        if not name:
            errors.append(f"{slug or '<blank>'}: name cannot be blank")
        if zone_slug not in zones:
            errors.append(f"{slug or '<blank>'}: unknown zone_slug {zone_slug!r}")
        if kind not in KINDS:
            errors.append(f"{slug or '<blank>'}: invalid kind {kind!r}")
        if not south <= lat <= north or not west <= lng <= east:
            errors.append(f"{slug or '<blank>'}: coordinates fall outside the campus bbox")
        output.append(
            {
                "slug": slug,
                "name": name,
                "zone_slug": zone_slug,
                "kind": kind,
                "lat": f"{lat:.7f}",
                "lng": f"{lng:.7f}",
                "aliases": normalize_aliases(row["aliases"], name),
            }
        )
    if errors:
        return [], errors
    return sorted(output, key=lambda row: row["slug"]), []


def write_locations(path: Path, rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=LOCATION_FIELDS)
        writer.writeheader()
        writer.writerows(rows)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, required=True, help="human-reviewed locations CSV")
    parser.add_argument("--zones", type=Path, required=True, help="approved zones.csv")
    parser.add_argument("--output", type=Path, required=True, help="locations.csv to write")
    parser.add_argument("--bbox", type=parse_bbox, required=True, help="south,west,north,east")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        rows = read_review(args.input)
        zones = read_zone_slugs(args.zones)
        promoted, errors = promote_rows(rows, zones, args.bbox)
        if errors:
            for error in errors:
                print(f"ERROR: {error}", file=sys.stderr)
            return 1
        write_locations(args.output, promoted)
    except (OSError, ValidationError, ValueError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Promoted {len(promoted)} approved locations to {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
