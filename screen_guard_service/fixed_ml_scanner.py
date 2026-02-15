"""
Fixed ML Scanner - Improved API Key Detection
Reduces false positives from NER models
"""

try:
    import cv2
    CV2_AVAILABLE = True
except ImportError:
    cv2 = None
    CV2_AVAILABLE = False
import numpy as np
import re
import ipaddress
import math
from typing import List, Dict, Tuple
from dataclasses import dataclass
from datetime import datetime
from monitor_runtime.pii_catalog import get_pii_regex_patterns
from safe_print import safe_print

try:
    import easyocr
    EASYOCR_AVAILABLE = True
except ImportError:
    EASYOCR_AVAILABLE = False

try:
    from transformers import pipeline
    import torch
    TRANSFORMERS_AVAILABLE = True
except ImportError:
    TRANSFORMERS_AVAILABLE = False


@dataclass
class Detection:
    rule_name: str
    matched_text: str
    severity: str
    confidence: float
    detection_method: str
    timestamp: datetime
    context: str = ""


class FixedMLScanner:
    """
    Fixed ML Scanner with improved API key detection
    - Prioritizes regex for API keys (more accurate)
    - Filters out spurious NER detections
    - Better placeholder detection
    """
    
    def __init__(self, use_gpu: bool = True, use_ner: bool = False, adaptive_context: bool = True):
        """
        Initialize scanner
        
        Args:
            use_gpu: Use GPU acceleration
            use_ner: Enable NER (disabled by default due to false positives)
            adaptive_context: Enable context-aware sensitive token detection
        """
        self.use_gpu = use_gpu
        self.use_ner_models = use_ner
        self.adaptive_context = adaptive_context
        
        safe_print(f"[INIT] Initializing Fixed ML Scanner...")
        safe_print(f"   GPU: {'Enabled' if use_gpu else 'Disabled'}")
        safe_print(f"   NER: {'Enabled' if use_ner else 'Disabled (regex-only mode)'}")
        safe_print(f"   Context Detection: {'Enabled' if adaptive_context else 'Disabled'}")
        
        # Initialize OCR
        self._init_ocr()
        
        # Initialize NER (optional)
        if use_ner:
            self._init_ner()
        else:
            self.ner_model = None
            self.pii_detector = None
        
        # Initialize regex patterns (PRIMARY detection method)
        self._init_regex_patterns()
        
        safe_print("[OK] Scanner initialized!")
    
    def _init_ocr(self):
        """Initialize OCR"""
        if EASYOCR_AVAILABLE:
            safe_print("   Loading EasyOCR...")
            self.ocr_reader = easyocr.Reader(
                ['en'],
                gpu=self.use_gpu,
                verbose=False
            )
            self.ocr_method = 'easyocr'
            safe_print("   [OK] EasyOCR loaded")
        else:
            import pytesseract
            self.ocr_reader = None
            self.ocr_method = 'tesseract'
            safe_print("   [OK] Using Tesseract")
    
    def _init_ner(self):
        """Initialize NER (optional, can cause false positives)"""
        if not TRANSFORMERS_AVAILABLE:
            self.ner_model = None
            self.pii_detector = None
            return
        
        try:
            safe_print("   Loading PII detector...")
            # Only use PII-specific model, skip generic NER
            self.pii_detector = pipeline(
                "token-classification",
                model="lakshyakh93/deberta_finetuned_pii",
                device=0 if self.use_gpu else -1,
                aggregation_strategy="simple"
            )
            self.ner_model = None  # Don't use generic NER
            safe_print("   [OK] PII detector loaded")
        except Exception as e:
            safe_print(f"   [WARN]  Could not load NER: {e}")
            self.ner_model = None
            self.pii_detector = None
    
    def _init_regex_patterns(self):
        """Initialize high-specificity regex patterns"""
        self.regex_patterns = get_pii_regex_patterns()
    
    def extract_text(self, image: np.ndarray) -> Tuple[str, float]:
        """Extract text from image"""
        if self.ocr_method == 'easyocr' and self.ocr_reader:
            results = self.ocr_reader.readtext(image)
            full_text = " ".join([text for (_, text, _) in results])
            avg_conf = np.mean([conf for (_, _, conf) in results]) if results else 0.0
            return full_text, avg_conf
        else:
            # Tesseract fallback
            import pytesseract
            if not CV2_AVAILABLE:
                text = pytesseract.image_to_string(image)
                return text, 0.65
            gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
            
            # Try multiple preprocessing for better accuracy
            texts = []
            
            # Method 1: Enhanced contrast
            enhanced = cv2.convertScaleAbs(gray, alpha=1.5, beta=0)
            texts.append(pytesseract.image_to_string(enhanced))
            
            # Method 2: Inverted (for dark themes)
            inverted = cv2.bitwise_not(gray)
            texts.append(pytesseract.image_to_string(inverted))
            
            # Use longest result
            full_text = max(texts, key=len)
            return full_text, 0.8
    
    def detect_with_regex(self, text: str) -> List[Detection]:
        """PRIMARY detection method - regex patterns"""
        detections = []
        case_sensitive_rules = {
            "swift_bic",
            "private_key_pem",
            "pgp_private_key_block",
            "public_key_pem",
            "ssh_public_key",
        }
        
        for rule_name, rule_info in self.regex_patterns.items():
            pattern = rule_info['pattern']
            severity = rule_info['severity']
            flags = 0 if rule_name in case_sensitive_rules else re.IGNORECASE
            matches = re.finditer(pattern, text, flags)
            
            for match in matches:
                matched_text = match.group(0)
                
                # For patterns with groups, extract the actual value
                if match.lastindex and match.lastindex >= 2:
                    value = match.group(match.lastindex)
                else:
                    value = matched_text
                
                # Skip placeholders
                if self._is_placeholder(value):
                    continue

                # Rule-aware validation to reduce false positives
                if not self._is_valid_detection(rule_name, value, matched_text):
                    continue

                confidence = self._confidence_for_detection(
                    rule_name=rule_name,
                    severity=severity,
                    value=value,
                    matched_text=matched_text
                )
                
                # Get context
                start = max(0, match.start() - 30)
                end = min(len(text), match.end() + 30)
                context = text[start:end]
                
                detection = Detection(
                    rule_name=rule_name,
                    matched_text=matched_text,
                    severity=severity,
                    confidence=confidence,
                    detection_method='regex',
                    timestamp=datetime.now(),
                    context=context
                )
                detections.append(detection)
        
        return detections
    
    def detect_with_ner(self, text: str) -> List[Detection]:
        """SECONDARY detection method - NER (optional, filtered)"""
        detections = []
        
        if not self.pii_detector:
            return detections
        
        try:
            entities = self.pii_detector(text)
            
            # STRICT filtering - only accept high-confidence, relevant entities
            for entity in entities:
                entity_type = entity['entity_group'].upper()
                confidence = entity['score']
                
                # FILTER 1: Only accept certain entity types
                allowed_types = {
                    'EMAIL', 'PHONE', 'PHONE_NUM', 
                    'CREDIT_CARD', 'SSN', 'SOCIALSECURITYNUMBER',
                    'PASSWORD', 'SECRET', 'KEY'
                }
                
                if entity_type not in allowed_types:
                    continue
                
                # FILTER 2: Require high confidence
                if confidence < 0.85:
                    continue
                
                # FILTER 3: Skip if already detected by regex
                word = entity['word']
                if self._already_detected_by_regex(word, text):
                    continue

                # FILTER 4: Skip placeholders/noisy tokens
                if self._is_placeholder(word):
                    continue
                
                severity_map = {
                    'EMAIL': 'medium',
                    'PHONE': 'medium',
                    'PHONE_NUM': 'medium',
                    'CREDIT_CARD': 'critical',
                    'SSN': 'critical',
                    'SOCIALSECURITYNUMBER': 'critical',
                    'PASSWORD': 'critical',
                    'SECRET': 'critical',
                    'KEY': 'high'
                }
                
                severity = severity_map.get(entity_type, 'medium')
                
                detection = Detection(
                    rule_name=f"ner_{entity_type.lower()}",
                    matched_text=word,
                    severity=severity,
                    confidence=confidence,
                    detection_method='ner',
                    timestamp=datetime.now(),
                    context=text[max(0, entity['start']-30):min(len(text), entity['end']+30)]
                )
                detections.append(detection)
                
        except Exception as e:
            safe_print(f"NER error: {e}")
        
        return detections
    
    def _already_detected_by_regex(self, word: str, text: str) -> bool:
        """Check if word was already caught by regex"""
        # Simple check: if word looks like an API key pattern
        api_patterns = [
            r'sk_live_', r'sk_test_', r'AKIA', r'ghp_', r'AIza',
            r'sk-proj-', r'xox[pbar]-'
        ]
        
        for pattern in api_patterns:
            if re.search(pattern, word):
                return True
        
        return False

    def _shannon_entropy(self, text: str) -> float:
        """Approximate token randomness to separate real secrets from common words."""
        if not text:
            return 0.0
        probs = [text.count(ch) / len(text) for ch in set(text)]
        return -sum(p * math.log2(p) for p in probs if p > 0)

    def _is_valid_credit_card(self, value: str) -> bool:
        digits = re.sub(r'\D', '', value)
        if not (13 <= len(digits) <= 19):
            return False
        # Luhn check
        total = 0
        reverse_digits = digits[::-1]
        for i, d in enumerate(reverse_digits):
            n = int(d)
            if i % 2 == 1:
                n *= 2
                if n > 9:
                    n -= 9
            total += n
        return total % 10 == 0

    def _is_valid_ssn(self, value: str) -> bool:
        m = re.match(r'^(\d{3})-(\d{2})-(\d{4})$', value)
        if not m:
            return False
        area, group, serial = m.groups()
        if area in {'000', '666'} or area.startswith('9'):
            return False
        if group == '00' or serial == '0000':
            return False
        return True

    def _is_valid_private_ip(self, value: str) -> bool:
        try:
            ip = ipaddress.ip_address(value)
            return ip.is_private
        except ValueError:
            return False

    def _normalize_ocr_numeric(self, value: str) -> str:
        """
        Normalize common OCR confusions inside numeric identifiers.
        """
        table = str.maketrans({
            'O': '0', 'o': '0',
            'I': '1', 'l': '1', '|': '1',
            'S': '5', 's': '5',
            'B': '8'
        })
        return value.translate(table)

    def _has_financial_context(self, context: str) -> bool:
        c = context.lower()
        terms = (
            "card", "credit", "debit", "visa", "mastercard", "amex", "discover",
            "payment", "billing", "cvv", "cvc", "expiry", "exp", "mm/yy"
        )
        return any(t in c for t in terms)

    def detect_credit_cards_robust(self, text: str) -> List[Detection]:
        """
        OCR-robust credit card detector:
        - handles common OCR substitutions
        - validates with Luhn when possible
        - uses financial context fallback for near-miss OCR
        """
        detections: List[Detection] = []
        seen = set()

        # 16-digit common grouping, allowing OCR-char substitutions.
        candidate_patterns = [
            r'(?<![A-Za-z0-9])(?:[0-9OIlSB]{4}[-\s]?){3}[0-9OIlSB]{4}(?![A-Za-z0-9])',
            # 15-digit AmEx-like grouping
            r'(?<![A-Za-z0-9])[0-9OIlSB]{4}[-\s]?[0-9OIlSB]{6}[-\s]?[0-9OIlSB]{5}(?![A-Za-z0-9])',
        ]

        for pat in candidate_patterns:
            for m in re.finditer(pat, text):
                raw = m.group(0)
                norm = self._normalize_ocr_numeric(raw)
                digits = re.sub(r'\D', '', norm)
                if not (13 <= len(digits) <= 19):
                    continue
                if len(set(digits)) <= 2:
                    continue

                start = max(0, m.start() - 40)
                end = min(len(text), m.end() + 40)
                context = text[start:end]

                if self._is_valid_credit_card(norm):
                    rule_name = "credit_card"
                    severity = "critical"
                    confidence = 0.96
                else:
                    # Fallback only when context strongly suggests payment/card details.
                    if not self._has_financial_context(context):
                        continue
                    rule_name = "credit_card_suspected"
                    severity = "high"
                    confidence = 0.78

                key = (rule_name, digits[-8:])
                if key in seen:
                    continue
                seen.add(key)

                detections.append(Detection(
                    rule_name=rule_name,
                    matched_text=raw,
                    severity=severity,
                    confidence=confidence,
                    detection_method='regex',
                    timestamp=datetime.now(),
                    context=context
                ))

        return detections

    def _token_looks_secret(self, token: str) -> bool:
        token = token.strip().strip('\'"')
        if len(token) < 16:
            return False
        charset_size = len(set(token))
        entropy = self._shannon_entropy(token)
        has_mixed_types = (
            re.search(r'[a-z]', token) is not None and
            re.search(r'[A-Z]', token) is not None and
            re.search(r'\d', token) is not None
        )
        return charset_size >= 8 and (entropy >= 3.0 or has_mixed_types)

    def _password_looks_sensitive(self, token: str) -> bool:
        token = token.strip().strip('\'"')
        if len(token) < 8 or len(token) > 128:
            return False
        if self._is_placeholder(token):
            return False
        mask_chars = set("*•.")
        if all(ch in mask_chars for ch in token):
            return False
        masked_ratio = sum(1 for ch in token if ch in mask_chars) / max(1, len(token))
        if masked_ratio >= 0.5:
            return False

        has_lower = re.search(r"[a-z]", token) is not None
        has_upper = re.search(r"[A-Z]", token) is not None
        has_digit = re.search(r"\d", token) is not None
        has_symbol = re.search(r"[^A-Za-z0-9]", token) is not None

        # Common real-world password shapes.
        if (has_lower and has_upper and has_digit) or (has_lower and has_digit and has_symbol):
            return True

        # Longer passphrases are still sensitive even with weaker character diversity.
        return len(token) >= 12

    def _looks_like_code_identifier(self, token: str) -> bool:
        stripped = token.strip()
        if stripped.endswith("="):
            return True
        if re.match(r"^[A-Z_][A-Z0-9_]{6,}$", stripped):
            return True
        if re.match(r"^[a-z_][a-z0-9_]{8,}$", stripped) and "_" in stripped and not re.search(r"\d", stripped):
            return True
        return False

    def _is_valid_detection(self, rule_name: str, value: str, matched_text: str) -> bool:
        """Rule-specific validation to reduce OCR/regex false positives."""
        rule_name = (rule_name or "").lower()
        if rule_name == 'credit_card':
            return self._is_valid_credit_card(value)
        if rule_name == 'ssn':
            return self._is_valid_ssn(value)
        if rule_name == 'private_ip':
            return self._is_valid_private_ip(value)
        if rule_name == 'iban':
            return 15 <= len(re.sub(r'[^A-Za-z0-9]', '', value)) <= 34
        if rule_name == 'swift_bic':
            stripped = re.sub(r'[^A-Za-z0-9]', '', value).upper()
            if len(stripped) not in {8, 11}:
                return False
            # SWIFT: 4 letters bank, 2 letters country, 2 alnum location, optional 3 alnum branch.
            if not re.match(r'^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$', stripped):
                return False
            country = stripped[4:6]
            valid_countries = {
                "US", "CA", "GB", "DE", "FR", "ES", "IT", "NL", "BE", "CH", "SE", "NO", "DK", "FI", "IE", "PT",
                "AU", "NZ", "JP", "KR", "SG", "HK", "CN", "IN", "AE", "SA", "ZA", "BR", "MX", "AR"
            }
            return country in valid_countries

        if rule_name in {'api_key_assignment', 'secret_assignment', 'token_assignment'}:
            return self._token_looks_secret(value)
        if rule_name in {'password_assignment', 'password_field_html'}:
            return self._password_looks_sensitive(value)
        if rule_name in {'private_key_assignment', 'public_key_assignment'}:
            return len(value.strip().strip("'\"")) >= 32
        if rule_name in {'private_key_pem', 'pgp_private_key_block'}:
            return ("BEGIN" in matched_text and "END" in matched_text and len(matched_text) >= 120)
        if rule_name in {'public_key_pem', 'ssh_public_key'}:
            return len(matched_text) >= 60

        if rule_name in {'email'}:
            # Skip obvious examples/placeholders.
            return 'example.' not in value.lower() and not self._is_placeholder(value)

        if rule_name in {'jwt_token'}:
            parts = value.split('.')
            return len(parts) == 3 and all(len(p) >= 8 for p in parts)

        # Known key formats are already high-specificity.
        return True

    def _confidence_for_detection(self, rule_name: str, severity: str, value: str, matched_text: str) -> float:
        rule_name = (rule_name or "").lower()
        base = {
            'critical': 0.92,
            'high': 0.88,
            'medium': 0.80,
            'low': 0.70
        }.get(severity, 0.80)

        if rule_name in {
            'openai_key', 'aws_access_key', 'github_token',
            'google_api_key', 'stripe_live_key', 'slack_token'
        }:
            base += 0.05
        if rule_name in {'private_key_pem', 'pgp_private_key_block'}:
            base = max(base, 0.97)
        if rule_name in {'public_key_pem', 'ssh_public_key', 'private_key_assignment', 'public_key_assignment'}:
            base = max(base, 0.90)

        if rule_name in {'api_key_assignment', 'secret_assignment', 'token_assignment'}:
            entropy = self._shannon_entropy(value)
            if entropy >= 3.5:
                base += 0.05
            elif entropy < 2.5:
                base -= 0.08
        if rule_name in {'password_assignment', 'password_field_html'}:
            if len(value) >= 12:
                base += 0.03
            if re.search(r"[^A-Za-z0-9]", value):
                base += 0.02

        if rule_name == 'phone' and len(re.sub(r'\D', '', matched_text)) != 10:
            base -= 0.06

        return max(0.60, min(0.99, base))

    def _context_signal_score(self, context: str) -> float:
        """
        Score context for sensitive intent. Uses soft lexical cues + optional PII model.
        """
        context_l = context.lower()
        lexical_score = 0.0
        signal_terms = {
            "password": 0.30,
            "passcode": 0.30,
            "pwd": 0.25,
            "secret": 0.28,
            "token": 0.24,
            "api key": 0.30,
            "apikey": 0.30,
            "auth": 0.18,
            "credential": 0.28,
            "login": 0.16,
            "ssh": 0.20,
            "bearer": 0.25,
            "credit card": 0.30,
            "card number": 0.30,
            "ssn": 0.30,
            "social security": 0.30,
            "private key": 0.32,
            "access key": 0.30,
        }
        for term, weight in signal_terms.items():
            if term in context_l:
                lexical_score += weight
        lexical_score = min(0.7, lexical_score)

        nlp_score = 0.0
        if self.pii_detector:
            try:
                entities = self.pii_detector(context)
                for ent in entities:
                    ent_type = (ent.get("entity_group") or "").upper()
                    conf = float(ent.get("score", 0.0))
                    if ent_type in {"PASSWORD", "SECRET", "KEY", "CREDIT_CARD", "SSN", "SOCIALSECURITYNUMBER"}:
                        nlp_score = max(nlp_score, min(0.4, conf * 0.4))
            except Exception:
                pass

        return min(1.0, lexical_score + nlp_score)

    def detect_with_context(self, text: str) -> List[Detection]:
        """
        Adaptive detector for exposed sensitive values in natural UI text.
        This avoids relying only on exact assignment-style regex patterns.
        """
        detections: List[Detection] = []
        if not self.adaptive_context or not text:
            return detections

        candidate_pattern = re.compile(r'(?<![A-Za-z0-9])([A-Za-z0-9_\-@!#$%^&*().+=]{8,80})(?![A-Za-z0-9])')
        for match in candidate_pattern.finditer(text):
            token = match.group(1)
            if self._is_placeholder(token):
                continue
            if token.lower() in {"undefined", "null", "password", "username", "email"}:
                continue
            if self._looks_like_code_identifier(token):
                continue

            entropy = self._shannon_entropy(token)
            token_like_secret = self._token_looks_secret(token) or self._password_looks_sensitive(token)
            if not token_like_secret:
                continue

            start = max(0, match.start() - 64)
            end = min(len(text), match.end() + 64)
            context = text[start:end]
            context_score = self._context_signal_score(context)

            # Need some contextual evidence to avoid random long words.
            if context_score < 0.22 and entropy < 3.2:
                continue

            # Enterprise guardrail: avoid generic source-code/log terms without value-like signal.
            if re.match(r"^[A-Za-z_][A-Za-z0-9_]{6,}$", token) and context_score < 0.40:
                continue

            confidence = min(0.95, 0.55 + (min(1.0, entropy / 5.0) * 0.20) + (context_score * 0.30))
            severity = "critical" if confidence >= 0.86 else "high"

            detections.append(Detection(
                rule_name="contextual_sensitive_token",
                matched_text=token,
                severity=severity,
                confidence=confidence,
                detection_method="contextual_nlp",
                timestamp=datetime.now(),
                context=context,
            ))

        return detections
    
    def _is_placeholder(self, text: str) -> bool:
        """Check if text is a placeholder"""
        text_lower = text.lower()
        
        placeholders = [
            'your_', 'example', 'test', 'sample', 'placeholder',
            'xxx', 'yyy', 'zzz', '000', '111', '123', 'abc',
            'dummy', 'fake', 'put_your', 'insert_', 'enter_',
            '<your', '[your', 'replace_this', 'change_me'
        ]
        
        # Check for placeholder keywords
        if any(p in text_lower for p in placeholders):
            return True
        
        # Check for repetitive patterns (like "aaaaaaa")
        if len(set(text)) < 4 and len(text) > 10:
            return True
        
        # Check if all same character
        if len(set(text)) == 1 and len(text) > 5:
            return True
        
        return False
    
    def scan_image(self, image: np.ndarray, min_confidence: float = 0.6) -> Dict:
        """
        Scan image for sensitive content
        
        Returns:
            Dictionary with detections and metadata
        """
        start_time = datetime.now()
        
        # Extract text
        text, ocr_conf = self.extract_text(image)
        
        # PRIMARY: Regex detection (most accurate for API keys)
        regex_detections = self.detect_with_regex(text)
        card_detections = self.detect_credit_cards_robust(text)
        
        # SECONDARY: NER detection (optional, filtered)
        ner_detections = []
        if self.use_ner_models and self.pii_detector:
            ner_detections = self.detect_with_ner(text)

        context_detections = self.detect_with_context(text)
        
        # Combine
        all_detections = regex_detections + card_detections + ner_detections + context_detections
        
        # Deduplicate
        filtered = self._deduplicate(all_detections, min_confidence)
        
        processing_time = (datetime.now() - start_time).total_seconds()
        
        return {
            'detections': filtered,
            'extracted_text': text,
            'ocr_confidence': ocr_conf,
            'total_detections': len(filtered),
            'processing_time': processing_time,
            'methods_used': {
                'regex': True,
                'credit_card_robust': True,
                'ner': self.use_ner_models and self.pii_detector is not None,
                'contextual_nlp': self.adaptive_context,
                'ocr': True
            }
        }
    
    def scan_text(self, text: str) -> Dict:
        """Scan text directly"""
        start_time = datetime.now()
        
        regex_detections = self.detect_with_regex(text)
        card_detections = self.detect_credit_cards_robust(text)
        
        ner_detections = []
        if self.use_ner_models and self.pii_detector:
            ner_detections = self.detect_with_ner(text)

        context_detections = self.detect_with_context(text)
        
        all_detections = regex_detections + card_detections + ner_detections + context_detections
        filtered = self._deduplicate(all_detections, 0.6)
        
        processing_time = (datetime.now() - start_time).total_seconds()
        
        return {
            'detections': filtered,
            'total_detections': len(filtered),
            'processing_time': processing_time
        }
    
    def _deduplicate(self, detections: List[Detection], min_confidence: float) -> List[Detection]:
        """Remove duplicates and filter by confidence"""
        # Filter by confidence
        filtered = [d for d in detections if d.confidence >= min_confidence]
        
        # Deduplicate by matched text
        seen = {}
        for detection in filtered:
            key = detection.matched_text.lower().strip()
            
            # Keep highest confidence
            if key not in seen or detection.confidence > seen[key].confidence:
                seen[key] = detection
        
        return list(seen.values())


def display_results(results: Dict):
    """Display scan results"""
    safe_print("\n" + "="*80)
    safe_print("[SCAN] SCAN RESULTS")
    safe_print("="*80)
    
    safe_print(f"\n[STATS] Metadata:")
    safe_print(f"   Processing Time: {results['processing_time']:.3f}s")
    if 'ocr_confidence' in results:
        safe_print(f"   OCR Confidence: {results['ocr_confidence']:.2%}")
    
    detections = results['detections']
    
    if not detections:
        safe_print("\n[OK] No sensitive content detected!")
        return
    
    safe_print(f"\n[WARN]  FOUND {len(detections)} SENSITIVE ITEM(S)")
    safe_print("="*80)
    
    severity_icons = {
        'low': '[INFO]',
        'medium': '[WARN]',
        'high': '[CRITICAL]',
        'critical': '[ALERT]'
    }
    
    # Sort by severity
    severity_order = {'critical': 0, 'high': 1, 'medium': 2, 'low': 3}
    sorted_detections = sorted(detections, 
                               key=lambda d: severity_order.get(d.severity, 4))
    
    for i, detection in enumerate(sorted_detections, 1):
        icon = severity_icons.get(detection.severity, '[WARN]')
        safe_print(f"\n{icon} Detection #{i}")
        safe_print(f"   Type: {detection.rule_name}")
        safe_print(f"   Severity: {detection.severity.upper()}")
        safe_print(f"   Method: {detection.detection_method}")
        safe_print(f"   Confidence: {detection.confidence:.2%}")
        safe_print(f"   Matched: {detection.matched_text[:60]}...")
        if detection.context:
            safe_print(f"   Context: ...{detection.context[:50]}...")
    
    safe_print("\n" + "="*80)


def demo():
    """Demo the fixed scanner"""
    safe_print("\n[INIT] FIXED ML SCANNER DEMO")
    safe_print("="*80)
    
    # Initialize scanner (NER disabled by default for better accuracy)
    scanner = FixedMLScanner(
        use_gpu=False,
        use_ner=False  # Disabled to avoid false positives
    )
    
    # Test with .env file content
    test_texts = [
        {
            'name': '.env File with API Keys',
            'text': '''
API_KEY=sk_live_51H8v9K2eZvKYlo2C9K2eZvKYlo2C
OPENAI_API_KEY=sk-proj-abcd1234efgh5678ijkl90mnopqrstuvwxyz1234567890
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
SECRET_KEY=MySecretPassword123!
DATABASE_URL=postgresql://user:pass@localhost/db
            '''
        },
        {
            'name': 'Credit Card Info',
            'text': 'Payment Card: 4532-1234-5678-9012, Email: john@example.com'
        }
    ]
    
    for test in test_texts:
        safe_print(f"\n\n{'='*80}")
        safe_print(f"TEST: {test['name']}")
        safe_print('='*80)
        safe_print(f"Input text:\n{test['text'][:150]}...")
        
        results = scanner.scan_text(test['text'])
        display_results(results)


if __name__ == "__main__":
    demo()
