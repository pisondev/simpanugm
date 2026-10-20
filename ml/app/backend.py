"""The models behind the endpoints. Everything that needs PyTorch lives here, so the rest of the service can be tested
with a stand in that implements the same protocol."""

import hashlib
import logging
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Protocol

import numpy as np
from numpy.typing import NDArray
from PIL import Image

from app.registry import IMAGE_ENCODER, OCR, TEXT_ENCODER, XMODAL_TEXT_ENCODER, Registry

log = logging.getLogger("simpanugm.ml")

Vectors = NDArray[np.float32]

# EasyOCR's CPU detector dominates photo-assist latency. 768px retains legible
# item labels while cutting detector input area by 44% from the old 1024px cap.
OCR_SIDE = 768
OCR_MIN_CONFIDENCE = 0.3


@dataclass(frozen=True)
class OcrLine:
    text: str
    confidence: float


class Backend(Protocol):
    def load(self) -> None: ...

    def embed_images(self, images: list[Image.Image]) -> Vectors: ...

    def embed_clip_text(self, texts: list[str]) -> Vectors: ...

    def embed_e5(self, texts: list[str]) -> Vectors: ...

    def embed_prompts(self, prompts: list[str]) -> Vectors:
        """English prompts embedded by the image encoder's own text tower, the space the zero shot score lives in."""
        ...

    def read_text(self, image: Image.Image) -> list[OcrLine]: ...


def _unit(v: NDArray[np.floating]) -> Vectors:
    arr = np.asarray(v, dtype=np.float32)
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    return arr / np.maximum(norms, 1e-12)


def _loaded(model: object) -> None:
    if model is None:
        raise RuntimeError("the model is not loaded, embed only after load()")


def sha256_of(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1 << 20), b""):
            digest.update(block)
    return digest.hexdigest()


class TorchBackend:
    """sentence-transformers encoders and EasyOCR, pinned by models.lock.json."""

    def __init__(self, registry: Registry, cache_dir: Path, threads: int) -> None:
        self.registry = registry
        self.cache_dir = cache_dir
        self.threads = threads
        self._image_model: Any = None
        self._xmodal_model: Any = None
        self._e5_model: Any = None
        self._reader: Any = None

    def load(self) -> None:
        import torch
        from sentence_transformers import SentenceTransformer

        torch.set_num_threads(self.threads)
        cache = str(self.cache_dir)
        for name in self.registry.components:
            spec = self.registry.get(name)
            if name == OCR:
                self._load_ocr(spec.sha256)
            else:
                model = SentenceTransformer(spec.model_id, revision=spec.revision, cache_folder=cache, device="cpu")
                if name == IMAGE_ENCODER:
                    self._image_model = model
                elif name == XMODAL_TEXT_ENCODER:
                    self._xmodal_model = model
                elif name == TEXT_ENCODER:
                    self._e5_model = model
            self.registry.mark_loaded(name)
            log.info("component loaded", extra={"component": name, "revision": spec.revision})

    def _load_ocr(self, expected: dict[str, str]) -> None:
        import easyocr

        directory = self.cache_dir / "easyocr"
        directory.mkdir(parents=True, exist_ok=True)
        self._reader = easyocr.Reader(  # type: ignore[attr-defined]
            ["id", "en"],
            gpu=False,
            model_storage_directory=str(directory),
            user_network_directory=str(directory),
            verbose=False,
        )
        # the weights come from release assets, so the lock pins their hashes instead of a repository revision
        for file, want in expected.items():
            got = sha256_of(directory / file)
            if got != want:
                raise RuntimeError(f"ocr weights {file} do not match models.lock.json: {got}")

    def embed_images(self, images: list[Image.Image]) -> Vectors:
        _loaded(self._image_model)
        return _unit(self._image_model.encode(images, convert_to_numpy=True, batch_size=8, show_progress_bar=False))

    def embed_prompts(self, prompts: list[str]) -> Vectors:
        _loaded(self._image_model)
        return _unit(self._image_model.encode(prompts, convert_to_numpy=True, batch_size=32, show_progress_bar=False))

    def embed_clip_text(self, texts: list[str]) -> Vectors:
        _loaded(self._xmodal_model)
        return _unit(self._xmodal_model.encode(texts, convert_to_numpy=True, batch_size=32, show_progress_bar=False))

    def embed_e5(self, texts: list[str]) -> Vectors:
        _loaded(self._e5_model)
        return _unit(self._e5_model.encode(texts, convert_to_numpy=True, batch_size=32, show_progress_bar=False))

    def read_text(self, image: Image.Image) -> list[OcrLine]:
        _loaded(self._reader)
        from app.imaging import downscale

        found = self._reader.readtext(np.asarray(downscale(image, OCR_SIDE)))
        return [OcrLine(str(text), float(conf)) for _, text, conf in found if float(conf) >= OCR_MIN_CONFIDENCE]
