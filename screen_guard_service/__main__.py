"""
Single-command service launcher.

Run:
  python3 -m screen_guard_service
"""

from __future__ import annotations

import os
import sys
from pathlib import Path

from .service_config import load_runtime_env


def _truthy(value: str) -> bool:
    return str(value).strip().lower() in {"1", "true", "yes", "on"}


def _build_argv(env: dict) -> list[str]:
    mode = env.get("MONITOR_MODE", "auto")
    argv = [
        "run_monitor.py",
        "--mode", mode,
        "--platforms", env.get("MONITOR_PLATFORMS", "zoom,slack"),
        "--scan-interval", env.get("MONITOR_SCAN_INTERVAL", "1.5"),
        "--check-interval", env.get("MONITOR_CHECK_INTERVAL", "3"),
        "--min-confidence", env.get("MONITOR_MIN_CONFIDENCE", "0.8"),
        "--focus-profile", env.get("MONITOR_FOCUS_PROFILE", "pii_full"),
        "--output-dir", env.get("MONITOR_OUTPUT_DIR", "monitor_output"),
        "--target-os", env.get("MONITOR_TARGET_OS", "auto"),
    ]
    if _truthy(env.get("MONITOR_ENSEMBLE_ALL", "true")):
        argv.append("--ensemble-all")
    if _truthy(env.get("MONITOR_ENABLE_DESKTOP_NOTIFICATIONS", "false")):
        argv.append("--desktop-notifications")
    if _truthy(env.get("MONITOR_CPU_ONLY", "false")):
        argv.append("--cpu-only")
    if _truthy(env.get("MONITOR_DISABLE_ADAPTIVE_CONTEXT", "false")):
        argv.append("--disable-adaptive-context")
    if _truthy(env.get("MONITOR_DISABLE_BERT_ENRICHMENT", "false")):
        argv.append("--disable-bert-enrichment")
    return argv


def main() -> None:
    base_dir = Path(__file__).resolve().parent
    os.chdir(base_dir)

    env = load_runtime_env(base_dir)
    argv = _build_argv(env)

    # Preserve explicit CLI overrides if provided after module invocation.
    # Example: python3 -m screen_guard_service --mode manual
    if len(sys.argv) > 1:
        argv.extend(sys.argv[1:])

    sys.argv = argv
    from run_monitor import main as run_main
    run_main()


if __name__ == "__main__":
    main()

