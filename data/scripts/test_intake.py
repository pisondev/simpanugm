import csv
import json
import shutil
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

    def test_shared_gps_fixture_is_stripped_by_intake(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = Path(directory) / "raw"
            image_dir = source / "images" / "I001"
            image_dir.mkdir(parents=True)
            with (source / "items.csv").open("w", newline="", encoding="utf-8") as handle:
                writer = csv.writer(handle)
                writer.writerow(["item_id", "item_name"])
                writer.writerow(["I001", "desk fixture"])
            fixture = Path(__file__).resolve().parents[1] / "fixtures" / "images" / "desk-gps.jpg"
            with Image.open(fixture) as image:
                self.assertTrue(image.getexif())
            shutil.copyfile(fixture, image_dir / "finder_1.jpg")

            output = Path(directory) / "prepared"
            intake_dataset(source, output, "fixture-1")
            with Image.open(output / "images" / "I001" / "finder_1.jpg") as image:
                self.assertFalse(image.getexif())

    def test_intake_rejects_non_empty_destination(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            output.mkdir()
            (output / "existing.txt").write_text("keep", encoding="utf-8")
            with self.assertRaises(IntakeError):
                intake_dataset(source, output, "trial-1")

    def test_intake_failure_does_not_leave_partial_output(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            (source / "images" / "I001" / "unsupported.gif").write_bytes(b"not an accepted image")
            output = Path(directory) / "prepared"
            with self.assertRaises(IntakeError):
                intake_dataset(source, output, "trial-1")
            self.assertFalse(output.exists())

    def test_intake_failure_preserves_existing_empty_destination(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            (source / "images" / "I001" / "unsupported.gif").write_bytes(b"not an accepted image")
            output = Path(directory) / "prepared"
            output.mkdir()
            with self.assertRaises(IntakeError):
                intake_dataset(source, output, "trial-1")
            self.assertEqual(list(output.iterdir()), [])

    def test_intake_rejects_symlink_image(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            symlink = source / "images" / "I001" / "finder_2.jpg"
            try:
                symlink.symlink_to(source / "images" / "I001" / "Finder-01.JPG")
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")
            with self.assertRaises(IntakeError):
                intake_dataset(source, Path(directory) / "prepared", "trial-1")

    def test_intake_rejects_symlink_image_directory(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            external = Path(directory) / "external-images"
            external.mkdir()
            symlink = source / "images" / "I002"
            try:
                symlink.symlink_to(external, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")
            with self.assertRaises(IntakeError):
                intake_dataset(source, Path(directory) / "prepared", "trial-1")

    def test_manifest_rejects_invalid_kind_and_policy(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            manifest_path = output / "manifest.json"
            document = json.loads(manifest_path.read_text(encoding="utf-8"))
            document["policy"]["image_names_normalized"] = False
            document["files"][0]["kind"] = "other"
            manifest_path.write_text(json.dumps(document), encoding="utf-8")
            errors = check_manifest(output)
            self.assertIn("manifest.policy.image_names_normalized must be true", errors)
            self.assertTrue(any("invalid manifest kind" in error for error in errors))

    def test_manifest_rejects_non_normalized_path(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            manifest_path = output / "manifest.json"
            document = json.loads(manifest_path.read_text(encoding="utf-8"))
            document["files"][0]["path"] = "./items.csv"
            manifest_path.write_text(json.dumps(document), encoding="utf-8")
            errors = check_manifest(output)
            self.assertIn("invalid manifest path: './items.csv'", errors)

    def test_manifest_rejects_symlink_file(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            image_path = output / "images" / "I001" / "finder_1.jpg"
            external = Path(directory) / "external.jpg"
            image_path.unlink()
            try:
                image_path.symlink_to(external)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")
            errors = check_manifest(output)
            self.assertTrue(any("must not be a symlink" in error for error in errors))

    def test_manifest_rejects_unlisted_symlink_directory(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-1")
            external = Path(directory) / "external-images"
            external.mkdir()
            symlink = output / "images" / "I002"
            try:
                symlink.symlink_to(external, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")
            errors = check_manifest(output)
            self.assertIn("unlisted symlink: images/I002", errors)

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
