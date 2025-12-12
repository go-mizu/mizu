package h2c

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
		w.Write([]byte("ok"))
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
		w.Write([]byte("ok"))
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

func TestBufferedConn(t *testing.T) {
	// This is a simplified test
	// In real usage, you'd need a real connection
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
