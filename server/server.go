package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"pasteguard/detector"
)

const (
	// MaxRequestSize limits request body size to 1MB
	MaxRequestSize = 1 * 1024 * 1024
	// RateLimitWindow is the time window for rate limiting (1 minute)
	RateLimitWindow = 1 * time.Minute
	// MaxRequestsPerWindow limits requests per IP per window
	MaxRequestsPerWindow = 100
)

// RateLimiter tracks requests per IP
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-RateLimitWindow)

	// Clean up old requests
	times := rl.requests[ip]
	validTimes := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}

	// Check if limit exceeded
	if len(validTimes) >= MaxRequestsPerWindow {
		return false
	}

	// Add current request
	validTimes = append(validTimes, now)
	rl.requests[ip] = validTimes

	return true
}

// Server wraps the HTTP server and dependencies
type Server struct {
	engine     *detector.Engine
	rateLimiter *RateLimiter
}

// NewServer creates a new server instance
func NewServer() *Server {
	return &Server{
		engine:      detector.NewEngine(),
		rateLimiter: NewRateLimiter(),
	}
}

// AnalyzeRequest represents the JSON request body
type AnalyzeRequest struct {
	Text string `json:"text"`
}

// AnalyzeResponse represents the JSON response
type AnalyzeResponse struct {
	OverallRisk   string             `json:"overall_risk"`
	RiskRationale string             `json:"risk_rationale"`
	Findings      []detector.Finding `json:"findings"`
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// rateLimitMiddleware applies rate limiting
func (s *Server) rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !s.rateLimiter.Allow(ip) {
			// Do NOT log the IP or any request details to avoid logging inputs
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})
			return
		}
		next(w, r)
	}
}

// analyzeHandler handles POST /analyze requests
func (s *Server) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Method not allowed",
		})
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestSize)

	// Read and parse JSON
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Do NOT log the error details or body content
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	var req AnalyzeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Do NOT log the error or body content
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid JSON",
		})
		return
	}

	// Analyze the text
	result := s.engine.Analyze(req.Text)

	// Build response
	response := AnalyzeResponse{
		OverallRisk:   result.OverallRisk,
		RiskRationale: result.RiskRationale,
		Findings:      result.Findings,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Only log encoding errors, not request content
		log.Printf("Error encoding response: %v", err)
	}
}

// healthHandler provides a simple health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// RegisterRoutes sets up HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/analyze", s.rateLimitMiddleware(s.analyzeHandler))
}

// Start starts the HTTP server on the given address
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	// Create server with timeouts
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Log server start (without any user input)
	log.Printf("Starting pasteguard server on %s", addr)
	return server.ListenAndServe()
}

