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
	
	// Calculate byte offsets for each line
	lineOffsets := make([]int, len(lines)+1)
	offset := 0
	for i, line := range lines {
		lineOffsets[i] = offset
		offset += len(line) + 1 // +1 for newline
	}
	lineOffsets[len(lines)] = offset
	
	// Pattern to match common password assignment patterns
	// Matches: password = "...", password="...", password: "...", etc.
	// Also matches: passwd, pwd, pass, secret, api_key, apiKey, etc.
	passwordPatterns := []*regexp.Regexp{
		// Pattern 1: With quotes (handles both single and double quotes)
		regexp.MustCompile(`(?i)(?:password|passwd|pwd|pass|secret|api[_-]?key|apikey)\s*[=:]\s*["']?([^"'\s]{8,})["']?`),
		// Pattern 2: Without quotes but with spaces (handles: password = secret123)
		regexp.MustCompile(`(?i)(?:password|passwd|pwd|pass|secret|api[_-]?key|apikey)\s*[=:]\s+([a-zA-Z0-9_\-+/=]{8,})(?:\s|$|[;"',)])`),
		// Pattern 3: Base64-like without quotes
		regexp.MustCompile(`(?i)(?:password|passwd|pwd|pass|secret|api[_-]?key|apikey)\s*[=:]\s*([a-zA-Z0-9+/=]{12,})`),
	}
	
	for lineNum, line := range lines {
		for _, pattern := range passwordPatterns {
			matches := pattern.FindAllStringSubmatchIndex(line, -1)
			for _, match := range matches {
				// match[0] and match[1] are the full match, match[2] and match[3] are the first capture group
				if len(match) >= 4 && match[2] != -1 && match[3] != -1 {
					value := line[match[2]:match[3]]
					if value != "" {
						// Check if it's not a comment or already redacted
						trimmedLine := strings.TrimSpace(line)
						if !strings.HasPrefix(trimmedLine, "//") && 
						   !strings.HasPrefix(trimmedLine, "#") && 
						   !strings.HasPrefix(trimmedLine, "*") &&
						   !strings.Contains(value, "...") {
							byteStart := lineOffsets[lineNum] + match[2]
							byteEnd := lineOffsets[lineNum] + match[3]
							
							findings = append(findings, Finding{
								Type:       "password_assignment",
								Severity:   "high",
								Confidence: "medium",
								Reason:     value,
								LineNumber: lineNum + 1,
								ByteStart:  byteStart,
								ByteEnd:    byteEnd,
								RawMatch:   value,
							})
						}
					}
				}
			}
		}
	}
	
	return findings
}

