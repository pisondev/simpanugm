#!/usr/bin/env python3
"""Publish a validated private dataset manifest without copying private files."""

from __future__ import annotations

import argparse
import os
import sys
import tempfile
from pathlib import Path

from check_manifest import check_manifest


class ManifestPublishError(ValueError):
    """A safe, actionable manifest publication failure."""


def publish_manifest(dataset_dir: Path, output: Path, replace: bool = False) -> Path:
    dataset_dir = dataset_dir.resolve()
    if not dataset_dir.is_dir():
        raise ManifestPublishError(f"dataset directory does not exist: {dataset_dir}")

    errors = check_manifest(dataset_dir)
    if errors:
        raise ManifestPublishError("dataset manifest is not valid:\n" + "\n".join(errors))

    source = dataset_dir / "manifest.json"
    if output.is_symlink():
        raise ManifestPublishError(f"output must not be a symlink: {output}")
    output = output.resolve()
    if output == source or source in output.parents:
        raise ManifestPublishError("output must be outside the private dataset directory")
    if output.exists() and not output.is_file():
        raise ManifestPublishError(f"output must be a file or absent: {output}")
    if output.exists() and not replace:
        raise ManifestPublishError(f"output exists; pass --replace to replace it: {output}")
    if not output.parent.is_dir():
        raise ManifestPublishError(f"output parent directory does not exist: {output.parent}")

    payload = source.read_bytes()
    temporary: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="wb", prefix=f".{output.name}.", suffix=".tmp", dir=output.parent, delete=False
        ) as handle:
            temporary = Path(handle.name)
            handle.write(payload)
            handle.flush()
            os.fsync(handle.fileno())
        os.replace(temporary, output)
    finally:
        if temporary is not None and temporary.exists():
            temporary.unlink()
    return output


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dataset-dir", type=Path, required=True, help="private prepared dataset directory")
    parser.add_argument("--output", type=Path, required=True, help="repository manifest path")
    parser.add_argument(
        "--replace",
        action="store_true",
        help="replace an existing manifest explicitly, for a reviewed new dataset version",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        output = publish_manifest(args.dataset_dir, args.output, replace=args.replace)
    except (OSError, ManifestPublishError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Published manifest: {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
