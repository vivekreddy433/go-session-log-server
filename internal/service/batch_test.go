package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"webhook-receiver/internal/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestBatcherAdd_WhenSizeBasedPostCallTriggered(t *testing.T) {
	router := gin.Default()
	router.Handle(http.MethodPost, "/webhook", func(ctx *gin.Context) {
		var payloads []model.Payload
		if err := ctx.ShouldBindBodyWithJSON(&payloads); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"status": err.Error(),
			})
			return
		}
		if len(payloads) != 2 {
			t.FailNow()
			t.Logf("Expected 2 recoreds in the request payload but got %d", len(payloads))
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "payload count not matching",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})
	testServer := httptest.NewServer(router)
	defer testServer.Close()
	logger := zap.NewNop().Sugar()
	batcher := NewBatcher(2, 100, testServer.URL+"/webhook", logger)

	batcher.Add(model.Payload{
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
	})
	batcher.Add(model.Payload{
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
	})
}

func TestBatcherAdd_WhenFirstPostAPICallFailedShouldMakeAnotherCallInRetry(t *testing.T) {
	router := gin.Default()
	apiCount := 0
	router.Handle(http.MethodPost, "/webhook", func(ctx *gin.Context) {
		if apiCount == 0 {
			apiCount = apiCount + 1
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "failing first API call intentially to test retry logic",
			})
			return
		}
		var payloads []model.Payload
		if err := ctx.ShouldBindBodyWithJSON(&payloads); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"status": err.Error(),
			})
			return
		}
		if len(payloads) != 1 {
			t.FailNow()
			t.Logf("Expected 2 recoreds in the request payload but got %d", len(payloads))
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "payload count not matching",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})
	testServer := httptest.NewServer(router)
	defer testServer.Close()
	logger := zap.NewNop().Sugar()
	batcher := NewBatcher(1, 100, testServer.URL+"/webhook", logger)

	batcher.Add(model.Payload{
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
	})
}

func TestBatcherAdd_WhenFirstPostAPICallFailedAfterAllRetries(t *testing.T) {
	router := gin.Default()
	router.Handle(http.MethodPost, "/webhook", func(ctx *gin.Context) {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "failing first API call intentially to test retry logic",
		})
	})
	testServer := httptest.NewServer(router)
	defer testServer.Close()
	logger := zap.NewNop().Sugar()
	batcher := NewBatcher(1, 100, testServer.URL+"/webhook", logger)

	batcher.Add(model.Payload{
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
	})
}

func TestBatcherAdd_WhenTimeBasedPostCallTriggered(t *testing.T) {
	router := gin.Default()
	router.Handle(http.MethodPost, "/webhook", func(ctx *gin.Context) {
		var payloads []model.Payload
		if err := ctx.ShouldBindBodyWithJSON(&payloads); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"status": err.Error(),
			})
			return
		}
		if len(payloads) != 1 {
			t.FailNow()
			t.Logf("Expected 1 recoreds in the request payload but got %d", len(payloads))
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "payload count not matching",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
	})
	testServer := httptest.NewServer(router)
	defer testServer.Close()
	logger := zap.NewNop().Sugar()
	batcher := NewBatcher(2, 2, testServer.URL+"/webhook", logger)
	go batcher.Run()
	batcher.Add(model.Payload{
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
	})
	time.Sleep(5 * time.Second)
}

func TestBatcherAdd_WhenSendBatchCalledwithEmptyPayload(t *testing.T) {
	logger := zap.NewNop().Sugar()
	batcher := NewBatcher(2, 2, "/webhook", logger)
	batcher.sendBatch()
}

func TestBatcherAdd_WhenTimeBasedPostCallTriggered_StopSignalTriggered(t *testing.T) {
	logger := zap.NewNop().Sugar()
	batcher := NewBatcher(2, 5, "/webhook", logger)
	go batcher.Run()
	batcher.stop <- struct{}{}
	time.Sleep(5 * time.Second)
}
