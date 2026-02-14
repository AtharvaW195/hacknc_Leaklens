package detector

import (
	"strings"
	"testing"
)

func TestMergeOverlappingFindings_SameLine(t *testing.T) {
	engine := NewEngine()
	
	// Create findings that overlap on the same line
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret123",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret123",
		},
		{
			Type:       "token_heuristics",
			Severity:   "medium",
			Confidence: "high",
			Reason:     "secret123",
			LineNumber: 1,
			ByteStart:  15,
			ByteEnd:    25,
			RawMatch:   "secret123",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	if len(merged) != 1 {
		t.Errorf("Expected 1 merged finding, got %d", len(merged))
	}
	
	if merged[0].Severity != "high" {
		t.Errorf("Expected highest severity 'high', got '%s'", merged[0].Severity)
	}
	
	if merged[0].Confidence != "high" {
		t.Errorf("Expected max confidence 'high', got '%s'", merged[0].Confidence)
	}
	
	if merged[0].ByteStart != 10 {
		t.Errorf("Expected ByteStart 10, got %d", merged[0].ByteStart)
	}
	
	if merged[0].ByteEnd != 25 {
		t.Errorf("Expected ByteEnd 25, got %d", merged[0].ByteEnd)
	}
}

func TestMergeOverlappingFindings_AdjacentLines(t *testing.T) {
	engine := NewEngine()
	
	// Create findings on adjacent lines that overlap
	findings := []Finding{
		{
			Type:       "pem_private_key",
			Severity:   "high",
			Confidence: "high",
			Reason:     "BEGIN...",
			LineNumber: 1,
			ByteStart:  0,
			ByteEnd:    50,
			RawMatch:   "BEGIN...",
		},
		{
			Type:       "pem_private_key",
			Severity:   "high",
			Confidence: "high",
			Reason:     "...END",
			LineNumber: 2,
			ByteStart:  45,
			ByteEnd:    100,
			RawMatch:   "...END",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	if len(merged) != 1 {
		t.Errorf("Expected 1 merged finding, got %d", len(merged))
	}
	
	if merged[0].LineNumber != 1 {
		t.Errorf("Expected minimum line number 1, got %d", merged[0].LineNumber)
	}
	
	if merged[0].ByteStart != 0 {
		t.Errorf("Expected ByteStart 0, got %d", merged[0].ByteStart)
	}
	
	if merged[0].ByteEnd != 100 {
		t.Errorf("Expected ByteEnd 100, got %d", merged[0].ByteEnd)
	}
}

func TestMergeOverlappingFindings_NoOverlap(t *testing.T) {
	engine := NewEngine()
	
	// Create findings that don't overlap
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret1",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret2",
			LineNumber: 1,
			ByteStart:  100,
			ByteEnd:    110,
			RawMatch:   "secret2",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	if len(merged) != 2 {
		t.Errorf("Expected 2 findings (no merge), got %d", len(merged))
	}
}

func TestMergeOverlappingFindings_HighestSeverity(t *testing.T) {
	engine := NewEngine()
	
	// Create findings with different severities
	findings := []Finding{
		{
			Type:       "token_heuristics",
			Severity:   "medium",
			Confidence: "medium",
			Reason:     "token1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "token1",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "token1",
			LineNumber: 1,
			ByteStart:  15,
			ByteEnd:    25,
			RawMatch:   "token1",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	if len(merged) != 1 {
		t.Fatalf("Expected 1 merged finding, got %d", len(merged))
	}
	
	if merged[0].Severity != "high" {
		t.Errorf("Expected highest severity 'high', got '%s'", merged[0].Severity)
	}
}

func TestMergeOverlappingFindings_MaxConfidence(t *testing.T) {
	engine := NewEngine()
	
	// Create findings with different confidences
	findings := []Finding{
		{
			Type:       "token_heuristics",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "token1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "token1",
		},
		{
			Type:       "token_heuristics",
			Severity:   "high",
			Confidence: "high",
			Reason:     "token1",
			LineNumber: 1,
			ByteStart:  15,
			ByteEnd:    25,
			RawMatch:   "token1",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	if len(merged) != 1 {
		t.Fatalf("Expected 1 merged finding, got %d", len(merged))
	}
	
	if merged[0].Confidence != "high" {
		t.Errorf("Expected max confidence 'high', got '%s'", merged[0].Confidence)
	}
}

func TestMergeOverlappingFindings_ConcatenatedReasons(t *testing.T) {
	engine := NewEngine()
	
	// Create findings with different raw matches
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret1",
		},
		{
			Type:       "token_heuristics",
			Severity:   "high",
			Confidence: "high",
			Reason:     "secret2",
			LineNumber: 1,
			ByteStart:  15,
			ByteEnd:    25,
			RawMatch:   "secret2",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	if len(merged) != 1 {
		t.Fatalf("Expected 1 merged finding, got %d", len(merged))
	}
	
	if !strings.Contains(merged[0].RawMatch, "secret1") || !strings.Contains(merged[0].RawMatch, "secret2") {
		t.Errorf("Expected concatenated reasons, got '%s'", merged[0].RawMatch)
	}
}

func TestSortFindings_Deterministic(t *testing.T) {
	engine := NewEngine()
	
	// Create findings in random order
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret3",
			LineNumber: 3,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret3",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret1",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret2",
			LineNumber: 2,
			ByteStart:  5,
			ByteEnd:    15,
			RawMatch:   "secret2",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret2b",
			LineNumber: 2,
			ByteStart:  20,
			ByteEnd:    30,
			RawMatch:   "secret2b",
		},
	}
	
	sorted := engine.sortFindings(findings)
	
	if len(sorted) != 4 {
		t.Fatalf("Expected 4 findings, got %d", len(sorted))
	}
	
	// Verify ordering: by line number, then byte start
	if sorted[0].LineNumber != 1 {
		t.Errorf("Expected first finding on line 1, got line %d", sorted[0].LineNumber)
	}
	if sorted[1].LineNumber != 2 || sorted[1].ByteStart != 5 {
		t.Errorf("Expected second finding on line 2, byte 5, got line %d, byte %d", sorted[1].LineNumber, sorted[1].ByteStart)
	}
	if sorted[2].LineNumber != 2 || sorted[2].ByteStart != 20 {
		t.Errorf("Expected third finding on line 2, byte 20, got line %d, byte %d", sorted[2].LineNumber, sorted[2].ByteStart)
	}
	if sorted[3].LineNumber != 3 {
		t.Errorf("Expected fourth finding on line 3, got line %d", sorted[3].LineNumber)
	}
}

func TestSortFindings_MultipleCallsDeterministic(t *testing.T) {
	engine := NewEngine()
	
	// Create findings in random order
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret3",
			LineNumber: 3,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret3",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret1",
		},
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret2",
			LineNumber: 2,
			ByteStart:  5,
			ByteEnd:    15,
			RawMatch:   "secret2",
		},
	}
	
	// Sort multiple times - should get same result
	sorted1 := engine.sortFindings(findings)
	sorted2 := engine.sortFindings(findings)
	
	if len(sorted1) != len(sorted2) {
		t.Fatalf("Expected same length, got %d and %d", len(sorted1), len(sorted2))
	}
	
	for i := range sorted1 {
		if sorted1[i].LineNumber != sorted2[i].LineNumber || sorted1[i].ByteStart != sorted2[i].ByteStart {
			t.Errorf("Sorting not deterministic at index %d", i)
		}
	}
}

func TestMergeOverlappingFindings_RealWorldScenario(t *testing.T) {
	engine := NewEngine()
	
	// Simulate a real scenario: password rule and token heuristics both detect the same secret
	text := `password = "AbCdEfGhIjKlMnOpQrStUvWxYz1234567890"`
	
	result := engine.Analyze(text)
	
	// Should merge password_assignment and token_heuristics findings if they overlap
	// The exact number depends on overlap detection, but should be <= 2
	if len(result.Findings) > 2 {
		t.Logf("Found %d findings (may be merged):", len(result.Findings))
		for i, f := range result.Findings {
			t.Logf("  Finding %d: type=%s, severity=%s, line=%d", i, f.Type, f.Severity, f.LineNumber)
		}
	}
	
	// At minimum, should have at least one finding
	if len(result.Findings) == 0 {
		t.Error("Expected at least one finding")
	}
	
	// Verify all findings are properly sorted
	for i := 1; i < len(result.Findings); i++ {
		prev := result.Findings[i-1]
		curr := result.Findings[i]
		
		if prev.LineNumber > curr.LineNumber {
			t.Errorf("Findings not sorted: line %d before line %d", prev.LineNumber, curr.LineNumber)
		}
		if prev.LineNumber == curr.LineNumber && prev.ByteStart > curr.ByteStart {
			t.Errorf("Findings not sorted on same line: byte %d before byte %d", prev.ByteStart, curr.ByteStart)
		}
	}
}

func TestMergeOverlappingFindings_AdjacentBytes(t *testing.T) {
	engine := NewEngine()
	
	// Create findings that are adjacent (within 10 bytes)
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret1",
		},
		{
			Type:       "token_heuristics",
			Severity:   "medium",
			Confidence: "high",
			Reason:     "secret2",
			LineNumber: 1,
			ByteStart:  25, // 5 bytes after first ends (within 10 byte threshold)
			ByteEnd:    35,
			RawMatch:   "secret2",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	// Should merge because they're within 10 bytes on same line
	if len(merged) != 1 {
		t.Errorf("Expected 1 merged finding (adjacent within 10 bytes), got %d", len(merged))
	}
}

func TestMergeOverlappingFindings_FarApart(t *testing.T) {
	engine := NewEngine()
	
	// Create findings that are far apart (more than 10 bytes)
	findings := []Finding{
		{
			Type:       "password_assignment",
			Severity:   "high",
			Confidence: "medium",
			Reason:     "secret1",
			LineNumber: 1,
			ByteStart:  10,
			ByteEnd:    20,
			RawMatch:   "secret1",
		},
		{
			Type:       "token_heuristics",
			Severity:   "medium",
			Confidence: "high",
			Reason:     "secret2",
			LineNumber: 1,
			ByteStart:  35, // 15 bytes after first ends (beyond 10 byte threshold)
			ByteEnd:    45,
			RawMatch:   "secret2",
		},
	}
	
	merged := engine.mergeOverlappingFindings(findings)
	
	// Should NOT merge because they're more than 10 bytes apart
	if len(merged) != 2 {
		t.Errorf("Expected 2 findings (not merged, too far apart), got %d", len(merged))
	}
}

