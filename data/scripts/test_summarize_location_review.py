import tempfile
import unittest
from pathlib import Path

from check_location_review import read_review
from osm_locations import review_rows, write_rows
from summarize_location_review import render_summary, summarize_rows, write_summary


class LocationReviewSummaryTests(unittest.TestCase):
    def rows(self) -> list[dict[str, str]]:
        return review_rows(
            {
                "elements": [
                    {
                        "type": "node",
                        "id": 1,
                        "lat": -7.77,
                        "lon": 110.38,
                        "tags": {"name": "Fasilitas Satu", "amenity": "library"},
                    },
                    {
                        "type": "way",
                        "id": 2,
                        "center": {"lat": -7.771, "lon": 110.381},
                        "tags": {"name": "Taman Dua", "leisure": "park"},
                    },
                ]
            }
        )

    def test_summary_counts_without_exposing_names(self) -> None:
        summary = summarize_rows(self.rows())
        self.assertEqual(summary["candidate_count"], 2)
        self.assertEqual(summary["by_kind"], {"facility": 1, "outdoor": 1})
        self.assertEqual(summary["by_osm_type"], {"node": 1, "way": 1})
        self.assertEqual(summary["empty_zone_count"], 2)
        rendered = render_summary(summary)
        self.assertIn("Candidates: **2**", rendered)
        self.assertNotIn("Fasilitas Satu", rendered)

    def test_summary_writes_atomically(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            review = root / "review.csv"
            output = root / "summary.md"
            write_rows(review, self.rows())
            write_summary(output, render_summary(summarize_rows(read_review(review))))
            self.assertIn("Location review summary", output.read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
