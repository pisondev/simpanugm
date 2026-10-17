import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SQL = (ROOT / "data" / "analytics" / "views.sql").read_text(encoding="utf-8")


class AnalyticsSqlTests(unittest.TestCase):
    def test_defines_all_reference_materialized_views(self) -> None:
        for view in (
            "mv_reports_daily",
            "mv_hour_weekday",
            "mv_return_times",
            "mv_match_outcomes",
        ):
            self.assertRegex(SQL, rf"CREATE MATERIALIZED VIEW IF NOT EXISTS {view}")
        self.assertEqual(len(re.findall(r"CREATE MATERIALIZED VIEW IF NOT EXISTS", SQL)), 4)

    def test_covers_required_sources_and_wib_grouping(self) -> None:
        for table in (
            "lost_reports",
            "lost_report_places",
            "found_reports",
            "locations",
            "zones",
            "categories",
            "status_history",
            "claims",
            "matches",
            "model_versions",
        ):
            self.assertIn(table, SQL)
        self.assertGreaterEqual(SQL.count("Asia/Jakarta"), 3)

    def test_suppression_is_delegated_to_query_layer(self) -> None:
        self.assertNotIn("CASE WHEN", SQL)
        self.assertNotIn("analytics.min_cell_count", SQL)
        self.assertIn("COUNT(*) AS report_count", SQL)

    def test_views_have_concurrent_refresh_indexes(self) -> None:
        for view in (
            "mv_reports_daily",
            "mv_hour_weekday",
            "mv_return_times",
            "mv_match_outcomes",
        ):
            self.assertRegex(SQL, rf"CREATE UNIQUE INDEX IF NOT EXISTS {view}_key")
            self.assertIn(f"REFRESH MATERIALIZED VIEW CONCURRENTLY {view}", SQL)

    def test_private_fields_are_not_referenced(self) -> None:
        for forbidden in (
            "private_details",
            "hidden_details",
            "reporter_id",
            "finder_id",
            "claimant_id",
            "description",
            "statement",
            "answers",
            "internal_note",
        ):
            self.assertNotIn(forbidden, SQL)

    def test_refresh_order_is_documented(self) -> None:
        order = [
            "mv_reports_daily",
            "mv_hour_weekday",
            "mv_return_times",
            "mv_match_outcomes",
        ]
        positions = [SQL.index(f"REFRESH MATERIALIZED VIEW CONCURRENTLY {view}") for view in order]
        self.assertEqual(positions, sorted(positions))


if __name__ == "__main__":
    unittest.main()
