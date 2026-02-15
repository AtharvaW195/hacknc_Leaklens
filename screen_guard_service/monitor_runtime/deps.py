"""Dependency checks with mode-specific requirements."""

from __future__ import annotations

import importlib.util
from typing import List


def _missing(packages: List[str]) -> List[str]:
    return [pkg for pkg in packages if importlib.util.find_spec(pkg) is None]


def check_dependencies(mode: str) -> bool:
    required = ["mss", "numpy"]
    if mode == "auto":
        required.append("psutil")
    optional = ["cv2", "easyocr", "transformers", "torch", "pytesseract"]

    missing_required = _missing(required)
    missing_optional = _missing(optional)

    if missing_required:
        print("Missing required packages:")
        for pkg in missing_required:
            print(f"  - {pkg}")
        print("\nInstall dependencies first: pip install -r requirements.txt")
        return False

    if missing_optional:
        print("Optional packages not installed (system will still run with reduced accuracy/features):")
        for pkg in missing_optional:
            print(f"  - {pkg}")
        print()

    return True

