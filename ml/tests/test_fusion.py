import json
from datetime import UTC, datetime, timedelta

import pytest

from eval.fusion import (
    NOT_STORED,
    Config,
    Pair,
    brand_relation,
    category_factor,
    colour_relation,
    location_term,
    score,
    time_score,
    time_score_hours,
)
from tests.conftest import FIXTURES

GO_DOCUMENT = FIXTURES.parents[1] / "api" / "internal" / "matching" / "fusion" / "fusion-linear-2026.11.0.json"


def test_the_reference_reads_the_same_document_as_the_go_scoring():
    assert Config.load().raw == json.loads(GO_DOCUMENT.read_text(encoding="utf-8"))


# matching-engine sections 6 and 7 against the fixture the Go tests use, six decimals
def test_reference_agrees_with_the_shared_fixture():
    cfg = Config.load()
    fixture = json.loads((FIXTURES / "fusion_cases.json").read_text(encoding="utf-8"))
    assert fixture["fusion_version"] == cfg.id
    for case in fixture["cases"]:
        visual = case["visual"] or {}
        t = case["time"]
        hours = t.get("hours_after_window", t.get("hours_before_start", 0.0))
        pair = Pair(
            mode=case["mode"],
            visual_cosine=visual.get("image_cosine", visual.get("cross_modal_cosine")),
            semantic_cosine=case["text"]["semantic_cosine"],
            colour=case["text"]["colour"],
            brand=case["text"]["brand"],
            location=None if case["location"] is None else location_term(cfg, case["location"]["distance_m"]),
            time=time_score_hours(cfg, relation=t["relation"], hours=hours),
            category=case["category_factor"],
        )
        got, want = score(cfg, pair), case["expected"]
        for name in ("visual", "semantic", "attributes", "text", "location"):
            expected = want[name]
            actual = getattr(got, name)
            assert (actual is None) == (expected is None), (case["id"], name)
            if expected is not None:
                assert actual == pytest.approx(expected, abs=1e-6), (case["id"], name)
        assert got.time == pytest.approx(want["time"], abs=1e-6), case["id"]
        assert got.fused == pytest.approx(want["fused"], abs=1e-6), case["id"]
        assert got.overall == pytest.approx(want["stored_score"], abs=1e-9), case["id"]
        assert got.tier == want["tier"], case["id"]


def test_missing_signals_leave_the_mean_instead_of_counting_as_zero():
    cfg = Config.load()
    only_time = score(cfg, Pair(time=1.0))
    assert only_time.fused == pytest.approx(1.0)
    assert score(cfg, Pair(time=0.0)).tier == NOT_STORED


def test_the_name_hit_lifts_the_text_signal_to_the_floor():
    cfg = Config.load()
    assert score(cfg, Pair(semantic_cosine=0.8, name_hit=True)).text == pytest.approx(0.95)
    assert score(cfg, Pair(name_hit=True)).text == pytest.approx(0.95)


def test_time_score_from_timestamps():
    cfg = Config.load()
    start = datetime(2026, 11, 2, 1, 0, tzinfo=UTC)
    end = start + timedelta(hours=4)
    assert time_score(cfg, start - timedelta(hours=3), start, end) == 0.0
    assert time_score(cfg, start - timedelta(hours=1), start, end) == 1.0
    assert time_score(cfg, end, start, end) == 1.0
    assert time_score(cfg, end + timedelta(hours=72), start, end) == pytest.approx(0.5)


def test_relations_and_category_factor():
    cfg = Config.load()
    assert colour_relation(["Hitam"], ["hitam", "putih"]) == "shared"
    assert colour_relation(["biru"], ["ungu"]) == "adjacent"
    assert colour_relation(["biru"], ["merah"]) == "different"
    assert colour_relation([], ["merah"]) == "missing"
    assert brand_relation(" Eiger", "eiger") == "same" and brand_relation("", "x") == "missing"
    assert category_factor(cfg, same_leaf=True, same_parent=True, compatible=None) == 1.0
    assert category_factor(cfg, same_leaf=False, same_parent=True, compatible=None) == 0.85
    assert category_factor(cfg, same_leaf=False, same_parent=False, compatible=None) == 0.35
    assert category_factor(cfg, same_leaf=False, same_parent=False, compatible=0.8) == 0.8
    assert location_term(cfg, 1000, zone_only=True) < 0.1 and location_term(cfg, 0, zone_only=True) == 0.6
