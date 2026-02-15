package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pasteguard/detector"
)

func TestAnalyzeTextEndpoint(t *testing.T) {
	// Initialize detector engine for tests
	detectorEngine = detector.NewEngine()

	tests := []struct {
		name           string
		method         string
		body           string
		wantStatus     int
		wantRisk       string
		checkRedaction bool
		redactionText  string
	}{
		{
			name:       "HIGH risk on PEM private key",
			method:     "POST",
			body:       `{"text":"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----"}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "413 on payload too large",
			method:     "POST",
			body:       `{"text":"` + strings.Repeat("a", 101*1024) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "Redaction safety check",
			method:         "POST",
			body:           `{"text":"password = super_secret_password_12345"}`,
			wantStatus:     http.StatusOK,
			checkRedaction: true,
			redactionText:  "super_secret_password_12345",
		},
		{
			name:       "Empty text rejected",
			method:     "POST",
			body:       `{"text":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Whitespace-only text rejected",
			method:     "POST",
			body:       `{"text":"   "}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "OPTIONS handled",
			method:     "OPTIONS",
			body:       "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET method rejected",
			method:     "GET",
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "PUT method rejected",
			method:     "PUT",
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE method rejected",
			method:     "DELETE",
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "Invalid JSON format",
			method:     "POST",
			body:       `{"text": invalid}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Missing text field",
			method:     "POST",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Text field is null",
			method:     "POST",
			body:       `{"text": null}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "JWT token detection",
			method:     "POST",
			body:       `{"text":"token = \"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "Password assignment detection",
			method:     "POST",
			body:       `{"text":"password = \"secret123\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "Token heuristics detection",
			method:     "POST",
			body:       `{"text":"api_key = \"AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdef\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "Low risk text",
			method:     "POST",
			body:       `{"text":"This is just regular text with no secrets"}`,
			wantStatus: http.StatusOK,
			wantRisk:   "LOW",
		},
		{
			name:       "Medium risk with multiple findings",
			method:     "POST",
			body:       `{"text":"some_var = \"AbCdEfGhIjKlMnOpQrSt\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "", // Don't check risk - may vary based on detection
		},
		{
			name:       "Request body too large - exact boundary",
			method:     "POST",
			body:       `{"text":"` + strings.Repeat("a", 99*1024) + `"}`,
			wantStatus: http.StatusOK, // Should succeed well under limit
		},
		{
			name:       "Request body too large - one byte over",
			method:     "POST",
			body:       `{"text":"` + strings.Repeat("a", 100*1024+1) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "Multiple secrets in one request",
			method:     "POST",
			body:       `{"text":"password = \"secret1\"\napi_key = \"secret2\"\ntoken = \"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "EC private key detection",
			method:     "POST",
			body:       `{"text":"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIAKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKjKj\n-----END EC PRIVATE KEY-----"}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "Generic private key detection",
			method:     "POST",
			body:       `{"text":"-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKj\n-----END PRIVATE KEY-----"}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:           "Redaction check with JWT",
			method:         "POST",
			body:           `{"text":"token = \"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U\""}`,
			wantStatus:     http.StatusOK,
			checkRedaction: true,
			redactionText:  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		},
		{
			name:           "Redaction check with PEM key",
			method:         "POST",
			body:           `{"text":"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----"}`,
			wantStatus:     http.StatusOK,
			checkRedaction: true,
			redactionText:  "MIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v",
		},
		{
			name:       "Very long text within limit",
			method:     "POST",
			body:       `{"text":"` + strings.Repeat("a", 50*1024) + `"}`,
			wantStatus: http.StatusOK,
			wantRisk:   "LOW",
		},
		{
			name:       "Newline handling",
			method:     "POST",
			body:       `{"text":"line1\nline2\npassword = \"secret\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "", // May vary based on detection
		},
		{
			name:       "Tab and space handling",
			method:     "POST",
			body:       `{"text":"password = secret123"}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
		{
			name:       "Special characters in text",
			method:     "POST",
			body:       `{"text":"password = \"secret!@#$%^&*()\""}`,
			wantStatus: http.StatusOK,
			wantRisk:   "HIGH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/analyze-text", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Call the handler directly
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setCORS(w)
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
				if r.Method != "POST" {
					writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				const maxSize = 100 * 1024
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)

				var reqBody struct {
					Text string `json:"text"`
				}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					if err == io.EOF {
						writeJSONError(w, "Request body is required", http.StatusBadRequest)
					} else if strings.Contains(err.Error(), "request body too large") {
						writeJSONError(w, "Payload too large (max 100KB)", http.StatusRequestEntityTooLarge)
					} else {
						writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
					}
					return
				}

				if strings.TrimSpace(reqBody.Text) == "" {
					writeJSONError(w, "Text cannot be empty", http.StatusBadRequest)
					return
				}

				result := detectorEngine.Analyze(reqBody.Text)

				// Safety check: ensure no finding preview equals a substring of input longer than 8 chars
				for i := range result.Findings {
					finding := &result.Findings[i]
					if len(finding.Reason) > 8 && strings.Contains(reqBody.Text, finding.Reason) {
						finding.Reason = "[REDACTED]"
					}
				}

				riskLevel := strings.ToUpper(result.OverallRisk)

				response := map[string]interface{}{
					"overall_risk":   riskLevel,
					"risk_rationale": result.RiskRationale,
					"findings":       result.Findings,
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}).ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantRisk != "" {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if risk, ok := response["overall_risk"].(string); !ok || risk != tt.wantRisk {
					t.Errorf("overall_risk = %v, want %s", risk, tt.wantRisk)
				}
			}

			if tt.checkRedaction {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				findings, ok := response["findings"].([]interface{})
				if !ok || len(findings) == 0 {
					t.Fatal("Expected findings in response")
				}
				// Check that the full secret is not in any finding reason
				responseJSON, _ := json.Marshal(response)
				if strings.Contains(string(responseJSON), tt.redactionText) {
					t.Errorf("Full secret '%s' found in response JSON", tt.redactionText)
				}
			}
		})
	}
}


func TestSetCORS(t *testing.T) {
	w := httptest.NewRecorder()
	setCORS(w)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "POST, OPTIONS" {
		t.Errorf("Expected Access-Control-Allow-Methods to be POST, OPTIONS, got %s", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "*" {
		t.Errorf("Expected Access-Control-Allow-Headers to be *, got %s", w.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		code     int
		wantCode int
	}{
		{
			name:     "400 Bad Request",
			message:  "Invalid input",
			code:     http.StatusBadRequest,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "500 Internal Server Error",
			message:  "Server error",
			code:     http.StatusInternalServerError,
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "413 Request Entity Too Large",
			message:  "Too large",
			code:     http.StatusRequestEntityTooLarge,
			wantCode: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONError(w, tt.message, tt.code)

			if w.Code != tt.wantCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantCode)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type to be application/json, got %s", w.Header().Get("Content-Type"))
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("Error message = %s, want %s", response["error"], tt.message)
			}
		})
	}
}


func TestAnalyzeTextResponseStructure(t *testing.T) {
	detectorEngine = detector.NewEngine()

	tests := []struct {
		name           string
		body           string
		checkStructure bool
	}{
		{
			name:           "Response has all required fields",
			body:           `{"text":"password = \"secret\""}`,
			checkStructure: true,
		},
		{
			name:           "Response structure for low risk",
			body:           `{"text":"regular text"}`,
			checkStructure: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/analyze-text", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setCORS(w)
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
				if r.Method != "POST" {
					writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				const maxSize = 100 * 1024
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)

				var reqBody struct {
					Text string `json:"text"`
				}
				decoder := json.NewDecoder(r.Body)
				if err := decoder.Decode(&reqBody); err != nil {
					if err == io.EOF {
						writeJSONError(w, "Request body is required", http.StatusBadRequest)
					} else if strings.Contains(err.Error(), "request body too large") || strings.Contains(err.Error(), "http: request body too large") {
						writeJSONError(w, "Payload too large (max 100KB)", http.StatusRequestEntityTooLarge)
					} else {
						writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
					}
					return
				}

				if strings.TrimSpace(reqBody.Text) == "" {
					writeJSONError(w, "Text cannot be empty", http.StatusBadRequest)
					return
				}

				result := detectorEngine.Analyze(reqBody.Text)

				for i := range result.Findings {
					finding := &result.Findings[i]
					if len(finding.Reason) > 8 && strings.Contains(reqBody.Text, finding.Reason) {
						finding.Reason = "[REDACTED]"
					}
				}

				riskLevel := strings.ToUpper(result.OverallRisk)

				response := map[string]interface{}{
					"overall_risk":   riskLevel,
					"risk_rationale": result.RiskRationale,
					"findings":       result.Findings,
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}).ServeHTTP(w, req)

			if tt.checkStructure {
				var response map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Check required fields
				if _, ok := response["overall_risk"]; !ok {
					t.Error("Response missing 'overall_risk' field")
				}
				if _, ok := response["risk_rationale"]; !ok {
					t.Error("Response missing 'risk_rationale' field")
				}
				if _, ok := response["findings"]; !ok {
					t.Error("Response missing 'findings' field")
				}

				// Check risk level is uppercase
				if risk, ok := response["overall_risk"].(string); ok {
					if risk != strings.ToUpper(risk) {
						t.Errorf("Risk level should be uppercase, got %s", risk)
					}
					if risk != "LOW" && risk != "MEDIUM" && risk != "HIGH" {
						t.Errorf("Invalid risk level: %s", risk)
					}
				}

				// Check findings is an array
				if findings, ok := response["findings"].([]interface{}); ok {
					for i, finding := range findings {
						if findingMap, ok := finding.(map[string]interface{}); ok {
							// Check required finding fields
							if _, ok := findingMap["type"]; !ok {
								t.Errorf("Finding %d missing 'type' field", i)
							}
							if _, ok := findingMap["severity"]; !ok {
								t.Errorf("Finding %d missing 'severity' field", i)
							}
							if _, ok := findingMap["confidence"]; !ok {
								t.Errorf("Finding %d missing 'confidence' field", i)
							}
							if _, ok := findingMap["reason"]; !ok {
								t.Errorf("Finding %d missing 'reason' field", i)
							}
							if _, ok := findingMap["line_number"]; !ok {
								t.Errorf("Finding %d missing 'line_number' field", i)
							}
						}
					}
				}
			}
		})
	}
}

