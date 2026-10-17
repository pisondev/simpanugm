#!/usr/bin/env python3
"""Validate distractors, duplicate pairs, and paraphrase queries."""

from __future__ import annotations

import argparse
import csv
import re
import sys
import unicodedata
from pathlib import Path

from check import ValidationError, read_csv, validate_categories
from check_dataset import IMAGE_RE
from check_manifest import check_manifest


DISTRACTOR_FIELDS = (
    "distractor_id",
    "leaf_category",
    "primary_colour",
    "distinctiveness",
    "description",
)
DUPLICATE_FIELDS = ("pair_id", "item_id", "first_report_id", "second_report_id")
QUERY_FIELDS = ("query_id", "query_text", "target_report_id")
ID_PATTERNS = {"D": re.compile(r"^D\d{3}$"), "P": re.compile(r"^P\d{3}$"), "Q": re.compile(r"^Q\d{3}$")}


def read_required_csv(path: Path, required_fields: tuple[str, ...]) -> list[dict[str, str]]:
    if not path.is_file():
        raise ValidationError(f"missing metadata file: {path.name}")
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        fields = tuple(reader.fieldnames or ())
        missing = [field for field in required_fields if field not in fields]
        if missing:
            raise ValidationError(f"{path.name} missing columns: {', '.join(missing)}")
        rows = list(reader)
    if not rows:
        raise ValidationError(f"{path.name} must contain at least one row")
    return rows


def read_palette(seed_dir: Path) -> set[str]:
    rows = read_csv(seed_dir / "colours.csv", ("name", "hex", "sort_order"))
    return {row["name"] for row in rows}


def expected_ids(prefix: str, count: int) -> set[str]:
    if count < 1:
        raise ValidationError(f"expected {prefix} count must be positive")
    return {f"{prefix}{index:03d}" for index in range(1, count + 1)}


def validate_id_set(rows: list[dict[str, str]], field: str, prefix: str, expected_count: int) -> list[str]:
    errors: list[str] = []
    ids = [row.get(field, "").strip() for row in rows]
    if len(set(ids)) != len(ids):
        errors.append(f"{field} values must be unique")
    expected = expected_ids(prefix, expected_count)
    if set(ids) != expected or len(ids) != expected_count:
        errors.append(f"{field} must contain {prefix}001 through {prefix}{expected_count:03d} exactly")
    pattern = ID_PATTERNS[prefix]
    for value in ids:
        if not pattern.fullmatch(value) or value == f"{prefix}000":
            errors.append(f"{field} has invalid value {value!r}")
    return errors


def image_files(dataset_dir: Path, record_id: str, kind: str) -> list[Path]:
    item_dir = dataset_dir / "images" / record_id
    if not item_dir.is_dir():
        return []
    return sorted(
        path
        for path in item_dir.iterdir()
        if path.is_file() and IMAGE_RE.fullmatch(path.name) and path.name.casefold().startswith(f"{kind}_")
    )


def validate_distractors(
    dataset_dir: Path,
    seed_dir: Path,
    expected_count: int = 400,
) -> list[str]:
    errors: list[str] = []
    try:
        rows = read_required_csv(dataset_dir / "distractors.csv", DISTRACTOR_FIELDS)
        leaves = validate_categories(seed_dir / "categories.yaml")
        colours = read_palette(seed_dir)
    except (OSError, KeyError, ValidationError) as exc:
        return [str(exc)]

    errors.extend(validate_id_set(rows, "distractor_id", "D", expected_count))
    for row in rows:
        record_id = row["distractor_id"] or "<blank>"
        if row["leaf_category"] not in leaves:
            errors.append(f"{record_id}: unknown leaf_category")
        if row["primary_colour"] not in colours:
            errors.append(f"{record_id}: invalid primary_colour")
        if row["distinctiveness"] not in {"generic", "distinctive"}:
            errors.append(f"{record_id}: invalid distinctiveness")
        description = " ".join(row["description"].split())
        if not 10 <= len(description) <= 500:
            errors.append(f"{record_id}: description must be 10 to 500 characters")
        if len(image_files(dataset_dir, record_id, "finder")) < 1:
            errors.append(f"{record_id}: at least one finder photo is required")
    return errors


def validate_duplicates(
    dataset_dir: Path,
    expected_items: set[str],
    expected_count: int = 40,
) -> list[str]:
    try:
        rows = read_required_csv(dataset_dir / "duplicates.csv", DUPLICATE_FIELDS)
    except (OSError, ValidationError) as exc:
        return [str(exc)]

    errors = validate_id_set(rows, "pair_id", "P", expected_count)
    item_ids = [row["item_id"].strip() for row in rows]
    if len(set(item_ids)) != len(item_ids):
        errors.append("duplicates.csv must contain one pair per physical item")
    unknown_items = sorted(set(item_ids) - expected_items)
    if unknown_items:
        errors.append(f"duplicates.csv references unknown item ids: {', '.join(unknown_items)}")

    report_ids: set[str] = set()
    for row in rows:
        pair_id = row["pair_id"] or "<blank>"
        first = row["first_report_id"].strip()
        second = row["second_report_id"].strip()
        if not first or not second:
            errors.append(f"{pair_id}: report ids cannot be blank")
        if first and second and first == second:
            errors.append(f"{pair_id}: report ids must be different")
        for report_id in (first, second):
            if report_id and report_id in report_ids:
                errors.append(f"{pair_id}: report id {report_id} appears in more than one pair")
            if report_id:
                report_ids.add(report_id)
    return errors


def normalize_query(value: str) -> str:
    return " ".join(unicodedata.normalize("NFKC", value).casefold().split())


def validate_queries(dataset_dir: Path, expected_count: int = 50) -> list[str]:
    try:
        rows = read_required_csv(dataset_dir / "queries.csv", QUERY_FIELDS)
    except (OSError, ValidationError) as exc:
        return [str(exc)]

    errors = validate_id_set(rows, "query_id", "Q", expected_count)
    query_texts: set[str] = set()
    for row in rows:
        query_id = row["query_id"] or "<blank>"
        query = normalize_query(row["query_text"])
        target = row["target_report_id"].strip()
        if len(query) < 5 or len(query) > 500:
            errors.append(f"{query_id}: query_text must be 5 to 500 characters")
        if query and query in query_texts:
            errors.append(f"{query_id}: query_text is duplicated")
        if query:
            query_texts.add(query)
        if not target:
            errors.append(f"{query_id}: target_report_id cannot be blank")
    return errors


def read_label_ids(dataset_dir: Path) -> set[str]:
    rows = read_csv(
        dataset_dir / "labels.csv",
        (
            "item_id",
            "leaf_category",
            "primary_colour",
            "secondary_colour",
            "brand",
            "distinctiveness",
            "has_owner_photo",
            "flags",
        ),
    )
    ids = [row["item_id"].strip() for row in rows]
    if any(not re.fullmatch(r"I[0-9]{3}", value) or value == "I000" for value in ids):
        raise ValidationError("labels.csv item_id values must match I001 format")
    if len(set(ids)) != len(ids):
        raise ValidationError("labels.csv item_id values must be unique")
    return set(ids)


def validate_auxiliary(
    dataset_dir: Path,
    seed_dir: Path,
    expected_distractors: int = 400,
    expected_duplicates: int = 40,
    expected_queries: int = 50,
) -> list[str]:
    errors = check_manifest(dataset_dir)
    try:
        item_ids = read_label_ids(dataset_dir)
    except (OSError, KeyError, ValidationError) as exc:
        errors.append(str(exc))
        return errors
    errors.extend(validate_distractors(dataset_dir, seed_dir, expected_distractors))
    errors.extend(validate_duplicates(dataset_dir, item_ids, expected_duplicates))
    errors.extend(validate_queries(dataset_dir, expected_queries))
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset-dir", type=Path, required=True)
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--expected-distractors", type=int, default=400)
    parser.add_argument("--expected-duplicates", type=int, default=40)
    parser.add_argument("--expected-queries", type=int, default=50)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_auxiliary(
        args.dataset_dir,
        args.seed_dir,
        expected_distractors=args.expected_distractors,
        expected_duplicates=args.expected_duplicates,
        expected_queries=args.expected_queries,
    )
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Auxiliary dataset check passed: {args.dataset_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
