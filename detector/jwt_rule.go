package detector

import (
	"regexp"
	"strings"
)

// JWTRule detects JSON Web Tokens
type JWTRule struct{}

// NewJWTRule creates a new JWT detection rule
func NewJWTRule() *JWTRule {
	return &JWTRule{}
}

// Name returns the name of the rule
func (r *JWTRule) Name() string {
	return "jwt_token"
}

// Analyze checks for JWT tokens
func (r *JWTRule) Analyze(text string) []Finding {
	var findings []Finding
	lines := strings.Split(text, "\n")
	
	// JWT pattern: three base64url-encoded segments separated by dots
	// Format: header.payload.signature
	// Each segment is base64url encoded (alphanumeric, -, _, =)
	jwtPattern := regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	
	for lineNum, line := range lines {
		matches := jwtPattern.FindAllString(line, -1)
		for _, match := range matches {
			// Validate it looks like a JWT (has 3 parts separated by dots)
			parts := strings.Split(match, ".")
			if len(parts) == 3 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(parts[2]) > 0 {
				findings = append(findings, Finding{
					Type:       "jwt_token",
					Severity:   "high",
					Confidence: "high",
					Reason:     match,
					LineNumber: lineNum + 1,
					RawMatch:   match,
				})
			}
		}
	}
	
	return findings
}

