import json
from collections.abc import Iterator
from pathlib import Path

import pytest
from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app

TOKEN = "test-token-0123456789abcdef"


@pytest.fixture
def lock(tmp_path: Path) -> Path:
    path = tmp_path / "models.lock.json"
    path.write_text(json.dumps({"version": 1, "components": {}}), encoding="utf-8")
    return path


@pytest.fixture
def client(lock: Path) -> Iterator[TestClient]:
    app = create_app(Settings(ml_internal_token=TOKEN, ml_models_lock=lock))
    with TestClient(app) as c:
        yield c
