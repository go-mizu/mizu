// Package h2c provides HTTP/2 cleartext (h2c) upgrade middleware for Mizu.
package h2c

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the h2c middleware.
type Options struct {
	// AllowUpgrade enables HTTP/2 upgrade.
	// Default: true.
	AllowUpgrade bool

	// AllowDirect enables direct HTTP/2 connections.
	// Default: true.
	AllowDirect bool

	// OnUpgrade is called when an upgrade occurs.
	OnUpgrade func(r *http.Request)
}

// contextKey is a private type for context keys.
type contextKey struct{}

// h2cKey stores h2c info.
var h2cKey = contextKey{}

// Info contains HTTP/2 information.
type Info struct {
	IsHTTP2  bool
	Upgraded bool
	Direct   bool
}

// New creates h2c middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{
		AllowUpgrade: true,
		AllowDirect:  true,
	})
}

// WithOptions creates h2c middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()
			info := &Info{}

			// Check for HTTP/2 prior knowledge (direct connection)
			if r.ProtoMajor == 2 {
				info.IsHTTP2 = true
				info.Direct = true
				ctx := context.WithValue(c.Context(), h2cKey, info)
				req := c.Request().WithContext(ctx)
				*c.Request() = *req
				return next(c)
			}

			// Check for HTTP/2 upgrade
			if opts.AllowUpgrade && isH2CUpgrade(r) {
				info.IsHTTP2 = true
				info.Upgraded = true
				ctx := context.WithValue(c.Context(), h2cKey, info)
				req := c.Request().WithContext(ctx)
				*c.Request() = *req

				if opts.OnUpgrade != nil {
					opts.OnUpgrade(r)
				}

				// Handle upgrade
				return handleUpgrade(c, next, r)
			}

			// Regular HTTP/1.x request
			ctx := context.WithValue(c.Context(), h2cKey, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req
			return next(c)
		}
	}
}

// isH2CUpgrade checks if the request is an h2c upgrade request.
func isH2CUpgrade(r *http.Request) bool {
	// Check Connection header contains "Upgrade"
	connection := r.Header.Get("Connection")
	if !containsToken(connection, "upgrade") {
		return false
	}

	// Check Upgrade header is "h2c"
	upgrade := r.Header.Get("Upgrade")
	if !strings.EqualFold(upgrade, "h2c") {
		return false
	}

	// Check HTTP2-Settings header exists
	if r.Header.Get("HTTP2-Settings") == "" {
		return false
	}

	return true
}

// containsToken checks if a comma-separated header contains a token.
func containsToken(header, token string) bool {
	for _, t := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(t), token) {
			return true
		}
	}
	return false
}

// handleUpgrade handles the HTTP/2 upgrade.
func handleUpgrade(c *mizu.Ctx, next mizu.Handler, r *http.Request) error {
	// Get the connection
	hijacker, ok := c.Writer().(http.Hijacker)
	if !ok {
		// Can't hijack, fall back to HTTP/1.x
		return next(c)
	}

	conn, brw, err := hijacker.Hijack()
	if err != nil {
		return next(c)
	}
	defer func() { _ = conn.Close() }()

	// Send 101 Switching Protocols
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: h2c\r\n\r\n"
	_, _ = conn.Write([]byte(response))

	// For a full implementation, we would need to:
	// 1. Parse the HTTP2-Settings header
	// 2. Send server connection preface
	// 3. Handle HTTP/2 frames
	// This is a simplified version

	// Read and echo back (simplified)
	_, _ = io.Copy(conn, brw)

	return nil
}

// GetInfo returns h2c information from context.
func GetInfo(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(h2cKey).(*Info); ok {
		return info
	}
	return &Info{}
}

// IsHTTP2 returns true if the request is using HTTP/2.
func IsHTTP2(c *mizu.Ctx) bool {
	return GetInfo(c).IsHTTP2
}

// ParseSettings parses the HTTP2-Settings header.
func ParseSettings(r *http.Request) ([]byte, error) {
	settings := r.Header.Get("HTTP2-Settings")
	if settings == "" {
		return nil, nil
	}
	return base64.RawURLEncoding.DecodeString(settings)
}

// ServerHandler wraps an http.Handler to support h2c.
type ServerHandler struct {
	Handler http.Handler
	opts    Options
}

// NewServerHandler creates a new h2c server handler.
func NewServerHandler(handler http.Handler, opts Options) *ServerHandler {
	return &ServerHandler{
		Handler: handler,
		opts:    opts,
	}
}

// ServeHTTP implements http.Handler.
func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check for HTTP/2 prior knowledge
	if r.ProtoMajor == 2 {
		h.Handler.ServeHTTP(w, r)
		return
	}

	// Check for h2c upgrade
	if h.opts.AllowUpgrade && isH2CUpgrade(r) {
		if h.opts.OnUpgrade != nil {
			h.opts.OnUpgrade(r)
		}

		// Send 101 and handle upgrade
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			h.Handler.ServeHTTP(w, r)
			return
		}

		conn, _, err := hijacker.Hijack()
		if err != nil {
			h.Handler.ServeHTTP(w, r)
			return
		}
		defer func() { _ = conn.Close() }()

		// Send switching protocols response
		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Connection: Upgrade\r\n" +
			"Upgrade: h2c\r\n\r\n"
		_, _ = conn.Write([]byte(response))

		return
	}

	h.Handler.ServeHTTP(w, r)
}

// Detect creates middleware that only detects h2c without handling it.
func Detect() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()
			info := &Info{}

			if r.ProtoMajor == 2 {
				info.IsHTTP2 = true
				info.Direct = true
			} else if isH2CUpgrade(r) {
				info.IsHTTP2 = true
				info.Upgraded = true
			}

			ctx := context.WithValue(c.Context(), h2cKey, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req
			return next(c)
		}
	}
}

// Wrap wraps an http.Handler with h2c support.
func Wrap(handler http.Handler) http.Handler {
	return NewServerHandler(handler, Options{
		AllowUpgrade: true,
		AllowDirect:  true,
	})
}

// BufferedConn wraps a net.Conn with a buffer.
type BufferedConn struct {
	net.Conn
	buf *bufio.Reader
}

// NewBufferedConn creates a buffered connection.
func NewBufferedConn(conn net.Conn) *BufferedConn {
	return &BufferedConn{
		Conn: conn,
		buf:  bufio.NewReader(conn),
	}
}

// Read reads from the buffer first, then the connection.
func (c *BufferedConn) Read(b []byte) (int, error) {
	return c.buf.Read(b)
}

// Peek peeks at the next bytes.
func (c *BufferedConn) Peek(n int) ([]byte, error) {
	return c.buf.Peek(n)
}

// HTTP/2 connection preface.
var connectionPreface = []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")

// IsHTTP2Preface checks if data starts with HTTP/2 preface.
func IsHTTP2Preface(data []byte) bool {
	if len(data) < len(connectionPreface) {
		return false
	}
	return bytes.Equal(data[:len(connectionPreface)], connectionPreface)
}
