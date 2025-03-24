package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"webhook-receiver/internal/model"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Creates a test logger for capturing logs during testing
func getTestLogger() *zap.SugaredLogger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, _ := cfg.Build()
	return logger.Sugar()
}

// Tests the addition of payloads and batch processing when the batch size is reached.
func TestBatcher_AddAndProcessBySize(t *testing.T) {
	logger := getTestLogger()
	sentCount := 0

	// Mock server to count batch sends
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payloads []model.Payload
		if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		sentCount += len(payloads)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	batcher := NewBatcher(2, 5, testServer.URL, logger)
	go batcher.Run()

	// Simulate adding payloads
	payload := model.Payload{
		UserID:    1,
		Total:     5.0,
		Title:     "Test",
		Completed: true,
	}

	batcher.Add(payload)
	batcher.Add(payload)

	// Allow time for batch processing
	time.Sleep(2 * time.Second)
	batcher.Stop()

	if sentCount != 2 {
		t.Errorf("Expected 2 payloads sent, but got %d", sentCount)
	}
}

// Verifies that the batcher shuts down gracefully without leaving active processes.
func TestBatcher_GracefulShutdown(t *testing.T) {
	logger := getTestLogger()
	batcher := NewBatcher(2, 1, "http://localhost", logger)
	go batcher.Run()

	batcher.Stop()

	select {
	case <-batcher.quit:
		// Success: Batcher quit gracefully
	case <-time.After(2 * time.Second):
		t.Fatal("Batcher did not shut down gracefully")
	}
}

// Tests retry logic for failed batch sends, ensuring retries happen as expected.
func TestBatcher_RetryLogic(t *testing.T) {
	logger := getTestLogger()
	retryCount := 0

	// Mock server to simulate failure and success on retry
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < 3 {
			http.Error(w, "failed", http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()

	batcher := NewBatcher(1, 1, testServer.URL, logger)
	go batcher.Run()

	payload := model.Payload{
		UserID:    1,
		Total:     5.0,
		Title:     "Retry Test",
		Completed: true,
	}

	batcher.Add(payload)
	time.Sleep(2 * time.Second)
	batcher.Stop()

	if retryCount != 3 {
		t.Errorf("Expected 3 retries, got %d", retryCount)
	}
}

// Tests the handling of an empty batch to ensure it is skipped gracefully.
func TestBatcher_EmptyBatch(t *testing.T) {
	logger := getTestLogger()
	batcher := NewBatcher(1, 1, "http://localhost", logger)

	// Intentionally sending an empty batch
	batcher.sendBatch([]model.Payload{})
}

// Ensures that the batch is not sent prematurely when the batch size has not been reached.
func TestBatcher_BatchSizeNotReached(t *testing.T) {
	logger := getTestLogger()
	sent := false

	// Mock server to track premature batch sends
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = true
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	batcher := NewBatcher(10, 3600, testServer.URL, logger)
	go batcher.Run()

	payload := model.Payload{
		UserID:    1,
		Total:     5.0,
		Title:     "Incomplete Batch",
		Completed: false,
	}

	batcher.Add(payload)
	batcher.Add(payload)

	// Allow some time to verify no premature sends
	time.Sleep(2 * time.Second)

	if sent {
		t.Error("Batch was sent prematurely before reaching the required size")
	} else {
		t.Log("Batch was not sent prematurely, as expected")
	}

	batcher.Stop()

	if !sent {
		t.Error("Batch was not sent after stopping the batcher")
	} else {
		t.Log("Batch was sent after stopping the batcher, as expected")
	}
}

// Verifies time-based batch processing when the interval is reached.
func TestBatcher_TimeBasedBatch(t *testing.T) {
	logger := getTestLogger()
	sentCount := 0

	// Mock server to count payloads sent
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payloads []model.Payload
		if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		sentCount += len(payloads)
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	batcher := NewBatcher(10, 1, testServer.URL, logger)
	go batcher.Run()

	payload := model.Payload{
		UserID:    1,
		Total:     3.5,
		Title:     "Time-Based Test",
		Completed: true,
	}

	batcher.Add(payload)
	time.Sleep(2 * time.Second)
	batcher.Stop()

	if sentCount != 1 {
		t.Errorf("Expected 1 payload sent due to time-based batch, but got %d", sentCount)
	}
}
