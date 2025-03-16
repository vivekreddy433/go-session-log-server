package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"webhook-receiver/internal/model"
	"webhook-receiver/internal/service"
)

func HealthCheck(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

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
