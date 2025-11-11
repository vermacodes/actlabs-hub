package middleware

import (
	"io"
	"strings"
	"time"

	"actlabs-hub/internal/logger"

	"github.com/gin-gonic/gin"
)

// LoggingConfig holds configuration for path-based logging
type LoggingConfig struct {
	// QuietPaths - paths that should log at DEBUG level instead of INFO (to reduce noise)
	QuietPaths []string
	// SkipPaths - paths that should not be logged at all
	SkipPaths []string
	// ErrorOnlyPaths - paths that should only log on errors (4xx/5xx)
	ErrorOnlyPaths []string
}

// DefaultLoggingConfig returns sensible defaults for common scenarios
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		QuietPaths: []string{
			// Add paths that are debug only.
		},
		SkipPaths: []string{
			// Add paths you never want to log (like very frequent internal calls)
		},
		ErrorOnlyPaths: []string{
			// Add paths where you only care about errors
			"/status",
			"/health",
			"/healthz",
			"/ping",
			"/metrics",
			"/server",
		},
	}
}

// GinLoggerWithTraceID creates a custom Gin logger middleware that includes trace_id
func GinLoggerWithTraceID() gin.HandlerFunc {
	return GinLoggerWithConfig(DefaultLoggingConfig())
}

// GinLoggerWithConfig creates a custom Gin logger middleware with path-based filtering
func GinLoggerWithConfig(config LoggingConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Check if we should skip logging this path entirely
		if shouldSkipPath(path, config.SkipPaths) {
			c.Next()
			return
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get context
		ctx := c.Request.Context()

		// Build the log entry
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}

		// Create structured log with context (trace_id and user_id automatically included)
		ctxLogger := logger.ContextLog(ctx)

		// Determine log level based on path and status
		logLevel := determineLogLevel(path, statusCode, config)

		// Log based on determined level
		logAttributes := []any{
			"path", path,
			"status", statusCode,
			"latency", latency.String(),
			"client_ip", clientIP,
			"method", method,
			"body_size", bodySize,
		}

		switch logLevel {
		case "debug":
			ctxLogger.Debug("gin request", logAttributes...)
		case "info":
			ctxLogger.Info("gin request", logAttributes...)
		case "warn":
			ctxLogger.Warn("gin request", logAttributes...)
		case "error":
			ctxLogger.Error("gin request", logAttributes...)
		case "skip":
			// Don't log anything
			return
		}
	}
}

// shouldSkipPath checks if a path should be completely skipped from logging
func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if matchesPath(path, skipPath) {
			return true
		}
	}
	return false
}

// determineLogLevel determines the appropriate log level for a request
func determineLogLevel(path string, statusCode int, config LoggingConfig) string {
	// Always log errors at error level
	if statusCode >= 400 {
		return "error"
	}

	// Check if this is an error-only path
	for _, errorOnlyPath := range config.ErrorOnlyPaths {
		if matchesPath(path, errorOnlyPath) {
			return "skip" // Skip successful requests for error-only paths
		}
	}

	// Check if this is a quiet path (health checks, etc.)
	for _, quietPath := range config.QuietPaths {
		if matchesPath(path, quietPath) {
			return "debug" // Log at debug level to reduce noise
		}
	}

	// Default to info level for normal paths
	return "info"
}

// matchesPath checks if a request path matches a configured path pattern
func matchesPath(requestPath, configPath string) bool {
	// Exact match
	if requestPath == configPath {
		return true
	}

	// Prefix match (for paths ending with *)
	if strings.HasSuffix(configPath, "*") {
		prefix := strings.TrimSuffix(configPath, "*")
		return strings.HasPrefix(requestPath, prefix)
	}

	return false
}

// DisableGinDefaultLogging disables Gin's default console logging
func DisableGinDefaultLogging() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
}
