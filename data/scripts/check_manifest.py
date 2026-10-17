#!/usr/bin/env python3
"""Verify file hashes and privacy properties recorded in a dataset manifest."""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any

from intake import sha256_file

try:
    from PIL import Image
except ModuleNotFoundError:  # pragma: no cover - exercised by the CLI error path
    Image = None


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
    if not isinstance(policy, dict) or policy.get("exif_stripped") is not True:
        errors.append("manifest.policy.exif_stripped must be true")
    entries = document.get("files")
    if not isinstance(entries, list) or not entries:
        errors.append("manifest.files must be a non-empty list")
        return errors

    listed: set[str] = set()
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
        ):
            errors.append(f"invalid manifest path: {relative!r}")
            continue
        if relative in listed:
            errors.append(f"duplicate manifest path: {relative}")
            continue
        listed.add(relative)
        path = dataset_dir / relative
        if not path.is_file():
            errors.append(f"manifest file is missing: {relative}")
            continue
        if entry.get("size") != path.stat().st_size:
            errors.append(f"size mismatch: {relative}")
        if entry.get("sha256") != sha256_file(path):
            errors.append(f"sha256 mismatch: {relative}")
        if entry.get("kind") == "image":
            if Image is None:
                errors.append("Pillow is required to check image EXIF metadata")
                continue
            try:
                with Image.open(path) as image:
                    if image.getexif():
                        errors.append(f"EXIF metadata remains: {relative}")
            except OSError as exc:
                errors.append(f"cannot open image {relative}: {exc}")

    actual = {
        path.relative_to(dataset_dir).as_posix()
        for path in dataset_dir.rglob("*")
        if path.is_file() and path.name != "manifest.json"
    }
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
