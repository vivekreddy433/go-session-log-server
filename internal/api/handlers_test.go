package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"webhook-receiver/internal/model"

	"github.com/gin-gonic/gin"
)

// MockBatcher implements the ServiceBatcher interface for testing
type MockBatcher struct {
	mu      sync.Mutex
	logs    []model.Payload
	stopped bool
}

// Add mock implementation to track received payloads
func (m *MockBatcher) Add(payload model.Payload) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, payload)
}

// Stop mock implementation to track graceful shutdown
func (m *MockBatcher) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
}

// GetLogs returns the collected logs
func (m *MockBatcher) GetLogs() []model.Payload {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logs
}

// IsStopped returns whether the batcher has been stopped
func (m *MockBatcher) IsStopped() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

// TestHealthCheck ensures the health check endpoint works correctly
func TestHealthCheck(t *testing.T) {
	router := gin.Default()
	router.GET("/healthz", HealthCheck)

	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Result().StatusCode)
	}
}

// TestHandleLog_ValidPayload ensures the log endpoint processes valid logs correctly
func TestHandleLog_ValidPayload(t *testing.T) {
	router := gin.Default()
	batcher := &MockBatcher{}
	router.POST("/log", HandleLog(batcher))

	payload := model.Payload{
		UserID:    1,
		Total:     100.50,
		Title:     "Test Log",
		Completed: true,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d", w.Result().StatusCode)
	}

	// Verify the log was received
	logs := batcher.GetLogs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].UserID != 1 {
		t.Errorf("expected UserID 1, got %d", logs[0].UserID)
	}
}

// TestHandleLog_BadRequest tests invalid JSON payload
func TestHandleLog_BadRequest(t *testing.T) {
	router := gin.Default()
	batcher := &MockBatcher{}
	router.POST("/log", HandleLog(batcher))

	req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer([]byte(`invalid payload`)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Result().StatusCode)
	}
}

// TestHandleLog_EmptyPayload tests empty JSON payload
func TestHandleLog_EmptyPayload(t *testing.T) {
	router := gin.Default()
	batcher := &MockBatcher{}
	router.POST("/log", HandleLog(batcher))

	req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer([]byte(`{}`)))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted for empty payload, got %d", w.Result().StatusCode)
	}
}

// TestHandleLog_Concurrency tests concurrent log requests for stability
func TestHandleLog_Concurrency(t *testing.T) {
	router := gin.Default()
	batcher := &MockBatcher{}
	router.POST("/log", HandleLog(batcher))

	var wg sync.WaitGroup
	payload := model.Payload{
		UserID:    42,
		Total:     99.99,
		Title:     "Concurrent Log",
		Completed: true,
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(payload)
			req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Result().StatusCode != http.StatusAccepted {
				t.Errorf("expected 202 Accepted, got %d", w.Result().StatusCode)
			}
		}()
	}
	wg.Wait()

	logs := batcher.GetLogs()
	if len(logs) != 100 {
		t.Errorf("expected 100 logs, got %d", len(logs))
	}
}

// TestHandleLog_LargePayload tests handling of large payloads
func TestHandleLog_LargePayload(t *testing.T) {
	router := gin.Default()
	batcher := &MockBatcher{}
	router.POST("/log", HandleLog(batcher))

	payload := model.Payload{
		UserID:    999,
		Total:     9999.99,
		Title:     "Large Payload Test",
		Completed: true,
	}

	// Simulate a large payload by repeating the title many times
	payload.Title = payload.Title + " " + string(make([]byte, 10000))

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusAccepted {
		t.Errorf("expected 202 Accepted, got %d", w.Result().StatusCode)
	}
}
