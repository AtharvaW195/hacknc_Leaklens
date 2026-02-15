"""
Real-time Screen Share Content Detector
Monitors screen content and detects sensitive information
"""

import cv2
import numpy as np
import mss
import pytesseract
import re
from typing import List, Dict, Tuple, Optional
from dataclasses import dataclass
from datetime import datetime
import threading
import time
from safe_print import safe_print


@dataclass
class DetectionRule:
    """Defines a rule for detecting sensitive content"""
    name: str
    pattern: str  # regex pattern
    severity: str  # "low", "medium", "high", "critical"
    description: str


@dataclass
class Detection:
    """Represents a detected sensitive item"""
    rule_name: str
    matched_text: str
    severity: str
    timestamp: datetime
    confidence: float
    location: Optional[Tuple[int, int, int, int]] = None  # x, y, width, height


class SensitiveContentDetector:
    """Main detector class for identifying sensitive content on screen"""
    
    def __init__(self):
        self.detection_rules = self._initialize_rules()
        self.is_running = False
        self.detection_callback = None
        self.scan_interval = 1.0  # seconds between scans
        self.last_detections: List[Detection] = []
        
    def _initialize_rules(self) -> List[DetectionRule]:
        """Initialize detection rules for various sensitive data types"""
        return [
            # Credit card numbers (basic Luhn check patterns)
            DetectionRule(
                name="credit_card",
                pattern=r'\b(?:\d{4}[-\s]?){3}\d{4}\b',
                severity="critical",
                description="Credit card number detected"
            ),
            
            # Social Security Numbers (US)
            DetectionRule(
                name="ssn",
                pattern=r'\b\d{3}-\d{2}-\d{4}\b',
                severity="critical",
                description="Social Security Number detected"
            ),
            
            # Email addresses
            DetectionRule(
                name="email",
                pattern=r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b',
                severity="medium",
                description="Email address detected"
            ),
            
            # API keys (generic patterns)
            DetectionRule(
                name="api_key",
                pattern=r'\b[A-Za-z0-9]{32,}\b',
                severity="high",
                description="Potential API key detected"
            ),
            
            # AWS Access Keys
            DetectionRule(
                name="aws_key",
                pattern=r'\b(AKIA[0-9A-Z]{16})\b',
                severity="critical",
                description="AWS Access Key detected"
            ),
            
            # Private/Internal markers
            DetectionRule(
                name="confidential_marker",
                pattern=r'\b(confidential|private|internal only|proprietary)\b',
                severity="high",
                description="Confidential content marker detected"
            ),
            
            # IP addresses (private ranges)
            DetectionRule(
                name="private_ip",
                pattern=r'\b(?:10\.|172\.(?:1[6-9]|2[0-9]|3[01])\.|192\.168\.)\d{1,3}\.\d{1,3}\b',
                severity="medium",
                description="Private IP address detected"
            ),
            
            # Password patterns (when labeled)
            DetectionRule(
                name="password",
                pattern=r'(?i)password\s*[:=]\s*\S+',
                severity="critical",
                description="Password detected"
            ),
        ]
    
    def add_custom_rule(self, rule: DetectionRule):
        """Add a custom detection rule"""
        self.detection_rules.append(rule)
    
    def capture_screen(self, monitor_number: int = 1) -> np.ndarray:
        """Capture the current screen content"""
        with mss.mss() as sct:
            monitor = sct.monitors[monitor_number]
            screenshot = sct.grab(monitor)
            # Convert to numpy array (BGR format for OpenCV)
            img = np.array(screenshot)
            return cv2.cvtColor(img, cv2.COLOR_BGRA2BGR)
    
    def extract_text_from_image(self, image: np.ndarray) -> str:
        """Extract text from screen capture using OCR"""
        # Convert to grayscale for better OCR
        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        
        # Apply some preprocessing for better OCR results
        # Increase contrast
        gray = cv2.convertScaleAbs(gray, alpha=1.5, beta=0)
        
        # Extract text
        text = pytesseract.image_to_string(gray)
        return text
    
    def detect_sensitive_content(self, text: str) -> List[Detection]:
        """Analyze text for sensitive content based on rules"""
        detections = []
        
        for rule in self.detection_rules:
            matches = re.finditer(rule.pattern, text, re.IGNORECASE)
            for match in matches:
                detection = Detection(
                    rule_name=rule.name,
                    matched_text=match.group(0),
                    severity=rule.severity,
                    timestamp=datetime.now(),
                    confidence=0.85  # You can implement more sophisticated confidence scoring
                )
                detections.append(detection)
        
        return detections
    
    def scan_once(self, monitor_number: int = 1) -> List[Detection]:
        """Perform a single scan of the screen"""
        try:
            # Capture screen
            screen_image = self.capture_screen(monitor_number)
            
            # Extract text
            text = self.extract_text_from_image(screen_image)
            
            # Detect sensitive content
            detections = self.detect_sensitive_content(text)
            
            self.last_detections = detections
            
            # Trigger callback if any detections and callback is set
            if detections and self.detection_callback:
                self.detection_callback(detections)
            
            return detections
            
        except Exception as e:
            print(f"Error during scan: {e}")
            return []
    
    def start_monitoring(self, monitor_number: int = 1, callback=None):
        """Start continuous monitoring in background thread"""
        if self.is_running:
            print("Monitoring already running")
            return
        
        self.is_running = True
        self.detection_callback = callback
        
        def monitoring_loop():
            while self.is_running:
                self.scan_once(monitor_number)
                time.sleep(self.scan_interval)
        
        thread = threading.Thread(target=monitoring_loop, daemon=True)
        thread.start()
        print(f"Started monitoring (scan interval: {self.scan_interval}s)")
    
    def stop_monitoring(self):
        """Stop the monitoring thread"""
        self.is_running = False
        print("Stopped monitoring")
    
    def get_detection_summary(self) -> Dict:
        """Get summary of recent detections"""
        if not self.last_detections:
            return {"status": "clean", "count": 0, "detections": []}
        
        severity_counts = {"low": 0, "medium": 0, "high": 0, "critical": 0}
        for detection in self.last_detections:
            severity_counts[detection.severity] += 1
        
        return {
            "status": "sensitive_content_detected",
            "count": len(self.last_detections),
            "severity_breakdown": severity_counts,
            "highest_severity": max(self.last_detections, key=lambda d: 
                ["low", "medium", "high", "critical"].index(d.severity)).severity,
            "detections": self.last_detections
        }


def alert_callback(detections: List[Detection]):
    """Example callback function for handling detections"""
    safe_print("\n" + "="*50)
    safe_print("[WARN]  SENSITIVE CONTENT DETECTED!")
    safe_print("="*50)
    
    for detection in detections:
        severity_icon = {
            "low": "[INFO]",
            "medium": "[WARN]",
            "high": "[CRITICAL]",
            "critical": "[ALERT]"
        }.get(detection.severity, "[WARN]")
        
        safe_print(f"\n{severity_icon} {detection.severity.upper()}: {detection.rule_name}")
        safe_print(f"   Description: {detection.description}")
        safe_print(f"   Matched: {detection.matched_text[:50]}...")  # Truncate for safety
        safe_print(f"   Time: {detection.timestamp.strftime('%H:%M:%S')}")
    
    safe_print("\n" + "="*50 + "\n")


# Example usage and demo
def main_demo():
    """Demonstration of the screen content detector"""
    
    print("Screen Share Content Detector Demo")
    print("=" * 50)
    
    # Initialize detector
    detector = SensitiveContentDetector()
    
    # Add a custom rule
    detector.add_custom_rule(DetectionRule(
        name="phone_number",
        pattern=r'\b\d{3}-\d{3}-\d{4}\b',
        severity="medium",
        description="Phone number detected"
    ))
    
    print(f"\nLoaded {len(detector.detection_rules)} detection rules")
    print("\nRules:")
    for rule in detector.detection_rules:
        print(f"  - {rule.name}: {rule.description} ({rule.severity})")
    
    print("\n" + "="*50)
    print("Starting single scan test...")
    print("="*50 + "\n")
    
    # Perform single scan
    detections = detector.scan_once()
    
    if detections:
        print(f"Found {len(detections)} sensitive items:")
        for det in detections:
            print(f"  - {det.rule_name} ({det.severity})")
    else:
        print("No sensitive content detected in current screen")
    
    # Example of continuous monitoring
    print("\n" + "="*50)
    print("For continuous monitoring, use:")
    print("="*50)
    print("""
    detector.start_monitoring(callback=alert_callback)
    # ... your application runs ...
    detector.stop_monitoring()
    """)


if __name__ == "__main__":
    main_demo()
