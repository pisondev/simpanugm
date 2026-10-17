#!/usr/bin/env python3
"""Create deterministic, synthetic image fixtures for media and analysis tests."""

from __future__ import annotations

import argparse
import hashlib
import os
import sys
import tempfile
from pathlib import Path

from PIL import Image, ImageDraw


ASSET_NAMES = ("swatch-biru.png", "swatch-merah.png", "desk-gps.jpg", "mock-card.png")

EXPECTED_SHA256 = {
    "swatch-biru.png": "0e2d08ea439102af9a98f96f2e96d44bf11c1fbeca6e143c91ae79bb99de7112",
    "swatch-merah.png": "f3e26572a7bce62546f17139c4e891d9465f2b19c46a70014a43b6984cba44e5",
    "desk-gps.jpg": "1b4c3b86f4b8d9c1336785f6caf19811b6f40fc7fa5cf63f767b176663e21e1a",
    "mock-card.png": "5d012f65934450a713849f0b3bb6a288412d1adeccd650023f8d6bd0e52d9283",
}


def _draw_desk() -> Image.Image:
    image = Image.new("RGB", (320, 240), (214, 184, 142))
    draw = ImageDraw.Draw(image)
    draw.rectangle((0, 0, 319, 55), fill=(181, 137, 92))
    draw.rectangle((35, 75, 205, 205), fill=(242, 242, 238), outline=(60, 60, 60), width=2)
    draw.rectangle((55, 92, 185, 185), fill=(205, 224, 235))
    draw.rectangle((225, 92, 280, 184), fill=(90, 62, 40), outline=(45, 30, 20), width=3)
    draw.ellipse((232, 78, 273, 112), fill=(235, 235, 230), outline=(45, 30, 20), width=2)
    draw.line((250, 112, 250, 192), fill=(45, 30, 20), width=3)
    return image


def _draw_mock_card() -> Image.Image:
    image = Image.new("RGB", (400, 240), (236, 243, 250))
    draw = ImageDraw.Draw(image)
    draw.rectangle((0, 0, 399, 42), fill=(24, 91, 145))
    draw.text((18, 12), "CONTOH", fill=(255, 255, 255))
    draw.rectangle((24, 65, 118, 190), fill=(205, 221, 232), outline=(50, 70, 85), width=2)
    draw.rectangle((38, 80, 104, 145), fill=(185, 205, 218))
    draw.text((145, 72), "KARTU SINTETIS", fill=(24, 91, 145))
    draw.text((145, 112), "NARA WIJAYA", fill=(35, 35, 35))
    draw.text((145, 145), "0000123456789012", fill=(35, 35, 35))
    draw.text((145, 178), "BUKAN DOKUMEN ASLI", fill=(165, 45, 45))
    return image


def _save_desk(path: Path) -> None:
    image = _draw_desk()
    exif = Image.Exif()
    exif[34853] = {
        1: "S",
        2: (7.775, 46.0, 30.0),
        3: "E",
        4: (110.3786, 22.0, 43.0),
    }
    image.save(path, format="JPEG", quality=95, exif=exif.tobytes())


def create_fixture_images(output: Path) -> list[Path]:
    if output.is_symlink():
        raise OSError(f"output must not be a symlink: {output}")
    output.mkdir(parents=True, exist_ok=True)
    created: list[Path] = []
    with tempfile.TemporaryDirectory(prefix=".fixture-images-", dir=output.parent) as staging_name:
        staging = Path(staging_name)
        Image.new("RGB", (64, 64), (35, 119, 190)).save(staging / "swatch-biru.png", format="PNG")
        Image.new("RGB", (64, 64), (210, 55, 55)).save(staging / "swatch-merah.png", format="PNG")
        _save_desk(staging / "desk-gps.jpg")
        _draw_mock_card().save(staging / "mock-card.png", format="PNG")
        for name in ASSET_NAMES:
            source = staging / name
            destination = output / name
            os.replace(source, destination)
            created.append(destination)
    return created


def check_fixture_images(output: Path) -> list[str]:
    """Compare committed assets with their reviewed integrity hashes."""
    errors: list[str] = []
    for name in ASSET_NAMES:
        committed = output / name
        if not committed.is_file():
            errors.append(f"missing fixture image: {committed}")
            continue
        actual = hashlib.sha256(committed.read_bytes()).hexdigest()
        expected = EXPECTED_SHA256[name]
        if actual != expected:
            errors.append(
                f"fixture image hash differs from reviewed asset: {committed} "
                f"(expected {expected}, got {actual})"
            )
    return errors


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=Path(__file__).resolve().parents[1] / "fixtures" / "images",
        help="fixture image directory",
    )
    parser.add_argument("--check", action="store_true", help="verify committed assets match reviewed hashes")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(sys.argv[1:] if argv is None else argv)
    if args.check:
        errors = check_fixture_images(args.output)
        if errors:
            for error in errors:
                print(f"ERROR: {error}")
            return 1
        print(f"Fixture images: OK ({args.output})")
        return 0
    for path in create_fixture_images(args.output):
        print(f"Created {path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
