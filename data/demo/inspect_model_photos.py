#!/usr/bin/env python3
"""Measure real ML category rankings for every image in the demo manifest."""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import os
import secrets
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any

import yaml


ROOT = Path(__file__).resolve().parents[2]
MANIFEST = ROOT / "data" / "demo" / "SOURCES.csv"
CATEGORIES = ROOT / "data" / "seed" / "categories.yaml"
OUTPUT = ROOT / "docs" / "ai" / "demo-photo-check.md"


def category_prompts(path: Path = CATEGORIES) -> list[dict[str, Any]]:
    document = yaml.safe_load(path.read_text(encoding="utf-8"))
    return sorted([
        {"slug": child["slug"], "prompts": child["clip_prompts"]}
        for parent in document["parents"]
        for child in parent.get("children", [])
        if child.get("clip_prompts")
    ], key=lambda category: category["slug"])


def multipart(fields: dict[str, str], filename: str, content: bytes) -> tuple[str, bytes]:
    boundary = "----SimpanUGM" + secrets.token_hex(16)
    chunks: list[bytes] = []
    for name, value in fields.items():
        chunks.extend(
            [
                f"--{boundary}\r\n".encode(),
                f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode(),
                value.encode(),
                b"\r\n",
            ]
        )
    chunks.extend(
        [
            f"--{boundary}\r\n".encode(),
            f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'.encode(),
            b"Content-Type: image/jpeg\r\n\r\n",
            content,
            b"\r\n",
            f"--{boundary}--\r\n".encode(),
        ]
    )
    return boundary, b"".join(chunks)


def analyze(path: Path, categories: list[dict[str, Any]], endpoint: str, token: str) -> list[dict[str, Any]]:
    category_json = json.dumps(categories, separators=(",", ":"))
    prompt_hash = hashlib.sha256(category_json.encode("utf-8")).hexdigest()
    boundary, body = multipart(
        {"categories": category_json, "prompt_set_hash": prompt_hash}, path.name, path.read_bytes()
    )
    request = urllib.request.Request(
        endpoint.rstrip("/") + "/v1/analyze/image",
        data=body,
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": f"multipart/form-data; boundary={boundary}",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=120) as response:
            result = json.loads(response.read().decode("utf-8"))
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as exc:
        raise RuntimeError(f"analysis failed for {path.name}: {exc}") from exc
    return result["categories"]


def load_rows(path: Path = MANIFEST) -> list[dict[str, str]]:
    with path.open(encoding="utf-8", newline="") as file:
        return list(csv.DictReader(file))


def markdown(rows: list[dict[str, str]], rankings: dict[str, list[dict[str, Any]]], label: str) -> str:
    lines = [
        f"## {label}",
        "",
        "Scores are the live ML endpoint response for the category prompts listed in `data/seed/categories.yaml`.",
        "",
        "| Image | Type | Expected | Top 1 | Top 1 score | Top 2 | Top 2 score | Top 3 | Top 3 score |",
        "|---|---|---|---|---:|---|---:|---|---:|",
    ]
    for row in rows:
        result = rankings[row["filename"]]
        top = result[:3] + [{"slug": "", "score": 0.0}] * max(0, 3 - len(result))
        values = [
            row["filename"],
            row["item_type"],
            row["category"],
            top[0]["slug"],
            f"{top[0]['score']:.4f}",
            top[1]["slug"],
            f"{top[1]['score']:.4f}",
            top[2]["slug"],
            f"{top[2]['score']:.4f}",
        ]
        lines.append("| " + " | ".join(str(value).replace("|", "/") for value in values) + " |")
    lines.append("")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--endpoint", default=os.environ.get("SIMPANUGM_ML_ENDPOINT", "http://localhost:8100"))
    parser.add_argument("--token", default=os.environ.get("SIMPANUGM_ML_TOKEN"))
    parser.add_argument("--label", default="Baseline")
    parser.add_argument("--output", type=Path, default=OUTPUT)
    parser.add_argument("--filename", action="append", help="Analyze one manifest image; repeat for multiple files")
    args = parser.parse_args()
    if not args.token:
        parser.error("set SIMPANUGM_ML_TOKEN or pass --token")

    rows = load_rows()
    if args.filename:
        requested = set(args.filename)
        rows = [row for row in rows if row["filename"] in requested]
        missing = requested - {row["filename"] for row in rows}
        if missing:
            parser.error(f"files are not in the manifest: {', '.join(sorted(missing))}")
    prompts = category_prompts()
    rankings: dict[str, list[dict[str, Any]]] = {}
    for index, row in enumerate(rows, start=1):
        image = ROOT / "data" / "demo" / "images" / row["filename"]
        if not image.is_file():
            raise FileNotFoundError(image)
        rankings[row["filename"]] = analyze(image, prompts, args.endpoint, args.token)
        print(f"Analyzed {index}/{len(rows)}: {row['filename']}", file=sys.stderr)

    section = markdown(rows, rankings, args.label)
    if args.output.exists():
        existing = args.output.read_text(encoding="utf-8").rstrip()
        args.output.write_text(existing + "\n\n" + section, encoding="utf-8")
    else:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text("# Demo photo model checks\n\n" + section, encoding="utf-8")
    print(f"Wrote {len(rows)} live results to {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
