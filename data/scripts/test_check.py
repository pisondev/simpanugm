import csv
import json
import shutil
import tempfile
import unittest
from argparse import ArgumentTypeError
from pathlib import Path
from unittest.mock import MagicMock, patch
from urllib.error import HTTPError

from check import (
    ValidationError,
    flatten_categories,
    parse_bbox as parse_seed_bbox,
    validate_brands,
    validate_seed_dir,
    validate_settings,
    validate_synonyms,
)
from check_trial import FIELDS, validate_trial
from osm_locations import (
    REVIEW_FIELDS,
    classify_kind,
    fetch_overpass,
    overpass_query,
    parse_bbox,
    review_rows,
    write_raw_response,
    write_rows,
)


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
COMPLETE_FILES = ("zones.csv", "locations.csv", "posts.yaml", "model_versions.yaml")


def copy_foundation_seed(destination: Path) -> Path:
    seed = destination / "seed"
    shutil.copytree(SEED_DIR, seed)
    for filename in COMPLETE_FILES:
        (seed / filename).unlink(missing_ok=True)
    return seed


class SeedCheckTests(unittest.TestCase):
    def test_foundation_seed_passes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            self.assertEqual(validate_seed_dir(copy_foundation_seed(Path(directory))), [])

    def test_seed_check_rejects_symlinked_seed_file(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            seed = Path(directory) / "seed"
            shutil.copytree(SEED_DIR, seed)
            external = Path(directory) / "categories.yaml"
            external.write_bytes((seed / "categories.yaml").read_bytes())
            (seed / "categories.yaml").unlink()
            try:
                (seed / "categories.yaml").symlink_to(external)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            errors = validate_seed_dir(seed)

            self.assertEqual(errors, ["seed file must not be a symlink: categories.yaml"])

    def test_seed_check_rejects_symlinked_seed_directory(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            seed = Path(directory) / "seed"
            shutil.copytree(SEED_DIR, seed)
            alias = Path(directory) / "seed-alias"
            try:
                alias.symlink_to(seed, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            errors = validate_seed_dir(alias)

            self.assertEqual(errors, [f"seed directory must not be a symlink: {alias}"])

    def test_complete_mode_keeps_deferred_files_explicit(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            errors = validate_seed_dir(
                copy_foundation_seed(Path(directory)), require_complete=True
            )
        self.assertEqual(
            errors,
            [
                "missing complete-seed file: zones.csv",
                "missing complete-seed file: locations.csv",
                "missing complete-seed file: posts.yaml",
                "missing complete-seed file: model_versions.yaml",
            ],
        )

    def test_seed_bbox_parser_rejects_non_finite_or_reversed_values(self) -> None:
        with self.assertRaises(ArgumentTypeError):
            parse_seed_bbox("-inf,110.36,-7.75,110.405")
        with self.assertRaises(ArgumentTypeError):
            parse_seed_bbox("-7.75,110.405,-7.79,110.36")

    def test_categories_have_expected_leaf_count(self) -> None:
        import check

        leaves, parents = flatten_categories(check.load_yaml(SEED_DIR / "categories.yaml"))
        self.assertEqual(len(parents), 13)
        self.assertEqual(len(leaves), 36)
        self.assertIn("ktm", leaves)
        self.assertIn("tumbler", leaves)

    def test_categories_reject_empty_parent_list(self) -> None:
        with self.assertRaisesRegex(ValidationError, "parents must contain at least one parent"):
            flatten_categories({"parents": []})

    def test_categories_reject_non_text_prompts_and_question_fields(self) -> None:
        document = {
            "parents": [
                {
                    "slug": "elektronik",
                    "name": "Elektronik",
                    "sort_order": 1,
                    "children": [
                        {
                            "slug": "ponsel",
                            "name": "Ponsel",
                            "sensitivity": "valuable",
                            "clip_prompts": [""],
                            "verification_questions": [
                                {
                                    "key": "phone_model",
                                    "question": "Sebutkan model.",
                                    "required": True,
                                    "check_hint": "Cocok.",
                                }
                            ],
                        }
                    ],
                }
            ]
        }
        with self.assertRaisesRegex(ValidationError, "clip prompts must be non-empty strings"):
            flatten_categories(document)

        document["parents"][0]["children"][0]["clip_prompts"] = ["a phone"]
        document["parents"][0]["children"][0]["verification_questions"][0]["required"] = "true"
        with self.assertRaisesRegex(ValidationError, "required must be boolean"):
            flatten_categories(document)

    def test_categories_reject_invalid_retention_days(self) -> None:
        document = {
            "parents": [
                {
                    "slug": "identitas",
                    "name": "Identitas",
                    "sort_order": 1,
                    "children": [
                        {
                            "slug": "ktm",
                            "name": "KTM",
                            "sensitivity": "identity",
                            "retention_days": 0,
                            "clip_prompts": ["a student card"],
                            "verification_questions": [
                                {
                                    "key": "identity_name",
                                    "question": "Sebutkan nama.",
                                    "required": True,
                                    "check_hint": "Sesuai.",
                                }
                            ],
                        }
                    ],
                }
            ]
        }
        with self.assertRaisesRegex(ValidationError, "retention_days must be a positive integer"):
            flatten_categories(document)

    def test_brand_aliases_reject_normalized_collisions(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "brands.csv"
            path.write_text(
                "name,aliases\nApple,apple;phone\nOther,OTHER; PHONE \n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValidationError, "normalized brand key"):
                validate_brands(path)

    def test_synonym_terms_reject_case_and_spacing_collisions(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "synonyms.csv"
            path.write_text(
                "term,expansion\nHP,ponsel\nhp  ,handphone\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValidationError, "unique after normalization"):
                validate_synonyms(path)

    def test_settings_reject_invalid_email_domain(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "settings.yaml"
            path.write_text(
                (SEED_DIR / "settings.yaml").read_text(encoding="utf-8").replace(
                    "mail.ugm.ac.id", "mail_ugm.ac.id"
                ),
                encoding="utf-8",
            )
            with self.assertRaisesRegex(ValidationError, "valid domain names"):
                validate_settings(path)

    def test_seed_check_reports_invalid_yaml(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            seed = Path(directory) / "seed"
            shutil.copytree(SEED_DIR, seed)
            (seed / "settings.yaml").write_text("auth: [\n", encoding="utf-8")

            errors = validate_seed_dir(seed)

            self.assertTrue(any("settings.yaml: invalid YAML" in error for error in errors))

    def test_csv_reader_rejects_short_and_extra_rows(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            short_path = Path(directory) / "short.csv"
            short_path.write_text("name,hex,sort_order\nblack,#000000\n", encoding="utf-8")
            extra_path = Path(directory) / "extra.csv"
            extra_path.write_text("name,hex,sort_order\nblack,#000000,1,unexpected\n", encoding="utf-8")

            with self.assertRaisesRegex(ValidationError, "row 2 must contain exactly 3 columns"):
                import check

                check.read_csv(short_path, ("name", "hex", "sort_order"))
            with self.assertRaisesRegex(ValidationError, "row 2 must contain exactly 3 columns"):
                check.read_csv(extra_path, ("name", "hex", "sort_order"))


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

    def test_classify_named_public_amenities_as_facilities(self) -> None:
        self.assertEqual(classify_kind({"amenity": "restaurant"}), "facility")
        self.assertEqual(classify_kind({"amenity": "place_of_worship"}), "facility")
        self.assertEqual(classify_kind({"building": "university"}), "facility")

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

    def test_review_csv_rejects_symlinked_parent(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            real_parent = root / "real-review"
            real_parent.mkdir()
            alias_parent = root / "review-alias"
            try:
                alias_parent.symlink_to(real_parent, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(OSError):
                write_rows(alias_parent / "locations.csv", [])

    def test_raw_response_is_written_as_json(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "overpass.json"
            write_raw_response(output, {"elements": [{"type": "node", "id": 1}]})
            self.assertEqual(json.loads(output.read_text(encoding="utf-8"))["elements"][0]["id"], 1)

    def test_raw_response_rejects_symlinked_parent(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            real_parent = root / "real-review"
            real_parent.mkdir()
            alias_parent = root / "review-alias"
            try:
                alias_parent.symlink_to(real_parent, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(OSError):
                write_raw_response(alias_parent / "overpass.json", {"elements": []})

    def test_review_rows_ignore_duplicate_osm_elements(self) -> None:
        element = {
            "type": "node",
            "id": 15,
            "lat": -7.77,
            "lon": 110.38,
            "tags": {"name": "Parkir Utara", "amenity": "parking"},
        }
        self.assertEqual(len(review_rows({"elements": [element, json.loads(json.dumps(element))]})), 1)

    def test_review_rows_filter_candidates_to_bbox(self) -> None:
        document = {
            "elements": [
                {
                    "type": "node",
                    "id": 16,
                    "lat": -7.77,
                    "lon": 110.38,
                    "tags": {"name": "Di Dalam", "amenity": "library"},
                },
                {
                    "type": "node",
                    "id": 17,
                    "lat": -7.80,
                    "lon": 110.38,
                    "tags": {"name": "Di Luar", "amenity": "library"},
                },
            ]
        }
        rows = review_rows(document, (-7.79, 110.36, -7.75, 110.405))
        self.assertEqual([row["name"] for row in rows], ["Di Dalam"])

    def test_overpass_query_limits_results_to_location_features(self) -> None:
        query = overpass_query((-7.79, 110.36, -7.75, 110.405))
        self.assertIn('nwr["name"]["building"]', query)
        self.assertIn('nwr["name"]["amenity"]', query)
        self.assertIn('nwr["name"]["highway"~"bus_stop|platform"]', query)
        self.assertNotIn('nwr["name"](-7.79', query)

    def test_bbox_rejects_non_finite_values(self) -> None:
        with self.assertRaises(ArgumentTypeError):
            parse_bbox("-inf,110.36,-7.75,110.405")

    def test_overpass_retries_transient_server_error(self) -> None:
        response = MagicMock()
        response.__enter__.return_value = response
        response.__exit__.return_value = False
        response.read.return_value = b'{"elements": []}'
        failure = HTTPError("https://example.test", 504, "gateway timeout", {}, None)
        with patch("osm_locations.urlopen", side_effect=[failure, response]) as urlopen_mock:
            with patch("osm_locations.time.sleep") as sleep_mock:
                self.assertEqual(fetch_overpass("https://example.test", (-7.79, 110.36, -7.75, 110.405)), {"elements": []})
        failure.close()
        self.assertEqual(urlopen_mock.call_count, 2)
        sleep_mock.assert_called_once_with(1)

    def test_overpass_does_not_retry_client_error(self) -> None:
        failure = HTTPError("https://example.test", 400, "bad request", {}, None)
        with patch("osm_locations.urlopen", side_effect=failure) as urlopen_mock:
            try:
                with self.assertRaises(HTTPError):
                    fetch_overpass("https://example.test", (-7.79, 110.36, -7.75, 110.405))
            finally:
                failure.close()
        urlopen_mock.assert_called_once()


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
