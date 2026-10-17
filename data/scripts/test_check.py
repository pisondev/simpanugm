import unittest
from pathlib import Path

from check import flatten_categories, validate_seed_dir


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


if __name__ == "__main__":
    unittest.main()
