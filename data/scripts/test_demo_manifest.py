"""Unit tests for demo photo pack manifest validator."""

from __future__ import annotations

import csv
import sys
import tempfile
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from PIL import Image

from data.demo.validate_manifest import (
    MAX_TOTAL_BYTES,
    REQUIRED_COLUMNS,
    validate_demo_pack,
)


class DemoManifestValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.images_dir = self.root / "images"
        self.images_dir.mkdir(parents=True, exist_ok=True)
        self.manifest_file = self.root / "SOURCES.csv"

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def _create_image(self, name: str, size: tuple[int, int] = (640, 480)) -> Path:
        p = self.images_dir / name
        img = Image.new("RGB", size, color=(200, 200, 200))
        img.save(p, "JPEG")
        return p

    def _write_manifest(self, rows: list[dict[str, str]]) -> None:
        with open(self.manifest_file, "w", newline="", encoding="utf-8") as f:
            writer = csv.DictWriter(f, fieldnames=REQUIRED_COLUMNS)
            writer.writeheader()
            writer.writerows(rows)

    def test_missing_manifest_fails(self) -> None:
        res = validate_demo_pack(self.root)
        self.assertFalse(res.ok)
        self.assertTrue(any("Manifest not found" in err for err in res.errors))

    def test_valid_pack_passes(self) -> None:
        self._create_image("item1.jpg", (500, 500))
        self._write_manifest([
            {
                "filename": "item1.jpg",
                "source": "wikimedia_commons",
                "author": "Test Author",
                "license": "CC-BY 4.0",
                "source_url": "https://commons.wikimedia.org/wiki/File:Item1.jpg",
                "category": "earphone",
                "item_type": "hero_found",
                "notes": "Test item notes",
            }
        ])

        res = validate_demo_pack(self.root)
        self.assertTrue(res.ok, msg=f"Validation failed with: {res.errors}")
        self.assertEqual(res.total_images, 1)
        self.assertGreater(res.total_bytes, 0)
        self.assertEqual(res.errors, [])

    def test_unapproved_source_fails(self) -> None:
        self._create_image("item1.jpg", (500, 500))
        self._write_manifest([
            {
                "filename": "item1.jpg",
                "source": "unapproved_website",
                "author": "Someone",
                "license": "CC0",
                "source_url": "https://example.com/item.jpg",
                "category": "earphone",
                "item_type": "hero_found",
                "notes": "Unauthorized",
            }
        ])
        res = validate_demo_pack(self.root)
        self.assertFalse(res.ok)
        self.assertTrue(any("not in approved sources" in err for err in res.errors))

    def test_unapproved_license_fails(self) -> None:
        self._create_image("item1.jpg", (500, 500))
        self._write_manifest([
            {
                "filename": "item1.jpg",
                "source": "wikimedia_commons",
                "author": "Someone",
                "license": "All Rights Reserved",
                "source_url": "https://commons.wikimedia.org/wiki/File:Item.jpg",
                "category": "earphone",
                "item_type": "hero_found",
                "notes": "Non-open license",
            }
        ])
        res = validate_demo_pack(self.root)
        self.assertFalse(res.ok)
        self.assertTrue(any("not an approved open license" in err for err in res.errors))

    def test_image_dimension_too_small_fails(self) -> None:
        self._create_image("item_small.jpg", (200, 200))
        self._write_manifest([
            {
                "filename": "item_small.jpg",
                "source": "team_fixture",
                "author": "Team",
                "license": "CC0 1.0",
                "source_url": "https://commons.wikimedia.org",
                "category": "earphone",
                "item_type": "hero_found",
                "notes": "Too small",
            }
        ])
        res = validate_demo_pack(self.root)
        self.assertFalse(res.ok)
        self.assertTrue(any("smaller than min 320px" in err for err in res.errors))

    def test_unmanifested_file_fails(self) -> None:
        self._create_image("item1.jpg", (500, 500))
        self._create_image("unmanifested.jpg", (500, 500))
        self._write_manifest([
            {
                "filename": "item1.jpg",
                "source": "wikimedia_commons",
                "author": "Author",
                "license": "CC0",
                "source_url": "https://commons.wikimedia.org/test",
                "category": "earphone",
                "item_type": "hero_found",
                "notes": "Manifested",
            }
        ])
        res = validate_demo_pack(self.root)
        self.assertFalse(res.ok)
        self.assertTrue(any("unmanifested files in images/" in err for err in res.errors))


if __name__ == "__main__":
    unittest.main()
