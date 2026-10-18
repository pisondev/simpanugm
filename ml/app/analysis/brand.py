import csv
import itertools
import re
from dataclasses import dataclass
from pathlib import Path

from rapidfuzz import fuzz

MIN_TOKEN = 3
MIN_RATIO = 90

_TOKEN = re.compile(r"[A-Za-z0-9]+")


@dataclass(frozen=True)
class BrandCandidate:
    name: str
    score: float


def load_brands(path: Path) -> dict[str, list[str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        return {row["name"]: [a for a in row["aliases"].split(";") if a] for row in csv.DictReader(handle)}


class BrandMatcher:
    """matching-engine section 4: OCR tokens against the brand list, a ratio of 90 or more is a candidate."""

    def __init__(self, brands: dict[str, list[str]]) -> None:
        self.terms: list[tuple[str, str]] = []
        for name, aliases in brands.items():
            for term in sorted({name.lower(), *(a.lower() for a in aliases)}):
                self.terms.append((name, term))

    def match(self, text: str) -> list[BrandCandidate]:
        tokens = [t.lower() for t in _TOKEN.findall(text) if len(t) >= MIN_TOKEN]
        # a brand of two words appears as two neighbouring tokens
        phrases = tokens + [f"{a} {b}" for a, b in itertools.pairwise(tokens)]
        best: dict[str, float] = {}
        for phrase in phrases:
            for brand, term in self.terms:
                ratio = fuzz.ratio(phrase, term)
                if ratio >= MIN_RATIO and ratio / 100 > best.get(brand, 0.0):
                    best[brand] = ratio / 100
        return [BrandCandidate(n, round(s, 4)) for n, s in sorted(best.items(), key=lambda kv: (-kv[1], kv[0]))]
