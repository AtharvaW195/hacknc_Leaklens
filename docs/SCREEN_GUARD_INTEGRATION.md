# Screen Guard Service Integration Architecture

## Current Architecture

### Go Server (`server/server.go`)
- **Entry Point**: `main.go` → `runServer()` → `server.NewServer().Start()`
- **Port**: Default `:8080` (configurable via `--addr` flag)
- **Routing**: Uses `http.ServeMux`, routes registered in `RegisterRoutes()`
- **Existing Endpoints**:
  - `GET /health` - Health check
  - `POST /analyze` - Text analysis
  - `POST /api/analyze-text` - Alias for extension
  - `POST /api/upload` - File upload
  - `POST /api/generate-upload-url` - Generate presigned URL
  - `GET /view/<id>` - One-time view links
- **Security**: Rate limiting (100 req/min per IP), CORS enabled, no input logging
- **Config**: Loads `.env` file, supports `AWS_BUCKET_NAME`, `VIEW_LINK_BASE_URL`

### Python Screen Guard Service (`screen_guard_service/`)
- **Entry Point**: `python3 -m screen_guard_service` (via `__main__.py`)
- **Startup**: Reads `.env` from `screen_guard_service/` directory, builds CLI args, calls `run_monitor.py`
- **Modes**: `manual` (continuous monitoring) or `auto` (platform-aware, starts/stops with screen sharing)
- **Process Type**: Long-running Python process, monitors screen captures
- **Output**: Logs and detection artifacts in `monitor_output/runs/<run_id>/`
- **Shutdown**: Handles `KeyboardInterrupt` (Ctrl+C) gracefully

### Browser Extension (`extension/`)
- **Manifest**: Manifest V3, service worker (`background.js`), content script (`content.js`), popup (`popup.html`)
- **API Communication**: Calls `http://localhost:8080` endpoints
- **Current Features**: Paste interception, link scanning, secure upload, stats tracking
- **Storage**: Uses `chrome.storage.sync` for settings, `chrome.storage.local` for stats

## New Integration Flow

### Control Plane Architecture

```
Extension (Toggle Button)
    ↓
POST /api/screen-guard/start or /stop
    ↓
Go Server (server/server.go)
    ↓
Screen Guard Manager (internal/screenguard/manager.go)
    ↓
Python Process (python3 -m screen_guard_service)
```

### New Components

1. **Screen Guard Manager** (`internal/screenguard/manager.go`)
   - Manages Python process lifecycle
   - Tracks state (running, pid, startedAt, lastError)
   - Enforces single-instance behavior (idempotent start/stop)
   - Handles graceful shutdown (SIGTERM) with timeout escalation to SIGKILL
   - Cross-platform process management (Windows/Unix compatible)

2. **New API Endpoints** (`server/server.go`)
   - `POST /api/screen-guard/start` - Start the service
   - `POST /api/screen-guard/stop` - Stop the service
   - `GET /api/screen-guard/status` - Get current status
   - `GET /api/screen-guard/health` - Optional health check (reuses existing `/health`)

3. **Extension Updates** (`extension/`)
   - New command: "Toggle Screen Guard" (or status bar indicator)
   - Calls `/api/screen-guard/status` to check current state
   - Calls `/start` or `/stop` based on current state
   - Polls `/status` until desired state reached
   - Shows status bar indicator: "Screen Guard: On/Off"
   - Toast notifications for success/failure

### State Management

**Manager State**:
- `running: bool` - Whether service is currently running
- `pid: int` - Process ID (if running)
- `startedAt: time.Time` - When service was started
- `lastError: string` - Last error message (if any)
- `logFile: string` - Path to Python service log file

**Persistence**:
- In-memory state (resets on server restart)
- PID file optional (for cross-process coordination if needed)
- Log file location tracked for debugging

### Process Management Details

**Starting**:
1. Check if already running → return current status (idempotent)
2. Resolve Python executable path (check `python3`, `python`, or env var)
3. Resolve service module path (relative to repo root: `screen_guard_service/`)
4. Spawn process with proper working directory
5. Wait for readiness (optional health check or timeout)
6. Track PID and start time

**Stopping**:
1. Check if not running → return success (idempotent)
2. Send SIGTERM (graceful shutdown)
3. Wait up to 5 seconds for process to exit
4. If still running, send SIGKILL (force kill)
5. Clean up state

**Cross-Platform Considerations**:
- Windows: Use `os/exec` with `cmd.exe /c` or direct Python invocation
- Unix: Use `os/exec` with signal handling
- Signal handling: `syscall.SIGTERM` and `syscall.SIGKILL` (Unix), `taskkill` (Windows)

### Security Considerations

- **Local-only**: All endpoints bind to `127.0.0.1` (localhost)
- **No authentication**: Since it's local-only, basic origin check may be sufficient
- **Path validation**: Ensure Python path and service path are within repo boundaries
- **Process isolation**: Python process runs with same user permissions as Go server

### Error Handling

- **Python not found**: Return clear error message
- **Service path not found**: Return clear error message
- **Process start failure**: Capture stderr, return error
- **Process already running**: Detect via PID check, return current status
- **Stop timeout**: Log warning, force kill, return success

### Configuration

**Environment Variables** (optional):
- `SCREEN_GUARD_PYTHON_PATH` - Override Python executable path
- `SCREEN_GUARD_SERVICE_DIR` - Override service directory path
- `SCREEN_GUARD_LOG_DIR` - Override log directory

**Defaults**:
- Python: `python3` or `python` (auto-detect)
- Service dir: `./screen_guard_service` (relative to repo root)
- Log dir: `./screen_guard_service/monitor_output` (default from service)

## Integration Contract

### Request/Response Formats

**POST /api/screen-guard/start**
```json
Request: {} (empty body)
Response: {
  "success": true,
  "status": {
    "running": true,
    "pid": 12345,
    "startedAt": "2024-01-01T12:00:00Z",
    "lastError": "",
    "logFile": "./screen_guard_service/monitor_output/runs/.../runtime.log"
  }
}
```

**POST /api/screen-guard/stop**
```json
Request: {} (empty body)
Response: {
  "success": true,
  "status": {
    "running": false,
    "pid": 0,
    "startedAt": "0001-01-01T00:00:00Z",
    "lastError": "",
    "logFile": ""
  }
}
```

**GET /api/screen-guard/status**
```json
Response: {
  "running": true,
  "pid": 12345,
  "startedAt": "2024-01-01T12:00:00Z",
  "lastError": "",
  "logFile": "./screen_guard_service/monitor_output/runs/.../runtime.log"
}
```

## Testing Strategy

1. **Unit Tests** (`internal/screenguard/manager_test.go`):
   - Test idempotent start/stop
   - Test state tracking
   - Test error handling
   - Mock process spawning

2. **Integration Tests** (`server/server_test.go`):
   - Test endpoint handlers
   - Test status reporting
   - Test error responses

3. **Manual Testing**:
   - Start service via extension
   - Verify Python process running
   - Stop service via extension
   - Verify Python process stopped
   - Test repeated toggles (idempotency)

## Documentation Updates Required

1. **README.md**: Add section on Screen Guard Service integration
2. **DEMO_FLOW.md**: Add step for toggling Screen Guard Service
3. **ARCHITECTURE.md**: Document new components and flow
4. **API Documentation**: Document new endpoints

