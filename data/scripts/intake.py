#!/usr/bin/env python3
"""Prepare a private evaluation-dataset directory and write its manifest."""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import re
import shutil
import sys
from pathlib import Path
from typing import Any

try:
    from PIL import Image
except ModuleNotFoundError:  # pragma: no cover - exercised by the CLI error path
    Image = None


METADATA_FILES = ("items.csv", "labels.csv", "distractors.csv", "duplicates.csv", "queries.csv")
IMAGE_RE = re.compile(r"^(finder|owner)[_-]?(\d+)\.(jpg|jpeg|png|webp)$", re.IGNORECASE)
ITEM_RE = re.compile(r"^(?:I|D)\d{3}$")


class IntakeError(ValueError):
    """A safe, actionable intake failure."""


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def canonical_image_path(relative_path: Path) -> Path:
    if (
        len(relative_path.parts) != 3
        or relative_path.parts[0] != "images"
        or not ITEM_RE.fullmatch(relative_path.parts[1])
    ):
        raise IntakeError(
            f"image path must be images/<I|D><number>/<finder|owner>_<number>.<ext>: {relative_path}"
        )
    match = IMAGE_RE.fullmatch(relative_path.parts[2])
    if match is None:
        raise IntakeError(f"unsupported image filename: {relative_path.name}")
    kind, number, extension = match.groups()
    return Path("images") / relative_path.parts[1] / f"{kind.lower()}_{int(number)}.{extension.lower()}"


def strip_exif(source: Path, destination: Path) -> None:
    if Image is None:
        raise IntakeError("Pillow is required for image intake; install data/scripts/requirements.txt")
    destination.parent.mkdir(parents=True, exist_ok=True)
    suffix = destination.suffix.lower()
    with Image.open(source) as image:
        image.load()
        if suffix in {".jpg", ".jpeg"}:
            image = image.convert("RGB")
            image.save(destination, format="JPEG", quality=95, optimize=True, exif=b"")
        elif suffix == ".png":
            image.save(destination, format="PNG", optimize=True)
        elif suffix == ".webp":
            image.save(destination, format="WEBP", quality=95, exif=b"")
        else:  # guarded by IMAGE_RE, kept for defensive use
            raise IntakeError(f"unsupported image extension: {suffix}")


def copy_metadata(source: Path, destination: Path) -> list[Path]:
    copied: list[Path] = []
    for filename in METADATA_FILES:
        source_file = source / filename
        if not source_file.exists():
            continue
        if not source_file.is_file():
            raise IntakeError(f"metadata path is not a file: {source_file}")
        with source_file.open(newline="", encoding="utf-8") as handle:
            list(csv.reader(handle))
        target = destination / filename
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source_file, target)
        copied.append(target)
    if not copied:
        raise IntakeError("source must contain at least one metadata CSV")
    return copied


def copy_images(source: Path, destination: Path) -> list[Path]:
    images_root = source / "images"
    if not images_root.is_dir():
        raise IntakeError("source must contain an images/ directory")
    copied: list[Path] = []
    output_paths: set[Path] = set()
    for image_path in sorted(path for path in images_root.rglob("*") if path.is_file()):
        relative = Path("images") / image_path.relative_to(images_root)
        target_relative = canonical_image_path(relative)
        if target_relative in output_paths:
            raise IntakeError(f"two source images map to the same canonical path: {target_relative}")
        output_paths.add(target_relative)
        target = destination / target_relative
        strip_exif(image_path, target)
        copied.append(target)
    if not copied:
        raise IntakeError("source images/ directory must contain at least one image")
    return copied


def manifest_entry(path: Path, root: Path, kind: str) -> dict[str, Any]:
    relative = path.relative_to(root).as_posix()
    return {
        "path": relative,
        "kind": kind,
        "size": path.stat().st_size,
        "sha256": sha256_file(path),
    }


def intake_dataset(source: Path, destination: Path, version: str) -> Path:
    source = source.resolve()
    destination = destination.resolve()
    if not source.is_dir():
        raise IntakeError(f"source directory does not exist: {source}")
    if destination == source or source in destination.parents:
        raise IntakeError("destination must not be inside the source directory")
    if destination.exists() and any(destination.iterdir()):
        raise IntakeError(f"destination must be empty or absent: {destination}")
    destination.mkdir(parents=True, exist_ok=True)

    metadata = copy_metadata(source, destination)
    images = copy_images(source, destination)
    entries = [manifest_entry(path, destination, "metadata") for path in metadata]
    entries.extend(manifest_entry(path, destination, "image") for path in images)
    entries.sort(key=lambda entry: entry["path"])
    manifest = {
        "dataset": "simpanugm-eval",
        "version": version,
        "generator": "data-intake/1",
        "policy": {"exif_stripped": True, "image_names_normalized": True},
        "files": entries,
    }
    manifest_path = destination / "manifest.json"
    manifest_path.write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    return manifest_path


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True, help="private raw intake directory")
    parser.add_argument("--output", type=Path, required=True, help="private prepared dataset directory")
    parser.add_argument("--version", required=True, help="dataset version label, for example v1 or trial-1")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        manifest = intake_dataset(args.source, args.output, args.version)
    except (OSError, IntakeError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Prepared dataset manifest: {manifest}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
