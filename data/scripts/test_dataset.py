import csv
import tempfile
import unittest
from pathlib import Path

from PIL import Image, ImageDraw

from check_dataset import validate_dataset
from intake import intake_dataset


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
LABEL_FIELDS = (
    "item_id",
    "leaf_category",
    "primary_colour",
    "secondary_colour",
    "brand",
    "distinctiveness",
    "has_owner_photo",
    "flags",
)


class DatasetQualityTests(unittest.TestCase):
    def create_source(
        self,
        directory: str,
        duplicate_owner_description: bool = False,
        duplicate_across_roles: bool = False,
    ) -> Path:
        source = Path(directory) / "raw"
        (source / "images" / "I001").mkdir(parents=True)
        (source / "images" / "I002").mkdir(parents=True)
        with (source / "items.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(
                handle,
                fieldnames=("item_id", "item_name", "owner_description", "finder_description"),
            )
            writer.writeheader()
            writer.writerow(
                {
                    "item_id": "I001",
                    "item_name": "botol biru",
                    "owner_description": "Botol biru dengan goresan kecil di dekat tutupnya.",
                    "finder_description": "Botol minum biru ditemukan di meja kelas.",
                }
            )
            writer.writerow(
                {
                    "item_id": "I002",
                    "item_name": "pouch hitam",
                    "owner_description": (
                        "Botol biru dengan goresan kecil di dekat tutupnya."
                        if duplicate_owner_description
                        else "Pouch hitam dengan resleting merah dan jahitan miring."
                    ),
                    "finder_description": (
                        "Botol minum biru ditemukan di meja kelas."
                        if duplicate_across_roles
                        else "Pouch kecil berwarna hitam ditemukan dekat kursi."
                    ),
                }
            )
        with (source / "labels.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=LABEL_FIELDS)
            writer.writeheader()
            writer.writerow(
                {
                    "item_id": "I001",
                    "leaf_category": "botol-minum",
                    "primary_colour": "biru",
                    "secondary_colour": "",
                    "brand": "",
                    "distinctiveness": "generic",
                    "has_owner_photo": "true",
                    "flags": "",
                }
            )
            writer.writerow(
                {
                    "item_id": "I002",
                    "leaf_category": "pouch",
                    "primary_colour": "hitam",
                    "secondary_colour": "",
                    "brand": "",
                    "distinctiveness": "distinctive",
                    "has_owner_photo": "false",
                    "flags": "",
                }
            )
        image_colours = {
            "I001": ("blue", "navy", "cyan"),
            "I002": ("red", "maroon"),
        }
        for item_id, names in {
            "I001": ("finder_1.jpg", "finder_2.jpg", "owner_1.jpg"),
            "I002": ("finder_1.jpg", "finder_2.jpg"),
        }.items():
            for name, colour in zip(names, image_colours[item_id]):
                image = Image.new("RGB", (1080, 1080), "white")
                draw = ImageDraw.Draw(image)
                if item_id == "I001":
                    draw.rectangle((180, 180, 720, 720), fill=colour)
                else:
                    draw.ellipse((180, 180, 720, 720), fill=colour)
                image.save(source / "images" / item_id / name)
        return source

    def test_quality_gate_accepts_complete_two_item_fixture(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            self.assertEqual(validate_dataset(output, SEED_DIR, expected_count=2), [])

    def test_quality_gate_rejects_duplicate_descriptions(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory, duplicate_owner_description=True)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            errors = validate_dataset(output, SEED_DIR, expected_count=2)
            self.assertTrue(any("owner_description duplicates I001" in error for error in errors))

    def test_quality_gate_rejects_cross_role_duplicate_descriptions(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory, duplicate_across_roles=True)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            errors = validate_dataset(output, SEED_DIR, expected_count=2)
            self.assertTrue(any("finder_description duplicates I001" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
