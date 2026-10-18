"""The matching texts of matching-engine section 3, the Python twin of api/internal/matching/text.go."""

from __future__ import annotations

import csv
import re
import unicodedata
from dataclasses import dataclass, field
from pathlib import Path

from app.masking import mask_numbers

CLIP_MAX = 400
_URL = re.compile(r"(?i)\b(?:https?://|www\.)\S+")
_WORD = re.compile(r"[^\W_]+", re.UNICODE)


@dataclass
class TextInput:
    item_name: str
    category: str
    parent: str
    colours: list[str] = field(default_factory=list)
    brand: str = ""
    description: str = ""
    private: str = ""
    visible_text: str = ""


def load_synonyms(path: Path) -> dict[str, str]:
    with path.open(newline="", encoding="utf-8") as handle:
        return {row["term"].lower(): row["expansion"].lower() for row in csv.DictReader(handle)}


def normalise(s: str) -> str:
    s = unicodedata.normalize("NFKC", s)
    s = _URL.sub(" ", s)
    s = mask_numbers(s)
    return " ".join(s.split())


def expand(text: str, synonyms: dict[str, str]) -> str:
    """Appends the canonical words of slang found in the text, the user's own words stay."""
    if not synonyms:
        return text
    words = _WORD.findall(text.lower())
    seen = set(words)
    extra: list[str] = []
    for w in words:
        for e in synonyms.get(w, "").split():
            if e not in seen:
                seen.add(e)
                extra.append(e)
    return text if not extra else f"{text} {' '.join(extra)}"


def _none(s: str) -> str:
    return s if s.strip() else "tidak ada"


def e5_text(t: TextInput, synonyms: dict[str, str] | None = None) -> str:
    parts = [
        f"{t.item_name}.",
        f"{t.category} ({t.parent}).",
        f"Warna: {_none(', '.join(t.colours))}.",
        f"Merek: {_none(t.brand)}.",
        t.description,
    ]
    parts += [p for p in (t.private, t.visible_text) if p]
    return "query: " + expand(normalise(" ".join(parts)), synonyms or {})


def clip_text(t: TextInput, synonyms: dict[str, str] | None = None) -> str:
    parts = [f"{t.item_name}.", f"{t.category}."]
    if t.colours:
        parts.append(", ".join(t.colours) + ".")
    if t.brand:
        parts.append(t.brand + ".")
    parts.append(t.description)
    return expand(normalise(" ".join(parts)), synonyms or {})[:CLIP_MAX]
