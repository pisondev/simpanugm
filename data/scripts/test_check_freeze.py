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


if __name__ == "__main__":
    unittest.main()
