import io
import json

from fastapi.testclient import TestClient
from PIL import Image

from app.backend import OcrLine
from app.config import Settings
from app.main import create_app
from app.registry import Registry
from tests.conftest import AUTH, COMPONENTS, TOKEN, png, wait_ready, write_lock
from tests.fakes import FakeBackend

# matching-engine section 9: the three inference endpoints and their error codes


def noise() -> bytes:
    import numpy as np

    buffer = io.BytesIO()
    pixels = np.random.default_rng(1).integers(0, 255, (64, 64, 3), dtype=np.uint8)
    Image.fromarray(pixels).save(buffer, format="PNG")
    return buffer.getvalue()


def files(*blobs: bytes, name: str = "files"):
    return [(name, (f"i{n}.png", blob, "image/png")) for n, blob in enumerate(blobs)]


def test_inference_needs_the_token(client):
    assert client.post("/v1/embed/text", json={"texts": ["a"], "kinds": ["e5"]}).status_code == 401
    assert client.post("/v1/embed/image", files=files(png())).status_code == 401
    assert client.post("/v1/analyze/image", files=files(png(), name="file")).status_code == 401


def test_embed_image_returns_one_unit_vector_per_file_with_the_model_echo(client):
    res = client.post("/v1/embed/image", files=files(png(), png((200, 30, 30))), headers=AUTH)
    assert res.status_code == 200
    body = res.json()
    assert (
        body["component"] == "image_encoder" and body["model"] == "test/image" and body["revision"] == "image-rev-0001"
    )
    assert len(body["embeddings"]) == 2 and len(body["embeddings"][0]) == 512
    assert abs(sum(x * x for x in body["embeddings"][0]) - 1) < 1e-4
    assert body["embeddings"][0] != body["embeddings"][1]


def test_embed_image_accepts_jpeg_and_webp(client):
    for fmt in ("JPEG", "WEBP"):
        assert client.post("/v1/embed/image", files=files(png(fmt=fmt)), headers=AUTH).status_code == 200


def test_embed_image_error_codes(make_client):
    client = make_client(ml_max_image_bytes=2000)
    codes = []
    for blobs in ([b"not an image"], [noise()], [png()] * 9):
        res = client.post("/v1/embed/image", files=files(*blobs), headers=AUTH)
        assert res.headers["content-type"] == "application/problem+json"
        codes.append((res.status_code, res.json()["code"]))
    assert codes == [(400, "invalid_image"), (413, "too_large"), (413, "too_many_items")]
    gif = io.BytesIO()
    Image.new("RGB", (8, 8)).save(gif, format="GIF")
    assert client.post("/v1/embed/image", files=files(gif.getvalue()), headers=AUTH).json()["code"] == "invalid_image"


def test_embed_image_is_not_ready_without_its_component(make_client):
    client = make_client(components={k: v for k, v in COMPONENTS.items() if k != "image_encoder"})
    res = client.post("/v1/embed/image", files=files(png()), headers=AUTH)
    assert res.status_code == 503 and res.json()["code"] == "not_ready"


def test_embed_text_answers_one_block_per_kind(client):
    res = client.post(
        "/v1/embed/text", json={"texts": ["query: dompet", "dompet kulit"], "kinds": ["e5", "clip_text"]}, headers=AUTH
    )
    assert res.status_code == 200
    body = res.json()
    assert body["e5"]["component"] == "text_encoder" and len(body["e5"]["embeddings"][0]) == 384
    assert body["clip_text"]["component"] == "xmodal_text_encoder" and len(body["clip_text"]["embeddings"][1]) == 512
    assert body["e5"]["revision"] == "text-rev-0001"
    only = client.post("/v1/embed/text", json={"texts": ["a"], "kinds": ["clip_text"]}, headers=AUTH).json()
    assert set(only) == {"clip_text"}


def test_embed_text_limits_and_validation(client):
    too_many = client.post("/v1/embed/text", json={"texts": ["a"] * 33, "kinds": ["e5"]}, headers=AUTH)
    assert too_many.status_code == 413 and too_many.json()["code"] == "too_many_items"
    assert (
        client.post("/v1/embed/text", json={"texts": [], "kinds": ["e5"]}, headers=AUTH).json()["code"]
        == "invalid_request"
    )
    assert client.post("/v1/embed/text", json={"texts": ["a"], "kinds": ["bert"]}, headers=AUTH).status_code == 422


def analyze(client, categories=None, **extra):
    data = {"categories": json.dumps(categories if categories is not None else []), "prompt_set_hash": "set-1", **extra}
    return client.post("/v1/analyze/image", files=files(png((37, 99, 235)), name="file"), data=data, headers=AUTH)


def test_analyze_returns_every_signal_and_the_versions(make_client):
    lines = [OcrLine("Eiger", 0.9), OcrLine("KTM 4532 0151 1283 0366", 0.95)]
    client = make_client(lines=lines)
    cats = [{"slug": "tumbler", "prompts": ["a tumbler", "a bottle"]}, {"slug": "payung", "prompts": ["an umbrella"]}]
    res = analyze(client, cats)
    assert res.status_code == 200
    body = res.json()
    assert len(body["categories"]) == 2 and sum(c["score"] for c in body["categories"]) <= 1.0001
    assert body["colours"][0]["name"] == "biru" and body["colours"][0]["hex"].startswith("#")
    assert (
        body["text"]["masked"] is True
        and "4532" not in body["text"]["value"]
        and body["text"]["value"].endswith("0366")
    )
    assert [b["name"] for b in body["brands"]] == ["Eiger"]
    assert body["versions"]["image_encoder"] == "test/image@image-rev-0001"
    assert body["versions"]["colour_extractor"] and body["versions"]["brand_matcher"]


def test_analyze_without_categories_or_text(client):
    body = analyze(client).json()
    assert body["categories"] == [] and body["brands"] == [] and body["text"] == {"value": "", "masked": False}


def test_analyze_caches_prompt_embeddings_by_hash(client):
    cats = [{"slug": "tumbler", "prompts": ["a tumbler"]}]
    backend = client.app.state.backend
    analyze(client, cats)
    analyze(client, cats)
    assert backend.prompt_calls == 1
    analyze(client, cats, prompt_set_hash="set-2")
    assert backend.prompt_calls == 2


def test_analyze_rejects_malformed_input(client):
    res = client.post("/v1/analyze/image", files=files(png(), name="file"), data={"categories": "{nope"}, headers=AUTH)
    assert res.status_code == 422 and res.json()["code"] == "invalid_request"
    res = client.post("/v1/analyze/image", files=files(b"x", name="file"), data={"categories": "[]"}, headers=AUTH)
    assert res.json()["code"] == "invalid_image"


def test_a_failed_load_never_reports_ready(tmp_path):
    lock = write_lock(tmp_path / "models.lock.json", COMPONENTS)
    backend = FakeBackend(Registry.from_lock(lock))
    backend.fail_load = True
    with TestClient(create_app(Settings(ml_internal_token=TOKEN, ml_models_lock=lock), backend)) as c:
        try:
            wait_ready(c, timeout=0.5)
        except AssertionError:
            pass
        else:
            raise AssertionError("a service whose models failed to load reported ready")
        assert c.get("/healthz").status_code == 200
        assert (
            c.post("/v1/embed/text", json={"texts": ["a"], "kinds": ["e5"]}, headers=AUTH).json()["code"] == "not_ready"
        )
