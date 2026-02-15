"""Alert callback composition (console + desktop + recorder)."""

from __future__ import annotations

from typing import Callable, Optional


def build_alert_callback(
    target_os: str,
    desktop_notifications: bool,
    recorder_append_fn: Optional[Callable] = None,
) -> Callable:
    from realtime_monitor import console_alert, desktop_notification_alert

    def callback(results):
        console_alert(results)
        if desktop_notifications:
            desktop_notification_alert(results, target_os=target_os)
        if recorder_append_fn is not None:
            recorder_append_fn(results)

    return callback

