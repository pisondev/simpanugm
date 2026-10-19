import importlib.util
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def load():
    spec = importlib.util.spec_from_file_location("export_openapi", ROOT / "scripts" / "export_openapi.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_committed_openapi_matches_the_service():
    # run `PYTHONPATH=. uv run python scripts/export_openapi.py` after changing an endpoint
    assert (ROOT / "openapi.json").read_text(encoding="utf-8") == load().document()


def test_the_contract_lists_the_endpoints_the_api_calls():
    paths = json.loads((ROOT / "openapi.json").read_text(encoding="utf-8"))["paths"]
    for path in (
        "/v1/models",
        "/v1/embed/image",
        "/v1/embed/text",
        "/v1/analyze/image",
        "/v1/cluster",
        "/healthz",
        "/readyz",
    ):
        assert path in paths
