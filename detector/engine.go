package detector

import "strings"

// Finding represents a single detection finding
type Finding struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	Reason     string `json:"reason"`
	LineNumber int    `json:"line_number"`
	// ByteStart and ByteEnd track the byte position in the original text (for overlap detection)
	ByteStart int `json:"-"`
	ByteEnd   int `json:"-"`
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

	// Merge overlapping findings
	mergedFindings := e.mergeOverlappingFindings(allFindings)

	// Sort findings deterministically (by line number, then byte start)
	mergedFindings = e.sortFindings(mergedFindings)

	// Determine overall risk
	overallRisk := "low"
	riskRationale := "No issues detected"

	if len(mergedFindings) > 0 {
		// Check for high severity findings - overall HIGH if any HIGH finding
		hasHigh := false
		for _, finding := range mergedFindings {
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
	redactedFindings := make([]Finding, len(mergedFindings))
	for i, finding := range mergedFindings {
		redactedFindings[i] = finding.Redact()
	}

	return AnalysisResult{
		OverallRisk:   overallRisk,
		RiskRationale: riskRationale,
		Findings:      redactedFindings,
	}
}

// mergeOverlappingFindings merges findings that overlap in byte range or are on the same line
func (e *Engine) mergeOverlappingFindings(findings []Finding) []Finding {
	if len(findings) == 0 {
		return findings
	}

	// Sort by line number first, then byte start for deterministic processing
	sorted := make([]Finding, len(findings))
	copy(sorted, findings)
	sortFindingsByPosition(sorted)

	merged := []Finding{}
	used := make([]bool, len(sorted))

	for i := 0; i < len(sorted); i++ {
		if used[i] {
			continue
		}

		current := sorted[i]
		toMerge := []int{i}

		// Find all overlapping findings
		for j := i + 1; j < len(sorted); j++ {
			if used[j] {
				continue
			}

			other := sorted[j]
			if e.findingsOverlap(current, other) {
				toMerge = append(toMerge, j)
			}
		}

		// Merge all overlapping findings
		if len(toMerge) > 1 {
			mergedFinding := e.mergeFindings(sorted, toMerge)
			merged = append(merged, mergedFinding)
			for _, idx := range toMerge {
				used[idx] = true
			}
		} else {
			merged = append(merged, current)
			used[i] = true
		}
	}

	return merged
}

// findingsOverlap checks if two findings overlap
func (e *Engine) findingsOverlap(f1, f2 Finding) bool {
	// Same line and byte ranges overlap or are adjacent
	if f1.LineNumber == f2.LineNumber {
		// Check if byte ranges overlap or are adjacent (within 10 bytes)
		return (f1.ByteStart <= f2.ByteEnd+10 && f2.ByteStart <= f1.ByteEnd+10) ||
			(f2.ByteStart <= f1.ByteEnd+10 && f1.ByteStart <= f2.ByteEnd+10)
	}
	// Adjacent lines (within 1 line) and byte ranges overlap
	if abs(f1.LineNumber-f2.LineNumber) <= 1 {
		return f1.ByteStart <= f2.ByteEnd && f2.ByteStart <= f1.ByteEnd
	}
	return false
}

// mergeFindings merges multiple findings into one
func (e *Engine) mergeFindings(findings []Finding, indices []int) Finding {
	if len(indices) == 0 {
		return Finding{}
	}
	if len(indices) == 1 {
		return findings[indices[0]]
	}

	merged := findings[indices[0]]

	// Find highest severity
	severityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
	for _, idx := range indices[1:] {
		f := findings[idx]
		if severityOrder[f.Severity] > severityOrder[merged.Severity] {
			merged.Severity = f.Severity
		}
	}

	// Find max confidence
	confidenceOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
	for _, idx := range indices[1:] {
		f := findings[idx]
		if confidenceOrder[f.Confidence] > confidenceOrder[merged.Confidence] {
			merged.Confidence = f.Confidence
		}
	}

	// Concatenate reasons (before redaction)
	reasons := []string{merged.RawMatch}
	for _, idx := range indices[1:] {
		f := findings[idx]
		if f.RawMatch != "" && f.RawMatch != merged.RawMatch {
			reasons = append(reasons, f.RawMatch)
		}
	}
	merged.RawMatch = strings.Join(reasons, ", ")
	merged.Reason = merged.RawMatch // Will be redacted later

	// Use minimum line number and byte start, maximum byte end
	for _, idx := range indices[1:] {
		f := findings[idx]
		if f.LineNumber < merged.LineNumber {
			merged.LineNumber = f.LineNumber
		}
		if f.ByteStart < merged.ByteStart {
			merged.ByteStart = f.ByteStart
		}
		if f.ByteEnd > merged.ByteEnd {
			merged.ByteEnd = f.ByteEnd
		}
	}

	// Use the first type (or combine if different)
	typeSet := make(map[string]bool)
	for _, idx := range indices {
		typeSet[findings[idx].Type] = true
	}
	if len(typeSet) > 1 {
		// Multiple types - use "multiple" or first type
		merged.Type = findings[indices[0]].Type
	}

	return merged
}

// sortFindings sorts findings deterministically
func (e *Engine) sortFindings(findings []Finding) []Finding {
	sorted := make([]Finding, len(findings))
	copy(sorted, findings)
	sortFindingsByPosition(sorted)
	return sorted
}

// sortFindingsByPosition sorts findings by line number, then byte start
func sortFindingsByPosition(findings []Finding) {
	for i := 0; i < len(findings)-1; i++ {
		for j := i + 1; j < len(findings); j++ {
			if findings[i].LineNumber > findings[j].LineNumber ||
				(findings[i].LineNumber == findings[j].LineNumber && findings[i].ByteStart > findings[j].ByteStart) {
				findings[i], findings[j] = findings[j], findings[i]
			}
		}
	}
}

// abs returns absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
