package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVideoMonitorStatus(t *testing.T) {
	srv := NewServer()
	
	req := httptest.NewRequest("GET", "/api/video-monitor/status", nil)
	w := httptest.NewRecorder()
	
	srv.videoMonitorStatusHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var status VideoMonitorStatus
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if status.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", status.Status)
	}
}

func TestVideoMonitorStartStop(t *testing.T) {
	srv := NewServer()
	
	// Test start (may fail if Python not available, but should not crash)
	req := httptest.NewRequest("POST", "/api/video-monitor/start", nil)
	w := httptest.NewRecorder()
	
	srv.videoMonitorStartHandler(w, req)
	
	// Accept either success (200) or error (500) - depends on Python availability
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
	
	// If start succeeded, test stop
	if w.Code == http.StatusOK {
		time.Sleep(100 * time.Millisecond) // Give it a moment to start
		
		req = httptest.NewRequest("POST", "/api/video-monitor/stop", nil)
		w = httptest.NewRecorder()
		
		srv.videoMonitorStopHandler(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for stop, got %d", w.Code)
		}
	}
}

func TestVideoMonitorEvents(t *testing.T) {
	srv := NewServer()
	
	// Test with invalid JSON (empty body)
	req := httptest.NewRequest("POST", "/api/video-monitor/events", nil)
	w := httptest.NewRecorder()
	
	srv.videoMonitorEventsHandler(w, req)
	
	// Should return 400 (bad request) since we didn't send valid JSON
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid request, got %d", w.Code)
	}
}

func TestVideoMonitorManager(t *testing.T) {
	vm := NewVideoMonitorManager()
	
	// Test initial status
	status := vm.GetStatus()
	if status.Status != "stopped" {
		t.Errorf("Expected initial status 'stopped', got '%s'", status.Status)
	}
	
	// Test client management
	clientCh := make(chan VideoMonitorEvent, 1)
	vm.AddClient(clientCh)
	
	// Send an event
	event := VideoMonitorEvent{
		Type:      "test",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: map[string]interface{}{
			"message": "test",
		},
	}
	
	vm.ReceiveEvent(event)
	
	// Wait a moment for broadcast
	time.Sleep(50 * time.Millisecond)
	
	// Check if event was received
	select {
	case received := <-clientCh:
		if received.Type != "test" {
			t.Errorf("Expected event type 'test', got '%s'", received.Type)
		}
	default:
		t.Error("Event was not received by client")
	}
	
	// Clean up
	vm.RemoveClient(clientCh)
}

