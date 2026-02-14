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
	
	// Calculate byte offsets for each line
	lineOffsets := make([]int, len(lines)+1)
	offset := 0
	for i, line := range lines {
		lineOffsets[i] = offset
		offset += len(line) + 1 // +1 for newline
	}
	lineOffsets[len(lines)] = offset
	
	// Pattern to match PEM private keys (RSA, EC, DSA, etc.)
	// Matches: -----BEGIN ... PRIVATE KEY-----
	beginPattern := regexp.MustCompile(`-----BEGIN\s+(?:RSA\s+)?(?:EC\s+)?(?:DSA\s+)?PRIVATE\s+KEY-----`)
	endPattern := regexp.MustCompile(`-----END\s+(?:RSA\s+)?(?:EC\s+)?(?:DSA\s+)?PRIVATE\s+KEY-----`)
	
	for lineNum, line := range lines {
		beginMatch := beginPattern.FindStringIndex(line)
		if beginMatch != nil {
			// Find the corresponding END marker
			remainingLines := lines[lineNum:]
			remainingText := strings.Join(remainingLines, "\n")
			
			endMatch := endPattern.FindStringIndex(remainingText)
			if endMatch != nil {
				// Extract the full PEM block including the END marker
				endIdx := endMatch[1]
				pemBlock := remainingText[:endIdx]
				
				// Calculate byte positions
				byteStart := lineOffsets[lineNum] + beginMatch[0]
				// Calculate byte end: find which line contains the end position
				byteEnd := byteStart
				currentOffset := 0
				for i := lineNum; i < len(lines); i++ {
					lineLen := len(lines[i])
					if currentOffset+lineLen >= endMatch[1] {
						// End marker is on this line
						byteEnd = lineOffsets[i] + (endMatch[1] - currentOffset)
						break
					}
					currentOffset += lineLen + 1 // +1 for newline
				}
				
				findings = append(findings, Finding{
					Type:       "pem_private_key",
					Severity:   "high",
					Confidence: "high",
					Reason:     pemBlock,
					LineNumber: lineNum + 1,
					ByteStart:  byteStart,
					ByteEnd:    byteEnd,
					RawMatch:   pemBlock,
				})
			}
		}
	}
	
	return findings
}

