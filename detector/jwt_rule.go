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
	
	// Calculate byte offsets for each line
	lineOffsets := make([]int, len(lines)+1)
	offset := 0
	for i, line := range lines {
		lineOffsets[i] = offset
		offset += len(line) + 1 // +1 for newline
	}
	lineOffsets[len(lines)] = offset
	
	// JWT pattern: three base64url-encoded segments separated by dots
	// Format: header.payload.signature
	// Each segment is base64url encoded (alphanumeric, -, _, =)
	jwtPattern := regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	
	for lineNum, line := range lines {
		matches := jwtPattern.FindAllStringIndex(line, -1)
		for _, match := range matches {
			token := line[match[0]:match[1]]
			// Validate it looks like a JWT (has 3 parts separated by dots)
			parts := strings.Split(token, ".")
			if len(parts) == 3 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(parts[2]) > 0 {
				byteStart := lineOffsets[lineNum] + match[0]
				byteEnd := lineOffsets[lineNum] + match[1]
				
				findings = append(findings, Finding{
					Type:       "jwt_token",
					Severity:   "high",
					Confidence: "high",
					Reason:     token,
					LineNumber: lineNum + 1,
					ByteStart:  byteStart,
					ByteEnd:    byteEnd,
					RawMatch:   token,
				})
			}
		}
	}
	
	return findings
}

