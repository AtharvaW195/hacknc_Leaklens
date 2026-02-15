package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]bool
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["ok"] {
		t.Errorf("Expected ok to be true, got %v", response["ok"])
	}
}

func TestAnalyzeHandler_POST(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	requestBody := AnalyzeRequest{
		Text: "password = \"secret123\"",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response AnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high', got '%s'", response.OverallRisk)
	}

	if len(response.Findings) == 0 {
		t.Error("Expected at least one finding")
	}
}

func TestAnalyzeHandler_GET(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/analyze", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestAnalyzeHandler_InvalidJSON(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["error"] == "" {
		t.Error("Expected error message in response")
	}
}

func TestAnalyzeHandler_EmptyText(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	requestBody := AnalyzeRequest{
		Text: "",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response AnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OverallRisk != "low" {
		t.Errorf("Expected overall_risk 'low', got '%s'", response.OverallRisk)
	}

	if len(response.Findings) != 0 {
		t.Errorf("Expected no findings, got %d", len(response.Findings))
	}
}

func TestAnalyzeHandler_PEMKey(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	requestBody := AnalyzeRequest{
		Text: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyK8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8vZ8v\n-----END RSA PRIVATE KEY-----",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response AnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high', got '%s'", response.OverallRisk)
	}

	foundPEM := false
	for _, finding := range response.Findings {
		if finding.Type == "pem_private_key" {
			foundPEM = true
			break
		}
	}
	if !foundPEM {
		t.Error("Expected PEM private key finding")
	}
}

func TestAnalyzeHandler_JWT(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	requestBody := AnalyzeRequest{
		Text: "token = \"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U\"",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response AnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.OverallRisk != "high" {
		t.Errorf("Expected overall_risk 'high', got '%s'", response.OverallRisk)
	}

	foundJWT := false
	for _, finding := range response.Findings {
		if finding.Type == "jwt_token" {
			foundJWT = true
			break
		}
	}
	if !foundJWT {
		t.Error("Expected JWT token finding")
	}
}

func TestAnalyzeHandler_RequestSizeLimit(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create a request body larger than MaxRequestSize (1MB)
	largeText := strings.Repeat("a", MaxRequestSize+1)
	requestBody := AnalyzeRequest{
		Text: largeText,
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should reject large request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for oversized request, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter()
	ip := "127.0.0.1"

	// Should allow requests up to the limit
	for i := 0; i < MaxRequestsPerWindow; i++ {
		if !rl.Allow(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should reject request beyond limit
	if rl.Allow(ip) {
		t.Error("Request beyond limit should be rejected")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter()
	ip1 := "127.0.0.1"
	ip2 := "192.168.1.1"

	// Exhaust limit for IP1
	for i := 0; i < MaxRequestsPerWindow; i++ {
		rl.Allow(ip1)
	}

	// IP2 should still be allowed
	if !rl.Allow(ip2) {
		t.Error("Different IP should still be allowed")
	}

	// IP1 should be rejected
	if rl.Allow(ip1) {
		t.Error("IP1 should be rate limited")
	}
}

func TestRateLimiter_TimeWindow(t *testing.T) {
	rl := NewRateLimiter()
	ip := "127.0.0.1"

	// Exhaust limit
	for i := 0; i < MaxRequestsPerWindow; i++ {
		rl.Allow(ip)
	}

	// Should be rejected
	if rl.Allow(ip) {
		t.Error("Should be rate limited")
	}

	// Manually expire old requests by manipulating the internal state
	// This is a bit of a hack, but necessary to test time-based expiration
	rl.mu.Lock()
	rl.requests[ip] = []time.Time{time.Now().Add(-RateLimitWindow - time.Second)}
	rl.mu.Unlock()

	// Should now be allowed (old requests expired)
	if !rl.Allow(ip) {
		t.Error("Should be allowed after time window expires")
	}
}

func TestAnalyzeHandler_RateLimit(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	requestBody := AnalyzeRequest{
		Text: "test",
	}
	body, _ := json.Marshal(requestBody)

	// Make requests up to the limit
	for i := 0; i < MaxRequestsPerWindow; i++ {
		req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got status %d", i+1, w.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["error"] != "Rate limit exceeded" {
		t.Errorf("Expected rate limit error, got '%s'", response["error"])
	}
}

func TestAnalyzeHandler_ResponseStructure(t *testing.T) {
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	requestBody := AnalyzeRequest{
		Text: "password = \"secret123\"",
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response AnalyzeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify response structure
	if response.OverallRisk == "" {
		t.Error("overall_risk should not be empty")
	}

	if response.RiskRationale == "" {
		t.Error("risk_rationale should not be empty")
	}

	if response.Findings == nil {
		t.Error("findings should not be nil")
	}

	// Verify finding structure
	if len(response.Findings) > 0 {
		finding := response.Findings[0]
		if finding.Type == "" {
			t.Error("finding.type should not be empty")
		}
		if finding.Severity == "" {
			t.Error("finding.severity should not be empty")
		}
		if finding.Confidence == "" {
			t.Error("finding.confidence should not be empty")
		}
		if finding.Reason == "" {
			t.Error("finding.reason should not be empty")
		}
		if finding.LineNumber == 0 {
			t.Error("finding.line_number should not be 0")
		}
	}
}

func TestAnalyzeHandler_NoInputLogging(t *testing.T) {
	// This test verifies that the server doesn't log user input
	// We can't directly test this, but we can verify that error responses
	// don't include the input text
	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Send invalid JSON with sensitive data
	sensitiveData := "password = \"secret123\""
	req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader([]byte(sensitiveData)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify response doesn't contain the sensitive data
	responseBody := w.Body.String()
	if strings.Contains(responseBody, sensitiveData) {
		t.Error("Response should not contain user input")
	}
}

func TestUploadAndView_TestMode(t *testing.T) {
	// Set test mode
	os.Setenv("BACKEND_TEST_MODE", "1")
	defer os.Unsetenv("BACKEND_TEST_MODE")

	srv := NewServer()
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Create a test file upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	testContent := []byte("test file content")
	part.Write(testContent)
	writer.Close()

	// Upload file
	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &uploadResp); err != nil {
		t.Fatalf("Failed to unmarshal upload response: %v", err)
	}

	if uploadResp.FileID == "" {
		t.Error("Expected fileId in upload response")
	}
	if uploadResp.ViewLink == "" {
		t.Error("Expected viewLink in upload response")
	}

	// First view should succeed
	viewPath := strings.TrimPrefix(uploadResp.ViewLink, "http://localhost:8080")
	req2 := httptest.NewRequest(http.MethodGet, viewPath, nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected first view status %d, got %d", http.StatusOK, w2.Code)
	}
	if w2.Header().Get("Content-Type") == "" {
		t.Error("Expected Content-Type header in first view")
	}
	if !bytes.Equal(w2.Body.Bytes(), testContent) {
		t.Error("First view should return the uploaded file content")
	}

	// Second view should return 410 with LINK_USED
	req3 := httptest.NewRequest(http.MethodGet, viewPath, nil)
	w3 := httptest.NewRecorder()
	mux.ServeHTTP(w3, req3)

	if w3.Code != http.StatusGone {
		t.Errorf("Expected second view status %d, got %d", http.StatusGone, w3.Code)
	}
	bodyText := w3.Body.String()
	if !strings.Contains(bodyText, "LINK_USED") {
		t.Errorf("Expected 'LINK_USED' in second view response, got: %s", bodyText)
	}
}

