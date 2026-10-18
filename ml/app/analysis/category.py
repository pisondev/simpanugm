import threading
from collections import OrderedDict
from collections.abc import Callable
from dataclasses import dataclass

import numpy as np
from numpy.typing import NDArray

TEMPERATURE = 0.01
TOP = 3
CACHE_SIZE = 4

Entry = tuple[list[str], list[slice], NDArray[np.float32]]


@dataclass(frozen=True)
class CategoryPrompts:
    slug: str
    prompts: list[str]


@dataclass(frozen=True)
class Scored:
    slug: str
    score: float


class ZeroShotClassifier:
    """Softmax with temperature 0.01 over the best cosine of each category's prompts (matching-engine section 4).

    Prompt embeddings are kept by the hash of the prompt set the api sends, so a master data change takes effect
    without a restart and an unchanged set is never embedded twice.
    """

    def __init__(self, embed_prompts: Callable[[list[str]], NDArray[np.float32]]) -> None:
        self._embed = embed_prompts
        self._cache: OrderedDict[str, Entry] = OrderedDict()
        self._lock = threading.Lock()

    def _entry(self, key: str, categories: list[CategoryPrompts]) -> Entry:
        with self._lock:
            if key in self._cache:
                self._cache.move_to_end(key)
                return self._cache[key]
        slugs: list[str] = []
        spans: list[slice] = []
        flat: list[str] = []
        for c in categories:
            spans.append(slice(len(flat), len(flat) + len(c.prompts)))
            slugs.append(c.slug)
            flat.extend(c.prompts)
        entry: Entry = (slugs, spans, self._embed(flat))
        with self._lock:
            self._cache[key] = entry
            while len(self._cache) > CACHE_SIZE:
                self._cache.popitem(last=False)
        return entry

    def classify(self, image_vector: NDArray[np.float32], key: str, categories: list[CategoryPrompts]) -> list[Scored]:
        usable = [c for c in categories if c.prompts]
        if not usable:
            return []
        slugs, spans, matrix = self._entry(key, usable)
        cosines = matrix @ image_vector
        best = np.array([cosines[s].max() for s in spans], dtype=np.float64)
        logits = best / TEMPERATURE
        logits -= logits.max()
        probs = np.exp(logits)
        probs /= probs.sum()
        order = np.argsort(-probs, kind="stable")[:TOP]
        return [Scored(slugs[i], round(float(probs[i]), 4)) for i in order]
