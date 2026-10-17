#!/usr/bin/env python3
"""Validate the completed private review record for the 20-item trial."""

from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

from check_agreement import AgreementError, read_annotations
from create_trial_review_workspace import REVIEW_LOG_FIELDS


FINAL_STATUS = "pass"
YES_NO = {"yes", "no"}


def read_review_log(path: Path, expected_count: int) -> list[dict[str, str]]:
    try:
        with path.open(newline="", encoding="utf-8") as handle:
            reader = csv.DictReader(handle)
            if tuple(reader.fieldnames or ()) != REVIEW_LOG_FIELDS:
                raise ValueError(f"{path}: expected columns {REVIEW_LOG_FIELDS}")
            rows = list(reader)
    except (OSError, csv.Error) as exc:
        raise ValueError(f"{path}: cannot read review log: {exc}") from exc

    expected_ids = [f"I{index:03d}" for index in range(1, expected_count + 1)]
    ids = [row.get("item_id", "").strip() for row in rows]
    if ids != expected_ids:
        raise ValueError(f"{path}: item_id must contain I001 through I{expected_count:03d} in order")
    return rows


def validate_trial_review(
    workspace: Path,
    seed_dir: Path,
    expected_count: int = 20,
) -> list[str]:
    errors: list[str] = []
    for filename in ("labels-a.csv", "labels-b.csv"):
        try:
            read_annotations(workspace / filename, seed_dir, expected_count)
        except AgreementError as exc:
            errors.append(f"{filename}: {exc}")

    try:
        rows = read_review_log(workspace / "review-log.csv", expected_count)
    except (OSError, ValueError) as exc:
        return errors + [str(exc)]

    for row in rows:
        item_id = row["item_id"]
        for field in ("consent_status", "photo_privacy_status", "photo_quality_status", "review_status"):
            if row[field].strip().casefold() != FINAL_STATUS:
                errors.append(f"{item_id}: {field} must be pass for trial completion")
        for field in ("category_disagreement", "distinctiveness_disagreement"):
            if row[field].strip().casefold() not in YES_NO:
                errors.append(f"{item_id}: {field} must be yes or no")
        if not row["private_drive_path"].strip():
            errors.append(f"{item_id}: private_drive_path cannot be blank")
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--workspace", type=Path, required=True, help="completed private trial-review workspace")
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--expected-count", type=int, default=20)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_trial_review(args.workspace, args.seed_dir, args.expected_count)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Trial review check passed: {args.workspace}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
