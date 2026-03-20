package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		attrs := []any{
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if status >= 500 {
			slog.Error("HTTP request", attrs...)
		} else if status >= 400 {
			slog.Warn("HTTP request", attrs...)
		} else {
			slog.Info("HTTP request", attrs...)
		}
	}
}
