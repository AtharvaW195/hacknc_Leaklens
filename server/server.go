package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
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

// FileMetadata stores metadata for uploaded files
type FileMetadata struct {
	S3Key        string
	OriginalName string
	ContentType  string
	UploadTime   time.Time
	Viewed       bool
}

// UploadRequest represents a request to generate an upload URL
type UploadRequest struct {
	FileName string `json:"fileName"`
	FileType string `json:"fileType"`
}

// UploadResponse represents the response from upload endpoints
type UploadResponse struct {
	UploadURL string `json:"uploadUrl,omitempty"`
	ViewLink  string `json:"viewLink"`
	FileID    string `json:"fileId"`
}

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
	engine         *detector.Engine
	rateLimiter    *RateLimiter
	s3Client       *s3.Client
	presigner      *s3.PresignClient
	bucketName     string
	viewBaseURL    string
	viewStore      map[string]*FileMetadata
	storeMu        sync.RWMutex
	testMode       bool
	testFileStore  map[string][]byte // In-memory file storage for test mode
	testFileMu     sync.RWMutex       // Mutex for test file store
	videoMonitor   *VideoMonitorManager
}

// NewServer creates a new server instance
func NewServer() *Server {
	// Load environment variables
	loadEnv()

	// Check for test mode
	testMode := os.Getenv("BACKEND_TEST_MODE") == "1"

	bucketName := os.Getenv("AWS_BUCKET_NAME")
	viewBaseURL := os.Getenv("VIEW_LINK_BASE_URL")
	if bucketName == "" {
		log.Println("WARNING: AWS_BUCKET_NAME not set. Put it in .env or set the env var. Uploads will fail.")
		bucketName = "guardrail-demo-bucket"
	}
	if viewBaseURL == "" {
		viewBaseURL = "http://localhost:8080"
	}
	viewBaseURL = strings.TrimSuffix(viewBaseURL, "/")

	// Initialize AWS config (optional - only needed for uploads, skip in test mode)
	var s3Client *s3.Client
	var presigner *s3.PresignClient
	if !testMode {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Printf("WARNING: Unable to load AWS config: %v. File uploads will not work.", err)
		} else {
			s3Client = s3.NewFromConfig(cfg)
			presigner = s3.NewPresignClient(s3Client)
		}
	} else {
		log.Println("BACKEND_TEST_MODE enabled: using in-memory storage, no AWS calls")
	}

	return &Server{
		engine:        detector.NewEngine(),
		rateLimiter:   NewRateLimiter(),
		s3Client:      s3Client,
		presigner:     presigner,
		bucketName:    bucketName,
		viewBaseURL:   viewBaseURL,
		viewStore:     make(map[string]*FileMetadata),
		testMode:      testMode,
		testFileStore: make(map[string][]byte),
		videoMonitor:  NewVideoMonitorManager(),
	}
}

// loadEnv reads .env from the current directory and sets env vars
func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		val = strings.Trim(val, "\"'")
		if key != "" {
			os.Setenv(key, val)
		}
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

// setCORS sets CORS headers
func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
	w.Header().Set("Access-Control-Allow-Headers", "*")
}

// writeJSONError writes a JSON error response
func writeJSONError(w http.ResponseWriter, message string, code int) {
	setCORS(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
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
	setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

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
	json.NewEncoder(w).Encode(map[string]bool{
		"ok": true,
	})
}

// uploadHandler handles POST /api/upload requests (proxy upload to S3)
func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	maxUpload := int64(50 << 20) // 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil {
		writeJSONError(w, "Bad request: need multipart form with 'file' field", http.StatusBadRequest)
		return
	}
	form := r.MultipartForm
	files := form.File["file"]
	if len(files) == 0 {
		writeJSONError(w, "No file in 'file' field", http.StatusBadRequest)
		return
	}
	fileHeader := files[0]
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = "."
	}
	allowed := map[string]bool{
		".pdf": true, ".docx": true, ".doc": true, ".xlsx": true, ".xls": true, ".csv": true,
		".pptx": true, ".txt": true, ".rtf": true, ".pem": true, ".key": true, ".env": true,
		".json": true, ".xml": true, ".yaml": true, ".yml": true, ".zip": true, ".tar": true, ".gz": true,
	}
	if !allowed[ext] {
		writeJSONError(w, "File type not allowed. Use PDF, Office, CSV, TXT, ZIP, etc.", http.StatusForbidden)
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		writeJSONError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	fileID := uuid.New().String()
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if s.testMode {
		// Test mode: store file bytes in memory
		fileBytes, err := io.ReadAll(f)
		if err != nil {
			writeJSONError(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		s.testFileMu.Lock()
		s.testFileStore[fileID] = fileBytes
		s.testFileMu.Unlock()
		s.storeMu.Lock()
		s.viewStore[fileID] = &FileMetadata{
			S3Key:        "", // Not used in test mode
			OriginalName: fileHeader.Filename,
			ContentType:  contentType,
			UploadTime:   time.Now(),
			Viewed:       false,
		}
		s.storeMu.Unlock()
	} else {
		// Production mode: upload to S3
		if s.s3Client == nil {
			writeJSONError(w, "S3 client not configured. Check AWS credentials.", http.StatusServiceUnavailable)
			return
		}
		key := fmt.Sprintf("uploads/%s/%s", fileID, fileHeader.Filename)
		_, err = s.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(s.bucketName),
			Key:         aws.String(key),
			Body:        f,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			log.Printf("S3 PutObject failed: %v", err)
			writeJSONError(w, "Upload to storage failed. Check AWS credentials and bucket name in .env", http.StatusInternalServerError)
			return
		}
		s.storeMu.Lock()
		s.viewStore[fileID] = &FileMetadata{
			S3Key:        key,
			OriginalName: fileHeader.Filename,
			ContentType:  contentType,
			UploadTime:   time.Now(),
			Viewed:       false,
		}
		s.storeMu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UploadResponse{
		ViewLink: fmt.Sprintf("%s/view/%s", s.viewBaseURL, fileID),
		FileID:   fileID,
	})
}

// generateUploadURLHandler handles POST /api/generate-upload-url requests
func (s *Server) generateUploadURLHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.presigner == nil {
		writeJSONError(w, "S3 client not configured. Check AWS credentials.", http.StatusServiceUnavailable)
		return
	}

	var req UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate File Type
	ext := strings.ToLower(filepath.Ext(req.FileName))
	allowed := map[string]bool{
		".pdf": true, ".docx": true, ".doc": true, ".xlsx": true, ".xls": true, ".csv": true,
		".pptx": true, ".txt": true, ".rtf": true, ".pem": true, ".key": true, ".env": true,
		".json": true, ".xml": true, ".yaml": true, ".yml": true, ".zip": true, ".tar": true, ".gz": true,
	}
	if !allowed[ext] {
		writeJSONError(w, "File type not allowed", http.StatusForbidden)
		return
	}

	fileID := uuid.New().String()
	key := fmt.Sprintf("uploads/%s/%s", fileID, req.FileName)

	// Generate Presigned PUT URL
	presignedReq, err := s.presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		ContentType: aws.String(req.FileType),
	}, s3.WithPresignExpires(15*time.Minute))

	if err != nil {
		log.Printf("Failed to presign request: %v", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Store metadata
	s.storeMu.Lock()
	s.viewStore[fileID] = &FileMetadata{
		S3Key:        key,
		OriginalName: req.FileName,
		ContentType:  req.FileType,
		UploadTime:   time.Now(),
		Viewed:       false,
	}
	s.storeMu.Unlock()

	resp := UploadResponse{
		UploadURL: presignedReq.URL,
		ViewLink:  fmt.Sprintf("%s/view/%s", s.viewBaseURL, fileID),
		FileID:    fileID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// viewHandler handles GET /view/<id> requests (one-time view links)
func (s *Server) viewHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	fileID := parts[2]

	s.storeMu.Lock()
	meta, exists := s.viewStore[fileID]
	if !exists {
		s.storeMu.Unlock()
		http.NotFound(w, r)
		return
	}

	if meta.Viewed {
		s.storeMu.Unlock()
		// Return stable error message for test mode
		if s.testMode {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("LINK_USED"))
		} else {
			http.Error(w, "This link has expired or already been viewed.", http.StatusGone)
		}
		return
	}

	// Mark as viewed strictly (One-Time Link)
	meta.Viewed = true
	s.storeMu.Unlock()

	if s.testMode {
		// Test mode: serve from in-memory store
		s.testFileMu.RLock()
		fileBytes, exists := s.testFileStore[fileID]
		s.testFileMu.RUnlock()
		if !exists {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", meta.ContentType)
		w.WriteHeader(http.StatusOK)
		w.Write(fileBytes)
		return
	}

	// Production mode: generate presigned S3 URL
	if s.presigner == nil {
		http.Error(w, "S3 client not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate Presigned GET URL for view-only (inline = display in browser when possible)
	presignedGet, err := s.presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket:                    aws.String(s.bucketName),
		Key:                       aws.String(meta.S3Key),
		ResponseContentDisposition: aws.String("inline"), // view-only: open in browser, don't force download
	}, s3.WithPresignExpires(5*time.Minute))

	if err != nil {
		log.Printf("Failed to generate view link: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Redirect to the S3 URL
	http.Redirect(w, r, presignedGet.URL, http.StatusTemporaryRedirect)
}

// videoMonitorStartHandler handles POST /api/video-monitor/start
func (s *Server) videoMonitorStartHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		log.Printf("[VIDEO_MONITOR] Invalid method: %s (expected POST)", r.Method)
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	log.Printf("[VIDEO_MONITOR] Start request received from %s", r.RemoteAddr)
	
	if err := s.videoMonitor.Start(); err != nil {
		log.Printf("[VIDEO_MONITOR] Failed to start: %v", err)
		writeJSONError(w, fmt.Sprintf("Failed to start video monitoring: %v", err), http.StatusInternalServerError)
		return
	}
	
	status := s.videoMonitor.GetStatus()
	log.Printf("[VIDEO_MONITOR] Started successfully, status: %s", status.Status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// videoMonitorStopHandler handles POST /api/video-monitor/stop
func (s *Server) videoMonitorStopHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		log.Printf("[VIDEO_MONITOR] Invalid method: %s (expected POST)", r.Method)
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	log.Printf("[VIDEO_MONITOR] Stop request received from %s", r.RemoteAddr)
	
	if err := s.videoMonitor.Stop(); err != nil {
		log.Printf("[VIDEO_MONITOR] Failed to stop: %v", err)
		writeJSONError(w, fmt.Sprintf("Failed to stop video monitoring: %v", err), http.StatusInternalServerError)
		return
	}
	
	status := s.videoMonitor.GetStatus()
	log.Printf("[VIDEO_MONITOR] Stopped successfully, status: %s", status.Status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// videoMonitorStatusHandler handles GET /api/video-monitor/status
func (s *Server) videoMonitorStatusHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	status := s.videoMonitor.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// videoMonitorStreamHandler handles GET /api/video-monitor/stream (SSE)
func (s *Server) videoMonitorStreamHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	
	// Create client channel
	clientCh := make(chan VideoMonitorEvent, 10)
	s.videoMonitor.AddClient(clientCh)
	defer s.videoMonitor.RemoveClient(clientCh)
	
	// Send initial status
	status := s.videoMonitor.GetStatus()
	initialEvent := VideoMonitorEvent{
		Type:      "status",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"status":  status.Status,
			"message": status.Message,
		},
	}
	
	eventData, _ := json.Marshal(initialEvent)
	fmt.Fprintf(w, "data: %s\n\n", eventData)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	
	// Stream events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-clientCh:
			eventData, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling event: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", eventData)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// videoMonitorEventsHandler handles POST /api/video-monitor/events (internal endpoint for video server)
func (s *Server) videoMonitorEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var event VideoMonitorEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("[VIDEO_MONITOR] Failed to decode event: %v", err)
		writeJSONError(w, fmt.Sprintf("Invalid event format: %v", err), http.StatusBadRequest)
		return
	}
	
	// Ensure timestamp is set
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	
	log.Printf("[VIDEO_MONITOR] Received event: type=%s from %s", event.Type, r.RemoteAddr)
	if event.Type == "detection" {
		if ruleName, ok := event.Data["rule_name"].(string); ok {
			log.Printf("[VIDEO_MONITOR] Detection: %s (severity: %v)", ruleName, event.Data["severity"])
		}
	}
	
	s.videoMonitor.ReceiveEvent(event)
	w.WriteHeader(http.StatusOK)
}

// RegisterRoutes sets up HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/analyze", s.rateLimitMiddleware(s.analyzeHandler))
	mux.HandleFunc("/api/analyze-text", s.rateLimitMiddleware(s.analyzeHandler)) // Alias for extension compatibility
	mux.HandleFunc("/api/upload", s.uploadHandler)
	mux.HandleFunc("/api/generate-upload-url", s.generateUploadURLHandler)
	mux.HandleFunc("/view/", s.viewHandler)
	
	// Video monitoring routes
	mux.HandleFunc("/api/video-monitor/start", s.videoMonitorStartHandler)
	mux.HandleFunc("/api/video-monitor/stop", s.videoMonitorStopHandler)
	mux.HandleFunc("/api/video-monitor/status", s.videoMonitorStatusHandler)
	mux.HandleFunc("/api/video-monitor/stream", s.videoMonitorStreamHandler)
	mux.HandleFunc("/api/video-monitor/events", s.videoMonitorEventsHandler) // Internal endpoint for video server
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
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Log server start (without any user input)
	log.Printf("Starting pasteguard server on %s", addr)
	return server.ListenAndServe()
}
