#!/usr/bin/env python3
"""Extract named OpenStreetMap elements into a human-review CSV."""

from __future__ import annotations

import argparse
import csv
import json
import math
import re
import sys
import tempfile
import time
import unicodedata
from pathlib import Path
from typing import Any, Iterable
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


REVIEW_FIELDS = (
    "slug",
    "name",
    "zone_slug",
    "kind",
    "lat",
    "lng",
    "aliases",
    "osm_type",
    "osm_id",
    "review_status",
)
DEFAULT_ENDPOINT = "https://overpass-api.de/api/interpreter"


def slugify(value: str) -> str:
    """Create a stable ASCII slug from an OSM display name."""
    normalized = unicodedata.normalize("NFKD", value)
    ascii_value = normalized.encode("ascii", "ignore").decode("ascii").lower()
    slug = re.sub(r"[^a-z0-9]+", "-", ascii_value).strip("-")
    return slug or "location"


def coordinates(element: dict[str, Any]) -> tuple[float, float] | None:
    if isinstance(element.get("lat"), (int, float)) and isinstance(element.get("lon"), (int, float)):
        return float(element["lat"]), float(element["lon"])
    center = element.get("center")
    if isinstance(center, dict) and isinstance(center.get("lat"), (int, float)) and isinstance(center.get("lon"), (int, float)):
        return float(center["lat"]), float(center["lon"])
    return None


def classify_kind(tags: dict[str, str]) -> str:
    if tags.get("amenity") == "parking" or tags.get("parking"):
        return "parking"
    if tags.get("public_transport") or tags.get("highway") in {"bus_stop", "platform"}:
        return "transport"
    if tags.get("leisure") in {"park", "pitch", "playground", "sports_centre"}:
        return "outdoor"
    if tags.get("amenity") in {"library", "school", "university", "college", "clinic", "hospital"}:
        return "facility"
    return "building"


def review_rows(
    document: dict[str, Any],
    bbox: tuple[float, float, float, float] | None = None,
) -> list[dict[str, str]]:
    elements = document.get("elements")
    if not isinstance(elements, list):
        raise ValueError("OSM response must contain an 'elements' list")

    rows: list[dict[str, str]] = []
    used_slugs: set[str] = set()
    seen_osm_ids: set[tuple[str, str]] = set()

    def sort_key(element: dict[str, Any]) -> tuple[str, str, str]:
        tags = element.get("tags")
        name = tags.get("name", "") if isinstance(tags, dict) else ""
        return (str(name).casefold(), str(element.get("type", "")), str(element.get("id", "")))

    candidates = sorted((element for element in elements if isinstance(element, dict)), key=sort_key)
    for element in candidates:
        if not isinstance(element, dict):
            continue
        tags = element.get("tags")
        if not isinstance(tags, dict) or not isinstance(tags.get("name"), str) or not tags["name"].strip():
            continue
        osm_type = element.get("type")
        osm_id = element.get("id")
        if not isinstance(osm_type, str) or not isinstance(osm_id, (int, str)):
            continue
        osm_key = (osm_type, str(osm_id))
        if osm_key in seen_osm_ids:
            continue
        point = coordinates(element)
        if point is None:
            continue
        if bbox is not None:
            south, west, north, east = bbox
            if not (south <= point[0] <= north and west <= point[1] <= east):
                continue
        seen_osm_ids.add(osm_key)
        name = tags["name"].strip()
        base_slug = slugify(name)
        slug = base_slug
        suffix = 2
        while slug in used_slugs:
            slug = f"{base_slug}-{suffix}"
            suffix += 1
        used_slugs.add(slug)
        rows.append(
            {
                "slug": slug,
                "name": name,
                "zone_slug": "",
                "kind": classify_kind({str(key): str(value) for key, value in tags.items()}),
                "lat": f"{point[0]:.7f}",
                "lng": f"{point[1]:.7f}",
                "aliases": "",
                "osm_type": osm_type,
                "osm_id": str(osm_id),
                "review_status": "needs_review",
            }
        )
    return sorted(rows, key=lambda row: (row["name"].casefold(), row["osm_type"], row["osm_id"]))


def overpass_query(bbox: tuple[float, float, float, float]) -> str:
    south, west, north, east = bbox
    box = f"{south},{west},{north},{east}"
    return (
        f"[out:json][timeout:25];("
        f"nwr[\"name\"][\"building\"]({box});"
        f"nwr[\"name\"][\"amenity\"]({box});"
        f"nwr[\"name\"][\"leisure\"]({box});"
        f"nwr[\"name\"][\"public_transport\"]({box});"
        f"nwr[\"name\"][\"highway\"~\"bus_stop|platform\"]({box});"
        f");out center tags;"
    )


def parse_bbox(value: str) -> tuple[float, float, float, float]:
    try:
        values = tuple(float(part.strip()) for part in value.split(","))
    except ValueError as exc:
        raise argparse.ArgumentTypeError("bbox must be south,west,north,east") from exc
    if (
        len(values) != 4
        or any(not math.isfinite(number) for number in values)
        or not values[0] < values[2]
        or not values[1] < values[3]
    ):
        raise argparse.ArgumentTypeError("bbox must be south,west,north,east with min values first")
    return values  # type: ignore[return-value]


RETRYABLE_HTTP_CODES = {408, 429, 500, 502, 503, 504}


def fetch_overpass(
    endpoint: str,
    bbox: tuple[float, float, float, float],
    retries: int = 2,
    request_timeout: float = 45,
) -> dict[str, Any]:
    payload = urlencode({"data": overpass_query(bbox)}).encode("utf-8")
    request = Request(endpoint, data=payload, headers={"User-Agent": "simpanugm-data-review/1.0"})
    for attempt in range(retries + 1):
        try:
            with urlopen(request, timeout=request_timeout) as response:  # noqa: S310 - endpoint is an explicit CLI argument
                return json.load(response)
        except HTTPError as exc:
            should_retry = exc.code in RETRYABLE_HTTP_CODES and attempt < retries
            exc.close()
            if not should_retry:
                raise
        except (TimeoutError, URLError):
            if attempt == retries:
                raise
        time.sleep(2**attempt)
    raise RuntimeError("unreachable Overpass retry state")


def write_raw_response(path: Path, document: dict[str, Any]) -> None:
    """Persist the source response atomically for a later human review."""
    if path.is_symlink():
        raise OSError(f"output must not be a symlink: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            encoding="utf-8",
            prefix=f".{path.name}.",
            suffix=".tmp",
            dir=path.parent,
            delete=False,
        ) as handle:
            temporary_path = Path(handle.name)
            json.dump(document, handle, ensure_ascii=False, indent=2)
            handle.write("\n")
        temporary_path.replace(path)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()


def write_rows(path: Path, rows: Iterable[dict[str, str]]) -> None:
    if path.is_symlink():
        raise OSError(f"output must not be a symlink: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary_path: Path | None = None
    try:
        with tempfile.NamedTemporaryFile(
            mode="w",
            newline="",
            encoding="utf-8",
            prefix=f".{path.name}.",
            suffix=".tmp",
            dir=path.parent,
            delete=False,
        ) as handle:
            temporary_path = Path(handle.name)
            writer = csv.DictWriter(handle, fieldnames=REVIEW_FIELDS)
            writer.writeheader()
            writer.writerows(rows)
        temporary_path.replace(path)
    finally:
        if temporary_path is not None and temporary_path.exists():
            temporary_path.unlink()


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--input", type=Path, help="saved Overpass JSON response")
    source.add_argument("--download", action="store_true", help="download a fresh Overpass response")
    parser.add_argument("--output", type=Path, required=True, help="review CSV output path")
    parser.add_argument("--bbox", type=parse_bbox, help="south,west,north,east; required with --download")
    parser.add_argument(
        "--raw-output",
        type=Path,
        help="save the downloaded Overpass JSON response for review",
    )
    parser.add_argument(
        "--retries",
        type=int,
        default=2,
        help="retries for transient Overpass errors (default: 2)",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=45,
        help="HTTP timeout in seconds for each Overpass request (default: 45)",
    )
    parser.add_argument("--endpoint", default=DEFAULT_ENDPOINT, help=f"Overpass endpoint (default: {DEFAULT_ENDPOINT})")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        if args.download:
            if args.bbox is None:
                raise ValueError("--bbox is required with --download")
            if args.retries < 0 or args.timeout <= 0:
                raise ValueError("--retries must be non-negative and --timeout must be positive")
            document = fetch_overpass(args.endpoint, args.bbox, args.retries, args.timeout)
            if args.raw_output is not None:
                write_raw_response(args.raw_output, document)
        else:
            if args.raw_output is not None:
                raise ValueError("--raw-output requires --download")
            with args.input.open(encoding="utf-8") as handle:
                document = json.load(handle)
        rows = review_rows(document, args.bbox)
        write_rows(args.output, rows)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    print(f"Wrote {len(rows)} review candidates to {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
