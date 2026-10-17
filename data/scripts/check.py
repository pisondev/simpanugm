#!/usr/bin/env python3
"""Validate the data seed foundation without requiring the application stack."""

from __future__ import annotations

import argparse
import csv
import re
import sys
from pathlib import Path
from typing import Any

try:
    import yaml
except ModuleNotFoundError:  # pragma: no cover - exercised by the CLI error path
    yaml = None


STATIC_FILES = (
    "categories.yaml",
    "colours.csv",
    "category_compatibility.csv",
    "brands.csv",
    "synonyms.csv",
    "settings.yaml",
)
COMPLETE_FILES = ("zones.csv", "locations.csv", "posts.yaml", "model_versions.yaml")
HEX_RE = re.compile(r"^#[0-9A-Fa-f]{6}$")
SLUG_RE = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*$")
SENSITIVITIES = {"normal", "valuable", "identity"}


class ValidationError(ValueError):
    """A human-readable seed validation failure."""


def load_yaml(path: Path) -> Any:
    if yaml is None:
        raise ValidationError(
            "PyYAML is required to validate YAML seeds; install it before running the check"
        )
    with path.open(encoding="utf-8") as handle:
        return yaml.safe_load(handle)


def read_csv(path: Path, expected_fields: tuple[str, ...]) -> list[dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        reader = csv.DictReader(handle)
        if tuple(reader.fieldnames or ()) != expected_fields:
            raise ValidationError(
                f"{path}: expected columns {expected_fields}, got {tuple(reader.fieldnames or ())}"
            )
        rows = list(reader)
    if not rows:
        raise ValidationError(f"{path}: file must contain at least one row")
    return rows


def flatten_categories(document: Any) -> tuple[set[str], list[dict[str, Any]]]:
    if not isinstance(document, dict) or not isinstance(document.get("parents"), list):
        raise ValidationError("categories.yaml: expected a top-level 'parents' list")

    leaves: set[str] = set()
    parents: list[dict[str, Any]] = []
    parent_slugs: set[str] = set()
    for parent in document["parents"]:
        if not isinstance(parent, dict):
            raise ValidationError("categories.yaml: every parent must be a mapping")
        for key in ("slug", "name", "sort_order", "children"):
            if key not in parent:
                raise ValidationError(f"categories.yaml: parent missing '{key}'")
        slug = parent["slug"]
        if not isinstance(slug, str) or not SLUG_RE.fullmatch(slug):
            raise ValidationError(f"categories.yaml: invalid parent slug {slug!r}")
        if slug in parent_slugs:
            raise ValidationError(f"categories.yaml: duplicate parent slug {slug}")
        parent_slugs.add(slug)
        if not isinstance(parent["children"], list) or not parent["children"]:
            raise ValidationError(f"categories.yaml: parent {slug} needs children")
        for child in parent["children"]:
            if not isinstance(child, dict):
                raise ValidationError(f"categories.yaml: child of {slug} must be a mapping")
            required = ("slug", "name", "sensitivity", "clip_prompts", "verification_questions")
            missing = [key for key in required if key not in child]
            if missing:
                raise ValidationError(
                    f"categories.yaml: leaf under {slug} missing {', '.join(missing)}"
                )
            child_slug = child["slug"]
            if not isinstance(child_slug, str) or not SLUG_RE.fullmatch(child_slug):
                raise ValidationError(f"categories.yaml: invalid leaf slug {child_slug!r}")
            if child_slug in leaves:
                raise ValidationError(f"categories.yaml: duplicate leaf slug {child_slug}")
            leaves.add(child_slug)
            if child["sensitivity"] not in SENSITIVITIES:
                raise ValidationError(
                    f"categories.yaml: invalid sensitivity for {child_slug}: {child['sensitivity']}"
                )
            if not isinstance(child["clip_prompts"], list) or not child["clip_prompts"]:
                raise ValidationError(f"categories.yaml: {child_slug} needs clip prompts")
            questions = child["verification_questions"]
            if not isinstance(questions, list) or not questions:
                raise ValidationError(f"categories.yaml: {child_slug} needs verification questions")
            question_keys: set[str] = set()
            for question in questions:
                if not isinstance(question, dict):
                    raise ValidationError(f"categories.yaml: question under {child_slug} must be a mapping")
                for key in ("key", "question", "required", "check_hint"):
                    if key not in question:
                        raise ValidationError(f"categories.yaml: question under {child_slug} missing '{key}'")
                if question["key"] in question_keys:
                    raise ValidationError(f"categories.yaml: duplicate question key under {child_slug}")
                question_keys.add(question["key"])
        parents.append(parent)
    return leaves, parents


def validate_colours(path: Path) -> None:
    rows = read_csv(path, ("name", "hex", "sort_order"))
    names = [row["name"] for row in rows]
    orders = [row["sort_order"] for row in rows]
    if len(set(names)) != len(names):
        raise ValidationError(f"{path}: colour names must be unique")
    if orders != [str(index) for index in range(1, len(rows) + 1)]:
        raise ValidationError(f"{path}: sort_order must be consecutive starting at 1")
    if set(names) != {
        "hitam", "putih", "abu-abu", "perak", "emas", "merah", "merah muda",
        "oranye", "kuning", "krem", "cokelat", "hijau", "biru", "ungu",
    }:
        raise ValidationError(f"{path}: palette must contain exactly the documented 14 colours")
    for row in rows:
        if not HEX_RE.fullmatch(row["hex"]):
            raise ValidationError(f"{path}: invalid hex colour {row['hex']!r}")


def validate_categories(path: Path) -> set[str]:
    leaves, parents = flatten_categories(load_yaml(path))
    if [parent["sort_order"] for parent in parents] != list(range(1, len(parents) + 1)):
        raise ValidationError("categories.yaml: parent sort_order must be consecutive starting at 1")
    return leaves


def validate_compatibility(path: Path, leaves: set[str]) -> None:
    rows = read_csv(path, ("category_a", "category_b", "factor"))
    pairs: set[tuple[str, str]] = set()
    for row in rows:
        pair = (row["category_a"], row["category_b"])
        if pair[0] >= pair[1]:
            raise ValidationError(f"{path}: category pair must be sorted: {pair}")
        if pair in pairs:
            raise ValidationError(f"{path}: duplicate category pair {pair}")
        pairs.add(pair)
        if pair[0] not in leaves or pair[1] not in leaves:
            raise ValidationError(f"{path}: compatibility pair references an unknown leaf: {pair}")
        try:
            factor = float(row["factor"])
        except ValueError as exc:
            raise ValidationError(f"{path}: factor is not numeric: {row['factor']!r}") from exc
        if not 0 <= factor <= 1:
            raise ValidationError(f"{path}: factor must be between 0 and 1")


def validate_brands(path: Path) -> None:
    rows = read_csv(path, ("name", "aliases"))
    names = [row["name"] for row in rows]
    if len(set(names)) != len(names):
        raise ValidationError(f"{path}: brand names must be unique")
    for row in rows:
        if not row["name"].strip() or not row["aliases"].strip():
            raise ValidationError(f"{path}: brand names and aliases cannot be blank")


def validate_synonyms(path: Path) -> None:
    rows = read_csv(path, ("term", "expansion"))
    terms = [row["term"] for row in rows]
    if len(set(terms)) != len(terms):
        raise ValidationError(f"{path}: synonym terms must be unique")
    for row in rows:
        if not row["term"].strip() or not row["expansion"].strip():
            raise ValidationError(f"{path}: terms and expansions cannot be blank")


def validate_settings(path: Path) -> None:
    document = load_yaml(path)
    if not isinstance(document, dict):
        raise ValidationError("settings.yaml: expected a mapping")
    expected: dict[str, tuple[type, float, float]] = {
        "lost.stale_days": (int, 14, 180),
        "lost.expiry_grace_days": (int, 3, 30),
        "lost.reopen_days": (int, 7, 90),
        "found.with_finder_max_hours": (int, 12, 72),
        "custody.retention_days.normal": (int, 7, 120),
        "custody.retention_days.valuable": (int, 14, 180),
        "custody.retention_days.identity": (int, 7, 90),
        "claim.max_open": (int, 1, 10),
        "claim.reject_flag_count": (int, 2, 10),
        "claim.reject_flag_window_days": (int, 7, 90),
        "claim.info_timeout_days": (int, 2, 30),
        "claim.approved_timeout_days": (int, 2, 30),
        "notify.match_throttle_hours": (int, 1, 24),
        "analytics.min_cell_count": (int, 3, 20),
    }

    def get_path(key: str) -> Any:
        value: Any = document
        for part in key.split("."):
            if not isinstance(value, dict) or part not in value:
                raise ValidationError(f"settings.yaml: missing key {key}")
            value = value[part]
        return value

    domains = get_path("auth.allowed_email_domains")
    if not isinstance(domains, list) or not domains or any(not isinstance(item, str) for item in domains):
        raise ValidationError("settings.yaml: allowed_email_domains must be a non-empty list of strings")
    for key, (value_type, lower, upper) in expected.items():
        value = get_path(key)
        if type(value) is not value_type or not lower <= value <= upper:
            raise ValidationError(f"settings.yaml: {key} must be {value_type.__name__} in [{lower}, {upper}]")
    quiet_hours = get_path("notify.email_quiet_hours")
    if quiet_hours != {"from": "22:00", "to": "06:00"}:
        raise ValidationError("settings.yaml: email_quiet_hours must match the documented WIB default")


def validate_seed_dir(seed_dir: Path, require_complete: bool = False) -> list[str]:
    errors: list[str] = []
    for filename in STATIC_FILES:
        if not (seed_dir / filename).is_file():
            errors.append(f"missing required seed file: {filename}")
    if errors:
        return errors

    try:
        leaves = validate_categories(seed_dir / "categories.yaml")
        validate_colours(seed_dir / "colours.csv")
        validate_compatibility(seed_dir / "category_compatibility.csv", leaves)
        validate_brands(seed_dir / "brands.csv")
        validate_synonyms(seed_dir / "synonyms.csv")
        validate_settings(seed_dir / "settings.yaml")
    except (OSError, ValidationError) as exc:
        errors.append(str(exc))

    if require_complete:
        for filename in COMPLETE_FILES:
            if not (seed_dir / filename).is_file():
                errors.append(f"missing complete-seed file: {filename}")
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--seed-dir",
        type=Path,
        default=Path("data/seed"),
        help="directory containing seed files (default: data/seed)",
    )
    parser.add_argument(
        "--require-complete",
        action="store_true",
        help="also require the location, post, zone and model-version files",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_seed_dir(args.seed_dir, require_complete=args.require_complete)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    mode = "complete" if args.require_complete else "foundation"
    print(f"Seed check passed ({mode}): {args.seed_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
