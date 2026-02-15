# Implementation Plan: Python Service via Go Proxy + Single Startup

## Current State Analysis

### Go Server
- Entry point: `main.go` → `runServer()` → `server.NewServer().Start()`
- Default port: `:8080` (configurable via `--addr`)
- Routes: `/health`, `/analyze`, `/api/*`, `/view/<id>`
- Config: Loads `.env`, supports `AWS_BUCKET_NAME`, `VIEW_LINK_BASE_URL`
- Screen Guard Manager: Already exists in `internal/screenguard/manager.go` but spawns Python CLI directly

### Python Service
- Entry point: `python3 -m screen_guard_service` → `__main__.py` → `run_monitor.py`
- Type: CLI tool (no web API currently)
- Runs as long-running process monitoring screen shares
- No HTTP server, no `/health` endpoint
- Outputs to `monitor_output/runs/<run_id>/`

### Extension
- Base URL: `http://localhost:8080` (hardcoded)
- Calls: `/api/analyze-text`, `/api/upload`, etc.
- Screen Guard toggle: Calls `/api/screen-guard/start`, `/stop`, `/status`

## Required Changes

### Step 1: Create Python Web API Wrapper
**File**: `screen_guard_service/api_server.py`
- FastAPI web server wrapping the monitor service
- Endpoints:
  - `GET /health` - Health check (returns 200 if monitor running)
  - `POST /start` - Start monitoring (if not running)
  - `POST /stop` - Stop monitoring (graceful shutdown)
  - `GET /status` - Get current status (running, pid, etc.)
- Bind to `127.0.0.1` only (configurable via env)
- Port: Default `8081` (configurable via `SCREEN_GUARD_API_PORT`)
- Run monitor in background thread/process

### Step 2: Update Python Service Entry Point
**File**: `screen_guard_service/__main__.py`
- Add `--api` flag to start API server instead of CLI
- Default behavior unchanged (CLI mode)
- When `--api` is used, start `api_server.py` instead of `run_monitor.py`

### Step 3: Add Go Reverse Proxy
**File**: `server/server.go`
- Add reverse proxy handler for `/api/screen-guard/*` paths
- Proxy to `SCREEN_GUARD_BASE_URL` (default `http://127.0.0.1:8081`)
- SSRF protection: Only allow proxying to localhost
- Timeout: 10 seconds
- Error handling: Return 502 if Python service down

### Step 4: Update Screen Guard Manager
**File**: `internal/screenguard/manager.go`
- Change from spawning CLI to starting API server
- Command: `python3 -m screen_guard_service --api`
- Health check: Poll `http://127.0.0.1:8081/health` instead of process check
- Status: Call `/status` endpoint instead of process state

### Step 5: Create Startup Scripts
**Files**: `start.sh`, `start.ps1`
- Start Python API server in background
- Wait for health check (poll `/health` with timeout)
- Start Go server in foreground
- Handle Ctrl+C: Stop Python gracefully, then exit
- Logs: `./logs/screen_guard_service.log`, `./logs/go_server.log`

### Step 6: Update Extension (if needed)
**Files**: `extension/popup.js`, `extension/background.js`
- Already uses Go endpoints - no changes needed
- Verify no direct Python calls exist

### Step 7: Documentation
**Files**: `README.md`, `DEMO_FLOW.md`
- Update startup instructions
- Document new architecture
- Update demo flow with single command

## Request Flow (Final)

1. Extension calls `POST http://localhost:8080/api/screen-guard/start`
2. Go server receives request at `/api/screen-guard/start`
3. Go reverse proxy forwards to `http://127.0.0.1:8081/start`
4. Python API server receives request, starts monitor in background
5. Python returns status JSON
6. Go proxy returns response to extension
7. Extension updates UI (toggle ON, status display)

## Files to Create/Modify

### New Files
- `screen_guard_service/api_server.py` - FastAPI wrapper
- `start.sh` - Bash startup script
- `start.ps1` - PowerShell startup script
- `logs/` directory (gitignored)

### Modified Files
- `screen_guard_service/__main__.py` - Add `--api` flag
- `screen_guard_service/requirements.txt` - Add `fastapi`, `uvicorn`
- `server/server.go` - Add reverse proxy
- `internal/screenguard/manager.go` - Use API instead of CLI
- `extension/content.js` - Fix proceed button (already done)
- `README.md` - Update startup instructions
- `DEMO_FLOW.md` - Update demo flow
- `.gitignore` - Add `logs/`

## Environment Variables

- `SCREEN_GUARD_API_PORT` - Python API port (default: 8081)
- `SCREEN_GUARD_BASE_URL` - Python API base URL (default: http://127.0.0.1:8081)
- `GO_PORT` - Go server port (default: 8080)
- `PYTHON` - Python executable (default: python3 or python)

