# Video Monitoring - Quick Start & Testing Guide

## Prerequisites

### 1. Install Python Dependencies
```powershell
cd screen_guard_service
pip install -r requirements.txt
```

**Note**: This installs OpenCV, EasyOCR, transformers, and other ML libraries. May take 5-10 minutes.

**Quick check**:
```powershell
python3 --version  # Should show Python 3.8+
python3 -c "import cv2, mss, easyocr; print('Dependencies OK')"
```

### 2. Verify Go Server Can Find Python
```powershell
# Test Python is accessible
python3 --version

# Or if using 'python' instead:
python --version
```

If Python is not in PATH, set environment variable:
```powershell
$env:VIDEO_MONITOR_PYTHON_PATH = "C:\Python39\python.exe"  # Adjust path
```

## Step-by-Step: Start to End

### Step 1: Start the Go Proxy Server

**Terminal 1** (keep this open):
```powershell
cd C:\Users\anand\Development\hacknc
go run . serve
```

**Expected output**:
```
Starting pasteguard server on :8080
```

**Verify it's running**:
```powershell
# In another terminal
Invoke-RestMethod -Uri http://localhost:8080/health
# Should return: {"ok":true}
```

### Step 2: Load the Browser Extension

1. Open Chrome/Edge
2. Go to `chrome://extensions/` or `edge://extensions/`
3. Enable "Developer mode" (toggle top-right)
4. Click "Load unpacked"
5. Select the `extension` folder: `C:\Users\anand\Development\hacknc\extension`
6. Extension should appear with "Privacy Guardrail" name

### Step 3: Open Extension Popup

1. Click the extension icon in the browser toolbar
2. You should see:
   - Session Stats section
   - Paste Guard Settings
   - **Video Monitoring** section (new!)
   - Status should show "Stopped"

### Step 4: Start Video Monitoring

1. In the extension popup, find "Video Monitoring" section
2. Click the **"Start"** button
3. **What to expect**:
   - Button changes to "Starting..."
   - Status changes to "Starting" (yellow)
   - Within 1-2 seconds, status changes to "Running" (green)
   - "Stop" button appears
   - "Recent Alerts" and "Live Logs" panels become visible

**If it fails**:
- Check Terminal 1 (Go server) for error messages
- Common issues:
  - Python not found: Set `VIDEO_MONITOR_PYTHON_PATH` env var
  - Script not found: Check `screen_guard_service/run_monitor.py` exists
  - Dependencies missing: Run `pip install -r requirements.txt`

### Step 5: Test Real-Time Detection

**Test 1: Password Detection**
1. Open a text editor (Notepad, VS Code, etc.)
2. Type or paste: `password = "mySecretPassword123"`
3. Make sure it's visible on your screen
4. Wait 2-3 seconds (scan interval)
5. **Check extension popup**:
   - Alert should appear in "Recent Alerts" panel
   - Shows: `password_assignment` rule, `HIGH` severity, confidence %, redacted text
   - Log shows: `[timestamp] [detector] Detection: password_assignment (high)`

**Test 2: API Key Detection**
1. In text editor, type: `api_key = "sk-1234567890abcdef"`
2. Wait 2-3 seconds
3. Another alert should appear

**Test 3: Multiple Detections**
1. Show multiple sensitive items on screen
2. Verify alerts stack in chronological order
3. Verify logs continue updating

### Step 6: Monitor Live Logs

1. Keep extension popup open
2. Watch "Live Logs" panel
3. **You should see**:
   - `[timestamp] [monitor] Video monitoring started successfully`
   - `[timestamp] [scanner] Scan completed`
   - `[timestamp] [ocr] OCR confidence: 0.95`
   - `[timestamp] [detector] Detection: ...` (when alerts occur)

### Step 7: Test Stop

1. Click **"Stop"** button in extension popup
2. **What to expect**:
   - Status changes to "Stopping" then "Stopped"
   - "Start" button reappears
   - Logs show: "Video monitoring stopping..." and "Video monitoring stopped"
   - Alerts and logs panels remain visible (showing history)

### Step 8: Test Error Handling

**Test Disconnection Recovery**:
1. Start video monitoring
2. Kill the Python process manually (or close Terminal 1)
3. **What to expect**:
   - Status changes to "Degraded" (red)
   - Error message appears: "Video monitoring process disconnected"
   - Logs show error event
   - Go proxy attempts automatic reconnection

**Test Invalid State**:
1. Try starting monitoring when already running
2. Should show appropriate error or ignore

## Testing Tips

### Tip 1: Use Browser DevTools

1. Open extension popup
2. Right-click in popup → "Inspect"
3. Go to **Console** tab
4. **Watch for**:
   - SSE connection messages
   - Event parsing errors
   - Network errors

**Common console messages**:
```
✅ Good: "Connected to SSE stream"
❌ Bad: "SSE connection error" or "Failed to connect"
```

### Tip 2: Monitor Go Server Logs

**Terminal 1** (Go server) shows:
- Process start/stop messages
- Event reception from Python
- Client connections/disconnections
- Errors

**Look for**:
```
✅ Good: "Video monitoring started"
❌ Bad: "Failed to start video monitoring: ..."
```

### Tip 3: Test Different Detection Types

Create test files with known sensitive content:

**test_secrets.txt**:
```
password = "secret123"
api_key = "sk-1234567890"
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----
```

Open in text editor and share screen to trigger detections.

### Tip 4: Verify SSE Connection

**In browser DevTools Console** (while popup is open):
```javascript
// Check if EventSource is connected
// (Extension code should show connection status)
```

**Or check Network tab**:
- Look for `GET /api/video-monitor/stream`
- Status should be "200" or "pending" (SSE keeps connection open)
- Type should be "eventsource"

### Tip 5: Test Status Persistence

1. Start video monitoring
2. Close extension popup
3. Reopen extension popup
4. **Expected**: Status should still show "Running" (fetched from Go proxy)
5. SSE should auto-reconnect

### Tip 6: Performance Testing

**Measure Latency**:
1. Start monitoring
2. Show sensitive content on screen
3. Time from content appearing to alert in extension
4. **Target**: < 500ms (includes scan interval + network)

**Monitor Resource Usage**:
- Python process CPU/memory (Task Manager)
- Go server memory
- Browser extension memory (DevTools → Performance)

## Troubleshooting

### Issue: Status Stuck on "Starting"

**Check**:
1. Go server logs for Python process errors
2. Python is installed: `python3 --version`
3. Dependencies installed: `pip list | findstr "opencv mss easyocr"`
4. Script path correct: Check `screen_guard_service/run_monitor.py` exists

**Fix**:
```powershell
# Set explicit Python path
$env:VIDEO_MONITOR_PYTHON_PATH = "python3"

# Or full path:
$env:VIDEO_MONITOR_PYTHON_PATH = "C:\Python39\python.exe"
```

### Issue: No Alerts Appearing

**Check**:
1. Video monitoring status is "Running" (not "Degraded")
2. Sensitive content is actually visible on screen (not minimized)
3. SSE connection is active (check browser console)
4. Go server is receiving events (check Terminal 1 logs)

**Debug**:
```powershell
# Manually test Python server
cd screen_guard_service
python3 run_monitor.py --mode manual --quiet
# Should start scanning (may take 10-20 seconds to initialize)
```

### Issue: SSE Connection Errors

**Check**:
1. Go server is running on port 8080
2. No firewall blocking localhost:8080
3. Browser allows localhost connections
4. Extension has permission for `http://localhost:8080/*`

**Fix**:
- Restart Go server
- Reload extension
- Check browser console for CORS errors

### Issue: Python Process Dies Immediately

**Check Go server logs** for:
- "Failed to start video monitoring"
- Python path errors
- Import errors from Python

**Fix**:
```powershell
# Test Python script directly
cd screen_guard_service
python3 run_monitor.py --mode manual --quiet
# If this fails, fix Python dependencies first
```

### Issue: Alerts Appear But UI Doesn't Update

**Check**:
1. Browser console for JavaScript errors
2. SSE events are being received (Network tab)
3. Extension popup is still open (SSE may disconnect if popup closes)

**Fix**:
- Close and reopen extension popup
- Check browser console for errors
- Verify `popup.js` has all video monitoring functions

## Quick Test Checklist

- [ ] Go server starts without errors
- [ ] Extension loads successfully
- [ ] Video monitoring section appears in popup
- [ ] Click "Start" → Status changes to "Running" within 2 seconds
- [ ] "Recent Alerts" and "Live Logs" panels appear
- [ ] Show password on screen → Alert appears within 3 seconds
- [ ] Logs update in real-time
- [ ] Click "Stop" → Status changes to "Stopped"
- [ ] Multiple detections stack correctly
- [ ] SSE reconnects after popup close/reopen

## Performance Benchmarks

**Target Metrics**:
- Start time: < 2 seconds
- Detection latency: < 500ms (from screen content to alert)
- Log update latency: < 100ms
- Memory usage: < 500MB (Python process)
- CPU usage: < 20% (when scanning)

**Measure**:
```powershell
# Monitor Python process
Get-Process python | Select-Object CPU, WorkingSet
```

## Next Steps After Testing

1. **Test with real screen sharing** (Zoom, Teams, etc.)
2. **Test with different sensitive content types** (PEM keys, JWT tokens, etc.)
3. **Test error recovery** (kill Python process, restart Go server)
4. **Test with multiple browser tabs** (each has separate extension popup)
5. **Test performance** under load (long-running monitoring session)

## Demo Script (60 seconds)

See `DEMO_FLOW.md` for the complete 60-second demo script that covers:
- Starting monitoring
- Showing live logs
- Triggering detections
- Displaying alerts
- Stopping monitoring

