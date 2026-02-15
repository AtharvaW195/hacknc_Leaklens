package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// VideoMonitorEvent represents a unified event from the video monitoring system
type VideoMonitorEvent struct {
	Type      string                 `json:"type"`      // "status", "detection", "log", "error"
	Timestamp string                 `json:"timestamp"` // ISO 8601
	Data      map[string]interface{} `json:"data"`
}

// VideoMonitorStatus represents the current status of video monitoring
type VideoMonitorStatus struct {
	Status      string `json:"status"`       // "starting", "running", "stopping", "stopped", "degraded"
	Message     string `json:"message"`      // Human-readable status message
	StartedAt   string `json:"started_at,omitempty"`
	StoppedAt   string `json:"stopped_at,omitempty"`
	Error       string `json:"error,omitempty"`
	Recoverable bool   `json:"recoverable,omitempty"`
}

// VideoMonitorManager manages the video processing server lifecycle
type VideoMonitorManager struct {
	mu              sync.RWMutex
	status          VideoMonitorStatus
	cmd             *exec.Cmd
	ctx             context.Context
	cancel          context.CancelFunc
	eventCh         chan VideoMonitorEvent
	clients         map[chan VideoMonitorEvent]bool
	clientsMu       sync.RWMutex
	pythonPath      string
	scriptPath      string
	lastEventTime   time.Time
	reconnectTicker *time.Ticker
}

// NewVideoMonitorManager creates a new video monitor manager
func NewVideoMonitorManager() *VideoMonitorManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Default paths - try to find Python automatically
	pythonPath := os.Getenv("VIDEO_MONITOR_PYTHON_PATH")
	if pythonPath == "" {
		// Try common Python executable names (Windows uses "python", Unix uses "python3")
		pythonPath = findPythonExecutable()
	}
	
	scriptPath := os.Getenv("VIDEO_MONITOR_SCRIPT_PATH")
	if scriptPath == "" {
		// Try to find run_monitor.py relative to current working directory
		wd, _ := os.Getwd()
		scriptPath = filepath.Join(wd, "screen_guard_service", "run_monitor.py")
	}
	
	vm := &VideoMonitorManager{
		status: VideoMonitorStatus{
			Status:  "stopped",
			Message: "Video monitoring is stopped",
		},
		ctx:        ctx,
		cancel:     cancel,
		eventCh:    make(chan VideoMonitorEvent, 100),
		clients:    make(map[chan VideoMonitorEvent]bool),
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}
	
	// Start event broadcaster
	go vm.broadcastEvents()
	
	return vm
}

// findPythonExecutable tries to find Python executable
func findPythonExecutable() string {
	// Try common names in order of preference
	candidates := []string{"python", "python3", "py"}
	
	for _, candidate := range candidates {
		// Try to run Python with --version to see if it exists
		cmd := exec.Command(candidate, "--version")
		if err := cmd.Run(); err == nil {
			log.Printf("[VIDEO_MONITOR] Found Python: %s", candidate)
			return candidate
		}
	}
	
	// Default fallback (will show error if not found)
	log.Printf("[VIDEO_MONITOR] Warning: Could not find Python, defaulting to 'python'")
	return "python"
}

// Start begins video monitoring
func (vm *VideoMonitorManager) Start() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	log.Printf("[VIDEO_MONITOR] Start() called, current status: %s", vm.status.Status)
	
	if vm.status.Status == "running" || vm.status.Status == "starting" {
		log.Printf("[VIDEO_MONITOR] Already %s, returning error", vm.status.Status)
		return fmt.Errorf("video monitoring is already %s", vm.status.Status)
	}
	
	log.Printf("[VIDEO_MONITOR] Using Python: %s", vm.pythonPath)
	log.Printf("[VIDEO_MONITOR] Using script: %s", vm.scriptPath)
	
	vm.updateStatus("starting", "Starting video monitoring...", "", false)
	vm.emitEvent("status", map[string]interface{}{
		"status":  "starting",
		"message": "Starting video monitoring...",
	})
	
	// Check if script exists
	if _, err := os.Stat(vm.scriptPath); os.IsNotExist(err) {
		log.Printf("[VIDEO_MONITOR] Script not found: %s", vm.scriptPath)
		vm.updateStatus("stopped", "Video monitoring script not found", err.Error(), false)
		vm.emitEvent("error", map[string]interface{}{
			"code":       "SCRIPT_NOT_FOUND",
			"message":    fmt.Sprintf("Video monitoring script not found: %s", vm.scriptPath),
			"recoverable": false,
		})
		return err
	}
	
	log.Printf("[VIDEO_MONITOR] Script found, starting Python process...")
	
	// Verify Python executable exists
	if _, err := exec.LookPath(vm.pythonPath); err != nil {
		log.Printf("[VIDEO_MONITOR] ERROR: Python executable not found: %s", vm.pythonPath)
		vm.updateStatus("stopped", fmt.Sprintf("Python not found: %s. Set VIDEO_MONITOR_PYTHON_PATH environment variable.", vm.pythonPath), err.Error(), false)
		vm.emitEvent("error", map[string]interface{}{
			"code":       "PYTHON_NOT_FOUND",
			"message":    fmt.Sprintf("Python executable not found: %s. Try setting VIDEO_MONITOR_PYTHON_PATH=python", vm.pythonPath),
			"recoverable": false,
		})
		return fmt.Errorf("python executable not found: %s", vm.pythonPath)
	}
	
	// Start Python process
	// Note: Removed --quiet flag so we can see debug logs
	cmd := exec.CommandContext(vm.ctx, vm.pythonPath, vm.scriptPath, "--mode", "manual")
	cmd.Dir = filepath.Dir(vm.scriptPath)
	cmd.Env = append(os.Environ(), "VIDEO_MONITOR_BRIDGE_URL=http://localhost:8080/api/video-monitor/events")
	
	log.Printf("[VIDEO_MONITOR] Environment: VIDEO_MONITOR_BRIDGE_URL=%s", "http://localhost:8080/api/video-monitor/events")
	
	log.Printf("[VIDEO_MONITOR] Command: %s %s", vm.pythonPath, vm.scriptPath)
	log.Printf("[VIDEO_MONITOR] Working directory: %s", cmd.Dir)
	
	// Capture stdout/stderr to see Python logs
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[VIDEO_MONITOR] Warning: Failed to capture stdout: %v", err)
	} else {
		go vm.captureOutput(stdout, "stdout")
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("[VIDEO_MONITOR] Warning: Failed to capture stderr: %v", err)
	} else {
		go vm.captureOutput(stderr, "stderr")
	}
	
	if err := cmd.Start(); err != nil {
		log.Printf("[VIDEO_MONITOR] cmd.Start() failed: %v", err)
		vm.updateStatus("stopped", "Failed to start video monitoring", err.Error(), true)
		vm.emitEvent("error", map[string]interface{}{
			"code":       "START_FAILED",
			"message":    fmt.Sprintf("Failed to start video server: %v", err),
			"recoverable": true,
		})
		return err
	}
	
	log.Printf("[VIDEO_MONITOR] Process started, PID: %d", cmd.Process.Pid)
	vm.cmd = cmd
	
	// Monitor process health
	go vm.monitorProcess()
	
	// Wait a moment to see if process starts successfully
	time.Sleep(500 * time.Millisecond)
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		exitCode := cmd.ProcessState.ExitCode()
		log.Printf("[VIDEO_MONITOR] Process exited immediately with code: %d", exitCode)
		vm.updateStatus("stopped", fmt.Sprintf("Video monitoring process exited immediately (code: %d)", exitCode), "", true)
		vm.emitEvent("error", map[string]interface{}{
			"code":       "PROCESS_EXITED",
			"message":    fmt.Sprintf("Process exited immediately with code %d", exitCode),
			"recoverable": true,
		})
		return fmt.Errorf("process exited immediately with code %d", exitCode)
	}
	
	log.Printf("[VIDEO_MONITOR] Process is running, updating status to 'running'")
	vm.updateStatus("running", "Video monitoring is running", "", false)
	vm.emitEvent("status", map[string]interface{}{
		"status":  "running",
		"message": "Video monitoring started successfully",
	})
	
	vm.lastEventTime = time.Now()
	
	return nil
}

// Stop stops video monitoring
func (vm *VideoMonitorManager) Stop() error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	if vm.status.Status == "stopped" || vm.status.Status == "stopping" {
		return nil
	}
	
	vm.updateStatus("stopping", "Stopping video monitoring...", "", false)
	vm.emitEvent("status", map[string]interface{}{
		"status":  "stopping",
		"message": "Stopping video monitoring...",
	})
	
	if vm.cmd != nil && vm.cmd.Process != nil {
		// Send interrupt signal
		if err := vm.cmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("Error sending interrupt to video server: %v", err)
		}
		
		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- vm.cmd.Wait()
		}()
		
		select {
		case <-done:
			// Process exited
		case <-time.After(5 * time.Second):
			// Force kill
			if vm.cmd.Process != nil {
				vm.cmd.Process.Kill()
			}
		}
	}
	
	vm.cmd = nil
	vm.updateStatus("stopped", "Video monitoring stopped", "", false)
	vm.emitEvent("status", map[string]interface{}{
		"status":  "stopped",
		"message": "Video monitoring stopped",
	})
	
	return nil
}

// GetStatus returns the current status
func (vm *VideoMonitorManager) GetStatus() VideoMonitorStatus {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.status
}

// ReceiveEvent receives an event from the video server
func (vm *VideoMonitorManager) ReceiveEvent(event VideoMonitorEvent) {
	vm.lastEventTime = time.Now()
	vm.eventCh <- event
}

// AddClient adds an SSE client to receive events
func (vm *VideoMonitorManager) AddClient(clientCh chan VideoMonitorEvent) {
	vm.clientsMu.Lock()
	defer vm.clientsMu.Unlock()
	vm.clients[clientCh] = true
}

// RemoveClient removes an SSE client
func (vm *VideoMonitorManager) RemoveClient(clientCh chan VideoMonitorEvent) {
	vm.clientsMu.Lock()
	defer vm.clientsMu.Unlock()
	delete(vm.clients, clientCh)
	close(clientCh)
}

// broadcastEvents broadcasts events to all connected clients
func (vm *VideoMonitorManager) broadcastEvents() {
	for event := range vm.eventCh {
		vm.clientsMu.RLock()
		for clientCh := range vm.clients {
			select {
			case clientCh <- event:
			default:
				// Client channel is full, skip
			}
		}
		vm.clientsMu.RUnlock()
	}
}

// captureOutput captures stdout/stderr from Python process and logs it
func (vm *VideoMonitorManager) captureOutput(pipe io.ReadCloser, source string) {
	defer pipe.Close()
	
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("[PYTHON-%s] %s", strings.ToUpper(source), line)
	}
	
	if err := scanner.Err(); err != nil {
		log.Printf("[VIDEO_MONITOR] Error reading %s: %v", source, err)
	}
}

// monitorProcess monitors the video server process health
func (vm *VideoMonitorManager) monitorProcess() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	log.Printf("[VIDEO_MONITOR] Process monitor started")
	
	for {
		select {
		case <-vm.ctx.Done():
			log.Printf("[VIDEO_MONITOR] Process monitor stopped (context cancelled)")
			return
		case <-ticker.C:
			vm.mu.RLock()
			cmd := vm.cmd
			status := vm.status.Status
			vm.mu.RUnlock()
			
			if status == "running" && cmd != nil {
				// Check if process has exited
				// On Windows, ProcessState is only set after Wait() is called
				// So we need to check it in a non-blocking way
				if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
					exitCode := cmd.ProcessState.ExitCode()
					log.Printf("[VIDEO_MONITOR] Process exited with code: %d", exitCode)
					// Process died
					vm.mu.Lock()
					vm.updateStatus("degraded", "Video monitoring process disconnected", fmt.Sprintf("Process exited with code %d", exitCode), true)
					vm.cmd = nil
					vm.mu.Unlock()
					
					vm.emitEvent("error", map[string]interface{}{
						"code":       "PROCESS_DIED",
						"message":    fmt.Sprintf("Video monitoring process exited unexpectedly (code: %d)", exitCode),
						"recoverable": true,
					})
					
					// Attempt reconnect
					go vm.attemptReconnect()
					return
				}
				
				// On Windows, Process.Signal() doesn't work, so we can't use it to check if process is alive
				// Instead, we rely on:
				// 1. ProcessState check above (if process has exited, ProcessState will be set)
				// 2. Event heartbeat check below (if no events for 30s, assume disconnected)
				// If ProcessState is nil, the process is still running (or hasn't been waited on yet)
				
				// Check if we've received events recently (heartbeat)
				vm.mu.RLock()
				timeSinceLastEvent := time.Since(vm.lastEventTime)
				vm.mu.RUnlock()
				
				if timeSinceLastEvent > 30*time.Second {
					// No events for 30 seconds - might be disconnected
					log.Printf("[VIDEO_MONITOR] No events received for %.0f seconds", timeSinceLastEvent.Seconds())
					vm.mu.Lock()
					if vm.status.Status == "running" {
						vm.updateStatus("degraded", "Video monitoring may be disconnected", "No events received for 30 seconds", true)
						vm.emitEvent("error", map[string]interface{}{
							"code":       "NO_HEARTBEAT",
							"message":    "No events received from video server for 30 seconds",
							"recoverable": true,
						})
					}
					vm.mu.Unlock()
				}
			}
		}
	}
}

// attemptReconnect attempts to reconnect to the video server
func (vm *VideoMonitorManager) attemptReconnect() {
	time.Sleep(2 * time.Second)
	
	vm.mu.Lock()
	defer vm.mu.Unlock()
	
	if vm.status.Status == "stopped" || vm.status.Status == "stopping" {
		return // User stopped it
	}
	
	vm.emitEvent("log", map[string]interface{}{
		"level":     "info",
		"component": "monitor",
		"message":   "Attempting to reconnect to video server...",
	})
	
	if err := vm.Start(); err != nil {
		log.Printf("Reconnect failed: %v", err)
	}
}

// updateStatus updates the internal status
func (vm *VideoMonitorManager) updateStatus(status, message, error string, recoverable bool) {
	now := time.Now().UTC().Format(time.RFC3339)
	
	if status == "running" && vm.status.StartedAt == "" {
		vm.status.StartedAt = now
	}
	if status == "stopped" && vm.status.StoppedAt == "" {
		vm.status.StoppedAt = now
	}
	
	vm.status.Status = status
	vm.status.Message = message
	vm.status.Error = error
	vm.status.Recoverable = recoverable
}

// emitEvent emits an event to the event channel
func (vm *VideoMonitorManager) emitEvent(eventType string, data map[string]interface{}) {
	event := VideoMonitorEvent{
		Type:      eventType,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}
	
	select {
	case vm.eventCh <- event:
	default:
		// Event channel is full, log warning
		log.Printf("Warning: video monitor event channel full, dropping event")
	}
}

