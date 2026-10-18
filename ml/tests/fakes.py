import hashlib

import numpy as np
from PIL import Image

from app.backend import OcrLine, Vectors
from app.registry import Registry


def _vector(key: bytes, dim: int) -> np.ndarray:
    seed = int.from_bytes(hashlib.sha256(key).digest()[:8], "little")
    v = np.random.default_rng(seed).standard_normal(dim).astype(np.float32)
    return v / np.linalg.norm(v)


class FakeBackend:
    """Deterministic stand in for the models: equal inputs embed equally, nothing needs PyTorch."""

    def __init__(self, registry: Registry, lines: list[OcrLine] | None = None) -> None:
        self.registry = registry
        self.lines = lines or []
        self.prompt_calls = 0
        self.fail_load = False

    def load(self) -> None:
        if self.fail_load:
            raise RuntimeError("weights missing")
        self.registry.load_all()

    def embed_images(self, images: list[Image.Image]) -> Vectors:
        return np.stack([_vector(img.tobytes()[:4096] + str(img.size).encode(), 512) for img in images])

    def embed_clip_text(self, texts: list[str]) -> Vectors:
        return np.stack([_vector(t.encode(), 512) for t in texts])

    def embed_e5(self, texts: list[str]) -> Vectors:
        return np.stack([_vector(t.encode(), 384) for t in texts])

    def embed_prompts(self, prompts: list[str]) -> Vectors:
        self.prompt_calls += 1
        return np.stack([_vector(p.encode(), 512) for p in prompts])

    def read_text(self, image: Image.Image) -> list[OcrLine]:
        return self.lines
