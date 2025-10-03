package services

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateUUID()
		}

		c.Set("request_id", requestID)

		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

func generateUUID() string {
	return uuid.New().String()
}

func GetRequestID(c *gin.Context) string {
	requestID, exists := c.Get("request_id")
	if !exists {
		return ""
	}

	if id, ok := requestID.(string); ok {
		return id
	}

	return ""
}
