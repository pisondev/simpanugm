import json
from pathlib import Path

import pytest

from app.registry import Registry
from eval import run as harness
from eval.dataset import leaf_parents, leakage, load
from eval.metrics import bootstrap_ci, ece, mrr, ndcg_at_10, recall_at_k, tier_precision, tier_recall
from eval.text import TextInput, clip_text, e5_text
from tests.conftest import COMPONENTS, write_lock
from tests.fakes import FakeBackend
from tests.synth import SEED, build


# evaluation.md section 12: metric functions against hand computed toy rankings
def test_metrics_on_a_toy_ranking():
    ranks = [1, 2, 5, None, 11]
    assert recall_at_k(ranks, 1) == pytest.approx(0.2)
    assert recall_at_k(ranks, 5) == pytest.approx(0.6)
    assert recall_at_k(ranks, 10) == pytest.approx(0.6)
    assert mrr(ranks) == pytest.approx((1 + 0.5 + 0.2 + 1 / 11) / 5)
    assert ndcg_at_10(ranks) == pytest.approx((1 + 1 / 1.5849625 + 1 / 2.5849625) / 5, rel=1e-5)
    assert recall_at_k([], 5) == 0.0


def test_tier_metrics_and_calibration_error():
    tiers = ["high", "high", "medium", "low"]
    truth = [True, False, True, False]
    assert tier_precision(tiers, truth, "high") == 0.5
    assert tier_precision(tiers, truth, "medium") == 1.0
    assert tier_precision(["low"], [True], "high") is None
    assert tier_recall(tiers, truth, "medium", n_true=2) == 1.0
    assert tier_recall(tiers, truth, "high", n_true=2) == 0.5
    assert ece([0.9, 0.9, 0.1, 0.1], [True, True, False, False]) == pytest.approx(0.1)


def test_bootstrap_is_seeded():
    ranks = [1, 2, None, 4, 1, 3, 7, 1]
    first = bootstrap_ci(ranks, lambda r: recall_at_k(r, 5))
    assert first == bootstrap_ci(ranks, lambda r: recall_at_k(r, 5))
    assert first[0] <= recall_at_k(ranks, 5) <= first[1]


# the Go test of api/internal/matching/text_test.go pins the same strings
def test_matching_texts_agree_with_the_go_twin():
    syn = {"hp": "ponsel handphone", "casan": "charger"}
    got = e5_text(
        TextInput(
            "dompet kulit",
            "Dompet",
            "Dompet dan Tas Kecil",
            ["cokelat"],
            "",
            "Dompet lipat dua, isi KTM dan kartu ATM.",
            "Ada gantungan kunci kecil berbentuk kucing.",
        ),
        syn,
    )
    assert got == (
        "query: dompet kulit. Dompet (Dompet dan Tas Kecil). Warna: cokelat. Merek: tidak ada. "
        "Dompet lipat dua, isi KTM dan kartu ATM. Ada gantungan kunci kecil berbentuk kucing."
    )
    masked = e5_text(
        TextInput("HP  hitam", "Ponsel", "Elektronik", [], "", "casan ikut, nomor kartu 4111 1111 1111 1111"), syn
    )
    assert "4111" not in masked and "  " not in masked and masked.endswith("1111 handphone charger")
    long = clip_text(TextInput("Tumbler biru", "Tumbler", "", ["biru"], "Corkcicle", "panjang " * 200, "rahasia"), syn)
    assert len(long) <= 400 and "rahasia" not in long and long.startswith("Tumbler biru. Tumbler. biru. Corkcicle.")


def test_the_leakage_check_catches_a_photo_shared_between_splits(tmp_path):
    assert leakage(load(build(tmp_path / "clean"), SEED)) == []
    problems = leakage(load(build(tmp_path / "leaky", leak=True), SEED))
    assert problems and "image" in problems[0]


def harness_run(tmp_path: Path, name: str, run_id: str = "r") -> Path:
    lock = write_lock(tmp_path / "lock.json", COMPONENTS)
    return harness.run(
        build(tmp_path / f"data-{name}"),
        "val",
        tmp_path / "runs",
        backend=FakeBackend(Registry.from_lock(lock)),
        lock=lock,
        allow_dirty=True,
        run_id=run_id,
    )


def test_a_run_writes_its_artefacts_and_is_deterministic(tmp_path):
    first = harness_run(tmp_path, "a", "first")
    second = harness_run(tmp_path, "b", "second")
    for name in ("metrics.json", "per_query.csv", "slices.csv", "config.yaml", "environment.json"):
        assert (first / name).is_file()
    assert (first / "metrics.json").read_text() == (second / "metrics.json").read_text()
    assert (first / "per_query.csv").read_text() == (second / "per_query.csv").read_text()
    metrics = json.loads((first / "metrics.json").read_text())
    assert {"B0", "B1", "B2", "B3", "A1", "A2", "A3", "A4", "A5", "A6a", "A7"} <= set(metrics)
    assert metrics["B3"]["overall"]["n"] == 10
    assert 0 <= metrics["B3"]["overall"]["recall@5"] <= 1
    assert "lost_photo=with" in metrics["B3"]["slices"] and "lost_photo=without" in metrics["B3"]["slices"]
    assert metrics["gates"] and all({"gate", "passed"} <= set(g) for g in metrics["gates"])
    assert (
        json.loads((first / "environment.json").read_text())["models"]["image_encoder"]["revision"] == "image-rev-0001"
    )


def test_a_run_refuses_to_overwrite_a_recorded_run(tmp_path):
    harness_run(tmp_path, "c")
    with pytest.raises(FileExistsError):
        harness_run(tmp_path, "d")


def test_the_seed_categories_are_known():
    assert leaf_parents(SEED)["tumbler"]


def test_calibration_writes_a_new_fusion_document_and_never_reuses_an_id(tmp_path):
    lock = write_lock(tmp_path / "lock.json", COMPONENTS)
    configs = tmp_path / "configs"
    configs.mkdir()
    kwargs = {
        "backend": FakeBackend(Registry.from_lock(lock)),
        "lock": lock,
        "allow_dirty": True,
        "config_dir": configs,
        "calibrate_as": "fusion-linear-2026.11.1",
        "components_too": False,
        "only": ["B3"],
    }
    harness.run(build(tmp_path / "data"), "val", tmp_path / "runs", run_id="cal", **kwargs)
    doc = json.loads((configs / "fusion-linear-2026.11.1.json").read_text())
    assert doc["id"] == "fusion-linear-2026.11.1" and doc["kind"] == "weighted_mean"
    w = doc["weights"]
    assert abs(w["visual_image"] + w["text"] + w["location"] + w["time"] - 1.0) < 1e-6
    assert doc["tiers"]["floor"] < doc["tiers"]["medium"] < doc["tiers"]["high"]
    with pytest.raises(FileExistsError):
        harness.run(build(tmp_path / "data2"), "val", tmp_path / "runs", run_id="cal2", **kwargs)


def test_calibration_refuses_the_test_split(tmp_path):
    lock = write_lock(tmp_path / "lock.json", COMPONENTS)
    with pytest.raises(SystemExit, match="val only"):
        harness.run(
            build(tmp_path / "data"),
            "test",
            tmp_path / "runs",
            backend=FakeBackend(Registry.from_lock(lock)),
            lock=lock,
            allow_dirty=True,
            calibrate_as="fusion-x",
            config_dir=tmp_path,
        )


def test_learned_fusion_experiment(tmp_path):
    from collections import defaultdict

    from eval import scenario, text
    from eval.dataset import leaf_parents, load
    from eval.engine import Embedder, Engine, load_compat
    from eval.fusion import Config
    from eval.learned_fusion import run_experiment
    from tests.fakes import FakeBackend

    root = Path(__file__).resolve().parents[2]
    seed_dir = root / "data" / "seed"
    cfg = Config.load()
    lock_path = root / "ml" / "models.lock.json"
    registry = Registry.from_lock(lock_path)
    backend = FakeBackend(registry)

    dataset_path = build(tmp_path / "data", items=10, distractors=6)
    dataset = load(dataset_path, seed_dir)
    parents = leaf_parents(seed_dir)
    locations = scenario.load_locations(root / "deploy" / "dev-seed" / "locations.csv")
    synonyms = text.load_synonyms(seed_dir / "synonyms.csv")
    compat_pairs = load_compat(seed_dir / "category_compatibility.csv")
    compatible: dict[str, list[str]] = defaultdict(list)
    for key in compat_pairs:
        a, b = key.split("|")
        compatible[a].append(b)
        compatible[b].append(a)

    items = {e.id: e.leaf for e in dataset.entries.values() if e.is_item}
    distractors = [e.id for e in dataset.entries.values() if not e.is_item]
    generated = scenario.Generator(locations, parents, compatible).generate(items, distractors)
    identity = "|".join(f"{n}={c.model_id}@{c.revision}" for n, c in sorted(registry.components.items()))
    embedder = Embedder(backend, identity, None)
    engine = Engine(dataset, generated, {loc.slug: loc for loc in locations}, embedder, parents, compat_pairs, synonyms)
    queries = engine.build("val", cfg)
    found_leaf = {e.id: e.leaf for e in dataset.split("val")}

    res = run_experiment(queries, cfg, parents.__getitem__, found_leaf)
    assert "recall@5" in res.b3_metrics and "recall@5" in res.b4_metrics
    assert "ece" in res.b4_metrics
    assert len(res.coefficients) == 13
    assert "recommendation" in res.gate_decision
