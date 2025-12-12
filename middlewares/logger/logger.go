// Package logger provides request logging middleware for Mizu.
package logger

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the logger middleware.
type Options struct {
	// Output is the writer for log output.
	// Default: os.Stdout.
	Output io.Writer

	// Format is the log format string.
	// Available tags: ${time}, ${status}, ${method}, ${path}, ${latency},
	// ${ip}, ${host}, ${protocol}, ${referer}, ${user_agent}, ${bytes_in},
	// ${bytes_out}, ${query}, ${header:name}, ${form:name}.
	// Default: "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n".
	Format string

	// TimeFormat is the time format.
	// Default: "2006/01/02 - 15:04:05".
	TimeFormat string

	// Skip is a function to skip logging for specific requests.
	Skip func(c *mizu.Ctx) bool
}

// New creates logger middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates logger middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.Format == "" {
		opts.Format = "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n"
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = "2006/01/02 - 15:04:05"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if opts.Skip != nil && opts.Skip(c) {
				return next(c)
			}

			start := time.Now()

			// Wrap writer to capture response size
			rw := &responseWriter{ResponseWriter: c.Writer(), status: 200}
			c.SetWriter(rw)

			err := next(c)

			latency := time.Since(start)
			status := rw.status
			if err != nil {
				status = 500
			}

			log := formatLog(opts.Format, c, status, latency, rw.size, opts.TimeFormat)
			_, _ = opts.Output.Write([]byte(log))

			return err
		}
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func formatLog(format string, c *mizu.Ctx, status int, latency time.Duration, size int, timeFormat string) string {
	r := c.Request()
	result := format

	// Replace tags
	result = strings.ReplaceAll(result, "${time}", time.Now().Format(timeFormat))
	result = strings.ReplaceAll(result, "${status}", strconv.Itoa(status))
	result = strings.ReplaceAll(result, "${method}", r.Method)
	result = strings.ReplaceAll(result, "${path}", r.URL.Path)
	result = strings.ReplaceAll(result, "${latency}", latency.String())
	result = strings.ReplaceAll(result, "${ip}", clientIP(c))
	result = strings.ReplaceAll(result, "${host}", r.Host)
	result = strings.ReplaceAll(result, "${protocol}", r.Proto)
	result = strings.ReplaceAll(result, "${referer}", r.Referer())
	result = strings.ReplaceAll(result, "${user_agent}", r.UserAgent())
	result = strings.ReplaceAll(result, "${bytes_out}", strconv.Itoa(size))
	result = strings.ReplaceAll(result, "${query}", r.URL.RawQuery)

	// Handle bytes_in
	if r.ContentLength > 0 {
		result = strings.ReplaceAll(result, "${bytes_in}", strconv.FormatInt(r.ContentLength, 10))
	} else {
		result = strings.ReplaceAll(result, "${bytes_in}", "0")
	}

	// Handle ${header:name}
	for {
		start := strings.Index(result, "${header:")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		tag := result[start : start+end+1]
		name := result[start+9 : start+end]
		result = strings.ReplaceAll(result, tag, r.Header.Get(name))
	}

	// Handle ${form:name}
	for {
		start := strings.Index(result, "${form:")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			break
		}
		tag := result[start : start+end+1]
		name := result[start+7 : start+end]
		result = strings.ReplaceAll(result, tag, r.FormValue(name))
	}

	return result
}

func clientIP(c *mizu.Ctx) string {
	// Check X-Forwarded-For
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		if comma := strings.Index(xff, ","); comma > 0 {
			return strings.TrimSpace(xff[:comma])
		}
		return xff
	}

	// Check X-Real-IP
	if xri := c.Request().Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	addr := c.Request().RemoteAddr
	if colon := strings.LastIndex(addr, ":"); colon > 0 {
		return addr[:colon]
	}
	return addr
}
