"""CLI parsing for monitor runtime."""

from __future__ import annotations

import argparse


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Real-time screen-share sensitive data monitor")
    p.add_argument("--mode", choices=["manual", "auto"], default="manual")
    p.add_argument("--scan-interval", type=float, default=1.5)
    p.add_argument("--check-interval", type=float, default=5.0, help="Only for auto mode")
    p.add_argument("--platforms", default="zoom,slack", help="Comma-separated platforms for auto mode")
    p.add_argument("--start-confirmations", type=int, default=2, help="Auto mode: consecutive positives before start")
    p.add_argument("--stop-confirmations", type=int, default=2, help="Auto mode: consecutive negatives before stop")
    p.add_argument("--monitor-number", type=int, default=1)
    p.add_argument("--min-confidence", type=float, default=0.7)
    p.add_argument("--confirmation-frames", type=int, default=2)
    p.add_argument("--confirmation-window", type=int, default=3)
    p.add_argument("--alert-cooldown-seconds", type=int, default=12)
    p.add_argument("--report-file", default=None)
    p.add_argument("--use-ner", action="store_true", help="Enable NER model in primary scanner")
    p.add_argument("--ensemble-all", action="store_true",
                   help="Force full ensemble: regex + primary NER + BERT enrichment")
    p.add_argument("--focus-profile", choices=["all", "pii_core", "pii_full"], default="pii_full",
                   help="Detection scope profile (pii_full covers broader PII/key material)")
    p.add_argument("--disable-bert-enrichment", action="store_true")
    p.add_argument("--disable-adaptive-context", action="store_true",
                   help="Disable contextual NLP sensitive-token detection")
    p.add_argument("--no-ml", action="store_true")
    p.add_argument("--cpu-only", action="store_true")
    p.add_argument("--quiet", action="store_true")
    p.add_argument("--target-os", choices=["auto", "mac", "windows", "linux"], default="auto",
                   help="Desktop notification OS target")
    p.add_argument("--desktop-notifications", action="store_true",
                   help="Enable desktop notifications in addition to console alerts")
    p.add_argument("--output-dir", default="monitor_output",
                   help="Directory for runtime logs and recorded data")
    return p
