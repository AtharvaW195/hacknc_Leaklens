# Modern ML-Based Sensitive Content Scanner
# Uses: Transformers, NER, Computer Vision, and Deep Learning

"""
Advanced Scanner Architecture:
1. EasyOCR (better than Tesseract) - 95%+ accuracy
2. Transformer-based NER - Identifies entities automatically
3. BERT for context understanding
4. Computer Vision for visual patterns
5. Ensemble detection for maximum accuracy
"""

import numpy as np
from typing import List, Dict, Tuple, Optional
from dataclasses import dataclass
from datetime import datetime
try:
    import cv2
    CV2_AVAILABLE = True
except ImportError:
    cv2 = None
    CV2_AVAILABLE = False
import re
from safe_print import safe_print

# Core ML imports (install via requirements)
try:
    import easyocr
    EASYOCR_AVAILABLE = True
except ImportError:
    EASYOCR_AVAILABLE = False
    safe_print("[WARN]  EasyOCR not available. Install: pip install easyocr")

try:
    from transformers import pipeline, AutoTokenizer, AutoModelForTokenClassification
    import torch
    TRANSFORMERS_AVAILABLE = True
except ImportError:
    TRANSFORMERS_AVAILABLE = False
    safe_print("[WARN]  Transformers not available. Install: pip install transformers torch")

try:
    import spacy
    SPACY_AVAILABLE = True
except ImportError:
    SPACY_AVAILABLE = False
    safe_print("[WARN]  Spacy not available. Install: pip install spacy")


@dataclass
class MLDetection:
    """Enhanced detection with ML confidence and method"""
    rule_name: str
    matched_text: str
    severity: str
    confidence: float
    detection_method: str  # 'regex', 'ner', 'ml', 'visual', 'ensemble'
    timestamp: datetime
    context: str = ""
    bbox: Optional[Tuple[int, int, int, int]] = None  # Bounding box in image


class ModernMLScanner:
    """
    State-of-the-art sensitive content scanner using ML/NLP
    
    Features:
    - EasyOCR for superior text extraction
    - Transformer-based NER
    - BERT for context understanding
    - Visual pattern recognition
    - Ensemble detection
    """
    
    def __init__(self, 
                 use_gpu: bool = True,
                 model_size: str = 'medium',  # 'small', 'medium', 'large'
                 languages: List[str] = ['en']):
        """
        Initialize the modern scanner
        
        Args:
            use_gpu: Use GPU acceleration if available
            model_size: Model size for speed/accuracy tradeoff
            languages: Languages to detect (default: English)
        """
        self.device = 'cuda' if use_gpu and torch.cuda.is_available() else 'cpu'
        self.model_size = model_size
        self.languages = languages
        
        safe_print(f"[INIT] Initializing Modern ML Scanner...")
        safe_print(f"   Device: {self.device}")
        safe_print(f"   Model Size: {model_size}")
        
        # Initialize OCR
        self._init_ocr()
        
        # Initialize NLP models
        self._init_nlp_models()
        
        # Initialize regex patterns (fallback)
        self._init_regex_patterns()
        
        safe_print("[OK] Scanner initialized!")
    
    def _init_ocr(self):
        """Initialize OCR engine"""
        if EASYOCR_AVAILABLE:
            safe_print("   Loading EasyOCR...")
            self.ocr_reader = easyocr.Reader(
                self.languages,
                gpu=self.device == 'cuda',
                verbose=False
            )
            self.ocr_method = 'easyocr'
            safe_print("   [OK] EasyOCR loaded")
        else:
            # Fallback to pytesseract
            import pytesseract
            self.ocr_reader = None
            self.ocr_method = 'tesseract'
            safe_print("   [OK] Using Tesseract (fallback)")
    
    def _init_nlp_models(self):
        """Initialize NLP models for NER and classification"""
        self.ner_model = None
        self.pii_detector = None
        
        if not TRANSFORMERS_AVAILABLE:
            safe_print("   [WARN]  Transformers not available, using regex only")
            return
        
        try:
            safe_print("   Loading NER model...")
            # Load pre-trained NER model for PII detection
            self.ner_model = pipeline(
                "ner",
                model="dslim/bert-base-NER",  # Fast and accurate
                device=0 if self.device == 'cuda' else -1,
                aggregation_strategy="simple"
            )
            safe_print("   [OK] NER model loaded")
            
            # Load PII-specific model if available
            try:
                safe_print("   Loading PII detection model...")
                self.pii_detector = pipeline(
                    "token-classification",
                    model="lakshyakh93/deberta_finetuned_pii",
                    device=0 if self.device == 'cuda' else -1,
                    aggregation_strategy="simple"
                )
                safe_print("   [OK] PII detector loaded")
            except:
                safe_print("   [WARN]  PII detector not available (using NER only)")
                self.pii_detector = None
            
        except Exception as e:
            safe_print(f"   [WARN]  Could not load NLP models: {e}")
            self.ner_model = None
            self.pii_detector = None
    
    def _init_regex_patterns(self):
        """Initialize regex patterns as fallback"""
        self.regex_patterns = {
            'credit_card': {
                'pattern': r'\b(?:\d{4}[-\s]?){3}\d{4}\b',
                'severity': 'critical'
            },
            'ssn': {
                'pattern': r'\b\d{3}-\d{2}-\d{4}\b',
                'severity': 'critical'
            },
            'email': {
                'pattern': r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b',
                'severity': 'medium'
            },
            'phone': {
                'pattern': r'\b\d{3}-\d{3}-\d{4}\b',
                'severity': 'medium'
            },
            'ip_address': {
                'pattern': r'\b(?:\d{1,3}\.){3}\d{1,3}\b',
                'severity': 'medium'
            },
            # API Keys with high specificity
            'stripe_key': {
                'pattern': r'\b(sk_(?:live|test)_[A-Za-z0-9]{24,})\b',
                'severity': 'critical'
            },
            'aws_key': {
                'pattern': r'\b(AKIA[0-9A-Z]{16})\b',
                'severity': 'critical'
            },
            'openai_key': {
                'pattern': r'\b(sk-[A-Za-z0-9]{48}|sk-proj-[A-Za-z0-9\-]{48,})\b',
                'severity': 'critical'
            },
            'github_token': {
                'pattern': r'\b(ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{82})\b',
                'severity': 'critical'
            },
            'google_api': {
                'pattern': r'\b(AIza[0-9A-Za-z\-_]{35})\b',
                'severity': 'critical'
            },
            'jwt_token': {
                'pattern': r'\beyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b',
                'severity': 'high'
            },
            # Generic patterns with context
            'api_key_assignment': {
                'pattern': r'(?i)(api[_-]?key|apikey)\s*[:=]\s*[\'"]?([A-Za-z0-9_\-]{20,})[\'"]?',
                'severity': 'high'
            },
            'secret_assignment': {
                'pattern': r'(?i)(secret|password|passwd|pwd|token)\s*[:=]\s*[\'"]?([A-Za-z0-9_\-@!#$%]{8,})[\'"]?',
                'severity': 'high'
            },
        }
    
    def extract_text_advanced(self, image: np.ndarray) -> Tuple[str, List[Dict], float]:
        """
        Advanced text extraction with bounding boxes
        
        Returns:
            (full_text, detections_with_boxes, overall_confidence)
        """
        if self.ocr_method == 'easyocr' and self.ocr_reader:
            # EasyOCR provides bounding boxes and confidence
            results = self.ocr_reader.readtext(image)
            
            full_text = " ".join([text for (bbox, text, conf) in results])
            detections = [
                {
                    'text': text,
                    'bbox': bbox,
                    'confidence': conf
                }
                for (bbox, text, conf) in results
            ]
            
            avg_confidence = np.mean([conf for (_, _, conf) in results]) if results else 0.0
            
            return full_text, detections, avg_confidence
        
        else:
            # Fallback to Tesseract
            import pytesseract
            if CV2_AVAILABLE:
                gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
                text = pytesseract.image_to_string(gray)
            else:
                text = pytesseract.image_to_string(image)
            return text, [], 0.8  # Assume 80% confidence for tesseract
    
    def detect_with_ner(self, text: str) -> List[MLDetection]:
        """Use NER models to detect PII"""
        detections = []
        
        if not self.ner_model and not self.pii_detector:
            return detections
        
        # Try PII-specific detector first
        models_to_try = [
            (self.pii_detector, 'pii_detector'),
            (self.ner_model, 'ner')
        ]
        
        for model, model_name in models_to_try:
            if not model:
                continue
            
            try:
                entities = model(text)
                
                for entity in entities:
                    entity_type = entity['entity_group'].upper()
                    
                    # Map NER entities to our severity levels
                    severity_map = {
                        'PER': 'medium',      # Person name
                        'PERSON': 'medium',
                        'ORG': 'low',         # Organization
                        'LOC': 'low',         # Location
                        'EMAIL': 'high',
                        'PHONE': 'high',
                        'PHONE_NUM': 'high',
                        'ID': 'critical',     # ID numbers
                        'CREDIT_CARD': 'critical',
                        'SSN': 'critical',
                        'PASSWORD': 'critical',
                        'SECRET': 'critical',
                        'KEY': 'critical',
                    }
                    
                    severity = severity_map.get(entity_type, 'medium')
                    
                    detection = MLDetection(
                        rule_name=f"ner_{entity_type.lower()}",
                        matched_text=entity['word'],
                        severity=severity,
                        confidence=entity['score'],
                        detection_method=model_name,
                        timestamp=datetime.now(),
                        context=text[max(0, entity['start']-30):min(len(text), entity['end']+30)]
                    )
                    detections.append(detection)
                    
            except Exception as e:
                print(f"NER detection error: {e}")
                continue
        
        return detections
    
    def detect_with_regex(self, text: str) -> List[MLDetection]:
        """Regex-based detection with high-specificity patterns"""
        detections = []
        
        for rule_name, rule_info in self.regex_patterns.items():
            pattern = rule_info['pattern']
            severity = rule_info['severity']
            
            matches = re.finditer(pattern, text, re.IGNORECASE)
            
            for match in matches:
                matched_text = match.group(0)
                
                # Skip obvious placeholders
                if self._is_placeholder(matched_text):
                    continue
                
                detection = MLDetection(
                    rule_name=rule_name,
                    matched_text=matched_text,
                    severity=severity,
                    confidence=0.95,  # High confidence for regex
                    detection_method='regex',
                    timestamp=datetime.now(),
                    context=text[max(0, match.start()-30):min(len(text), match.end()+30)]
                )
                detections.append(detection)
        
        return detections
    
    def _is_placeholder(self, text: str) -> bool:
        """Check if text is a placeholder"""
        placeholders = [
            'your_', 'example', 'test', 'sample', 'placeholder',
            'xxx', '000', '111', '123', 'abc', 'dummy'
        ]
        text_lower = text.lower()
        
        # Check for placeholder keywords
        if any(p in text_lower for p in placeholders):
            return True
        
        # Check for repetitive patterns
        if len(set(text)) < 4 and len(text) > 10:
            return True
        
        return False
    
    def detect_visual_patterns(self, image: np.ndarray) -> List[MLDetection]:
        """
        Detect sensitive patterns visually (without OCR)
        Useful for detecting masked credit cards (****-****-****-1234)
        """
        detections = []
        if not CV2_AVAILABLE:
            return detections
        
        # Convert to grayscale
        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        
        # Detect credit card-like patterns (groups of 4 digits)
        # This is a simplified example - real implementation would use CNN
        # to detect visual patterns of sensitive data
        
        return detections
    
    def scan_image(self, 
                   image: np.ndarray,
                   use_ner: bool = True,
                   use_regex: bool = True,
                   min_confidence: float = 0.5) -> Dict:
        """
        Main scanning function - uses ensemble of methods
        
        Args:
            image: Input image to scan
            use_ner: Use NER models
            use_regex: Use regex patterns
            min_confidence: Minimum confidence threshold
            
        Returns:
            Dictionary with detections and metadata
        """
        start_time = datetime.now()
        
        # Step 1: Extract text with OCR
        text, ocr_detections, ocr_confidence = self.extract_text_advanced(image)
        
        all_detections = []
        
        # Step 2: NER-based detection
        if use_ner and (self.ner_model or self.pii_detector):
            ner_detections = self.detect_with_ner(text)
            all_detections.extend(ner_detections)
        
        # Step 3: Regex-based detection
        if use_regex:
            regex_detections = self.detect_with_regex(text)
            all_detections.extend(regex_detections)
        
        # Step 4: Visual pattern detection (future enhancement)
        # visual_detections = self.detect_visual_patterns(image)
        # all_detections.extend(visual_detections)
        
        # Step 5: Deduplicate and filter by confidence
        filtered_detections = self._deduplicate_detections(all_detections, min_confidence)
        
        # Calculate processing time
        processing_time = (datetime.now() - start_time).total_seconds()
        
        return {
            'detections': filtered_detections,
            'extracted_text': text,
            'ocr_confidence': ocr_confidence,
            'ocr_method': self.ocr_method,
            'total_detections': len(filtered_detections),
            'processing_time': processing_time,
            'methods_used': {
                'ner': use_ner and (self.ner_model is not None),
                'regex': use_regex,
                'ocr': True
            }
        }
    
    def _deduplicate_detections(self, 
                                detections: List[MLDetection],
                                min_confidence: float) -> List[MLDetection]:
        """Remove duplicate detections and filter by confidence"""
        
        # Filter by confidence
        filtered = [d for d in detections if d.confidence >= min_confidence]
        
        # Group by matched text and keep highest confidence
        seen = {}
        for detection in filtered:
            key = detection.matched_text.lower().strip()
            
            if key not in seen or detection.confidence > seen[key].confidence:
                seen[key] = detection
        
        return list(seen.values())
    
    def scan_text_only(self, text: str) -> Dict:
        """
        Scan text directly without image processing
        Useful for clipboard monitoring or text analysis
        """
        start_time = datetime.now()
        
        all_detections = []
        
        # NER detection
        if self.ner_model or self.pii_detector:
            ner_detections = self.detect_with_ner(text)
            all_detections.extend(ner_detections)
        
        # Regex detection
        regex_detections = self.detect_with_regex(text)
        all_detections.extend(regex_detections)
        
        # Deduplicate
        filtered_detections = self._deduplicate_detections(all_detections, 0.5)
        
        processing_time = (datetime.now() - start_time).total_seconds()
        
        return {
            'detections': filtered_detections,
            'total_detections': len(filtered_detections),
            'processing_time': processing_time
        }


def display_ml_results(results: Dict):
    """Display scan results in a formatted way"""
    
    safe_print("\n" + "="*80)
    safe_print("[SCAN] ML SCANNER RESULTS")
    safe_print("="*80)
    
    safe_print(f"\n[STATS] Scan Metadata:")
    safe_print(f"   Processing Time: {results['processing_time']:.3f}s")
    if 'ocr_confidence' in results:
        safe_print(f"   OCR Confidence: {results['ocr_confidence']:.2%}")
        safe_print(f"   OCR Method: {results['ocr_method']}")
    if 'methods_used' in results:
        methods = results['methods_used']
        safe_print(f"   Methods: NER={'[OK]' if methods['ner'] else '[X]'}, "
              f"Regex={'[OK]' if methods['regex'] else '[X]'}, "
              f"OCR={'[OK]' if methods['ocr'] else '[X]'}")
    
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
    
    # Summary by severity
    severity_counts = {}
    for d in detections:
        severity_counts[d.severity] = severity_counts.get(d.severity, 0) + 1
    
    if severity_counts:
        print("\n📈 Summary by Severity:")
        for severity in ['critical', 'high', 'medium', 'low']:
            if severity in severity_counts:
                print(f"   {severity.capitalize()}: {severity_counts[severity]}")


# Demo function
def demo_ml_scanner():
    """Demonstrate the ML scanner"""
    print("\n" + "="*80)
    print("🚀 MODERN ML SCANNER DEMO")
    print("="*80)
    
    # Initialize scanner
    scanner = ModernMLScanner(use_gpu=True, model_size='medium')
    
    # Test cases
    test_texts = [
        {
            'name': 'API Keys in .env file',
            'text': '''
STRIPE_API_KEY=sk_live_51H8v9K2eZvKYlo2C9K2eZvKYlo2C
OPENAI_API_KEY=sk-proj-abcd1234efgh5678ijkl90mnopqrstuvwxyz1234567890
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
DATABASE_PASSWORD=MySecretPassword123!
            '''
        },
        {
            'name': 'Credit Card and PII',
            'text': '''
Customer: John Smith
Email: john.smith@company.com
Phone: 555-123-4567
Credit Card: 4532-1234-5678-9012
SSN: 123-45-6789
            '''
        },
        {
            'name': 'GitHub Token',
            'text': '''
export GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz
git clone https://github.com/user/repo.git
            '''
        }
    ]
    
    for test in test_texts:
        print(f"\n\n{'='*80}")
        print(f"TEST: {test['name']}")
        print('='*80)
        print(f"Input text preview: {test['text'][:100]}...")
        
        results = scanner.scan_text_only(test['text'])
        display_ml_results(results)


if __name__ == "__main__":
    demo_ml_scanner()
