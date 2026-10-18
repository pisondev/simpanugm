import json
from pathlib import Path

import numpy as np
import pytest
from PIL import Image, ImageDraw

from app.analysis.brand import BrandMatcher, load_brands
from app.analysis.category import CategoryPrompts, ZeroShotClassifier
from app.analysis.colour import ColourExtractor, load_palette
from app.masking import mask_numbers
from tests.conftest import FIXTURES

SEED = FIXTURES.parent / "seed"


def test_masking_agrees_with_the_shared_fixture():
    cases = json.loads((FIXTURES / "masking_cases.json").read_text(encoding="utf-8"))["cases"]
    assert cases
    for case in cases:
        assert mask_numbers(case["input"]) == case["expected"], case["id"]


@pytest.fixture(scope="module")
def colours() -> ColourExtractor:
    return ColourExtractor(load_palette(SEED / "colours.csv"))


@pytest.mark.parametrize(("file", "name"), [("swatch-biru.png", "biru"), ("swatch-merah.png", "merah")])
def test_swatches_are_named_by_the_palette(colours, file, name):
    found = colours.extract(Image.open(FIXTURES / "images" / file))
    assert found[0].name == name and found[0].share > 0.9


def test_two_colours_are_returned_largest_first(colours):
    image = Image.new("RGB", (200, 200), (17, 24, 39))
    ImageDraw.Draw(image).rectangle((0, 0, 200, 70), fill=(220, 38, 38))
    found = colours.extract(image)
    assert [c.name for c in found] == ["hitam", "merah"]
    assert found[0].share > found[1].share


def test_small_clusters_are_dropped_and_names_merge(colours):
    # a speck under 10 percent never becomes a colour, two shades of one palette colour count once
    image = Image.new("RGB", (200, 200), (37, 99, 235))
    ImageDraw.Draw(image).rectangle((90, 90, 110, 110), fill=(220, 38, 38))
    assert [c.name for c in colours.extract(image)] == ["biru"]
    split = Image.new("RGB", (200, 200), (37, 99, 235))
    ImageDraw.Draw(split).rectangle((0, 0, 200, 100), fill=(30, 90, 225))
    only = colours.extract(split)
    assert [c.name for c in only] == ["biru"] and only[0].share > 0.95


def test_the_palette_has_the_fourteen_documented_colours():
    assert len(load_palette(SEED / "colours.csv")) == 14


@pytest.fixture(scope="module")
def brands() -> BrandMatcher:
    return BrandMatcher(load_brands(SEED / "brands.csv"))


def test_brands_match_ocr_tokens_with_a_ratio_of_ninety(brands):
    assert [b.name for b in brands.match("EIGER 1989 outdoor")] == ["Eiger"]
    assert [b.name for b in brands.match("Eigr")] == []
    # one character off a seven letter name is 93 percent, two off falls below the bar
    assert [(b.name, b.score) for b in brands.match("Samsun")] == [("Samsung", 0.9231)]
    assert brands.match("Samsu") == []
    assert [b.name for b in brands.match("Samsung Eiger")] == ["Eiger", "Samsung"]
    assert brands.match("ab") == []


def test_aliases_find_their_brand(brands):
    assert [b.name for b in brands.match("iphone 15")] == ["Apple"]


def test_every_match_is_at_least_ninety_percent(brands):
    for b in brands.match("Adidas Anker Apple Asus"):
        assert b.score >= 0.9


class OneHot:
    """Prompt embeddings on the axes, so the expected softmax can be worked out by hand."""

    def __init__(self) -> None:
        self.calls = 0

    def __call__(self, prompts: list[str]) -> np.ndarray:
        self.calls += 1
        out = np.zeros((len(prompts), 4), dtype=np.float32)
        for i, p in enumerate(prompts):
            out[i, int(p[-1])] = 1.0
        return out


CATS = [
    CategoryPrompts("a", ["x0", "x1"]),
    CategoryPrompts("b", ["x2"]),
    CategoryPrompts("c", ["x3"]),
    CategoryPrompts("empty", []),
]


def test_zero_shot_takes_the_best_prompt_of_each_category():
    clf = ZeroShotClassifier(OneHot())
    # closest to x1 (category a), then x2 (b), then x3 (c)
    vector = np.array([0.0, 0.9, 0.5, 0.1], dtype=np.float32)
    ranked = clf.classify(vector, "k", CATS)
    assert [r.slug for r in ranked] == ["a", "b", "c"]
    assert sum(r.score for r in ranked) == pytest.approx(1.0, abs=1e-3)
    # temperature 0.01 over cosines 0.9, 0.5, 0.1: the leader takes everything
    assert ranked[0].score > 0.999


def test_zero_shot_softens_when_cosines_are_close():
    clf = ZeroShotClassifier(OneHot())
    vector = np.array([0.0, 0.30, 0.295, 0.0], dtype=np.float32)
    first, second, _ = clf.classify(vector, "k", CATS)
    assert (first.slug, second.slug) == ("a", "b")
    assert first.score == pytest.approx(0.6225, abs=2e-3)


def test_zero_shot_embeds_each_prompt_set_once_and_evicts_old_ones():
    embed = OneHot()
    clf = ZeroShotClassifier(embed)
    v = np.ones(4, dtype=np.float32)
    clf.classify(v, "one", CATS)
    clf.classify(v, "one", CATS)
    assert embed.calls == 1
    for key in ("two", "three", "four", "five"):
        clf.classify(v, key, CATS)
    clf.classify(v, "one", CATS)
    assert embed.calls == 6


def test_zero_shot_without_prompts_returns_nothing():
    assert ZeroShotClassifier(OneHot()).classify(np.ones(4, dtype=np.float32), "k", [CategoryPrompts("a", [])]) == []


def test_fixture_directory_is_the_repository_one():
    assert Path(FIXTURES, "masking_cases.json").is_file()
