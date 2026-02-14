package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestJSONShape(t *testing.T) {
	// Test with --text flag
	cmd := exec.Command("go", "run", ".", "--text", "test input")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify JSON structure
	if result.OverallRisk == "" {
		t.Error("overall_risk field is missing or empty")
	}

	if result.RiskRationale == "" {
		t.Error("risk_rationale field is missing or empty")
	}

	if result.Findings == nil {
		t.Error("findings field is missing or nil")
	}
}

func TestCLIExitCode(t *testing.T) {
	// Test with --text flag
	cmd := exec.Command("go", "run", ".", "--text", "test input")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0, got: %v", err)
	}
}

func TestCLIStdin(t *testing.T) {
	// Test with stdin
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader("test input from stdin")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify JSON structure
	if result.OverallRisk == "" {
		t.Error("overall_risk field is missing or empty")
	}
}

func TestCLIExitCodeStdin(t *testing.T) {
	// Test exit code with stdin
	cmd := exec.Command("go", "run", ".")
	cmd.Stdin = strings.NewReader("test input")
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Expected exit code 0, got: %v", err)
	}
}

func TestCLIEmptyString(t *testing.T) {
	// Test with empty string using --text=
	cmd := exec.Command("go", "run", ".", "--text=")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Empty string should return low risk with no findings
	if result.OverallRisk != "low" {
		t.Errorf("Expected overall_risk 'low' for empty string, got '%s'", result.OverallRisk)
	}

	if len(result.Findings) != 0 {
		t.Errorf("Expected no findings for empty string, got %d", len(result.Findings))
	}
}

func TestCLIPasswordDetection(t *testing.T) {
	// Test password detection through CLI
	cmd := exec.Command("go", "run", ".", "--text", `password = "secret123"`)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify overall risk is HIGH
	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high', got '%s'", result.OverallRisk)
	}

	// Verify we have findings
	if len(result.Findings) == 0 {
		t.Fatal("Expected at least one finding, got 0")
	}

	// Verify all metrics for password finding
	foundPassword := false
	for _, finding := range result.Findings {
		if finding.Type == "password_assignment" {
			foundPassword = true
			
			// Verify all required fields
			if finding.Type != "password_assignment" {
				t.Errorf("Expected type 'password_assignment', got '%s'", finding.Type)
			}
			if finding.Severity != "high" {
				t.Errorf("Expected severity 'high', got '%s'", finding.Severity)
			}
			if finding.Confidence != "medium" {
				t.Errorf("Expected confidence 'medium', got '%s'", finding.Confidence)
			}
			if finding.Reason == "" {
				t.Error("Expected reason to be set (redacted)")
			}
			if finding.LineNumber == 0 {
				t.Error("Expected line_number to be > 0")
			}
			
			// Verify reason is redacted (contains ...)
			if !strings.Contains(finding.Reason, "...") {
				t.Errorf("Expected reason to be redacted (contain ...), got '%s'", finding.Reason)
			}
			
			// Verify full secret is NOT in output
			if strings.Contains(string(output), "secret123") {
				t.Error("Full secret 'secret123' should not appear in JSON output")
			}
		}
	}

	if !foundPassword {
		t.Error("Expected to find password_assignment finding")
	}
}

func TestCLIPasswordDetectionWithoutQuotes(t *testing.T) {
	// Test password detection when quotes are stripped (PowerShell behavior)
	cmd := exec.Command("go", "run", ".", "--text", "password = secret123")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should detect password even without quotes
	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for password without quotes, got '%s'", result.OverallRisk)
	}

	if len(result.Findings) == 0 {
		t.Error("Expected to find password even without quotes")
	}
}

func TestCLIPEMDetection(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--text", "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for PEM key, got '%s'", result.OverallRisk)
	}

	foundPEM := false
	for _, finding := range result.Findings {
		if finding.Type == "pem_private_key" {
			foundPEM = true
			if finding.Severity != "high" {
				t.Errorf("Expected severity 'high', got '%s'", finding.Severity)
			}
			if finding.Confidence != "high" {
				t.Errorf("Expected confidence 'high', got '%s'", finding.Confidence)
			}
			if finding.LineNumber == 0 {
				t.Error("Expected line_number to be > 0")
			}
			if !strings.Contains(finding.Reason, "...") {
				t.Error("Expected PEM to be redacted")
			}
		}
	}

	if !foundPEM {
		t.Error("Expected to find pem_private_key finding")
	}
}

func TestCLIJWTDetection(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--text", `token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for JWT, got '%s'", result.OverallRisk)
	}

	foundJWT := false
	for _, finding := range result.Findings {
		if finding.Type == "jwt_token" {
			foundJWT = true
			if finding.Severity != "high" {
				t.Errorf("Expected severity 'high', got '%s'", finding.Severity)
			}
			if finding.Confidence != "high" {
				t.Errorf("Expected confidence 'high', got '%s'", finding.Confidence)
			}
			if finding.LineNumber == 0 {
				t.Error("Expected line_number to be > 0")
			}
			if !strings.Contains(finding.Reason, "...") {
				t.Error("Expected JWT to be redacted")
			}
			// Verify full JWT is not in output
			if strings.Contains(string(output), "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
				t.Error("Full JWT should not appear in output")
			}
		}
	}

	if !foundJWT {
		t.Error("Expected to find jwt_token finding")
	}
}

func TestCLITokenHeuristicsDetection(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--text", `token = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890+/=AbCdEfGhIjKlMnOp"`)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Should detect as token_heuristics (might be HIGH or MEDIUM depending on score)
	if result.OverallRisk == "low" {
		t.Errorf("Expected overall_risk to be 'high' or 'medium' for token heuristics, got '%s'", result.OverallRisk)
	}

	foundTokenHeuristics := false
	for _, finding := range result.Findings {
		if finding.Type == "token_heuristics" {
			foundTokenHeuristics = true
			if finding.Severity != "high" && finding.Severity != "medium" {
				t.Errorf("Expected severity 'high' or 'medium', got '%s'", finding.Severity)
			}
			if finding.Confidence == "" {
				t.Error("Expected confidence to be set")
			}
			if finding.LineNumber == 0 {
				t.Error("Expected line_number to be > 0")
			}
			if !strings.Contains(finding.Reason, "...") {
				t.Error("Expected token to be redacted (majority masked)")
			}
		}
	}

	if !foundTokenHeuristics {
		t.Error("Expected to find token_heuristics finding")
	}
}

func TestCLIRedactionNoSecretsLeak(t *testing.T) {
	// Test that full secrets never appear in output
	testCases := []struct {
		name    string
		input   string
		secret  string
	}{
		{"password", `password = "mySecretPassword123"`, "mySecretPassword123"},
		{"jwt", `token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"api_key", `api_key = "sk-1234567890abcdef"`, "sk-1234567890abcdef"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", "--text", tc.input)
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Command failed: %v", err)
			}

			outputStr := string(output)
			if strings.Contains(outputStr, tc.secret) {
				t.Errorf("Full secret '%s' should not appear in output", tc.secret)
			}

			// Verify redacted version is present
			if !strings.Contains(outputStr, "...") {
				t.Error("Expected redacted version with '...' to be present")
			}
		})
	}
}

func TestCLIRiskScoring(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		expectedRisk string
	}{
		{"password", `password = "secret123"`, "high"},
		{"jwt", `token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`, "high"},
		{"pem", "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----", "high"},
		{"regular text", "This is just regular text", "low"},
		{"uuid", `uuid = "550e8400-e29b-41d4-a716-446655440000"`, "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", "--text", tc.input)
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Command failed: %v", err)
			}

			var result Result
			if err := json.Unmarshal(output, &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if result.OverallRisk != tc.expectedRisk {
				t.Errorf("Expected overall_risk '%s', got '%s'", tc.expectedRisk, result.OverallRisk)
			}
		})
	}
}

func TestCLIAllFindingMetrics(t *testing.T) {
	// Test that all findings have all required metrics
	cmd := exec.Command("go", "run", ".", "--text", `password = "secret123"`)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(result.Findings) == 0 {
		t.Fatal("Expected at least one finding")
	}

	for i, finding := range result.Findings {
		// Verify all required fields are present
		if finding.Type == "" {
			t.Errorf("Finding %d: type field is missing", i)
		}
		if finding.Severity == "" {
			t.Errorf("Finding %d: severity field is missing", i)
		}
		if finding.Confidence == "" {
			t.Errorf("Finding %d: confidence field is missing", i)
		}
		if finding.Reason == "" {
			t.Errorf("Finding %d: reason field is missing", i)
		}
		if finding.LineNumber == 0 {
			t.Errorf("Finding %d: line_number field is missing or 0", i)
		}

		// Verify valid values
		validSeverities := map[string]bool{"high": true, "medium": true, "low": true}
		if !validSeverities[finding.Severity] {
			t.Errorf("Finding %d: invalid severity '%s'", i, finding.Severity)
		}

		validConfidences := map[string]bool{"high": true, "medium": true, "low": true}
		if !validConfidences[finding.Confidence] {
			t.Errorf("Finding %d: invalid confidence '%s'", i, finding.Confidence)
		}
	}
}

func TestCLIMultipleRulesDetection(t *testing.T) {
	// Test that multiple rules can detect different secrets
	cmd := exec.Command("go", "run", ".", "--text", `password = "secret123"`+"\n"+`token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"`)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	var result Result
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high' for multiple secrets, got '%s'", result.OverallRisk)
	}

	if len(result.Findings) < 2 {
		t.Errorf("Expected at least 2 findings, got %d", len(result.Findings))
	}

	// Verify we have both password and JWT findings
	hasPassword := false
	hasJWT := false
	for _, finding := range result.Findings {
		if finding.Type == "password_assignment" {
			hasPassword = true
		}
		if finding.Type == "jwt_token" {
			hasJWT = true
		}
	}

	if !hasPassword {
		t.Error("Expected to find password_assignment")
	}
	if !hasJWT {
		t.Error("Expected to find jwt_token")
	}
}

func TestMain(m *testing.M) {
	// Build the binary first to ensure it compiles
	buildCmd := exec.Command("go", "build", "-o", "pasteguard.exe", ".")
	if err := buildCmd.Run(); err != nil {
		// If build fails, try without .exe extension (for non-Windows)
		buildCmd = exec.Command("go", "build", "-o", "pasteguard", ".")
		if err := buildCmd.Run(); err != nil {
			os.Exit(1)
		}
	}
	
	code := m.Run()
	os.Exit(code)
}

