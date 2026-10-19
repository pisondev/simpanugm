"""A tiny dataset in the prepared layout of dataset-collection.md section 6, for exercising the harness."""

import csv
import random
from pathlib import Path

import numpy as np
from PIL import Image

from tests.conftest import FIXTURES

LEAVES = ["tumbler", "dompet", "payung", "ktm"]
COLOURS = ["biru", "cokelat", "hitam", "putih"]
SEED = FIXTURES.parent / "seed"


def _image(path: Path, seed: int) -> None:
    rng = np.random.default_rng(seed)
    pixels = rng.integers(0, 255, (96, 96, 3), dtype=np.uint8)
    path.parent.mkdir(parents=True, exist_ok=True)
    Image.fromarray(pixels).save(path, format="JPEG")


def _write(path: Path, header: list[str], rows: list[list[str]]) -> None:
    with path.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.writer(handle)
        writer.writerow(header)
        writer.writerows(rows)


def build(root: Path, items: int = 10, distractors: int = 6, leak: bool = False, val_count: int | None = None) -> Path:
    rng = random.Random(7)  # noqa: S311
    item_rows, label_rows, split_rows, dist_rows, dsplit_rows = [], [], [], [], []
    for n in range(1, items + 1):
        iid = f"I{n:03d}"
        leaf, colour = LEAVES[n % len(LEAVES)], COLOURS[n % len(COLOURS)]
        photo = n % 3 == 0
        item_rows.append(
            [
                iid,
                f"{leaf} {colour} {n}",
                f"{leaf} {colour} milik saya nomor {n} dengan goresan",
                f"{leaf} {colour} ditemukan nomor {n} di meja",
            ]
        )
        label_rows.append(
            [
                iid,
                leaf,
                colour,
                "",
                "Eiger" if n % 4 == 0 else "",
                "distinctive" if n % 2 else "generic",
                "true" if photo else "false",
            ]
        )
        split = "val" if (val_count is None or n <= val_count) else "test"
        split_rows.append([iid, split, "x", "x"])
        _image(root / "images" / iid / "finder_1.jpg", n)
        _image(root / "images" / iid / "finder_2.jpg", 100 + n)
        if photo:
            _image(root / "images" / iid / "owner_1.jpg", 200 + n)
    for n in range(1, distractors + 1):
        did = f"D{n:03d}"
        leaf = LEAVES[rng.randrange(len(LEAVES))]
        dist_rows.append(
            [did, leaf, COLOURS[rng.randrange(len(COLOURS))], "generic", f"{leaf} tanpa pemilik nomor {n}"]
        )
        dist_val = (val_count * distractors // items) if val_count is not None else distractors
        dsplit = "val" if (val_count is None or n <= dist_val) else "test"
        dsplit_rows.append([did, dsplit])
        _image(root / "images" / did / "finder_1.jpg", 300 + n)
    if leak:
        # the finder photo of I001 filed again under a distractor of the test split
        dsplit_rows[0][1] = "test"
        (root / "images" / "D001" / "finder_1.jpg").write_bytes(
            (root / "images" / "I001" / "finder_1.jpg").read_bytes()
        )
    _write(root / "items.csv", ["item_id", "item_name", "owner_description", "finder_description"], item_rows)
    _write(
        root / "labels.csv",
        [
            "item_id",
            "leaf_category",
            "primary_colour",
            "secondary_colour",
            "brand",
            "distinctiveness",
            "has_owner_photo",
        ],
        label_rows,
    )
    _write(root / "splits.csv", ["item_id", "split", "parent_category", "distinctiveness"], split_rows)
    _write(
        root / "distractors.csv",
        ["distractor_id", "leaf_category", "primary_colour", "distinctiveness", "description"],
        dist_rows,
    )
    _write(root / "distractor-splits.csv", ["distractor_id", "split"], dsplit_rows)
    _write(
        root / "duplicates.csv",
        ["pair_id", "item_id", "first_report_id", "second_report_id"],
        [["P1", "I001", "I001", "D002"]],
    )
    _write(
        root / "queries.csv", ["query_id", "query_text", "target_report_id"], [["Q1", "tumbler biru ditemukan", "I001"]]
    )
    return root
