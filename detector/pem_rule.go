package detector

import (
	"regexp"
	"strings"
)

// PEMRule detects PEM-encoded private keys
type PEMRule struct{}

// NewPEMRule creates a new PEM detection rule
func NewPEMRule() *PEMRule {
	return &PEMRule{}
}

// Name returns the name of the rule
func (r *PEMRule) Name() string {
	return "pem_private_key"
}

// Analyze checks for PEM-encoded private keys
func (r *PEMRule) Analyze(text string) []Finding {
	var findings []Finding
	lines := strings.Split(text, "\n")
	
	// Pattern to match PEM private keys (RSA, EC, DSA, etc.)
	// Matches: -----BEGIN ... PRIVATE KEY-----
	beginPattern := regexp.MustCompile(`-----BEGIN\s+(?:RSA\s+)?(?:EC\s+)?(?:DSA\s+)?PRIVATE\s+KEY-----`)
	endPattern := regexp.MustCompile(`-----END\s+(?:RSA\s+)?(?:EC\s+)?(?:DSA\s+)?PRIVATE\s+KEY-----`)
	
	for lineNum, line := range lines {
		if beginPattern.MatchString(line) {
			// Find the corresponding END marker
			remainingLines := lines[lineNum:]
			remainingText := strings.Join(remainingLines, "\n")
			
			endMatch := endPattern.FindStringIndex(remainingText)
			if endMatch != nil {
				// Extract the full PEM block including the END marker
				endIdx := endMatch[1]
				pemBlock := remainingText[:endIdx]
				
				findings = append(findings, Finding{
					Type:       "pem_private_key",
					Severity:   "high",
					Confidence: "high",
					Reason:     pemBlock,
					LineNumber: lineNum + 1,
					RawMatch:   pemBlock,
				})
			}
		}
	}
	
	return findings
}

