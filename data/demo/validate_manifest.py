#!/usr/bin/env python3
"""Validate demo photo pack manifest, licensing, and image constraints.

Validates that:
- SOURCES.csv exists and has correct columns
- Every listed file exists in images/ and vice versa
- Total images size does not exceed 150 MB
- Image dimensions meet min 320px and max 1024px longest edge
- Licenses belong to approved open licenses (CC0, CC-BY, CC-BY-SA, Public Domain)
- No unauthorized external sources
"""

from __future__ import annotations

import csv
import sys
from pathlib import Path
from typing import NamedTuple

from PIL import Image

MAX_TOTAL_BYTES = 150 * 1024 * 1024  # 150 MB
MIN_SHORT_EDGE = 320
MAX_LONG_EDGE = 1024

REQUIRED_COLUMNS = [
    "filename",
    "source",
    "author",
    "license",
    "source_url",
    "category",
    "item_type",
    "notes",
]

ALLOWED_SOURCES = {"wikimedia_commons", "derived", "team_fixture", "open_images"}

ALLOWED_LICENSE_SUBSTRINGS = [
    "cc0",
    "public domain",
    "pd",
    "cc-by",
    "cc by",
    "cc-by-sa",
    "cc by-sa",
]


class ValidationResult(NamedTuple):
    ok: bool
    total_images: int
    total_bytes: int
    errors: list[str]


def validate_demo_pack(demo_dir: Path) -> ValidationResult:
    errors: list[str] = []
    manifest_file = demo_dir / "SOURCES.csv"
    images_dir = demo_dir / "images"

    if not manifest_file.is_file():
        return ValidationResult(False, 0, 0, [f"Manifest not found: {manifest_file}"])

    if not images_dir.is_dir():
        return ValidationResult(False, 0, 0, [f"Images directory not found: {images_dir}"])

    rows: list[dict[str, str]] = []
    try:
        with open(manifest_file, "r", encoding="utf-8") as f:
            reader = csv.DictReader(f)
            if reader.fieldnames != REQUIRED_COLUMNS:
                errors.append(
                    f"Invalid CSV header. Expected {REQUIRED_COLUMNS}, got {reader.fieldnames}"
                )
            for i, r in enumerate(reader, start=2):
                rows.append(r)
    except Exception as exc:
        return ValidationResult(False, 0, 0, [f"Error reading {manifest_file}: {exc}"])

    if not rows:
        errors.append("Manifest is empty")
        return ValidationResult(False, 0, 0, errors)

    manifest_filenames = set()
    total_bytes = 0

    for r in rows:
        fn = r.get("filename", "").strip()
        if not fn:
            errors.append("Empty filename in manifest row")
            continue
        manifest_filenames.add(fn)

        src = r.get("source", "").strip()
        if src not in ALLOWED_SOURCES:
            errors.append(f"{fn}: source {src!r} not in approved sources: {ALLOWED_SOURCES}")

        lic = r.get("license", "").strip().lower()
        if not any(k in lic for k in ALLOWED_LICENSE_SUBSTRINGS):
            errors.append(f"{fn}: license {lic!r} is not an approved open license")

        img_path = images_dir / fn
        if not img_path.is_file():
            errors.append(f"{fn}: listed in manifest but file does not exist at {img_path}")
            continue

        file_size = img_path.stat().st_size
        total_bytes += file_size

        if file_size == 0:
            errors.append(f"{fn}: file is 0 bytes")
            continue

        try:
            with Image.open(img_path) as img:
                w, h = img.size
                if min(w, h) < MIN_SHORT_EDGE:
                    errors.append(f"{fn}: dimensions ({w}x{h}) smaller than min {MIN_SHORT_EDGE}px")
                if max(w, h) > MAX_LONG_EDGE:
                    errors.append(f"{fn}: dimensions ({w}x{h}) exceed max {MAX_LONG_EDGE}px")
        except Exception as exc:
            errors.append(f"{fn}: cannot read as valid image: {exc}")

    # Check for unmanifested images
    disk_files = {p.name for p in images_dir.glob("*") if p.is_file()}
    unmanifested = disk_files - manifest_filenames
    if unmanifested:
        errors.append(f"Found unmanifested files in images/: {sorted(unmanifested)}")

    if total_bytes > MAX_TOTAL_BYTES:
        errors.append(
            f"Total images size ({total_bytes} bytes / {total_bytes / 1024 / 1024:.2f} MB) "
            f"exceeds limit ({MAX_TOTAL_BYTES} bytes / 150 MB)"
        )

    return ValidationResult(
        ok=len(errors) == 0,
        total_images=len(manifest_filenames),
        total_bytes=total_bytes,
        errors=errors,
    )


def main() -> None:
    demo_dir = Path("data/demo")
    res = validate_demo_pack(demo_dir)

    print(f"Validated {res.total_images} demo images ({res.total_bytes / 1024 / 1024:.2f} MB)")
    if not res.ok:
        print("Validation errors:", file=sys.stderr)
        for err in res.errors:
            print(f"  - {err}", file=sys.stderr)
        sys.exit(1)
    else:
        print("All demo pack constraints and licenses are valid.")


if __name__ == "__main__":
    main()
