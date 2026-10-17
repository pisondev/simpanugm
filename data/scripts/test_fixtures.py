import json
import tempfile
import unittest
from pathlib import Path

from check_fixtures import mask_visible_text, validate_fixture_dir


ROOT = Path(__file__).resolve().parents[2]
FIXTURE_DIR = ROOT / "data" / "fixtures"


class FixtureBundleTests(unittest.TestCase):
    def test_shared_fixture_bundle_passes(self) -> None:
        self.assertEqual(validate_fixture_dir(FIXTURE_DIR), [])

    def test_masking_fixture_uses_last_four_digits(self) -> None:
        self.assertEqual(
            mask_visible_text("Kartu sintetis 4532015112830366"),
            "Kartu sintetis ************0366",
        )

    def test_fixture_validator_reports_malformed_json(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            fixture_dir = Path(directory)
            (fixture_dir / "fusion_cases.json").write_text("{", encoding="utf-8")

            errors = validate_fixture_dir(fixture_dir)

            self.assertTrue(any("fusion_cases.json: cannot read JSON" in error for error in errors))

    def test_fixture_validator_reports_non_object_cases(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            fixture_dir = Path(directory)
            (fixture_dir / "fusion_cases.json").write_text(
                json.dumps({"schema_version": 1, "fusion_version": "x", "cases": [[]]}),
                encoding="utf-8",
            )
            (fixture_dir / "masking_cases.json").write_text(
                json.dumps({"schema_version": 1, "rule": "x", "cases": [[]]}),
                encoding="utf-8",
            )
            (fixture_dir / "fuzzy_cases.json").write_text(
                json.dumps(
                    {
                        "schema_version": 1,
                        "metric": "x",
                        "identity_name_threshold": 85,
                        "cases": [[]],
                    }
                ),
                encoding="utf-8",
            )

            errors = validate_fixture_dir(fixture_dir)

            self.assertTrue(any("fusion case 1: expected an object" in error for error in errors))
            self.assertTrue(any("masking case 1: expected an object" in error for error in errors))
            self.assertTrue(any("fuzzy case 1: expected an object" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
