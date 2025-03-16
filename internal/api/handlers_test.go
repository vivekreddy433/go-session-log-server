package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
	"net/http/httptest"
	"testing"
	"webhook-receiver/internal/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestHealthCheck(t *testing.T){
	router := gin.Default()
	logger := zap.NewNop().Sugar()
	router.GET("/health", RequestLogger(logger), HealthCheck)
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected 200 but got %d", w.Result().StatusCode)
	}
}

type MockBatcher struct{}

func (m *MockBatcher) Add(payload model.Payload) {}

func TestHandleLog(t *testing.T) {
	router := gin.Default()
	mockBatcher := &MockBatcher{}
	router.POST("/log", HandleLog(mockBatcher))

	payload := model.Payload{
		UserID: 1,
		Total:  1,
		Title:  "fake-user-login",
		Meta: model.Meta{
			Logins: []model.Login{
				{
					Time: time.Now(),
					IP:   "127.0.0.1",
				},
			},
			PhoneNumbers: model.PhoneNumbers{
				Home:   "0987654321",
				Mobile: "1234567890",
			},
		},
		Completed: true,
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer(payloadBytes))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusAccepted {
		t.Errorf("expected 200 but got %d", w.Result().StatusCode)
	}
}

func TestHandleLog_BadRequest(t *testing.T) {
	router := gin.Default()
	mockBatcher := &MockBatcher{}
	router.POST("/log", HandleLog(mockBatcher))

	req, _ := http.NewRequest(http.MethodPost, "/log", bytes.NewBuffer([]byte(`{"user_id": "1"}`)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Errorf("expected 200 but got %d", w.Result().StatusCode)
	}
}