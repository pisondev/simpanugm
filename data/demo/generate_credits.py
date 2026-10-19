#!/usr/bin/env python3
"""Generate docs/project/demo-credits.md from data/demo/SOURCES.csv.

Follows strict project rules:
- Zero em dashes, zero en dashes, zero emoji
- English documentation
- Clear attribution of all demo images, licenses, and sources
"""

from __future__ import annotations

import csv
import sys
from pathlib import Path


def generate_credits(csv_path: Path, output_md_path: Path) -> int:
    if not csv_path.exists():
        print(f"Manifest not found: {csv_path}", file=sys.stderr)
        return 0

    rows: list[dict[str, str]] = []
    with open(csv_path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for r in reader:
            rows.append(r)

    rows.sort(key=lambda x: (x.get("item_type", ""), x.get("filename", "")))

    lines: list[str] = [
        "# Demo photo pack credits and licenses | SimpanUGM",
        "",
        "This document lists attribution, sources, and open licenses for all photographs and media fixtures used in the SimpanUGM live demonstration system.",
        "",
        "## Compliance and licensing rules",
        "",
        "- Sources: Wikimedia Commons official API, Open Images CC subset, or team fixtures.",
        "- Licenses: Only CC0 1.0, CC-BY, CC-BY-SA, or Public Domain.",
        "- Privacy: No photographs contain identifiable faces, persons, or real identity documents.",
        "- Storage: All images downscaled to max 1024px longest edge to keep repo footprint minimal.",
        "",
        f"Total images registered: {len(rows)}",
        "",
        "## Image inventory",
        "",
        "| Filename | Category | Type | Author | License | Source |",
        "|---|---|---|---|---|---|",
    ]

    for r in rows:
        fn = r.get("filename", "").strip()
        cat = r.get("category", "").strip()
        itype = r.get("item_type", "").strip()
        author = r.get("author", "").strip().replace("|", "/")
        lic = r.get("license", "").strip().replace("|", "/")
        url = r.get("source_url", "").strip()
        if url.startswith("http"):
            src_col = f"[{r.get('source', 'link')}]({url})"
        else:
            src_col = r.get("source", "local")
        lines.append(f"| `{fn}` | {cat} | {itype} | {author} | {lic} | {src_col} |")

    lines.append("")
    content = "\n".join(lines)

    output_md_path.parent.mkdir(parents=True, exist_ok=True)
    with open(output_md_path, "w", encoding="utf-8") as f:
        f.write(content)

    print(f"Generated credits documentation with {len(rows)} items at {output_md_path}")
    return len(rows)


def main() -> None:
    csv_file = Path("data/demo/SOURCES.csv")
    out_file = Path("docs/project/demo-credits.md")
    generate_credits(csv_file, out_file)


if __name__ == "__main__":
    main()
