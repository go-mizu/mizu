package live

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/go-mizu/mizu"
)

// TestSession creates a test session with zero-value state.
func NewTestSession[T any]() *Session[T] {
	return &Session[T]{
		ID:       "test-session-id",
		dirty:    newDirtySet(),
		regions:  make(map[string]string),
		created:  time.Now(),
		lastSeen: time.Now(),
	}
}

// NewTestCtx creates a test live context.
func NewTestCtx() *Ctx {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	mc := mizu.CtxFromRequest(w, req)

	return &Ctx{
		Ctx:       mc,
		SessionID: "test-session-id",
	}
}

// TestClient provides a test client for live pages.
type TestClient struct {
	t       TestingT
	server  *httptest.Server
	conn    *wsConn
	mu      sync.Mutex
	patches []PatchPayload
	errors  []ErrorPayload
}

// TestingT is a subset of testing.T used for assertions.
type TestingT interface {
	Error(args ...any)
	Errorf(format string, args ...any)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Helper()
}

// NewTestClient creates a test client for the given app.
func NewTestClient(t TestingT, app *mizu.App) *TestClient {
	server := httptest.NewServer(app)

	return &TestClient{
		t:       t,
		server:  server,
		patches: make([]PatchPayload, 0),
		errors:  make([]ErrorPayload, 0),
	}
}

// Close shuts down the test client.
func (c *TestClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.server.Close()
}

// Connect establishes a WebSocket connection to a live page.
func (c *TestClient) Connect(path string) *TestConn {
	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(c.server.URL, "http") + "/_live/websocket"

	conn, err := dialWebSocket(wsURL, c.server.URL)
	if err != nil {
		c.t.Fatal("Failed to connect:", err)
	}

	c.conn = conn

	tc := &TestConn{
		client:   c,
		conn:     conn,
		path:     path,
		received: make(chan *Message, 100),
		done:     make(chan struct{}),
	}

	// Start receive goroutine
	go tc.receiveLoop()

	// Send JOIN
	tc.sendJoin(path)

	// Wait for initial reply
	tc.waitForReply()

	return tc
}

// TestConn represents a test WebSocket connection.
type TestConn struct {
	client    *TestClient
	conn      *wsConn
	path      string
	sessionID string
	html      string
	received  chan *Message
	done      chan struct{}
	mu        sync.Mutex
}

func (c *TestConn) receiveLoop() {
	for {
		data, err := c.conn.ReadMessage()
		if err != nil {
			close(c.done)
			return
		}

		msg, err := decodeMessage([]byte(data))
		if err != nil {
			continue
		}

		select {
		case c.received <- msg:
		default:
		}
	}
}

func (c *TestConn) sendJoin(path string) {
	msg := Message{
		Type: MsgTypeJoin,
		Ref:  1,
	}

	payload := JoinPayload{
		URL:   path,
		Token: "",
	}

	payloadBytes, _ := json.Marshal(payload)
	msg.Payload = payloadBytes

	data, _ := json.Marshal(msg)
	c.conn.WriteMessage(string(data))
}

func (c *TestConn) waitForReply() {
	select {
	case msg := <-c.received:
		if msg.Type == MsgTypeReply {
			var reply ReplyPayload
			msg.parsePayload(&reply)
			c.sessionID = reply.SessionID

			// Build initial HTML
			var buf bytes.Buffer
			for id, html := range reply.Rendered {
				buf.WriteString("<div id=\"")
				buf.WriteString(id)
				buf.WriteString("\">")
				buf.WriteString(html)
				buf.WriteString("</div>")
			}
			c.html = buf.String()
		}
	case <-time.After(5 * time.Second):
		c.client.t.Fatal("Timeout waiting for reply")
	}
}

// Click simulates a click event.
func (c *TestConn) Click(event string, values ...map[string]string) {
	c.sendEvent(event, values...)
}

// Submit simulates a form submit.
func (c *TestConn) Submit(event string, form map[string][]string) {
	payload := eventPayload{
		Name: event,
		Form: form,
	}

	c.sendEventPayload(payload)
}

// Change simulates an input change.
func (c *TestConn) Change(event string, name, value string) {
	payload := eventPayload{
		Name: event,
		Form: map[string][]string{name: {value}},
	}

	c.sendEventPayload(payload)
}

func (c *TestConn) sendEvent(event string, values ...map[string]string) {
	payload := eventPayload{
		Name: event,
	}

	if len(values) > 0 {
		payload.Values = values[0]
	}

	c.sendEventPayload(payload)
}

func (c *TestConn) sendEventPayload(payload eventPayload) {
	payloadBytes, _ := json.Marshal(payload)

	msg := Message{
		Type:    MsgTypeEvent,
		Ref:     2,
		Payload: payloadBytes,
	}

	data, _ := json.Marshal(msg)
	c.conn.WriteMessage(string(data))
}

// Wait waits for patches to arrive.
func (c *TestConn) Wait() {
	c.WaitTimeout(2 * time.Second)
}

// WaitTimeout waits for patches with a timeout.
func (c *TestConn) WaitTimeout(timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case msg := <-c.received:
			c.handleMessage(msg)
			if msg.Type == MsgTypePatch {
				return
			}
		case <-timer.C:
			return
		}
	}
}

func (c *TestConn) handleMessage(msg *Message) {
	switch msg.Type {
	case MsgTypeReply:
		// Already handled
	case MsgTypePatch:
		var patch PatchPayload
		msg.parsePayload(&patch)

		c.mu.Lock()
		// Update HTML based on patches
		for _, region := range patch.Regions {
			// Simple replacement for tests
			c.html = strings.ReplaceAll(c.html,
				"<div id=\""+region.ID+"\">",
				"<div id=\""+region.ID+"\">"+region.HTML+"</div><!--replaced-->")
		}
		c.mu.Unlock()

		c.client.mu.Lock()
		c.client.patches = append(c.client.patches, patch)
		c.client.mu.Unlock()

	case MsgTypeError:
		var errPayload ErrorPayload
		msg.parsePayload(&errPayload)

		c.client.mu.Lock()
		c.client.errors = append(c.client.errors, errPayload)
		c.client.mu.Unlock()
	}
}

// HTML returns the current rendered HTML.
func (c *TestConn) HTML() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.html
}

// SessionID returns the session ID.
func (c *TestConn) SessionID() string {
	return c.sessionID
}

// Close closes the connection.
func (c *TestConn) Close() {
	c.conn.Close()
	<-c.done
}

// Patches returns all received patches.
func (c *TestClient) Patches() []PatchPayload {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.patches
}

// Errors returns all received errors.
func (c *TestClient) Errors() []ErrorPayload {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.errors
}

// ClearPatches clears recorded patches.
func (c *TestClient) ClearPatches() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.patches = nil
}
