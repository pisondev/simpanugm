#!/usr/bin/env python3
"""Summarize a private OSM location-review CSV without copying its names."""

from __future__ import annotations

import argparse
import os
import sys
import tempfile
from collections import Counter
from pathlib import Path
from typing import Iterable

from check_location_review import read_review


def summarize_rows(rows: Iterable[dict[str, str]]) -> dict[str, object]:
    rows = list(rows)
    if not rows:
        raise ValueError("review CSV must contain at least one candidate")
    try:
        latitudes = [float(row["lat"]) for row in rows]
        longitudes = [float(row["lng"]) for row in rows]
    except (KeyError, ValueError) as exc:
        raise ValueError(f"review CSV coordinates must be numeric: {exc}") from exc
    return {
        "candidate_count": len(rows),
        "by_kind": dict(sorted(Counter(row["kind"] for row in rows).items())),
        "by_status": dict(sorted(Counter(row["review_status"] for row in rows).items())),
        "by_osm_type": dict(sorted(Counter(row["osm_type"] for row in rows).items())),
        "latitude_range": (min(latitudes), max(latitudes)),
        "longitude_range": (min(longitudes), max(longitudes)),
        "empty_zone_count": sum(not row["zone_slug"] for row in rows),
        "empty_alias_count": sum(not row["aliases"] for row in rows),
    }


def render_summary(summary: dict[str, object]) -> str:
    def bullets(values: object) -> str:
        return "\n".join(f"- `{key}`: {value}" for key, value in dict(values).items())

    latitude_range = summary["latitude_range"]
    longitude_range = summary["longitude_range"]
    return f"""# Location review summary

This summary is generated from the private OSM review CSV. It contains counts only; the
candidate names and raw response remain in the private review workspace.

## Counts

- Candidates: **{summary['candidate_count']}**
- Empty `zone_slug` values awaiting review: **{summary['empty_zone_count']}**
- Empty `aliases` values awaiting review: **{summary['empty_alias_count']}**

### By kind

{bullets(summary['by_kind'])}

### By review status

{bullets(summary['by_status'])}

### By OSM element type

{bullets(summary['by_osm_type'])}

## Coordinate range

- Latitude: `{latitude_range[0]:.7f}` to `{latitude_range[1]:.7f}`
- Longitude: `{longitude_range[0]:.7f}` to `{longitude_range[1]:.7f}`

Every candidate remains `needs_review` until a human confirms its official name, aliases,
zone, kind, and entrance coordinate. This file is a review aid, not a production seed.
"""


def write_summary(path: Path, content: str) -> None:
    if path.is_symlink():
        raise OSError(f"output must not be a symlink: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            prefix=f".{path.name}.",
            suffix=".tmp",
            dir=path.parent,
            delete=False,
        ) as handle:
            temporary_path = Path(handle.name)
            handle.write(content)
        os.replace(temporary_path, path)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--review", type=Path, required=True, help="private location-review CSV")
    parser.add_argument("--output", type=Path, help="optional private Markdown summary path")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        summary = render_summary(summarize_rows(read_review(args.review)))
        if args.output is None:
            print(summary, end="")
        else:
            write_summary(args.output, summary)
            print(f"Wrote location review summary: {args.output}")
    except (OSError, ValueError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
