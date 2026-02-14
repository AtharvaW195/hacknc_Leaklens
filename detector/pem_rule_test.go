package detector

import "testing"

func TestPEMRule_Name(t *testing.T) {
	rule := NewPEMRule()
	if rule.Name() != "pem_private_key" {
		t.Errorf("Expected name 'pem_private_key', got '%s'", rule.Name())
	}
}

func TestPEMRule_DetectRSAPrivateKey(t *testing.T) {
	rule := NewPEMRule()
	
	pemKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v
Z8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ
8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v
Z8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8
-----END RSA PRIVATE KEY-----`
	
	findings := rule.Analyze(pemKey)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find PEM private key, got 0 findings")
	}
	
	finding := findings[0]
	if finding.Type != "pem_private_key" {
		t.Errorf("Expected type 'pem_private_key', got '%s'", finding.Type)
	}
	if finding.Severity != "high" {
		t.Errorf("Expected severity 'high', got '%s'", finding.Severity)
	}
	if finding.Confidence != "high" {
		t.Errorf("Expected confidence 'high', got '%s'", finding.Confidence)
	}
	if finding.LineNumber != 1 {
		t.Errorf("Expected line number 1, got %d", finding.LineNumber)
	}
	if finding.RawMatch == "" {
		t.Error("Expected RawMatch to be set")
	}
}

func TestPEMRule_DetectECPrivateKey(t *testing.T) {
	rule := NewPEMRule()
	
	pemKey := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIAKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKj
-----END EC PRIVATE KEY-----`
	
	findings := rule.Analyze(pemKey)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find EC private key, got 0 findings")
	}
	
	finding := findings[0]
	if finding.Type != "pem_private_key" {
		t.Errorf("Expected type 'pem_private_key', got '%s'", finding.Type)
	}
}

func TestPEMRule_DetectGenericPrivateKey(t *testing.T) {
	rule := NewPEMRule()
	
	pemKey := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKj
-----END PRIVATE KEY-----`
	
	findings := rule.Analyze(pemKey)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find private key, got 0 findings")
	}
}

func TestPEMRule_NoFalsePositives(t *testing.T) {
	rule := NewPEMRule()
	
	text := "This is just regular text with no private keys"
	findings := rule.Analyze(text)
	
	if len(findings) != 0 {
		t.Errorf("Expected no findings, got %d", len(findings))
	}
}

func TestPEMRule_DetectInMultiLineText(t *testing.T) {
	rule := NewPEMRule()
	
	text := `Some code here
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v
Z8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ
-----END RSA PRIVATE KEY-----
More code here`
	
	findings := rule.Analyze(text)
	
	if len(findings) == 0 {
		t.Fatal("Expected to find PEM private key, got 0 findings")
	}
	
	if findings[0].LineNumber != 2 {
		t.Errorf("Expected line number 2, got %d", findings[0].LineNumber)
	}
}

