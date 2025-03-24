package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"webhook-receiver/config"
	"webhook-receiver/internal/api"
	"webhook-receiver/internal/service"
)

// middleware to set logger at handler level
func RequestLogger(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		logger.Infof("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
	}
}

// ConfigureLogger sets up the structured logger
func ConfigureLogger(c config.Config) *zap.SugaredLogger {
	var level zapcore.Level
	switch c.LogLevel {
	case "DEBUG":
		level = zap.DebugLevel
	case "INFO":
		level = zap.InfoLevel
	case "WARN":
		level = zap.WarnLevel
	case "ERROR":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:    "timestamp",
		LevelKey:   "level",
		MessageKey: "message",
		CallerKey:  "caller",
		EncodeTime: zapcore.ISO8601TimeEncoder,
	}

	var encoder zapcore.Encoder
	if c.LogFormat == "TEXT" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), level)
	return zap.New(core).Sugar()
}

func main() {

	config := config.New()
	logger := ConfigureLogger(config)
	logger.Infof("Starting server with batch size: %d, interval: %d seconds, post endpoint: %s", config.BatchSize, config.BatchInterval, config.ExternalPostEndpoint)

	batcher := service.NewBatcher(config.BatchSize, config.BatchInterval, config.ExternalPostEndpoint, logger)
	go batcher.Run()

	router := gin.New()
	router.Use(gin.Recovery(), RequestLogger(logger))

	router.GET("/healthz", api.HealthCheck)
	router.POST("/log", api.HandleLog(batcher))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Infof("Shutdown signal received")
		batcher.Stop()
		if err := srv.Close(); err != nil {
			logger.Errorf("Server shutdown error: %v", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	logger.Info("Server stopped gracefully")
}
