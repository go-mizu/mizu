package middleware

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/google/uuid"
)

// responseWriterWrapper wraps http.ResponseWriter to capture status code.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriterWrapper(w http.ResponseWriter) *responseWriterWrapper {
	return &responseWriterWrapper{ResponseWriter: w, statusCode: 200}
}

func (w *responseWriterWrapper) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker for WebSocket support.
func (w *responseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("hijack not supported")
}

// Flush implements http.Flusher.
func (w *responseWriterWrapper) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// LoggingConfig configures the logging middleware.
type LoggingConfig struct {
	// LogsStore is the store for writing logs.
	LogsStore store.LogsStore

	// IgnorePaths lists paths to exclude from logging.
	IgnorePaths []string

	// IgnorePathPrefixes lists path prefixes to exclude from logging.
	IgnorePathPrefixes []string
}

// DefaultLoggingConfig returns default logging configuration.
func DefaultLoggingConfig(logsStore store.LogsStore) *LoggingConfig {
	return &LoggingConfig{
		LogsStore: logsStore,
		IgnorePaths: []string{
			"/health",
		},
		IgnorePathPrefixes: []string{
			"/api/logs", // Avoid logging log queries
		},
	}
}

// Logging returns a middleware that logs all HTTP requests.
func Logging(config *LoggingConfig) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			start := time.Now()
			requestID := uuid.New().String()

			// Check if path should be ignored
			path := c.Request().URL.Path
			for _, ignorePath := range config.IgnorePaths {
				if path == ignorePath {
					return next(c)
				}
			}
			for _, prefix := range config.IgnorePathPrefixes {
				if strings.HasPrefix(path, prefix) {
					return next(c)
				}
			}

			// Capture request info before handler
			method := c.Request().Method
			userAgent := c.Request().Header.Get("User-Agent")
			apiKey := c.Request().Header.Get("apikey")
			if apiKey == "" {
				apiKey = extractBearerToken(c.Request().Header.Get("Authorization"))
			}

			// Get user ID from context if available (set by auth middleware)
			var userID *string
			if uid := c.Request().Header.Get("X-Localbase-User-ID"); uid != "" {
				userID = &uid
			}

			// Capture request headers (selective)
			requestHeaders := make(map[string]string)
			for _, header := range []string{
				"Accept", "Content-Type", "Content-Length",
				"User-Agent", "Referer", "Origin",
				"X-Forwarded-For", "X-Real-IP",
			} {
				if val := c.Request().Header.Get(header); val != "" {
					requestHeaders[header] = val
				}
			}

			// Wrap the response writer to capture status code
			wrapped := newResponseWriterWrapper(c.Writer())

			// Create a new request with the wrapped response writer
			// Note: We use the wrapper to capture the status code
			originalWriter := c.Writer()
			c.SetWriter(wrapped)

			// Call the handler
			err := next(c)

			// Restore original writer
			c.SetWriter(originalWriter)

			// Capture response info after handler
			duration := time.Since(start)
			statusCode := wrapped.statusCode
			if statusCode == 0 {
				statusCode = 200 // Default if not set
			}
			// If there was an error and no explicit status was written, assume 500
			if err != nil && !wrapped.written {
				statusCode = 500
			}

			// Determine source based on path
			source := determineSource(path)

			// Determine severity based on status code
			severity := determineSeverity(statusCode, err)

			// Mask API key for display
			maskedAPIKey := maskAPIKey(apiKey)

			// Build event message
			eventMessage := fmt.Sprintf("%s | %d | %s | %s",
				method, statusCode, path, c.Request().RemoteAddr)

			// Create log entry asynchronously
			go func() {
				entry := &store.LogEntry{
					Timestamp:      start,
					EventMessage:   eventMessage,
					RequestID:      &requestID,
					Method:         method,
					Path:           path,
					StatusCode:     statusCode,
					Source:         source,
					Severity:       severity,
					UserID:         userID,
					UserAgent:      userAgent,
					APIKey:         maskedAPIKey,
					RequestHeaders: requestHeaders,
					DurationMs:     int(duration.Milliseconds()),
					Metadata:       buildMetadata(c, err),
				}
				config.LogsStore.CreateLog(context.Background(), entry)
			}()

			return err
		}
	}
}

// determineSource determines the log source based on the request path.
func determineSource(path string) string {
	switch {
	case strings.HasPrefix(path, "/auth/"):
		return "auth"
	case strings.HasPrefix(path, "/storage/"):
		return "storage"
	case strings.HasPrefix(path, "/rest/"):
		return "postgrest"
	case strings.HasPrefix(path, "/functions/"):
		return "functions"
	case strings.HasPrefix(path, "/realtime/"):
		return "realtime"
	default:
		return "edge"
	}
}

// determineSeverity determines the log severity based on status code and error.
func determineSeverity(statusCode int, err error) string {
	// If there's an error, it's at least WARNING
	if err != nil {
		if statusCode >= 500 {
			return "ERROR"
		}
		return "WARNING"
	}

	// Determine by status code
	switch {
	case statusCode >= 500:
		return "ERROR"
	case statusCode >= 400:
		return "WARNING"
	case statusCode >= 300:
		return "NOTICE"
	default:
		return "INFO"
	}
}

// maskAPIKey masks an API key for logging (shows first 10 chars + "...")
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 15 {
		return key[:len(key)/2] + "... <masked>"
	}
	return key[:15] + "... <masked>"
}

// extractBearerToken extracts a bearer token from Authorization header.
func extractBearerToken(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

// buildMetadata builds additional metadata for the log entry.
func buildMetadata(c *mizu.Ctx, err error) map[string]any {
	metadata := make(map[string]any)

	// Add query parameters if present
	if query := c.Request().URL.RawQuery; query != "" {
		metadata["query"] = query
	}

	// Add role if set
	if role := c.Request().Header.Get("X-Localbase-Role"); role != "" {
		metadata["role"] = role
	}

	// Add error if present
	if err != nil {
		metadata["error"] = err.Error()
	}

	return metadata
}
