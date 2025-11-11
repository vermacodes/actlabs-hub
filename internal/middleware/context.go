package middleware

import (
	"context"

	"actlabs-hub/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetContextFromGin extracts context from Gin context
func GetContextFromGin(c *gin.Context) context.Context {
	return c.Request.Context()
}

// SetUserIDInGin sets user ID in Gin context
func SetUserIDInGin(c *gin.Context, userID string) {
	ctx := logger.WithUserID(c.Request.Context(), userID)
	c.Request = c.Request.WithContext(ctx)
}

// ContextMiddleware adds trace ID and other context data to requests
func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing X-Trace-ID header, otherwise generate new one
		traceID := c.GetHeader("X-Trace-ID")
		if _, err := uuid.Parse(traceID); err != nil {
			traceID = uuid.New().String()
		}

		// Get or create context
		ctx := c.Request.Context()

		// Add trace ID to context
		ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)

		// Update request with new context
		c.Request = c.Request.WithContext(ctx)

		// Set trace ID header for downstream services
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
