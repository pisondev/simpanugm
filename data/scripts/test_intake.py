import csv
import json
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from check_manifest import check_manifest
from intake import IntakeError, intake_dataset


class IntakeTests(unittest.TestCase):
    def create_source(self, directory: str) -> Path:
        source = Path(directory) / "raw"
        (source / "images" / "I001").mkdir(parents=True)
        with (source / "items.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.writer(handle)
            writer.writerow(["item_id", "item_name"])
            writer.writerow(["I001", "botol biru"])
        with (source / "labels.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.writer(handle)
            writer.writerow(["item_id", "leaf_category"])
            writer.writerow(["I001", "botol-minum"])
        image = Image.new("RGB", (64, 64), "blue")
        image.save(
            source / "images" / "I001" / "Finder-01.JPG",
            exif=b"Exif\x00\x00test",
        )
        return source

    def test_intake_normalizes_names_strips_exif_and_writes_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            manifest_path = intake_dataset(source, output, "trial-1")
            self.assertEqual(manifest_path.name, "manifest.json")
            image_path = output / "images" / "I001" / "finder_1.jpg"
            self.assertTrue(image_path.is_file())
            document = json.loads(manifest_path.read_text(encoding="utf-8"))
            self.assertEqual(document["version"], "trial-1")
            self.assertEqual(check_manifest(output), [])
            with Image.open(image_path) as image:
                self.assertFalse(image.getexif())

    def test_intake_rejects_non_empty_destination(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            output.mkdir()
            (output / "existing.txt").write_text("keep", encoding="utf-8")
            with self.assertRaises(IntakeError):
                intake_dataset(source, output, "trial-1")

    def test_manifest_detects_tampered_file(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            (output / "items.csv").write_text("tampered\n", encoding="utf-8")
            errors = check_manifest(output)
            self.assertIn("sha256 mismatch: items.csv", errors)


if __name__ == "__main__":
    unittest.main()
