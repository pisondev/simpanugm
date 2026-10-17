#!/usr/bin/env python3
"""Verify file hashes and privacy properties recorded in a dataset manifest."""

from __future__ import annotations

import argparse
import json
import math
import re
import sys
from pathlib import Path
from statistics import median
from typing import Any

from intake import IMAGE_RE, ITEM_RE, METADATA_FILES, canonical_image_path, sha256_file

try:
    from PIL import Image
except ModuleNotFoundError:  # pragma: no cover - exercised by the CLI error path
    Image = None


ALLOWED_KINDS = {"metadata", "image"}
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
MIN_IMAGE_SHORT_SIDE = 1080
PHASH_GRID_SIZE = 32
PHASH_LOW_FREQUENCY = 8
PHASH_DISTANCE_THRESHOLD = 4
DCT_BASIS = tuple(
    tuple(
        math.cos((2 * position + 1) * frequency * math.pi / (2 * PHASH_GRID_SIZE))
        for position in range(PHASH_GRID_SIZE)
    )
    for frequency in range(PHASH_LOW_FREQUENCY)
)


def perceptual_hash(image: Any) -> int:
    """Return a compact DCT perceptual hash for cross-item near-duplicate checks."""
    resized = image.convert("L").resize(
        (PHASH_GRID_SIZE, PHASH_GRID_SIZE),
        resample=Image.Resampling.LANCZOS,
    )
    pixels = [
        [resized.getpixel((x, y)) for x in range(PHASH_GRID_SIZE)]
        for y in range(PHASH_GRID_SIZE)
    ]
    coefficients: list[float] = []
    for vertical in range(PHASH_LOW_FREQUENCY):
        for horizontal in range(PHASH_LOW_FREQUENCY):
            total = sum(
                pixels[y][x] * DCT_BASIS[vertical][y] * DCT_BASIS[horizontal][x]
                for y in range(PHASH_GRID_SIZE)
                for x in range(PHASH_GRID_SIZE)
            )
            vertical_scale = 1 / math.sqrt(2) if vertical == 0 else 1
            horizontal_scale = 1 / math.sqrt(2) if horizontal == 0 else 1
            coefficients.append(0.25 * vertical_scale * horizontal_scale * total)

    threshold = median(coefficients[1:])
    return sum(
        1 << index
        for index, coefficient in enumerate(coefficients[1:])
        if coefficient > threshold
    )


def check_manifest(dataset_dir: Path) -> list[str]:
    errors: list[str] = []
    manifest_path = dataset_dir / "manifest.json"
    if not manifest_path.is_file():
        return [f"missing manifest: {manifest_path}"]
    try:
        document: Any = json.loads(manifest_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        return [f"cannot read manifest: {exc}"]
    if not isinstance(document, dict):
        return ["manifest must be a JSON object"]
    if document.get("dataset") != "simpanugm-eval":
        errors.append("manifest.dataset must be simpanugm-eval")
    if not isinstance(document.get("version"), str) or not document["version"]:
        errors.append("manifest.version must be a non-empty string")
    policy = document.get("policy")
    if not isinstance(policy, dict):
        errors.append("manifest.policy must be an object")
    else:
        if policy.get("exif_stripped") is not True:
            errors.append("manifest.policy.exif_stripped must be true")
        if policy.get("image_names_normalized") is not True:
            errors.append("manifest.policy.image_names_normalized must be true")
    entries = document.get("files")
    if not isinstance(entries, list) or not entries:
        errors.append("manifest.files must be a non-empty list")
        return errors

    listed: set[str] = set()
    image_hashes: dict[str, str] = {}
    image_phashes: dict[str, int] = {}
    for entry in entries:
        if not isinstance(entry, dict):
            errors.append("manifest.files entries must be objects")
            continue
        relative = entry.get("path")
        path_value = Path(relative) if isinstance(relative, str) else Path(".")
        if (
            not isinstance(relative, str)
            or not relative
            or path_value.is_absolute()
            or ".." in path_value.parts
            or "\\" in relative
            or path_value.as_posix() != relative
        ):
            errors.append(f"invalid manifest path: {relative!r}")
            continue
        if relative in listed:
            errors.append(f"duplicate manifest path: {relative}")
            continue
        listed.add(relative)
        kind = entry.get("kind")
        if kind not in ALLOWED_KINDS:
            errors.append(f"invalid manifest kind for {relative}: {kind!r}")
        elif kind == "metadata" and relative not in METADATA_FILES:
            errors.append(f"invalid metadata manifest path: {relative}")
        elif kind == "image":
            try:
                normalized = canonical_image_path(path_value)
            except ValueError:
                errors.append(f"invalid image manifest path: {relative}")
            else:
                if normalized != path_value:
                    errors.append(f"image manifest path is not normalized: {relative}")
        if kind == "image" and (
            len(path_value.parts) != 3
            or path_value.parts[0] != "images"
            or not ITEM_RE.fullmatch(path_value.parts[1])
            or IMAGE_RE.fullmatch(path_value.parts[2]) is None
        ):
            errors.append(f"invalid image manifest path: {relative}")
        path = dataset_dir / relative
        if path.is_symlink():
            errors.append(f"manifest file must not be a symlink: {relative}")
            continue
        if not path.is_file():
            errors.append(f"manifest file is missing: {relative}")
            continue
        size = entry.get("size")
        if type(size) is not int or size < 0:
            errors.append(f"invalid manifest size: {relative}")
        elif size != path.stat().st_size:
            errors.append(f"size mismatch: {relative}")
        actual_sha256: str | None = None
        sha256 = entry.get("sha256")
        if not isinstance(sha256, str) or SHA256_RE.fullmatch(sha256) is None:
            errors.append(f"invalid manifest sha256: {relative}")
        else:
            actual_sha256 = sha256_file(path)
            if sha256 != actual_sha256:
                errors.append(f"sha256 mismatch: {relative}")
        if kind == "image":
            if actual_sha256 is not None:
                previous = image_hashes.get(actual_sha256)
                if previous is not None:
                    errors.append(f"duplicate image content: {relative} matches {previous}")
                else:
                    image_hashes[actual_sha256] = relative
            if Image is None:
                errors.append("Pillow is required to check image dimensions and EXIF metadata")
                continue
            try:
                with Image.open(path) as image:
                    width, height = image.size
                    if min(width, height) < MIN_IMAGE_SHORT_SIDE:
                        errors.append(
                            f"image short side must be at least {MIN_IMAGE_SHORT_SIDE} px: {relative}"
                        )
                    if image.getexif():
                        errors.append(f"EXIF metadata remains: {relative}")
                    image_phashes[relative] = perceptual_hash(image)
            except OSError as exc:
                errors.append(f"cannot open image {relative}: {exc}")

    image_paths = sorted(image_phashes)
    for index, first_path in enumerate(image_paths):
        first_record = Path(first_path).parts[1] if len(Path(first_path).parts) > 1 else first_path
        for second_path in image_paths[index + 1 :]:
            second_record = (
                Path(second_path).parts[1] if len(Path(second_path).parts) > 1 else second_path
            )
            if first_record == second_record:
                continue
            distance = (image_phashes[first_path] ^ image_phashes[second_path]).bit_count()
            if distance <= PHASH_DISTANCE_THRESHOLD:
                errors.append(
                    f"near-duplicate image content: {first_path} and {second_path} "
                    f"(perceptual hash distance {distance} <= {PHASH_DISTANCE_THRESHOLD})"
                )

    actual: set[str] = set()
    for path in dataset_dir.rglob("*"):
        relative = path.relative_to(dataset_dir).as_posix()
        if path.is_symlink():
            if relative not in listed:
                errors.append(f"unlisted symlink: {relative}")
            continue
        if path.is_file() and path.name != "manifest.json":
            actual.add(relative)
    for relative in sorted(actual - listed):
        errors.append(f"file is not listed in manifest: {relative}")
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset-dir", type=Path, required=True)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    errors = check_manifest(args.dataset_dir)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    print(f"Manifest check passed: {args.dataset_dir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
