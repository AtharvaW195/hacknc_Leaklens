"""Structured run recording for observability and audits."""

from __future__ import annotations

import json
import os
import platform
import uuid
from dataclasses import asdict, dataclass
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, Optional


@dataclass
class RunContext:
    run_id: str
    run_dir: str
    run_log_file: str
    detections_file: str
    metadata_file: str


class RunRecorder:
    def __init__(self, output_dir: str):
        self.output_dir = Path(output_dir)
        self.context = self._init_context()
        self._metadata: Dict[str, Any] = {}

    def _init_context(self) -> RunContext:
        ts = datetime.now().strftime("%Y%m%d_%H%M%S")
        run_id = f"run_{ts}_{uuid.uuid4().hex[:8]}"
        run_dir = self.output_dir / "runs" / run_id
        run_dir.mkdir(parents=True, exist_ok=True)
        return RunContext(
            run_id=run_id,
            run_dir=str(run_dir),
            run_log_file=str(run_dir / "runtime.log"),
            detections_file=str(run_dir / "detections.jsonl"),
            metadata_file=str(run_dir / "run_metadata.json"),
        )

    def write_metadata(self, args: Any) -> None:
        self._metadata = {
            "run_id": self.context.run_id,
            "created_at": datetime.now().isoformat(),
            "host": {
                "platform": platform.platform(),
                "system": platform.system(),
                "release": platform.release(),
                "python_version": platform.python_version(),
                "node": platform.node(),
            },
            "args": vars(args),
        }
        with open(self.context.metadata_file, "w", encoding="utf-8") as f:
            json.dump(self._metadata, f, indent=2)

    def append_detection_batch(self, results: Dict[str, Any]) -> None:
        detections = results.get("detections", [])
        if not detections:
            return
        event = {
            "timestamp": datetime.now().isoformat(),
            "total": len(detections),
            "scan_time": results.get("scan_time"),
            "ocr_confidence": results.get("ocr_confidence"),
            "detections": [
                {
                    "rule_name": getattr(d, "rule_name", "unknown"),
                    "severity": getattr(d, "severity", "unknown"),
                    "confidence": float(getattr(d, "confidence", 0.0)),
                    "detection_method": getattr(d, "detection_method", "unknown"),
                    "matched_text": getattr(d, "matched_text", "")[:120],
                    "context": getattr(d, "context", "")[:240],
                }
                for d in detections
            ],
        }
        with open(self.context.detections_file, "a", encoding="utf-8") as f:
            f.write(json.dumps(event, default=str) + os.linesep)

    def finalize(self, report_path: Optional[str]) -> None:
        if not self._metadata:
            return
        self._metadata["completed_at"] = datetime.now().isoformat()
        self._metadata["session_report_file"] = report_path
        with open(self.context.metadata_file, "w", encoding="utf-8") as f:
            json.dump(self._metadata, f, indent=2)

