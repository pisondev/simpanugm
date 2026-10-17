import re
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SQL = (ROOT / "data" / "analytics" / "views.sql").read_text(encoding="utf-8")


class AnalyticsSqlTests(unittest.TestCase):
    def test_defines_all_reference_materialized_views(self) -> None:
        for view in (
            "analytics_report_events",
            "analytics_location_daily",
            "analytics_weekly_time",
            "analytics_weekly_outcomes",
        ):
            self.assertRegex(SQL, rf"CREATE MATERIALIZED VIEW IF NOT EXISTS {view}")
        self.assertEqual(len(re.findall(r"CREATE MATERIALIZED VIEW IF NOT EXISTS", SQL)), 4)

    def test_covers_required_sources_and_wib_grouping(self) -> None:
        for table in (
            "lost_reports",
            "lost_report_places",
            "found_reports",
            "locations",
            "categories",
            "status_history",
        ):
            self.assertIn(table, SQL)
        self.assertGreaterEqual(SQL.count("Asia/Jakarta"), 3)

    def test_suppression_is_explicit_for_public_aggregates(self) -> None:
        public_views = SQL.split("CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_report_events", 1)[1]
        self.assertGreaterEqual(len(re.findall(r"CASE WHEN [^\n]+ >= 3 THEN", public_views)), 7)
        self.assertIn("analytics.min_cell_count", SQL)
        self.assertIn("ELSE NULL END AS", SQL)

    def test_private_fields_are_not_referenced(self) -> None:
        for forbidden in ("private_details", "hidden_details", "reporter_id", "finder_id", "description"):
            self.assertNotIn(forbidden, SQL)

    def test_refresh_order_is_documented(self) -> None:
        order = [
            "analytics_report_events",
            "analytics_location_daily",
            "analytics_weekly_time",
            "analytics_weekly_outcomes",
        ]
        positions = [SQL.index(f"REFRESH MATERIALIZED VIEW {view}") for view in order]
        self.assertEqual(positions, sorted(positions))


if __name__ == "__main__":
    unittest.main()
