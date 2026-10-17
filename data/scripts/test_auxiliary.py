import csv
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from check_auxiliary import validate_auxiliary
from intake import intake_dataset


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
LABEL_FIELDS = (
    "item_id",
    "leaf_category",
    "primary_colour",
    "secondary_colour",
    "brand",
    "distinctiveness",
    "has_owner_photo",
    "flags",
)


class AuxiliaryCheckTests(unittest.TestCase):
    def create_source(self, directory: str, duplicate_report: bool = False) -> Path:
        source = Path(directory) / "raw"
        (source / "images" / "D001").mkdir(parents=True)
        (source / "images" / "D002").mkdir(parents=True)
        with (source / "labels.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=LABEL_FIELDS)
            writer.writeheader()
            writer.writerows(
                {
                    "item_id": item_id,
                    "leaf_category": leaf,
                    "primary_colour": colour,
                    "secondary_colour": "",
                    "brand": "",
                    "distinctiveness": distinctiveness,
                    "has_owner_photo": "false",
                    "flags": "",
                }
                for item_id, leaf, colour, distinctiveness in (
                    ("I001", "botol-minum", "biru", "generic"),
                    ("I002", "pouch", "hitam", "distinctive"),
                )
            )
        with (source / "distractors.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(
                handle,
                fieldnames=("distractor_id", "leaf_category", "primary_colour", "distinctiveness", "description"),
            )
            writer.writeheader()
            writer.writerows(
                {
                    "distractor_id": distractor_id,
                    "leaf_category": leaf,
                    "primary_colour": colour,
                    "distinctiveness": distinctiveness,
                    "description": description,
                }
                for distractor_id, leaf, colour, distinctiveness, description in (
                    ("D001", "botol-minum", "merah", "generic", "Botol merah polos di atas meja kelas."),
                    ("D002", "pouch", "hitam", "distinctive", "Pouch hitam dengan resleting merah."),
                )
            )
        with (source / "duplicates.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(
                handle,
                fieldnames=("pair_id", "item_id", "first_report_id", "second_report_id"),
            )
            writer.writeheader()
            writer.writerow(
                {
                    "pair_id": "P001",
                    "item_id": "I001",
                    "first_report_id": "FND-001",
                    "second_report_id": "FND-002" if not duplicate_report else "FND-001",
                }
            )
        with (source / "queries.csv").open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=("query_id", "query_text", "target_report_id"))
            writer.writeheader()
            writer.writerows(
                {
                    "query_id": query_id,
                    "query_text": query,
                    "target_report_id": target,
                }
                for query_id, query, target in (
                    ("Q001", "botol merah di kelas", "FND-001"),
                    ("Q002", "pouch resleting merah", "FND-003"),
                )
            )
        for distractor_id in ("D001", "D002"):
            Image.new("RGB", (64, 64), "blue").save(
                source / "images" / distractor_id / "finder_1.jpg"
            )
        return source

    def test_auxiliary_metadata_passes(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-aux")
            self.assertEqual(
                validate_auxiliary(
                    output,
                    SEED_DIR,
                    expected_distractors=2,
                    expected_duplicates=1,
                    expected_queries=2,
                ),
                [],
            )

    def test_auxiliary_check_rejects_reused_report_id(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory, duplicate_report=True)
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-aux")
            errors = validate_auxiliary(
                output,
                SEED_DIR,
                expected_distractors=2,
                expected_duplicates=1,
                expected_queries=2,
            )
            self.assertTrue(any("report ids must be different" in error for error in errors))

    def test_auxiliary_check_rejects_zero_based_record_id(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            distractors = source / "distractors.csv"
            distractors.write_text(
                distractors.read_text(encoding="utf-8").replace("D001", "D000"),
                encoding="utf-8",
            )
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-aux")
            errors = validate_auxiliary(
                output,
                SEED_DIR,
                expected_distractors=2,
                expected_duplicates=1,
                expected_queries=2,
            )
            self.assertTrue(any("distractor_id has invalid value 'D000'" in error for error in errors))

    def test_auxiliary_check_rejects_duplicate_label_id(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            source = self.create_source(directory)
            labels = source / "labels.csv"
            labels.write_text(
                labels.read_text(encoding="utf-8").replace("I002", "I001"),
                encoding="utf-8",
            )
            output = Path(directory) / "prepared"
            intake_dataset(source, output, "trial-aux")
            errors = validate_auxiliary(
                output,
                SEED_DIR,
                expected_distractors=2,
                expected_duplicates=1,
                expected_queries=2,
            )
            self.assertIn("labels.csv item_id values must be unique", errors)


if __name__ == "__main__":
    unittest.main()
