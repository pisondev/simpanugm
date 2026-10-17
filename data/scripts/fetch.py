#!/usr/bin/env python3
"""Fetch manifest-listed evaluation files from a local private dataset copy."""

from __future__ import annotations

import argparse
import json
import os
import shutil
import sys
import tempfile
from pathlib import Path
from typing import Any

from check_manifest import check_manifest
from intake import sha256_file


class FetchError(ValueError):
    """A safe, actionable dataset fetch failure."""


def _relative_path(value: Any, label: str) -> Path:
    if not isinstance(value, str) or not value:
        raise FetchError(f"{label} must be a non-empty relative path")
    path = Path(value)
    if (
        path.is_absolute()
        or ".." in path.parts
        or "\\" in value
        or path.as_posix() != value
    ):
        raise FetchError(f"{label} is not a safe relative path: {value!r}")
    return path


def _load_manifest(manifest_path: Path) -> tuple[bytes, list[tuple[str, int, str]]]:
    if manifest_path.is_symlink() or not manifest_path.is_file():
        raise FetchError(f"manifest does not exist as a regular file: {manifest_path}")
    try:
        payload = manifest_path.read_bytes()
        document: Any = json.loads(payload)
    except (OSError, json.JSONDecodeError) as exc:
        raise FetchError(f"cannot read manifest: {exc}") from exc
    if not isinstance(document, dict):
        raise FetchError("manifest must be a JSON object")
    entries = document.get("files")
    if not isinstance(entries, list) or not entries:
        raise FetchError("manifest.files must be a non-empty list")

    files: list[tuple[str, int, str]] = []
    seen: set[str] = set()
    for entry in entries:
        if not isinstance(entry, dict):
            raise FetchError("manifest.files entries must be objects")
        relative = _relative_path(entry.get("path"), "manifest file path")
        relative_text = relative.as_posix()
        if relative_text in seen:
            raise FetchError(f"duplicate manifest path: {relative_text}")
        seen.add(relative_text)
        size = entry.get("size")
        if type(size) is not int or size < 0:
            raise FetchError(f"invalid manifest size: {relative_text}")
        sha256 = entry.get("sha256")
        if not isinstance(sha256, str) or len(sha256) != 64:
            raise FetchError(f"invalid manifest sha256: {relative_text}")
        files.append((relative_text, size, sha256))
    return payload, files


def _private_source_path(source_root: Path, relative: str) -> Path:
    source_path = source_root / relative
    current = source_root
    for part in Path(relative).parts:
        current /= part
        if current.is_symlink():
            raise FetchError(f"source path must not contain symlinks: {relative}")
    try:
        resolved = source_path.resolve(strict=True)
    except OSError as exc:
        raise FetchError(f"source file is unavailable: {relative}: {exc}") from exc
    if source_root not in resolved.parents:
        raise FetchError(f"source path escapes the source directory: {relative}")
    if not resolved.is_file():
        raise FetchError(f"source path is not a regular file: {relative}")
    return resolved


def _validate_source(source_root: Path, files: list[tuple[str, int, str]]) -> dict[str, Path]:
    if source_root.is_symlink() or not source_root.is_dir():
        raise FetchError(f"source directory does not exist as a regular directory: {source_root}")
    source_root = source_root.resolve()
    validated: dict[str, Path] = {}
    for relative, expected_size, expected_sha256 in files:
        source_path = _private_source_path(source_root, relative)
        actual_size = source_path.stat().st_size
        if actual_size != expected_size:
            raise FetchError(
                f"source size mismatch for {relative}: expected {expected_size}, got {actual_size}"
            )
        actual_sha256 = sha256_file(source_path)
        if actual_sha256 != expected_sha256:
            raise FetchError(f"source sha256 mismatch for {relative}")
        validated[relative] = source_path
    return validated


def _check_destination(output_dir: Path, files: list[tuple[str, int, str]]) -> None:
    if output_dir.is_symlink():
        raise FetchError(f"output directory must not be a symlink: {output_dir}")
    if output_dir.exists() and not output_dir.is_dir():
        raise FetchError(f"output path must be a directory or absent: {output_dir}")
    listed = {relative for relative, _, _ in files}
    if not output_dir.exists():
        return
    for path in output_dir.rglob("*"):
        relative = path.relative_to(output_dir).as_posix()
        if path.is_symlink():
            raise FetchError(f"output must not contain symlinks: {relative}")
        if path.is_file() and relative not in listed and relative not in {"README.md", "manifest.json"}:
            raise FetchError(f"output contains an unlisted file; use a fresh directory: {relative}")
    for relative, _, _ in files:
        current = output_dir
        for part in Path(relative).parts[:-1]:
            current /= part
            if current.is_symlink():
                raise FetchError(f"output path must not contain symlinks: {relative}")
        destination = output_dir / relative
        if destination.is_symlink():
            raise FetchError(f"output file must not be a symlink: {relative}")


def fetch_dataset(manifest: Path, source: Path, output: Path) -> Path:
    """Copy and verify a manifest-listed dataset into ``output``."""
    manifest = manifest.absolute()
    source = source.absolute()
    output = output.absolute()
    manifest_payload, files = _load_manifest(manifest)
    validated_source = _validate_source(source, files)
    _check_destination(output, files)
    output.parent.mkdir(parents=True, exist_ok=True)

    staging: Path | None = None
    try:
        with tempfile.TemporaryDirectory(prefix=f".{output.name}.fetch-", dir=output.parent) as temporary:
            staging = Path(temporary)
            (staging / "manifest.json").write_bytes(manifest_payload)
            for relative, _, _ in files:
                target = staging / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(validated_source[relative], target)
            errors = check_manifest(staging)
            if errors:
                raise FetchError("staged dataset failed manifest validation:\n" + "\n".join(errors))

            output.mkdir(parents=True, exist_ok=True)
            for relative, _, _ in [("manifest.json", len(manifest_payload), ""), *files]:
                staged_path = staging / relative
                destination = output / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                temporary_destination: Path | None = None
                try:
                    with tempfile.NamedTemporaryFile(
                        mode="wb", prefix=f".{destination.name}.", suffix=".tmp", dir=destination.parent, delete=False
                    ) as handle:
                        temporary_destination = Path(handle.name)
                        handle.write(staged_path.read_bytes())
                        handle.flush()
                        os.fsync(handle.fileno())
                    os.replace(temporary_destination, destination)
                finally:
                    if temporary_destination is not None and temporary_destination.exists():
                        temporary_destination.unlink()
    except OSError as exc:
        raise FetchError(str(exc)) from exc

    for relative, expected_size, expected_sha256 in files:
        destination = output / relative
        if destination.stat().st_size != expected_size or sha256_file(destination) != expected_sha256:
            raise FetchError(f"installed file failed verification: {relative}")
    return output


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=Path, required=True, help="reviewed manifest.json in the repository")
    parser.add_argument("--source", type=Path, required=True, help="local copy or mounted folder of the private dataset")
    parser.add_argument("--output", type=Path, required=True, help="local evaluation dataset directory")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        output = fetch_dataset(args.manifest, args.source, args.output)
    except (OSError, FetchError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Evaluation dataset fetched and verified: {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
