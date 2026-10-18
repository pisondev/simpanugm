import csv
import tempfile
import unittest
from pathlib import Path

from check_agreement import (
    AgreementError,
    cohen_kappa,
    read_annotations,
    render_report,
    summarize_agreement,
    write_report,
)


ROOT = Path(__file__).resolve().parents[2]
SEED_DIR = ROOT / "data" / "seed"
FIELDS = ("item_id", "leaf_category", "distinctiveness")


class AgreementTests(unittest.TestCase):
    def write_sheet(self, directory: str, name: str, rows: list[dict[str, str]]) -> Path:
        path = Path(directory) / name
        with path.open("w", newline="", encoding="utf-8") as handle:
            writer = csv.DictWriter(handle, fieldnames=FIELDS)
            writer.writeheader()
            writer.writerows(rows)
        return path

    def rows(self, flipped: bool = False) -> list[dict[str, str]]:
        values = (
            ("botol-minum", "generic"),
            ("tumbler", "distinctive"),
            ("payung", "generic"),
            ("dompet", "distinctive"),
        )
        if flipped:
            values = tuple((leaf, "distinctive" if value == "generic" else "generic") for leaf, value in values)
        return [
            {"item_id": f"I{index:03d}", "leaf_category": leaf, "distinctiveness": distinctiveness}
            for index, (leaf, distinctiveness) in enumerate(values, start=1)
        ]

    def test_perfect_agreement_has_kappa_one(self) -> None:
        first = {row["item_id"]: row for row in self.rows()}
        second = {row["item_id"]: row for row in self.rows()}
        summary = summarize_agreement(first, second)
        self.assertEqual(summary["record_count"], 4)
        self.assertEqual(summary["metrics"]["leaf_category"]["cohen_kappa"], 1.0)
        self.assertEqual(summary["metrics"]["distinctiveness"]["cohen_kappa"], 1.0)

    def test_kappa_reports_known_disagreement(self) -> None:
        observed, kappa = cohen_kappa(["a", "a", "b", "b"], ["a", "b", "a", "b"])
        self.assertEqual(observed, 0.5)
        self.assertEqual(kappa, 0.0)

    def test_report_contains_aggregate_values_only(self) -> None:
        first = {row["item_id"]: row for row in self.rows()}
        report = render_report(summarize_agreement(first, first))
        self.assertIn("Records: **4**", report)
        self.assertIn("1.0000", report)
        self.assertNotIn("I001", report)

    def test_reader_rejects_unknown_category(self) -> None:
        rows = self.rows()
        rows[0]["leaf_category"] = "not-a-category"
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_sheet(directory, "first.csv", rows)
            with self.assertRaises(AgreementError):
                read_annotations(path, SEED_DIR, expected_count=4)

    def test_reader_requires_matching_ids(self) -> None:
        rows = self.rows()
        rows[0]["item_id"] = "I005"
        with tempfile.TemporaryDirectory() as directory:
            path = self.write_sheet(directory, "first.csv", rows)
            with self.assertRaises(AgreementError):
                read_annotations(path, SEED_DIR, expected_count=4)

    def test_reader_rejects_short_and_extra_rows(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            short_path = Path(directory) / "short.csv"
            short_path.write_text(
                "item_id,leaf_category,distinctiveness\nI001,botol-minum\n",
                encoding="utf-8",
            )
            extra_path = Path(directory) / "extra.csv"
            extra_path.write_text(
                "item_id,leaf_category,distinctiveness\nI001,botol-minum,generic,unexpected\n",
                encoding="utf-8",
            )
            with self.assertRaisesRegex(AgreementError, "row 2 has missing or extra fields"):
                read_annotations(short_path, SEED_DIR, expected_count=1)
            with self.assertRaisesRegex(AgreementError, "row 2 has missing or extra fields"):
                read_annotations(extra_path, SEED_DIR, expected_count=1)

    def test_sheets_must_have_same_ids(self) -> None:
        first = {row["item_id"]: row for row in self.rows()}
        second = {row["item_id"]: row for row in self.rows()[:-1]}
        with self.assertRaises(AgreementError):
            summarize_agreement(first, second)

    def test_report_writes_atomically(self) -> None:
        first = {row["item_id"]: row for row in self.rows()}
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "agreement.md"
            write_report(output, render_report(summarize_agreement(first, first)))
            self.assertIn("Label agreement report", output.read_text(encoding="utf-8"))

    def test_report_writer_rejects_symlinked_parent(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            real_parent = root / "real-report"
            real_parent.mkdir()
            alias_parent = root / "report-alias"
            try:
                alias_parent.symlink_to(real_parent, target_is_directory=True)
            except OSError as exc:
                self.skipTest(f"symlink is unavailable: {exc}")

            with self.assertRaises(OSError):
                write_report(alias_parent / "agreement.md", "report\n")


if __name__ == "__main__":
    unittest.main()
