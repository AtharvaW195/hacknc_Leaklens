# Paste Analysis Integration - Audit Report

## 1. Current Backend Configuration

### Backend Port
- **Port**: `8080`
- **Base URL**: `http://localhost:8080`
- **Location**: `backend/main.go` (line 304-305)

### Extension Base URL Usage
- **manifest.json**: `"http://localhost:8080/*"` (line 14, host_permissions)
- **background.js**: `const UPLOAD_URL = "http://localhost:8080/api/upload"` (line 1)
- **content.js**: `const API_BASE = "http://localhost:8080"` (line 182), `const UPLOAD_URL = "http://localhost:8080/api/upload"` (line 64)
- **popup.js**: `const API_ENDPOINT = "http://localhost:8080/api"` (line 12)

### Fetch Call Locations
- **background.js** (line 13): Direct `fetch()` to `/api/upload` - handles file uploads via message passing
- **content.js** (line 80): `XMLHttpRequest` to `/api/upload` - direct upload from content script
- **popup.js**: No direct fetch calls found (uses content.js or background.js)

**Note**: Content scripts can make direct fetch calls because `http://localhost:8080/*` is in `host_permissions`.

## 2. Module Structure Analysis

### `/backend` Directory
- **Module**: `module backend` (separate Go module)
- **Package**: `package main`
- **Dependencies**: AWS SDK, UUID
- **Purpose**: File upload service (S3 integration)
- **Current Endpoints**:
  - `POST /api/upload` - File upload handler
  - `POST /api/generate-upload-url` - Presigned URL generation
  - `GET /view/{fileID}` - One-time view link
- **Port**: 8080
- **CORS**: Enabled (`Access-Control-Allow-Origin: *`)

### `/detector` Directory (Root Module)
- **Module**: Part of `module pasteguard` (root go.mod)
- **Package**: `package detector`
- **Dependencies**: Standard library only
- **Purpose**: Secret detection engine
- **Key Components**:
  - `engine.go` - Core detection engine
  - `rule.go` - Rule interface
  - `pem_rule.go`, `jwt_rule.go`, `password_rule.go`, `token_heuristics_rule.go` - Detection rules
- **Export**: `NewEngine()`, `Engine.Analyze(text string)`, `AnalysisResult`

### `/server` Directory (Root Module)
- **Module**: Part of `module pasteguard` (root go.mod)
- **Package**: `package server`
- **Dependencies**: `pasteguard/detector`
- **Purpose**: HTTP server for pasteguard CLI tool
- **Current Endpoints**:
  - `GET /health` - Health check
  - `POST /analyze` - Text analysis (uses detector)
- **Port**: 8787 (default)
- **CORS**: Not explicitly set (needs to be added for extension)

### Root Module (`pasteguard`)
- **Module**: `module pasteguard`
- **Main Entry**: `main.go` (CLI tool)
- **Packages**: `detector`, `server`
- **Port**: 8787 (when running `pasteguard serve`)

## 3. Integration Path Proposal

### Option A: Add Endpoint to `/backend` (Recommended - Smallest Change)

**Rationale**: 
- Extension already configured for port 8080
- `/backend` already has CORS configured
- Minimal changes to extension code
- Keeps file upload and text analysis in same service

**Changes Required**:

1. **backend/go.mod**
   - Add dependency: `pasteguard/detector` (requires path-based import or module replacement)
   - **Challenge**: `/backend` is separate module, `/detector` is in root module
   - **Solution**: Use Go workspace OR move detector to shared location OR use replace directive

2. **backend/main.go**
   - Import detector package (need to resolve module path)
   - Add handler: `POST /api/analyze-text`
   - Use `detector.NewEngine().Analyze(text)` 
   - Return JSON response matching pasteguard format

3. **Extension Files** (if needed):
   - Add fetch call in `content.js` or route via `background.js`

**Module Path Challenge**:
- `/backend` module cannot directly import `pasteguard/detector` (different modules)
- **Solutions**:
  - **Option A1**: Use Go workspace (`go.work`) to link modules
  - **Option A2**: Move `/detector` to shared package or vendor it
  - **Option A3**: Use `replace` directive in backend/go.mod: `replace pasteguard/detector => ../detector`
  - **Option A4**: Copy detector code to backend (not recommended)

### Option B: Use Existing Pasteguard Server (Port 8787)

**Rationale**:
- Detector already integrated
- No module path issues
- Server package already exists

**Changes Required**:

1. **server/server.go**
   - Add CORS headers to `/analyze` endpoint
   - Change endpoint from `/analyze` to `/api/analyze-text` (or add new route)

2. **extension/manifest.json**
   - Add `"http://localhost:8787/*"` to host_permissions

3. **Extension Files**
   - Update base URL to `http://localhost:8787`
   - Add fetch call to `/api/analyze-text`

**Drawback**: Requires running two servers (8080 for uploads, 8787 for analysis)

### Option C: Merge Backends (Larger Refactor)

**Rationale**:
- Single server for all functionality
- Unified CORS and security

**Changes Required**:
- Move file upload logic from `/backend` to root module
- Consolidate into single server
- **Not recommended** for "smallest integration path"

## 4. Recommended Integration Path (Smallest Change)

### Recommended: Option A with Module Replace

**Files to Change**:

1. **backend/go.mod**
   ```go
   module backend
   
   go 1.25.0
   
   replace pasteguard/detector => ../detector
   
   require (
       pasteguard/detector v0.0.0
       // ... existing deps
   )
   ```

2. **backend/main.go**
   - Add import: `"pasteguard/detector"`
   - Add handler function:
     ```go
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
         
         var req struct {
             Text string `json:"text"`
         }
         if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
             writeJSONError(w, "Invalid JSON", http.StatusBadRequest)
             return
         }
         
         engine := detector.NewEngine()
         result := engine.Analyze(req.Text)
         
         w.Header().Set("Content-Type", "application/json")
         json.NewEncoder(w).Encode(map[string]interface{}{
             "overall_risk": result.OverallRisk,
             "risk_rationale": result.RiskRationale,
             "findings": result.Findings,
         })
     })
     ```
   - Add before `log.Println("Server starting on :8080...")`

3. **extension/content.js** (or background.js)
   - Add paste event listener
   - On paste, extract text
   - Call: `fetch("http://localhost:8080/api/analyze-text", { method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({text: pastedText}) })`
   - Handle response and show warnings

## 5. CORS and Permissions Analysis

### Current CORS Setup
- **backend/main.go**: CORS enabled via `setCORS()` function (line 79-83)
  - `Access-Control-Allow-Origin: *`
  - `Access-Control-Allow-Methods: POST, OPTIONS`
  - `Access-Control-Allow-Headers: *`

### Extension Permissions
- **manifest.json**: 
  - `host_permissions`: `["<all_urls>", "http://localhost:8080/*"]`
  - Content scripts can make direct fetch to localhost:8080

### CORS Constraints
- ✅ **Content scripts → backend (8080)**: Allowed (CORS + host_permissions)
- ✅ **Background script → backend (8080)**: Allowed (CORS + host_permissions)
- ⚠️ **Content scripts → pasteguard (8787)**: Would need CORS headers added to server/server.go

### Recommendation
- **Route via background.js** if:
  - Need to avoid CORS issues
  - Want centralized error handling
  - Need to store analysis results
- **Direct fetch from content.js** if:
  - Simpler implementation
  - Real-time analysis needed
  - Current CORS setup is sufficient

**For paste analysis**: Direct fetch from content.js is fine since backend already has CORS configured.

## 6. Exact Files to Change

### Minimal Integration (Option A)

1. **backend/go.mod**
   - Add `replace` directive
   - Add `require pasteguard/detector`

2. **backend/main.go**
   - Add import: `"pasteguard/detector"`
   - Add `/api/analyze-text` handler (before server start)

3. **extension/content.js**
   - Add paste event listener
   - Add `analyzePastedText(text)` function
   - Add UI feedback for findings

### Alternative: Use Background Script Routing

1. **backend/main.go** (same as above)

2. **extension/background.js**
   - Add message handler for `"analyze-text"` type
   - Forward to backend API
   - Return results

3. **extension/content.js**
   - Send message to background: `chrome.runtime.sendMessage({type: "analyze-text", text: pastedText})`
   - Handle response

## 7. Module Path Resolution

### Current Structure
```
hacknc/
├── go.mod (module pasteguard)
│   ├── detector/ (package detector)
│   └── server/ (package server)
└── backend/
    └── go.mod (module backend)  ← Separate module
```

### Solution: Use Replace Directive
In `backend/go.mod`:
```go
replace pasteguard/detector => ../detector
```

This allows `/backend` to import `pasteguard/detector` as if it were a dependency, resolving to the local `../detector` directory.

## Summary

**Smallest Integration Path**:
1. Add `replace` directive in `backend/go.mod`
2. Add `/api/analyze-text` endpoint in `backend/main.go` using detector
3. Add paste listener in `extension/content.js` with direct fetch call

**Files to Modify**:
- `backend/go.mod` (2 lines)
- `backend/main.go` (~30 lines)
- `extension/content.js` (~50 lines for paste handling)

**No CORS issues**: Backend already has CORS configured for extension access.

