#!/usr/bin/env python3
"""Run all private evaluation-dataset freeze gates without publishing data."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from check_auxiliary import validate_auxiliary
from check_dataset import validate_dataset
from check_trial_review import validate_trial_review
from split_eval import (
    read_distractor_ids,
    read_distractor_split,
    read_labels,
    read_split,
    verify_distractor_split_rows,
    verify_split_rows,
)


def prefix_errors(prefix: str, errors: list[str]) -> list[str]:
    return [f"{prefix}: {error}" for error in errors]


def validate_freeze(
    dataset_dir: Path,
    seed_dir: Path,
    profile: str,
    expected_count: int | None = None,
    review_workspace: Path | None = None,
    split_path: Path | None = None,
    distractor_split_path: Path | None = None,
    val_count: int | None = None,
    expected_distractors: int = 400,
    expected_duplicates: int = 40,
    expected_queries: int = 50,
) -> list[str]:
    if profile not in {"trial", "full"}:
        return [f"unknown profile: {profile}"]
    item_count = expected_count if expected_count is not None else (20 if profile == "trial" else 150)
    errors = prefix_errors("dataset", validate_dataset(dataset_dir, seed_dir, item_count))

    if profile == "trial":
        if review_workspace is None:
            errors.append("trial: --review-workspace is required")
        else:
            errors.extend(
                prefix_errors(
                    "trial review",
                    validate_trial_review(review_workspace, seed_dir, item_count),
                )
            )
        if split_path is not None or distractor_split_path is not None or val_count is not None:
            errors.append("trial: split arguments are only valid for the full profile")
        return errors

    errors.extend(
        prefix_errors(
            "auxiliary",
            validate_auxiliary(
                dataset_dir,
                seed_dir,
                expected_distractors=expected_distractors,
                expected_duplicates=expected_duplicates,
                expected_queries=expected_queries,
            ),
        )
    )
    if split_path is None or distractor_split_path is None or val_count is None:
        errors.append("full: --split, --distractor-split, and --val-count are required")
        return errors
    try:
        labels = read_labels(dataset_dir / "labels.csv", seed_dir)
        split_errors = verify_split_rows(read_split(split_path), labels, val_count=val_count)
        errors.extend(prefix_errors("split", split_errors))
        distractor_ids = read_distractor_ids(dataset_dir / "distractors.csv")
        distractor_errors = verify_distractor_split_rows(
            read_distractor_split(distractor_split_path),
            distractor_ids,
            val_count,
            len(labels),
        )
        errors.extend(prefix_errors("distractor split", distractor_errors))
    except (OSError, ValueError) as exc:
        errors.append(f"split: cannot read private split files: {exc}")
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset-dir", type=Path, required=True)
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--profile", choices=("trial", "full"), required=True)
    parser.add_argument("--expected-count", type=int)
    parser.add_argument("--review-workspace", type=Path)
    parser.add_argument("--split", type=Path, help="private item split CSV for the full profile")
    parser.add_argument("--distractor-split", type=Path, help="private distractor split CSV for the full profile")
    parser.add_argument("--val-count", type=int, help="number of item records in val")
    parser.add_argument("--expected-distractors", type=int, default=400)
    parser.add_argument("--expected-duplicates", type=int, default=40)
    parser.add_argument("--expected-queries", type=int, default=50)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_freeze(
        args.dataset_dir,
        args.seed_dir,
        args.profile,
        expected_count=args.expected_count,
        review_workspace=args.review_workspace,
        split_path=args.split,
        distractor_split_path=args.distractor_split,
        val_count=args.val_count,
        expected_distractors=args.expected_distractors,
        expected_duplicates=args.expected_duplicates,
        expected_queries=args.expected_queries,
    )
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Evaluation freeze checks passed: {args.dataset_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
