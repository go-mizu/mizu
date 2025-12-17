package live

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name: "valid upgrade",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "Upgrade",
			},
			expected: true,
		},
		{
			name: "case insensitive",
			headers: map[string]string{
				"Upgrade":    "WebSocket",
				"Connection": "upgrade",
			},
			expected: true,
		},
		{
			name: "connection with keep-alive",
			headers: map[string]string{
				"Upgrade":    "websocket",
				"Connection": "keep-alive, Upgrade",
			},
			expected: true,
		},
		{
			name: "missing upgrade",
			headers: map[string]string{
				"Connection": "Upgrade",
			},
			expected: false,
		},
		{
			name: "missing connection",
			headers: map[string]string{
				"Upgrade": "websocket",
			},
			expected: false,
		},
		{
			name:     "no headers",
			headers:  map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := isWebSocketUpgrade(req); got != tt.expected {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestComputeAcceptKey(t *testing.T) {
	// Test case from RFC 6455
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	result := computeAcceptKey(key)
	if result != expected {
		t.Errorf("computeAcceptKey(%q) = %q, want %q", key, result, expected)
	}
}

func TestHandleConn_NotWebSocket(t *testing.T) {
	srv := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleConn_ForbiddenOrigin(t *testing.T) {
	srv := New(Options{
		Origins: []string{"https://allowed.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://forbidden.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestHandleConn_AllowedOrigin(t *testing.T) {
	srv := New(Options{
		Origins: []string{"https://allowed.com"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://allowed.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	// Will fail to hijack in test, but shouldn't be forbidden
	if rec.Code == http.StatusForbidden {
		t.Error("should not be forbidden for allowed origin")
	}
}

func TestHandleConn_WildcardOrigin(t *testing.T) {
	srv := New(Options{
		Origins: []string{"*"},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://any.com")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Error("wildcard should allow any origin")
	}
}

func TestHandleConn_MissingKey(t *testing.T) {
	srv := New(Options{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	// Missing Sec-WebSocket-Key
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleConn_AuthFailed(t *testing.T) {
	srv := New(Options{
		OnAuth: func(ctx context.Context, r *http.Request) (Meta, error) {
			return nil, ErrAuthFailed
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

// mockConn is a mock net.Conn for testing
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
	mu       sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.EOF
	}
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestWsConn_WriteMessage(t *testing.T) {
	mock := newMockConn()
	ws := &wsConn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := ws.writeMessage(wsTextMessage, []byte("hello"))
	if err != nil {
		t.Errorf("writeMessage error: %v", err)
	}

	if mock.writeBuf.Len() == 0 {
		t.Error("expected data to be written")
	}
}

func TestWsConn_WriteMessageLengths(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short", 10},
		{"medium 126", 126},
		{"large 127", 200},
		{"very large", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockConn()
			ws := &wsConn{
				conn:   mock,
				reader: bufio.NewReader(mock.readBuf),
				writer: bufio.NewWriter(mock.writeBuf),
			}

			data := make([]byte, tt.length)
			err := ws.writeMessage(wsBinaryMessage, data)
			if err != nil {
				t.Errorf("writeMessage error: %v", err)
			}

			if mock.writeBuf.Len() == 0 {
				t.Error("expected data to be written")
			}
		})
	}
}

func TestWsConn_ReadMessage(t *testing.T) {
	tests := []struct {
		name     string
		frame    []byte
		wantType int
		wantData []byte
		wantErr  bool
	}{
		{
			name: "short unmasked text",
			frame: []byte{
				0x81, // FIN + text opcode
				0x05, // length 5
				'h', 'e', 'l', 'l', 'o',
			},
			wantType: wsTextMessage,
			wantData: []byte("hello"),
		},
		{
			name: "masked text",
			frame: []byte{
				0x81,                   // FIN + text opcode
				0x85,                   // masked + length 5
				0x37, 0xfa, 0x21, 0x3d, // mask key
				0x7f, 0x9f, 0x4d, 0x51, 0x58, // masked "Hello"
			},
			wantType: wsTextMessage,
			wantData: []byte("Hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockConn()
			mock.readBuf.Write(tt.frame)

			ws := &wsConn{
				conn:   mock,
				reader: bufio.NewReader(mock.readBuf),
				writer: bufio.NewWriter(mock.writeBuf),
			}

			msgType, data, err := ws.readMessage()
			if (err != nil) != tt.wantErr {
				t.Errorf("readMessage error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if msgType != tt.wantType {
					t.Errorf("got type %d, want %d", msgType, tt.wantType)
				}
				if !bytes.Equal(data, tt.wantData) {
					t.Errorf("got data %q, want %q", data, tt.wantData)
				}
			}
		})
	}
}

func TestWsConn_ReadMessageExtendedLength(t *testing.T) {
	mock := newMockConn()
	data := make([]byte, 200)
	for i := range data {
		data[i] = 'a'
	}
	frame := []byte{
		0x82,       // FIN + binary opcode
		0x7e,       // length 126 indicator
		0x00, 0xc8, // 200 in big endian
	}
	frame = append(frame, data...)
	mock.readBuf.Write(frame)

	ws := &wsConn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	msgType, received, err := ws.readMessage()
	if err != nil {
		t.Errorf("readMessage error: %v", err)
		return
	}
	if msgType != wsBinaryMessage {
		t.Errorf("got type %d, want %d", msgType, wsBinaryMessage)
	}
	if len(received) != 200 {
		t.Errorf("got length %d, want 200", len(received))
	}
}

func TestWsConn_ReadMessageErrors(t *testing.T) {
	// Test read error on first byte
	mock := newMockConn()
	ws := &wsConn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	_, _, err := ws.readMessage()
	if err == nil {
		t.Error("expected error on empty read")
	}

	// Test read error on second byte
	mock2 := newMockConn()
	mock2.readBuf.Write([]byte{0x81}) // Only first byte
	ws2 := &wsConn{
		conn:   mock2,
		reader: bufio.NewReader(mock2.readBuf),
		writer: bufio.NewWriter(mock2.writeBuf),
	}

	_, _, err = ws2.readMessage()
	if err == nil {
		t.Error("expected error on partial frame")
	}
}

func TestMessageConstants(t *testing.T) {
	if wsTextMessage != 1 {
		t.Errorf("wsTextMessage = %d, want 1", wsTextMessage)
	}
	if wsBinaryMessage != 2 {
		t.Errorf("wsBinaryMessage = %d, want 2", wsBinaryMessage)
	}
	if wsCloseMessage != 8 {
		t.Errorf("wsCloseMessage = %d, want 8", wsCloseMessage)
	}
	if wsPingMessage != 9 {
		t.Errorf("wsPingMessage = %d, want 9", wsPingMessage)
	}
	if wsPongMessage != 10 {
		t.Errorf("wsPongMessage = %d, want 10", wsPongMessage)
	}
}

// Integration test with a real HTTP test server
func TestServer_Integration(t *testing.T) {
	var mu sync.Mutex
	var receivedMessages []Message
	var closedSessions []*Session

	// Create server pointer first so we can reference it in callbacks
	var srv *Server

	srv = New(Options{
		QueueSize: 10,
		OnAuth: func(ctx context.Context, r *http.Request) (Meta, error) {
			token := r.Header.Get("Authorization")
			if token == "" {
				return nil, ErrAuthFailed
			}
			return Meta{"token": token}, nil
		},
		OnMessage: func(ctx context.Context, s *Session, msg Message) {
			mu.Lock()
			receivedMessages = append(receivedMessages, msg)
			mu.Unlock()

			// Handle subscribe
			if msg.Type == "subscribe" {
				srv.PubSub().Subscribe(s, msg.Topic)
				_ = s.Send(Message{Type: "ack", Topic: msg.Topic, Ref: msg.Ref})
			}
		},
		OnClose: func(s *Session, err error) {
			mu.Lock()
			closedSessions = append(closedSessions, s)
			mu.Unlock()
		},
	})

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Test 1: Auth required
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("HTTP GET error: %v", err)
	}
	resp.Body.Close()
	// Not a WebSocket request, should get bad request
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("non-WS request status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	// Test 2: WebSocket upgrade without auth
	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	// No Authorization header

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP request error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated WS request status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// Verify closedSessions is used
	_ = closedSessions
	_ = receivedMessages
}

// Test that server properly handles origin checking
func TestServer_Origins(t *testing.T) {
	tests := []struct {
		name           string
		serverOrigins  []string
		requestOrigin  string
		expectForbid   bool
	}{
		{
			name:          "no origins allows all",
			serverOrigins: nil,
			requestOrigin: "https://any.com",
			expectForbid:  false,
		},
		{
			name:          "wildcard allows all",
			serverOrigins: []string{"*"},
			requestOrigin: "https://any.com",
			expectForbid:  false,
		},
		{
			name:          "exact match allowed",
			serverOrigins: []string{"https://allowed.com"},
			requestOrigin: "https://allowed.com",
			expectForbid:  false,
		},
		{
			name:          "no match forbidden",
			serverOrigins: []string{"https://allowed.com"},
			requestOrigin: "https://other.com",
			expectForbid:  true,
		},
		{
			name:          "multiple origins allowed",
			serverOrigins: []string{"https://a.com", "https://b.com"},
			requestOrigin: "https://b.com",
			expectForbid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(Options{Origins: tt.serverOrigins})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			req.Header.Set("Origin", tt.requestOrigin)
			rec := httptest.NewRecorder()

			srv.Handler().ServeHTTP(rec, req)

			if tt.expectForbid && rec.Code != http.StatusForbidden {
				t.Errorf("expected forbidden, got %d", rec.Code)
			}
			if !tt.expectForbid && rec.Code == http.StatusForbidden {
				t.Error("unexpected forbidden")
			}
		})
	}
}

// Test error variables
func TestErrors(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{ErrSessionClosed, "live: session closed"},
		{ErrQueueFull, "live: send queue full"},
		{ErrAuthFailed, "live: authentication failed"},
		{ErrUpgradeFailed, "live: websocket upgrade failed"},
		{ErrInvalidMessage, "live: invalid message"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.want {
			t.Errorf("%v.Error() = %s, want %s", tt.err, tt.err.Error(), tt.want)
		}
	}
}

// Test sync notifier integration
func TestSyncNotifier(t *testing.T) {
	srv := New(Options{})
	s := newSession("s1", nil, 10, srv)
	srv.addSession(s)
	srv.PubSub().Subscribe(s, "sync:test-scope")

	notifier := SyncNotifier(srv, "sync:")
	notifier.Notify("test-scope", 42)

	select {
	case msg := <-s.sendCh:
		if msg.Type != "sync" {
			t.Errorf("msg.Type = %s, want sync", msg.Type)
		}
		if msg.Topic != "sync:test-scope" {
			t.Errorf("msg.Topic = %s, want sync:test-scope", msg.Topic)
		}
		if !strings.Contains(string(msg.Body), "42") {
			t.Errorf("msg.Body = %s, want to contain 42", msg.Body)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected sync notification")
	}
}
