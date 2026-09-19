package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs one structured line per request. It replaces gin.Logger,
// whose output is unstructured text.
func RequestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if q := c.Request.URL.RawQuery; q != "" {
			attrs = append(attrs, "query", q)
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "errors", c.Errors.String())
		}

		switch {
		case c.Writer.Status() >= 500:
			log.Error("request failed", attrs...)
		case c.Writer.Status() >= 400:
			log.Warn("request rejected", attrs...)
		default:
			log.Info("request", attrs...)
		}
	}
}
