"""T9 and the Sprint 0 CPU spike: per component latency on this machine, `uv run python -m eval.latency`.

It uses the committed fixture images so anyone can repeat it. The numbers go next to the budget of matching-engine
section 10 in the evaluation report, together with the machine they were taken on.
"""

from __future__ import annotations

import argparse
import json
import os
import platform
import statistics
import time
from collections.abc import Callable
from pathlib import Path

from PIL import Image

from app.analysis.brand import BrandMatcher, load_brands
from app.analysis.colour import ColourExtractor, load_palette
from app.backend import Backend, TorchBackend
from app.registry import Registry
from eval.run import ROOT, SEED_DIR

FIXTURES = ROOT / "data" / "fixtures"
TEXT = "query: tumbler biru dongker 500 ml, tutup hitam, stiker kucing di sisi. Warna: biru. Merek: tidak ada."


def measure(fn: Callable[[], object], runs: int) -> dict[str, float]:
    fn()  # the first call pays for lazy initialisation
    times = []
    for _ in range(runs):
        start = time.perf_counter()
        fn()
        times.append((time.perf_counter() - start) * 1000)
    times.sort()
    return {
        "p50_ms": round(statistics.median(times), 1),
        "p95_ms": round(times[max(0, int(0.95 * len(times)) - 1)], 1),
        "runs": float(runs),
    }


def run(backend: Backend, runs: int) -> dict[str, dict[str, float]]:
    desk = Image.open(FIXTURES / "images" / "desk-gps.jpg").convert("RGB")
    card = Image.open(FIXTURES / "images" / "mock-card.png").convert("RGB")
    colours = ColourExtractor(load_palette(SEED_DIR / "colours.csv"))
    brands = BrandMatcher(load_brands(SEED_DIR / "brands.csv"))
    return {
        "image_encoder (1 image)": measure(lambda: backend.embed_images([desk]), runs),
        "image_encoder (2 images)": measure(lambda: backend.embed_images([desk, card]), runs),
        "xmodal_text_encoder (1 text)": measure(lambda: backend.embed_clip_text([TEXT]), runs),
        "text_encoder e5 (1 text)": measure(lambda: backend.embed_e5([TEXT]), runs),
        "ocr (mock card)": measure(lambda: backend.read_text(card), max(3, runs // 3)),
        "colour_extractor": measure(lambda: colours.extract(desk), runs),
        "brand_matcher": measure(lambda: brands.match("EIGER 1989 Samsung Galaxy"), runs),
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--hf-home", type=Path, default=Path(os.environ.get("HF_HOME", Path.home() / ".cache" / "huggingface"))
    )
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--runs", type=int, default=15)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args()
    registry = Registry.from_lock(ROOT / "ml" / "models.lock.json")
    backend = TorchBackend(registry, args.hf_home, args.threads)
    backend.load()
    result = {
        "machine": {
            "platform": platform.platform(),
            "processor": platform.processor(),
            "cpu_count": os.cpu_count(),
            "threads": args.threads,
        },
        "components": run(backend, args.runs),
    }
    text = json.dumps(result, indent=2) + "\n"
    print(text)
    if args.out:
        args.out.write_text(text, encoding="utf-8")


if __name__ == "__main__":
    main()
