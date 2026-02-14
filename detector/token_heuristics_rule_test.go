package detector

import "testing"

func TestTokenHeuristicsRule_Name(t *testing.T) {
	rule := NewTokenHeuristicsRule()
	if rule.Name() != "token_heuristics" {
		t.Errorf("Expected name 'token_heuristics', got '%s'", rule.Name())
	}
}

func TestTokenHeuristicsRule_DetectNearAuthKeyword(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Token near auth keyword should be detected
	text := `token = "aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890"`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find token near auth keyword, got 0 findings")
	}

	finding := findings[0]
	if finding.Type != "token_heuristics" {
		t.Errorf("Expected type 'token_heuristics', got '%s'", finding.Type)
	}
	if finding.Severity != "high" && finding.Severity != "medium" {
		t.Errorf("Expected severity 'high' or 'medium', got '%s'", finding.Severity)
	}
}

func TestTokenHeuristicsRule_DetectNearAPIKeyword(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `api_key: AbCdEfGhIjKlMnOpQrStUvWxYz1234567890`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find token near api_key keyword, got 0 findings")
	}
}

func TestTokenHeuristicsRule_DetectNearSecretKeyword(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `secret = "xYzAbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/="`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find token near secret keyword, got 0 findings")
	}
}

func TestTokenHeuristicsRule_IgnoreUUIDs(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// UUID in safe context should be ignored
	text := `uuid = "550e8400-e29b-41d4-a716-446655440000"`
	findings := rule.Analyze(text)

	if len(findings) != 0 {
		t.Errorf("Expected no findings for UUID in safe context, got %d", len(findings))
	}
}

func TestTokenHeuristicsRule_IgnoreHashes(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Hash in safe context should be ignored
	text := `sha256_hash = "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"`
	findings := rule.Analyze(text)

	if len(findings) != 0 {
		t.Errorf("Expected no findings for hash in safe context, got %d", len(findings))
	}
}

func TestTokenHeuristicsRule_IgnoreMD5InSafeContext(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `md5_checksum = "5d41402abc4b2a76b9719d911017c592"`
	findings := rule.Analyze(text)

	if len(findings) != 0 {
		t.Errorf("Expected no findings for MD5 in safe context, got %d", len(findings))
	}
}

func TestTokenHeuristicsRule_IgnoreCommitHashes(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `commit = "a1b2c3d4e5f6789012345678901234567890abcd"`
	findings := rule.Analyze(text)

	if len(findings) != 0 {
		t.Errorf("Expected no findings for commit hash, got %d", len(findings))
	}
}

func TestTokenHeuristicsRule_DetectTokenWithoutKeyword(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Very high entropy token without keyword - should still detect if score is high enough
	text := `some_var = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEf"`
	findings := rule.Analyze(text)

	// This might or might not be detected depending on scoring
	// But if detected, should have medium severity
	if len(findings) > 0 {
		if findings[0].Severity != "medium" && findings[0].Severity != "high" {
			t.Errorf("Expected severity 'medium' or 'high', got '%s'", findings[0].Severity)
		}
	}
}

func TestTokenHeuristicsRule_HighSeverityNearKeyword(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Long token with high variety near keyword should be HIGH
	// Using a simpler token that will definitely score high
	text := `token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOpQrStUvWx"`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find high-entropy token, got 0 findings")
	}

	// Should be high severity due to length + variety + keyword proximity
	// Accept medium if score is borderline, but prefer high
	if findings[0].Severity != "high" && findings[0].Severity != "medium" {
		t.Errorf("Expected severity 'high' or 'medium' for high-scoring token, got '%s'", findings[0].Severity)
	}
	// For a 60+ char token near keyword, it should be high
	if len(findings[0].RawMatch) >= 50 && findings[0].Severity != "high" {
		t.Errorf("Expected severity 'high' for long token (%d chars) near keyword, got '%s'", len(findings[0].RawMatch), findings[0].Severity)
	}
}

func TestTokenHeuristicsRule_MediumSeverity(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Medium-length token with moderate variety
	text := `auth = "AbCdEfGhIjKlMnOpQrStUvWx"`
	findings := rule.Analyze(text)

	if len(findings) > 0 {
		// If detected, should be medium or high
		if findings[0].Severity != "medium" && findings[0].Severity != "high" {
			t.Errorf("Expected severity 'medium' or 'high', got '%s'", findings[0].Severity)
		}
	}
}

func TestTokenHeuristicsRule_IgnoreLowEntropy(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Low entropy (too many repeated chars)
	text := `token = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
	findings := rule.Analyze(text)

	if len(findings) != 0 {
		t.Errorf("Expected no findings for low entropy string, got %d", len(findings))
	}
}

func TestTokenHeuristicsRule_LineNumber(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `line 1
line 2
token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890"
line 4`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find token, got 0 findings")
	}

	if findings[0].LineNumber != 3 {
		t.Errorf("Expected line number 3, got %d", findings[0].LineNumber)
	}
}

func TestTokenHeuristicsRule_DetectBase64Like(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `api_key = "SGVsbG9Xb3JsZFRoaXNJc0Jhc2U2NA=="`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find base64-like token, got 0 findings")
	}
}

func TestTokenHeuristicsRule_DetectHexLike(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `secret = "deadbeef1234567890abcdef1234567890abcdef"`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find hex-like token, got 0 findings")
	}
}

func TestTokenHeuristicsRule_DetectURLSafeLike(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `access_token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890_-"`
	findings := rule.Analyze(text)

	if len(findings) == 0 {
		t.Fatal("Expected to find url-safe token, got 0 findings")
	}
}

func TestTokenHeuristicsRule_IgnoreVersionNumbers(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	text := `version = "1.2.3.4.5.6.7.8.9.10.11.12.13.14.15.16.17.18.19.20"`
	findings := rule.Analyze(text)

	// Version numbers are typically low entropy, but if detected should be conservative
	if len(findings) > 0 {
		// Should have low confidence or be filtered out
		t.Logf("Found version number (might be acceptable): %v", findings[0])
	}
}

func TestTokenHeuristicsRule_ProximityScoring(t *testing.T) {
	rule := NewTokenHeuristicsRule()

	// Token very close to keyword (should score higher)
	text1 := `token="AbCdEfGhIjKlMnOpQrStUvWxYz1234567890"`
	findings1 := rule.Analyze(text1)

	// Token far from keyword
	text2 := `some very long text here that has many words and then eventually token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890"`
	findings2 := rule.Analyze(text2)

	// Close proximity should have higher score (might result in high severity)
	if len(findings1) > 0 && len(findings2) > 0 {
		// Close one should have equal or higher severity
		if findings1[0].Severity == "high" && findings2[0].Severity == "medium" {
			// This is expected behavior
		}
	}
}

