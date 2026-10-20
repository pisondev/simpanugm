from __future__ import annotations

import csv
import io
import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from PIL import Image

import fetch_public


def jpeg_bytes(color: tuple[int, int, int]) -> bytes:
    image = Image.new("RGB", (320, 320), color)
    data = io.BytesIO()
    image.save(data, format="JPEG")
    return data.getvalue()


def candidate(number: int) -> dict[str, str | int]:
    return {
        "title": f"File:Reusable bottle {number}.jpg",
        "artist": f"Creator {number}",
        "license": "CC BY-SA 4.0",
        "description": "A reusable bottle on a plain table",
        "source_url": f"https://commons.wikimedia.org/wiki/File:Reusable_bottle_{number}.jpg",
        "download_url": f"https://upload.wikimedia.org/bottle-{number}.jpg",
        "width": 640,
        "height": 480,
    }


class FetchPublicTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.images = self.root / "images"
        self.images.mkdir()
        self.manifest = self.root / "SOURCES.csv"

    def read_rows(self) -> list[dict[str, str]]:
        with self.manifest.open(encoding="utf-8", newline="") as source:
            return list(csv.DictReader(source))

    def test_target_growth_preserves_existing_metadata_and_files(self) -> None:
        original = fetch_public.fixture_row("old.jpg", "buku", "background")
        original.update(source="wikimedia_commons", author="Original author", source_url="https://commons.wikimedia.org/wiki/File:old.jpg")
        fetch_public.write_manifest(self.manifest, [original])
        original_bytes = jpeg_bytes((10, 20, 30))
        (self.images / "old.jpg").write_bytes(original_bytes)
        results = [candidate(i) for i in range(4)]

        def save(url: str, path: Path) -> bool:
            path.write_bytes(jpeg_bytes((int(url.rsplit("-", 1)[1].split(".")[0]) * 20, 60, 90)))
            return True

        with patch.object(fetch_public.time, "sleep"), patch.object(
            fetch_public, "search_commons", side_effect=lambda query, limit, offset=0: [results.pop(0)]
        ), patch.object(fetch_public, "matches_category", return_value=True), patch.object(
            fetch_public, "download_and_process_image", side_effect=save
        ):
            count = fetch_public.run_fetch(self.root, self.manifest, target=3)

        rows = self.read_rows()
        self.assertEqual(count, 3)
        self.assertEqual(rows[0], original)
        self.assertEqual((self.images / "old.jpg").read_bytes(), original_bytes)
        self.assertEqual([row["item_type"] for row in rows[1:]], ["background", "background"])
        self.assertTrue(any("holdout" in row["notes"].lower() for row in rows))

    def test_search_rejects_missing_license_and_private_content_metadata(self) -> None:
        page = {
            "query": {
                "pages": {
                    "1": {
                        "title": "File:Wallet on table.jpg",
                        "imageinfo": [{
                            "mime": "image/jpeg",
                            "url": "https://upload.wikimedia.org/wallet.jpg",
                            "descriptionurl": "https://commons.wikimedia.org/wiki/File:Wallet_on_table.jpg",
                            "width": 640,
                            "height": 480,
                            "extmetadata": {"LicenseShortName": {"value": ""}},
                        }],
                    },
                    "2": {
                        "title": "File:Personal wallet.jpg",
                        "imageinfo": [{
                            "mime": "image/jpeg",
                            "url": "https://upload.wikimedia.org/wallet2.jpg",
                            "descriptionurl": "https://commons.wikimedia.org/wiki/File:Personal_wallet.jpg",
                            "width": 640,
                            "height": 480,
                            "extmetadata": {
                                "LicenseShortName": {"value": "CC BY 4.0"},
                                "Artist": {"value": "Creator"},
                                "ImageDescription": {"value": "A passport beside a wallet"},
                            },
                        }],
                    },
                }
            }
        }

        with patch.object(fetch_public.urllib.request, "urlopen", return_value=io.BytesIO(json.dumps(page).encode())):
            self.assertEqual(fetch_public.search_commons("wallet"), [])

    def test_duplicate_source_and_duplicate_bytes_are_skipped(self) -> None:
        original = candidate(1)
        original_row = fetch_public.result_row(original, "bg-dompet-001.jpg", "dompet", "background")
        fetch_public.write_manifest(self.manifest, [original_row])
        (self.images / original_row["filename"]).write_bytes(jpeg_bytes((30, 60, 90)))
        repeated = candidate(1)

        with patch.object(fetch_public.time, "sleep"), patch.object(
            fetch_public, "search_commons", return_value=[repeated]
        ), patch.object(fetch_public, "matches_category", return_value=True), patch.object(
            fetch_public, "download_and_process_image"
        ) as download:
            fetch_public.run_fetch(self.root, self.manifest, target=2)
            download.assert_not_called()
        self.assertEqual(len(self.read_rows()), 1)

    def test_offline_placeholder_is_explicitly_a_team_fixture(self) -> None:
        with patch.object(fetch_public, "TARGET_QUERIES", [("buku", "book", "background", "fixture.jpg")]):
            fetch_public.run_fetch(self.root, self.manifest, use_online=False)
        row = self.read_rows()[0]
        self.assertEqual(row["source"], "team_fixture")
        self.assertEqual(row["source_url"], "")
        self.assertIn("not a photograph", row["notes"])

    def test_target_bounds_reject_unbounded_collection(self) -> None:
        for value in (0, fetch_public.MAX_TARGET + 1):
            with self.subTest(target=value), self.assertRaises(ValueError):
                fetch_public.run_fetch(self.root, self.manifest, target=value)

    def test_license_parser_rejects_restricted_licenses(self) -> None:
        self.assertTrue(fetch_public.is_allowed_license("CC BY-SA 4.0"))
        self.assertTrue(fetch_public.is_allowed_license("Public domain"))
        self.assertFalse(fetch_public.is_allowed_license("CC BY-NC 4.0"))
        self.assertFalse(fetch_public.is_allowed_license("CC BY-ND 4.0"))


if __name__ == "__main__":
    unittest.main()
