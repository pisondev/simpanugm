import json
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path

FLOATING_REVISIONS = {"main", "master", "latest", "head"}


@dataclass(frozen=True)
class Component:
    name: str
    model_id: str
    revision: str
    dimension: int | None = None


@dataclass
class Registry:
    components: dict[str, Component] = field(default_factory=dict)
    loaded_at: dict[str, datetime] = field(default_factory=dict)

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
            components[name] = Component(name, spec["model_id"], revision, spec.get("dimension"))
        return cls(components=components)

    def load_all(self) -> None:
        # encoders arrive with the inference endpoints, the lock already decides what gets loaded
        for name in self.components:
            self.loaded_at[name] = datetime.now(UTC)

    @property
    def ready(self) -> bool:
        return set(self.loaded_at) == set(self.components)
