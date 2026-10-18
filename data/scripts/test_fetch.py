import json
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from fetch import FetchError, fetch_dataset
from intake import intake_dataset


class FetchTests(unittest.TestCase):
    def create_source(self, directory: str) -> Path:
        source = Path(directory) / "raw"
        (source / "images" / "I001").mkdir(parents=True)
        (source / "items.csv").write_text("item_id,item_name\nI001,botol sintetis\n", encoding="utf-8")
        Image.new("RGB", (1080, 1080), "blue").save(source / "images" / "I001" / "finder_1.jpg")
        return source

    def test_fetches_only_manifest_files_and_preserves_dataset_card(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            output = root / "repo" / "data" / "eval"
            output.mkdir(parents=True)
            (output / "README.md").write_text("dataset card\n", encoding="utf-8")

            fetched = fetch_dataset(prepared / "manifest.json", prepared, output)

            self.assertEqual(fetched.resolve(), output.resolve())
            self.assertEqual((output / "README.md").read_text(encoding="utf-8"), "dataset card\n")
            self.assertEqual(
                (output / "items.csv").read_text(encoding="utf-8"),
                (prepared / "items.csv").read_text(encoding="utf-8"),
            )
            self.assertEqual(
                json.loads((output / "manifest.json").read_text(encoding="utf-8")),
                json.loads((prepared / "manifest.json").read_text(encoding="utf-8")),
            )

    def test_tampered_source_does_not_modify_output(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            output = root / "output"
            output.mkdir()
            (output / "README.md").write_text("keep\n", encoding="utf-8")
            (prepared / "items.csv").write_text("tampered\n", encoding="utf-8")

            with self.assertRaises(FetchError):
                fetch_dataset(prepared / "manifest.json", prepared, output)

            self.assertEqual((output / "README.md").read_text(encoding="utf-8"), "keep\n")
            self.assertFalse((output / "manifest.json").exists())

    def test_stale_output_is_rejected_without_deleting_it(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            output = root / "output"
            output.mkdir()
            stale = output / "old-private-file.csv"
            stale.write_text("keep\n", encoding="utf-8")

            with self.assertRaises(FetchError):
                fetch_dataset(prepared / "manifest.json", prepared, output)

            self.assertEqual(stale.read_text(encoding="utf-8"), "keep\n")

    def test_symlink_source_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            external = root / "external.csv"
            external.write_text("outside\n", encoding="utf-8")
            target = prepared / "items.csv"
            target.unlink()
            try:
                target.symlink_to(external)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(FetchError):
                fetch_dataset(prepared / "manifest.json", prepared, root / "output")

    def test_output_inside_private_source_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            output = prepared / "fetched"

            with self.assertRaises(FetchError):
                fetch_dataset(prepared / "manifest.json", prepared, output)

            self.assertFalse(output.exists())

    def test_output_through_symlinked_parent_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            prepared = root / "prepared"
            intake_dataset(self.create_source(directory), prepared, "trial-1")
            real_parent = root / "real-output"
            real_parent.mkdir()
            alias_parent = root / "output-alias"
            try:
                alias_parent.symlink_to(real_parent, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(FetchError):
                fetch_dataset(prepared / "manifest.json", prepared, alias_parent / "fetched")

            self.assertFalse((real_parent / "fetched").exists())


if __name__ == "__main__":
    unittest.main()
