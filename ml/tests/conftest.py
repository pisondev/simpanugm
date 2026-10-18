import io
import json
import time
from collections.abc import Callable, Iterator
from pathlib import Path

import pytest
from fastapi.testclient import TestClient
from PIL import Image

from app.backend import OcrLine
from app.config import Settings
from app.main import create_app
from app.registry import Registry
from tests.fakes import FakeBackend

TOKEN = "test-token-0123456789abcdef"
AUTH = {"Authorization": f"Bearer {TOKEN}"}
FIXTURES = Path(__file__).resolve().parents[2] / "data" / "fixtures"

COMPONENTS = {
    "image_encoder": {"model_id": "test/image", "revision": "image-rev-0001", "dimension": 512},
    "xmodal_text_encoder": {"model_id": "test/xmodal", "revision": "xmodal-rev-0001", "dimension": 512},
    "text_encoder": {"model_id": "test/text", "revision": "text-rev-0001", "dimension": 384},
    "ocr": {"model_id": "test/ocr", "revision": "ocr-rev-0001"},
}


def write_lock(path: Path, components: dict[str, dict[str, object]]) -> Path:
    path.write_text(json.dumps({"version": 1, "components": components}), encoding="utf-8")
    return path


@pytest.fixture
def lock(tmp_path: Path) -> Path:
    return write_lock(tmp_path / "models.lock.json", {})


def wait_ready(client: TestClient, timeout: float = 5.0) -> None:
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        if client.get("/readyz").status_code == 200:
            return
        time.sleep(0.02)
    raise AssertionError("service never became ready")


@pytest.fixture
def make_client(tmp_path: Path) -> Iterator[Callable[..., TestClient]]:
    opened: list[TestClient] = []

    def make(
        components: dict[str, dict[str, object]] | None = None, lines: list[OcrLine] | None = None, **settings: object
    ) -> TestClient:
        lock_path = write_lock(tmp_path / "models.lock.json", COMPONENTS if components is None else components)
        config = Settings(ml_internal_token=TOKEN, ml_models_lock=lock_path, **settings)  # type: ignore[arg-type]
        backend = FakeBackend(Registry.from_lock(lock_path), lines)
        client = TestClient(create_app(config, backend))
        client.__enter__()
        opened.append(client)
        client.app.state.backend = backend  # type: ignore[attr-defined]
        wait_ready(client)
        return client

    yield make
    for c in opened:
        c.__exit__(None, None, None)


@pytest.fixture
def client(make_client: Callable[..., TestClient]) -> TestClient:
    return make_client()


def png(colour: tuple[int, int, int] = (30, 60, 200), size: tuple[int, int] = (64, 64), fmt: str = "PNG") -> bytes:
    buffer = io.BytesIO()
    Image.new("RGB", size, colour).save(buffer, format=fmt)
    return buffer.getvalue()
