import csv
import json
import tempfile
import unittest
from argparse import ArgumentTypeError
from pathlib import Path
from unittest.mock import patch

from check import flatten_categories, validate_seed_dir
from check_trial import FIELDS, validate_trial
from osm_locations import REVIEW_FIELDS, overpass_query, parse_bbox, review_rows, write_raw_response, write_rows


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"


class SeedCheckTests(unittest.TestCase):
    def test_foundation_seed_passes(self) -> None:
        self.assertEqual(validate_seed_dir(SEED_DIR), [])

    def test_complete_mode_keeps_deferred_files_explicit(self) -> None:
        errors = validate_seed_dir(SEED_DIR, require_complete=True)
        self.assertEqual(
            errors,
            [
                "missing complete-seed file: zones.csv",
                "missing complete-seed file: locations.csv",
                "missing complete-seed file: posts.yaml",
                "missing complete-seed file: model_versions.yaml",
            ],
        )

    def test_categories_have_expected_leaf_count(self) -> None:
        import check

        leaves, parents = flatten_categories(check.load_yaml(SEED_DIR / "categories.yaml"))
        self.assertEqual(len(parents), 13)
        self.assertEqual(len(leaves), 36)
        self.assertIn("ktm", leaves)
        self.assertIn("tumbler", leaves)


class OsmLocationTests(unittest.TestCase):
    def test_review_rows_use_centres_and_leave_review_fields_empty(self) -> None:
        rows = review_rows(
            {
                "elements": [
                    {
                        "type": "way",
                        "id": 12,
                        "center": {"lat": -7.77012345, "lon": 110.37876543},
                        "tags": {"name": "Perpustakaan Pusat", "amenity": "library"},
                    },
                    {
                        "type": "node",
                        "id": 13,
                        "lat": -7.771,
                        "lon": 110.379,
                        "tags": {"name": "Perpustakaan Pusat", "amenity": "library"},
                    },
                    {"type": "node", "id": 14, "tags": {"amenity": "parking"}},
                ]
            }
        )
        self.assertEqual([row["slug"] for row in rows], ["perpustakaan-pusat", "perpustakaan-pusat-2"])
        self.assertEqual(rows[0]["kind"], "facility")
        self.assertEqual(rows[0]["zone_slug"], "")
        self.assertEqual(rows[0]["review_status"], "needs_review")

    def test_review_csv_has_stable_columns(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "locations-review.csv"
            write_rows(output, review_rows({"elements": []}))
            self.assertEqual(output.read_text(encoding="utf-8").splitlines()[0], ",".join(REVIEW_FIELDS))

    def test_review_csv_write_failure_preserves_existing_output(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "locations-review.csv"
            output.write_text("keep\n", encoding="utf-8")
            with patch("osm_locations.csv.DictWriter", side_effect=OSError("disk full")):
                with self.assertRaises(OSError):
                    write_rows(output, [])
            self.assertEqual(output.read_text(encoding="utf-8"), "keep\n")

    def test_raw_response_is_written_as_json(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "overpass.json"
            write_raw_response(output, {"elements": [{"type": "node", "id": 1}]})
            self.assertEqual(json.loads(output.read_text(encoding="utf-8"))["elements"][0]["id"], 1)

    def test_review_rows_ignore_duplicate_osm_elements(self) -> None:
        element = {
            "type": "node",
            "id": 15,
            "lat": -7.77,
            "lon": 110.38,
            "tags": {"name": "Parkir Utara", "amenity": "parking"},
        }
        self.assertEqual(len(review_rows({"elements": [element, json.loads(json.dumps(element))]})), 1)

    def test_overpass_query_limits_results_to_location_features(self) -> None:
        query = overpass_query((-7.79, 110.36, -7.75, 110.405))
        self.assertIn('nwr["name"]["building"]', query)
        self.assertIn('nwr["name"]["amenity"]', query)
        self.assertIn('nwr["name"]["highway"~"bus_stop|platform"]', query)
        self.assertNotIn('nwr["name"](-7.79', query)

    def test_bbox_rejects_non_finite_values(self) -> None:
        with self.assertRaises(ArgumentTypeError):
            parse_bbox("-inf,110.36,-7.75,110.405")


class TrialCheckTests(unittest.TestCase):
    def write_rows(self, directory: str, rows: list[dict[str, str]]) -> Path:
        path = Path(directory) / "labels.csv"
        with path.open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=FIELDS)
            writer.writeheader()
            writer.writerows(rows)
        return path

    def valid_rows(self) -> list[dict[str, str]]:
        return [
            {
                "item_id": f"I{index:03d}",
                "leaf_category": "tumbler",
                "primary_colour": "biru",
                "secondary_colour": "",
                "brand": "",
                "distinctiveness": "generic",
                "has_owner_photo": "false",
                "flags": "",
            }
            for index in range(1, 21)
        ]

    def test_completed_trial_labels_pass(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_rows(directory, self.valid_rows())
            self.assertEqual(validate_trial(path, SEED_DIR), [])

    def test_trial_labels_reject_unknown_values(self) -> None:
        rows = self.valid_rows()
        rows[0]["leaf_category"] = "not-a-category"
        rows[0]["flags"] = "private_data"
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_rows(directory, rows)
            errors = validate_trial(path, SEED_DIR)
            self.assertEqual(len(errors), 2)

    def test_blank_template_is_not_accepted_as_completed_data(self) -> None:
        template = ROOT / "data" / "protocol" / "trial-intake-template.csv"
        errors = validate_trial(template, SEED_DIR)
        self.assertTrue(errors)


if __name__ == "__main__":
    unittest.main()
