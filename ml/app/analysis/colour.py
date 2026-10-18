import csv
from dataclasses import dataclass
from pathlib import Path

import numpy as np
from numpy.typing import NDArray
from PIL import Image
from skimage.color import deltaE_ciede2000, rgb2lab
from sklearn.cluster import KMeans

SAMPLE_SIDE = 128
CENTRE_KEEP = 0.70
CLUSTERS = 4
MIN_SHARE = 0.10
MAX_COLOURS = 2


@dataclass(frozen=True)
class PaletteColour:
    name: str
    hex: str


@dataclass(frozen=True)
class Detected:
    name: str
    hex: str
    share: float


def load_palette(path: Path) -> list[PaletteColour]:
    with path.open(newline="", encoding="utf-8") as handle:
        return [PaletteColour(row["name"], row["hex"]) for row in csv.DictReader(handle)]


def _lab(rgb: NDArray[np.float64]) -> NDArray[np.float64]:
    return np.asarray(rgb2lab(rgb.reshape(-1, 1, 3) / 255.0)).reshape(-1, 3)


class ColourExtractor:
    """matching-engine section 4: k-means in CIELAB, named by the nearest palette colour under CIEDE2000."""

    def __init__(self, palette: list[PaletteColour]) -> None:
        self.palette = palette
        rgb = np.array([[int(c.hex[i : i + 2], 16) for i in (1, 3, 5)] for c in palette], dtype=np.float64)
        self._palette_lab = _lab(rgb)

    def name_of(self, lab: NDArray[np.float64]) -> int:
        distances = deltaE_ciede2000(np.tile(lab, (len(self.palette), 1)), self._palette_lab)
        return int(np.argmin(distances))

    def extract(self, image: Image.Image) -> list[Detected]:
        small = image.convert("RGB").resize((SAMPLE_SIDE, SAMPLE_SIDE), Image.Resampling.BILINEAR)
        margin = int(SAMPLE_SIDE * (1 - CENTRE_KEEP) / 2)
        pixels = np.asarray(small, dtype=np.float64)[margin : SAMPLE_SIDE - margin, margin : SAMPLE_SIDE - margin]
        lab = _lab(pixels.reshape(-1, 3))
        k = min(CLUSTERS, len(np.unique(lab, axis=0)))
        model = KMeans(n_clusters=k, n_init=10, random_state=0).fit(lab)
        counts = np.bincount(model.labels_, minlength=k) / len(lab)
        shares: dict[int, float] = {}
        for centre, share in zip(model.cluster_centers_, counts, strict=True):
            if share < MIN_SHARE:
                continue
            index = self.name_of(centre)
            shares[index] = shares.get(index, 0.0) + float(share)
        ranked = sorted(shares.items(), key=lambda kv: kv[1], reverse=True)[:MAX_COLOURS]
        return [Detected(self.palette[i].name, self.palette[i].hex, round(s, 4)) for i, s in ranked]
