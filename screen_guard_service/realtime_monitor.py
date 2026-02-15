"""
Real-Time Screen Share Monitor with ML Scanner
Continuously monitors screen during screen sharing sessions
"""

try:
    import cv2
    CV2_AVAILABLE = True
except ImportError:
    cv2 = None
    CV2_AVAILABLE = False
import numpy as np
import mss
import time
import threading
from typing import List, Dict, Callable, Optional
from dataclasses import dataclass
from datetime import datetime
from collections import deque
import json
import logging
from monitor_runtime.pii_catalog import is_rule_in_scope

RULE_LABELS = {
    "credit_card": "Credit Card Number",
    "credit_card_suspected": "Potential Credit Card Number",
    "ssn": "Social Security Number",
    "email": "Email Address",
    "phone": "Phone Number",
    "private_ip": "Private IP Address",
    "iban": "IBAN",
    "swift_bic": "SWIFT/BIC",
    "openai_key": "OpenAI API Key",
    "aws_access_key": "AWS Access Key",
    "github_token": "GitHub Token",
    "google_api_key": "Google API Key",
    "slack_token": "Slack Token",
    "jwt_token": "JWT Token",
    "api_key_assignment": "API Key Assignment",
    "secret_assignment": "Secret Assignment",
    "token_assignment": "Auth Token Assignment",
    "password_assignment": "Visible Password",
    "password_field_html": "Visible Password Field",
    "private_key_pem": "Private Key (PEM)",
    "pgp_private_key_block": "PGP Private Key",
    "public_key_pem": "Public Key (PEM)",
    "ssh_public_key": "SSH Public Key",
    "private_key_assignment": "Private Key Assignment",
    "public_key_assignment": "Public Key Assignment",
    "contextual_sensitive_token": "Contextual Sensitive Token",
}


def _reason_for_detection(detection) -> str:
    rule = (getattr(detection, "rule_name", "") or "").lower()
    method = (getattr(detection, "detection_method", "") or "").lower()
    if rule in {"credit_card", "credit_card_suspected"}:
        return "Financial pattern matched (card-like sequence)."
    if "password" in rule:
        return "Password-related context with visible non-masked value."
    if "private_key" in rule or "secret" in rule or "token" in rule or "api_key" in rule:
        return "Secret/key pattern detected."
    if method.startswith("contextual"):
        return "Token matched sensitive context + token-quality signals."
    if method.startswith("ner"):
        return "NER model classified PII entity."
    return "Pattern or model matched sensitive data signal."


def _print_readable_alert(detections: List, title_prefix: str = "ALERT"):
    if not detections:
        return
    severity_order = {"critical": 0, "high": 1, "medium": 2, "low": 3}
    grouped = {"critical": [], "high": [], "medium": [], "low": []}
    for d in detections:
        sev = (getattr(d, "severity", "medium") or "medium").lower()
        grouped.setdefault(sev, []).append(d)
    total = len(detections)
    print(f"\n\n🚨 {title_prefix} {datetime.now().strftime('%H:%M:%S')} | {total} sensitive item(s)")
    for sev in ["critical", "high", "medium", "low"]:
        items = grouped.get(sev, [])
        if not items:
            continue
        print(f"\n[{sev.upper()}] {len(items)} item(s)")
        items = sorted(items, key=lambda d: -float(getattr(d, "confidence", 0.0)))
        for i, d in enumerate(items[:8], 1):
            rule = getattr(d, "rule_name", "unknown")
            label = RULE_LABELS.get(rule, rule.replace("_", " ").title())
            conf = float(getattr(d, "confidence", 0.0))
            method = getattr(d, "detection_method", "unknown")
            matched = getattr(d, "matched_text", "")[:72]
            reason = _reason_for_detection(d)
            print(f"  {i}. {label} | conf={conf:.2f} | via={method}")
            print(f"     evidence: '{matched}'")
            print(f"     why: {reason}")
    print()

# Import scanners (prefer fixed scanner for better real-time precision)
FIXED_ML_AVAILABLE = False
MODERN_ML_AVAILABLE = False

try:
    from fixed_ml_scanner import FixedMLScanner
    FIXED_ML_AVAILABLE = True
except ImportError:
    pass

try:
    from modern_ml_scanner import ModernMLScanner
    MODERN_ML_AVAILABLE = True
except ImportError:
    pass

TRADITIONAL_AVAILABLE = False
try:
    from screen_content_detector import SensitiveContentDetector
    TRADITIONAL_AVAILABLE = True
except ImportError:
    SensitiveContentDetector = None

if not FIXED_ML_AVAILABLE and not MODERN_ML_AVAILABLE:
    print("⚠️  ML scanners not available")


@dataclass
class ScreenShareSession:
    """Track a screen sharing session"""
    session_id: str
    start_time: datetime
    end_time: Optional[datetime] = None
    total_scans: int = 0
    total_detections: int = 0
    critical_detections: int = 0
    high_detections: int = 0
    detections: List = None
    
    def __post_init__(self):
        if self.detections is None:
            self.detections = []


class RealTimeScreenShareMonitor:
    """
    Real-time monitor for screen sharing with ML detection
    
    Features:
    - Continuous screen scanning
    - Smart frame skipping (don't scan identical frames)
    - Alert callbacks
    - Session tracking
    - Performance optimization
    """
    
    def __init__(self,
                 scan_interval: float = 2.0,
                 use_ml: bool = True,
                 use_gpu: bool = True,
                 alert_callback: Optional[Callable] = None,
                 min_confidence: float = 0.6,
                 use_ner: bool = False,
                 confirmation_frames: int = 2,
                 confirmation_window: int = 3,
                 alert_cooldown_seconds: int = 12,
                 enable_bert_enrichment: bool = True,
                 adaptive_context_detection: bool = True,
                 focus_profile: str = "pii_full",
                 log_file: str = "monitor_events.log",
                 verbose_logging: bool = True):
        """
        Initialize real-time monitor
        
        Args:
            scan_interval: Seconds between scans
            use_ml: Use ML scanner (recommended)
            use_gpu: Use GPU acceleration
            alert_callback: Function to call when detection found
            min_confidence: Minimum detection confidence
            use_ner: Enable NER model detections (higher recall, lower precision)
            confirmation_frames: Required repeated hits before alerting non-critical items
            confirmation_window: Number of scans for confirmation window
            alert_cooldown_seconds: Cooldown to suppress duplicate repeated alerts
            enable_bert_enrichment: Use ModernMLScanner NER as secondary corroboration
            adaptive_context_detection: Context-aware NLP detector in primary scanner
            focus_profile: "pii_full" covers broader PII + key material
            log_file: Log file path for runtime events
            verbose_logging: Emit scan-level logs in console
        """
        
        self.scan_interval = scan_interval
        self.use_ml = use_ml
        self.min_confidence = min_confidence
        self.alert_callback = alert_callback
        self.use_ner = use_ner
        self.confirmation_frames = max(1, confirmation_frames)
        self.confirmation_window = max(1, confirmation_window)
        self.alert_cooldown_seconds = max(0, alert_cooldown_seconds)
        self.enable_bert_enrichment = enable_bert_enrichment
        self.adaptive_context_detection = adaptive_context_detection
        self.focus_profile = (focus_profile or "all").lower()
        self.log_file = log_file
        self.verbose_logging = verbose_logging
        self.logger = self._build_logger()
        
        # Initialize scanner
        print(f"🔧 Initializing monitor...")
        print(f"   Scan interval: {scan_interval}s")
        print(f"   ML Scanner: {'Enabled' if use_ml else 'Disabled'}")
        print(f"   GPU: {'Enabled' if use_gpu else 'Disabled'}")
        
        if use_ml and FIXED_ML_AVAILABLE:
            self.scanner = FixedMLScanner(
                use_gpu=use_gpu,
                use_ner=use_ner,
                adaptive_context=adaptive_context_detection
            )
            self.scanner_type = 'fixed_ml'
        elif use_ml and MODERN_ML_AVAILABLE:
            self.scanner = ModernMLScanner(
                use_gpu=use_gpu,
                model_size='medium'  # Balance speed/accuracy
            )
            self.scanner_type = 'modern_ml'
        else:
            if not TRADITIONAL_AVAILABLE:
                raise RuntimeError(
                    "No scanner backend available. Install required dependencies "
                    "(opencv-python, pytesseract/easyocr, transformers optional)."
                )
            self.scanner = SensitiveContentDetector()
            self.scanner_type = 'traditional'

        # Optional secondary scanner for BERT enrichment/corroboration.
        self.secondary_scanner = None
        if (
            self.enable_bert_enrichment
            and self.scanner_type == 'fixed_ml'
            and MODERN_ML_AVAILABLE
        ):
            try:
                self.secondary_scanner = ModernMLScanner(
                    use_gpu=use_gpu,
                    model_size='small'
                )
                print("   ✓ BERT enrichment enabled (ModernMLScanner)")
            except Exception as e:
                print(f"   ⚠️  BERT enrichment disabled: {e}")
                self.secondary_scanner = None

        if self.alert_callback is None:
            self.alert_callback = self._default_console_alert
        
        # Monitoring state
        self.is_monitoring = False
        self.monitor_thread = None
        self.current_session = None
        
        # Performance optimization
        self.last_frame_hash = None
        self.frame_skip_count = 0
        self.consecutive_identical_frames = 0
        
        # Detection history (last 100 detections)
        self.detection_history = deque(maxlen=100)
        self.pending_confirmation: Dict[str, Dict] = {}
        self.recent_alerts: Dict[str, datetime] = {}
        
        # Statistics
        self.stats = {
            'total_scans': 0,
            'frames_skipped': 0,
            'total_detections': 0,
            'avg_scan_time': 0.0,
            'scan_times': deque(maxlen=50),
            'filtered_unconfirmed': 0,
            'suppressed_duplicates': 0
        }
        
        print("✅ Monitor initialized!")
        self._log_event("monitor_initialized", {
            "scanner_type": self.scanner_type,
            "secondary_scanner": self.secondary_scanner is not None,
            "scan_interval": self.scan_interval,
            "min_confidence": self.min_confidence,
            "adaptive_context_detection": self.adaptive_context_detection,
            "focus_profile": self.focus_profile
        })

    def _build_logger(self) -> logging.Logger:
        logger = logging.getLogger(f"realtime_monitor_{id(self)}")
        logger.setLevel(logging.INFO)
        logger.propagate = False
        if logger.handlers:
            return logger
        stream_handler = logging.StreamHandler()
        stream_handler.setFormatter(logging.Formatter("%(asctime)s | %(levelname)s | %(message)s"))
        logger.addHandler(stream_handler)
        if self.log_file:
            file_handler = logging.FileHandler(self.log_file)
            file_handler.setFormatter(logging.Formatter("%(message)s"))
            logger.addHandler(file_handler)
        return logger

    def _log_event(self, event: str, payload: Dict, level: str = "info"):
        entry = {
            "timestamp": datetime.now().isoformat(),
            "event": event,
            **payload
        }
        msg = json.dumps(entry, default=str)
        if level == "error":
            self.logger.error(msg)
        elif level == "warning":
            self.logger.warning(msg)
        else:
            self.logger.info(msg)
    
    def capture_screen(self, monitor_number: int = 1) -> np.ndarray:
        """Capture current screen"""
        with mss.mss() as sct:
            if monitor_number < 1 or monitor_number >= len(sct.monitors):
                raise ValueError(f"Invalid monitor_number={monitor_number}. Available: 1..{len(sct.monitors)-1}")
            monitor = sct.monitors[monitor_number]
            screenshot = sct.grab(monitor)
            img = np.array(screenshot)
            if CV2_AVAILABLE:
                return cv2.cvtColor(img, cv2.COLOR_BGRA2BGR)
            # BGRA -> BGR fallback without OpenCV
            return img[:, :, :3].copy()
    
    def compute_frame_hash(self, frame: np.ndarray) -> int:
        """Compute hash of frame for change detection"""
        # Downsample for faster comparison
        if CV2_AVAILABLE:
            small = cv2.resize(frame, (100, 100))
        else:
            h, w = frame.shape[:2]
            step_h = max(1, h // 100)
            step_w = max(1, w // 100)
            small = frame[::step_h, ::step_w][:100, :100]
        return hash(small.tobytes())
    
    def has_frame_changed(self, frame: np.ndarray, threshold: int = 3) -> bool:
        """
        Check if frame has changed significantly since last scan
        Avoids scanning identical frames
        """
        current_hash = self.compute_frame_hash(frame)
        
        if self.last_frame_hash is None:
            self.last_frame_hash = current_hash
            return True
        
        if current_hash == self.last_frame_hash:
            self.consecutive_identical_frames += 1

            # For static screens, rescan periodically so confirmation logic can trigger.
            if self.consecutive_identical_frames % max(1, threshold) == 0:
                return True
            return False
        else:
            self.consecutive_identical_frames = 0
            self.last_frame_hash = current_hash
            return True
    
    def scan_frame(self, frame: np.ndarray) -> Dict:
        """Scan a single frame for sensitive content"""
        start_time = time.time()
        
        if self.scanner_type == 'modern_ml':
            results = self.scanner.scan_image(
                frame,
                use_ner=self.use_ner,
                use_regex=True,
                min_confidence=self.min_confidence
            )
        elif self.scanner_type == 'fixed_ml':
            results = self.scanner.scan_image(
                frame,
                min_confidence=self.min_confidence
            )
        else:
            # Traditional scanner
            text = self.scanner.extract_text_from_image(frame)
            detections = self.scanner.detect_sensitive_content(text)
            results = {
                'detections': detections,
                'total_detections': len(detections),
                'extracted_text': text,
                'processing_time': 0
            }
        
        scan_time = time.time() - start_time
        results['scan_time'] = scan_time

        if self.secondary_scanner is not None:
            secondary = self._scan_with_secondary(frame, primary_results=results)
            results = self._merge_primary_secondary(results, secondary)
        
        # Update stats
        self.stats['scan_times'].append(scan_time)
        self.stats['avg_scan_time'] = np.mean(self.stats['scan_times'])

        if self.verbose_logging:
            self._log_event("scan_completed", {
                "scanner_type": self.scanner_type,
                "scan_time": round(scan_time, 4),
                "ocr_confidence": round(float(results.get('ocr_confidence', 0.0)), 4),
                "detections": int(results.get('total_detections', 0))
            })
        
        return results

    def _scan_with_secondary(self, frame: np.ndarray, primary_results: Optional[Dict] = None) -> Dict:
        try:
            extracted = (primary_results or {}).get("extracted_text", "")
            if extracted and hasattr(self.secondary_scanner, "scan_text_only"):
                return self.secondary_scanner.scan_text_only(extracted)
            return self.secondary_scanner.scan_image(
                frame,
                use_ner=True,
                use_regex=False,
                min_confidence=max(self.min_confidence, 0.70)
            )
        except Exception as e:
            self._log_event("secondary_scan_failed", {"error": str(e)}, level="warning")
            return {"detections": [], "total_detections": 0}

    def _merge_primary_secondary(self, primary: Dict, secondary: Dict) -> Dict:
        primary_detections = list(primary.get('detections', []))
        secondary_detections = list(secondary.get('detections', []))
        if not secondary_detections:
            return primary

        merged: Dict[str, object] = {}
        corroborated_keys = set()

        for d in primary_detections:
            merged[self._detection_key(d)] = d

        for d in secondary_detections:
            key = self._detection_key(d)
            if key in merged:
                base = merged[key]
                boosted = min(0.99, max(float(getattr(base, 'confidence', 0.0)), float(getattr(d, 'confidence', 0.0))) + 0.05)
                setattr(base, 'confidence', boosted)
                setattr(base, 'detection_method', f"{getattr(base, 'detection_method', 'regex')}+bert")
                corroborated_keys.add(key)
                merged[key] = base
                continue

            sev_rank = self._severity_rank(getattr(d, 'severity', 'low'))
            conf = float(getattr(d, 'confidence', 0.0))
            # Add BERT-only detections only if high-signal.
            if sev_rank >= self._severity_rank('high') or conf >= 0.92:
                setattr(d, 'detection_method', f"{getattr(d, 'detection_method', 'ner')}_bert")
                merged[key] = d

        primary['detections'] = list(merged.values())
        primary['total_detections'] = len(primary['detections'])
        methods = dict(primary.get('methods_used', {}))
        methods['bert_enrichment'] = True
        methods['corroborated_hits'] = len(corroborated_keys)
        primary['methods_used'] = methods
        return primary

    def _severity_rank(self, severity: str) -> int:
        return {'critical': 4, 'high': 3, 'medium': 2, 'low': 1}.get((severity or '').lower(), 1)

    def _detection_key(self, detection) -> str:
        rule = getattr(detection, 'rule_name', 'unknown')
        text = getattr(detection, 'matched_text', '')
        normalized = "".join(ch.lower() for ch in text if ch.isalnum())
        return f"{rule}:{normalized[:64]}"

    def _requires_confirmation(self, detection) -> bool:
        severity = (getattr(detection, 'severity', 'medium') or 'medium').lower()
        confidence = float(getattr(detection, 'confidence', 0.0))
        # Escalate immediately only for high-confidence critical findings.
        if severity == 'critical' and confidence >= 0.90:
            return False
        return self.confirmation_frames > 1

    def _within_cooldown(self, key: str, now: datetime) -> bool:
        last_alert = self.recent_alerts.get(key)
        if not last_alert:
            return False
        return (now - last_alert).total_seconds() < self.alert_cooldown_seconds

    def _is_detection_in_scope(self, detection) -> bool:
        rule = (getattr(detection, "rule_name", "") or "").lower()
        return is_rule_in_scope(rule, self.focus_profile)

    def _confirm_and_filter_detections(self, detections: List, results: Dict) -> List:
        confirmed = []
        now = datetime.now()
        current_scan = max(1, self.stats['total_scans'])

        ocr_confidence = float(results.get('ocr_confidence', 1.0))
        # When OCR is weak, suppress non-critical/noisy findings.
        weak_ocr = ocr_confidence < 0.45

        for detection in detections:
            severity = (getattr(detection, 'severity', 'medium') or 'medium').lower()
            confidence = float(getattr(detection, 'confidence', 0.0))
            key = self._detection_key(detection)

            if not self._is_detection_in_scope(detection):
                self.stats['filtered_unconfirmed'] += 1
                continue

            if weak_ocr and self._severity_rank(severity) < self._severity_rank('high') and confidence < 0.90:
                self.stats['filtered_unconfirmed'] += 1
                continue

            if self._requires_confirmation(detection):
                state = self.pending_confirmation.get(key)
                if not state or (current_scan - state['last_seen_scan']) > self.confirmation_window:
                    self.pending_confirmation[key] = {
                        'count': 1,
                        'last_seen_scan': current_scan,
                        'detection': detection,
                    }
                    self.stats['filtered_unconfirmed'] += 1
                    continue

                state['count'] += 1
                state['last_seen_scan'] = current_scan

                if confidence > float(getattr(state['detection'], 'confidence', 0.0)):
                    state['detection'] = detection

                if state['count'] < self.confirmation_frames:
                    self.stats['filtered_unconfirmed'] += 1
                    continue

                detection = state['detection']

            if self._within_cooldown(key, now):
                self.stats['suppressed_duplicates'] += 1
                continue

            self.recent_alerts[key] = now
            confirmed.append(detection)

        # Keep memory bounded
        stale_keys = [
            k for k, v in self.pending_confirmation.items()
            if (current_scan - v['last_seen_scan']) > (self.confirmation_window * 2)
        ]
        for k in stale_keys:
            del self.pending_confirmation[k]

        return confirmed
    
    def handle_detections(self, results: Dict):
        """Handle detected sensitive content"""
        detections = self._confirm_and_filter_detections(results['detections'], results)
        results['detections'] = detections
        results['total_detections'] = len(detections)
        
        if not detections:
            return
        
        # Add to session
        if self.current_session:
            self.current_session.total_detections += len(detections)
            
            for detection in detections:
                severity = getattr(detection, 'severity', 'medium')
                if severity == 'critical':
                    self.current_session.critical_detections += 1
                elif severity == 'high':
                    self.current_session.high_detections += 1
                
                self.current_session.detections.append({
                    'timestamp': datetime.now().isoformat(),
                    'rule_name': getattr(detection, 'rule_name', 'unknown'),
                    'severity': severity,
                    'confidence': getattr(detection, 'confidence', 0.0),
                    'matched_text': getattr(detection, 'matched_text', '')[:50]
                })
        
        # Add to global stats
        self.stats['total_detections'] += len(detections)
        
        # Add to history
        self.detection_history.extend(detections)

        self._log_event("detections_confirmed", {
            "count": len(detections),
            "scanner_type": self.scanner_type,
            "items": [
                {
                    "rule": getattr(d, 'rule_name', 'unknown'),
                    "severity": getattr(d, 'severity', 'unknown'),
                    "confidence": round(float(getattr(d, 'confidence', 0.0)), 4),
                    "method": getattr(d, 'detection_method', 'unknown'),
                    "matched_text": getattr(d, 'matched_text', '')[:60]
                }
                for d in detections[:10]
            ]
        })
        
        # Call alert callback
        if self.alert_callback:
            try:
                self.alert_callback(results)
            except Exception as e:
                print(f"Error in alert callback: {e}")
                self._log_event("alert_callback_error", {"error": str(e)}, level="error")
    
    def monitoring_loop(self, monitor_number: int = 1):
        """Main monitoring loop - runs in background thread"""
        print(f"\n🔍 Starting monitoring loop...")
        print(f"   Scanning every {self.scan_interval}s")
        print(f"   Press Ctrl+C to stop\n")
        
        while self.is_monitoring:
            try:
                # Capture screen
                frame = self.capture_screen(monitor_number)
                
                # Check if frame has changed
                if not self.has_frame_changed(frame):
                    self.stats['frames_skipped'] += 1
                    time.sleep(self.scan_interval)
                    continue
                
                # Scan frame
                results = self.scan_frame(frame)
                
                # Update session stats
                if self.current_session:
                    self.current_session.total_scans += 1
                
                self.stats['total_scans'] += 1
                
                # Handle any detections
                self.handle_detections(results)
                
                # Print progress
                if self.stats['total_scans'] % 10 == 0:
                    self._print_progress()
                
                # Wait before next scan
                time.sleep(self.scan_interval)
                
            except Exception as e:
                print(f"❌ Error in monitoring loop: {e}")
                self._log_event("monitoring_loop_error", {"error": str(e)}, level="error")
                time.sleep(self.scan_interval)

    def _default_console_alert(self, results: Dict):
        detections = results.get('detections', [])
        _print_readable_alert(detections, title_prefix="ALERT")
    
    def _print_progress(self):
        """Print monitoring progress"""
        print(f"\r📊 Scans: {self.stats['total_scans']} | "
              f"Skipped: {self.stats['frames_skipped']} | "
              f"Detections: {self.stats['total_detections']} | "
              f"PendingFiltered: {self.stats['filtered_unconfirmed']} | "
              f"DupSuppressed: {self.stats['suppressed_duplicates']} | "
              f"Avg Time: {self.stats['avg_scan_time']:.2f}s", 
              end='', flush=True)
    
    def start_monitoring(self, monitor_number: int = 1):
        """Start real-time monitoring"""
        if self.is_monitoring:
            print("⚠️  Monitoring already running")
            return
        
        # Start new session
        self.current_session = ScreenShareSession(
            session_id=f"session_{datetime.now().strftime('%Y%m%d_%H%M%S')}",
            start_time=datetime.now()
        )
        
        self.is_monitoring = True
        
        # Start monitoring thread
        self.monitor_thread = threading.Thread(
            target=self.monitoring_loop,
            args=(monitor_number,),
            daemon=True
        )
        self.monitor_thread.start()
        
        print(f"✅ Monitoring started!")
        print(f"   Session ID: {self.current_session.session_id}")
        self._log_event("monitoring_started", {
            "session_id": self.current_session.session_id,
            "monitor_number": monitor_number
        })
    
    def stop_monitoring(self):
        """Stop monitoring"""
        if not self.is_monitoring:
            print("⚠️  Monitoring not running")
            return
        
        print("\n\n🛑 Stopping monitoring...")
        self.is_monitoring = False
        
        # Wait for thread to finish
        if self.monitor_thread:
            self.monitor_thread.join(timeout=5)
        
        # Finalize session
        if self.current_session:
            self.current_session.end_time = datetime.now()
        
        print("✅ Monitoring stopped!")
        self._log_event("monitoring_stopped", {
            "session_id": self.current_session.session_id if self.current_session else None,
            "total_scans": self.stats['total_scans'],
            "total_detections": self.stats['total_detections']
        })
        
        # Print session summary
        self._print_session_summary()
    
    def _print_session_summary(self):
        """Print summary of monitoring session"""
        if not self.current_session:
            return
        
        session = self.current_session
        duration = (session.end_time - session.start_time).total_seconds()
        
        print("\n" + "="*80)
        print("📊 SESSION SUMMARY")
        print("="*80)
        print(f"\nSession ID: {session.session_id}")
        print(f"Duration: {duration/60:.1f} minutes")
        print(f"Total Scans: {session.total_scans}")
        print(f"Frames Skipped: {self.stats['frames_skipped']}")
        print(f"Scan Efficiency: {(1 - self.stats['frames_skipped']/max(1, self.stats['total_scans']))*100:.1f}%")
        print(f"Average Scan Time: {self.stats['avg_scan_time']:.2f}s")
        print(f"Filtered (Unconfirmed/Low OCR): {self.stats['filtered_unconfirmed']}")
        print(f"Suppressed (Duplicate Cooldown): {self.stats['suppressed_duplicates']}")
        
        print(f"\n🔍 Detections:")
        print(f"   Total: {session.total_detections}")
        print(f"   Critical: {session.critical_detections}")
        print(f"   High: {session.high_detections}")
        
        if session.detections:
            print(f"\n⚠️  Recent Detections:")
            for detection in session.detections[-5:]:  # Show last 5
                print(f"   • [{detection['severity'].upper()}] "
                      f"{detection['rule_name']}: "
                      f"{detection['matched_text']}")
        
        print("\n" + "="*80)
    
    def export_session_report(self, filepath: str = None):
        """Export session report as JSON"""
        if not self.current_session:
            print("⚠️  No active session")
            return
        
        if filepath is None:
            filepath = f"session_report_{self.current_session.session_id}.json"
        
        report = {
            'session_id': self.current_session.session_id,
            'start_time': self.current_session.start_time.isoformat(),
            'end_time': self.current_session.end_time.isoformat() if self.current_session.end_time else None,
            'duration_seconds': (self.current_session.end_time - self.current_session.start_time).total_seconds() if self.current_session.end_time else None,
            'total_scans': self.current_session.total_scans,
            'total_detections': self.current_session.total_detections,
            'critical_detections': self.current_session.critical_detections,
            'high_detections': self.current_session.high_detections,
            'detections': self.current_session.detections,
            'stats': {
                'frames_skipped': self.stats['frames_skipped'],
                'avg_scan_time': self.stats['avg_scan_time'],
                'filtered_unconfirmed': self.stats['filtered_unconfirmed'],
                'suppressed_duplicates': self.stats['suppressed_duplicates'],
                'scanner_type': self.scanner_type
            }
        }
        
        with open(filepath, 'w') as f:
            json.dump(report, f, indent=2)
        
        print(f"✅ Report exported to: {filepath}")
        self._log_event("session_report_exported", {"path": filepath})


# Alert callback examples
def console_alert(results: Dict):
    """Simple console alert"""
    detections = results['detections']
    _print_readable_alert(detections, title_prefix="ALERT")


def desktop_notification_alert(results: Dict, target_os: str = "auto"):
    """Show desktop notification"""
    detections = results['detections']
    if not detections:
        return
    
    critical_count = sum(1 for d in detections if getattr(d, 'severity', '') == 'critical')
    high_count = sum(1 for d in detections if getattr(d, 'severity', '') == 'high')
    
    import platform
    import subprocess
    
    title = "⚠️ Sensitive Content Detected!"
    message = f"Found {len(detections)} sensitive items\n"
    if critical_count:
        message += f"Critical: {critical_count} "
    if high_count:
        message += f"High: {high_count}"
    
    system = platform.system()
    os_map = {
        "mac": "Darwin",
        "macos": "Darwin",
        "darwin": "Darwin",
        "windows": "Windows",
        "win": "Windows",
        "linux": "Linux",
        "auto": system
    }
    selected = os_map.get((target_os or "auto").lower(), system)
    
    try:
        if selected == "Darwin":  # macOS
            script = f'display notification "{message}" with title "{title}" sound name "Glass"'
            subprocess.run(["osascript", "-e", script])
        elif selected == "Linux":
            subprocess.run(["notify-send", "-u", "critical", title, message])
        elif selected == "Windows":
            safe_title = title.replace('"', "'")
            safe_message = message.replace('"', "'")
            ps_script = f"""
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$xml = [xml]$template.GetXml()
$xml.toast.visual.binding.text[0].AppendChild($xml.CreateTextNode("{safe_title}")) | Out-Null
$xml.toast.visual.binding.text[1].AppendChild($xml.CreateTextNode("{safe_message}")) | Out-Null
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("Screen Share Monitor").Show($toast)
"""
            subprocess.run(["powershell", "-NoProfile", "-Command", ps_script])
    except:
        pass


def webhook_alert(webhook_url: str):
    """Create webhook alert function"""
    import requests
    
    def alert(results: Dict):
        detections = results['detections']
        if not detections:
            return
        
        payload = {
            'text': f"🚨 Sensitive content detected: {len(detections)} items",
            'detections': [
                {
                    'rule': getattr(d, 'rule_name', 'unknown'),
                    'severity': getattr(d, 'severity', 'unknown'),
                    'confidence': getattr(d, 'confidence', 0.0)
                }
                for d in detections[:5]  # First 5
            ]
        }
        
        try:
            requests.post(webhook_url, json=payload, timeout=5)
        except:
            pass
    
    return alert


# Demo/Usage
def demo_realtime_monitor():
    """Demonstrate real-time monitoring"""
    
    print("\n" + "="*80)
    print("🔴 REAL-TIME SCREEN SHARE MONITOR")
    print("="*80)
    
    # Create monitor with alert callback
    monitor = RealTimeScreenShareMonitor(
        scan_interval=2.0,  # Scan every 2 seconds
        use_ml=True,        # Use ML scanner
        use_gpu=True,       # Use GPU if available
        alert_callback=console_alert,  # Print alerts to console
        min_confidence=0.6,  # Confidence threshold
        use_ner=False,       # Better precision for live monitoring
        confirmation_frames=2,
        confirmation_window=3,
        alert_cooldown_seconds=12,
        enable_bert_enrichment=True,
        log_file="monitor_events.log",
        verbose_logging=True
    )
    
    print("\n📋 Commands:")
    print("   - Press Enter to start monitoring")
    print("   - Press Ctrl+C to stop and see summary")
    print("   - Open sensitive documents to test detection")
    
    input("\nPress Enter to start monitoring...")
    
    # Start monitoring
    monitor.start_monitoring()
    
    try:
        # Keep running until Ctrl+C
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n\n⚠️  Stopping...")
        monitor.stop_monitoring()
        
        # Export report
        monitor.export_session_report()


if __name__ == "__main__":
    demo_realtime_monitor()
