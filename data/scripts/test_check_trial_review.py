import csv
import tempfile
import unittest
from pathlib import Path

from check_trial_review import validate_trial_review
from create_trial_review_workspace import REVIEW_LOG_FIELDS, create_review_workspace


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
ANNOTATION_FIELDS = ("item_id", "leaf_category", "distinctiveness")


class TrialReviewCheckTests(unittest.TestCase):
    def write_rows(self, path: Path, fields: tuple[str, ...], rows: list[dict[str, str]]) -> None:
        with path.open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=fields)
            writer.writeheader()
            writer.writerows(rows)

    def read_rows(self, path: Path) -> list[dict[str, str]]:
        with path.open(newline="", encoding="utf-8") as handle:
            return list(csv.DictReader(handle))

    def complete_workspace(self, directory: str) -> Path:
        workspace = Path(directory) / "review"
        create_review_workspace(workspace, item_count=3)
        annotations = [
            {"item_id": "I001", "leaf_category": "botol-minum", "distinctiveness": "generic"},
            {"item_id": "I002", "leaf_category": "tumbler", "distinctiveness": "distinctive"},
            {"item_id": "I003", "leaf_category": "payung", "distinctiveness": "generic"},
        ]
        self.write_rows(workspace / "labels-a.csv", ANNOTATION_FIELDS, annotations)
        self.write_rows(workspace / "labels-b.csv", ANNOTATION_FIELDS, annotations)
        review_rows = [
            {
                "item_id": row["item_id"],
                "consent_status": "pass",
                "photo_privacy_status": "pass",
                "photo_quality_status": "pass",
                "retake_reason": "",
                "category_disagreement": "no",
                "distinctiveness_disagreement": "no",
                "review_status": "pass",
                "private_drive_path": "/private/trial-20",
            }
            for row in annotations
        ]
        self.write_rows(workspace / "review-log.csv", REVIEW_LOG_FIELDS, review_rows)
        return workspace

    def test_completed_review_passes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            self.assertEqual(validate_trial_review(self.complete_workspace(directory), SEED_DIR, 3), [])

    def test_blank_workspace_is_not_complete(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            workspace = Path(directory) / "review"
            create_review_workspace(workspace, item_count=3)
            errors = validate_trial_review(workspace, SEED_DIR, 3)
            self.assertTrue(any("labels-a.csv" in error for error in errors))
            self.assertTrue(any("consent_status must be pass" in error for error in errors))

    def test_failed_photo_review_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            workspace = self.complete_workspace(directory)
            rows = self.read_rows(workspace / "review-log.csv")
            rows[0]["photo_privacy_status"] = "fail"
            self.write_rows(workspace / "review-log.csv", REVIEW_LOG_FIELDS, rows)
            errors = validate_trial_review(workspace, SEED_DIR, 3)
            self.assertIn("I001: photo_privacy_status must be pass for trial completion", errors)

    def test_review_log_requires_disagreement_value_and_drive_path(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            workspace = self.complete_workspace(directory)
            rows = self.read_rows(workspace / "review-log.csv")
            rows[0]["category_disagreement"] = ""
            rows[0]["private_drive_path"] = ""
            self.write_rows(workspace / "review-log.csv", REVIEW_LOG_FIELDS, rows)
            errors = validate_trial_review(workspace, SEED_DIR, 3)
            self.assertIn("I001: category_disagreement must be yes or no", errors)
            self.assertIn("I001: private_drive_path cannot be blank", errors)


if __name__ == "__main__":
    unittest.main()
