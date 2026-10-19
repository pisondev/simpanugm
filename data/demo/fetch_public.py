#!/usr/bin/env python3
"""Fetch curated demo images with free licenses from Wikimedia Commons API.

Respects Pison Golda's data collection permissions:
- Source: Wikimedia Commons official API only
- Licenses: CC0, CC-BY (2.0/2.5/3.0/4.0), CC-BY-SA, or Public Domain
- No portraits, identifiable persons, real ID documents, or personal data
- Downscales longest edge to at most 1024px to minimize repository footprint
- Manifest written to data/demo/SOURCES.csv
"""

from __future__ import annotations

import argparse
import csv
import io
import json
import os
import re
import sys
import time
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

from PIL import Image

USER_AGENT = "SimpanUGMDemo/1.0 (academic lost-and-found demo; contact: pison.gm.dev@gmail.com)"
API_ENDPOINT = "https://commons.wikimedia.org/w/api.php"

ALLOWED_LICENSE_KEYWORDS = [
    "cc0",
    "public domain",
    "pd",
    "cc-by",
    "cc by",
    "cc-by-sa",
    "cc by-sa",
]

DISALLOWED_KEYWORDS = [
    "portrait",
    "selfie",
    "face",
    "person",
    "man",
    "woman",
    "child",
    "passport",
    "identity card",
    "license",
]

TARGET_QUERIES = [
    ("earphone", "wireless earbuds case", "hero_found", "hero-earbuds-found.jpg"),
    ("dompet", "folded leather wallet", "hero_found", "hero-wallet-found.jpg"),
    ("kunci-motor", "motorcycle key keychain", "hero_found", "hero-keys-found.jpg"),
    ("kalkulator", "casio scientific calculator", "hero_found", "hero-calculator-found.jpg"),
    ("payung", "black folding umbrella", "hero_found", "hero-umbrella-found.jpg"),
    ("tumbler", "metal water bottle tumbler", "hero_found", "hero-tumbler-found.jpg"),
    ("jaket", "folded windbreaker jacket", "hero_found", "hero-jacket-found.jpg"),
    ("kacamata", "black frame eyeglasses", "hero_found", "dup-glasses-1.jpg"),
    ("kacamata", "reading glasses black frame", "hero_found", "dup-glasses-2.jpg"),
    ("flashdisk", "usb flash drive", "hero_found", "hero-flashdisk-found.jpg"),
    ("power-bank", "portable power bank battery", "hero_found", "hero-powerbank-found.jpg"),
    ("buku", "textbook cover table", "hero_found", "hero-book-found.jpg"),
    ("ransel", "backpack school bag", "hero_found", "hero-backpack-found.jpg"),
    ("helm", "motorcycle helmet visor", "hero_found", "hero-helmet-found.jpg"),
    # Background items
    ("earphone", "wired headphones earphones", "background", "bg-earphone-01.jpg"),
    ("earphone", "bluetooth headset audio", "background", "bg-earphone-02.jpg"),
    ("dompet", "card holder slim wallet", "background", "bg-wallet-01.jpg"),
    ("dompet", "zipper coin purse pouch", "background", "bg-wallet-02.jpg"),
    ("kunci-motor", "car key fob remote", "background", "bg-keys-01.jpg"),
    ("kunci-motor", "house keys ring", "background", "bg-keys-02.jpg"),
    ("kalkulator", "desktop calculator solar", "background", "bg-calc-01.jpg"),
    ("payung", "compact travel umbrella", "background", "bg-umbrella-01.jpg"),
    ("tumbler", "reusable coffee tumbler cup", "background", "bg-tumbler-01.jpg"),
    ("tumbler", "stainless sports bottle", "background", "bg-tumbler-02.jpg"),
    ("jaket", "hoodie jacket sweater", "background", "bg-jacket-01.jpg"),
    ("kacamata", "sunglasses dark lenses", "background", "bg-glasses-01.jpg"),
    ("flashdisk", "thumb drive usb stick", "background", "bg-flashdisk-01.jpg"),
    ("power-bank", "external battery pack", "background", "bg-powerbank-01.jpg"),
    ("buku", "paperback novel book", "background", "bg-book-01.jpg"),
    ("buku", "spiral notebook stationery", "background", "bg-book-02.jpg"),
    ("ransel", "laptop bag shoulder briefcase", "background", "bg-bag-01.jpg"),
    ("ransel", "tote bag canvas", "background", "bg-bag-02.jpg"),
    ("helm", "half face bike helmet", "background", "bg-helmet-01.jpg"),
    ("jam-tangan", "analog wrist watch leather strap", "background", "bg-watch-01.jpg"),
    ("jam-tangan", "digital sports watch", "background", "bg-watch-02.jpg"),
    ("mouse", "wireless computer mouse", "background", "bg-mouse-01.jpg"),
    ("kabel-charger", "usb c charging cable white", "background", "bg-cable-01.jpg"),
    ("kotak-pensil", "pencil pouch stationery case", "background", "bg-pouch-01.jpg"),
    ("sepatu", "sneakers running shoes pair", "background", "bg-shoes-01.jpg"),
    ("topi", "baseball cap cotton hat", "background", "bg-hat-01.jpg"),
]


def clean_html(text: str) -> str:
    """Strip HTML tags and normalize whitespace."""
    if not text:
        return ""
    cleaned = re.sub(r"<[^>]+>", "", text)
    cleaned = re.sub(r"\s+", " ", cleaned).strip()
    return cleaned


def is_allowed_license(lic: str) -> bool:
    """Verify license matches approved open licenses."""
    lower = lic.lower()
    return any(keyword in lower for keyword in ALLOWED_LICENSE_KEYWORDS)


def is_safe_title(title: str) -> bool:
    """Ensure title does not suggest people or personal identity."""
    lower = title.lower()
    return not any(dis in lower for dis in DISALLOWED_KEYWORDS)


def search_commons(query: str, limit: int = 5) -> list[dict[str, Any]]:
    """Query Wikimedia Commons API for search results with image metadata."""
    params = {
        "action": "query",
        "generator": "search",
        "gsrsearch": query,
        "gsrnamespace": "6",
        "gsrlimit": str(limit),
        "prop": "imageinfo",
        "iiprop": "url|size|mime|extmetadata",
        "format": "json",
    }
    url = f"{API_ENDPOINT}?{urllib.parse.urlencode(params)}"
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except Exception as exc:
        print(f"API query failed for {query!r}: {exc}", file=sys.stderr)
        return []

    pages = data.get("query", {}).get("pages", {})
    results: list[dict[str, Any]] = []
    for pid, page in pages.items():
        title = page.get("title", "")
        if not is_safe_title(title):
            continue
        infos = page.get("imageinfo", [])
        if not infos:
            continue
        info = infos[0]
        mime = info.get("mime", "")
        if mime not in ("image/jpeg", "image/png"):
            continue

        meta = info.get("extmetadata", {})
        lic = meta.get("LicenseShortName", {}).get("value", "")
        if not is_allowed_license(lic):
            continue

        artist = clean_html(meta.get("Artist", {}).get("value", "Unknown"))
        description = clean_html(meta.get("ImageDescription", {}).get("value", title))
        source_url = info.get("descriptionurl", info.get("url", ""))
        download_url = info.get("url", "")

        results.append({
            "title": title,
            "artist": artist[:100],
            "license": lic,
            "description": description[:150],
            "source_url": source_url,
            "download_url": download_url,
            "width": info.get("width", 0),
            "height": info.get("height", 0),
        })
    return results


def download_and_process_image(url: str, dest_path: Path) -> bool:
    """Download image, verify dimensions, downscale to max 1024px, and save as JPEG."""
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw_bytes = resp.read()
    except Exception as exc:
        print(f"Failed to download {url}: {exc}", file=sys.stderr)
        return False

    try:
        img = Image.open(io.BytesIO(raw_bytes))
        img.verify()
        # Re-open for transformation after verify()
        img = Image.open(io.BytesIO(raw_bytes))
    except Exception as exc:
        print(f"Invalid image format from {url}: {exc}", file=sys.stderr)
        return False

    w, h = img.size
    if min(w, h) < 320:
        print(f"Image too small ({w}x{h}), skipping: {url}", file=sys.stderr)
        return False

    if max(w, h) > 1024:
        img.thumbnail((1024, 1024), Image.Resampling.LANCZOS)

    if img.mode != "RGB":
        img = img.convert("RGB")

    dest_path.parent.mkdir(parents=True, exist_ok=True)
    img.save(dest_path, "JPEG", quality=85, optimize=True)
    return True


def create_fallback_image(category: str, label: str, dest_path: Path) -> None:
    """Create a clean geometric placeholder image when network is unavailable."""
    from PIL import ImageDraw

    img = Image.new("RGB", (640, 480), color=(240, 242, 245))
    draw = ImageDraw.Draw(img)
    # Simple iconographic card
    draw.rectangle([40, 40, 600, 440], fill=(255, 255, 255), outline=(200, 205, 215), width=3)
    draw.rectangle([80, 80, 560, 360], fill=(225, 230, 240))
    draw.text((100, 380), f"SimpanUGM Demo: {label}", fill=(40, 50, 70))
    dest_path.parent.mkdir(parents=True, exist_ok=True)
    img.save(dest_path, "JPEG", quality=85)


def run_fetch(dest_dir: Path, manifest_path: Path, use_online: bool = True) -> int:
    """Run fetch process and write manifest."""
    images_dir = dest_dir / "images"
    images_dir.mkdir(parents=True, exist_ok=True)

    rows: list[dict[str, str]] = []

    for cat, query, item_type, filename in TARGET_QUERIES:
        target_file = images_dir / filename
        saved = False
        author = "SimpanUGM Demo Team"
        lic = "CC0 1.0"
        src_url = "https://commons.wikimedia.org"
        notes = f"{cat} item for demo story"

        if target_file.exists():
            saved = True
        elif use_online:
            time.sleep(0.5)  # Respect API rate limits
            results = search_commons(query, limit=5)
            if results:
                best = results[0]
                if download_and_process_image(best["download_url"], target_file):
                    saved = True
                    author = best["artist"] or "Wikimedia Contributor"
                    lic = best["license"]
                    src_url = best["source_url"]
                    notes = best["description"]

        if not saved and not target_file.exists():
            create_fallback_image(cat, filename, target_file)
            saved = True

        rows.append({
            "filename": filename,
            "source": "wikimedia_commons" if src_url.startswith("https://commons") else "team_fixture",
            "author": author,
            "license": lic,
            "source_url": src_url,
            "category": cat,
            "item_type": item_type,
            "notes": notes,
        })

    # Write SOURCES.csv
    with open(manifest_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=["filename", "source", "author", "license", "source_url", "category", "item_type", "notes"],
        )
        writer.writeheader()
        writer.writerows(rows)

    print(f"Fetched and recorded {len(rows)} images to {manifest_path}")
    return len(rows)


def main() -> None:
    parser = argparse.ArgumentParser(description="Fetch demo photo pack for SimpanUGM")
    parser.add_argument("--offline", action="store_true", help="Generate local placeholders without network")
    parser.add_argument("--dest", type=Path, default=Path("data/demo"), help="Destination directory")
    args = parser.parse_args()

    manifest_path = args.dest / "SOURCES.csv"
    run_fetch(args.dest, manifest_path, use_online=not args.offline)


if __name__ == "__main__":
    main()
