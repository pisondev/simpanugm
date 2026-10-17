import json
from pathlib import Path

import pytest

from app.registry import Registry
from tests.conftest import TOKEN


def test_health_and_readiness(client):
    assert client.get("/healthz").json() == {"status": "ok"}
    ready = client.get("/readyz")
    assert ready.status_code == 200
    assert ready.json()["status"] == "ready"


def test_models_need_the_internal_token(client):
    assert client.get("/v1/models").status_code == 401
    assert client.get("/v1/models", headers={"Authorization": "Bearer wrong-token-value"}).status_code == 401
    res = client.get("/v1/models", headers={"Authorization": f"Bearer {TOKEN}"})
    assert res.status_code == 200
    assert res.json() == {"items": []}


def test_metrics_are_exposed(client):
    client.get("/healthz")
    assert "http_requests_total" in client.get("/metrics").text


def test_lock_requires_exact_revisions(tmp_path: Path):
    path = tmp_path / "models.lock.json"
    spec = {"model_id": "sentence-transformers/clip-ViT-B-32", "revision": "main"}
    path.write_text(json.dumps({"version": 1, "components": {"image_encoder": spec}}), encoding="utf-8")
    with pytest.raises(ValueError, match="exact revision"):
        Registry.from_lock(path)


def test_registry_reports_loaded_components(tmp_path: Path):
    path = tmp_path / "models.lock.json"
    spec = {"model_id": "sentence-transformers/clip-ViT-B-32", "revision": "0123456789abcdef", "dimension": 512}
    path.write_text(json.dumps({"version": 1, "components": {"image_encoder": spec}}), encoding="utf-8")
    registry = Registry.from_lock(path)
    assert not registry.ready
    registry.load_all()
    assert registry.ready
    assert registry.components["image_encoder"].dimension == 512


def test_repository_lock_is_valid():
    Registry.from_lock(Path(__file__).resolve().parent.parent / "models.lock.json")
