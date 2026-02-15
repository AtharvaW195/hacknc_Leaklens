# Video Monitoring Integration Plan

## Chosen Streaming Mechanism: **Server-Sent Events (SSE)**

**Why SSE over WebSocket:**
- Built into Go's `net/http` (no new dependencies)
- Simpler implementation for one-way server->client push
- Automatic reconnection support in browsers
- Fits hackathon timeline better
- Extension only needs to receive events (no bidirectional needed)

## Event Schema

Unified JSON event structure:

```json
{
  "type": "status" | "detection" | "log" | "error",
  "timestamp": "2024-02-15T03:30:00Z",
  "data": {
    // For status events:
    "status": "starting" | "running" | "stopping" | "stopped" | "degraded",
    "message": "Monitoring started",
    
    // For detection events:
    "rule_name": "password_assignment",
    "severity": "critical" | "high" | "medium" | "low",
    "confidence": 0.95,
    "matched_text": "secr...t123",  // redacted
    "detection_method": "regex",
    "frame_time": 123.45,  // optional video timestamp
    
    // For log events:
    "level": "info" | "warning" | "error",
    "component": "monitor" | "scanner" | "ocr",
    "message": "Scan completed",
    
    // For error events:
    "code": "VIDEO_SERVER_DOWN",
    "message": "Video processing server disconnected",
    "recoverable": true
  }
}
```

## Files to Change

### Go Proxy Layer
1. `server/server.go` - Add video monitoring routes and SSE handler
2. `server/video_monitor.go` - New file: video server process management
3. `go.mod` - No new dependencies needed (SSE is stdlib)

### Video Processing Server Integration
4. `screen_guard_service/http_bridge.py` - New file: HTTP API wrapper that emits events to Go proxy
5. `screen_guard_service/realtime_monitor.py` - Modify alert callback to also send to HTTP bridge

### Extension Changes
6. `extension/popup.html` - Add video monitoring UI section
7. `extension/popup.js` - Add video monitoring controls and SSE client
8. `extension/background.js` - Add message handlers for video monitoring
9. `extension/styles.css` - Add styles for video monitoring UI

### Documentation
10. `DEMO_FLOW.md` - Add video monitoring demo steps
11. `README.md` - Update with new endpoints and env vars

## Minimal Test Plan

1. **Go Route Tests** (`server/video_monitor_test.go`):
   - Test start/stop endpoints
   - Test status endpoint
   - Test SSE stream connection
   - Test error handling when video server fails

2. **Integration Test** (manual or e2e):
   - Start video monitoring via extension
   - Verify status changes to "running"
   - Trigger a detection (show password on screen)
   - Verify alert appears in extension UI
   - Verify logs update in real-time
   - Stop monitoring and verify status changes

## Implementation Approach

1. **Go proxy manages video server as subprocess** (simpler than HTTP API for now)
2. **Video server emits events via HTTP POST to Go proxy** (Go proxy exposes internal endpoint)
3. **Go proxy forwards events to all connected SSE clients**
4. **Extension connects to SSE stream on startup if monitoring is active**

## Ports and Endpoints

- Go proxy: `http://localhost:8080` (existing)
- New endpoints:
  - `POST /api/video-monitor/start` - Start monitoring
  - `POST /api/video-monitor/stop` - Stop monitoring
  - `GET /api/video-monitor/status` - Get current status
  - `GET /api/video-monitor/stream` - SSE event stream
  - `POST /api/video-monitor/events` - Internal endpoint for video server to send events

## Environment Variables

- `VIDEO_MONITOR_PYTHON_PATH` - Path to Python executable (default: `python3`)
- `VIDEO_MONITOR_SCRIPT_PATH` - Path to run_monitor.py (default: `./screen_guard_service/run_monitor.py`)

