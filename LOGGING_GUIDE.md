# End-to-End Logging Guide

## Log Format Schema

All logs follow this consistent format:
```
[timestamp] | [level] | [component] | [event] | [requestId] | [state] | [message] | [extra_data]
```

### Fields:
- **timestamp**: ISO 8601 UTC timestamp (e.g., `2026-02-15T01:33:56.730Z`)
- **level**: `INFO`, `WARN`, `ERROR`, `DEBUG`
- **component**: `POPUP`, `BACKGROUND`, `GO_PROXY`, `PYTHON_API`, `START_PS1`
- **event**: Event type (e.g., `TOGGLE_CLICK`, `START_REQUEST`, `PROXY_REQUEST`)
- **requestId**: Unique ID for tracking a request across components (e.g., `1736995236730-abc123`)
- **state**: Current state (e.g., `INIT`, `SENDING`, `SUCCESS`, `FAILED`)
- **message**: Human-readable message
- **extra_data**: Optional JSON object with additional context

## RequestId Propagation

Every "Start" click generates a new `requestId` that flows through:
1. **Extension Popup** → generates requestId
2. **HTTP Header** → `X-Request-Id: <requestId>`
3. **Go Proxy** → extracts/forwards requestId
4. **Python API** → extracts requestId from headers
5. **All logs** → include the same requestId

## Where to View Logs

### 1. Extension Logs (Chrome DevTools)

#### Popup Console:
1. Open extension popup (click extension icon)
2. Right-click in popup → "Inspect"
3. Go to "Console" tab
4. Filter by component: `POPUP` or search for `requestId`

#### Service Worker Console (Background):
1. Go to `chrome://extensions/`
2. Find "Privacy Guardrail" extension
3. Click "service worker" link (or "Inspect views: service worker")
4. Console tab shows `BACKGROUND` component logs

#### Content Script Console:
1. Open any webpage
2. Press F12 → Console tab
3. Look for logs from content scripts

### 2. Backend Logs (PowerShell/File System)

#### Real-time Python Logs:
```powershell
# View stdout (structured logs)
Get-Content logs\backend.log -Wait -Tail 50

# View stderr (errors)
Get-Content logs\backend_stderr.log -Wait -Tail 50

# Or use the helper script
.\view-logs.ps1 python
```

#### Go Server Logs:
```powershell
Get-Content logs\go_server.log -Wait -Tail 50
```

#### PowerShell Startup Logs:
The `start.ps1` script outputs structured logs to console. All output is also captured in log files.

### 3. Finding a Specific Request

To trace a specific request through all components:

1. **Get the requestId from popup console** (first log when you click Start)
2. **Search all log files**:
   ```powershell
   # Search Python logs
   Select-String -Path "logs\backend.log" -Pattern "your-request-id-here"
   
   # Search Go logs
   Select-String -Path "logs\go_server.log" -Pattern "your-request-id-here"
   ```

## Log Components

### Extension Components:
- **POPUP**: User interface interactions (button clicks, status updates)
- **BACKGROUND**: Service worker message handling

### Backend Components:
- **GO_PROXY**: Go server proxy layer (request forwarding)
- **PYTHON_API**: Python FastAPI server (request handling, process management)
- **START_PS1**: PowerShell startup script (process spawning, health checks)

## Example Log Flow for "Start" Click

```
# 1. User clicks Start in popup
2026-02-15T01:33:56.730Z | INFO | POPUP | TOGGLE_CLICK | 1736995236730-abc123 | INIT | Toggle changed: START requested

# 2. Popup sends request
2026-02-15T01:33:56.731Z | INFO | POPUP | START_REQUEST | 1736995236730-abc123 | SENDING | Sending POST to http://localhost:8080/api/screen-guard/start

# 3. Go proxy receives and forwards
2026-02-15T01:33:56.732Z | INFO | GO_PROXY | PROXY_REQUEST | 1736995236730-abc123 | INIT | Proxying POST /api/screen-guard/start to http://127.0.0.1:8081/start
2026-02-15T01:33:56.733Z | INFO | GO_PROXY | PROXY_REQUEST | 1736995236730-abc123 | SENDING | Sending request to Python API

# 4. Python API receives
2026-02-15T01:33:56.734Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | RECEIVED | POST /start request received from 127.0.0.1

# 5. Python API spawns monitor
2026-02-15T01:33:56.735Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | SPAWN | Spawning: python -m screen_guard_service
2026-02-15T01:33:56.850Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | SPAWNED | Monitor process spawned with PID: 12345

# 6. Python API responds
2026-02-15T01:33:57.850Z | INFO | PYTHON_API | START_REQUEST | 1736995236730-abc123 | SUCCESS | Monitor started successfully (PID: 12345)

# 7. Go proxy forwards response
2026-02-15T01:33:57.851Z | INFO | GO_PROXY | PROXY_REQUEST | 1736995236730-abc123 | RESPONSE | Python API responded: 200 OK

# 8. Popup receives response
2026-02-15T01:33:57.852Z | INFO | POPUP | START_REQUEST | 1736995236730-abc123 | SUCCESS | Start response received

# 9. Popup updates UI
2026-02-15T01:33:57.853Z | INFO | POPUP | UI_UPDATE | 1736995236730-abc123 | RUNNING | UI updated: Running (PID: 12345)
```

## Heartbeat Logs

Python API server logs a heartbeat every 30 seconds:
```
2026-02-15T01:34:00.000Z | INFO | PYTHON_API | HEARTBEAT | hb-1736995240 | RUNNING | Server alive: monitor_running=true, monitor_pid=12345, uptime=240s
```

## Error Logging

All errors include:
- Full error message
- Stack trace (in `extra_data`)
- Component where error occurred
- RequestId for correlation

Example error log:
```
2026-02-15T01:33:56.800Z | ERROR | PYTHON_API | START_REQUEST | 1736995236730-abc123 | FAILED | Monitor process exited immediately (code: 1) | {"stderr": "...", "traceback": "..."}
```

## Verifying Backend is Running

### Check Python API:
```powershell
# Test health endpoint
Invoke-WebRequest -Uri "http://127.0.0.1:8081/health"

# Check if process is running
Get-Process | Where-Object {$_.ProcessName -like "*python*"} | Select-Object Id, ProcessName
```

### Check Go Server:
```powershell
# Test health endpoint
Invoke-WebRequest -Uri "http://localhost:8080/health"

# Check if port is listening
Get-NetTCPConnection -LocalPort 8080
```

## Troubleshooting

### No logs appearing:
1. Check browser console is open (F12)
2. Verify extension is reloaded (`chrome://extensions/` → reload)
3. Check log files exist: `Test-Path logs\backend.log`

### RequestId not propagating:
1. Check Go proxy logs for requestId extraction
2. Verify `X-Request-Id` header is being sent from extension
3. Check Python API logs for requestId in headers

### Python logs not appearing:
1. Check `logs\backend.log` and `logs\backend_stderr.log`
2. Verify Python process is running: `Get-Process | Where-Object {$_.ProcessName -like "*python*"}`
3. Check if port 8081 is listening: `Get-NetTCPConnection -LocalPort 8081`

