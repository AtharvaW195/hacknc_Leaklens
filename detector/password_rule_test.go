package detector

import "testing"

func TestPasswordRule_Name(t *testing.T) {
	rule := NewPasswordRule()
	if rule.Name() != "password_assignment" {
		t.Errorf("Expected name 'password_assignment', got '%s'", rule.Name())
	}
}

func TestPasswordRule_DetectPasswordAssignment(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `password = "mySecretPassword123"`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find password assignment, got 0 findings")
	}
	
	finding := findings[0]
	if finding.Type != "password_assignment" {
		t.Errorf("Expected type 'password_assignment', got '%s'", finding.Type)
	}
	if finding.Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", finding.Severity)
	}
	if finding.Confidence != "medium" {
		t.Errorf("Expected confidence 'medium', got '%s'", finding.Confidence)
	}
	if finding.RawMatch != "mySecretPassword123" {
		t.Errorf("Expected RawMatch 'mySecretPassword123', got '%s'", finding.RawMatch)
	}
}

func TestPasswordRule_DetectAPIKey(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `api_key = "sk-1234567890abcdef"`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find API key, got 0 findings")
	}
	
	if findings[0].Type != "password_assignment" {
		t.Errorf("Expected type 'password_assignment', got '%s'", findings[0].Type)
	}
}

func TestPasswordRule_DetectSecret(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `secret = "super_secret_value_12345"`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find secret, got 0 findings")
	}
}

func TestPasswordRule_DetectPasswd(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `passwd = "mypassword"`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find passwd, got 0 findings")
	}
}

func TestPasswordRule_DetectColonSyntax(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `password: "secret123"`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find password with colon syntax, got 0 findings")
	}
}

func TestPasswordRule_IgnoreComments(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `// password = "commented out"`
	
	findings := rule.Analyze(code)
	
	if len(findings) != 0 {
		t.Errorf("Expected no findings in comments, got %d", len(findings))
	}
}

func TestPasswordRule_IgnoreAlreadyRedacted(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `password = "abcd...xyz"`
	
	findings := rule.Analyze(code)
	
	if len(findings) != 0 {
		t.Errorf("Expected no findings for already redacted values, got %d", len(findings))
	}
}

func TestPasswordRule_LineNumber(t *testing.T) {
	rule := NewPasswordRule()
	
	code := `line 1
line 2
password = "secret123"
line 4`
	
	findings := rule.Analyze(code)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find password, got 0 findings")
	}
	
	if findings[0].LineNumber != 3 {
		t.Errorf("Expected line number 3, got %d", findings[0].LineNumber)
	}
}

