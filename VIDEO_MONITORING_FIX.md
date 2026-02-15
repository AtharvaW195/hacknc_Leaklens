# Video Monitoring - No Alerts Fix

## Changes Made

### 1. Added Comprehensive Logging
- **http_bridge.py**: Now logs when sending events, success/failure
- **realtime_monitor.py**: Logs when detections are found, filtered, and callbacks called
- **run_monitor.py**: Logs when bridge is imported and initialized
- **server.go**: Logs when events are received from Python

### 2. Removed `--quiet` Flag
- Python process now runs without `--quiet` so debug logs are visible
- You'll see scan activity in the Python process output

### 3. Enhanced Error Handling
- Bridge failures are now logged (not silent)
- Import errors are logged
- Callback errors show full traceback

## How to Debug

### Step 1: Restart Go Server
```powershell
# Stop current server (Ctrl+C)
go run . serve
```

### Step 2: Start Video Monitoring from Extension
Click "Start" button

### Step 3: Check Go Server Logs
You should see:
```
[VIDEO_MONITOR] Start request received
[VIDEO_MONITOR] Process started, PID: XXXX
```

### Step 4: Check Python Process Output
The Python process output should be visible. Look for:
```
[RUN_MONITOR] HTTP bridge is available, initializing...
[RUN_MONITOR] Bridge URL: http://localhost:8080/api/video-monitor/events
[REALTIME_MONITOR] Starting monitoring loop...
[REALTIME_MONITOR] Scanning frame #1...
[REALTIME_MONITOR] Scan complete: X raw detections found
```

**If you don't see Python output**:
- The process might be outputting to a log file
- Check: `screen_guard_service/monitor_output/runs/*/runtime.log`

### Step 5: Show Sensitive Content
1. Open Notepad
2. Type: `password = "mySecret123"`
3. Wait 2-3 seconds

### Step 6: Check Logs

**In Python output (or log file), you should see**:
```
[REALTIME_MONITOR] handle_detections called with X raw detections
[REALTIME_MONITOR] After filtering: Y confirmed detections
[REALTIME_MONITOR] Calling alert_callback with Y detections
[HTTP_BRIDGE] Alert callback invoked
[HTTP_BRIDGE] Found Y detections
[HTTP_BRIDGE] Sending detection 1/Y: password_assignment
[HTTP_BRIDGE] Successfully sent detection event
```

**In Go server logs, you should see**:
```
[VIDEO_MONITOR] Received event: type=detection from 127.0.0.1:XXXX
[VIDEO_MONITOR] Detection: password_assignment (severity: high)
```

**In Extension console (popup DevTools), you should see**:
```
[VIDEO_MONITOR] Event received: {type: "detection", ...}
```

## Common Issues & Fixes

### Issue: "HTTP bridge not available"
**Symptom**: `[RUN_MONITOR] WARNING: HTTP bridge not available`

**Fix**: Check if `http_bridge.py` exists and `requests` is installed:
```powershell
cd screen_guard_service
python3 -c "import requests; print('OK')"
python3 -c "from http_bridge import get_bridge; print('OK')"
```

### Issue: "No detections found"
**Symptom**: `[REALTIME_MONITOR] Scan complete: 0 raw detections found`

**Possible causes**:
1. Content not visible on screen (minimized window)
2. OCR not working (check if EasyOCR/Tesseract is installed)
3. Detection rules not matching

**Fix**: Test with very obvious content:
```
password = "test123"
api_key = "sk-test1234567890"
```

### Issue: "Detections filtered out"
**Symptom**: `[REALTIME_MONITOR] After filtering: 0 confirmed detections`

**Cause**: Confirmation logic requires detections to appear in multiple frames

**Fix**: Lower confirmation requirements or show content for longer:
- Content must be visible for at least `confirmation_frames * scan_interval` seconds
- Default: 2 frames * 1.5s = 3 seconds minimum

### Issue: "Bridge not sending events"
**Symptom**: `[HTTP_BRIDGE] Failed to send detection event: ...`

**Possible causes**:
1. Go server not running
2. Wrong bridge URL
3. Network/firewall issue

**Fix**: Check bridge URL matches Go server:
```powershell
# Should be: http://localhost:8080/api/video-monitor/events
# Check Go server is running:
Invoke-RestMethod -Uri http://localhost:8080/health
```

### Issue: "Go server not receiving events"
**Symptom**: No `[VIDEO_MONITOR] Received event` logs

**Fix**: Check Go server logs for errors when Python tries to POST

### Issue: "Extension not receiving SSE events"
**Symptom**: No alerts in extension UI

**Fix**:
1. Check extension console for SSE connection errors
2. Verify SSE stream endpoint: `GET /api/video-monitor/stream`
3. Check Network tab for SSE connection

## Expected Flow (When Working)

1. **Python starts scanning**:
   ```
   [REALTIME_MONITOR] Scanning frame #1...
   [REALTIME_MONITOR] Scan complete: 1 raw detections found
   ```

2. **Detections confirmed**:
   ```
   [REALTIME_MONITOR] After filtering: 1 confirmed detections
   [REALTIME_MONITOR] Calling alert_callback
   ```

3. **Bridge sends event**:
   ```
   [HTTP_BRIDGE] Sending detection event
   [HTTP_BRIDGE] Successfully sent detection event
   ```

4. **Go server receives**:
   ```
   [VIDEO_MONITOR] Received event: type=detection
   [VIDEO_MONITOR] Detection: password_assignment
   ```

5. **Extension receives via SSE**:
   - Alert appears in "Recent Alerts" panel
   - Log shows in "Live Logs" panel

## Quick Test

1. Restart Go server
2. Start monitoring from extension
3. Open Notepad
4. Type: `password = "test123"`
5. Wait 5 seconds (to pass confirmation)
6. Check all three logs:
   - Python output (or log file)
   - Go server terminal
   - Extension console

If you see logs in Python and Go but not in extension → SSE connection issue
If you see logs in Python but not Go → Bridge sending issue
If you see no logs in Python → Scanning/detection issue

