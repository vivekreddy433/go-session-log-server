package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"webhook-receiver/internal/api"
	"webhook-receiver/internal/service"
)

// GetEnv retrieves an environment variable or returns a default if not set
func GetEnv(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}

// GetEnvInt reads an integer from environment variables
func GetEnvInt(key string, defaultVal int) int {
	valStr := GetEnv(key, strconv.Itoa(defaultVal))
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

// Configures the application logger based on environment settings
func ConfigureLogger() *zap.SugaredLogger {
	logLevel := GetEnv("LOG_LEVEL", "INFO")   // Default: INFO
	logFormat := GetEnv("LOG_FORMAT", "JSON") // Default: JSON

	// Set the log level
	var level zapcore.Level
	switch logLevel {
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
		EncodeTime: zapcore.TimeEncoderOfLayout(time.RFC3339),
	}

	// Use JSON for structured logging or plain text for readability.
	var encoder zapcore.Encoder
	if logFormat == "TEXT" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), level)
	logger := zap.New(core)

	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Error syncing logger: %v", err)
		}
	}()

	return logger.Sugar()
}

func main() {
	// Load batch processing configuration from environment variables.
    // If not set, it defaults to batch size of 10, interval of 60 seconds,
    // and a predefined post endpoint for sending logs.
	batchSize := GetEnvInt("BATCH_SIZE", 10)
	batchInterval := GetEnvInt("BATCH_INTERVAL", 60)
	postEndpoint := GetEnv("POST_ENDPOINT", "https://webhook.site/bfaf021f-150a-48dd-bf4e-d1110b5f5874")

	// initializes a structured logger with configurable format and level.
	logger := ConfigureLogger()
	logger.Infof("Starting server with batch size: %d, interval: %d seconds, post endpoint: %s", batchSize, batchInterval, postEndpoint)

	// Initialize batch processor
	batcher := service.NewBatcher(batchSize, batchInterval, postEndpoint, logger)
	go batcher.Run()

	// Initialize API router
	router := gin.New()
	router.Use(gin.Recovery(), api.RequestLogger(logger))

	router.GET("/healthz", api.HealthCheck)
	router.POST("/log", api.HandleLog(batcher))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Infof("Shutdown signal received")
		batcher.Stop()
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server error: ", err)
	}
}
