package middleware

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	ContextRequestID = "request_id"
	ContextLogger    = "logger"
)

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

type requestIDKey struct{}
type loggerKey struct{}

// RequestID middleware generates or passes through X-Request-ID,
// injects it into context, and attaches a structured logger with requestID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = generateRequestID()
		}

		c.Header("X-Request-ID", rid)
		c.Set(ContextRequestID, rid)

		logger := slog.With("requestID", rid)
		c.Set(ContextLogger, logger)

		ctx := context.WithValue(c.Request.Context(), requestIDKey{}, rid)
		ctx = context.WithValue(ctx, loggerKey{}, logger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequestIDFromContext extracts requestID from gin.Context.
func RequestIDFromContext(c *gin.Context) string {
	if v, ok := c.Get(ContextRequestID); ok {
		if rid, ok := v.(string); ok {
			return rid
		}
	}
	return ""
}

// LoggerFromContext extracts the slog.Logger from gin.Context.
func LoggerFromContext(c *gin.Context) *slog.Logger {
	if v, ok := c.Get(ContextLogger); ok {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

// LoggerFromStdContext extracts the slog.Logger from context.Context.
func LoggerFromStdContext(ctx context.Context) *slog.Logger {
	if v := ctx.Value(loggerKey{}); v != nil {
		if logger, ok := v.(*slog.Logger); ok {
			return logger
		}
	}
	return slog.Default()
}

// RequestIDFromStdContext extracts requestID from context.Context.
func RequestIDFromStdContext(ctx context.Context) string {
	if v := ctx.Value(requestIDKey{}); v != nil {
		if rid, ok := v.(string); ok {
			return rid
		}
	}
	return ""
}

func generateRequestID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
