package config

import (
	"os"
	"strconv"
)

type Config struct {
	BatchSize            int
	BatchInterval        int
	ExternalPostEndpoint string
	LogLevel             string
	LogFormat            string
}

func New() (c Config) {
	c.LogLevel = GetEnv("LOG_LEVEL", "INFO")
	c.LogFormat = GetEnv("LOG_FORMAT", "JSON")
	c.BatchSize = GetEnvInt("BATCH_SIZE", 5)
	c.BatchInterval = GetEnvInt("BATCH_INTERVAL", 15)
	c.ExternalPostEndpoint = GetEnv("POST_ENDPOINT", "https://webhook.site/68b6a469-ef5a-4ec9-992a-b78f7c7694ee")
	return c
}

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
