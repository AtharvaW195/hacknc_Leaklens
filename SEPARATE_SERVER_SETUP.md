# Running Go and Python Servers Separately

This guide shows how to run the Go server and Python server independently for testing and development.

## Prerequisites

- Python 3.x with dependencies installed (`pip install -r screen_guard_service/requirements.txt`)
- Go 1.x installed
- Both servers can run on the same machine

## Quick Start

### Terminal 1: Start Python Server
```powershell
cd screen_guard_service
python api_server.py
```

### Terminal 2: Start Go Server
```powershell
$env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:8081"
go run . serve --addr :8080
```

## Detailed Steps

### Step 1: Start Python API Server

**Option A: Direct Python execution**
```powershell
# Navigate to service directory
cd screen_guard_service

# Activate virtual environment (if using one)
# .\.venv\Scripts\Activate.ps1  # or venv\Scripts\Activate.ps1

# Set environment variables (optional, defaults shown)
$env:SCREEN_GUARD_API_HOST = "127.0.0.1"
$env:SCREEN_GUARD_API_PORT = "8081"

# Start the server
python api_server.py
```

**Option B: Run as module**
```powershell
# From repo root
cd screen_guard_service
python -m api_server
```

**Expected output:**
```
2026-02-15T01:33:56.730Z | INFO | PYTHON_API | SERVER_START | startup-1736995236730 | INIT | Starting Screen Guard API server on 127.0.0.1:8081
2026-02-15T01:33:56.739Z | INFO | PYTHON_API | SERVER_START | startup-1736995236730 | CONFIG | Python: C:\Users\...\python.exe, CWD: C:\Users\...\screen_guard_service
INFO:     Started server process [12345]
INFO:     Waiting for application startup.
INFO:     Application startup complete.
INFO:     Uvicorn running on http://127.0.0.1:8081 (Press CTRL+C to quit)
```

**Verify Python server is running:**
```powershell
# Test health endpoint
Invoke-WebRequest -Uri "http://127.0.0.1:8081/health"

# Should return: {"status":"healthy","api_server":"running"}
```

### Step 2: Start Go Server

**In a new terminal (keep Python server running):**

```powershell
# Navigate to repo root
cd C:\Users\anand\Development\hacknc

# Set environment variable to point Go server to Python API
$env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:8081"

# Optional: Set Go server port (default is :8080)
$env:GO_PORT = "8080"  # Optional, default is 8080

# Start Go server
go run . serve --addr :8080
```

**Expected output:**
```
Starting pasteguard server on :8080
[Screen Guard Proxy] | INFO | GO_PROXY | ... | ... | ... | Proxy configured for http://127.0.0.1:8081
```

**Verify Go server is running:**
```powershell
# Test Go server health
Invoke-WebRequest -Uri "http://localhost:8080/health"

# Test Go proxy to Python (should return Python health)
Invoke-WebRequest -Uri "http://localhost:8080/api/screen-guard/health"
```

### Step 3: Verify Connection

**Test the full flow:**

```powershell
# 1. Check Python API directly
Invoke-WebRequest -Uri "http://127.0.0.1:8081/status"

# 2. Check through Go proxy
Invoke-WebRequest -Uri "http://localhost:8080/api/screen-guard/status"

# 3. Start monitor through Go proxy
$body = @{} | ConvertTo-Json
Invoke-RestMethod -Uri "http://localhost:8080/api/screen-guard/start" -Method POST -ContentType "application/json" -Body $body
```

## Environment Variables

### Python Server

| Variable | Default | Description |
|----------|---------|-------------|
| `SCREEN_GUARD_API_HOST` | `127.0.0.1` | Host to bind to (use 127.0.0.1 for localhost only) |
| `SCREEN_GUARD_API_PORT` | `8081` | Port to listen on |

**Example:**
```powershell
$env:SCREEN_GUARD_API_HOST = "127.0.0.1"
$env:SCREEN_GUARD_API_PORT = "8081"
python api_server.py
```

### Go Server

| Variable | Default | Description |
|----------|---------|-------------|
| `SCREEN_GUARD_BASE_URL` | `http://127.0.0.1:8081` | Python API base URL (MUST match Python server) |
| `GO_PORT` | `8080` | Go server port (set via `--addr` flag) |

**Example:**
```powershell
$env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:8081"
go run . serve --addr :8080
```

## Complete Example Session

### Terminal 1 (Python Server):
```powershell
PS C:\Users\anand\Development\hacknc> cd screen_guard_service
PS C:\Users\anand\Development\hacknc\screen_guard_service> $env:SCREEN_GUARD_API_PORT = "8081"
PS C:\Users\anand\Development\hacknc\screen_guard_service> python api_server.py
2026-02-15T01:33:56.730Z | INFO | PYTHON_API | SERVER_START | startup-... | INIT | Starting Screen Guard API server on 127.0.0.1:8081
INFO:     Uvicorn running on http://127.0.0.1:8081
```

### Terminal 2 (Go Server):
```powershell
PS C:\Users\anand\Development\hacknc> $env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:8081"
PS C:\Users\anand\Development\hacknc> go run . serve --addr :8080
Starting pasteguard server on :8080
```

### Terminal 3 (Testing):
```powershell
# Test Python directly
PS> Invoke-WebRequest -Uri "http://127.0.0.1:8081/health"
StatusCode: 200

# Test through Go proxy
PS> Invoke-WebRequest -Uri "http://localhost:8080/api/screen-guard/health"
StatusCode: 200

# Start monitor
PS> Invoke-RestMethod -Uri "http://localhost:8080/api/screen-guard/start" -Method POST -ContentType "application/json"
```

## Troubleshooting

### Issue: Go server can't connect to Python

**Symptoms:**
- Error: `Failed to connect to screen guard service`
- Error: `Timeout connecting to screen guard service`

**Solutions:**
1. **Verify Python server is running:**
   ```powershell
   Get-NetTCPConnection -LocalPort 8081
   # Should show LISTENING state
   ```

2. **Test Python API directly:**
   ```powershell
   Invoke-WebRequest -Uri "http://127.0.0.1:8081/health"
   # Should return 200 OK
   ```

3. **Check SCREEN_GUARD_BASE_URL matches:**
   ```powershell
   # In Go server terminal, verify:
   echo $env:SCREEN_GUARD_BASE_URL
   # Should be: http://127.0.0.1:8081
   ```

4. **Check firewall/antivirus:**
   - Ensure localhost connections aren't blocked
   - Try disabling firewall temporarily for testing

### Issue: Port already in use

**Python port (8081) in use:**
```powershell
# Find process using port
Get-NetTCPConnection -LocalPort 8081 | Select-Object OwningProcess

# Kill it (replace <PID> with actual process ID)
Stop-Process -Id <PID> -Force

# Or use different port
$env:SCREEN_GUARD_API_PORT = "8082"
# Then update Go server:
$env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:8082"
```

**Go port (8080) in use:**
```powershell
# Find and kill process
Get-NetTCPConnection -LocalPort 8080 | Select-Object OwningProcess
Stop-Process -Id <PID> -Force

# Or use different port
go run . serve --addr :8081
```

### Issue: Python server starts but crashes

**Check Python logs:**
```powershell
# If running with start.ps1, check:
Get-Content logs\backend_stderr.log -Tail 50

# If running directly, errors appear in terminal
```

**Common causes:**
- Missing dependencies: `pip install -r screen_guard_service/requirements.txt`
- Python version incompatible
- Port conflict

### Issue: Extension can't connect

**Symptoms:**
- Extension shows "Unable to connect to server"
- Browser console shows CORS errors

**Solutions:**
1. **Verify Go server is running:**
   ```powershell
   Invoke-WebRequest -Uri "http://localhost:8080/health"
   ```

2. **Check extension is pointing to correct URL:**
   - Open `extension/popup.js`
   - Verify `SCREEN_GUARD_API = "http://localhost:8080/api/screen-guard"`

3. **Reload extension:**
   - Go to `chrome://extensions/`
   - Click reload on "Privacy Guardrail"

## Log Locations

When running separately:

- **Python logs**: Terminal output (stdout/stderr)
- **Go logs**: Terminal output
- **No log files created** (unless you redirect output)

To capture logs to files:

**Python:**
```powershell
python api_server.py > ..\logs\python_server.log 2>&1
```

**Go:**
```powershell
go run . serve --addr :8080 2>&1 | Tee-Object -FilePath "logs\go_server.log"
```

## Testing the Connection

### Quick Test Script

Save as `test-connection.ps1`:

```powershell
Write-Host "Testing Python API..." -ForegroundColor Yellow
try {
    $py = Invoke-WebRequest -Uri "http://127.0.0.1:8081/health" -TimeoutSec 2
    Write-Host "✓ Python API: OK ($($py.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "✗ Python API: FAILED - $_" -ForegroundColor Red
}

Write-Host "`nTesting Go Server..." -ForegroundColor Yellow
try {
    $go = Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 2
    Write-Host "✓ Go Server: OK ($($go.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "✗ Go Server: FAILED - $_" -ForegroundColor Red
}

Write-Host "`nTesting Go → Python Proxy..." -ForegroundColor Yellow
try {
    $proxy = Invoke-WebRequest -Uri "http://localhost:8080/api/screen-guard/health" -TimeoutSec 2
    Write-Host "✓ Proxy: OK ($($proxy.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "✗ Proxy: FAILED - $_" -ForegroundColor Red
}

Write-Host "`nTesting Status Endpoint..." -ForegroundColor Yellow
try {
    $status = Invoke-RestMethod -Uri "http://localhost:8080/api/screen-guard/status" -TimeoutSec 2
    Write-Host "✓ Status: OK - Running=$($status.running), PID=$($status.pid)" -ForegroundColor Green
} catch {
    Write-Host "✗ Status: FAILED - $_" -ForegroundColor Red
}
```

Run it:
```powershell
.\test-connection.ps1
```

## Order of Operations

**Correct order:**
1. Start Python server first (Terminal 1)
2. Wait for Python to be ready (check health endpoint)
3. Start Go server (Terminal 2)
4. Verify connection (test proxy endpoint)

**Why this order matters:**
- Go server tries to connect to Python on startup
- If Python isn't ready, Go will show connection errors
- Python must be listening before Go can proxy requests

## Stopping Servers

**Python server:**
- Press `Ctrl+C` in Python terminal
- Or kill process: `Get-Process python | Stop-Process -Force`

**Go server:**
- Press `Ctrl+C` in Go terminal
- Or kill process: `Get-Process go | Stop-Process -Force`

## Advanced: Using Different Ports

If you need to use different ports:

**Python on 9091:**
```powershell
$env:SCREEN_GUARD_API_PORT = "9091"
cd screen_guard_service
python api_server.py
```

**Go on 9090, pointing to Python on 9091:**
```powershell
$env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:9091"
go run . serve --addr :9090
```

**Update extension:**
- Change `SCREEN_GUARD_API` in `extension/popup.js` to `http://localhost:9090/api/screen-guard`

## Summary

**To run separately:**

1. **Terminal 1 - Python:**
   ```powershell
   cd screen_guard_service
   python api_server.py
   ```

2. **Terminal 2 - Go:**
   ```powershell
   $env:SCREEN_GUARD_BASE_URL = "http://127.0.0.1:8081"
   go run . serve --addr :8080
   ```

3. **Verify:**
   ```powershell
   Invoke-WebRequest -Uri "http://localhost:8080/api/screen-guard/health"
   ```

That's it! The servers are now running independently and connected.

