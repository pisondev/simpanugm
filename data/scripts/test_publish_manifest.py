import csv
import json
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from intake import intake_dataset
from publish_manifest import ManifestPublishError, publish_manifest


class PublishManifestTests(unittest.TestCase):
    def create_source(self, directory: str) -> Path:
        source = Path(directory) / "raw"
        (source / "images" / "I001").mkdir(parents=True)
        with (source / "items.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.writer(handle)
            writer.writerow(["item_id", "item_name"])
            writer.writerow(["I001", "botol sintetis"])
        Image.new("RGB", (1080, 1080), "blue").save(source / "images" / "I001" / "finder_1.jpg")
        return source

    def test_publishes_only_the_validated_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            output_dir = root / "repo" / "data" / "eval"
            output_dir.mkdir(parents=True)
            output = output_dir / "manifest.json"

            published = publish_manifest(prepared, output)

            self.assertEqual(published, output.resolve())
            self.assertEqual(output.read_bytes(), (prepared / "manifest.json").read_bytes())
            self.assertFalse((output_dir / "images").exists())

    def test_existing_manifest_requires_explicit_replace(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            output = root / "manifest.json"
            output.write_text("keep\n", encoding="utf-8")

            with self.assertRaises(ManifestPublishError):
                publish_manifest(prepared, output)
            self.assertEqual(output.read_text(encoding="utf-8"), "keep\n")

            publish_manifest(prepared, output, replace=True)
            self.assertEqual(json.loads(output.read_text(encoding="utf-8"))["version"], "trial-1")

    def test_invalid_private_dataset_is_not_published(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            (prepared / "items.csv").write_text("tampered\n", encoding="utf-8")
            output = root / "manifest.json"

            with self.assertRaises(ManifestPublishError):
                publish_manifest(prepared, output)
            self.assertFalse(output.exists())

    def test_symlink_output_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            target = root / "target.json"
            target.write_text("keep\n", encoding="utf-8")
            output = root / "manifest.json"
            try:
                output.symlink_to(target)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(ManifestPublishError):
                publish_manifest(prepared, output, replace=True)
            self.assertEqual(target.read_text(encoding="utf-8"), "keep\n")


if __name__ == "__main__":
    unittest.main()
