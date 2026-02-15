# Video Monitoring Debug Guide

**Last Updated**: After adding stdout/stderr capture to see Python logs in Go server output

## Issue 1: No Logs When Clicking "Start"

### Check Extension Console
1. Open extension popup
2. Right-click in popup → "Inspect"
3. Go to **Console** tab
4. Click "Start" button
5. **Look for**:
   - `[VIDEO_MONITOR] Start button clicked`
   - `[VIDEO_MONITOR] Sending POST to /api/video-monitor/start`
   - `[VIDEO_MONITOR] Response status: 200` (or error code)

**If you see errors**:
- `Failed to fetch` → Go server not running
- `404` → Route not registered
- `500` → Server error (check Go server logs)

### Check Go Server Logs
When you click "Start", you should see:
```
[VIDEO_MONITOR] Start request received from [IP]
[VIDEO_MONITOR] Start() called, current status: stopped
[VIDEO_MONITOR] Using Python: python3
[VIDEO_MONITOR] Using script: C:\Users\anand\Development\hacknc\screen_guard_service\run_monitor.py
[VIDEO_MONITOR] Script found, starting Python process...
[VIDEO_MONITOR] Command: python3 C:\Users\anand\Development\hacknc\screen_guard_service\run_monitor.py
[VIDEO_MONITOR] Process started, PID: [number]
[VIDEO_MONITOR] Process is running, updating status to 'running'
[VIDEO_MONITOR] Started successfully, status: running
```

**Then you should see Python output** (captured from stdout/stderr):
```
[PYTHON-STDOUT] [RUN_MONITOR] HTTP bridge is available, initializing...
[PYTHON-STDOUT] [RUN_MONITOR] Bridge URL: http://localhost:8080/api/video-monitor/events
[PYTHON-STDOUT] [REALTIME_MONITOR] Starting monitoring loop...
[PYTHON-STDOUT] [REALTIME_MONITOR] Scanning frame #1...
[PYTHON-STDOUT] [REALTIME_MONITOR] Scan complete: X raw detections found
```

**If you don't see Python output**:
- Python process might have crashed immediately
- Check for `[PYTHON-STDERR]` messages (errors)
- Process might be waiting for dependencies to load (first scan can take 10-20 seconds)

**If you don't see these logs**:
- Extension might not be making the request
- Check browser console for errors
- Verify Go server is actually running

## Issue 2: Is It Really Started?

### Check Process List
```powershell
# Check if Python process is running
Get-Process python* | Select-Object Id, ProcessName, StartTime

# Or check by command line
Get-WmiObject Win32_Process | Where-Object {$_.CommandLine -like "*run_monitor.py*"} | Select-Object ProcessId, CommandLine
```

### Check Extension Status
1. Open extension popup
2. Look at "Video Monitoring" section
3. Status should show "Running" (green) if started
4. "Stop" button should be visible

### Check Go Server Status Endpoint
```powershell
Invoke-RestMethod -Uri http://localhost:8080/api/video-monitor/status
```

Should return:
```json
{
  "status": "running",
  "message": "Video monitoring is running",
  "started_at": "2024-02-15T..."
}
```

## Issue 3: Process Stops By Itself

### Common Causes

#### 1. Python Dependencies Missing
**Symptom**: Process starts then exits immediately (exit code 1)

**Check**:
```powershell
cd screen_guard_service
python3 run_monitor.py --mode manual --quiet
```

**If you see import errors**:
```powershell
pip install -r requirements.txt
```

#### 2. Script Path Wrong
**Symptom**: "Script not found" error in Go logs

**Fix**: Set environment variable:
```powershell
$env:VIDEO_MONITOR_SCRIPT_PATH = "C:\Users\anand\Development\hacknc\screen_guard_service\run_monitor.py"
```

#### 3. Python Path Wrong
**Symptom**: "executable file not found" error

**Fix**: Set environment variable:
```powershell
# Check Python path
where.exe python3
# Or
where.exe python

# Set it
$env:VIDEO_MONITOR_PYTHON_PATH = "python3"
# Or full path:
$env:VIDEO_MONITOR_PYTHON_PATH = "C:\Python39\python.exe"
```

#### 4. Process Exits Due to Error
**Check Go server logs** for:
- `[VIDEO_MONITOR] Process exited with code: [number]`
- Exit code 1 = Python error (check dependencies)
- Exit code 2 = Script error

**Debug**:
```powershell
# Run Python script manually to see errors
cd screen_guard_service
python3 run_monitor.py --mode manual
# Don't use --quiet so you can see errors
```

#### 5. No Events Received (30 second timeout)
**Symptom**: Status changes to "Degraded" after 30 seconds

**Cause**: Python server started but not sending events to Go proxy

**Check**:
- Is `http_bridge.py` being imported?
- Is `VIDEO_MONITOR_BRIDGE_URL` environment variable set?
- Check Python process is actually running (not crashed)

**Debug**:
```powershell
# Check if bridge URL is set in process
Get-WmiObject Win32_Process | Where-Object {$_.CommandLine -like "*run_monitor.py*"} | Select-Object CommandLine
# Should show: VIDEO_MONITOR_BRIDGE_URL=http://localhost:8080/api/video-monitor/events
```

## Step-by-Step Debugging

### Step 1: Verify Extension is Calling API
1. Open extension popup
2. Open DevTools (right-click → Inspect)
3. Go to **Network** tab
4. Click "Start"
5. **Look for**: `POST /api/video-monitor/start`
   - Status should be 200
   - Response should show status JSON

### Step 2: Verify Go Server Receives Request
Check Go server terminal for:
```
[VIDEO_MONITOR] Start request received from [IP]
```

If you don't see this → Extension not making request (check browser console)

### Step 3: Verify Python Process Starts
Check Go server logs for:
```
[VIDEO_MONITOR] Process started, PID: [number]
```

If you see "Failed to start" → Check Python path and script path

### Step 4: Verify Process Stays Running
```powershell
# Check process is still running after 10 seconds
Start-Sleep -Seconds 10
Get-Process python* | Select-Object Id, ProcessName
```

If process disappeared → Check Python script for errors

### Step 5: Verify Events Are Being Sent
Check Go server logs for:
```
[VIDEO_MONITOR] No events received for X seconds
```

If you see this → Python server not sending events (check http_bridge.py)

## Quick Fixes

### Fix 1: Python Not Found
**The code now auto-detects Python**, but if it still fails:

```powershell
# Find Python
where.exe python
# Should show: C:\Users\...\Python\python.exe

# If python3 doesn't work, set explicit path:
$env:VIDEO_MONITOR_PYTHON_PATH = "python"
go run . serve
```

**Or use full path**:
```powershell
$env:VIDEO_MONITOR_PYTHON_PATH = "C:\Users\anand\AppData\Local\Programs\Python\Python313\python.exe"
go run . serve
```

### Fix 2: Script Not Found
```powershell
# Verify script exists
Test-Path "screen_guard_service\run_monitor.py"

# Set explicit path
$env:VIDEO_MONITOR_SCRIPT_PATH = "C:\Users\anand\Development\hacknc\screen_guard_service\run_monitor.py"
go run . serve
```

### Fix 3: Dependencies Missing
```powershell
cd screen_guard_service
pip install -r requirements.txt

# Test manually
python3 run_monitor.py --mode manual
# Press Ctrl+C to stop
```

### Fix 4: Process Exits Immediately
```powershell
# Run script manually to see error
cd screen_guard_service
python3 run_monitor.py --mode manual
# Look for import errors or other Python errors
```

## Expected Behavior

### When Starting:
1. Extension: Button shows "Starting..."
2. Go server: Logs show "Start request received"
3. Go server: Logs show "Process started, PID: X"
4. Extension: Status changes to "Running" (green)
5. Extension: "Stop" button appears
6. Extension: Alerts and Logs panels appear

### When Running:
1. Python process should be visible in Task Manager
2. Go server: Logs show "Process monitor started"
3. Extension: Logs panel shows activity
4. Extension: Alerts appear when sensitive content detected

### When Stopping:
1. Extension: Button shows "Stopping..."
2. Go server: Logs show "Stop request received"
3. Python process terminates
4. Extension: Status changes to "Stopped"
5. Extension: "Start" button reappears

## Still Not Working?

1. **Check all logs**:
   - Browser console (extension popup)
   - Go server terminal
   - Python script output (if running manually)

2. **Verify paths**:
   ```powershell
   # Python
   python3 --version
   
   # Script
   Test-Path "screen_guard_service\run_monitor.py"
   
   # Dependencies
   python3 -c "import cv2, mss, easyocr; print('OK')"
   ```

3. **Test manually**:
   ```powershell
   cd screen_guard_service
   python3 run_monitor.py --mode manual
   # Should start scanning (may take 10-20 seconds to initialize)
   ```

4. **Check environment variables**:
   ```powershell
   $env:VIDEO_MONITOR_PYTHON_PATH
   $env:VIDEO_MONITOR_SCRIPT_PATH
   ```

