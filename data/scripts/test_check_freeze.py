import unittest
from pathlib import Path
from unittest.mock import patch

from check_freeze import validate_freeze


class FreezeCheckTests(unittest.TestCase):
    def test_trial_requires_review_workspace(self) -> None:
        with patch("check_freeze.validate_dataset", return_value=[]):
            errors = validate_freeze(Path("dataset"), Path("seed"), "trial")
        self.assertIn("trial: --review-workspace is required", errors)

    def test_trial_runs_dataset_and_review_gates(self) -> None:
        with (
            patch("check_freeze.validate_dataset", return_value=[]) as dataset_check,
            patch("check_freeze.validate_trial_review", return_value=[]) as review_check,
        ):
            errors = validate_freeze(
                Path("dataset"),
                Path("seed"),
                "trial",
                expected_count=3,
                review_workspace=Path("review"),
            )
        self.assertEqual(errors, [])
        dataset_check.assert_called_once_with(Path("dataset"), Path("seed"), 3)
        review_check.assert_called_once_with(Path("review"), Path("seed"), 3)

    def test_full_requires_split_inputs(self) -> None:
        with (
            patch("check_freeze.validate_dataset", return_value=[]),
            patch("check_freeze.validate_auxiliary", return_value=[]),
        ):
            errors = validate_freeze(Path("dataset"), Path("seed"), "full")
        self.assertIn("full: --split, --distractor-split, and --val-count are required", errors)

    def test_full_runs_split_gates_with_private_configuration(self) -> None:
        labels = [{"item_id": "I001"}]
        with (
            patch("check_freeze.validate_dataset", return_value=[]) as dataset_check,
            patch("check_freeze.validate_auxiliary", return_value=[]) as auxiliary_check,
            patch("check_freeze.read_labels", return_value=labels) as labels_reader,
            patch("check_freeze.read_split", return_value=[]) as split_reader,
            patch("check_freeze.verify_split_rows", return_value=[]) as split_check,
            patch("check_freeze.read_distractor_ids", return_value=["D001"]),
            patch("check_freeze.read_distractor_split", return_value=[]) as distractor_reader,
            patch("check_freeze.verify_distractor_split_rows", return_value=[]) as distractor_check,
        ):
            errors = validate_freeze(
                Path("dataset"),
                Path("seed"),
                "full",
                expected_count=2,
                split_path=Path("split.csv"),
                distractor_split_path=Path("distractor-splits.csv"),
                val_count=1,
                expected_distractors=4,
                expected_duplicates=2,
                expected_queries=3,
            )

        self.assertEqual(errors, [])
        dataset_check.assert_called_once_with(Path("dataset"), Path("seed"), 2)
        auxiliary_check.assert_called_once_with(
            Path("dataset"),
            Path("seed"),
            expected_distractors=4,
            expected_duplicates=2,
            expected_queries=3,
        )
        labels_reader.assert_called_once_with(Path("dataset") / "labels.csv", Path("seed"))
        split_reader.assert_called_once_with(Path("split.csv"))
        split_check.assert_called_once_with([], labels, val_count=1)
        distractor_reader.assert_called_once_with(Path("distractor-splits.csv"))
        distractor_check.assert_called_once_with([], ["D001"], 1, 1)

    def test_unknown_profile_is_rejected(self) -> None:
        errors = validate_freeze(Path("dataset"), Path("seed"), "unknown")
        self.assertEqual(errors, ["unknown profile: unknown"])

    def test_trial_rejects_split_arguments(self) -> None:
        with patch("check_freeze.validate_dataset", return_value=[]):
            errors = validate_freeze(
                Path("dataset"),
                Path("seed"),
                "trial",
                review_workspace=Path("review"),
                split_path=Path("split.csv"),
            )
        self.assertIn("trial: split arguments are only valid for the full profile", errors)

    def test_main_writes_freeze_report(self) -> None:
        import json
        import tempfile
        from check_freeze import main

        with tempfile.TemporaryDirectory() as tmp_dir:
            out_file = Path(tmp_dir) / "freeze_report.json"
            with patch("check_freeze.validate_freeze", return_value=[]):
                code = main(["--dataset-dir", "fake_dir", "--profile", "trial", "--output", str(out_file)])
            self.assertEqual(code, 0)
            self.assertTrue(out_file.is_file())
            data = json.loads(out_file.read_text(encoding="utf-8"))
            self.assertEqual(data["status"], "passed")
            self.assertTrue(data["passed"])
            self.assertEqual(data["profile"], "trial")
            self.assertTrue(data["is_synthetic"])


if __name__ == "__main__":
    unittest.main()
