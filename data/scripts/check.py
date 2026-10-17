#!/usr/bin/env python3
"""Validate the data seed foundation without requiring the application stack."""

from __future__ import annotations

import argparse
import csv
import math
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
ZONE_FIELDS = ("slug", "name", "centroid_lat", "centroid_lng")
LOCATION_FIELDS = ("slug", "name", "zone_slug", "kind", "lat", "lng", "aliases")
WEEKDAYS = ("mon", "tue", "wed", "thu", "fri", "sat", "sun")
TIME_RE = re.compile(r"^(?:[01]\d|2[0-3]):[0-5]\d$")
LOCATION_KINDS = {"building", "outdoor", "facility", "transport", "parking"}
MODEL_COMPONENTS = {
    "image_encoder",
    "xmodal_text_encoder",
    "text_encoder",
    "ocr",
    "category_classifier",
    "colour_extractor",
    "brand_matcher",
    "fusion",
}


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


def parse_coordinate(value: str, field: str) -> float:
    try:
        result = float(value)
    except ValueError as exc:
        raise ValidationError(f"{field} must be numeric: {value!r}") from exc
    if not math.isfinite(result):
        raise ValidationError(f"{field} must be finite: {value!r}")
    return result


def parse_bbox(value: str) -> tuple[float, float, float, float]:
    """Parse a finite south,west,north,east bounding box."""
    try:
        south, west, north, east = (float(part) for part in value.split(","))
    except ValueError as exc:
        raise argparse.ArgumentTypeError("bbox must be south,west,north,east") from exc
    values = (south, west, north, east)
    if not all(math.isfinite(part) for part in values):
        raise argparse.ArgumentTypeError("bbox values must be finite")
    if south > north or west > east:
        raise argparse.ArgumentTypeError("bbox must have minimum values first")
    return values


def validate_zones(path: Path) -> set[str]:
    rows = read_csv(path, ZONE_FIELDS)
    slugs = [row["slug"].strip() for row in rows]
    if len(set(slugs)) != len(slugs):
        raise ValidationError(f"{path}: zone slugs must be unique")
    if set(slugs) != {"saintek", "soshum", "agro", "medika", "pusat"}:
        raise ValidationError(f"{path}: must contain the five documented campus zones")
    for row in rows:
        slug = row["slug"].strip()
        if not SLUG_RE.fullmatch(slug) or not row["name"].strip():
            raise ValidationError(f"{path}: invalid zone row for {slug!r}")
        lat = parse_coordinate(row["centroid_lat"], f"{slug}.centroid_lat")
        lng = parse_coordinate(row["centroid_lng"], f"{slug}.centroid_lng")
        if not -90 <= lat <= 90 or not -180 <= lng <= 180:
            raise ValidationError(f"{path}: coordinates out of range for {slug}")
    return set(slugs)


def validate_locations(
    path: Path,
    zone_slugs: set[str],
    campus_bbox: tuple[float, float, float, float] | None = None,
) -> set[str]:
    rows = read_csv(path, LOCATION_FIELDS)
    slugs = [row["slug"].strip() for row in rows]
    if len(set(slugs)) != len(slugs):
        raise ValidationError(f"{path}: location slugs must be unique")
    for row in rows:
        slug = row["slug"].strip()
        name = " ".join(row["name"].split())
        zone_slug = row["zone_slug"].strip()
        kind = row["kind"].strip()
        if not SLUG_RE.fullmatch(slug) or not name:
            raise ValidationError(f"{path}: invalid location row for {slug!r}")
        if zone_slug not in zone_slugs:
            raise ValidationError(f"{path}: {slug} references unknown zone {zone_slug!r}")
        if kind not in LOCATION_KINDS:
            raise ValidationError(f"{path}: {slug} has invalid kind {kind!r}")
        lat = parse_coordinate(row["lat"], f"{slug}.lat")
        lng = parse_coordinate(row["lng"], f"{slug}.lng")
        if not -90 <= lat <= 90 or not -180 <= lng <= 180:
            raise ValidationError(f"{path}: coordinates out of range for {slug}")
        if campus_bbox is not None:
            south, west, north, east = campus_bbox
            if not south <= lat <= north or not west <= lng <= east:
                raise ValidationError(f"{path}: {slug} is outside the campus bbox")
        aliases = [alias.strip() for alias in row["aliases"].split(";") if alias.strip()]
        if len({alias.casefold() for alias in aliases}) != len(aliases):
            raise ValidationError(f"{path}: duplicate aliases for {slug}")
        if name.casefold() in {alias.casefold() for alias in aliases}:
            raise ValidationError(f"{path}: location name repeated in aliases for {slug}")
    return set(slugs)


def parse_time(value: Any, field: str) -> int:
    if not isinstance(value, str) or not TIME_RE.fullmatch(value):
        raise ValidationError(f"{field} must be HH:MM")
    hour, minute = (int(part) for part in value.split(":"))
    return hour * 60 + minute


def validate_posts(path: Path, location_slugs: set[str]) -> None:
    document = load_yaml(path)
    if not isinstance(document, list) or not document:
        raise ValidationError(f"{path}: expected a non-empty YAML list")
    seen: set[str] = set()
    for post in document:
        if not isinstance(post, dict):
            raise ValidationError(f"{path}: every post must be a mapping")
        required = ("slug", "name", "location_slug", "phone", "opening_hours")
        missing = [key for key in required if key not in post]
        if missing:
            raise ValidationError(f"{path}: post missing {', '.join(missing)}")
        slug = post["slug"]
        if not isinstance(slug, str) or not SLUG_RE.fullmatch(slug):
            raise ValidationError(f"{path}: invalid post slug {slug!r}")
        if slug in seen:
            raise ValidationError(f"{path}: duplicate post slug {slug}")
        seen.add(slug)
        if not isinstance(post["name"], str) or not post["name"].strip():
            raise ValidationError(f"{path}: post {slug} needs a name")
        if post["location_slug"] not in location_slugs:
            raise ValidationError(f"{path}: post {slug} references unknown location")
        if not isinstance(post["phone"], str):
            raise ValidationError(f"{path}: post {slug} phone must be a string")
        hours = post["opening_hours"]
        if not isinstance(hours, dict) or set(hours) != set(WEEKDAYS):
            raise ValidationError(f"{path}: post {slug} opening_hours must contain all weekdays")
        for weekday in WEEKDAYS:
            slots = hours[weekday]
            if not isinstance(slots, list):
                raise ValidationError(f"{path}: post {slug} {weekday} must be a list")
            for index, slot in enumerate(slots):
                if not isinstance(slot, list) or len(slot) != 2:
                    raise ValidationError(f"{path}: post {slug} {weekday}[{index}] must be [start, end]")
                start = parse_time(slot[0], f"{path}:{slug}.{weekday}[{index}].start")
                end = parse_time(slot[1], f"{path}:{slug}.{weekday}[{index}].end")
                if start >= end:
                    raise ValidationError(f"{path}: post {slug} {weekday}[{index}] must end after start")


def validate_model_versions(path: Path) -> None:
    document = load_yaml(path)
    if not isinstance(document, list) or not document:
        raise ValidationError(f"{path}: expected a non-empty YAML list")
    names: set[str] = set()
    components: set[str] = set()
    active_by_component: dict[str, int] = {}
    for version in document:
        if not isinstance(version, dict):
            raise ValidationError(f"{path}: every model version must be a mapping")
        required = ("component", "name", "identity", "status")
        missing = [key for key in required if key not in version]
        if missing:
            raise ValidationError(f"{path}: model version missing {', '.join(missing)}")
        component = version["component"]
        name = version["name"]
        status = version["status"]
        if component not in MODEL_COMPONENTS:
            raise ValidationError(f"{path}: unknown component {component!r}")
        if not isinstance(name, str) or not name.strip() or name in names:
            raise ValidationError(f"{path}: model names must be non-empty and unique")
        if not isinstance(version["identity"], dict) or not version["identity"]:
            raise ValidationError(f"{path}: {name} identity must be a non-empty mapping")
        if status not in {"candidate", "active", "retired"}:
            raise ValidationError(f"{path}: {name} has invalid status {status!r}")
        if component == "fusion" and not isinstance(version.get("config"), dict):
            raise ValidationError(f"{path}: fusion version {name} needs a config mapping")
        names.add(name)
        components.add(component)
        if status == "active":
            active_by_component[component] = active_by_component.get(component, 0) + 1
    if components != MODEL_COMPONENTS:
        missing = sorted(MODEL_COMPONENTS - components)
        raise ValidationError(f"{path}: missing model components: {', '.join(missing)}")
    invalid_active = sorted(component for component in MODEL_COMPONENTS if active_by_component.get(component) != 1)
    if invalid_active:
        raise ValidationError(f"{path}: exactly one active version required for: {', '.join(invalid_active)}")


def validate_complete_seed_dir(
    seed_dir: Path,
    campus_bbox: tuple[float, float, float, float] | None = None,
) -> list[str]:
    errors: list[str] = []
    try:
        zones = validate_zones(seed_dir / "zones.csv")
    except (OSError, ValidationError) as exc:
        errors.append(str(exc))
        zones = set()
    try:
        locations = validate_locations(seed_dir / "locations.csv", zones, campus_bbox)
    except (OSError, ValidationError) as exc:
        errors.append(str(exc))
        locations = set()
    try:
        validate_posts(seed_dir / "posts.yaml", locations)
    except (OSError, ValidationError) as exc:
        errors.append(str(exc))
    try:
        validate_model_versions(seed_dir / "model_versions.yaml")
    except (OSError, ValidationError) as exc:
        errors.append(str(exc))
    return errors


def validate_seed_dir(
    seed_dir: Path,
    require_complete: bool = False,
    campus_bbox: tuple[float, float, float, float] | None = None,
) -> list[str]:
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
        missing = [filename for filename in COMPLETE_FILES if not (seed_dir / filename).is_file()]
        errors.extend(f"missing complete-seed file: {filename}" for filename in missing)
        if not missing:
            if campus_bbox is None:
                errors.append("complete-seed validation requires --bbox")
            else:
                errors.extend(validate_complete_seed_dir(seed_dir, campus_bbox))
    else:
        present = [filename for filename in COMPLETE_FILES if (seed_dir / filename).is_file()]
        if present:
            errors.append(
                "complete-seed files are present; rerun with --require-complete and --bbox: "
                + ", ".join(present)
            )
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
    parser.add_argument(
        "--bbox",
        type=parse_bbox,
        help="campus bounding box as south,west,north,east (required for a complete seed)",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = validate_seed_dir(
        args.seed_dir,
        require_complete=args.require_complete,
        campus_bbox=args.bbox,
    )
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    mode = "complete" if args.require_complete else "foundation"
    print(f"Seed check passed ({mode}): {args.seed_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
