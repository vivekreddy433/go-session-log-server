package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestLogger(logger *zap.SugaredLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		logger.Infof("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration)
	}
}
