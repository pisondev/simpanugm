import threading
from dataclasses import dataclass
from pathlib import Path

from PIL import Image

from app.analysis.brand import BrandCandidate, BrandMatcher, load_brands
from app.analysis.category import CategoryPrompts, Scored, ZeroShotClassifier
from app.analysis.colour import ColourExtractor, Detected, load_palette
from app.backend import Backend, Vectors
from app.masking import mask_numbers
from app.metrics import measure_inference
from app.registry import Registry

TEXT_LIMIT = 300

# versions of the parts that are code rather than weights, bumped when their behaviour changes
COLOUR_EXTRACTOR_VERSION = "kmeans-lab4-ciede2000-v1"
BRAND_MATCHER_VERSION = "rapidfuzz-ratio90-v1"
CLASSIFIER_VERSION = "zero-shot-t0.01-v1"


@dataclass(frozen=True)
class Analysis:
    categories: list[Scored]
    colours: list[Detected]
    text: str
    masked: bool
    brands: list[BrandCandidate]
    versions: dict[str, str]


class Inference:
    """The operations behind the /v1 endpoints. A semaphore keeps the CPU from being oversubscribed (section 9)."""

    def __init__(self, backend: Backend, registry: Registry, seed_dir: Path, concurrency: int) -> None:
        self.backend = backend
        self.registry = registry
        self.classifier = ZeroShotClassifier(backend.embed_prompts)
        self.colours = ColourExtractor(load_palette(seed_dir / "colours.csv"))
        self.brands = BrandMatcher(load_brands(seed_dir / "brands.csv"))
        self._slots = threading.BoundedSemaphore(concurrency)

    def embed_images(self, images: list[Image.Image]) -> Vectors:
        with self._slots, measure_inference("image_encoder"):
            return self.backend.embed_images(images)

    def embed_texts(self, texts: list[str], kind: str) -> Vectors:
        component = "text_encoder" if kind == "e5" else "xmodal_text_encoder"
        with self._slots, measure_inference(component):
            return self.backend.embed_e5(texts) if kind == "e5" else self.backend.embed_clip_text(texts)

    def analyze(self, image: Image.Image, categories: list[CategoryPrompts], prompt_set_hash: str) -> Analysis:
        with self._slots:
            with measure_inference("image_encoder"):
                vector = self.backend.embed_images([image])[0]
            with measure_inference("category_classifier"):
                scored = self.classifier.classify(vector, prompt_set_hash, categories)
            with measure_inference("ocr"):
                lines = self.backend.read_text(image)
        raw = " ".join(line.text.strip() for line in lines if line.text.strip())
        masked_text = mask_numbers(raw)
        value = masked_text[:TEXT_LIMIT]
        with measure_inference("colour_extractor"):
            colours = self.colours.extract(image)
        with measure_inference("brand_matcher"):
            brands = self.brands.match(raw)
        versions = {name: f"{c.model_id}@{c.revision}" for name, c in self.registry.components.items()}
        versions.update(
            category_classifier=CLASSIFIER_VERSION,
            colour_extractor=COLOUR_EXTRACTOR_VERSION,
            brand_matcher=BRAND_MATCHER_VERSION,
        )
        return Analysis(scored, colours, value, masked_text != raw, brands, versions)
