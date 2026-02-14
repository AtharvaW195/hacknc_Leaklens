package detector

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedact_TokenHeuristicsMasksMajority(t *testing.T) {
	// Long token should mask majority
	finding := Finding{
		Type:       "token_heuristics",
		Severity:   "high",
		Confidence: "high",
		Reason:     "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp",
		LineNumber: 1,
		RawMatch:   "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp",
	}

	redacted := finding.Redact()

	// Should show only first 4 and last 4, masking the majority
	expected := "AbCd...MnOp"
	if redacted.Reason != expected {
		t.Errorf("Expected redacted reason '%s', got '%s'", expected, redacted.Reason)
	}

	// Verify majority is masked (more than 50%)
	originalLen := len(finding.RawMatch)
	maskedLen := len(strings.ReplaceAll(redacted.Reason, "...", ""))
	visibleChars := maskedLen - 2 // subtract the "..." if present
	if visibleChars > originalLen/2 {
		t.Errorf("Expected majority of token to be masked, but %d/%d chars visible", visibleChars, originalLen)
	}
}

func TestRedact_TokenHeuristicsMediumLength(t *testing.T) {
	finding := Finding{
		Type:       "token_heuristics",
		Severity:   "medium",
		Confidence: "medium",
		Reason:     "AbCdEfGhIjKlMnOpQrSt",
		LineNumber: 1,
		RawMatch:   "AbCdEfGhIjKlMnOpQrSt",
	}

	redacted := finding.Redact()

	// For medium tokens (20 chars), should show first 3 and last 3
	expected := "AbCd...QrSt"
	if redacted.Reason != expected {
		t.Errorf("Expected redacted reason '%s', got '%s'", expected, redacted.Reason)
	}
}

func TestRedact_TokenHeuristicsShort(t *testing.T) {
	finding := Finding{
		Type:       "token_heuristics",
		Severity:   "medium",
		Confidence: "medium",
		Reason:     "AbCdEfGh",
		LineNumber: 1,
		RawMatch:   "AbCdEfGh",
	}

	redacted := finding.Redact()

	// For short tokens (8 chars), should show first 2 and last 2
	expected := "Ab...Gh"
	if redacted.Reason != expected {
		t.Errorf("Expected redacted reason '%s', got '%s'", expected, redacted.Reason)
	}
}

func TestRedact_TokenHeuristicsJSONDoesNotLeak(t *testing.T) {
	finding := Finding{
		Type:       "token_heuristics",
		Severity:   "high",
		Confidence: "high",
		Reason:     "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp",
		LineNumber: 1,
		RawMatch:   "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp",
	}

	redacted := finding.Redact()

	// Marshal to JSON
	jsonData, err := json.Marshal(redacted)
	if err != nil {
		t.Fatalf("Failed to marshal finding: %v", err)
	}

	jsonStr := string(jsonData)

	// Verify full token is not in JSON
	if strings.Contains(jsonStr, finding.RawMatch) {
		t.Error("JSON output should not contain the full raw token")
	}

	// Verify majority is masked
	if strings.Count(jsonStr, "...") == 0 {
		t.Error("JSON output should contain redaction markers")
	}

	// Count visible characters in redacted version
	redactedToken := redacted.Reason
	visibleChars := len(strings.ReplaceAll(redactedToken, "...", ""))
	originalLen := len(finding.RawMatch)

	// Majority should be masked (visible chars < 50% of original)
	if visibleChars >= originalLen/2 {
		t.Errorf("Expected majority to be masked, but %d/%d chars visible", visibleChars, originalLen)
	}
}

func TestRedact_TokenHeuristicsEngineRedacts(t *testing.T) {
	engine := NewEngine()

	text := `token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp"`

	result := engine.Analyze(text)

	// Marshal entire result to JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	jsonStr := string(jsonData)

	// Verify no full tokens are leaked
	if strings.Contains(jsonStr, "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp") {
		t.Error("JSON output should not contain full token")
	}

	// Verify redacted version is present
	if !strings.Contains(jsonStr, "...") {
		t.Error("JSON output should contain redacted version with ...")
	}

	// Find token_heuristics findings
	for _, finding := range result.Findings {
		if finding.Type == "token_heuristics" {
			// Verify majority is masked
			visibleChars := len(strings.ReplaceAll(finding.Reason, "...", ""))
			// For a 50-char token, we should show max 8 chars (4+4)
			if visibleChars > 10 {
				t.Errorf("Expected majority of token to be masked, but %d chars visible in '%s'", visibleChars, finding.Reason)
			}
		}
	}
}

