package screenguard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// Status represents the current state of the screen guard service
type Status struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"startedAt"`
	LastError string    `json:"lastError,omitempty"`
	LogFile   string    `json:"logFile,omitempty"`
}

// Manager manages the lifecycle of the screen guard service
type Manager struct {
	mu          sync.RWMutex
	process     *os.Process
	status      Status
	pythonPath  string
	serviceDir  string
	logDir      string
	stopTimeout time.Duration
}

// NewManager creates a new screen guard service manager
func NewManager() (*Manager, error) {
	// Get repo root (assume we're running from repo root)
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Find Python executable
	pythonPath, err := findPython()
	if err != nil {
		return nil, fmt.Errorf("python not found: %w", err)
	}

	// Resolve service directory
	serviceDir := filepath.Join(repoRoot, "screen_guard_service")
	if _, err := os.Stat(serviceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("service directory not found: %s", serviceDir)
	}

	// Default log directory (service will create its own structure)
	logDir := filepath.Join(serviceDir, "monitor_output")

	return &Manager{
		status: Status{
			Running:   false,
			PID:       0,
			StartedAt: time.Time{},
			LastError: "",
			LogFile:   "",
		},
		pythonPath:  pythonPath,
		serviceDir:  serviceDir,
		logDir:      logDir,
		stopTimeout: 5 * time.Second,
	}, nil
}

// findPython finds the Python executable
func findPython() (string, error) {
	// Check environment variable first
	if path := os.Getenv("SCREEN_GUARD_PYTHON_PATH"); path != "" {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}

	// Try common Python executable names
	candidates := []string{"python3", "python"}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("python executable not found in PATH")
}

// Start starts the screen guard service
func (m *Manager) Start() (*Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Idempotent: if already running, return current status
	if m.status.Running && m.process != nil {
		// Verify process is still alive
		if err := m.process.Signal(syscall.Signal(0)); err == nil {
			return &m.status, nil
		}
		// Process is dead, reset state
		m.status.Running = false
		m.process = nil
		m.status.PID = 0
	}

	// Build command
	cmd := exec.Command(m.pythonPath, "-m", "screen_guard_service")
	cmd.Dir = m.serviceDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables (preserve existing, add service-specific)
	cmd.Env = os.Environ()

	// Start process
	if err := cmd.Start(); err != nil {
		m.status.LastError = err.Error()
		return &m.status, fmt.Errorf("failed to start service: %w", err)
	}

	// Track process
	m.process = cmd.Process
	m.status.Running = true
	m.status.PID = cmd.Process.Pid
	m.status.StartedAt = time.Now()
	m.status.LastError = ""
	
	// Try to find log file (service creates it, so we'll update this later)
	// For now, set a placeholder
	m.status.LogFile = filepath.Join(m.logDir, "runtime.log")

	// Start goroutine to monitor process
	go m.monitorProcess()

	return &m.status, nil
}

// Stop stops the screen guard service
func (m *Manager) Stop() (*Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Idempotent: if not running, return success
	if !m.status.Running || m.process == nil {
		m.status.Running = false
		m.status.PID = 0
		m.status.StartedAt = time.Time{}
		m.status.LastError = ""
		m.status.LogFile = ""
		return &m.status, nil
	}

	// Try graceful shutdown first
	if err := m.gracefulStop(); err != nil {
		// If graceful stop fails, force kill
		if killErr := m.forceStop(); killErr != nil {
			m.status.LastError = fmt.Sprintf("stop failed: %v, kill failed: %v", err, killErr)
			return &m.status, fmt.Errorf("failed to stop service: %w", killErr)
		}
	}

	// Reset state
	m.status.Running = false
	m.process = nil
	m.status.PID = 0
	m.status.StartedAt = time.Time{}
	m.status.LastError = ""
	m.status.LogFile = ""

	return &m.status, nil
}

// gracefulStop attempts graceful shutdown (SIGTERM)
func (m *Manager) gracefulStop() error {
	if m.process == nil {
		return nil
	}

	// Send SIGTERM
	var err error
	if runtime.GOOS == "windows" {
		// Windows: use taskkill for graceful termination
		cmd := exec.Command("taskkill", "/PID", fmt.Sprintf("%d", m.process.Pid), "/T")
		err = cmd.Run()
	} else {
		// Unix: send SIGTERM
		err = m.process.Signal(syscall.SIGTERM)
	}

	if err != nil {
		return err
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		_, waitErr := m.process.Wait()
		done <- waitErr
	}()

	select {
	case <-done:
		return nil
	case <-time.After(m.stopTimeout):
		return fmt.Errorf("graceful stop timeout")
	}
}

// forceStop forces the process to stop (SIGKILL)
func (m *Manager) forceStop() error {
	if m.process == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		// Windows: use taskkill /F for force kill
		cmd := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", m.process.Pid), "/T")
		return cmd.Run()
	}

	// Unix: send SIGKILL
	if err := m.process.Signal(syscall.SIGKILL); err != nil {
		return err
	}

	// Wait a bit for process to die
	_, err := m.process.Wait()
	return err
}

// GetStatus returns the current status
func (m *Manager) GetStatus() *Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Make a copy to avoid race conditions
	status := m.status

	// Verify process is still alive if we think it's running
	if status.Running && m.process != nil {
		if err := m.process.Signal(syscall.Signal(0)); err != nil {
			// Process is dead, update state
			m.mu.RUnlock()
			m.mu.Lock()
			m.status.Running = false
			m.process = nil
			m.status.PID = 0
			status = m.status
			m.mu.Unlock()
			m.mu.RLock()
		}
	}

	return &status
}

// monitorProcess monitors the process and updates state when it exits
func (m *Manager) monitorProcess() {
	if m.process == nil {
		return
	}

	// Store PID before waiting (process may be nil after wait)
	pid := m.process.Pid

	// Wait for process to exit
	_, err := m.process.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Process exited, update state (only if it's still the same process)
	if m.process != nil && m.process.Pid == pid {
		m.status.Running = false
		if err != nil {
			m.status.LastError = err.Error()
		}
		m.process = nil
		m.status.PID = 0
	}
}

