package api

import (
	"net/http"

	"webhook-receiver/internal/model"
	"webhook-receiver/internal/service"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns the health status.
func HealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// HandleLog handles incoming log payloads.
func HandleLog(batcher service.ServiceBatcher) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload model.Payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		batcher.Add(payload)
		c.Status(http.StatusAccepted)
	}
}
