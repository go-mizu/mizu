package mobile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestDeviceDetection(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())
	app.Get("/test", func(c *mizu.Ctx) error {
		device := DeviceFromCtx(c)
		return c.JSON(200, device)
	})

	tests := []struct {
		name        string
		headers     map[string]string
		wantPlatform Platform
		wantVersion string
		wantAppVersion string
		wantDeviceID string
	}{
		{
			name: "iOS device",
			headers: map[string]string{
				"User-Agent":    "MyApp/1.0 (iPhone; iOS 17.2; Scale/3.00)",
				"X-Device-ID":   "abc123",
				"X-App-Version": "1.0.0",
			},
			wantPlatform:   PlatformIOS,
			wantVersion:    "17.2",
			wantAppVersion: "1.0.0",
			wantDeviceID:   "abc123",
		},
		{
			name: "Android device",
			headers: map[string]string{
				"User-Agent":    "MyApp/1.0 (Linux; Android 14.0; SM-S918B)",
				"X-Device-ID":   "xyz789",
				"X-App-Version": "2.0.0",
			},
			wantPlatform:   PlatformAndroid,
			wantVersion:    "14.0",
			wantAppVersion: "2.0.0",
			wantDeviceID:   "xyz789",
		},
		{
			name: "Web browser",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			},
			wantPlatform: PlatformWeb,
		},
		{
			name:         "No headers",
			headers:      map[string]string{},
			wantPlatform: PlatformUnknown,
		},
		{
			name: "iPad",
			headers: map[string]string{
				"User-Agent": "MyApp/1.0 (iPad; iOS 17.0; Scale/2.00)",
			},
			wantPlatform: PlatformIOS,
			wantVersion:  "17.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}

			var device Device
			if err := json.Unmarshal(rec.Body.Bytes(), &device); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if device.Platform != tt.wantPlatform {
				t.Errorf("platform = %v, want %v", device.Platform, tt.wantPlatform)
			}
			if device.Version != tt.wantVersion {
				t.Errorf("version = %v, want %v", device.Version, tt.wantVersion)
			}
			if device.AppVersion != tt.wantAppVersion {
				t.Errorf("app_version = %v, want %v", device.AppVersion, tt.wantAppVersion)
			}
			if device.DeviceID != tt.wantDeviceID {
				t.Errorf("device_id = %v, want %v", device.DeviceID, tt.wantDeviceID)
			}
		})
	}
}

func TestRequiredHeaders(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(WithOptions(Options{
		RequireDeviceID:   true,
		RequireAppVersion: true,
	}))
	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(200, "ok")
	})

	t.Run("missing device ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-App-Version", "1.0.0")

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("missing app version", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Device-ID", "abc123")

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("all headers present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Device-ID", "abc123")
		req.Header.Set("X-App-Version", "1.0.0")

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestPlatformMethods(t *testing.T) {
	t.Run("Is", func(t *testing.T) {
		if !PlatformIOS.Is(PlatformIOS) {
			t.Error("iOS.Is(iOS) should be true")
		}
		if PlatformIOS.Is(PlatformAndroid) {
			t.Error("iOS.Is(Android) should be false")
		}
	})

	t.Run("IsMobile", func(t *testing.T) {
		if !PlatformIOS.IsMobile() {
			t.Error("iOS should be mobile")
		}
		if !PlatformAndroid.IsMobile() {
			t.Error("Android should be mobile")
		}
		if PlatformWeb.IsMobile() {
			t.Error("Web should not be mobile")
		}
		if PlatformUnknown.IsMobile() {
			t.Error("Unknown should not be mobile")
		}
	})
}

func TestVersionParsing(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"v1", Version{Major: 1}, false},
		{"v2", Version{Major: 2}, false},
		{"v1.2", Version{Major: 1, Minor: 2}, false},
		{"1", Version{Major: 1}, false},
		{"1.5", Version{Major: 1, Minor: 5}, false},
		{"V1", Version{Major: 1}, false},
		{"", Version{}, true},
		{"vX", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && v != tt.want {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, v, tt.want)
			}
		})
	}
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		a, b Version
		want int
	}{
		{Version{1, 0}, Version{1, 0}, 0},
		{Version{2, 0}, Version{1, 0}, 1},
		{Version{1, 0}, Version{2, 0}, -1},
		{Version{1, 2}, Version{1, 1}, 1},
		{Version{1, 1}, Version{1, 2}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.a.String()+"_vs_"+tt.b.String(), func(t *testing.T) {
			if got := tt.a.Compare(tt.b); got != tt.want {
				t.Errorf("Compare(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersionMiddleware(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(VersionMiddleware(VersionOptions{
		Default: Version{Major: 1},
		Supported: []Version{
			{Major: 1},
			{Major: 2},
		},
		Deprecated: []Version{
			{Major: 1},
		},
	}))
	app.Get("/test", func(c *mizu.Ctx) error {
		v := VersionFromCtx(c)
		return c.JSON(200, map[string]string{"version": v.String()})
	})

	t.Run("default version", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-API-Deprecated") != "true" {
			t.Error("expected deprecation warning for v1")
		}
	})

	t.Run("v2 not deprecated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Version", "v2")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-API-Deprecated") != "" {
			t.Error("v2 should not be deprecated")
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Version", "v3")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for unsupported version, got %d", rec.Code)
		}
	})
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		query       string
		wantPage    int
		wantPerPage int
	}{
		{"", 1, 20},
		{"?page=2", 2, 20},
		{"?page=3&per_page=50", 3, 50},
		{"?per_page=200", 1, 100}, // max is 100
		{"?page=0", 1, 20},        // min page is 1
		{"?limit=30", 1, 30},      // limit alias
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			app := mizu.NewRouter()
			var gotPage, gotPerPage int
			app.Get("/test", func(c *mizu.Ctx) error {
				gotPage, gotPerPage = Paginate(c)
				return c.Text(200, "ok")
			})

			req := httptest.NewRequest("GET", "/test"+tt.query, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if gotPage != tt.wantPage {
				t.Errorf("page = %d, want %d", gotPage, tt.wantPage)
			}
			if gotPerPage != tt.wantPerPage {
				t.Errorf("perPage = %d, want %d", gotPerPage, tt.wantPerPage)
			}
		})
	}
}

func TestPage(t *testing.T) {
	items := []string{"a", "b", "c"}
	page := NewPage(items, 1, 3, 10)

	if !page.HasMore {
		t.Error("expected HasMore = true")
	}
	if len(page.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(page.Items))
	}

	// Last page
	lastPage := NewPage(items, 4, 3, 10)
	if lastPage.HasMore {
		t.Error("last page should not HasMore")
	}
}

func TestCursorPage(t *testing.T) {
	items := []string{"a", "b", "c"}
	page := NewCursorPage(items, 20, "next123")

	if !page.HasMore {
		t.Error("expected HasMore = true when cursor present")
	}
	if page.NextCursor != "next123" {
		t.Errorf("NextCursor = %q, want 'next123'", page.NextCursor)
	}

	// No more results
	lastPage := NewCursorPage(items, 20, "")
	if lastPage.HasMore {
		t.Error("should not HasMore when no cursor")
	}
}

func TestETag(t *testing.T) {
	data := map[string]string{"name": "test"}

	etag := ETag(data)
	if etag == "" {
		t.Error("ETag should not be empty")
	}
	if etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Error("ETag should be quoted")
	}

	// Same data = same ETag
	etag2 := ETag(data)
	if etag != etag2 {
		t.Error("same data should produce same ETag")
	}

	// Different data = different ETag
	data["name"] = "other"
	etag3 := ETag(data)
	if etag == etag3 {
		t.Error("different data should produce different ETag")
	}
}

func TestConditional(t *testing.T) {
	app := mizu.NewRouter()
	app.Get("/test", func(c *mizu.Ctx) error {
		etag := `"abc123"`
		if Conditional(c, etag) {
			return nil
		}
		return c.Text(200, "content")
	})

	t.Run("no If-None-Match", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
		if rec.Header().Get("ETag") != `"abc123"` {
			t.Error("ETag header should be set")
		}
	})

	t.Run("matching ETag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("If-None-Match", `"abc123"`)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotModified {
			t.Errorf("expected 304, got %d", rec.Code)
		}
	})

	t.Run("non-matching ETag", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("If-None-Match", `"xyz789"`)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestCacheControl(t *testing.T) {
	tests := []struct {
		name string
		cc   CacheControl
		want string
	}{
		{"no-store", CacheControl{NoStore: true}, "no-store"},
		{"no-cache", CacheControl{NoCache: true}, "no-cache"},
		{"private", CacheControl{Private: true, MaxAge: 5 * time.Minute}, "private, max-age=300"},
		{"public", CacheControl{MaxAge: time.Hour}, "public, max-age=3600"},
		{"immutable", CacheControl{MaxAge: 365 * 24 * time.Hour, Immutable: true}, "public, max-age=31536000, immutable"},
		{"must-revalidate", CacheControl{MaxAge: time.Minute, MustRevalidate: true}, "public, max-age=60, must-revalidate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cc.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSyncToken(t *testing.T) {
	now := time.Now()
	token := NewSyncToken(now)

	if token == "" {
		t.Error("token should not be empty")
	}

	parsed, err := ParseSyncTokenTime(token)
	if err != nil {
		t.Fatalf("ParseSyncTokenTime failed: %v", err)
	}

	// Compare with nanosecond precision
	if parsed.UnixNano() != now.UnixNano() {
		t.Errorf("parsed time = %v, want %v", parsed, now)
	}
}

func TestSyncDelta(t *testing.T) {
	delta := NewSyncDelta[string]()

	if !delta.IsEmpty() {
		t.Error("new delta should be empty")
	}

	delta.AddCreated("item1")
	delta.AddUpdated("item2")
	delta.AddDeleted("item3")

	if delta.IsEmpty() {
		t.Error("delta should not be empty")
	}
	if delta.Total() != 3 {
		t.Errorf("Total() = %d, want 3", delta.Total())
	}

	resp := delta.ToSyncResponse(false)
	if resp.SyncState.HasMore {
		t.Error("HasMore should be false")
	}
	if resp.SyncState.SyncToken == "" {
		t.Error("SyncToken should not be empty")
	}
}

func TestPushToken(t *testing.T) {
	app := mizu.NewRouter()
	app.Use(New())
	app.Get("/test", func(c *mizu.Ctx) error {
		token := ParsePushToken(c)
		return c.JSON(200, token)
	})

	t.Run("iOS device", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "MyApp/1.0 (iPhone; iOS 17.0)")
		req.Header.Set("X-Device-ID", "abc123")
		req.Header.Set("X-Push-Token", "apns-token-here")

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		var token PushToken
		json.Unmarshal(rec.Body.Bytes(), &token)

		if token.Type != TokenAPNS {
			t.Errorf("expected APNS, got %v", token.Type)
		}
		if token.Token != "apns-token-here" {
			t.Errorf("token mismatch")
		}
		if token.DeviceID != "abc123" {
			t.Errorf("device ID mismatch")
		}
	})

	t.Run("Android device", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", "MyApp/1.0 (Android 14)")
		req.Header.Set("X-Device-ID", "xyz789")
		req.Header.Set("X-Push-Token", "fcm-token-here")

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		var token PushToken
		json.Unmarshal(rec.Body.Bytes(), &token)

		if token.Type != TokenFCM {
			t.Errorf("expected FCM, got %v", token.Type)
		}
	})

	t.Run("no push token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		var token PushToken
		json.Unmarshal(rec.Body.Bytes(), &token)

		if token.IsValid() {
			t.Error("token should not be valid without header")
		}
	})
}

func TestError(t *testing.T) {
	err := NewError(ErrCodeNotFound, "User not found").
		WithDetails(map[string]string{"user_id": "123"}).
		WithTraceID("trace-abc")

	if err.Code != ErrCodeNotFound {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeNotFound)
	}
	if err.Message != "User not found" {
		t.Errorf("Message = %q, want 'User not found'", err.Message)
	}
	if err.Details["user_id"] != "123" {
		t.Error("Details should contain user_id")
	}
	if err.TraceID != "trace-abc" {
		t.Errorf("TraceID = %q, want 'trace-abc'", err.TraceID)
	}
	if err.Error() != "User not found" {
		t.Error("Error() should return message")
	}
}

func TestSendError(t *testing.T) {
	app := mizu.NewRouter()
	app.Get("/test", func(c *mizu.Ctx) error {
		return SendError(c, http.StatusNotFound, NewError(ErrCodeNotFound, "Not found"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}

	var resp map[string]Error
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["error"].Code != ErrCodeNotFound {
		t.Error("error code mismatch")
	}
}

func TestParseLocale(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"en-US", "en-US"},
		{"en-US,en;q=0.9", "en-US"},
		{"en-GB;q=0.8", "en-GB"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLocale(tt.input)
			if got != tt.want {
				t.Errorf("parseLocale(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
