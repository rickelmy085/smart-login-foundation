package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RequestIDKey is the context key for request ID.
type requestIDKey string

const RequestIDKey requestIDKey = "request_id"

// RequestIDMiddleware extracts or generates a request ID and adds it to context and logs.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to response header
		w.Header().Set("X-Request-ID", requestID)

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add to slog context
		ctx = WithRequestID(ctx, requestID)

		// Log request start
		slog.Info("request started",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))

		// Log request completion
		duration := time.Since(start)
		slog.Info("request completed",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// WithRequestID adds request ID to context for logging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// LogAttrs returns slog attributes with request ID from context.
func LogAttrs(ctx context.Context, args ...any) []any {
	requestID := GetRequestID(ctx)
	if requestID == "" {
		return args
	}
	return append(args, "request_id", requestID)
}

// LoggingMiddleware provides structured logging with request ID and records metrics.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			requestID = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", requestID)

		ctx := WithRequestID(r.Context(), requestID)

		start := time.Now()
		slog.Info("request started", LogAttrs(ctx, "method", r.Method, "path", r.URL.Path)...)

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r.WithContext(ctx))

		duration := time.Since(start)
		slog.Info("request completed",
			LogAttrs(ctx, "method", r.Method, "path", r.URL.Path, "duration_ms", duration.Milliseconds(), "status", wrapped.statusCode)...,
		)

		// Record metrics based on path
		path := r.URL.Path
		switch {
		case strings.Contains(path, "/api/chat"):
			if wrapped.statusCode >= 400 {
				GlobalMetrics.RecordChatError()
			}
			GlobalMetrics.RecordChatRequest(duration)
		case strings.Contains(path, "/api/tasks"):
			if strings.HasSuffix(path, "/process") || strings.HasSuffix(path, "/data") || 
			   strings.HasSuffix(path, "/validate") || strings.HasSuffix(path, "/generate") {
				if wrapped.statusCode >= 400 {
					GlobalMetrics.RecordWorkflowFailed()
				}
			}
		}
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}