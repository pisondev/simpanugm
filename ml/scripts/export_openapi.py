"""Writes ml/openapi.json, the contract between api and ml (matching-engine section 9). Run from ml/."""

import json
import sys
from pathlib import Path

from app.config import Settings
from app.main import create_app

PATH = Path(__file__).resolve().parent.parent / "openapi.json"


def document() -> str:
    app = create_app(Settings(ml_internal_token="export-only-token-0000"))  # noqa: S106
    return json.dumps(app.openapi(), indent=2, sort_keys=True) + "\n"


if __name__ == "__main__":
    text = document()
    if "--check" in sys.argv:
        sys.exit(0 if PATH.read_text(encoding="utf-8") == text else 1)
    PATH.write_text(text, encoding="utf-8", newline="\n")
