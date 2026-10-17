import csv
import tempfile
import unittest
from pathlib import Path

from create_eval_templates import TemplateError, create_templates


class EvaluationTemplateTests(unittest.TestCase):
    def test_creates_expected_files_ids_and_counts(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "raw"
            created = create_templates(
                output,
                item_count=3,
                distractor_count=4,
                duplicate_count=2,
                query_count=5,
            )
            self.assertEqual(len(created), 5)
            self.assertTrue((output / "images").is_dir())
            expectations = {
                "items.csv": ("item_id", 3, "I003"),
                "labels.csv": ("item_id", 3, "I003"),
                "distractors.csv": ("distractor_id", 4, "D004"),
                "duplicates.csv": ("pair_id", 2, "P002"),
                "queries.csv": ("query_id", 5, "Q005"),
            }
            for filename, (id_field, count, last_id) in expectations.items():
                with (output / filename).open(newline="", encoding="utf-8") as handle:
                    rows = list(csv.DictReader(handle))
                self.assertEqual(len(rows), count)
                self.assertEqual(rows[-1][id_field], last_id)
                self.assertTrue(all(value == "" for key, value in rows[0].items() if key != id_field))

    def test_refuses_non_empty_output(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "raw"
            output.mkdir()
            (output / "keep.txt").write_text("do not overwrite", encoding="utf-8")
            with self.assertRaises(TemplateError):
                create_templates(output, item_count=1, distractor_count=1, duplicate_count=1, query_count=1)


if __name__ == "__main__":
    unittest.main()
