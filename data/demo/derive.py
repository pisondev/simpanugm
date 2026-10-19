#!/usr/bin/env python3
"""Derive second photos for hero lost/found pairs.

Per demo-first-plan.md section 4:
If a second photograph is not directly provided, derive a realistic variation
(crop, angle, lighting adjustments) of the same item and document as derived.
"""

from __future__ import annotations

import argparse
import csv
from pathlib import Path

from PIL import Image, ImageEnhance


def derive_image(src_path: Path, dest_path: Path) -> bool:
    """Generate a realistic alternative angle/crop variation of an item photo."""
    try:
        img = Image.open(src_path)
        if img.mode != "RGB":
            img = img.convert("RGB")
    except Exception as exc:
        print(f"Error opening source image {src_path}: {exc}")
        return False

    w, h = img.size
    # 1. Subtle zoom crop (center 88%)
    crop_w = int(w * 0.88)
    crop_h = int(h * 0.88)
    left = (w - crop_w) // 2
    top = (h - crop_h) // 2
    cropped = img.crop((left, top, left + crop_w, top + crop_h))

    # 2. Subtle rotation (2 degrees) with bicubic interpolation
    rotated = cropped.rotate(2.0, resample=Image.Resampling.BICUBIC, expand=False)

    # 3. Slight brightness and contrast adjustment to simulate different room lighting
    brightener = ImageEnhance.Brightness(rotated)
    adjusted = brightener.enhance(1.04)
    contrast = ImageEnhance.Contrast(adjusted)
    final_img = contrast.enhance(0.98)

    # Ensure max 1024px
    fw, fh = final_img.size
    if max(fw, fh) > 1024:
        final_img.thumbnail((1024, 1024), Image.Resampling.LANCZOS)

    dest_path.parent.mkdir(parents=True, exist_ok=True)
    final_img.save(dest_path, "JPEG", quality=85, optimize=True)
    return True


def run_derive(demo_dir: Path) -> int:
    """Derive lost versions for all hero found reports and update SOURCES.csv."""
    manifest_path = demo_dir / "SOURCES.csv"
    images_dir = demo_dir / "images"

    if not manifest_path.exists():
        print(f"Manifest not found: {manifest_path}")
        return 0

    rows: list[dict[str, str]] = []
    with open(manifest_path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for r in reader:
            rows.append(r)

    derived_rows: list[dict[str, str]] = []
    count = 0

    for r in rows:
        fn = r["filename"]
        if r["item_type"] == "hero_found" and fn.endswith("-found.jpg"):
            lost_fn = fn.replace("-found.jpg", "-lost.jpg")
            src_file = images_dir / fn
            dest_file = images_dir / lost_fn

            if src_file.exists():
                derive_image(src_file, dest_file)
                derived_rows.append({
                    "filename": lost_fn,
                    "source": "derived",
                    "author": f"derived_from:{fn}",
                    "license": r["license"],
                    "source_url": r["source_url"],
                    "category": r["category"],
                    "item_type": "hero_lost",
                    "notes": "Derived crop, lighting and perspective shift of hero object",
                })
                count += 1

    # Merge rows avoiding duplicates
    existing_filenames = {r["filename"] for r in rows}
    for dr in derived_rows:
        if dr["filename"] not in existing_filenames:
            rows.append(dr)

    with open(manifest_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=["filename", "source", "author", "license", "source_url", "category", "item_type", "notes"],
        )
        writer.writeheader()
        writer.writerows(rows)

    print(f"Derived {count} hero pair variations, total manifest entries: {len(rows)}")
    return count


def main() -> None:
    parser = argparse.ArgumentParser(description="Derive second photos for hero pairs")
    parser.add_argument("--demo-dir", type=Path, default=Path("data/demo"), help="Demo directory")
    args = parser.parse_args()
    run_derive(args.demo_dir)


if __name__ == "__main__":
    main()
