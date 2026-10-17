#!/usr/bin/env python3
"""Create and verify deterministic, item-grouped evaluation splits."""

from __future__ import annotations

import argparse
import csv
import math
import random
import re
import sys
from collections import defaultdict
from pathlib import Path
from typing import Iterable

from check import ValidationError, load_yaml


LABEL_FIELDS = ("item_id", "leaf_category", "distinctiveness")
SPLIT_FIELDS = ("item_id", "split", "parent_category", "distinctiveness")
DISTRACTOR_SPLIT_FIELDS = ("distractor_id", "split")
SPLITS = {"val", "test"}
ITEM_ID_RE = re.compile(r"^I[0-9]{3}$")
DISTRACTOR_ID_RE = re.compile(r"^D[0-9]{3}$")


class SplitError(ValueError):
    """A human-readable split validation failure."""


def leaf_parents(seed_dir: Path) -> dict[str, str]:
    document = load_yaml(seed_dir / "categories.yaml")
    if not isinstance(document, dict) or not isinstance(document.get("parents"), list):
        raise SplitError("categories.yaml: expected a top-level 'parents' list")
    result: dict[str, str] = {}
    for parent in document["parents"]:
        if not isinstance(parent, dict):
            raise SplitError("categories.yaml: every parent must be a mapping")
        parent_slug = parent.get("slug")
        for child in parent.get("children", []):
            if not isinstance(child, dict) or not child.get("slug"):
                raise SplitError(f"categories.yaml: invalid child under {parent_slug}")
            leaf_slug = child["slug"]
            if leaf_slug in result:
                raise SplitError(f"categories.yaml: duplicate leaf {leaf_slug}")
            result[leaf_slug] = parent_slug
    return result


def read_labels(path: Path, seed_dir: Path) -> list[dict[str, str]]:
    parents = leaf_parents(seed_dir)
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        fields = tuple(reader.fieldnames or ())
        missing = [field for field in LABEL_FIELDS if field not in fields]
        if missing:
            raise SplitError(f"{path}: missing columns: {', '.join(missing)}")
        rows = list(reader)

    if not rows:
        raise SplitError(f"{path}: labels file must contain at least one row")
    seen: set[str] = set()
    normalized: list[dict[str, str]] = []
    for row in rows:
        item_id = row.get("item_id", "").strip()
        leaf_category = row.get("leaf_category", "").strip()
        distinctiveness = row.get("distinctiveness", "").strip()
        if not item_id:
            raise SplitError(f"{path}: item_id cannot be blank")
        if not ITEM_ID_RE.fullmatch(item_id) or item_id == "I000":
            raise SplitError(f"{path}: item_id must match I001 format: {item_id!r}")
        if item_id in seen:
            raise SplitError(f"{path}: duplicate item_id {item_id}")
        seen.add(item_id)
        if leaf_category not in parents:
            raise SplitError(f"{path}: {item_id} has an unknown leaf_category")
        if distinctiveness not in {"generic", "distinctive"}:
            raise SplitError(f"{path}: {item_id} has invalid distinctiveness")
        normalized.append(
            {
                "item_id": item_id,
                "parent_category": parents[leaf_category],
                "distinctiveness": distinctiveness,
            }
        )
    return normalized


def allocate_counts(groups: dict[tuple[str, str], list[dict[str, str]]], target: int) -> dict[tuple[str, str], int]:
    total = sum(len(rows) for rows in groups.values())
    if not 0 < target < total:
        raise SplitError(f"val count must be between 1 and {total - 1}")

    raw = {key: len(rows) * target / total for key, rows in groups.items()}
    counts = {key: math.floor(value) for key, value in raw.items()}
    remaining = target - sum(counts.values())
    order = sorted(
        groups,
        key=lambda key: (-(raw[key] - counts[key]), key[0], key[1]),
    )
    for key in order[:remaining]:
        counts[key] += 1
    return counts


def make_split_rows(rows: Iterable[dict[str, str]], val_count: int, seed: int) -> list[dict[str, str]]:
    groups: dict[tuple[str, str], list[dict[str, str]]] = defaultdict(list)
    for row in rows:
        groups[(row["parent_category"], row["distinctiveness"])].append(row)

    counts = allocate_counts(groups, val_count)
    result: list[dict[str, str]] = []
    for key in sorted(groups):
        group = list(groups[key])
        random.Random(f"{seed}:{key[0]}:{key[1]}").shuffle(group)
        val_rows = group[: counts[key]]
        test_rows = group[counts[key] :]
        for split, selected in (("val", val_rows), ("test", test_rows)):
            result.extend(
                {
                    "item_id": row["item_id"],
                    "split": split,
                    "parent_category": row["parent_category"],
                    "distinctiveness": row["distinctiveness"],
                }
                for row in selected
            )
    return sorted(result, key=lambda row: row["item_id"])


def verify_split_rows(
    rows: Iterable[dict[str, str]],
    labels: Iterable[dict[str, str]],
    val_count: int | None = None,
) -> list[str]:
    errors: list[str] = []
    split_rows = list(rows)
    label_rows = list(labels)
    by_id: dict[str, dict[str, str]] = {}
    for row in split_rows:
        item_id = row.get("item_id", "")
        if item_id in by_id:
            errors.append(f"duplicate split assignment for {item_id}")
        by_id[item_id] = row
        if row.get("split") not in SPLITS:
            errors.append(f"{item_id}: split must be val or test")
    label_ids = {row["item_id"] for row in label_rows}
    split_ids = set(by_id)
    missing = sorted(label_ids - split_ids)
    extra = sorted(split_ids - label_ids)
    if missing:
        errors.append(f"missing split assignments: {', '.join(missing)}")
    if extra:
        errors.append(f"split contains unknown item ids: {', '.join(extra)}")
    for row in label_rows:
        assigned = by_id.get(row["item_id"])
        if not assigned:
            continue
        for field in ("parent_category", "distinctiveness"):
            if assigned.get(field) != row[field]:
                errors.append(f"{row['item_id']}: split {field} does not match labels")
    if {row.get("split") for row in split_rows} != SPLITS:
        errors.append("split output must contain both val and test")
    if val_count is not None and sum(row.get("split") == "val" for row in split_rows) != val_count:
        errors.append(f"split output must contain exactly {val_count} val items")
    return errors


def read_split(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        if tuple(reader.fieldnames or ()) != SPLIT_FIELDS:
            raise SplitError(f"{path}: expected columns {SPLIT_FIELDS}")
        return list(reader)


def write_split(path: Path, rows: Iterable[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=SPLIT_FIELDS)
        writer.writeheader()
        writer.writerows(rows)


def read_distractor_ids(path: Path) -> list[str]:
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        if "distractor_id" not in (reader.fieldnames or ()):
            raise SplitError(f"{path}: missing column distractor_id")
        rows = list(reader)
    if not rows:
        raise SplitError(f"{path}: distractors file must contain at least one row")
    ids = [row.get("distractor_id", "").strip() for row in rows]
    if any(not DISTRACTOR_ID_RE.fullmatch(value) or value == "D000" for value in ids):
        raise SplitError(f"{path}: distractor_id values must match D001 format")
    if len(set(ids)) != len(ids):
        raise SplitError(f"{path}: distractor_id values must be unique")
    return ids


def expected_distractor_val_count(distractor_count: int, item_val_count: int, item_count: int) -> int:
    if item_count <= 0 or not 0 < item_val_count < item_count:
        raise SplitError("item split must contain both val and test before splitting distractors")
    target = math.floor(distractor_count * item_val_count / item_count + 0.5)
    if distractor_count > 1:
        return min(max(target, 1), distractor_count - 1)
    return target


def make_distractor_split_rows(
    distractor_ids: Iterable[str],
    item_val_count: int,
    item_count: int,
    seed: int,
) -> list[dict[str, str]]:
    ids = sorted(distractor_ids)
    val_count = expected_distractor_val_count(len(ids), item_val_count, item_count)
    random.Random(f"{seed}:distractors").shuffle(ids)
    rows = [
        {"distractor_id": distractor_id, "split": "val" if index < val_count else "test"}
        for index, distractor_id in enumerate(ids)
    ]
    return sorted(rows, key=lambda row: row["distractor_id"])


def verify_distractor_split_rows(
    rows: Iterable[dict[str, str]],
    distractor_ids: Iterable[str],
    item_val_count: int,
    item_count: int,
) -> list[str]:
    errors: list[str] = []
    split_rows = list(rows)
    expected_ids = set(distractor_ids)
    seen: set[str] = set()
    for row in split_rows:
        distractor_id = row.get("distractor_id", "")
        if distractor_id in seen:
            errors.append(f"duplicate distractor split assignment for {distractor_id}")
        seen.add(distractor_id)
        if row.get("split") not in SPLITS:
            errors.append(f"{distractor_id}: split must be val or test")
    if seen != expected_ids:
        missing = sorted(expected_ids - seen)
        extra = sorted(seen - expected_ids)
        if missing:
            errors.append(f"missing distractor split assignments: {', '.join(missing)}")
        if extra:
            errors.append(f"split contains unknown distractor ids: {', '.join(extra)}")
    expected_val = expected_distractor_val_count(len(expected_ids), item_val_count, item_count)
    if sum(row.get("split") == "val" for row in split_rows) != expected_val:
        errors.append(f"distractor split output must contain exactly {expected_val} val records")
    if len(expected_ids) > 1 and {row.get("split") for row in split_rows} != SPLITS:
        errors.append("distractor split output must contain both val and test")
    return errors


def read_distractor_split(path: Path) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        if tuple(reader.fieldnames or ()) != DISTRACTOR_SPLIT_FIELDS:
            raise SplitError(f"{path}: expected columns {DISTRACTOR_SPLIT_FIELDS}")
        return list(reader)


def write_distractor_split(path: Path, rows: Iterable[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=DISTRACTOR_SPLIT_FIELDS)
        writer.writeheader()
        writer.writerows(rows)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--labels", type=Path, required=True, help="private labels.csv")
    parser.add_argument("--output", type=Path, help="private split CSV to create")
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--val-count", type=int, required=True, help="number of items assigned to val")
    parser.add_argument("--seed", type=int, default=2026)
    parser.add_argument("--verify", type=Path, help="verify an existing split instead of creating one")
    parser.add_argument("--distractors", type=Path, help="private distractors.csv to split")
    parser.add_argument("--distractor-output", type=Path, help="private distractor split CSV to create")
    parser.add_argument("--verify-distractors", type=Path, help="verify an existing distractor split")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        labels = read_labels(args.labels, args.seed_dir)
        distractor_ids = read_distractor_ids(args.distractors) if args.distractors else None
        if args.verify_distractors and not args.verify:
            raise SplitError("--verify is required with --verify-distractors")
        if args.verify:
            errors = verify_split_rows(read_split(args.verify), labels, val_count=args.val_count)
            if args.verify_distractors:
                if distractor_ids is None:
                    raise SplitError("--distractors is required with --verify-distractors")
                errors.extend(
                    verify_distractor_split_rows(
                        read_distractor_split(args.verify_distractors),
                        distractor_ids,
                        args.val_count,
                        len(labels),
                    )
                )
            if errors:
                raise SplitError("; ".join(errors))
            print(f"Evaluation split passed: {args.verify}")
            if args.verify_distractors:
                print(f"Distractor split passed: {args.verify_distractors}")
            return 0
        if not args.output:
            raise SplitError("--output is required when creating a split")
        if args.distractors and not args.distractor_output:
            raise SplitError("--distractor-output is required with --distractors")
        if args.distractor_output and not args.distractors:
            raise SplitError("--distractors is required with --distractor-output")
        rows = make_split_rows(labels, args.val_count, args.seed)
        errors = verify_split_rows(rows, labels, val_count=args.val_count)
        distractor_rows: list[dict[str, str]] = []
        if distractor_ids is not None:
            distractor_rows = make_distractor_split_rows(
                distractor_ids,
                args.val_count,
                len(labels),
                args.seed,
            )
            errors.extend(
                verify_distractor_split_rows(
                    distractor_rows,
                    distractor_ids,
                    args.val_count,
                    len(labels),
                )
            )
        if errors:
            raise SplitError("; ".join(errors))
        write_split(args.output, rows)
        if args.distractor_output:
            write_distractor_split(args.distractor_output, distractor_rows)
    except (OSError, ValidationError, SplitError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Evaluation split written: {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
