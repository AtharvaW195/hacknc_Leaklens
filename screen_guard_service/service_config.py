"""Service-level config loading from .env and process environment."""

from __future__ import annotations

import os
from pathlib import Path
from typing import Dict


def _load_env_file(env_path: Path) -> Dict[str, str]:
    data: Dict[str, str] = {}
    if not env_path.exists():
        return data
    for raw in env_path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        data[key.strip()] = value.strip().strip('"').strip("'")
    return data


def load_runtime_env(base_dir: Path) -> Dict[str, str]:
    merged = {}
    merged.update(_load_env_file(base_dir / ".env"))
    merged.update(os.environ)
    return merged

