"""Central PII/sensitive-data catalog used across scanner and monitor scope filters."""

from __future__ import annotations

from typing import Dict


def get_pii_regex_patterns() -> Dict[str, Dict[str, str]]:
    """
    Central regex pattern registry.
    Keep high-signal patterns here so scanner behavior is modular and consistent.
    """
    return {
        # API keys / secrets
        "stripe_live_key": {"pattern": r"\b(sk_live_[A-Za-z0-9]{24,})\b", "severity": "critical"},
        "stripe_test_key": {"pattern": r"\b(sk_test_[A-Za-z0-9]{24,})\b", "severity": "high"},
        "openai_key": {"pattern": r"\b(sk-[A-Za-z0-9]{48}|sk-proj-[A-Za-z0-9\-_]{48,})\b", "severity": "critical"},
        "aws_access_key": {"pattern": r"\b(AKIA[0-9A-Z]{16})\b", "severity": "critical"},
        "github_token": {"pattern": r"\b(ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{82})\b", "severity": "critical"},
        "google_api_key": {"pattern": r"\b(AIza[0-9A-Za-z\-_]{35})\b", "severity": "critical"},
        "slack_token": {"pattern": r"\b(xox[pbar]-[0-9]{10,13}-[0-9]{10,13}-[A-Za-z0-9]{24,})\b", "severity": "critical"},
        "jwt_token": {"pattern": r"\beyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b", "severity": "high"},
        "api_key_assignment": {"pattern": r"(?i)(api[_\-]?key|apikey)\s*[:=]\s*[\'\"]?([A-Za-z0-9_\-]{20,})[\'\"]?", "severity": "critical"},
        "secret_assignment": {"pattern": r"(?i)(secret[_\-]?key|secret|client[_\-]?secret)\s*[:=]\s*[\'\"]?([A-Za-z0-9_\-]{16,})[\'\"]?", "severity": "critical"},
        "token_assignment": {"pattern": r"(?i)(access[_\-]?token|auth[_\-]?token|token)\s*[:=]\s*[\'\"]?([A-Za-z0-9_\-]{20,})[\'\"]?", "severity": "high"},
        "password_assignment": {"pattern": r"(?i)(password|passwd|pwd|passcode)\s*(?:[:=]|is|->)?\s*[\'\"]?([A-Za-z0-9_\-@!#$%^&*().+=]{8,64})[\'\"]?", "severity": "critical"},
        "password_field_html": {
            "pattern": r"(?i)(?:type\s*=\s*[\"']password[\"'][^>\n]{0,120}?value\s*=\s*[\"']([^\"']{8,80})[\"'])|(?:value\s*=\s*[\"']([^\"']{8,80})[\"'][^>\n]{0,120}?type\s*=\s*[\"']password[\"'])",
            "severity": "critical",
        },

        # PII / financial
        "credit_card": {"pattern": r"\b(?:\d{4}[-\s]?){3}\d{4}\b", "severity": "critical"},
        "ssn": {"pattern": r"\b\d{3}-\d{2}-\d{4}\b", "severity": "critical"},
        "email": {"pattern": r"\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b", "severity": "medium"},
        "phone": {"pattern": r"\b\d{3}[-\s]?\d{3}[-\s]?\d{4}\b", "severity": "medium"},
        "private_ip": {"pattern": r"\b(?:10\.|172\.(?:1[6-9]|2[0-9]|3[01])\.|192\.168\.)\d{1,3}\.\d{1,3}\b", "severity": "medium"},
        "iban": {"pattern": r"\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b", "severity": "high"},
        "swift_bic": {"pattern": r"\b[A-Z]{6}[A-Z0-9]{2}(?:[A-Z0-9]{3})?\b", "severity": "medium"},

        # Public/private keys and cryptographic material
        "private_key_pem": {
            "pattern": r"-----BEGIN(?: RSA| EC| DSA| OPENSSH)? PRIVATE KEY-----[\s\S]{80,}?-----END(?: RSA| EC| DSA| OPENSSH)? PRIVATE KEY-----",
            "severity": "critical",
        },
        "pgp_private_key_block": {
            "pattern": r"-----BEGIN PGP PRIVATE KEY BLOCK-----[\s\S]{80,}?-----END PGP PRIVATE KEY BLOCK-----",
            "severity": "critical",
        },
        "public_key_pem": {
            "pattern": r"-----BEGIN PUBLIC KEY-----[\s\S]{50,}?-----END PUBLIC KEY-----",
            "severity": "high",
        },
        "ssh_public_key": {
            "pattern": r"\b(?:ssh-rsa|ssh-ed25519|ecdsa-sha2-nistp(?:256|384|521))\s+[A-Za-z0-9+/]{50,}(?:\s+[^\n\r]+)?",
            "severity": "high",
        },
        "private_key_assignment": {
            "pattern": r"(?i)(private[_\- ]?key)\s*[:=]\s*[\'\"]?([A-Za-z0-9+/=\-_.]{32,})[\'\"]?",
            "severity": "critical",
        },
        "public_key_assignment": {
            "pattern": r"(?i)(public[_\- ]?key)\s*[:=]\s*[\'\"]?([A-Za-z0-9+/=\-_.]{32,})[\'\"]?",
            "severity": "high",
        },
    }


CORE_RULES = {
    "credit_card",
    "credit_card_suspected",
    "ssn",
    "email",
    "phone",
    "stripe_live_key",
    "stripe_test_key",
    "openai_key",
    "aws_access_key",
    "github_token",
    "google_api_key",
    "slack_token",
    "jwt_token",
    "api_key_assignment",
    "secret_assignment",
    "token_assignment",
    "password_assignment",
    "password_field_html",
    "contextual_sensitive_token",
}


FULL_RULES = CORE_RULES | {
    "private_ip",
    "iban",
    "swift_bic",
    "private_key_pem",
    "pgp_private_key_block",
    "public_key_pem",
    "ssh_public_key",
    "private_key_assignment",
    "public_key_assignment",
}


NER_PREFIXES = (
    "ner_email",
    "ner_phone",
    "ner_phone_num",
    "ner_credit_card",
    "ner_ssn",
    "ner_socialsecuritynumber",
    "ner_password",
    "ner_secret",
    "ner_key",
)


def is_rule_in_scope(rule_name: str, focus_profile: str) -> bool:
    profile = (focus_profile or "pii_full").lower()
    rule = (rule_name or "").lower()

    if profile == "all":
        return True

    if rule.startswith(NER_PREFIXES):
        return True

    if profile == "pii_core":
        return rule in CORE_RULES

    # Default to broader full profile.
    return rule in FULL_RULES

