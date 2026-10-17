import csv
import tempfile
import unittest
from pathlib import Path

from create_trial_review_workspace import (
    REVIEW_LOG_FIELDS,
    ReviewWorkspaceError,
    create_review_workspace,
)


class TrialReviewWorkspaceTests(unittest.TestCase):
    def test_creates_two_annotation_sheets_and_review_log(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "review"
            created = create_review_workspace(output, item_count=3)
            self.assertEqual([path.name for path in created], ["labels-a.csv", "labels-b.csv", "review-log.csv"])
            for path in created:
                with path.open(newline="", encoding="utf-8") as handle:
                    reader = csv.DictReader(handle)
                    rows = list(reader)
                expected_fields = REVIEW_LOG_FIELDS if path.name == "review-log.csv" else (
                    "item_id",
                    "leaf_category",
                    "distinctiveness",
                )
                self.assertEqual(tuple(reader.fieldnames or ()), expected_fields)
                self.assertEqual([row["item_id"] for row in rows], ["I001", "I002", "I003"])
                self.assertTrue(all(not any(row[field] for field in expected_fields[1:]) for row in rows))

    def test_refuses_non_empty_output(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "review"
            output.mkdir()
            (output / "keep.txt").write_text("keep", encoding="utf-8")
            with self.assertRaises(ReviewWorkspaceError):
                create_review_workspace(output)
            self.assertEqual((output / "keep.txt").read_text(encoding="utf-8"), "keep")

    def test_rejects_non_positive_item_count(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(ReviewWorkspaceError):
                create_review_workspace(Path(directory) / "review", item_count=0)

    def test_rejects_symlinked_output_parent(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            real_parent = root / "real-output"
            real_parent.mkdir()
            alias_parent = root / "output-alias"
            try:
                alias_parent.symlink_to(real_parent, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(ReviewWorkspaceError):
                create_review_workspace(alias_parent / "review", item_count=1)

            self.assertFalse((real_parent / "review").exists())


if __name__ == "__main__":
    unittest.main()
