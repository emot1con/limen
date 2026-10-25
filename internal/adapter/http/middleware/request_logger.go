package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func RequestLogger(logger *logrus.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		status := c.Writer.Status()
		entry := logger.WithFields(logrus.Fields{
			"method":      c.Request.Method,
			"path":        c.FullPath(),
			"status":      status,
			"duration_ms": time.Since(startedAt).Milliseconds(),
			"client_ip":   c.ClientIP(),
			"request_id":  c.GetString("request_id"),
		})
		if len(c.Errors) > 0 {
			entry = entry.WithField("error_count", len(c.Errors))
		}
		if operation := c.GetString("internal_error_operation"); operation != "" {
			entry = entry.WithField("operation", operation)
		}
		if errorType := c.GetString("internal_error_type"); errorType != "" {
			entry = entry.WithField("error_type", errorType)
		}
		if errorCode := c.GetString("internal_error_code"); errorCode != "" {
			entry = entry.WithField("error_code", errorCode)
		}
		switch {
		case status >= 500:
			entry.Error("http request completed")
		case status >= 400:
			entry.Warn("http request completed")
		default:
			entry.Info("http request completed")
		}
	}
}
