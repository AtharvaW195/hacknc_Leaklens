"""
HTTP Bridge for Video Monitoring Service
Sends events from the video processing server to the Go proxy
"""

import json
import os
import requests
import threading
from typing import Dict, Optional
from datetime import datetime


class VideoMonitorHTTPBridge:
    """Bridge that sends video monitoring events to the Go proxy"""
    
    def __init__(self, bridge_url: Optional[str] = None):
        self.bridge_url = bridge_url or os.getenv(
            "VIDEO_MONITOR_BRIDGE_URL",
            "http://localhost:8080/api/video-monitor/events"
        )
        self.enabled = True
        self._lock = threading.Lock()
        self._session = requests.Session()
        self._session.timeout = 2  # 2 second timeout
        
    def send_event(self, event_type: str, data: Dict) -> bool:
        """Send an event to the Go proxy"""
        if not self.enabled:
            print(f"[HTTP_BRIDGE] Bridge disabled, not sending {event_type}")
            return False
        
        event = {
            "type": event_type,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "data": data
        }
        
        try:
            print(f"[HTTP_BRIDGE] Sending {event_type} event to {self.bridge_url}")
            response = self._session.post(
                self.bridge_url,
                json=event,
                timeout=2
            )
            response.raise_for_status()
            print(f"[HTTP_BRIDGE] Successfully sent {event_type} event (status: {response.status_code})")
            return True
        except Exception as e:
            # Log the error so we can debug
            print(f"[HTTP_BRIDGE] Failed to send {event_type} event: {e}")
            print(f"[HTTP_BRIDGE] Bridge URL: {self.bridge_url}")
            return False
    
    def send_status(self, status: str, message: str):
        """Send a status event"""
        return self.send_event("status", {
            "status": status,
            "message": message
        })
    
    def send_detection(self, detection) -> bool:
        """Send a detection event from a detection object"""
        # Extract detection data
        rule_name = getattr(detection, "rule_name", "unknown")
        severity = getattr(detection, "severity", "medium")
        confidence = float(getattr(detection, "confidence", 0.0))
        matched_text = getattr(detection, "matched_text", "")
        detection_method = getattr(detection, "detection_method", "unknown")
        
        # Redact matched text (show first 4, last 4 chars)
        if len(matched_text) > 8:
            redacted = matched_text[:4] + "..." + matched_text[-4:]
        else:
            redacted = "***"  # Too short to show
        
        return self.send_event("detection", {
            "rule_name": rule_name,
            "severity": severity,
            "confidence": confidence,
            "matched_text": redacted,
            "detection_method": detection_method
        })
    
    def send_log(self, level: str, component: str, message: str):
        """Send a log event"""
        return self.send_event("log", {
            "level": level,
            "component": component,
            "message": message
        })
    
    def send_error(self, code: str, message: str, recoverable: bool = True):
        """Send an error event"""
        return self.send_event("error", {
            "code": code,
            "message": message,
            "recoverable": recoverable
        })


# Global bridge instance
_bridge: Optional[VideoMonitorHTTPBridge] = None
_bridge_lock = threading.Lock()


def get_bridge() -> VideoMonitorHTTPBridge:
    """Get or create the global bridge instance"""
    global _bridge
    with _bridge_lock:
        if _bridge is None:
            _bridge = VideoMonitorHTTPBridge()
        return _bridge


def create_bridge_alert_callback(original_callback=None):
    """Create an alert callback that sends events to the bridge"""
    bridge = get_bridge()
    print(f"[HTTP_BRIDGE] Creating bridge alert callback, bridge URL: {bridge.bridge_url}")
    
    def bridge_alert_callback(results: Dict):
        print(f"[HTTP_BRIDGE] Alert callback invoked, results keys: {list(results.keys())}")
        
        # Call original callback if provided
        if original_callback:
            try:
                original_callback(results)
            except Exception as e:
                print(f"[HTTP_BRIDGE] Error in original alert callback: {e}")
        
        # Send detections to bridge
        detections = results.get("detections", [])
        print(f"[HTTP_BRIDGE] Found {len(detections)} detections")
        
        if detections:
            for i, detection in enumerate(detections):
                print(f"[HTTP_BRIDGE] Sending detection {i+1}/{len(detections)}: {getattr(detection, 'rule_name', 'unknown')}")
                success = bridge.send_detection(detection)
                if not success:
                    print(f"[HTTP_BRIDGE] Failed to send detection {i+1}")
        else:
            print(f"[HTTP_BRIDGE] No detections to send")
    
    return bridge_alert_callback

