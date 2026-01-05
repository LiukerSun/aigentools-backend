package middleware

import (
	"aigentools-backend/pkg/logger"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Logger returns a gin.HandlerFunc (middleware) that logs requests using zap
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Request ID
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header("X-Request-ID", requestID)
		}
		// Also set it in context for other parts of the app to use
		c.Set("RequestID", requestID)

		// Process request
		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		if len(c.Errors) > 0 {
			// Append error field if there are errors
			for _, e := range c.Errors.Errors() {
				logger.Log.Error(e, zap.String("request_id", requestID))
			}
		} else {
			fields := []zap.Field{
				zap.String("request_id", requestID),
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.Duration("latency", latency),
			}

			if c.Writer.Status() >= 500 {
				logger.Log.Error("Server Error", fields...)
			} else if c.Writer.Status() >= 400 {
				logger.Log.Warn("Client Error", fields...)
			} else {
				logger.Log.Info("Request", fields...)
			}
		}
	}
}
