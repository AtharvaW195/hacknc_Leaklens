package detector

import "strings"

// Finding represents a single detection finding
type Finding struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
	LineNumber int    `json:"line_number"`
	// RawMatch stores the original match for redaction purposes (not in JSON output)
	RawMatch string `json:"-"`
}

// AnalysisResult contains the overall analysis results
type AnalysisResult struct {
	OverallRisk   string
	RiskRationale string
	Findings      []Finding
}

// Engine coordinates rule execution and result aggregation
type Engine struct {
	rules []Rule
}

// NewEngine creates a new detection engine with default rules
func NewEngine() *Engine {
	engine := &Engine{
		rules: []Rule{},
	}
	// Register default rules
	engine.AddRule(NewPEMRule())
	engine.AddRule(NewJWTRule())
	engine.AddRule(NewPasswordRule())
	engine.AddRule(NewTokenHeuristicsRule())
	return engine
}

// AddRule adds a rule to the engine
func (e *Engine) AddRule(rule Rule) {
	e.rules = append(e.rules, rule)
}

// Analyze runs all rules against the input text and returns results
func (e *Engine) Analyze(text string) AnalysisResult {
	allFindings := []Finding{}

	// Run all rules
	for _, rule := range e.rules {
		findings := rule.Analyze(text)
		allFindings = append(allFindings, findings...)
	}

	// Determine overall risk
	overallRisk := "low"
	riskRationale := "No issues detected"

	if len(allFindings) > 0 {
		// Check for high severity findings - overall HIGH if any HIGH finding
		hasHigh := false
		for _, finding := range allFindings {
			if finding.Severity == "high" {
				hasHigh = true
				break
			}
		}

		if hasHigh {
			overallRisk = "high"
			riskRationale = "High severity issues detected"
		} else {
			overallRisk = "medium"
			riskRationale = "Some issues detected"
		}
	}

	// Redact findings to prevent leaking full secrets
	redactedFindings := make([]Finding, len(allFindings))
	for i, finding := range allFindings {
		redactedFindings[i] = finding.Redact()
	}

	return AnalysisResult{
		OverallRisk:   overallRisk,
		RiskRationale: riskRationale,
		Findings:      redactedFindings,
	}
}

// Redact masks sensitive information in the finding
func (f Finding) Redact() Finding {
	redacted := f
	// Redact the reason field if it contains the raw match
	if f.RawMatch != "" {
		var masked string
		// For token_heuristics, mask the majority of the token (more aggressive)
		if f.Type == "token_heuristics" {
			if len(f.RawMatch) > 16 {
				// Show first 4 and last 4, mask the rest (majority masked)
				masked = f.RawMatch[:4] + "..." + f.RawMatch[len(f.RawMatch)-4:]
			} else if len(f.RawMatch) > 8 {
				// For medium tokens, show first 3 and last 3
				masked = f.RawMatch[:3] + "..." + f.RawMatch[len(f.RawMatch)-3:]
			} else {
				// For short tokens, mask most of it
				if len(f.RawMatch) > 4 {
					masked = f.RawMatch[:2] + "..." + f.RawMatch[len(f.RawMatch)-2:]
				} else {
					masked = "****"
				}
			}
		} else {
			// For other types, use standard redaction
			if len(f.RawMatch) > 8 {
				// Show first 4 and last 4 characters, mask the middle
				masked = f.RawMatch[:4] + "..." + f.RawMatch[len(f.RawMatch)-4:]
			} else {
				// For short matches, just show first 2 and mask the rest
				if len(f.RawMatch) > 4 {
					masked = f.RawMatch[:2] + "..." + f.RawMatch[len(f.RawMatch)-2:]
				} else {
					masked = "****"
				}
			}
		}
		
		// Replace raw match in reason if it contains the raw match
		if f.Reason == f.RawMatch {
			redacted.Reason = masked
		} else if f.Reason != "" && strings.Contains(f.Reason, f.RawMatch) {
			// Replace the raw match substring in reason
			redacted.Reason = strings.ReplaceAll(f.Reason, f.RawMatch, masked)
		} else if f.Reason == "" {
			redacted.Reason = masked
		}
	}
	// Clear RawMatch from output
	redacted.RawMatch = ""
	return redacted
}

