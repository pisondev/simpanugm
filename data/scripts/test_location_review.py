import csv
import tempfile
import unittest
from pathlib import Path

from check_location_review import validate_provenance, validate_review
from osm_locations import REVIEW_FIELDS, review_rows, write_raw_response, write_rows


class LocationReviewTests(unittest.TestCase):
    def raw_document(self) -> dict:
        return {
            "elements": [
                {
                    "type": "way",
                    "id": 42,
                    "center": {"lat": -7.77, "lon": 110.38},
                    "tags": {"name": "Gedung Sintetis", "building": "yes"},
                },
                {
                    "type": "node",
                    "id": 43,
                    "lat": -7.771,
                    "lon": 110.381,
                    "tags": {"name": "Taman Sintetis", "leisure": "park"},
                },
            ]
        }

    def write_bundle(self, directory: str) -> tuple[Path, Path]:
        root = Path(directory)
        raw_path = root / "overpass.json"
        review_path = root / "locations-review.csv"
        document = self.raw_document()
        write_raw_response(raw_path, document)
        write_rows(review_path, review_rows(document))
        return raw_path, review_path

    def test_generated_review_matches_raw_response(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            raw_path, review_path = self.write_bundle(directory)
            self.assertEqual(validate_review(raw_path, review_path, (-7.78, 110.37, -7.76, 110.39)), [])

    def test_review_rejects_changed_source_fields(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            raw_path, review_path = self.write_bundle(directory)
            with review_path.open(newline="", encoding="utf-8") as handle:
                rows = list(csv.DictReader(handle))
            rows[0]["name"] = "Nama Diubah"
            with review_path.open("w", newline="", encoding="utf-8") as handle:
                writer = csv.DictWriter(handle, fieldnames=REVIEW_FIELDS)
                writer.writeheader()
                writer.writerows(rows)
            errors = validate_review(raw_path, review_path)
            self.assertTrue(any("name differs from raw extraction" in error for error in errors))

    def test_review_rejects_manual_values_before_review(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            raw_path, review_path = self.write_bundle(directory)
            with review_path.open(newline="", encoding="utf-8") as handle:
                rows = list(csv.DictReader(handle))
            rows[0]["zone_slug"] = "pusat"
            rows[0]["review_status"] = "approved"
            with review_path.open("w", newline="", encoding="utf-8") as handle:
                writer = csv.DictWriter(handle, fieldnames=REVIEW_FIELDS)
                writer.writeheader()
                writer.writerows(rows)
            errors = validate_review(raw_path, review_path)
            self.assertTrue(any("must remain needs_review" in error for error in errors))
            self.assertTrue(any("zone_slug and aliases must remain empty" in error for error in errors))

    def test_provenance_allows_review_fields_after_human_review(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            raw_path, review_path = self.write_bundle(directory)
            with review_path.open(newline="", encoding="utf-8") as handle:
                rows = list(csv.DictReader(handle))
            rows[0]["zone_slug"] = "pusat"
            rows[0]["aliases"] = "gedung sintetis"
            rows[0]["review_status"] = "approved"
            with review_path.open("w", newline="", encoding="utf-8") as handle:
                writer = csv.DictWriter(handle, fieldnames=REVIEW_FIELDS)
                writer.writeheader()
                writer.writerows(rows)

            self.assertEqual(validate_provenance(raw_path, review_path), [])


if __name__ == "__main__":
    unittest.main()
