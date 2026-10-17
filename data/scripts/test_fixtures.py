import unittest
from pathlib import Path

from check_fixtures import mask_visible_text, validate_fixture_dir


ROOT = Path(__file__).resolve().parents[2]
FIXTURE_DIR = ROOT / "data" / "fixtures"


class FixtureBundleTests(unittest.TestCase):
    def test_shared_fixture_bundle_passes(self) -> None:
        self.assertEqual(validate_fixture_dir(FIXTURE_DIR), [])

    def test_sixteen_digit_runs_are_masked_without_luhn(self) -> None:
        self.assertEqual(mask_visible_text("NIK 3404012345678901"), "NIK ************8901")

    def test_masking_fixture_uses_last_four_digits(self) -> None:
        self.assertEqual(
            mask_visible_text("Kartu sintetis 4532015112830366"),
            "Kartu sintetis ************0366",
        )


if __name__ == "__main__":
    unittest.main()
