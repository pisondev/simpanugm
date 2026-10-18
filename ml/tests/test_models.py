"""Golden checks against the real pinned weights. They need the models in HF_HOME, so they only run on request:
`HF_HOME=... uv run pytest -m models`. The CI job runs without weights and skips them."""

import os
from pathlib import Path

import pytest
from PIL import Image

from app.analysis.colour import ColourExtractor, load_palette
from app.backend import TorchBackend
from app.masking import mask_numbers
from app.registry import Registry
from tests.conftest import FIXTURES

pytestmark = pytest.mark.models

LOCK = Path(__file__).resolve().parent.parent / "models.lock.json"


@pytest.fixture(scope="module")
def backend() -> TorchBackend:
    home = os.environ.get("HF_HOME")
    if not home:
        pytest.skip("HF_HOME is not set")
    b = TorchBackend(Registry.from_lock(LOCK), Path(home), threads=4)
    b.load()
    return b


def image(name: str) -> Image.Image:
    return Image.open(FIXTURES / "images" / name).convert("RGB")


def test_vectors_have_the_documented_shape_and_unit_length(backend):
    for vectors, dim in (
        (backend.embed_images([image("desk-gps.jpg")]), 512),
        (backend.embed_clip_text(["tumbler biru"]), 512),
        (backend.embed_e5(["query: tumbler biru"]), 384),
    ):
        assert vectors.shape == (1, dim)
        assert abs(float((vectors[0] ** 2).sum()) - 1) < 1e-4


def test_zero_shot_prompts_live_in_the_image_space(backend):
    blue, red = backend.embed_images([image("swatch-biru.png"), image("swatch-merah.png")])
    prompts = backend.embed_prompts(["a photo of a blue square", "a photo of a red square"])
    assert prompts[0] @ blue > prompts[1] @ blue
    assert prompts[1] @ red > prompts[0] @ red


def test_indonesian_text_aligns_with_the_image_space(backend):
    blue, red = backend.embed_images([image("swatch-biru.png"), image("swatch-merah.png")])
    text = backend.embed_clip_text(["sebuah kotak berwarna biru", "sebuah kotak berwarna merah"])
    assert text[0] @ blue > text[1] @ blue
    assert text[1] @ red > text[0] @ red


def test_a_paraphrase_is_closer_than_an_unrelated_text(backend):
    q, para, other = backend.embed_e5(
        [
            "query: dompet kulit warna cokelat",
            "query: dompet coklat dari bahan kulit",
            "query: payung lipat hitam gagang kayu",
        ]
    )
    assert q @ para > q @ other + 0.05


def test_ocr_reads_the_mock_card_and_the_number_is_masked(backend):
    lines = backend.read_text(image("mock-card.png"))
    text = " ".join(line.text for line in lines)
    assert "KARTU SINTETIS" in text.upper()
    masked = mask_numbers(text)
    assert "0000123456789012" not in masked and masked.count("*") >= 12


def test_the_blue_swatch_is_named_biru():
    extractor = ColourExtractor(load_palette(FIXTURES.parent / "seed" / "colours.csv"))
    assert extractor.extract(image("swatch-biru.png"))[0].name == "biru"
