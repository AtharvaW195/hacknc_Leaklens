package detector

import "testing"

func TestEngineRiskScoring_HighRisk(t *testing.T) {
	engine := NewEngine()
	
	// Text with a high severity finding (JWT)
	text := `token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	
	result := engine.Analyze(text)
	
	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' when high severity finding exists, got '%s'", result.OverallRisk)
	}
	
	if result.RiskRationale != "High severity issues detected" {
		t.Errorf("Expected risk_rationale 'High severity issues detected', got '%s'", result.RiskRationale)
	}
}

func TestEngineRiskScoring_PEMKey(t *testing.T) {
	engine := NewEngine()
	
	text := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v
-----END RSA PRIVATE KEY-----`
	
	result := engine.Analyze(text)
	
	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for PEM key, got '%s'", result.OverallRisk)
	}
}

func TestEngineRiskScoring_PasswordAssignment(t *testing.T) {
	engine := NewEngine()
	
	text := `password = "mySecretPassword123"`
	
	result := engine.Analyze(text)
	
	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for password assignment, got '%s'", result.OverallRisk)
	}
}

func TestEngineRiskScoring_MultipleHighFindings(t *testing.T) {
	engine := NewEngine()
	
	text := `password = "secret123"
token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`
	
	result := engine.Analyze(text)
	
	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for multiple high findings, got '%s'", result.OverallRisk)
	}
}

func TestEngineRiskScoring_NoFindings(t *testing.T) {
	engine := NewEngine()
	
	text := "This is just regular text with no secrets"
	
	result := engine.Analyze(text)
	
	if result.OverallRisk != "low" {
		t.Errorf("Expected overall_risk 'low' for no findings, got '%s'", result.OverallRisk)
	}
	
	if result.RiskRationale != "No issues detected" {
		t.Errorf("Expected risk_rationale 'No issues detected', got '%s'", result.RiskRationale)
	}
}

