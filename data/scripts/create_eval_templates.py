#!/usr/bin/env python3
"""Create blank private CSV templates for the full evaluation dataset."""

from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path


ITEM_FIELDS = ("item_id", "item_name", "owner_description", "finder_description")
LABEL_FIELDS = (
    "item_id",
    "leaf_category",
    "primary_colour",
    "secondary_colour",
    "brand",
    "distinctiveness",
    "has_owner_photo",
    "flags",
)
DISTRACTOR_FIELDS = (
    "distractor_id",
    "leaf_category",
    "primary_colour",
    "distinctiveness",
    "description",
)
DUPLICATE_FIELDS = ("pair_id", "item_id", "first_report_id", "second_report_id")
QUERY_FIELDS = ("query_id", "query_text", "target_report_id")


class TemplateError(ValueError):
    """A safe, actionable template-generation failure."""


def write_template(path: Path, fields: tuple[str, ...], rows: list[dict[str, str]]) -> None:
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)


def blank_rows(prefix: str, count: int, fields: tuple[str, ...]) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    id_field = fields[0]
    for index in range(1, count + 1):
        rows.append({field: (f"{prefix}{index:03d}" if field == id_field else "") for field in fields})
    return rows


def create_templates(
    output: Path,
    item_count: int = 150,
    distractor_count: int = 400,
    duplicate_count: int = 40,
    query_count: int = 50,
) -> list[Path]:
    counts = {
        "items": item_count,
        "distractors": distractor_count,
        "duplicates": duplicate_count,
        "queries": query_count,
    }
    if any(count < 1 for count in counts.values()):
        raise TemplateError("all template counts must be positive")
    if output.exists() and any(output.iterdir()):
        raise TemplateError(f"output must be empty or absent: {output}")
    output.mkdir(parents=True, exist_ok=True)

    templates = {
        "items.csv": (ITEM_FIELDS, blank_rows("I", item_count, ITEM_FIELDS)),
        "labels.csv": (LABEL_FIELDS, blank_rows("I", item_count, LABEL_FIELDS)),
        "distractors.csv": (DISTRACTOR_FIELDS, blank_rows("D", distractor_count, DISTRACTOR_FIELDS)),
        "duplicates.csv": (DUPLICATE_FIELDS, blank_rows("P", duplicate_count, DUPLICATE_FIELDS)),
        "queries.csv": (QUERY_FIELDS, blank_rows("Q", query_count, QUERY_FIELDS)),
    }
    created: list[Path] = []
    for filename, (fields, rows) in templates.items():
        path = output / filename
        write_template(path, fields, rows)
        created.append(path)
    (output / "images").mkdir()
    return created


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True, help="private raw dataset directory")
    parser.add_argument("--items", type=int, default=150)
    parser.add_argument("--distractors", type=int, default=400)
    parser.add_argument("--duplicates", type=int, default=40)
    parser.add_argument("--queries", type=int, default=50)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        created = create_templates(
            args.output,
            item_count=args.items,
            distractor_count=args.distractors,
            duplicate_count=args.duplicates,
            query_count=args.queries,
        )
    except (OSError, TemplateError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Created {len(created)} private evaluation templates in {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
