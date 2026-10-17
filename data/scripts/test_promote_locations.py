import csv
import tempfile
import unittest
from pathlib import Path

from promote_locations import LOCATION_FIELDS, promote_rows, write_locations


class LocationPromotionTests(unittest.TestCase):
    BBOX = (-7.80, 110.35, -7.74, 110.41)
    ZONES = {"saintek", "pusat"}

    def approved_rows(self) -> list[dict[str, str]]:
        return [
            {
                "slug": "perpustakaan-pusat",
                "name": "Perpustakaan Pusat",
                "zone_slug": "pusat",
                "kind": "facility",
                "lat": "-7.77012345",
                "lng": "110.37876543",
                "aliases": "perpus; Perpustakaan Pusat; perpus  pusat",
                "osm_type": "way",
                "osm_id": "12",
                "review_status": "approved",
            },
            {
                "slug": "gedung-saintek",
                "name": "Gedung Saintek",
                "zone_slug": "saintek",
                "kind": "building",
                "lat": "-7.771",
                "lng": "110.379",
                "aliases": "",
                "osm_type": "node",
                "osm_id": "13",
                "review_status": "approved",
            },
            {
                "slug": "ditolak",
                "name": "Kandidat Ditolak",
                "zone_slug": "pusat",
                "kind": "building",
                "lat": "-7.771",
                "lng": "110.379",
                "aliases": "",
                "osm_type": "node",
                "osm_id": "14",
                "review_status": "rejected",
            },
        ]

    def test_promotes_only_approved_rows_and_normalizes_aliases(self) -> None:
        promoted, errors = promote_rows(self.approved_rows(), self.ZONES, self.BBOX)
        self.assertEqual(errors, [])
        self.assertEqual([row["slug"] for row in promoted], ["gedung-saintek", "perpustakaan-pusat"])
        self.assertEqual(promoted[1]["aliases"], "perpus;perpus pusat")
        self.assertEqual(tuple(promoted[0]), LOCATION_FIELDS)

    def test_rejects_unreviewed_and_out_of_scope_rows(self) -> None:
        rows = self.approved_rows()
        rows[0]["review_status"] = "needs_review"
        rows[1]["zone_slug"] = "unknown"
        rows[1]["lat"] = "-7.90"
        promoted, errors = promote_rows(rows, self.ZONES, self.BBOX)
        self.assertEqual(promoted, [])
        self.assertTrue(any("review_status must be approved" in error for error in errors))
        self.assertTrue(any("unknown zone_slug" in error for error in errors))
        self.assertTrue(any("outside the campus bbox" in error for error in errors))

    def test_rejects_duplicate_osm_provenance(self) -> None:
        rows = self.approved_rows()
        rows[1]["osm_type"] = "way"
        rows[1]["osm_id"] = "12"
        promoted, errors = promote_rows(rows, self.ZONES, self.BBOX)
        self.assertEqual(promoted, [])
        self.assertTrue(any("duplicate OSM element way/12" in error for error in errors))

    def test_output_has_seed_columns(self) -> None:
        promoted, errors = promote_rows(self.approved_rows(), self.ZONES, self.BBOX)
        self.assertEqual(errors, [])
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "locations.csv"
            write_locations(output, promoted)
            with output.open(newline="", encoding="utf-8") as handle:
                self.assertEqual(tuple(csv.DictReader(handle).fieldnames or ()), LOCATION_FIELDS)


if __name__ == "__main__":
    unittest.main()
