import csv
import json
import shutil
import tempfile
import unittest
from pathlib import Path

from check import validate_complete_seed_dir, validate_seed_dir


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
ZONE_FIELDS = ("slug", "name", "centroid_lat", "centroid_lng")
LOCATION_FIELDS = ("slug", "name", "zone_slug", "kind", "lat", "lng", "aliases")


class CompleteSeedTests(unittest.TestCase):
    def write_complete_seed(self, directory: str) -> Path:
        seed = Path(directory) / "seed"
        shutil.copytree(SEED_DIR, seed)
        with (seed / "zones.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=ZONE_FIELDS)
            writer.writeheader()
            writer.writerows(
                {
                    "slug": slug,
                    "name": name,
                    "centroid_lat": "-7.77",
                    "centroid_lng": "110.38",
                }
                for slug, name in (
                    ("saintek", "Klaster Sains dan Teknologi"),
                    ("soshum", "Klaster Sosial Humaniora"),
                    ("agro", "Klaster Agro"),
                    ("medika", "Klaster Kesehatan"),
                    ("pusat", "Fasilitas Pusat"),
                )
            )
        with (seed / "locations.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=LOCATION_FIELDS)
            writer.writeheader()
            writer.writerows(
                (
                    {
                        "slug": "perpustakaan-pusat",
                        "name": "Perpustakaan Pusat",
                        "zone_slug": "pusat",
                        "kind": "facility",
                        "lat": "-7.7701",
                        "lng": "110.3787",
                        "aliases": "perpus",
                    },
                    {
                        "slug": "gedung-saintek",
                        "name": "Gedung Saintek",
                        "zone_slug": "saintek",
                        "kind": "building",
                        "lat": "-7.7710",
                        "lng": "110.3790",
                        "aliases": "",
                    },
                )
            )
        opening_hours = {weekday: [["07:00", "16:00"]] for weekday in ("mon", "tue", "wed", "thu", "fri", "sat", "sun")}
        posts = [
            {
                "slug": "pos-pusat",
                "name": "Pos Pusat",
                "location_slug": "perpustakaan-pusat",
                "phone": "+62274123456",
                "opening_hours": opening_hours,
            }
        ]
        (seed / "posts.yaml").write_text(json.dumps(posts), encoding="utf-8")
        model_versions = [
            {
                "component": component,
                "name": f"{component}-fixture-2026.11.0",
                "identity": {"source": "fixture"},
                "config": {"id": "fixture"} if component == "fusion" else None,
                "status": "active",
            }
            for component in (
                "image_encoder",
                "xmodal_text_encoder",
                "text_encoder",
                "ocr",
                "category_classifier",
                "colour_extractor",
                "brand_matcher",
                "fusion",
            )
        ]
        (seed / "model_versions.yaml").write_text(json.dumps(model_versions), encoding="utf-8")
        return seed

    def test_complete_seed_validates_all_references(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            seed = self.write_complete_seed(directory)
            self.assertEqual(validate_complete_seed_dir(seed), [])
            self.assertEqual(validate_seed_dir(seed, require_complete=True), [])

    def test_complete_seed_rejects_unknown_references(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            seed = self.write_complete_seed(directory)
            locations = (seed / "locations.csv").read_text(encoding="utf-8").replace(",pusat,", ",unknown,")
            (seed / "locations.csv").write_text(locations, encoding="utf-8")
            posts = (seed / "posts.yaml").read_text(encoding="utf-8").replace("perpustakaan-pusat", "missing-location")
            (seed / "posts.yaml").write_text(posts, encoding="utf-8")
            errors = validate_complete_seed_dir(seed)
            self.assertTrue(any("unknown zone" in error for error in errors))
            self.assertTrue(any("unknown location" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
