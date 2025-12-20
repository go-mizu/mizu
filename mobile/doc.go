// Package mobile provides comprehensive mobile backend integration for Mizu.
//
// It enables Go backends to seamlessly support mobile applications across
// iOS, Android, Windows, and cross-platform frameworks (Flutter, React Native).
//
// # Quick Start
//
// Basic mobile middleware setup:
//
//	app := mizu.New()
//
//	// Add mobile device detection
//	app.Use(mobile.New())
//
//	// Add API versioning
//	app.Use(mobile.VersionMiddleware(mobile.VersionOptions{
//	    Supported:  []mobile.Version{{1, 0}, {2, 0}},
//	    Deprecated: []mobile.Version{{1, 0}},
//	}))
//
//	app.Get("/api/users", func(c *mizu.Ctx) error {
//	    device := mobile.DeviceFromCtx(c)
//	    version := mobile.VersionFromCtx(c)
//
//	    // Platform-specific logic
//	    if device.Platform == mobile.PlatformIOS {
//	        // iOS-specific response
//	    }
//
//	    // Version-specific logic
//	    if version.AtLeast(2, 0) {
//	        return c.JSON(200, v2Response)
//	    }
//	    return c.JSON(200, v1Response)
//	})
//
//	app.Listen(":3000")
//
// # Device Detection
//
// The middleware parses device information from headers and User-Agent:
//
//	app.Use(mobile.WithOptions(mobile.Options{
//	    RequireDeviceID:   true,  // Require X-Device-ID header
//	    RequireAppVersion: true,  // Require X-App-Version header
//	    MinAppVersion:     "1.5.0", // Minimum required version
//	}))
//
// Access device information in handlers:
//
//	func handler(c *mizu.Ctx) error {
//	    device := mobile.DeviceFromCtx(c)
//	    fmt.Println(device.Platform)    // "ios"
//	    fmt.Println(device.OSVersion)   // "17.0"
//	    fmt.Println(device.AppVersion)  // "1.2.3"
//	    fmt.Println(device.DeviceID)    // "550e8400-..."
//	    fmt.Println(device.DeviceModel) // "iPhone15,2"
//	    fmt.Println(device.Timezone)    // "America/New_York"
//	    return nil
//	}
//
// # API Versioning
//
// Support multiple API versions with deprecation warnings:
//
//	app.Use(mobile.VersionMiddleware(mobile.VersionOptions{
//	    Header:     "X-API-Version",     // Version header
//	    Default:    mobile.Version{1, 0}, // Default version
//	    Supported:  []mobile.Version{{1, 0}, {2, 0}},
//	    Deprecated: []mobile.Version{{1, 0}},
//	}))
//
// Version-aware handlers:
//
//	func handler(c *mizu.Ctx) error {
//	    v := mobile.VersionFromCtx(c)
//
//	    if v.Before(2, 0) {
//	        return c.JSON(200, legacyResponse)
//	    }
//	    return c.JSON(200, modernResponse)
//	}
//
// # Pagination
//
// Built-in pagination helpers:
//
//	func listItems(c *mizu.Ctx) error {
//	    page := mobile.ParsePageRequest(c) // Parses ?page=1&per_page=20
//
//	    items, total := db.ListItems(page.Offset(), page.Limit())
//	    return c.JSON(200, mobile.NewPage(items, page, total))
//	}
//
// Cursor-based pagination:
//
//	func listItems(c *mizu.Ctx) error {
//	    page := mobile.ParsePageRequest(c) // Parses ?cursor=xxx
//
//	    items, nextCursor := db.ListAfter(page.CursorValue(), page.Limit())
//	    return c.JSON(200, mobile.NewCursorPage(items, nextCursor, "", len(items) == page.Limit()))
//	}
//
// # Structured Errors
//
// Consistent error responses for mobile clients:
//
//	func handler(c *mizu.Ctx) error {
//	    if !authorized {
//	        return mobile.SendError(c, 401, mobile.NewError(
//	            mobile.ErrUnauthorized,
//	            "Invalid credentials",
//	        ).WithDetails("reason", "token_expired"))
//	    }
//	    return nil
//	}
//
// Error response format:
//
//	{
//	    "code": "unauthorized",
//	    "message": "Invalid credentials",
//	    "details": {"reason": "token_expired"},
//	    "trace_id": "req-abc-123"
//	}
//
// # ETag Support
//
// Efficient caching with ETags:
//
//	func getUser(c *mizu.Ctx) error {
//	    user := db.GetUser(id)
//
//	    // Auto ETag handling with 304 support
//	    return mobile.Conditional(c, user)
//	}
//
// Manual ETag control:
//
//	func getUser(c *mizu.Ctx) error {
//	    user := db.GetUser(id)
//	    etag := mobile.ETag(user)
//
//	    if mobile.CheckETag(c, etag) {
//	        return nil // 304 Not Modified
//	    }
//
//	    c.Header().Set("ETag", etag)
//	    return c.JSON(200, user)
//	}
//
// # Offline Sync
//
// Delta synchronization for offline-first apps:
//
//	func syncItems(c *mizu.Ctx) error {
//	    req := mobile.ParseSyncRequest(c)
//
//	    var delta mobile.Delta[Item]
//	    if req.IsInitial() {
//	        delta.Created = db.GetAllItems()
//	    } else {
//	        since := req.Since()
//	        delta.Created = db.GetCreatedSince(since)
//	        delta.Updated = db.GetUpdatedSince(since)
//	        delta.Deleted = db.GetDeletedSince(since)
//	    }
//
//	    token := mobile.NewSyncToken(time.Now())
//	    return c.JSON(200, mobile.NewSyncDelta(delta, token, false))
//	}
//
// # Push Notifications
//
// Push token management:
//
//	func registerPush(c *mizu.Ctx) error {
//	    token := mobile.ParsePushToken(c)
//	    if token == nil {
//	        return mobile.SendError(c, 400, mobile.NewError(
//	            mobile.ErrInvalidRequest,
//	            "Missing push token",
//	        ))
//	    }
//
//	    // Validate token format
//	    if !mobile.ValidateToken(token.Token, token.Provider) {
//	        return mobile.SendError(c, 400, mobile.NewError(
//	            mobile.ErrValidation,
//	            "Invalid push token format",
//	        ))
//	    }
//
//	    db.SavePushToken(token)
//	    return c.NoContent()
//	}
//
// # Deep Linking
//
// Universal link configuration:
//
//	link := mobile.DeepLink{
//	    Scheme:   "myapp",
//	    Host:     "example.com",
//	    Paths:    []string{"/share/*", "/invite/*"},
//	    Fallback: "https://example.com",
//	}
//
//	app.Use(mobile.UniversalLinkMiddleware(mobile.UniversalLinkConfig{
//	    Apple: []mobile.AppleAppConfig{
//	        {TeamID: "ABCD1234", BundleID: "com.example.app", Paths: []string{"*"}},
//	    },
//	    Android: []mobile.AndroidAppConfig{
//	        {PackageName: "com.example.app", Fingerprints: []string{"AA:BB:CC:..."}},
//	    },
//	}))
//
// # App Version Checking
//
// Force update support:
//
//	provider := mobile.NewStaticAppInfo("2.0.0", "1.5.0", "https://apps.apple.com/app/id123")
//	app.Get("/api/app-info", mobile.AppInfoHandler(provider))
//
// # HTTP Headers
//
// Request headers parsed by the middleware:
//
//	X-Device-ID       Unique device identifier
//	X-App-Version     Client app version (e.g., "1.2.3")
//	X-App-Build       Build number
//	X-Device-Model    Device model (e.g., "iPhone15,2")
//	X-Platform        Platform override (ios, android)
//	X-OS-Version      OS version
//	X-Timezone        IANA timezone
//	X-Locale          Device locale
//	X-Push-Token      Push notification token
//	X-API-Version     API version (e.g., "v2")
//	X-Sync-Token      Sync state token
//
// Response headers set by the middleware:
//
//	X-API-Version     API version used
//	X-API-Deprecated  Deprecation warning
//	X-Sync-Token      New sync token
//	X-Min-App-Version Minimum required version
//
// # Mobile Client Integration
//
// iOS (Swift):
//
//	class APIClient {
//	    func request(_ endpoint: String) async throws -> Data {
//	        var request = URLRequest(url: URL(string: baseURL + endpoint)!)
//	        request.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")
//	        request.setValue(Bundle.main.appVersion, forHTTPHeaderField: "X-App-Version")
//	        request.setValue("v2", forHTTPHeaderField: "X-API-Version")
//	        return try await URLSession.shared.data(for: request).0
//	    }
//	}
//
// Android (Kotlin):
//
//	class ApiClient {
//	    fun createRequest(endpoint: String) = Request.Builder()
//	        .url("$baseUrl$endpoint")
//	        .header("X-Device-ID", deviceId)
//	        .header("X-App-Version", BuildConfig.VERSION_NAME)
//	        .header("X-API-Version", "v2")
//	        .build()
//	}
//
// React Native:
//
//	const headers = {
//	    'X-Device-ID': deviceId,
//	    'X-App-Version': DeviceInfo.getVersion(),
//	    'X-Platform': Platform.OS,
//	    'X-API-Version': 'v2',
//	};
//
// Flutter:
//
//	final headers = {
//	    'X-Device-ID': deviceId,
//	    'X-App-Version': packageInfo.version,
//	    'X-Platform': Platform.operatingSystem,
//	    'X-API-Version': 'v2',
//	};
package mobile
