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

