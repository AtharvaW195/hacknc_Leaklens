package main

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

// In-memory store for one-time links (for demo purposes)
var (
    viewStore   = make(map[string]*FileMetadata)
    storeMu     sync.RWMutex
    bucketName  string
    viewBaseURL string
    // Shared detector engine instance (initialized once, reused for all requests)
    detectorEngine *detector.Engine
)

func init() {
    loadEnv()
    // Initialize shared detector engine once
    detectorEngine = detector.NewEngine()
}

// loadEnv reads .env from the current directory and sets env vars. Keeps secrets out of the shell.
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

type FileMetadata struct {
    S3Key     string
    OriginalName string
    ContentType  string
    UploadTime   time.Time
    Viewed       bool
}

type UploadRequest struct {
    FileName string `json:"fileName"`
    FileType string `json:"fileType"`
}

type UploadResponse struct {
    UploadURL string `json:"uploadUrl"`
    ViewLink  string `json:"viewLink"`
    FileID    string `json:"fileId"`
}

func setCORS(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "*")
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
    setCORS(w)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func main() {
    bucketName = os.Getenv("AWS_BUCKET_NAME")
    viewBaseURL = os.Getenv("VIEW_LINK_BASE_URL")
    if bucketName == "" {
        log.Println("WARNING: AWS_BUCKET_NAME not set. Put it in backend/.env or set the env var. Uploads will fail.")
        bucketName = "guardrail-demo-bucket"
    }
    if viewBaseURL == "" {
        viewBaseURL = "http://localhost:8080"
    }
    viewBaseURL = strings.TrimSuffix(viewBaseURL, "/")

    // Load AWS Config
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        log.Fatalf("unable to load SDK config, %v", err)
    }

    s3Client := s3.NewFromConfig(cfg)
    presigner := s3.NewPresignClient(s3Client)

    // Proxy upload: browser sends file to backend, backend uploads to S3. Works from any webpage (no S3 CORS).
    http.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
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
        key := fmt.Sprintf("uploads/%s/%s", fileID, fileHeader.Filename)
        contentType := fileHeader.Header.Get("Content-Type")
        if contentType == "" {
            contentType = "application/octet-stream"
        }
        _, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
            Bucket:      aws.String(bucketName),
            Key:         aws.String(key),
            Body:        f,
            ContentType: aws.String(contentType),
        })
        if err != nil {
            log.Printf("S3 PutObject failed: %v", err)
            writeJSONError(w, "Upload to storage failed. Check AWS credentials and bucket name in .env", http.StatusInternalServerError)
            return
        }
        storeMu.Lock()
        viewStore[fileID] = &FileMetadata{
            S3Key:        key,
            OriginalName: fileHeader.Filename,
            ContentType:  contentType,
            UploadTime:   time.Now(),
            Viewed:       false,
        }
        storeMu.Unlock()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(UploadResponse{
            ViewLink: fmt.Sprintf("%s/view/%s", viewBaseURL, fileID),
            FileID:   fileID,
        })
    })

    http.HandleFunc("/api/generate-upload-url", func(w http.ResponseWriter, r *http.Request) {
        // CORS headers
        w.Header().Set("Access-Control-Allow-Origin", "*") // For demo; restrict in prod
        w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        if r.Method != "POST" {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        var req UploadRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        // Validate File Type (Simple check)
        ext := strings.ToLower(filepath.Ext(req.FileName))
        allowed := map[string]bool{
            ".pdf": true, ".docx": true, ".doc": true, ".xlsx": true, ".xls": true, ".csv": true,
            ".pptx": true, ".txt": true, ".rtf": true, ".pem": true, ".key": true, ".env": true,
            ".json": true, ".xml": true, ".yaml": true, ".yml": true, ".zip": true, ".tar": true, ".gz": true,
        }
        if !allowed[ext] {
            http.Error(w, "File type not allowed", http.StatusForbidden)
            return
        }

        fileID := uuid.New().String()
        key := fmt.Sprintf("uploads/%s/%s", fileID, req.FileName)

        // Generate Presigned PUT URL
        presignedReq, err := presigner.PresignPutObject(context.TODO(), &s3.PutObjectInput{
            Bucket: aws.String(bucketName),
            Key:    aws.String(key),
            ContentType: aws.String(req.FileType),
        }, s3.WithPresignExpires(15*time.Minute))

        if err != nil {
            log.Printf("Failed to presign request: %v", err)
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }

        // Store metadata
        storeMu.Lock()
        viewStore[fileID] = &FileMetadata{
            S3Key:        key,
            OriginalName: req.FileName,
            ContentType:  req.FileType,
            UploadTime:   time.Now(),
            Viewed:       false,
        }
        storeMu.Unlock()

        resp := UploadResponse{
            UploadURL: presignedReq.URL,
            ViewLink:  fmt.Sprintf("%s/view/%s", viewBaseURL, fileID),
            FileID:    fileID,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    })

    http.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
        // Extract ID from URL path
        parts := strings.Split(r.URL.Path, "/")
        if len(parts) < 3 {
             http.NotFound(w, r)
             return
        }
        fileID := parts[2]

        storeMu.Lock()
        meta, exists := viewStore[fileID]
        if !exists {
            storeMu.Unlock()
            http.NotFound(w, r)
            return
        }

        if meta.Viewed {
            storeMu.Unlock()
            http.Error(w, "This link has expired or already been viewed.", http.StatusGone)
            return
        }

        // Mark as viewed strictly (One-Time Link)
        meta.Viewed = true 
        storeMu.Unlock()

        // Generate Presigned GET URL for view-only (inline = display in browser when possible)
        presignedGet, err := presigner.PresignGetObject(context.TODO(), &s3.GetObjectInput{
            Bucket:                    aws.String(bucketName),
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
    })

    // Paste analysis endpoint
    http.HandleFunc("/api/analyze-text", func(w http.ResponseWriter, r *http.Request) {
        setCORS(w)
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        if r.Method != "POST" {
            writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        // Enforce 100KB limit
        const maxSize = 100 * 1024 // 100KB
        r.Body = http.MaxBytesReader(w, r.Body, maxSize)

        var req struct {
            Text string `json:"text"`
        }
        decoder := json.NewDecoder(r.Body)
        if err := decoder.Decode(&req); err != nil {
            if err == io.EOF {
                writeJSONError(w, "Request body is required", http.StatusBadRequest)
            } else if strings.Contains(err.Error(), "request body too large") || strings.Contains(err.Error(), "http: request body too large") {
                writeJSONError(w, "Payload too large (max 100KB)", http.StatusRequestEntityTooLarge)
            } else {
                writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
            }
            return
        }

        // Reject empty text
        if strings.TrimSpace(req.Text) == "" {
            writeJSONError(w, "Text cannot be empty", http.StatusBadRequest)
            return
        }

        // Analyze text using shared engine
        result := detectorEngine.Analyze(req.Text)

        // Safety check: ensure no finding preview equals a substring of input longer than 8 chars
        for i := range result.Findings {
            finding := &result.Findings[i]
            if len(finding.Reason) > 8 && strings.Contains(req.Text, finding.Reason) {
                finding.Reason = "[REDACTED]"
            }
        }

        // Normalize risk levels to uppercase for stable API
        riskLevel := strings.ToUpper(result.OverallRisk)

        // Build response
        response := map[string]interface{}{
            "overall_risk":   riskLevel,
            "risk_rationale": result.RiskRationale,
            "findings":       result.Findings,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    // Add /analyze endpoint (alias for /api/analyze-text) for extension compatibility
    http.HandleFunc("/analyze", func(w http.ResponseWriter, r *http.Request) {
        setCORS(w)
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        if r.Method != "POST" {
            writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        // Enforce 100KB limit
        const maxSize = 100 * 1024 // 100KB
        r.Body = http.MaxBytesReader(w, r.Body, maxSize)

        var req struct {
            Text string `json:"text"`
        }
        decoder := json.NewDecoder(r.Body)
        if err := decoder.Decode(&req); err != nil {
            if err == io.EOF {
                writeJSONError(w, "Request body is required", http.StatusBadRequest)
            } else if strings.Contains(err.Error(), "request body too large") || strings.Contains(err.Error(), "http: request body too large") {
                writeJSONError(w, "Payload too large (max 100KB)", http.StatusRequestEntityTooLarge)
            } else {
                writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
            }
            return
        }

        // Reject empty text
        if strings.TrimSpace(req.Text) == "" {
            writeJSONError(w, "Text cannot be empty", http.StatusBadRequest)
            return
        }

        // Analyze text using shared engine
        result := detectorEngine.Analyze(req.Text)

        // Safety check: ensure no finding preview equals a substring of input longer than 8 chars
        for i := range result.Findings {
            finding := &result.Findings[i]
            if len(finding.Reason) > 8 && strings.Contains(req.Text, finding.Reason) {
                finding.Reason = "[REDACTED]"
            }
        }

        // Normalize risk levels to lowercase for consistency with pasteguard server
        riskLevel := strings.ToLower(result.OverallRisk)

        // Build response
        response := map[string]interface{}{
            "overall_risk":   riskLevel,
            "risk_rationale": result.RiskRationale,
            "findings":       result.Findings,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    log.Println("Server starting on :8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
