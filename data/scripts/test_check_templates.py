import csv
import tempfile
import unittest
from pathlib import Path

from check_templates import validate_template_dir
from create_eval_templates import create_templates, create_trial_templates


class TemplateCheckTests(unittest.TestCase):
    def test_trial_review_log_template_is_blank_and_stable(self) -> None:
        path = Path(__file__).resolve().parents[1] / "protocol" / "trial-review-log-template.csv"
        with path.open(newline="", encoding="utf-8") as handle:
            reader = csv.DictReader(handle)
            rows = list(reader)
        self.assertEqual(
            tuple(reader.fieldnames or ()),
            (
                "item_id",
                "consent_status",
                "photo_privacy_status",
                "photo_quality_status",
                "retake_reason",
                "category_disagreement",
                "distinctiveness_disagreement",
                "review_status",
                "private_drive_path",
            ),
        )
        self.assertEqual([row["item_id"] for row in rows], [f"I{index:03d}" for index in range(1, 21)])
        self.assertTrue(all(not any(row[field] for field in reader.fieldnames[1:]) for row in rows))

    def test_full_template_passes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "full"
            create_templates(output, item_count=3, distractor_count=4, duplicate_count=2, query_count=2)
            self.assertEqual(
                validate_template_dir(
                    output,
                    "full",
                    {"items.csv": 3, "labels.csv": 3, "distractors.csv": 4, "duplicates.csv": 2, "queries.csv": 2},
                ),
                [],
            )

    def test_trial_template_passes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "trial"
            create_trial_templates(output, item_count=3)
            self.assertEqual(
                validate_template_dir(output, "trial", {"items.csv": 3, "labels.csv": 3}),
                [],
            )

    def test_template_rejects_early_data_entry(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "trial"
            create_trial_templates(output, item_count=3)
            labels = output / "labels.csv"
            labels.write_text(labels.read_text(encoding="utf-8").replace(",,,,,,,", ",tumbler,biru,,,,,"), encoding="utf-8")
            errors = validate_template_dir(output, "trial", {"items.csv": 3, "labels.csv": 3})
            self.assertTrue(any("must be blank" in error for error in errors))

    def test_template_rejects_unexpected_file_and_image(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "trial"
            create_trial_templates(output, item_count=3)
            (output / "extra.csv").write_text("unexpected\n", encoding="utf-8")
            (output / "images" / "I001").mkdir()
            errors = validate_template_dir(output, "trial", {"items.csv": 3, "labels.csv": 3})
            self.assertIn("unexpected template entry: extra.csv", errors)
            self.assertTrue(any("template images/ must be empty" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
