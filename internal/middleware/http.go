// Package middleware holds the Gin middleware used by the API. These were
// previously defined inline in cmd/api/main.go, where they could not be tested
// or reused by any other entry point.
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Nuu-maan/video-streaming-service/internal/config"
	"github.com/Nuu-maan/video-streaming-service/pkg/appctx"
	"github.com/Nuu-maan/video-streaming-service/pkg/logger"
)

// RequestIDHeader carries the correlation ID in and out of the service.
const RequestIDHeader = "X-Request-ID"

// RequestID attaches a correlation ID to the request context and echoes it
// back on the response, reusing an inbound ID when the caller supplies one.
//
// The ID goes into the context under a typed key (see pkg/appctx). The previous
// implementation used a bare string key, which go vet flags because any other
// package using the same string silently collides.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Header(RequestIDHeader, requestID)
		c.Request = c.Request.WithContext(appctx.WithRequestID(c.Request.Context(), requestID))

		c.Next()
	}
}

// Logger writes one structured line per request. Server errors are logged at
// error level so they are not buried among successful requests.
func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		fields := map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
			"client_ip":  c.ClientIP(),
			"bytes":      c.Writer.Size(),
		}

		ctx := c.Request.Context()
		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			log.Error(ctx, "request failed", c.Errors.Last(), fields)
		case c.Writer.Status() >= http.StatusBadRequest:
			log.Warn(ctx, "request rejected", fields)
		default:
			log.Info(ctx, "request completed", fields)
		}
	}
}

// CORS applies the configured cross-origin policy.
//
// The origin is echoed back only when it appears in the allowlist. The previous
// implementation sent `Access-Control-Allow-Origin: *` together with
// `Access-Control-Allow-Credentials: true` — a combination browsers reject
// outright, and which would be a CSRF hole if they honoured it.
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(int(cfg.MaxAge.Seconds()))

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if origin != "" && cfg.AllowsOrigin(origin) {
			header := c.Writer.Header()

			if cfg.AllowsAnyOrigin() {
				// Wildcard and credentials are mutually exclusive. Config
				// validation forbids this pairing in production; in development
				// we honour the wildcard and drop credentials rather than emit
				// a combination the browser will reject.
				header.Set("Access-Control-Allow-Origin", "*")
			} else {
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Credentials", "true")
				// The response varies by origin, so a shared cache must not
				// serve one origin's response to another.
				header.Add("Vary", "Origin")
			}

			header.Set("Access-Control-Allow-Methods", allowedMethods)
			header.Set("Access-Control-Allow-Headers", allowedHeaders)
			header.Set("Access-Control-Max-Age", maxAge)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Recovery converts a panic into a 500 with the request ID preserved, logging
// the panic with its stack rather than printing it to stdout as gin's default
// recovery does.
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		log.Error(c.Request.Context(), "panic recovered", nil, map[string]interface{}{
			"panic": recovered,
			"path":  c.Request.URL.Path,
		})
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Internal server error",
			},
		})
	})
}
