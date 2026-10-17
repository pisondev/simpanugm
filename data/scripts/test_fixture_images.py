import tempfile
import unittest
from pathlib import Path

from PIL import Image

from create_fixture_images import ASSET_NAMES, check_fixture_images, create_fixture_images


ROOT = Path(__file__).resolve().parents[2]
FIXTURE_DIR = ROOT / "data" / "fixtures" / "images"


class FixtureImageTests(unittest.TestCase):
    def test_committed_fixture_images_have_expected_properties(self) -> None:
        for name in ASSET_NAMES:
            self.assertTrue((FIXTURE_DIR / name).is_file(), name)
        with Image.open(FIXTURE_DIR / "swatch-biru.png") as image:
            self.assertEqual(image.size, (64, 64))
        with Image.open(FIXTURE_DIR / "swatch-merah.png") as image:
            self.assertEqual(image.size, (64, 64))
        with Image.open(FIXTURE_DIR / "desk-gps.jpg") as image:
            self.assertEqual(image.size, (320, 240))
            self.assertIn(34853, image.getexif())
        with Image.open(FIXTURE_DIR / "mock-card.png") as image:
            self.assertEqual(image.size, (400, 240))

    def test_generator_recreates_all_assets_in_a_clean_directory(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "images"
            created = create_fixture_images(output)
            self.assertEqual([path.name for path in created], list(ASSET_NAMES))
            self.assertEqual(sorted(path.name for path in output.iterdir()), sorted(ASSET_NAMES))

    def test_committed_images_match_reviewed_hashes(self) -> None:
        self.assertEqual(check_fixture_images(FIXTURE_DIR), [])

    def test_reviewed_hashes_reject_changed_bytes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            for name in ASSET_NAMES:
                (output / name).write_bytes((FIXTURE_DIR / name).read_bytes())
            changed = bytearray((output / "swatch-biru.png").read_bytes())
            changed[-1] ^= 1
            (output / "swatch-biru.png").write_bytes(changed)

            errors = check_fixture_images(output)

        self.assertEqual(len(errors), 1)
        self.assertIn("swatch-biru.png", errors[0])


if __name__ == "__main__":
    unittest.main()
