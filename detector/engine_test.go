package detector

import "testing"

func TestNewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if engine.rules == nil {
		t.Error("Engine rules slice is nil")
	}
	// NewEngine now includes 4 default rules (PEM, JWT, Password, TokenHeuristics)
	if len(engine.rules) != 4 {
		t.Errorf("Expected 4 default rules, got %d rules", len(engine.rules))
	}
}

func TestEngineAnalyzeEmptyText(t *testing.T) {
	engine := NewEngine()
	result := engine.Analyze("")
	
	if result.OverallRisk != "low" {
		t.Errorf("Expected overall_risk 'low', got '%s'", result.OverallRisk)
	}
	
	if result.RiskRationale == "" {
		t.Error("risk_rationale should not be empty")
	}
	
	if result.Findings == nil {
		t.Error("findings should not be nil")
	}
	
	if len(result.Findings) != 0 {
		t.Errorf("Expected no findings, got %d", len(result.Findings))
	}
}

func TestEngineAnalyzeWithText(t *testing.T) {
	engine := NewEngine()
	result := engine.Analyze("some test text")
	
	if result.OverallRisk == "" {
		t.Error("overall_risk should not be empty")
	}
	
	if result.RiskRationale == "" {
		t.Error("risk_rationale should not be empty")
	}
	
	if result.Findings == nil {
		t.Error("findings should not be nil")
	}
}

func TestEngineAddRule(t *testing.T) {
	engine := &Engine{
		rules: []Rule{},
	}
	
	// Create a mock rule
	mockRule := &mockRule{name: "test-rule"}
	
	engine.AddRule(mockRule)
	
	if len(engine.rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(engine.rules))
	}
	
	if engine.rules[0].Name() != "test-rule" {
		t.Errorf("Expected rule name 'test-rule', got '%s'", engine.rules[0].Name())
	}
}

func TestEngineAddRuleToDefaultEngine(t *testing.T) {
	engine := NewEngine()
	
	// NewEngine includes 4 default rules
	if len(engine.rules) != 4 {
		t.Errorf("Expected 4 default rules, got %d", len(engine.rules))
	}
	
	// Create a mock rule
	mockRule := &mockRule{name: "test-rule"}
	
	engine.AddRule(mockRule)
	
	if len(engine.rules) != 5 {
		t.Errorf("Expected 5 rules after adding one, got %d", len(engine.rules))
	}
	
	if engine.rules[4].Name() != "test-rule" {
		t.Errorf("Expected last rule name 'test-rule', got '%s'", engine.rules[4].Name())
	}
}

// mockRule is a test implementation of the Rule interface
type mockRule struct {
	name string
}

func (m *mockRule) Name() string {
	return m.name
}

func (m *mockRule) Analyze(text string) []Finding {
	return []Finding{}
}

