import csv
import tempfile
import unittest
from pathlib import Path

from split_eval import (
    make_distractor_split_rows,
    make_split_rows,
    read_distractor_ids,
    read_labels,
    verify_distractor_split_rows,
    verify_split_rows,
    write_split,
)


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"


class EvaluationSplitTests(unittest.TestCase):
    def labels_file(self, directory: str) -> Path:
        path = Path(directory) / "labels.csv"
        with path.open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(
                handle,
                fieldnames=("item_id", "leaf_category", "distinctiveness"),
            )
            writer.writeheader()
            rows = (
                ("I001", "botol-minum", "generic"),
                ("I002", "botol-minum", "distinctive"),
                ("I003", "tumbler", "generic"),
                ("I004", "tumbler", "distinctive"),
                ("I005", "payung", "generic"),
                ("I006", "payung", "distinctive"),
                ("I007", "dompet", "generic"),
                ("I008", "dompet", "distinctive"),
                ("I009", "pouch", "generic"),
                ("I010", "pouch", "distinctive"),
            )
            writer.writerows(
                {
                    "item_id": item_id,
                    "leaf_category": leaf,
                    "distinctiveness": distinctiveness,
                }
                for item_id, leaf, distinctiveness in rows
            )
        return path

    def test_split_is_deterministic_stratified_and_disjoint(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            labels_path = self.labels_file(directory)
            labels = read_labels(labels_path, SEED_DIR)
            first = make_split_rows(labels, val_count=4, seed=2026)
            second = make_split_rows(labels, val_count=4, seed=2026)
            self.assertEqual(first, second)
            self.assertEqual(sum(row["split"] == "val" for row in first), 4)
            self.assertEqual(sum(row["split"] == "test" for row in first), 6)
            val_ids = {row["item_id"] for row in first if row["split"] == "val"}
            test_ids = {row["item_id"] for row in first if row["split"] == "test"}
            self.assertTrue(val_ids.isdisjoint(test_ids))
            self.assertEqual(verify_split_rows(first, labels, val_count=4), [])

    def test_written_split_can_be_verified(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            labels_path = self.labels_file(directory)
            labels = read_labels(labels_path, SEED_DIR)
            output = Path(directory) / "splits.csv"
            rows = make_split_rows(labels, val_count=4, seed=2026)
            write_split(output, rows)
            with output.open(newline="", encoding="utf-8") as handle:
                saved_rows = list(csv.DictReader(handle))
            self.assertEqual(verify_split_rows(saved_rows, labels, 4), [])

    def test_verifier_rejects_duplicate_assignment(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            labels_path = self.labels_file(directory)
            labels = read_labels(labels_path, SEED_DIR)
            rows = make_split_rows(labels, val_count=4, seed=2026)
            rows.append(dict(rows[0]))
            errors = verify_split_rows(rows, labels, val_count=4)
            self.assertTrue(any("duplicate split assignment" in error for error in errors))

    def test_labels_reject_non_contract_item_id(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "labels.csv"
            with path.open("w", newline="", encoding="utf-8") as handle:
                writer = csv.DictWriter(
                    handle,
                    fieldnames=("item_id", "leaf_category", "distinctiveness"),
                )
                writer.writeheader()
                writer.writerow(
                    {
                        "item_id": "I000",
                        "leaf_category": "botol-minum",
                        "distinctiveness": "generic",
                    }
                )
            with self.assertRaises(ValueError):
                read_labels(path, SEED_DIR)

    def test_distractors_reject_zero_id(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "distractors.csv"
            with path.open("w", newline="", encoding="utf-8") as handle:
                writer = csv.DictWriter(handle, fieldnames=("distractor_id",))
                writer.writeheader()
                writer.writerow({"distractor_id": "D000"})
            with self.assertRaises(ValueError):
                read_distractor_ids(path)

    def test_distractors_follow_the_item_split_proportion(self) -> None:
        distractors = [f"D{index:03d}" for index in range(1, 11)]
        first = make_distractor_split_rows(distractors, item_val_count=4, item_count=10, seed=2026)
        second = make_distractor_split_rows(distractors, item_val_count=4, item_count=10, seed=2026)
        self.assertEqual(first, second)
        self.assertEqual(sum(row["split"] == "val" for row in first), 4)
        self.assertEqual(
            verify_distractor_split_rows(first, distractors, item_val_count=4, item_count=10),
            [],
        )


if __name__ == "__main__":
    unittest.main()
