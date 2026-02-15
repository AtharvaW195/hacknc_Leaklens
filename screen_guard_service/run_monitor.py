"""
Run the real-time sensitive screen-share monitor.

Usage examples:
  python3 run_monitor.py --mode manual
  python3 run_monitor.py --mode auto
"""

from __future__ import annotations

import time
from pathlib import Path

from monitor_runtime.alerts import build_alert_callback
from monitor_runtime.cli import build_parser
from monitor_runtime.deps import check_dependencies
from monitor_runtime.recording import RunRecorder

# Try to import HTTP bridge (may not be available in all environments)
try:
    from http_bridge import get_bridge, create_bridge_alert_callback
    HTTP_BRIDGE_AVAILABLE = True
    print("[RUN_MONITOR] HTTP bridge imported successfully")
except ImportError as e:
    HTTP_BRIDGE_AVAILABLE = False
    print(f"[RUN_MONITOR] WARNING: Failed to import http_bridge: {e}")
    print(f"[RUN_MONITOR] Detections will not be sent to Go proxy")
except Exception as e:
    HTTP_BRIDGE_AVAILABLE = False
    print(f"[RUN_MONITOR] ERROR: Unexpected error importing http_bridge: {e}")


def _build_report_path(args, recorder: RunRecorder) -> str:
    if args.report_file:
        return args.report_file
    return str(Path(recorder.context.run_dir) / "session_report.json")


def run_manual(args, recorder: RunRecorder):
    from realtime_monitor import RealTimeScreenShareMonitor

    use_ner = True if args.ensemble_all else args.use_ner
    enable_bert = True if args.ensemble_all else (not args.disable_bert_enrichment)
    alert_callback = build_alert_callback(
        target_os=args.target_os,
        desktop_notifications=args.desktop_notifications,
        recorder_append_fn=recorder.append_detection_batch,
    )
    
    # Wrap with HTTP bridge if available
    if HTTP_BRIDGE_AVAILABLE:
        print("[RUN_MONITOR] HTTP bridge is available, initializing...")
        bridge = get_bridge()
        print(f"[RUN_MONITOR] Bridge URL: {bridge.bridge_url}")
        bridge.send_status("starting", "Video monitoring starting...")
        alert_callback = create_bridge_alert_callback(alert_callback)
        print("[RUN_MONITOR] Bridge alert callback created and set")
    else:
        print("[RUN_MONITOR] WARNING: HTTP bridge not available (http_bridge.py import failed)")
    
    monitor = RealTimeScreenShareMonitor(
        scan_interval=args.scan_interval,
        use_ml=not args.no_ml,
        use_gpu=not args.cpu_only,
        min_confidence=args.min_confidence,
        use_ner=use_ner,
        confirmation_frames=args.confirmation_frames,
        confirmation_window=args.confirmation_window,
        alert_cooldown_seconds=args.alert_cooldown_seconds,
        enable_bert_enrichment=enable_bert,
        adaptive_context_detection=not args.disable_adaptive_context,
        focus_profile=args.focus_profile,
        log_file=recorder.context.run_log_file,
        verbose_logging=not args.quiet,
        alert_callback=alert_callback,
    )

    report_path = _build_report_path(args, recorder)
    print("Starting manual monitoring. Press Ctrl+C to stop.")
    print(f"Run artifacts: {recorder.context.run_dir}")
    
    if HTTP_BRIDGE_AVAILABLE:
        bridge = get_bridge()
        bridge.send_status("running", "Video monitoring started")
    
    monitor.start_monitoring(monitor_number=args.monitor_number)
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nStopping monitor...")
        if HTTP_BRIDGE_AVAILABLE:
            bridge = get_bridge()
            bridge.send_status("stopping", "Video monitoring stopping...")
        monitor.stop_monitoring()
        if HTTP_BRIDGE_AVAILABLE:
            bridge = get_bridge()
            bridge.send_status("stopped", "Video monitoring stopped")
        monitor.export_session_report(report_path)
        recorder.finalize(report_path)


def run_auto(args, recorder: RunRecorder):
    from platform_integration import AutoScreenShareMonitor

    use_ner = True if args.ensemble_all else args.use_ner
    enable_bert = True if args.ensemble_all else (not args.disable_bert_enrichment)
    alert_callback = build_alert_callback(
        target_os=args.target_os,
        desktop_notifications=args.desktop_notifications,
        recorder_append_fn=recorder.append_detection_batch,
    )
    platforms = [p.strip().lower() for p in args.platforms.split(",") if p.strip()]
    auto = AutoScreenShareMonitor(
        scan_interval=args.scan_interval,
        check_interval=args.check_interval,
        use_ml=not args.no_ml,
        use_gpu=not args.cpu_only,
        alert_callback=alert_callback,
        platforms=platforms,
        start_confirmations=args.start_confirmations,
        stop_confirmations=args.stop_confirmations,
        monitor_kwargs={
            "min_confidence": args.min_confidence,
            "use_ner": use_ner,
            "confirmation_frames": args.confirmation_frames,
            "confirmation_window": args.confirmation_window,
            "alert_cooldown_seconds": args.alert_cooldown_seconds,
            "enable_bert_enrichment": enable_bert,
            "adaptive_context_detection": not args.disable_adaptive_context,
            "focus_profile": args.focus_profile,
            "log_file": recorder.context.run_log_file,
            "verbose_logging": not args.quiet,
        },
    )
    print(f"Run artifacts: {recorder.context.run_dir}")
    try:
        auto.run()
    finally:
        report_path = _build_report_path(args, recorder)
        if getattr(auto.monitor, "current_session", None):
            auto.monitor.export_session_report(report_path)
            recorder.finalize(report_path)
        else:
            recorder.finalize(None)


def main():
    parser = build_parser()
    args = parser.parse_args()

    if not check_dependencies(args.mode):
        raise SystemExit(1)

    recorder = RunRecorder(args.output_dir)
    recorder.write_metadata(args)

    if args.mode == "manual":
        run_manual(args, recorder)
    else:
        run_auto(args, recorder)


if __name__ == "__main__":
    main()
