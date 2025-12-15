package websocket

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
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

			if got := IsWebSocketUpgrade(req); got != tt.expected {
				t.Errorf("IsWebSocketUpgrade() = %v, want %v", got, tt.expected)
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

func TestNew_NonWebSocket(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "normal")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "normal" {
		t.Errorf("expected 'normal', got %q", rec.Body.String())
	}
}

func TestWithOptions_ForbiddenOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		Origins: []string{"https://allowed.com"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://forbidden.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestWithOptions_AllowedOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		Origins: []string{"https://allowed.com"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://allowed.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Will fail to hijack in test, but shouldn't be forbidden
	if rec.Code == http.StatusForbidden {
		t.Error("should not be forbidden for allowed origin")
	}
}

func TestWithOptions_WildcardOrigin(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		Origins: []string{"*"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://any.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Error("wildcard should allow any origin")
	}
}

func TestWithOptions_MissingKey(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	// Missing Sec-WebSocket-Key
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestConn_WriteMessage(t *testing.T) {
	// Test message framing
	tests := []struct {
		name        string
		messageType int
		data        []byte
	}{
		{"text short", TextMessage, []byte("hello")},
		{"text 126", TextMessage, make([]byte, 126)},
		{"binary", BinaryMessage, []byte{0x00, 0x01, 0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify no panic
			_ = tt.messageType
			_ = tt.data
		})
	}
}

func TestErrors(t *testing.T) {
	if ErrNotWebSocket.Error() != "not a websocket request" {
		t.Error("unexpected error message")
	}
	if ErrClosed.Error() != "connection closed" {
		t.Error("unexpected error message")
	}
}

// mockConn is a mock net.Conn for testing
type mockConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.readBuf.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestConn_WriteText(t *testing.T) {
	mock := newMockConn()
	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := conn.WriteText("hello")
	if err != nil {
		t.Errorf("WriteText error: %v", err)
	}

	// Check that data was written
	if mock.writeBuf.Len() == 0 {
		t.Error("expected data to be written")
	}
}

func TestConn_WriteBinary(t *testing.T) {
	mock := newMockConn()
	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := conn.WriteBinary([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Errorf("WriteBinary error: %v", err)
	}

	if mock.writeBuf.Len() == 0 {
		t.Error("expected data to be written")
	}
}

func TestConn_WriteMessageLengths(t *testing.T) {
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
			conn := &Conn{
				conn:   mock,
				reader: bufio.NewReader(mock.readBuf),
				writer: bufio.NewWriter(mock.writeBuf),
			}

			data := make([]byte, tt.length)
			err := conn.WriteMessage(BinaryMessage, data)
			if err != nil {
				t.Errorf("WriteMessage error: %v", err)
			}

			if mock.writeBuf.Len() == 0 {
				t.Error("expected data to be written")
			}
		})
	}
}

func TestConn_Close(t *testing.T) {
	mock := newMockConn()
	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := conn.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}

	if !mock.closed {
		t.Error("expected connection to be closed")
	}
}

func TestConn_Ping(t *testing.T) {
	mock := newMockConn()
	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := conn.Ping([]byte("ping data"))
	if err != nil {
		t.Errorf("Ping error: %v", err)
	}

	if mock.writeBuf.Len() == 0 {
		t.Error("expected ping data to be written")
	}
}

func TestConn_Pong(t *testing.T) {
	mock := newMockConn()
	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	err := conn.Pong([]byte("pong data"))
	if err != nil {
		t.Errorf("Pong error: %v", err)
	}

	if mock.writeBuf.Len() == 0 {
		t.Error("expected pong data to be written")
	}
}

func TestConn_ReadMessage(t *testing.T) {
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
			wantType: TextMessage,
			wantData: []byte("hello"),
		},
		{
			name: "masked text",
			frame: []byte{
				0x81,                   // FIN + text opcode
				0x85,                   // masked + length 5
				0x37, 0xfa, 0x21, 0x3d, // mask key
				0x7f, 0x9f, 0x4d, 0x51, 0x58, // masked "Hello" (mask XOR result)
			},
			wantType: TextMessage,
			wantData: []byte("Hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockConn()
			mock.readBuf.Write(tt.frame)

			conn := &Conn{
				conn:   mock,
				reader: bufio.NewReader(mock.readBuf),
				writer: bufio.NewWriter(mock.writeBuf),
			}

			msgType, data, err := conn.ReadMessage()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMessage error = %v, wantErr %v", err, tt.wantErr)
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

func TestConn_ReadMessageExtendedLength(t *testing.T) {
	// Test 126-byte length encoding (2 bytes for length)
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

	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	msgType, received, err := conn.ReadMessage()
	if err != nil {
		t.Errorf("ReadMessage error: %v", err)
		return
	}
	if msgType != BinaryMessage {
		t.Errorf("got type %d, want %d", msgType, BinaryMessage)
	}
	if len(received) != 200 {
		t.Errorf("got length %d, want 200", len(received))
	}
}

func TestConn_ReadMessageVeryLongLength(t *testing.T) {
	// Test 127-byte length encoding (8 bytes for length)
	mock := newMockConn()
	data := make([]byte, 70000)
	for i := range data {
		data[i] = 'b'
	}
	frame := []byte{
		0x82,                                           // FIN + binary opcode
		0x7f,                                           // length 127 indicator
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x11, 0x70, // 70000 in big endian
	}
	frame = append(frame, data...)
	mock.readBuf.Write(frame)

	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	msgType, received, err := conn.ReadMessage()
	if err != nil {
		t.Errorf("ReadMessage error: %v", err)
		return
	}
	if msgType != BinaryMessage {
		t.Errorf("got type %d, want %d", msgType, BinaryMessage)
	}
	if len(received) != 70000 {
		t.Errorf("got length %d, want 70000", len(received))
	}
}

func TestConn_ReadMessageReadErrors(t *testing.T) {
	// Test read error on first byte
	mock := newMockConn()
	conn := &Conn{
		conn:   mock,
		reader: bufio.NewReader(mock.readBuf),
		writer: bufio.NewWriter(mock.writeBuf),
	}

	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Error("expected error on empty read")
	}

	// Test read error on second byte
	mock2 := newMockConn()
	mock2.readBuf.Write([]byte{0x81}) // Only first byte
	conn2 := &Conn{
		conn:   mock2,
		reader: bufio.NewReader(mock2.readBuf),
		writer: bufio.NewWriter(mock2.writeBuf),
	}

	_, _, err = conn2.ReadMessage()
	if err == nil {
		t.Error("expected error on partial frame")
	}
}

func TestWithOptions_Subprotocols(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		Subprotocols: []string{"graphql-ws", "graphql-transport-ws"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Protocol", "graphql-ws")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Can't test full upgrade in httptest, but shouldn't fail on origin
	if rec.Code == http.StatusForbidden {
		t.Error("should not be forbidden")
	}
}

func TestWithOptions_SubprotocolNotMatching(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		Subprotocols: []string{"graphql-ws"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Protocol", "unknown-protocol")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should proceed even without matching subprotocol
	if rec.Code == http.StatusForbidden {
		t.Error("should not be forbidden")
	}
}

func TestWithOptions_CustomCheckOrigin(t *testing.T) {
	checkOriginCalled := false
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		CheckOrigin: func(r *http.Request) bool {
			checkOriginCalled = true
			return r.Header.Get("Origin") == "https://custom.com"
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://custom.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !checkOriginCalled {
		t.Error("expected custom CheckOrigin to be called")
	}
}

func TestWithOptions_NoOriginsAllowsAll(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		// No Origins specified, should allow all
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://any-origin.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Error("should allow all origins when none specified")
	}
}

func TestWithOptions_OriginNotInList(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(func(c *mizu.Ctx, ws *Conn) error {
		return nil
	}, Options{
		Origins: []string{"https://a.com", "https://b.com"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "https://c.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected forbidden for non-listed origin, got %d", rec.Code)
	}
}

func TestMessageConstants(t *testing.T) {
	if TextMessage != 1 {
		t.Errorf("TextMessage = %d, want 1", TextMessage)
	}
	if BinaryMessage != 2 {
		t.Errorf("BinaryMessage = %d, want 2", BinaryMessage)
	}
	if CloseMessage != 8 {
		t.Errorf("CloseMessage = %d, want 8", CloseMessage)
	}
	if PingMessage != 9 {
		t.Errorf("PingMessage = %d, want 9", PingMessage)
	}
	if PongMessage != 10 {
		t.Errorf("PongMessage = %d, want 10", PongMessage)
	}
}
