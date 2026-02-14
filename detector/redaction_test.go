package detector

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedact_LongSecret(t *testing.T) {
	finding := Finding{
		Type:       "password_assignment",
		Severity:   "high",
		Confidence: "medium",
		Reason:     "myVeryLongSecretPassword12345",
		LineNumber: 1,
		RawMatch:   "myVeryLongSecretPassword12345",
	}
	
	redacted := finding.Redact()
	
	if redacted.RawMatch != "" {
		t.Error("Redacted finding should not have RawMatch")
	}
	
	// Should be masked: first 4 + ... + last 4
	expected := "myVe...2345"
	if redacted.Reason != expected {
		t.Errorf("Expected redacted reason '%s', got '%s'", expected, redacted.Reason)
	}
}

func TestRedact_ShortSecret(t *testing.T) {
	finding := Finding{
		Type:       "password_assignment",
		Severity:   "high",
		Confidence: "medium",
		Reason:     "secret",
		LineNumber: 1,
		RawMatch:   "secret",
	}
	
	redacted := finding.Redact()
	
	// For short secrets, should still mask
	if redacted.Reason == "secret" {
		t.Error("Short secret should still be redacted")
	}
}

func TestRedact_NoRawMatch(t *testing.T) {
	finding := Finding{
		Type:       "password_assignment",
		Severity:   "high",
		Confidence: "medium",
		Reason:     "Some reason without raw match",
		LineNumber: 1,
		RawMatch:   "",
	}
	
	redacted := finding.Redact()
	
	if redacted.Reason != finding.Reason {
		t.Errorf("Reason should not change when RawMatch is empty, got '%s'", redacted.Reason)
	}
}

func TestRedact_JSONOutputDoesNotContainRawMatch(t *testing.T) {
	finding := Finding{
		Type:       "jwt_token",
		Severity:   "high",
		Confidence: "high",
		Reason:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		LineNumber: 1,
		RawMatch:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
	}
	
	redacted := finding.Redact()
	
	// Marshal to JSON to verify RawMatch is not included
	jsonData, err := json.Marshal(redacted)
	if err != nil {
		t.Fatalf("Failed to marshal finding: %v", err)
	}
	
	jsonStr := string(jsonData)
	if strings.Contains(jsonStr, "RawMatch") {
		t.Error("JSON output should not contain RawMatch field")
	}
	
	// Verify the full secret is not in the JSON
	if strings.Contains(jsonStr, finding.RawMatch) {
		t.Error("JSON output should not contain the full raw match")
	}
	
	// Verify it contains the redacted version
	if !strings.Contains(jsonStr, "...") {
		t.Error("JSON output should contain redacted version with ...")
	}
}

func TestRedact_EngineRedactsAllFindings(t *testing.T) {
	engine := NewEngine()
	
	text := `password = "mySecretPassword123"
token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	
	result := engine.Analyze(text)
	
	// Marshal entire result to JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}
	
	jsonStr := string(jsonData)
	
	// Verify no full secrets are leaked
	if strings.Contains(jsonStr, "mySecretPassword123") {
		t.Error("JSON output should not contain full password")
	}
	
	if strings.Contains(jsonStr, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U") {
		t.Error("JSON output should not contain full JWT")
	}
	
	// Verify redacted versions are present
	if !strings.Contains(jsonStr, "...") {
		t.Error("JSON output should contain redacted versions")
	}
}

func TestRedact_PEMKey(t *testing.T) {
	finding := Finding{
		Type:       "pem_private_key",
		Severity:   "high",
		Confidence: "high",
		Reason:     "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----",
		LineNumber: 1,
		RawMatch:   "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----",
	}
	
	redacted := finding.Redact()
	
	// Verify the full PEM is not in the reason
	if redacted.Reason == finding.RawMatch {
		t.Error("PEM key should be redacted")
	}
	
	// Verify it's masked
	if !strings.Contains(redacted.Reason, "...") {
		t.Error("Redacted PEM should contain ...")
	}
}

