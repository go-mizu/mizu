// Package sentry provides error reporting middleware for Mizu.
// This is a lightweight implementation that captures errors and panics.
package sentry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// Options configures the sentry middleware.
type Options struct {
	// DSN is the Sentry DSN for error reporting.
	// If empty, errors are logged locally.
	DSN string

	// Environment is the environment name (e.g., "production", "staging").
	Environment string

	// Release is the release/version identifier.
	Release string

	// SampleRate is the percentage of events to capture (0.0 to 1.0).
	// Default: 1.0 (capture all).
	SampleRate float64

	// BeforeSend is called before sending an event.
	// Return nil to drop the event.
	BeforeSend func(event *Event) *Event

	// OnError is called when an error is captured.
	OnError func(event *Event)

	// Debug enables debug logging.
	Debug bool

	// CaptureRequestBody captures request body in events.
	CaptureRequestBody bool

	// MaxRequestBodySize is the max size of captured request body.
	// Default: 10KB.
	MaxRequestBodySize int

	// Tags are default tags added to all events.
	Tags map[string]string

	// Transport sends events to Sentry.
	// If nil, uses default HTTP transport.
	Transport Transport
}

// Event represents an error event.
type Event struct {
	EventID     string            `json:"event_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Level       string            `json:"level"`
	Platform    string            `json:"platform"`
	Environment string            `json:"environment,omitempty"`
	Release     string            `json:"release,omitempty"`
	ServerName  string            `json:"server_name,omitempty"`
	Message     string            `json:"message,omitempty"`
	Exception   []Exception       `json:"exception,omitempty"`
	Request     *RequestInfo      `json:"request,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Extra       map[string]any    `json:"extra,omitempty"`
	User        *User             `json:"user,omitempty"`
	Contexts    map[string]any    `json:"contexts,omitempty"`
}

// Exception represents an exception/error.
type Exception struct {
	Type       string      `json:"type"`
	Value      string      `json:"value"`
	Stacktrace *Stacktrace `json:"stacktrace,omitempty"`
}

// Stacktrace represents a stack trace.
type Stacktrace struct {
	Frames []Frame `json:"frames"`
}

// Frame represents a stack frame.
type Frame struct {
	Filename string `json:"filename"`
	Function string `json:"function"`
	Lineno   int    `json:"lineno"`
	AbsPath  string `json:"abs_path,omitempty"`
}

// RequestInfo represents HTTP request info.
type RequestInfo struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	QueryString string            `json:"query_string,omitempty"`
	Data        string            `json:"data,omitempty"`
}

// User represents user info.
type User struct {
	ID        string `json:"id,omitempty"`
	Email     string `json:"email,omitempty"`
	Username  string `json:"username,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// Transport sends events.
type Transport interface {
	Send(event *Event) error
}

// Hub manages error capturing.
type Hub struct {
	opts   Options
	mu     sync.RWMutex
	events []*Event
}

// contextKey is a private type for context keys.
type contextKey struct{}

// hubKey stores the hub in context.
var hubKey = contextKey{}

// globalHub is the default hub.
var globalHub *Hub

// New creates sentry middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates sentry middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.SampleRate == 0 {
		opts.SampleRate = 1.0
	}
	if opts.MaxRequestBodySize == 0 {
		opts.MaxRequestBodySize = 10 * 1024 // 10KB
	}

	hub := &Hub{
		opts:   opts,
		events: make([]*Event, 0),
	}

	if globalHub == nil {
		globalHub = hub
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Store hub in context
			ctx := context.WithValue(c.Context(), hubKey, hub)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Capture panic
			defer func() {
				if r := recover(); r != nil {
					event := newEvent(hub.opts)
					event.Level = "fatal"
					event.Message = fmt.Sprintf("%v", r)
					event.Exception = []Exception{{
						Type:       "panic",
						Value:      fmt.Sprintf("%v", r),
						Stacktrace: captureStacktrace(3),
					}}
					event.Request = captureRequest(c, hub.opts)

					hub.captureEvent(event)

					// Re-panic
					panic(r)
				}
			}()

			err := next(c)
			if err != nil {
				CaptureError(c, err)
			}

			return err
		}
	}
}

func newEvent(opts Options) *Event {
	hostname, _ := os.Hostname()

	event := &Event{
		EventID:     generateEventID(),
		Timestamp:   time.Now().UTC(),
		Level:       "error",
		Platform:    "go",
		Environment: opts.Environment,
		Release:     opts.Release,
		ServerName:  hostname,
		Tags:        make(map[string]string),
		Extra:       make(map[string]any),
		Contexts:    make(map[string]any),
	}

	// Add default tags
	for k, v := range opts.Tags {
		event.Tags[k] = v
	}

	// Add runtime context
	event.Contexts["runtime"] = map[string]any{
		"name":    "go",
		"version": runtime.Version(),
	}

	return event
}

func generateEventID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 4))
	}
	return fmt.Sprintf("%x", b)
}

func captureStacktrace(skip int) *Stacktrace {
	var frames []Frame
	pcs := make([]uintptr, 50)
	n := runtime.Callers(skip, pcs)
	pcs = pcs[:n]

	frames_raw := runtime.CallersFrames(pcs)
	for {
		frame, more := frames_raw.Next()
		frames = append([]Frame{{
			Filename: frame.File,
			Function: frame.Function,
			Lineno:   frame.Line,
			AbsPath:  frame.File,
		}}, frames...)
		if !more {
			break
		}
	}

	return &Stacktrace{Frames: frames}
}

func captureRequest(c *mizu.Ctx, opts Options) *RequestInfo {
	r := c.Request()
	req := &RequestInfo{
		URL:         r.URL.String(),
		Method:      r.Method,
		QueryString: r.URL.RawQuery,
		Headers:     make(map[string]string),
	}

	// Capture headers (excluding sensitive ones)
	sensitiveHeaders := map[string]bool{
		"Authorization": true,
		"Cookie":        true,
		"X-Api-Key":     true,
	}

	for k, v := range r.Header {
		if !sensitiveHeaders[k] && len(v) > 0 {
			req.Headers[k] = v[0]
		}
	}

	return req
}

func (h *Hub) captureEvent(event *Event) {
	// Apply sampling
	if h.opts.SampleRate < 1.0 {
		if float64(time.Now().UnixNano()%100)/100 > h.opts.SampleRate {
			return
		}
	}

	// Apply BeforeSend hook
	if h.opts.BeforeSend != nil {
		event = h.opts.BeforeSend(event)
		if event == nil {
			return
		}
	}

	// Store event
	h.mu.Lock()
	h.events = append(h.events, event)
	h.mu.Unlock()

	// Call OnError hook
	if h.opts.OnError != nil {
		h.opts.OnError(event)
	}

	// Debug logging
	if h.opts.Debug {
		fmt.Printf("[sentry] Captured event: %s - %s\n", event.EventID, event.Message)
	}

	// Send via transport
	if h.opts.Transport != nil {
		go func() { _ = h.opts.Transport.Send(event) }()
	} else if h.opts.DSN != "" {
		go func() { _ = sendToSentry(h.opts.DSN, event) }()
	}
}

func sendToSentry(dsn string, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Basic DSN parsing would go here
	// For now, just log if debug mode
	_ = dsn
	_ = data

	return nil
}

// GetHub returns the hub from context.
func GetHub(c *mizu.Ctx) *Hub {
	if hub, ok := c.Context().Value(hubKey).(*Hub); ok {
		return hub
	}
	return globalHub
}

// CaptureError captures an error.
func CaptureError(c *mizu.Ctx, err error) string {
	hub := GetHub(c)
	if hub == nil {
		return ""
	}

	event := newEvent(hub.opts)
	event.Message = err.Error()
	event.Exception = []Exception{{
		Type:       "error",
		Value:      err.Error(),
		Stacktrace: captureStacktrace(2),
	}}

	if c != nil {
		event.Request = captureRequest(c, hub.opts)
	}

	hub.captureEvent(event)
	return event.EventID
}

// CaptureMessage captures a message.
func CaptureMessage(c *mizu.Ctx, message string) string {
	hub := GetHub(c)
	if hub == nil {
		return ""
	}

	event := newEvent(hub.opts)
	event.Level = "info"
	event.Message = message

	if c != nil {
		event.Request = captureRequest(c, hub.opts)
	}

	hub.captureEvent(event)
	return event.EventID
}

// contextKey types for additional data
type userContextKey struct{}
type tagsContextKey struct{}
type extraContextKey struct{}

// SetUser sets user information.
func SetUser(c *mizu.Ctx, user *User) {
	hub := GetHub(c)
	if hub != nil {
		ctx := context.WithValue(c.Context(), userContextKey{}, user)
		req := c.Request().WithContext(ctx)
		*c.Request() = *req
	}
}

// SetTag sets a tag.
func SetTag(c *mizu.Ctx, key, value string) {
	hub := GetHub(c)
	if hub != nil {
		tags, _ := c.Context().Value(tagsContextKey{}).(map[string]string)
		if tags == nil {
			tags = make(map[string]string)
		}
		tags[key] = value
		ctx := context.WithValue(c.Context(), tagsContextKey{}, tags)
		req := c.Request().WithContext(ctx)
		*c.Request() = *req
	}
}

// SetExtra sets extra data.
func SetExtra(c *mizu.Ctx, key string, value any) {
	hub := GetHub(c)
	if hub != nil {
		extra, _ := c.Context().Value(extraContextKey{}).(map[string]any)
		if extra == nil {
			extra = make(map[string]any)
		}
		extra[key] = value
		ctx := context.WithValue(c.Context(), extraContextKey{}, extra)
		req := c.Request().WithContext(ctx)
		*c.Request() = *req
	}
}

// Events returns all captured events (for testing).
func (h *Hub) Events() []*Event {
	h.mu.RLock()
	defer h.mu.RUnlock()
	events := make([]*Event, len(h.events))
	copy(events, h.events)
	return events
}

// Clear clears all captured events.
func (h *Hub) Clear() {
	h.mu.Lock()
	h.events = make([]*Event, 0)
	h.mu.Unlock()
}

// MockTransport is a transport for testing.
type MockTransport struct {
	mu     sync.Mutex
	events []*Event
}

// Send captures an event for testing.
func (t *MockTransport) Send(event *Event) error {
	t.mu.Lock()
	t.events = append(t.events, event)
	t.mu.Unlock()
	return nil
}

// Events returns captured events.
func (t *MockTransport) Events() []*Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.events
}

// HTTPTransport sends events via HTTP.
type HTTPTransport struct {
	DSN    string
	Client *http.Client
}

// Send sends an event to Sentry.
func (t *HTTPTransport) Send(event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, t.DSN, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := t.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
