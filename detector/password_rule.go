package detector

import (
	"regexp"
	"strings"
)

// PasswordRule detects password assignments in code
type PasswordRule struct{}

// NewPasswordRule creates a new password assignment detection rule
func NewPasswordRule() *PasswordRule {
	return &PasswordRule{}
}

// Name returns the name of the rule
func (r *PasswordRule) Name() string {
	return "password_assignment"
}

// Analyze checks for password assignments
func (r *PasswordRule) Analyze(text string) []Finding {
	var findings []Finding
	lines := strings.Split(text, "\n")
	
	// Pattern to match common password assignment patterns
	// Matches: password = "...", password="...", password: "...", etc.
	// Also matches: passwd, pwd, pass, secret, api_key, apiKey, etc.
	passwordPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:password|passwd|pwd|pass|secret|api[_-]?key|apikey)\s*[=:]\s*["']([^"']{8,})["']`),
		regexp.MustCompile(`(?i)(?:password|passwd|pwd|pass|secret|api[_-]?key|apikey)\s*[=:]\s*([a-zA-Z0-9+/=]{12,})`), // base64-like
	}
	
	for lineNum, line := range lines {
		for _, pattern := range passwordPatterns {
			matches := pattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 1 && match[1] != "" {
					// Check if it's not a comment or already redacted
					trimmedLine := strings.TrimSpace(line)
					if !strings.HasPrefix(trimmedLine, "//") && 
					   !strings.HasPrefix(trimmedLine, "#") && 
					   !strings.HasPrefix(trimmedLine, "*") &&
					   !strings.Contains(match[1], "...") {
						findings = append(findings, Finding{
							Type:       "password_assignment",
							Severity:   "high",
							Confidence: "medium",
							Reason:     match[1],
							LineNumber: lineNum + 1,
							RawMatch:   match[1],
						})
					}
				}
			}
		}
	}
	
	return findings
}

