import json
import threading
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path

FLOATING_REVISIONS = {"main", "master", "latest", "head"}

# the components the api registers as model versions, in the order of model-lifecycle section 1
IMAGE_ENCODER = "image_encoder"
XMODAL_TEXT_ENCODER = "xmodal_text_encoder"
TEXT_ENCODER = "text_encoder"
OCR = "ocr"


@dataclass(frozen=True)
class Component:
    name: str
    model_id: str
    revision: str
    dimension: int | None = None
    # file name to SHA-256 for weights that do not come from a revisioned repository
    sha256: dict[str, str] = field(default_factory=dict)


@dataclass
class Registry:
    components: dict[str, Component] = field(default_factory=dict)
    loaded_at: dict[str, datetime] = field(default_factory=dict)
    _lock: threading.Lock = field(default_factory=threading.Lock, repr=False)

    @classmethod
    def from_lock(cls, path: Path) -> "Registry":
        raw = json.loads(path.read_text(encoding="utf-8"))
        if raw.get("version") != 1:
            raise ValueError(f"{path}: unsupported lock version {raw.get('version')!r}")
        components = {}
        for name, spec in raw.get("components", {}).items():
            revision = spec["revision"]
            if len(revision) < 7 or revision.lower() in FLOATING_REVISIONS:
                raise ValueError(f"{path}: component {name} must pin an exact revision, NFR-REPRO-02")
            components[name] = Component(
                name, spec["model_id"], revision, spec.get("dimension"), spec.get("sha256", {})
            )
        return cls(components=components)

    def mark_loaded(self, name: str) -> None:
        with self._lock:
            self.loaded_at[name] = datetime.now(UTC)

    def load_all(self) -> None:
        """Marks every component loaded when no backend does the loading, as in tests of the HTTP surface."""
        for name in self.components:
            self.mark_loaded(name)

    @property
    def ready(self) -> bool:
        return set(self.loaded_at) == set(self.components)

    def get(self, name: str) -> Component:
        return self.components[name]
