"""Small path-safety helpers for private data workflows."""

from __future__ import annotations

from pathlib import Path


def first_symlink_component(path: Path) -> Path | None:
    """Return the first symlink in ``path`` without resolving through it."""
    absolute = path if path.is_absolute() else Path.cwd() / path
    current = Path(absolute.anchor)
    for part in absolute.parts[1:]:
        current /= part
        if current.is_symlink():
            return current
    return None
