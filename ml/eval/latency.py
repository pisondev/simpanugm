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


def resolve_seed_dir(custom_path: Path | None = None) -> Path:
    if custom_path and custom_path.is_dir():
        return custom_path
    env_path = os.environ.get("ML_SEED_DIR")
    if env_path and Path(env_path).is_dir():
        return Path(env_path)
    if Path("/app/seed").is_dir():
        return Path("/app/seed")
    if SEED_DIR.is_dir():
        return SEED_DIR
    return Path("seed")


def resolve_lock_path(custom_path: Path | None = None) -> Path:
    if custom_path and custom_path.is_file():
        return custom_path
    candidates = [
        Path("models.lock.json"),
        Path("/app/models.lock.json"),
        ROOT / "ml" / "models.lock.json",
    ]
    return next((p for p in candidates if p.is_file()), ROOT / "ml" / "models.lock.json")


def load_sample_images(fixtures_dir: Path | None = None) -> tuple[Image.Image, Image.Image]:
    base_dir = fixtures_dir or FIXTURES
    desk_path = base_dir / "images" / "desk-gps.jpg"
    card_path = base_dir / "images" / "mock-card.png"
    if desk_path.is_file() and card_path.is_file():
        desk = Image.open(desk_path).convert("RGB")
        card = Image.open(card_path).convert("RGB")
    else:
        # Fallback for container runtime without fixtures directory
        desk = Image.new("RGB", (224, 224), color=(120, 130, 140))
        card = Image.new("RGB", (320, 200), color=(240, 240, 240))
    return desk, card


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


def run(
    backend: Backend,
    runs: int,
    seed_dir: Path | None = None,
    fixtures_dir: Path | None = None,
) -> dict[str, dict[str, float]]:
    desk, card = load_sample_images(fixtures_dir)
    target_seed = resolve_seed_dir(seed_dir)
    colours = ColourExtractor(load_palette(target_seed / "colours.csv"))
    brands = BrandMatcher(load_brands(target_seed / "brands.csv"))
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
    parser.add_argument("--lock", type=Path, help="path to models.lock.json")
    parser.add_argument("--seed-dir", type=Path, help="path to seed directory containing colours.csv and brands.csv")
    parser.add_argument("--fixtures-dir", type=Path, help="path to fixtures directory containing test images")
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--runs", type=int, default=15)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args()

    lock_path = resolve_lock_path(args.lock)
    registry = Registry.from_lock(lock_path)
    backend = TorchBackend(registry, args.hf_home, args.threads)
    backend.load()
    result = {
        "machine": {
            "platform": platform.platform(),
            "processor": platform.processor(),
            "cpu_count": os.cpu_count(),
            "threads": args.threads,
        },
        "components": run(backend, args.runs, seed_dir=args.seed_dir, fixtures_dir=args.fixtures_dir),
    }
    text = json.dumps(result, indent=2) + "\n"
    print(text)
    if args.out:
        args.out.write_text(text, encoding="utf-8")


if __name__ == "__main__":
    main()
