# End-to-End Logging Implementation Summary

## Overview

Implemented comprehensive structured logging with requestId propagation across all components:
- Extension (Popup + Background)
- Go Proxy Server
- Python API Server
- PowerShell Startup Script

## Files Modified

### 1. Extension Files

#### `extension/logger.js` (NEW)
- Structured logging utility class
- RequestId generation function
- Consistent log format: `[timestamp] | [level] | [component] | [event] | [requestId] | [state] | [message]`

#### `extension/popup.html`
- Added `<script src="logger.js"></script>` before popup.js

#### `extension/popup.js`
**Changes:**
- Added logger initialization: `const logger = new ExtensionLogger('POPUP')`
- All functions now generate/use requestId
- `startScreenGuard()`: Logs INIT, SENDING, SUCCESS/FAILED with requestId
- `stopScreenGuard()`: Logs INIT, SENDING, SUCCESS/FAILED with requestId
- `checkScreenGuardStatus()`: Logs with requestId
- `updateScreenGuardUI()`: Logs UI state changes
- `initScreenGuard()`: Logs toggle clicks, polling, heartbeat
- All fetch calls include `X-Request-Id` header

**Key additions:**
```javascript
const requestId = generateRequestId();
logger.info('TOGGLE_CLICK', requestId, 'INIT', `Toggle changed: ${action} requested`);
// ... in fetch headers:
headers: { 
    'Content-Type': 'application/json',
    'X-Request-Id': requestId
}
```

#### `extension/background.js`
**Changes:**
- Added inline logger class (service workers can't import modules)
- Extracts requestId from messages: `const requestId = msg.requestId || generateRequestId()`
- Logs all message types with requestId

### 2. Python API Server

#### `screen_guard_service/api_server.py`
**Changes:**
- Added `setup_logging()` function for structured logging
- Added `log_structured()` function with consistent format
- All endpoints extract requestId from `X-Request-Id` header:
  ```python
  request_id = request.headers.get("X-Request-Id", f"py-{int(time.time() * 1000)}-{os.getpid()}")
  ```
- `/start` endpoint: Logs RECEIVED, PREPARE, CONFIG, SPAWN, SPAWNED, SUCCESS/FAILED
- `/stop` endpoint: Logs RECEIVED, STOPPING, TERMINATE, TERMINATED/KILLED, SUCCESS
- `/status` endpoint: Logs with requestId
- `/health` endpoint: Logs with requestId
- Added heartbeat thread (logs every 30 seconds)
- All errors include full traceback in extra_data

**Key additions:**
```python
def log_structured(level, event, request_id, state, message, extra=None):
    timestamp = datetime.utcnow().isoformat() + "Z"
    parts = [timestamp, level.upper(), "PYTHON_API", event or "-", request_id or "-", state or "-", message]
    if extra:
        parts.append(str(extra))
    log_line = " | ".join(parts)
    # ... log to appropriate handler
```

### 3. Go Proxy Server

#### `server/server.go`
**Changes:**
- Extracts requestId from incoming request or generates one
- Propagates `X-Request-Id` header to Python API
- Structured logging format matching schema
- Logs: INIT, SENDING, RESPONSE, TIMEOUT, FAILED

**Key additions:**
```go
requestID := r.Header.Get("X-Request-Id")
if requestID == "" {
    requestID = fmt.Sprintf("go-%d-%d", time.Now().UnixMilli(), os.Getpid())
    r.Header.Set("X-Request-Id", requestID)
}
log.Printf("[Screen Guard Proxy] | INFO | GO_PROXY | PROXY_REQUEST | %s | INIT | Proxying %s %s to %s", 
    requestID, r.Method, r.URL.Path, targetURL)
```

### 4. PowerShell Startup Script

#### `start.ps1`
**Changes:**
- Structured log output for process start
- Health check logging with requestId
- Log file renamed to `backend.log` and `backend_stderr.log`
- All major events logged with structured format

**Key additions:**
```powershell
$startupId = "ps1-$(Get-Date -Format 'yyyyMMddHHmmss')-$($PY_PROCESS.Id)"
$timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.fffZ"
Write-Output "$timestamp | INFO | START_PS1 | PROCESS_START | $startupId | INIT | Python process started: PID=$($PY_PROCESS.Id)"
```

## Log File Locations

- **Python stdout**: `logs/backend.log`
- **Python stderr**: `logs/backend_stderr.log`
- **Go server**: `logs/go_server.log`
- **Extension**: Browser console (popup/service worker)

## How to Use

### 1. Start Services
```powershell
.\start.ps1
```

### 2. View Logs in Real-Time

**Python logs:**
```powershell
Get-Content logs\backend.log -Wait -Tail 50
Get-Content logs\backend_stderr.log -Wait -Tail 50
```

**Go logs:**
```powershell
Get-Content logs\go_server.log -Wait -Tail 50
```

**Extension logs:**
- Open extension popup → Right-click → Inspect → Console
- Or: `chrome://extensions/` → Service worker → Console

### 3. Trace a Specific Request

1. Click "Start" in extension
2. Note the requestId from first log (e.g., `1736995236730-abc123`)
3. Search all logs:
   ```powershell
   Select-String -Path "logs\*.log" -Pattern "1736995236730-abc123"
   ```

## Example Log Flow

When you click "Start", you'll see:

```
# Popup
2026-02-15T01:33:56.730Z | INFO | POPUP | TOGGLE_CLICK | 1736995236730-abc123 | INIT | Toggle changed: START requested
2026-02-15T01:33:56.731Z | INFO | POPUP | START_REQUEST | 1736995236730-abc123 | SENDING | Sending POST to http://localhost:8080/api/screen-guard/start

# Go Proxy
2026-02-15T01:33:56.732Z | INFO | GO_PROXY | PROXY_REQUEST | 1736995236730-abc123 | INIT | Proxying POST /api/screen-guard/start to http://127.0.0.1:8081/start
2026-02-15T01:33:56.733Z | INFO | GO_PROXY | PROXY_REQUEST | 1736995236730-abc123 | SENDING | Sending request to Python API

# Python API
2026-02-15T01:33:56.734Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | RECEIVED | POST /start request received from 127.0.0.1
2026-02-15T01:33:56.735Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | SPAWN | Spawning: python -m screen_guard_service
2026-02-15T01:33:56.850Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | SPAWNED | Monitor process spawned with PID: 12345
2026-02-15T01:33:57.850Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | SUCCESS | Monitor started successfully (PID: 12345)

# Go Proxy Response
2026-02-15T01:33:57.851Z | INFO | GO_PROXY | PROXY_REQUEST | 1736995236730-abc123 | RESPONSE | Python API responded: 200 OK

# Popup Response
2026-02-15T01:33:57.852Z | INFO | POPUP | START_REQUEST | 1736995236730-abc123 | SUCCESS | Start response received
2026-02-15T01:33:57.853Z | INFO | POPUP | UI_UPDATE | 1736995236730-abc123 | RUNNING | UI updated: Running (PID: 12345)
```

## Heartbeat Logs

Python API logs heartbeat every 30 seconds:
```
2026-02-15T01:34:00.000Z | INFO | PYTHON_API | HEARTBEAT | hb-1736995240 | RUNNING | Server alive: monitor_running=true, monitor_pid=12345, uptime=240s
```

## Verification Checklist

- [ ] Extension popup console shows logs when clicking Start
- [ ] Service worker console shows BACKGROUND logs
- [ ] `logs/backend.log` contains Python API logs
- [ ] `logs/go_server.log` contains Go proxy logs
- [ ] RequestId appears in all logs for the same request
- [ ] Heartbeat logs appear every 30 seconds in Python logs
- [ ] Errors include full stack traces

## Troubleshooting

**No logs in extension:**
- Reload extension: `chrome://extensions/` → Reload
- Check console is open (F12)

**No Python logs:**
- Check `logs/backend.log` exists
- Verify Python process is running: `Get-Process | Where-Object {$_.ProcessName -like "*python*"}`

**RequestId not propagating:**
- Check Go proxy logs for requestId extraction
- Verify `X-Request-Id` header in network tab (F12 → Network)

