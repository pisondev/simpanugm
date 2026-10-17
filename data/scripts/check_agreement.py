#!/usr/bin/env python3
"""Validate two private label sheets and report Cohen's kappa."""

from __future__ import annotations

import argparse
import csv
import os
import sys
import tempfile
from pathlib import Path

from check import ValidationError, validate_categories


ANNOTATION_FIELDS = ("item_id", "leaf_category", "distinctiveness")
ALLOWED_DISTINCTIVENESS = {"generic", "distinctive"}


class AgreementError(ValueError):
    """A human-readable agreement-validation failure."""


def expected_ids(count: int) -> list[str]:
    if count < 1:
        raise AgreementError("expected count must be positive")
    return [f"I{index:03d}" for index in range(1, count + 1)]


def read_annotations(
    path: Path,
    seed_dir: Path,
    expected_count: int = 20,
) -> dict[str, dict[str, str]]:
    try:
        leaves = validate_categories(seed_dir / "categories.yaml")
        with path.open(newline="", encoding="utf-8") as handle:
            reader = csv.DictReader(handle)
            fields = set(reader.fieldnames or ())
            missing = [field for field in ANNOTATION_FIELDS if field not in fields]
            if missing:
                raise AgreementError(f"{path}: missing columns: {', '.join(missing)}")
            rows = list(reader)
    except (OSError, csv.Error, ValidationError) as exc:
        raise AgreementError(str(exc)) from exc

    ids = expected_ids(expected_count)
    row_ids = [row.get("item_id", "").strip() for row in rows]
    if len(row_ids) != expected_count or set(row_ids) != set(ids):
        raise AgreementError(f"{path}: item_id must contain I001 through I{expected_count:03d} exactly")
    if len(set(row_ids)) != len(row_ids):
        raise AgreementError(f"{path}: item_id values must be unique")

    annotations: dict[str, dict[str, str]] = {}
    for row in rows:
        item_id = row["item_id"].strip()
        leaf_category = row["leaf_category"].strip()
        distinctiveness = row["distinctiveness"].strip()
        if leaf_category not in leaves:
            raise AgreementError(f"{path}: {item_id} has an unknown leaf_category")
        if distinctiveness not in ALLOWED_DISTINCTIVENESS:
            raise AgreementError(f"{path}: {item_id} has an invalid distinctiveness value")
        annotations[item_id] = {
            "leaf_category": leaf_category,
            "distinctiveness": distinctiveness,
        }
    return annotations


def cohen_kappa(first: list[str], second: list[str]) -> tuple[float, float]:
    if len(first) != len(second) or not first:
        raise AgreementError("agreement inputs must have the same non-zero length")
    labels = sorted(set(first) | set(second))
    total = len(first)
    observed = sum(left == right for left, right in zip(first, second)) / total
    first_counts = {label: first.count(label) for label in labels}
    second_counts = {label: second.count(label) for label in labels}
    expected = sum(
        first_counts[label] * second_counts[label] for label in labels
    ) / (total * total)
    kappa = 1.0 if expected == 1.0 and observed == 1.0 else (
        0.0 if expected == 1.0 else (observed - expected) / (1 - expected)
    )
    return observed, kappa


def summarize_agreement(
    first: dict[str, dict[str, str]],
    second: dict[str, dict[str, str]],
) -> dict[str, object]:
    if set(first) != set(second):
        raise AgreementError("annotation sheets must contain the same item ids")
    item_ids = sorted(first)
    metrics: dict[str, dict[str, float]] = {}
    for field in ("leaf_category", "distinctiveness"):
        observed, kappa = cohen_kappa(
            [first[item_id][field] for item_id in item_ids],
            [second[item_id][field] for item_id in item_ids],
        )
        metrics[field] = {"observed_agreement": observed, "cohen_kappa": kappa}
    return {"record_count": len(item_ids), "metrics": metrics}


def render_report(summary: dict[str, object]) -> str:
    metrics = summary["metrics"]
    return f"""# Label agreement report

This report contains aggregate agreement statistics only. The private annotation sheets
remain outside the repository.

- Records: **{summary['record_count']}**

| Field | Observed agreement | Cohen's kappa |
|---|---:|---:|
| `leaf_category` | {metrics['leaf_category']['observed_agreement']:.4f} | {metrics['leaf_category']['cohen_kappa']:.4f} |
| `distinctiveness` | {metrics['distinctiveness']['observed_agreement']:.4f} | {metrics['distinctiveness']['cohen_kappa']:.4f} |
"""


def write_report(path: Path, content: str) -> None:
    if path.is_symlink():
        raise OSError(f"output must not be a symlink: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            prefix=f".{path.name}.",
            suffix=".tmp",
            dir=path.parent,
            delete=False,
        ) as handle:
            temporary = Path(handle.name)
            handle.write(content)
        os.replace(temporary, path)
    finally:
        if temporary is not None and temporary.exists():
            temporary.unlink()


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--first", type=Path, required=True, help="first private annotation CSV")
    parser.add_argument("--second", type=Path, required=True, help="second private annotation CSV")
    parser.add_argument("--seed-dir", type=Path, default=Path("data/seed"))
    parser.add_argument("--expected-count", type=int, default=20)
    parser.add_argument("--output", type=Path, help="optional private Markdown report")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        first = read_annotations(args.first, args.seed_dir, args.expected_count)
        second = read_annotations(args.second, args.seed_dir, args.expected_count)
        report = render_report(summarize_agreement(first, second))
        if args.output is None:
            print(report, end="")
        else:
            write_report(args.output, report)
            print(f"Wrote label agreement report: {args.output}")
    except (AgreementError, OSError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
