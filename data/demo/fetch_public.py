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
import hashlib
import io
import json
import os
import re
import sys
import tempfile
import time
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

from PIL import Image

USER_AGENT = "SimpanUGMDemo/1.0 (academic lost-and-found demo; contact: pison.gm.dev@gmail.com)"
API_ENDPOINT = "https://commons.wikimedia.org/w/api.php"
MAX_TARGET = 300
MAX_DOWNLOAD_BYTES = 25 * 1024 * 1024
MAX_PIXELS = 50_000_000

DISALLOWED_KEYWORDS = [
    "portrait",
    "selfie",
    "face",
    "people",
    "person",
    "man",
    "woman",
    "women",
    "boy",
    "girl",
    "kid",
    "hiker",
    "protester",
    "student",
    "female",
    "male",
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

BACKGROUND_SEARCHES = {
    "mouse": ["wireless computer mouse", "computer mouse isolated", "optical mouse device", "computer accessory mouse"],
    "kabel-charger": ["USB charging cable product", "USB C charging cable coiled", "phone charging cord isolated", "power cable connector close up"],
    "kotak-pensil": ["pencil case pouch product", "zipper pencil pouch isolated", "school pencil box stationery", "empty pencil case on desk"],
    "sepatu": ["sneaker shoes product photo", "running shoes pair isolated", "casual shoes on white background", "leather footwear product"],
    "topi": ["baseball cap product photo", "cotton cap laid flat", "fabric hat isolated", "cap on a plain table", "beanie hat product", "visor cap product"],
    "kunci-motor": ["motorcycle ignition key close up", "motorbike key fob product", "scooter ignition key and ring", "motorcycle remote key isolated"],
    "kunci-lain": ["house key set on table", "single metal house key isolated", "keys with keyring product photo", "door key on plain background"],
    "tumbler": ["insulated stainless steel travel tumbler", "vacuum insulated tumbler with lid", "reusable travel mug product photo", "personal thermal water flask"],
    "ransel": ["backpack product photo isolated", "school backpack on a plain background", "daypack product photo", "empty campus backpack laid flat"],
    "tote-bag": ["canvas tote bag product photo", "reusable tote bag isolated", "cloth shopping bag on plain background", "empty shoulder tote bag"],
    "jaket": ["folded jacket garment product", "zippered jacket on a hanger", "windbreaker jacket isolated", "empty jacket laid flat"],
    "payung": ["folded rain umbrella product", "compact travel umbrella isolated", "open umbrella product photo", "umbrella handle close up"],
    "earphone": ["wireless earbuds case", "wired earphones product photo", "bluetooth headset isolated", "in ear headphones on plain background"],
    "dompet": ["leather wallet product photo", "zip coin purse isolated", "folded billfold wallet", "small card wallet on table"],
    "kalkulator": ["scientific calculator product photo", "school calculator isolated", "handheld calculator on white background", "desktop calculator product"],
    "kacamata": ["eyeglasses frame product photo", "reading glasses isolated", "sunglasses on plain table", "spectacles frame close up"],
    "flashdisk": ["USB flash drive product photo", "thumb drive isolated", "portable USB stick", "memory stick device close up"],
    "power-bank": ["portable power bank product photo", "external battery pack isolated", "USB power bank device", "phone battery pack on table"],
    "buku": ["closed textbook cover product photo", "paperback book cover", "notebook on plain desk", "school textbook isolated"],
    "helm": ["motorcycle helmet product display", "bicycle helmet isolated", "half face helmet product photo", "helmet on a plain shelf"],
    "jam-tangan": ["analog wristwatch product photo", "digital wrist watch isolated", "watch with strap on table", "wristwatch close up"],
}

CATEGORY_TERMS = {
    "earphone": ("earphone", "headphone", "headset", "earbud", "audio"),
    "dompet": ("wallet", "purse", "billfold", "card holder"),
    "kunci-motor": ("key", "ignition", "fob"),
    "kalkulator": ("calculator",),
    "payung": ("umbrella",),
    "tumbler": ("tumbler", "travel mug", "thermos", "flask", "water bottle"),
    "jaket": ("jacket", "coat", "anorak", "windbreaker"),
    "kacamata": ("glasses", "spectacles", "eyeglasses", "sunglasses"),
    "flashdisk": ("flash drive", "thumb drive", "usb stick", "pendrive", "usb flash"),
    "power-bank": ("power bank", "powerbank", "battery pack", "external battery"),
    "buku": ("book", "textbook", "notebook"),
    "ransel": ("backpack", "rucksack", "daypack"),
    "helm": ("helmet",),
    "jam-tangan": ("watch", "wristwatch"),
    "mouse": ("mouse",),
    "kabel-charger": ("charging cable", "usb cable", "power cable", "cable"),
    "kotak-pensil": ("pencil case", "pencil pouch", "pencil box", "stationery case"),
    "sepatu": ("shoe", "shoes", "sneaker", "footwear"),
    "topi": ("cap", "hat"),
}


def clean_html(text: str) -> str:
    """Strip HTML tags and normalize whitespace."""
    if not text:
        return ""
    cleaned = re.sub(r"<[^>]+>", "", text)
    cleaned = re.sub(r"\s+", " ", cleaned).strip()
    return cleaned


def is_allowed_license(lic: str) -> bool:
    """Accept only CC0, CC-BY, CC-BY-SA and public domain labels."""
    value = re.sub(r"\s+", " ", lic.strip().lower()).replace("_", " ")
    if value in {"public domain", "pd", "cc0", "cc0 1.0"}:
        return True
    return re.fullmatch(r"cc[ -]by(?:[ -]sa)?(?:\s+[0-9.]+)?", value) is not None


def is_safe_title(title: str, description: str = "") -> bool:
    """Reject metadata that suggests people, identity documents or private text."""
    lower = f"{title} {description}".lower()
    return not any(dis in lower for dis in DISALLOWED_KEYWORDS)


def matches_category(result: dict[str, Any], category: str) -> bool:
    terms = CATEGORY_TERMS.get(category, ())
    text = f"{result.get('title', '')} {result.get('description', '')}".lower().replace("_", " ")
    return any(term in text for term in terms)


def search_commons(query: str, limit: int = 5, offset: int = 0) -> list[dict[str, Any]]:
    """Query Wikimedia Commons API for search results with image metadata."""
    params = {
        "action": "query",
        "generator": "search",
        "gsrsearch": query,
        "gsrnamespace": "6",
        "gsrlimit": str(limit),
        "gsroffset": str(offset),
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
        infos = page.get("imageinfo", [])
        if not infos:
            continue
        info = infos[0]
        mime = info.get("mime", "")
        if mime not in ("image/jpeg", "image/png"):
            continue
        width, height = int(info.get("width", 0)), int(info.get("height", 0))
        if width < 320 or height < 320 or width * height > MAX_PIXELS:
            continue

        meta = info.get("extmetadata", {})
        lic = meta.get("LicenseShortName", {}).get("value", "")
        if not is_allowed_license(lic):
            continue

        artist = clean_html(meta.get("Artist", {}).get("value", "Unknown"))
        description = clean_html(meta.get("ImageDescription", {}).get("value", title))
        if not is_safe_title(title, description):
            continue
        source_url = info.get("descriptionurl", info.get("url", ""))
        download_url = info.get("url", "")

        results.append({
            "title": title,
            "artist": artist[:100],
            "license": lic,
            "description": description[:150],
            "source_url": source_url,
            "download_url": download_url,
            "width": width,
            "height": height,
        })
    return results


def download_and_process_image(url: str, dest_path: Path) -> bool:
    """Download image, verify dimensions, downscale to max 1024px, and save as JPEG."""
    req = urllib.request.Request(url, headers={"User-Agent": USER_AGENT})
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw_bytes = resp.read(MAX_DOWNLOAD_BYTES + 1)
            if len(raw_bytes) > MAX_DOWNLOAD_BYTES:
                print(f"Image download exceeds the 25 MB source limit: {url}", file=sys.stderr)
                return False
    except Exception as exc:
        print(f"Failed to download {url}: {exc}", file=sys.stderr)
        return False

    try:
        img = Image.open(io.BytesIO(raw_bytes))
        if img.width * img.height > MAX_PIXELS:
            print(f"Image exceeds the {MAX_PIXELS} pixel safety limit: {url}", file=sys.stderr)
            return False
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
    with tempfile.NamedTemporaryFile(dir=dest_path.parent, suffix=".jpg", delete=False) as tmp:
        temp_path = Path(tmp.name)
    try:
        img.save(temp_path, "JPEG", quality=85, optimize=True)
        with Image.open(temp_path) as check:
            check.verify()
        os.replace(temp_path, dest_path)
    finally:
        temp_path.unlink(missing_ok=True)
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


FIELDS = ["filename", "source", "author", "license", "source_url", "category", "item_type", "notes"]
REJECTED_FIELDS = ["filename", "source_url", "reason"]


def read_manifest(path: Path) -> list[dict[str, str]]:
    if not path.is_file():
        return []
    with path.open("r", newline="", encoding="utf-8") as file:
        return list(csv.DictReader(file))


def write_manifest(path: Path, rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", newline="", encoding="utf-8", dir=path.parent, delete=False) as tmp:
        writer = csv.DictWriter(tmp, fieldnames=FIELDS)
        writer.writeheader()
        writer.writerows(rows)
        temp_path = Path(tmp.name)
    os.replace(temp_path, path)


def read_rejected(path: Path) -> list[dict[str, str]]:
    if not path.is_file():
        return []
    with path.open("r", newline="", encoding="utf-8") as file:
        return list(csv.DictReader(file))


def fixture_row(filename: str, category: str, item_type: str) -> dict[str, str]:
    return {
        "filename": filename,
        "source": "team_fixture",
        "author": "SimpanUGM Demo Team",
        "license": "CC0 1.0",
        "source_url": "",
        "category": category,
        "item_type": item_type,
        "notes": "Locally drawn placeholder, not a photograph or a Wikimedia download",
    }


def result_row(result: dict[str, Any], filename: str, category: str, item_type: str) -> dict[str, str]:
    return {
        "filename": filename,
        "source": "wikimedia_commons",
        "author": result["artist"] or "Wikimedia Contributor",
        "license": result["license"],
        "source_url": result["source_url"],
        "category": category,
        "item_type": item_type,
        "notes": result["description"],
    }


def _unique_state(rows: list[dict[str, str]], images_dir: Path) -> tuple[set[str], set[str], set[str]]:
    filenames = {row["filename"] for row in rows}
    sources = {row["source_url"] for row in rows if row.get("source_url")}
    hashes = set()
    for filename in filenames:
        path = images_dir / filename
        if path.is_file():
            hashes.add(hashlib.sha256(path.read_bytes()).hexdigest())
    return filenames, sources, hashes


def _next_background_filename(rows: list[dict[str, str]], category: str) -> str:
    pattern = re.compile(rf"^bg-{re.escape(category)}-(\d+)\.jpg$")
    indices = [int(match.group(1)) for row in rows if (match := pattern.match(row["filename"]))]
    return f"bg-{category}-{max(indices, default=0) + 1:03d}.jpg"


def run_fetch(
    dest_dir: Path,
    manifest_path: Path,
    use_online: bool = True,
    target: int | None = None,
) -> int:
    """Add licensed images without replacing existing files or manifest rows."""
    if target is not None and not 1 <= target <= MAX_TARGET:
        raise ValueError(f"target must be between 1 and {MAX_TARGET}")
    images_dir = dest_dir / "images"
    images_dir.mkdir(parents=True, exist_ok=True)
    rows = read_manifest(manifest_path)
    filenames, source_urls, content_hashes = _unique_state(rows, images_dir)
    rejected_urls = {row["source_url"] for row in read_rejected(dest_dir / "REJECTED.csv")}
    source_urls.update(rejected_urls)

    if target is None:
        for category, query, item_type, filename in TARGET_QUERIES:
            if filename in filenames:
                continue
            target_file = images_dir / filename
            if target_file.exists():
                continue
            row = None
            if use_online:
                time.sleep(0.5)
                for candidate in search_commons(query, limit=20):
                    if not matches_category(candidate, category):
                        continue
                    if candidate["source_url"] in source_urls:
                        continue
                    if download_and_process_image(candidate["download_url"], target_file):
                        digest = hashlib.sha256(target_file.read_bytes()).hexdigest()
                        if digest in content_hashes:
                            target_file.unlink(missing_ok=True)
                            continue
                        row = result_row(candidate, filename, category, item_type)
                        source_urls.add(candidate["source_url"])
                        content_hashes.add(digest)
                        break
            if row is None:
                create_fallback_image(category, filename, target_file)
                row = fixture_row(filename, category, item_type)
            rows.append(row)
            filenames.add(filename)
            write_manifest(manifest_path, rows)

    elif len(rows) < target and use_online:
        queries = [(slug, query) for slug, terms in BACKGROUND_SEARCHES.items() for query in terms]
        holdout_reserved = any("holdout" in row.get("notes", "").lower() for row in rows)
        added_holdout = holdout_reserved
        for page in range(5):
            before_pass = len(rows)
            for category, query in queries:
                if len(rows) >= target:
                    break
                time.sleep(0.5)
                for candidate in search_commons(query, limit=20, offset=page * 20):
                    if candidate["source_url"] in source_urls:
                        continue
                    filename = _next_background_filename(rows, category)
                    target_file = images_dir / filename
                    if not download_and_process_image(candidate["download_url"], target_file):
                        continue
                    digest = hashlib.sha256(target_file.read_bytes()).hexdigest()
                    if digest in content_hashes:
                        target_file.unlink(missing_ok=True)
                        continue
                    row = result_row(candidate, filename, category, "background")
                    if not added_holdout:
                        row["notes"] = f"{row['notes']} Reserved holdout image, excluded from deterministic demo seeds."
                        added_holdout = True
                    rows.append(row)
                    filenames.add(filename)
                    source_urls.add(candidate["source_url"])
                    content_hashes.add(digest)
                    write_manifest(manifest_path, rows)
                    print(f"Added {filename} ({len(rows)}/{target})", file=sys.stderr)
                    break
            if len(rows) == before_pass:
                break

    print(f"Recorded {len(rows)} images in {manifest_path}")
    return len(rows)


def fetch_specific(
    dest_dir: Path,
    manifest_path: Path,
    query: str,
    category: str,
    filename: str,
) -> bool:
    """Add one reviewed candidate from a focused query as a background photo."""
    rows = read_manifest(manifest_path)
    images_dir = dest_dir / "images"
    filenames, source_urls, content_hashes = _unique_state(rows, images_dir)
    if filename in filenames or (images_dir / filename).exists():
        raise ValueError(f"destination already exists: {filename}")
    for candidate in search_commons(query, limit=20):
        if not matches_category(candidate, category) or candidate["source_url"] in source_urls:
            continue
        target_file = images_dir / filename
        if not download_and_process_image(candidate["download_url"], target_file):
            continue
        digest = hashlib.sha256(target_file.read_bytes()).hexdigest()
        if digest in content_hashes:
            target_file.unlink(missing_ok=True)
            continue
        rows.append(result_row(candidate, filename, category, "background"))
        write_manifest(manifest_path, rows)
        print(f"Added {filename} from {candidate['source_url']}")
        return True
    print(f"No unique, licensed image matched {category!r} for {query!r}", file=sys.stderr)
    return False


def main() -> None:
    parser = argparse.ArgumentParser(description="Fetch demo photo pack for SimpanUGM")
    parser.add_argument("--offline", action="store_true", help="Use clearly labelled local fixtures for fixed rows")
    parser.add_argument("--dest", type=Path, default=Path("data/demo"), help="Destination directory")
    parser.add_argument("--target", type=int, help=f"Grow the manifest with background photos up to {MAX_TARGET} rows")
    parser.add_argument("--search", help="Inspect filtered Commons results without downloading")
    parser.add_argument("--fetch-query", help="Fetch one image from a focused Commons search")
    parser.add_argument("--category", help="Category slug for --fetch-query")
    parser.add_argument("--filename", help="Stable destination name for --fetch-query")
    args = parser.parse_args()

    if args.search:
        for result in search_commons(args.search, limit=20):
            print(f"{result['title']} | {result['license']} | {result['artist']} | {result['description']} | {result['source_url']}")
        return
    manifest_path = args.dest / "SOURCES.csv"
    try:
        if args.fetch_query:
            if not args.category or not args.filename:
                parser.error("--fetch-query requires --category and --filename")
            if not fetch_specific(args.dest, manifest_path, args.fetch_query, args.category, args.filename):
                raise SystemExit(1)
            return
        run_fetch(args.dest, manifest_path, use_online=not args.offline, target=args.target)
    except ValueError as exc:
        parser.error(str(exc))


if __name__ == "__main__":
    main()
