package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// TokenHeuristicsRule detects high-entropy token-like substrings
type TokenHeuristicsRule struct{}

// NewTokenHeuristicsRule creates a new token heuristics detection rule
func NewTokenHeuristicsRule() *TokenHeuristicsRule {
	return &TokenHeuristicsRule{}
}

// Name returns the name of the rule
func (r *TokenHeuristicsRule) Name() string {
	return "token_heuristics"
}

// Analyze checks for high-entropy token-like substrings
func (r *TokenHeuristicsRule) Analyze(text string) []Finding {
	var findings []Finding
	lines := strings.Split(text, "\n")

	// Auth-related keywords that increase suspicion
	authKeywords := []string{
		"token", "auth", "api", "key", "secret", "password", "passwd",
		"credential", "bearer", "jwt", "session", "cookie", "access",
		"refresh", "oauth", "apikey", "api_key", "client_id", "client_secret",
	}

	// Patterns for different token types
	base64Pattern := regexp.MustCompile(`[A-Za-z0-9+/=]{20,}`)
	hexPattern := regexp.MustCompile(`[0-9a-fA-F]{24,}`)
	urlSafePattern := regexp.MustCompile(`[A-Za-z0-9_-]{20,}`)

	// Known safe contexts (conservative approach)
	safeContexts := []string{
		"uuid", "guid", "hash", "sha", "md5", "checksum", "version",
		"commit", "branch", "ref", "id", "identifier", "url", "uri",
	}

	for lineNum, line := range lines {
		lineLower := strings.ToLower(line)

		// Check for known safe contexts - skip entire line if found (conservative)
		isSafeContext := false
		for _, safe := range safeContexts {
			// Check if it's actually a safe context (not just part of another word)
			// Use word boundary or underscore/hyphen boundaries for compound words
			pattern := regexp.MustCompile(`(?:\b|_|-)` + regexp.QuoteMeta(safe) + `(?:\b|_|-)`)
			if pattern.MatchString(lineLower) {
				isSafeContext = true
				break
			}
		}
		if isSafeContext {
			continue
		}

		// Check for tokens near auth keywords
		hasAuthKeyword := false
		keywordPositions := []int{}
		for _, keyword := range authKeywords {
			pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
			matches := pattern.FindAllStringIndex(lineLower, -1)
			for _, match := range matches {
				keywordPositions = append(keywordPositions, match[0])
				hasAuthKeyword = true
			}
		}

		// Find potential tokens
		allPatterns := []struct {
			pattern *regexp.Regexp
			name    string
		}{
			{base64Pattern, "base64"},
			{hexPattern, "hex"},
			{urlSafePattern, "urlsafe"},
		}

		for _, p := range allPatterns {
			matches := p.pattern.FindAllStringIndex(line, -1)
			for _, match := range matches {
				token := line[match[0]:match[1]]

				// Skip if it's clearly not a token (e.g., too many repeated chars, too uniform)
				if !r.isHighEntropy(token) {
					continue
				}

				// Calculate score
				score := r.calculateScore(token, hasAuthKeyword, match[0], keywordPositions)

				// Only report if score is significant
				if score >= 3 {
					severity := "medium"
					// High severity if: score >= 6, OR (score >= 5 AND near keyword), OR (long token >= 32 AND near keyword)
					if score >= 6 {
						severity = "high"
					} else if score >= 5 && hasAuthKeyword {
						severity = "high"
					} else if len(token) >= 32 && hasAuthKeyword && score >= 3 {
						severity = "high"
					}

					confidence := "medium"
					if hasAuthKeyword && score >= 5 {
						confidence = "high"
					}

					findings = append(findings, Finding{
						Type:       "token_heuristics",
						Severity:   severity,
						Confidence: confidence,
						Reason:     token,
						LineNumber: lineNum + 1,
						RawMatch:    token,
					})
				}
			}
		}
	}

	return findings
}

// isHighEntropy checks if a string has sufficient entropy characteristics
func (r *TokenHeuristicsRule) isHighEntropy(s string) bool {
	if len(s) < 20 {
		return false
	}

	// Check for too many repeated characters (low entropy)
	charCounts := make(map[rune]int)
	for _, c := range s {
		charCounts[c]++
	}
	maxRepeat := 0
	for _, count := range charCounts {
		if count > maxRepeat {
			maxRepeat = count
		}
	}
	// If any character appears more than 30% of the time, likely low entropy
	if maxRepeat > len(s)*3/10 {
		return false
	}

	// Check charset variety
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, c := range s {
		if unicode.IsUpper(c) {
			hasUpper = true
		} else if unicode.IsLower(c) {
			hasLower = true
		} else if unicode.IsDigit(c) {
			hasDigit = true
		} else {
			hasSpecial = true
		}
	}

	// Need at least 2 character classes for reasonable entropy
	classes := 0
	if hasUpper {
		classes++
	}
	if hasLower {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSpecial {
		classes++
	}

	return classes >= 2
}

// calculateScore calculates a heuristic score for a potential token
func (r *TokenHeuristicsRule) calculateScore(token string, hasAuthKeyword bool, tokenPos int, keywordPositions []int) int {
	score := 0

	// Length scoring (longer = more suspicious)
	if len(token) >= 32 {
		score += 2
	} else if len(token) >= 24 {
		score += 1
	}

	// Charset variety scoring
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, c := range token {
		if unicode.IsUpper(c) {
			hasUpper = true
		} else if unicode.IsLower(c) {
			hasLower = true
		} else if unicode.IsDigit(c) {
			hasDigit = true
		} else {
			hasSpecial = true
		}
	}

	classes := 0
	if hasUpper {
		classes++
	}
	if hasLower {
		classes++
	}
	if hasDigit {
		classes++
	}
	if hasSpecial {
		classes++
	}

	// More character classes = higher score
	if classes >= 3 {
		score += 2
	} else if classes == 2 {
		score += 1
	}

	// Proximity to auth keywords (very important)
	if hasAuthKeyword {
		// Check if token is near any keyword (within 50 chars)
		for _, kwPos := range keywordPositions {
			distance := abs(tokenPos - kwPos)
			if distance <= 50 {
				score += 2
				// Very close proximity (within 10 chars) gets extra point
				if distance <= 10 {
					score += 1
				}
				break // Only count once
			}
		}
	}

	return score
}

// abs returns absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

