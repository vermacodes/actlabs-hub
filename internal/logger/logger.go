package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Context keys to avoid collisions
type contextKey string

const (
	UserIDKey  contextKey = "user_id"
	TraceIDKey contextKey = "trace_id"
)

// CustomHandler wraps TextHandler to format source as "file:line"
type CustomHandler struct {
	handler slog.Handler
}

func NewCustomHandler(w io.Writer, opts *slog.HandlerOptions) *CustomHandler {
	return &CustomHandler{
		handler: slog.NewTextHandler(w, opts),
	}
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return &CustomHandler{handler: h.handler.WithGroup(name)}
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	// Replace the default source attribute with our custom format
	if r.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		if frame.File != "" {
			// Get project-relative path instead of full system path
			relativePath := getProjectRelativePath(frame.File)
			r.AddAttrs(slog.String("source", relativePath+":"+strconv.Itoa(frame.Line)))
		}
	}

	return h.handler.Handle(ctx, r)
}

// getProjectRelativePath converts absolute path to project-relative path
func getProjectRelativePath(fullPath string) string {
	// Look for common project root indicators for hub
	projectRootMarkers := []string{
		"/actlabs-hub/",
		"/actlabs/hub/",
		"/hub/",
	}

	for _, marker := range projectRootMarkers {
		if idx := strings.LastIndex(fullPath, marker); idx != -1 {
			// Return path relative to project root
			return fullPath[idx+len(marker):]
		}
	}

	// Fallback: if no project marker found, try to get relative to current working directory
	if wd, err := os.Getwd(); err == nil {
		if relPath, err := filepath.Rel(wd, fullPath); err == nil {
			return relPath
		}
	}

	// Final fallback: just return the filename
	return filepath.Base(fullPath)
}

// SetupLogger initializes the global logger with custom formatting
//
// ACTLABS_HUB_LOG_LEVEL environment variable supports both formats:
// String: DEBUG, INFO, WARN, ERROR (recommended)
// Number: -4 (debug), 0 (info), 4 (warn), 8 (error)
func SetupLogger(ctx context.Context) {
	logLevel := os.Getenv("ACTLABS_HUB_LOG_LEVEL")
	if logLevel == "" {
		slog.Info("ACTLABS_HUB_LOG_LEVEL not set defaulting to INFO")
		logLevel = "INFO" // Default to INFO
	}

	var level slog.Level

	// First try string format (case-insensitive)
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		// Fallback to numeric format for backward compatibility
		logLevelInt, err := strconv.Atoi(logLevel)
		if err != nil {
			slog.Error("Invalid ACTLABS_HUB_LOG_LEVEL value", "value", logLevel, "error", err.Error())
			level = slog.LevelInfo // Default to INFO on error
		} else {
			// Convert numeric to slog.Level
			switch {
			case logLevelInt <= -4: // Debug and below
				level = slog.LevelDebug
			case logLevelInt <= 0: // Info level
				level = slog.LevelInfo
			case logLevelInt <= 4: // Warn level
				level = slog.LevelWarn
			default: // Error and above
				level = slog.LevelError
			}
		}
	}

	slog.Info("Setting up logger",
		"log_level_env", logLevel,
		"resolved_level", level.String())

	opts := &slog.HandlerOptions{
		AddSource: false, // We handle source ourselves
		Level:     level,
	}

	// Use our custom handler for simplified source format
	customHandler := NewCustomHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(customHandler))
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// ContextLog creates a structured logger with context fields
func ContextLog(ctx context.Context) *slog.Logger {
	logger := slog.Default()

	if traceID := GetTraceID(ctx); traceID != "" {
		logger = logger.With("trace_id", traceID)
	}

	if userID := GetUserID(ctx); userID != "" {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// Convenience logging functions that automatically include context and correct source location
func LogInfo(ctx context.Context, msg string, args ...any) {
	logger := ContextLog(ctx)
	if !logger.Enabled(ctx, slog.LevelInfo) {
		return
	}

	var pc uintptr
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, LogInfo]
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, pc)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}

func LogError(ctx context.Context, msg string, args ...any) {
	logger := ContextLog(ctx)
	if !logger.Enabled(ctx, slog.LevelError) {
		return
	}

	var pc uintptr
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, LogError]
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), slog.LevelError, msg, pc)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}

func LogDebug(ctx context.Context, msg string, args ...any) {
	logger := ContextLog(ctx)
	if !logger.Enabled(ctx, slog.LevelDebug) {
		return
	}

	var pc uintptr
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, LogDebug]
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), slog.LevelDebug, msg, pc)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}

func LogWarning(ctx context.Context, msg string, args ...any) {
	logger := ContextLog(ctx)
	if !logger.Enabled(ctx, slog.LevelWarn) {
		return
	}

	var pc uintptr
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, LogWarn]
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), slog.LevelWarn, msg, pc)
	r.Add(args...)
	_ = logger.Handler().Handle(ctx, r)
}

// CreateBackgroundContextWithValues creates a background context preserving trace and user IDs
func CreateBackgroundContextWithValues(sourceCtx context.Context) context.Context {
	bgCtx := context.Background()

	if traceID := GetTraceID(sourceCtx); traceID != "" {
		bgCtx = context.WithValue(bgCtx, TraceIDKey, traceID)
	}
	if userID := GetUserID(sourceCtx); userID != "" {
		bgCtx = WithUserID(bgCtx, userID)
	}

	return bgCtx
}
