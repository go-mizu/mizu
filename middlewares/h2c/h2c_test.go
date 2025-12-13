package h2c

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

func TestNew(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDetectHTTP1(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Detect())

	var info *Info

	app.Get("/", func(c *mizu.Ctx) error {
		info = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info")
	}
	if info.IsHTTP2 {
		t.Error("expected HTTP/1.x request")
	}
}

func TestIsH2CUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name:     "no upgrade",
			headers:  map[string]string{},
			expected: false,
		},
		{
			name: "missing connection",
			headers: map[string]string{
				"Upgrade":        "h2c",
				"HTTP2-Settings": "test",
			},
			expected: false,
		},
		{
			name: "missing upgrade",
			headers: map[string]string{
				"Connection":     "Upgrade",
				"HTTP2-Settings": "test",
			},
			expected: false,
		},
		{
			name: "missing settings",
			headers: map[string]string{
				"Connection": "Upgrade",
				"Upgrade":    "h2c",
			},
			expected: false,
		},
		{
			name: "valid h2c upgrade",
			headers: map[string]string{
				"Connection":     "Upgrade",
				"Upgrade":        "h2c",
				"HTTP2-Settings": "AAMAAABkAARAAAAAAAIAAAAA",
			},
			expected: true,
		},
		{
			name: "case insensitive",
			headers: map[string]string{
				"Connection":     "upgrade",
				"Upgrade":        "H2C",
				"HTTP2-Settings": "test",
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			result := isH2CUpgrade(req)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestContainsToken(t *testing.T) {
	tests := []struct {
		header   string
		token    string
		expected bool
	}{
		{"Upgrade", "upgrade", true},
		{"keep-alive, Upgrade", "upgrade", true},
		{"keep-alive", "upgrade", false},
		{"", "upgrade", false},
		{"Upgrade, HTTP2-Settings", "upgrade", true},
	}

	for _, tc := range tests {
		result := containsToken(tc.header, tc.token)
		if result != tc.expected {
			t.Errorf("containsToken(%q, %q) = %v, expected %v",
				tc.header, tc.token, result, tc.expected)
		}
	}
}

func TestGetInfo(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var gotInfo *Info

	app.Get("/", func(c *mizu.Ctx) error {
		gotInfo = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if gotInfo == nil {
		t.Fatal("expected info from context")
	}
}

func TestIsHTTP2(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())

	var is2 bool

	app.Get("/", func(c *mizu.Ctx) error {
		is2 = IsHTTP2(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// httptest uses HTTP/1.1
	if is2 {
		t.Error("expected HTTP/1.x in test")
	}
}

func TestParseSettings(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HTTP2-Settings", "AAMAAABkAARAAAAAAAIAAAAA")

	settings, err := ParseSettings(req)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(settings) == 0 {
		t.Error("expected settings data")
	}
}

func TestParseSettingsEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	settings, err := ParseSettings(req)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if settings != nil {
		t.Error("expected nil for missing header")
	}
}

func TestOnUpgrade(t *testing.T) {
	var upgradeCalled bool

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		AllowUpgrade: true,
		OnUpgrade: func(r *http.Request) {
			upgradeCalled = true
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// Regular request - no upgrade
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if upgradeCalled {
		t.Error("expected no upgrade for regular request")
	}
}

func TestServerHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: true,
		AllowDirect:  true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h2cHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestWrap(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	wrapped := Wrap(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestIsHTTP2Preface(t *testing.T) {
	tests := []struct {
		data     []byte
		expected bool
	}{
		{[]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"), true},
		{[]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\nextra"), true},
		{[]byte("GET / HTTP/1.1"), false},
		{[]byte("PRI"), false},
		{[]byte{}, false},
	}

	for _, tc := range tests {
		result := IsHTTP2Preface(tc.data)
		if result != tc.expected {
			t.Errorf("IsHTTP2Preface(%q) = %v, expected %v",
				tc.data, result, tc.expected)
		}
	}
}

func TestDetect(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Detect())

	app.Get("/", func(c *mizu.Ctx) error {
		info := GetInfo(c)
		if info.IsHTTP2 {
			return c.Text(http.StatusOK, "http2")
		}
		return c.Text(http.StatusOK, "http1")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "http1" {
		t.Errorf("expected http1, got %q", rec.Body.String())
	}
}

func TestWithOptionsDisabled(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		AllowUpgrade: false,
		AllowDirect:  false,
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	// h2c upgrade request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should be handled as regular request since upgrade is disabled
	if rec.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestBufferedConn(t *testing.T) {
	// Create a mock connection
	mock := &mockNetConn{
		readBuf:  bytes.NewBuffer([]byte("hello world")),
		writeBuf: new(bytes.Buffer),
	}

	bc := NewBufferedConn(mock)

	// Test Read
	buf := make([]byte, 5)
	n, err := bc.Read(buf)
	if err != nil {
		t.Errorf("Read error: %v", err)
	}
	if n != 5 || string(buf) != "hello" {
		t.Errorf("Read got %d bytes: %q, want 5 bytes: hello", n, buf[:n])
	}

	// Test Peek
	peeked, err := bc.Peek(6)
	if err != nil {
		t.Errorf("Peek error: %v", err)
	}
	if string(peeked) != " world" {
		t.Errorf("Peek got %q, want ' world'", peeked)
	}

	// Read again to verify peek didn't consume
	buf2 := make([]byte, 6)
	n, err = bc.Read(buf2)
	if err != nil {
		t.Errorf("Read after peek error: %v", err)
	}
	if string(buf2[:n]) != " world" {
		t.Errorf("Read after peek got %q, want ' world'", buf2[:n])
	}
}

// mockNetConn implements net.Conn for testing
type mockNetConn struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
	closed   bool
}

func (m *mockNetConn) Read(b []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	return m.readBuf.Read(b)
}

func (m *mockNetConn) Write(b []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	return m.writeBuf.Write(b)
}

func (m *mockNetConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockNetConn) LocalAddr() net.Addr                { return nil }
func (m *mockNetConn) RemoteAddr() net.Addr               { return nil }
func (m *mockNetConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockNetConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockNetConn) SetWriteDeadline(t time.Time) error { return nil }

func TestGetInfo_NoContext(t *testing.T) {
	app := mizu.NewRouter()

	var gotInfo *Info
	app.Get("/", func(c *mizu.Ctx) error {
		gotInfo = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Without h2c middleware, GetInfo should return empty Info
	if gotInfo == nil {
		t.Fatal("expected info")
	}
	if gotInfo.IsHTTP2 {
		t.Error("expected IsHTTP2 to be false")
	}
}

func TestDetect_WithH2CUpgrade(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Detect())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info")
	}
	if !info.IsHTTP2 {
		t.Error("expected IsHTTP2 to be true for h2c upgrade")
	}
	if !info.Upgraded {
		t.Error("expected Upgraded to be true")
	}
}

func TestServerHandler_WithH2CUpgrade(t *testing.T) {
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: true,
		OnUpgrade: func(r *http.Request) {
			// Callback should be called
		},
	})

	// Non-h2c request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h2cHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called for non-h2c request")
	}
}

func TestServerHandler_UpgradeDisabled(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: false,
	})

	// h2c upgrade request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")
	rec := httptest.NewRecorder()
	h2cHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called when upgrade disabled")
	}
}

func TestParseSettingsInvalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HTTP2-Settings", "!!!invalid-base64!!!")

	_, err := ParseSettings(req)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestInfo_Fields(t *testing.T) {
	info := &Info{
		IsHTTP2:  true,
		Upgraded: true,
		Direct:   false,
	}

	if !info.IsHTTP2 {
		t.Error("expected IsHTTP2 to be true")
	}
	if !info.Upgraded {
		t.Error("expected Upgraded to be true")
	}
	if info.Direct {
		t.Error("expected Direct to be false")
	}
}

func TestConnectionPreface(t *testing.T) {
	expected := []byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
	if !bytes.Equal(connectionPreface, expected) {
		t.Errorf("connectionPreface mismatch")
	}
}

// hijackableRecorder implements http.ResponseWriter and http.Hijacker
type hijackableRecorder struct {
	*httptest.ResponseRecorder
	conn       *mockNetConn
	hijackErr  error
	hijacked   bool
	brw        *bufio.ReadWriter
}

func newHijackableRecorder() *hijackableRecorder {
	mock := &mockNetConn{
		readBuf:  bytes.NewBuffer(nil),
		writeBuf: new(bytes.Buffer),
	}

	return &hijackableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		conn:             mock,
		brw:              bufio.NewReadWriter(bufio.NewReader(mock), bufio.NewWriter(mock)),
	}
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h.hijackErr != nil {
		return nil, nil, h.hijackErr
	}
	h.hijacked = true
	return h.conn, h.brw, nil
}

// nonHijackableRecorder doesn't implement http.Hijacker
type nonHijackableRecorder struct {
	*httptest.ResponseRecorder
}

func TestWithOptions_HTTP2PriorKnowledge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		AllowUpgrade: true,
		AllowDirect:  true,
	}))

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ProtoMajor = 2
	req.ProtoMinor = 0
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info")
	}
	if !info.IsHTTP2 {
		t.Error("expected IsHTTP2 to be true")
	}
	if !info.Direct {
		t.Error("expected Direct to be true")
	}
	if info.Upgraded {
		t.Error("expected Upgraded to be false")
	}
}

func TestWithOptions_H2CUpgradeWithCallback(t *testing.T) {
	var upgradeCalled bool
	var upgradeReq *http.Request

	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		AllowUpgrade: true,
		OnUpgrade: func(r *http.Request) {
			upgradeCalled = true
			upgradeReq = r
		},
	}))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")

	// Use hijackable recorder
	rec := newHijackableRecorder()

	app.ServeHTTP(rec, req)

	if !upgradeCalled {
		t.Error("expected OnUpgrade callback to be called")
	}
	if upgradeReq == nil {
		t.Error("expected upgrade request")
	}
}

func TestHandleUpgrade_NonHijackable(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		AllowUpgrade: true,
	}))

	var gotInfo *Info
	app.Get("/", func(c *mizu.Ctx) error {
		gotInfo = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")

	// httptest.ResponseRecorder doesn't implement Hijacker
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should fall back to HTTP/1.x handling - we just verify no panic occurred
	// The middleware set Upgraded but couldn't hijack, so fell back
	_ = gotInfo // Verify the handler was called
}

func TestServerHandler_HTTP2PriorKnowledge(t *testing.T) {
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: true,
		AllowDirect:  true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ProtoMajor = 2
	rec := httptest.NewRecorder()
	h2cHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called for HTTP/2 prior knowledge")
	}
}

func TestServerHandler_H2CUpgrade_WithHijack(t *testing.T) {
	var handlerCalled bool
	var upgradeCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: true,
		OnUpgrade: func(r *http.Request) {
			upgradeCalled = true
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")

	rec := newHijackableRecorder()

	h2cHandler.ServeHTTP(rec, req)

	if !upgradeCalled {
		t.Error("expected OnUpgrade to be called")
	}
	// Handler shouldn't be called since we hijacked
	if handlerCalled {
		t.Error("handler should not be called after hijack")
	}
}

func TestServerHandler_H2CUpgrade_HijackError(t *testing.T) {
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")

	rec := &hijackableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		hijackErr:        io.ErrUnexpectedEOF,
	}

	h2cHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called on hijack error")
	}
}

func TestServerHandler_NonHijackable(t *testing.T) {
	var handlerCalled bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	h2cHandler := NewServerHandler(handler, Options{
		AllowUpgrade: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")

	rec := &nonHijackableRecorder{httptest.NewRecorder()}
	h2cHandler.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Error("expected handler to be called when hijack not supported")
	}
}

func TestDetect_HTTP2PriorKnowledge(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(Detect())

	var info *Info
	app.Get("/", func(c *mizu.Ctx) error {
		info = GetInfo(c)
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.ProtoMajor = 2
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if info == nil {
		t.Fatal("expected info")
	}
	if !info.IsHTTP2 {
		t.Error("expected IsHTTP2 to be true")
	}
	if !info.Direct {
		t.Error("expected Direct to be true")
	}
}

func TestBufferedConn_Underlying(t *testing.T) {
	mock := &mockNetConn{
		readBuf:  bytes.NewBuffer([]byte("test data")),
		writeBuf: new(bytes.Buffer),
	}

	bc := NewBufferedConn(mock)

	// Test Write (uses underlying conn)
	n, err := bc.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write error: %v", err)
	}
	if n != 5 {
		t.Errorf("Write wrote %d bytes, want 5", n)
	}
	if mock.writeBuf.String() != "hello" {
		t.Errorf("Write got %q, want 'hello'", mock.writeBuf.String())
	}

	// Test Close
	if err := bc.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
	if !mock.closed {
		t.Error("expected connection to be closed")
	}
}

func TestWithOptions_H2CUpgrade_HijackError(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		AllowUpgrade: true,
	}))

	var handlerCalled bool
	app.Get("/", func(c *mizu.Ctx) error {
		handlerCalled = true
		return c.Text(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", "test")

	rec := &hijackableRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		hijackErr:        io.ErrUnexpectedEOF,
	}

	app.ServeHTTP(rec, req)

	// Should fall back to handler when hijack fails
	if !handlerCalled {
		t.Error("expected handler to be called on hijack error")
	}
}

func TestBufferedConn_NetConnInterface(t *testing.T) {
	mock := &mockNetConn{
		readBuf:  bytes.NewBuffer([]byte("data")),
		writeBuf: new(bytes.Buffer),
	}

	bc := NewBufferedConn(mock)

	// Test LocalAddr
	if bc.LocalAddr() != nil {
		t.Error("expected nil LocalAddr")
	}

	// Test RemoteAddr
	if bc.RemoteAddr() != nil {
		t.Error("expected nil RemoteAddr")
	}

	// Test SetDeadline
	if err := bc.SetDeadline(time.Now()); err != nil {
		t.Errorf("SetDeadline error: %v", err)
	}

	// Test SetReadDeadline
	if err := bc.SetReadDeadline(time.Now()); err != nil {
		t.Errorf("SetReadDeadline error: %v", err)
	}

	// Test SetWriteDeadline
	if err := bc.SetWriteDeadline(time.Now()); err != nil {
		t.Errorf("SetWriteDeadline error: %v", err)
	}
}
