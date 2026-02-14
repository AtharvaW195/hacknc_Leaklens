package detector

import "testing"

func TestJWTRule_Name(t *testing.T) {
	rule := NewJWTRule()
	if rule.Name() != "jwt_token" {
		t.Errorf("Expected name 'jwt_token', got '%s'", rule.Name())
	}
}

func TestJWTRule_DetectJWT(t *testing.T) {
	rule := NewJWTRule()
	
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	
	findings := rule.Analyze(jwt)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find JWT, got 0 findings")
	}
	
	finding := findings[0]
	if finding.Type != "jwt_token" {
		t.Errorf("Expected type 'jwt_token', got '%s'", finding.Type)
	}
	if finding.Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", finding.Severity)
	}
	if finding.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", finding.Confidence)
	}
	if finding.RawMatch != jwt {
		t.Errorf("Expected RawMatch to match input JWT")
	}
}

func TestJWTRule_DetectJWTInCode(t *testing.T) {
	rule := NewJWTRule()
	
	code := `const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U";`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find JWT, got 0 findings")
	}
	
	if findings[0].LineNumber != 1 {
		t.Errorf("Expected line number 1, got %d", findings[0].LineNumber)
	}
}

func TestJWTRule_DetectMultipleJWTs(t *testing.T) {
	rule := NewJWTRule()
	
	text := `token1 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
token2 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI5ODc2NTQzMjEwIn0.abc123def456ghi789jkl012mno345pqr678stu901vwx234"`
	
	findings := rule.Analyze(text)
	
	if len(findings) != 2 {
		t.Errorf("Expected 2 findings, got %d", len(findings))
	}
}

func TestJWTRule_NoFalsePositives(t *testing.T) {
	rule := NewJWTRule()
	
	text := "This is just regular text with no JWT tokens"
	findings := rule.Analyze(text)
	
	if len(findings) != 0 {
		t.Errorf("Expected no findings, got %d", len(findings))
	}
}

func TestJWTRule_InvalidJWTFormat(t *testing.T) {
	rule := NewJWTRule()
	
	// Invalid: only 2 parts instead of 3
	text := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"
	
	findings := rule.Analyze(text)
	
	if len(findings) != 0 {
		t.Errorf("Expected no findings for invalid JWT format, got %d", len(findings))
	}
}

