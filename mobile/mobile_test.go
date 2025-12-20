package mobile

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
)

func TestParsePlatformString(t *testing.T) {
	tests := []struct {
		input    string
		expected Platform
	}{
		{"ios", PlatformIOS},
		{"iOS", PlatformIOS},
		{"IOS", PlatformIOS},
		{"iphone", PlatformIOS},
		{"ipad", PlatformIOS},
		{"android", PlatformAndroid},
		{"Android", PlatformAndroid},
		{"windows", PlatformWindows},
		{"macos", PlatformMacOS},
		{"mac", PlatformMacOS},
		{"osx", PlatformMacOS},
		{"web", PlatformWeb},
		{"unknown", PlatformUnknown},
		{"", PlatformUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parsePlatformString(tt.input)
			if result != tt.expected {
				t.Errorf("parsePlatformString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseUserAgent(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		platform   Platform
		hasVersion bool
	}{
		{
			name:       "iPhone Safari",
			userAgent:  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
			platform:   PlatformIOS,
			hasVersion: true,
		},
		{
			name:       "iPad Safari",
			userAgent:  "Mozilla/5.0 (iPad; CPU OS 16_5 like Mac OS X) AppleWebKit/605.1.15",
			platform:   PlatformIOS,
			hasVersion: true,
		},
		{
			name:       "Android Chrome",
			userAgent:  "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 Chrome/120.0",
			platform:   PlatformAndroid,
			hasVersion: true,
		},
		{
			name:       "Windows Chrome",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			platform:   PlatformWindows,
			hasVersion: true,
		},
		{
			name:       "macOS Safari",
			userAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15",
			platform:   PlatformMacOS,
			hasVersion: true,
		},
		{
			name:      "Generic Browser",
			userAgent: "Mozilla/5.0 (compatible; bot)",
			platform:  PlatformWeb,
		},
		{
			name:     "Empty",
			userAgent: "",
			platform: PlatformUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			platform, version := parseUserAgent(tt.userAgent)
			if platform != tt.platform {
				t.Errorf("platform = %q, want %q", platform, tt.platform)
			}
			if tt.hasVersion && version == "" {
				t.Error("expected version to be parsed")
			}
		})
	}
}

func TestPlatformMethods(t *testing.T) {
	tests := []struct {
		platform  Platform
		isMobile  bool
		isDesktop bool
		isNative  bool
	}{
		{PlatformIOS, true, false, true},
		{PlatformAndroid, true, false, true},
		{PlatformWindows, false, true, true},
		{PlatformMacOS, false, true, true},
		{PlatformWeb, false, false, false},
		{PlatformUnknown, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.platform.String(), func(t *testing.T) {
			if tt.platform.IsMobile() != tt.isMobile {
				t.Errorf("IsMobile() = %v, want %v", tt.platform.IsMobile(), tt.isMobile)
			}
			if tt.platform.IsDesktop() != tt.isDesktop {
				t.Errorf("IsDesktop() = %v, want %v", tt.platform.IsDesktop(), tt.isDesktop)
			}
			if tt.platform.IsNative() != tt.isNative {
				t.Errorf("IsNative() = %v, want %v", tt.platform.IsNative(), tt.isNative)
			}
		})
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		input   string
		major   int
		minor   int
		wantErr bool
	}{
		{"v1", 1, 0, false},
		{"v2.5", 2, 5, false},
		{"1", 1, 0, false},
		{"1.2", 1, 2, false},
		{"V3", 3, 0, false},
		{"", 0, 0, false},
		{"invalid", 0, 0, true},
		{"v", 0, 0, false}, // "v" alone is valid zero version
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if v.Major != tt.major || v.Minor != tt.minor {
					t.Errorf("version = %v, want {%d, %d}", v, tt.major, tt.minor)
				}
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b     Version
		expected int
	}{
		{Version{1, 0}, Version{1, 0}, 0},
		{Version{1, 0}, Version{2, 0}, -1},
		{Version{2, 0}, Version{1, 0}, 1},
		{Version{1, 1}, Version{1, 2}, -1},
		{Version{1, 2}, Version{1, 1}, 1},
		{Version{2, 0}, Version{1, 9}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.a.String()+"_vs_"+tt.b.String(), func(t *testing.T) {
			result := tt.a.Compare(tt.b)
			if result != tt.expected {
				t.Errorf("Compare() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	v := Version{2, 5}

	tests := []struct {
		major, minor int
		expected     bool
	}{
		{1, 0, true},
		{2, 0, true},
		{2, 5, true},
		{2, 6, false},
		{3, 0, false},
	}

	for _, tt := range tests {
		result := v.AtLeast(tt.major, tt.minor)
		if result != tt.expected {
			t.Errorf("AtLeast(%d, %d) = %v, want %v", tt.major, tt.minor, result, tt.expected)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.2.3", "1.2.3", 0},
		{"1.10.0", "1.9.0", 1},
		{"2.0.0-beta", "2.0.0", 0}, // Stops at prerelease, compares as equal
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			result := compareVersions(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestError(t *testing.T) {
	err := NewError("test_code", "test message")
	if err.Error() != "test_code: test message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test_code: test message")
	}

	err = err.WithDetails("key", "value").WithTraceID("trace-123")
	if err.Details["key"] != "value" {
		t.Error("WithDetails did not set value")
	}
	if err.TraceID != "trace-123" {
		t.Error("WithTraceID did not set value")
	}
}

func TestPageRequest(t *testing.T) {
	app := mizu.New()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		page := ParsePageRequest(c)
		return c.JSON(200, page)
	})

	tests := []struct {
		query   string
		page    int
		perPage int
		cursor  string
	}{
		{"", 1, 20, ""},
		{"?page=2", 2, 20, ""},
		{"?page=3&per_page=50", 3, 50, ""},
		{"?cursor=abc123", 1, 20, "abc123"},
		{"?after=xyz", 1, 20, ""},
		{"?per_page=200", 1, 100, ""}, // Capped at 100
		{"?limit=30", 1, 30, ""},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test"+tt.query, nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			var page PageRequest
			json.Unmarshal(w.Body.Bytes(), &page)

			if page.Page != tt.page {
				t.Errorf("Page = %d, want %d", page.Page, tt.page)
			}
			if page.PerPage != tt.perPage {
				t.Errorf("PerPage = %d, want %d", page.PerPage, tt.perPage)
			}
			if page.Cursor != tt.cursor {
				t.Errorf("Cursor = %q, want %q", page.Cursor, tt.cursor)
			}
		})
	}
}

func TestNewPage(t *testing.T) {
	items := []string{"a", "b", "c"}
	req := PageRequest{Page: 2, PerPage: 10}
	page := NewPage(items, req, 25)

	if len(page.Data) != 3 {
		t.Errorf("Data length = %d, want 3", len(page.Data))
	}
	if page.Page != 2 {
		t.Errorf("Page = %d, want 2", page.Page)
	}
	if page.PerPage != 10 {
		t.Errorf("PerPage = %d, want 10", page.PerPage)
	}
	if page.Total != 25 {
		t.Errorf("Total = %d, want 25", page.Total)
	}
	if page.TotalPages != 3 {
		t.Errorf("TotalPages = %d, want 3", page.TotalPages)
	}
	if !page.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestNewCursorPage(t *testing.T) {
	items := []string{"a", "b", "c"}
	page := NewCursorPage(items, "next123", "prev456", true)

	if len(page.Data) != 3 {
		t.Errorf("Data length = %d, want 3", len(page.Data))
	}
	if page.NextCursor != "next123" {
		t.Errorf("NextCursor = %q, want %q", page.NextCursor, "next123")
	}
	if page.PrevCursor != "prev456" {
		t.Errorf("PrevCursor = %q, want %q", page.PrevCursor, "prev456")
	}
	if !page.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestETag(t *testing.T) {
	data := map[string]string{"key": "value"}
	etag := ETag(data)

	if etag == "" {
		t.Error("ETag should not be empty")
	}
	if etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Error("ETag should be quoted")
	}

	// Same data should produce same ETag
	etag2 := ETag(data)
	if etag != etag2 {
		t.Error("Same data should produce same ETag")
	}

	// Different data should produce different ETag
	data["key"] = "other"
	etag3 := ETag(data)
	if etag == etag3 {
		t.Error("Different data should produce different ETag")
	}
}

func TestWeakETag(t *testing.T) {
	data := map[string]string{"key": "value"}
	etag := WeakETag(data)

	if etag == "" {
		t.Error("WeakETag should not be empty")
	}
	if etag[:2] != "W/" {
		t.Error("WeakETag should start with W/")
	}
}

func TestCacheControl(t *testing.T) {
	tests := []struct {
		cc       CacheControl
		expected string
	}{
		{CacheNone, "no-store, no-cache"},
		{CachePrivate, "no-cache, private"},
		{CacheShort, "private, max-age=300"},
		{CacheLong, "private, max-age=86400"},
		{CacheImmutable, "public, max-age=31536000, immutable"},
		{CacheControl{}, "no-cache"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.cc.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSyncToken(t *testing.T) {
	now := time.Now()
	token := NewSyncToken(now)

	if token.IsEmpty() {
		t.Error("Token should not be empty")
	}

	parsed := token.Time()
	if parsed.IsZero() {
		t.Error("Parsed time should not be zero")
	}

	// Should be within 1 second of original
	diff := parsed.Sub(now)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("Time difference too large: %v", diff)
	}
}

func TestSyncTokenEmpty(t *testing.T) {
	var token SyncToken
	if !token.IsEmpty() {
		t.Error("Empty token should be empty")
	}
	if !token.Time().IsZero() {
		t.Error("Empty token should return zero time")
	}
}

func TestDelta(t *testing.T) {
	delta := Delta[string]{
		Created: []string{"a", "b"},
		Updated: []string{"c"},
		Deleted: []string{"d", "e", "f"},
	}

	if delta.IsEmpty() {
		t.Error("Delta should not be empty")
	}
	if delta.Count() != 6 {
		t.Errorf("Count() = %d, want 6", delta.Count())
	}

	emptyDelta := Delta[string]{}
	if !emptyDelta.IsEmpty() {
		t.Error("Empty delta should be empty")
	}
}

func TestValidateAPNS(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if !ValidateAPNS(valid) {
		t.Error("Valid APNS token should validate")
	}

	invalid := "invalid"
	if ValidateAPNS(invalid) {
		t.Error("Invalid APNS token should not validate")
	}
}

func TestCheckUpdate(t *testing.T) {
	tests := []struct {
		client, latest, min string
		available, required bool
	}{
		{"1.0.0", "1.0.0", "1.0.0", false, false},
		{"1.0.0", "2.0.0", "1.0.0", true, false},
		{"1.0.0", "2.0.0", "1.5.0", true, true},
		{"1.5.0", "2.0.0", "1.5.0", true, false},
		{"2.0.0", "2.0.0", "1.5.0", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.client, func(t *testing.T) {
			status := CheckUpdate(tt.client, tt.latest, tt.min)
			if status.Available != tt.available {
				t.Errorf("Available = %v, want %v", status.Available, tt.available)
			}
			if status.Required != tt.required {
				t.Errorf("Required = %v, want %v", status.Required, tt.required)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	app := mizu.New()
	app.Use(New())

	app.Get("/test", func(c *mizu.Ctx) error {
		device := DeviceFromCtx(c)
		if device == nil {
			return c.Text(500, "no device")
		}
		return c.JSON(200, device)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Device-ID", "test-device-123")
	req.Header.Set("X-App-Version", "1.2.3")
	req.Header.Set("X-Platform", "ios")
	req.Header.Set("User-Agent", "TestApp/1.0")

	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}

	var device Device
	if err := json.Unmarshal(w.Body.Bytes(), &device); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if device.DeviceID != "test-device-123" {
		t.Errorf("DeviceID = %q, want %q", device.DeviceID, "test-device-123")
	}
	if device.AppVersion != "1.2.3" {
		t.Errorf("AppVersion = %q, want %q", device.AppVersion, "1.2.3")
	}
	if device.Platform != PlatformIOS {
		t.Errorf("Platform = %q, want %q", device.Platform, PlatformIOS)
	}
}

func TestMiddlewareRequireDeviceID(t *testing.T) {
	app := mizu.New()
	app.Use(WithOptions(Options{RequireDeviceID: true}))
	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(200, "ok")
	})

	// Without device ID
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Status = %d, want 400", w.Code)
	}

	// With device ID
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Device-ID", "device-123")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}
}

func TestMiddlewareMinVersion(t *testing.T) {
	app := mizu.New()
	app.Use(WithOptions(Options{MinAppVersion: "2.0.0"}))
	app.Get("/test", func(c *mizu.Ctx) error {
		return c.Text(200, "ok")
	})

	// Below minimum
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-App-Version", "1.0.0")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 426 {
		t.Errorf("Status = %d, want 426", w.Code)
	}

	// At minimum
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-App-Version", "2.0.0")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}
}

func TestVersionMiddleware(t *testing.T) {
	app := mizu.New()
	app.Use(VersionMiddleware(VersionOptions{
		Supported:  []Version{{1, 0}, {2, 0}},
		Deprecated: []Version{{1, 0}},
	}))
	app.Get("/test", func(c *mizu.Ctx) error {
		v := VersionFromCtx(c)
		return c.JSON(200, v)
	})

	// Valid version
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Version", "v2")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}

	// Deprecated version (should still work)
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Version", "v1")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}
	if w.Header().Get("X-API-Deprecated") != "true" {
		t.Error("Expected deprecation header")
	}

	// Unsupported version
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Version", "v3")
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Status = %d, want 400", w.Code)
	}
}

func TestDeepLinkAppleAppSiteAssociation(t *testing.T) {
	link := DeepLink{
		Scheme: "myapp",
		Host:   "example.com",
		Paths:  []string{"/share/*", "/invite/*"},
	}

	aasa := link.AppleAppSiteAssociation("TEAM123", "com.example.app")
	if len(aasa) == 0 {
		t.Error("AASA should not be empty")
	}

	var parsed map[string]any
	if err := json.Unmarshal(aasa, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["applinks"] == nil {
		t.Error("Missing applinks key")
	}
}

func TestDeepLinkAssetLinks(t *testing.T) {
	link := DeepLink{
		Scheme: "myapp",
		Host:   "example.com",
	}

	al := link.AssetLinks("com.example.app", "AA:BB:CC")
	if len(al) == 0 {
		t.Error("AssetLinks should not be empty")
	}

	var parsed []any
	if err := json.Unmarshal(al, &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if len(parsed) == 0 {
		t.Error("AssetLinks should have entries")
	}
}

func TestUniversalLinkMiddleware(t *testing.T) {
	app := mizu.New()
	app.Use(UniversalLinkMiddleware(UniversalLinkConfig{
		Apple: []AppleAppConfig{
			{TeamID: "TEAM123", BundleID: "com.example.app"},
		},
		Android: []AndroidAppConfig{
			{PackageName: "com.example.app", Fingerprints: []string{"AA:BB:CC"}},
		},
	}))
	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "home")
	})

	// Test apple-app-site-association
	req := httptest.NewRequest("GET", "/.well-known/apple-app-site-association", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected JSON content type")
	}

	// Test assetlinks.json
	req = httptest.NewRequest("GET", "/.well-known/assetlinks.json", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Status = %d, want 200", w.Code)
	}
}

func TestStaticAppInfo(t *testing.T) {
	provider := NewStaticAppInfo("2.0.0", "1.5.0", "https://example.com")
	provider.WithPlatformApp(PlatformIOS, "com.example.app", &AppInfo{
		CurrentVersion: "2.1.0",
		MinimumVersion: "1.8.0",
		UpdateURL:      "https://apps.apple.com/app/id123",
	})

	// Default
	info, err := provider.GetAppInfo(nil, PlatformAndroid, "")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if info.CurrentVersion != "2.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "2.0.0")
	}

	// Platform-specific
	info, err = provider.GetAppInfo(nil, PlatformIOS, "com.example.app")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if info.CurrentVersion != "2.1.0" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "2.1.0")
	}
}

func TestPushPayload(t *testing.T) {
	payload := &PushPayload{
		Title: "Test",
		Body:  "Hello",
	}
	payload.SetBadge(5).WithData("custom", "value")

	apns := payload.ToAPNS()
	if apns["aps"] == nil {
		t.Error("Missing aps key")
	}

	fcm := payload.ToFCM()
	if fcm["notification"] == nil {
		t.Error("Missing notification key")
	}
}
