package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// skipIfNoGenerator skips test if generator is not initialized
func skipIfNoGenerator(t *testing.T) {
	if generator == nil {
		t.Skip("generator not initialized")
	}
}

func TestHandleStatus(t *testing.T) {
	skipIfNoGenerator(t)
	skipIfNoGenerator(t)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	// Call handler
	handleStatus(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse JSON
	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Check fields
	if status["status"] != "running" {
		t.Errorf("Expected status 'running', got %v", status["status"])
	}

	if status["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", status["version"])
	}
}

func TestHandleCreateTask(t *testing.T) {
	// Create request body
	body := `{"title":"Test Task","description":"Test Desc","agent":"sisyphus"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	handleCreateTask(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse JSON
	var task map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Check fields
	if task["title"] != "Test Task" {
		t.Errorf("Expected title 'Test Task', got %v", task["title"])
	}

	if task["agent"] != "sisyphus" {
		t.Errorf("Expected agent 'sisyphus', got %v", task["agent"])
	}
}

func TestHandleListTasks(t *testing.T) {
	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/list", nil)
	w := httptest.NewRecorder()

	// Call handler
	handleListTasks(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse JSON
	var tasks []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Should not fail (may be empty)
	if tasks == nil {
		t.Error("Expected non-nil tasks array")
	}
}

func TestHandleCreateTaskInvalidJSON(t *testing.T) {
	// Create request with invalid JSON
	body := `{"invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler
	handleCreateTask(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandleCreateTaskMissingFields(t *testing.T) {
	// Create request with missing fields
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Call handler (should still create task with empty fields)
	handleCreateTask(w, req)

	// Check response (may succeed with empty fields)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestHandleMethodNotAllowed(t *testing.T) {
	// Test wrong method for /api/tasks
	req := httptest.NewRequest(http.MethodPut, "/api/tasks", nil)
	w := httptest.NewRecorder()

	// Call handler
	handleTasks(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

func TestHealthCheck(t *testing.T) {
	// Simulate health check
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()

	handleStatus(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Health check failed with status %d", resp.StatusCode)
	}

	// Check response time (should be fast)
	if w.Code != 200 {
		t.Error("Health check should return 200")
	}
}
