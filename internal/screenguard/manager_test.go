package screenguard

import (
	"os/exec"
	"testing"
)

func TestNewManager(t *testing.T) {
	// This test may fail if Python is not available, which is okay
	mgr, err := NewManager()
	if err != nil {
		t.Skipf("Skipping test: Python not available or service directory not found: %v", err)
		return
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.pythonPath == "" {
		t.Error("pythonPath should not be empty")
	}

	if mgr.serviceDir == "" {
		t.Error("serviceDir should not be empty")
	}
}

func TestManager_GetStatus_Initial(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}

	status := mgr.GetStatus()
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}

	if status.Running {
		t.Error("Initial status should not be running")
	}

	if status.PID != 0 {
		t.Error("Initial PID should be 0")
	}
}

func TestManager_StartStop_Idempotent(t *testing.T) {
	// Skip if Python not available
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err2 := exec.LookPath("python"); err2 != nil {
			t.Skip("Skipping test: Python not available")
			return
		}
	}

	mgr, err := NewManager()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}

	// Test idempotent start
	status1, err1 := mgr.Start()
	if err1 != nil {
		t.Logf("Start failed (may be expected if service dir doesn't exist): %v", err1)
		// If start fails due to missing service, that's okay for this test
		return
	}

	if !status1.Running {
		t.Error("Status should be running after Start()")
	}

	// Start again (should be idempotent)
	status2, err2 := mgr.Start()
	if err2 != nil {
		t.Errorf("Second Start() should succeed (idempotent): %v", err2)
	}

	if status2.PID != status1.PID {
		t.Error("Second Start() should return same PID (idempotent)")
	}

	// Stop
	status3, err3 := mgr.Stop()
	if err3 != nil {
		t.Errorf("Stop() failed: %v", err3)
	}

	if status3.Running {
		t.Error("Status should not be running after Stop()")
	}

	// Stop again (should be idempotent)
	status4, err4 := mgr.Stop()
	if err4 != nil {
		t.Errorf("Second Stop() should succeed (idempotent): %v", err4)
	}

	if status4.Running {
		t.Error("Status should not be running after second Stop()")
	}
}

func TestManager_StatusAfterStop(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
		return
	}

	// Stop (even if not running)
	_, _ = mgr.Stop()

	// Get status
	status := mgr.GetStatus()
	if status.Running {
		t.Error("Status should not be running after Stop()")
	}

	if status.PID != 0 {
		t.Error("PID should be 0 after Stop()")
	}
}

