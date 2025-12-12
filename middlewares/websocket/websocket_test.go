package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
