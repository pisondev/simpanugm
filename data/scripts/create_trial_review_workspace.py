#!/usr/bin/env python3
"""Create a blank private workspace for the 20-item trial review."""

from __future__ import annotations

import argparse
import csv
import sys
import tempfile
from pathlib import Path

from check_agreement import ANNOTATION_FIELDS
from path_safety import first_symlink_component


REVIEW_LOG_FIELDS = (
    "item_id",
    "consent_status",
    "photo_privacy_status",
    "photo_quality_status",
    "retake_reason",
    "category_disagreement",
    "distinctiveness_disagreement",
    "review_status",
    "private_drive_path",
)


class ReviewWorkspaceError(ValueError):
    """A safe, actionable review-workspace generation failure."""


def blank_rows(count: int, fields: tuple[str, ...]) -> list[dict[str, str]]:
    if count < 1:
        raise ReviewWorkspaceError("item count must be positive")
    id_field = fields[0]
    return [
        {field: (f"I{index:03d}" if field == id_field else "") for field in fields}
        for index in range(1, count + 1)
    ]


def write_csv(path: Path, fields: tuple[str, ...], rows: list[dict[str, str]]) -> None:
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)


def create_review_workspace(output: Path, item_count: int = 20) -> list[Path]:
    if output.is_symlink():
        raise ReviewWorkspaceError(f"output must not be a symlink: {output}")
    parent_symlink = first_symlink_component(output.parent)
    if parent_symlink is not None:
        raise ReviewWorkspaceError(f"output path must not contain a symlink: {parent_symlink}")
    if output.exists():
        if not output.is_dir():
            raise ReviewWorkspaceError(f"output must be a directory or absent: {output}")
        if any(output.iterdir()):
            raise ReviewWorkspaceError(f"output must be empty or absent: {output}")

    templates = {
        "labels-a.csv": (ANNOTATION_FIELDS, blank_rows(item_count, ANNOTATION_FIELDS)),
        "labels-b.csv": (ANNOTATION_FIELDS, blank_rows(item_count, ANNOTATION_FIELDS)),
        "review-log.csv": (REVIEW_LOG_FIELDS, blank_rows(item_count, REVIEW_LOG_FIELDS)),
    }
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(
        prefix=f".{output.name}.review-", dir=output.parent
    ) as staging_name:
        staging = Path(staging_name)
        for filename, (fields, rows) in templates.items():
            write_csv(staging / filename, fields, rows)

        if output.exists():
            for child in staging.iterdir():
                child.rename(output / child.name)
        else:
            staging.rename(output)
    return [output / filename for filename in templates]


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True, help="private review workspace")
    parser.add_argument("--items", type=int, default=20, help="number of trial items")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        created = create_review_workspace(args.output, args.items)
    except (OSError, ReviewWorkspaceError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Created {len(created)} private trial-review templates in {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
