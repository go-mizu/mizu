// Package mobile provides primitives for building mobile-optimized APIs with Mizu.
//
// It offers device detection, API versioning, pagination, caching helpers,
// offline sync primitives, and push notification token handling.
//
// # Quick Start
//
//	app := mizu.New()
//	app.Use(mobile.New())
//
//	app.Get("/api/profile", func(c *mizu.Ctx) error {
//	    device := mobile.DeviceFromCtx(c)
//	    if device.Platform.IsMobile() {
//	        // Mobile-specific logic
//	    }
//	    return c.JSON(200, profile)
//	})
//
// # Device Detection
//
// The middleware parses device information from HTTP headers:
//
//	app.Use(mobile.New())
//
//	// Or with options
//	app.Use(mobile.WithOptions(mobile.Options{
//	    RequireDeviceID:   true,
//	    RequireAppVersion: true,
//	    SkipUserAgent:     false, // User-Agent parsing enabled by default
//	}))
//
// Access device info in handlers:
//
//	device := mobile.DeviceFromCtx(c)
//	fmt.Println(device.Platform)   // "ios", "android", "web", "unknown"
//	fmt.Println(device.AppVersion) // From X-App-Version header
//	fmt.Println(device.DeviceID)   // From X-Device-ID header
//
// # API Versioning
//
// Version negotiation via headers:
//
//	app.Use(mobile.VersionMiddleware(mobile.VersionOptions{
//	    Default: mobile.Version{Major: 1},
//	    Supported: []mobile.Version{
//	        {Major: 1},
//	        {Major: 2},
//	    },
//	    Deprecated: []mobile.Version{
//	        {Major: 1},
//	    },
//	}))
//
//	app.Get("/api/users", func(c *mizu.Ctx) error {
//	    v := mobile.VersionFromCtx(c)
//	    if v.AtLeast(mobile.Version{Major: 2}) {
//	        return c.JSON(200, getUsersV2())
//	    }
//	    return c.JSON(200, getUsersV1())
//	})
//
// # Pagination
//
// Parse pagination from query params:
//
//	func listUsers(c *mizu.Ctx) error {
//	    page, perPage := mobile.Paginate(c) // ?page=1&per_page=20
//	    users, total := fetchUsers(page, perPage)
//	    return c.JSON(200, mobile.NewPage(users, page, perPage, total))
//	}
//
// Cursor-based pagination:
//
//	func listFeed(c *mizu.Ctx) error {
//	    cursor := mobile.Cursor(c) // ?cursor=abc123
//	    items, nextCursor := fetchFeed(cursor, 20)
//	    return c.JSON(200, mobile.NewCursorPage(items, 20, nextCursor))
//	}
//
// # Caching & ETags
//
// Enable conditional requests for bandwidth savings:
//
//	func getProfile(c *mizu.Ctx) error {
//	    profile := fetchProfile()
//	    etag := mobile.ETag(profile)
//	    if mobile.Conditional(c, etag) {
//	        return nil // 304 Not Modified sent
//	    }
//	    mobile.CachePrivate.Set(c)
//	    return c.JSON(200, profile)
//	}
//
// # Offline Sync
//
// Support offline-first mobile apps:
//
//	func syncItems(c *mizu.Ctx) error {
//	    syncToken := mobile.ParseSyncToken(c)
//	    lastSync, _ := mobile.ParseSyncTokenTime(syncToken)
//
//	    delta := mobile.NewSyncDelta[Item]()
//	    for _, item := range getChangesSince(lastSync) {
//	        if item.DeletedAt != nil {
//	            delta.AddDeleted(item.ID)
//	        } else if item.CreatedAt.After(lastSync) {
//	            delta.AddCreated(item)
//	        } else {
//	            delta.AddUpdated(item)
//	        }
//	    }
//
//	    return c.JSON(200, delta.ToSyncResponse(false))
//	}
//
// # Error Responses
//
// Structured error format for mobile clients:
//
//	if err := validate(input); err != nil {
//	    return mobile.SendError(c, 400, mobile.NewError(
//	        mobile.ErrCodeInvalidRequest,
//	        "Validation failed",
//	    ).WithDetails(map[string]string{
//	        "field": "email",
//	        "error": "invalid format",
//	    }))
//	}
//
// # Push Notifications
//
// Handle push token registration:
//
//	func registerToken(c *mizu.Ctx) error {
//	    var req mobile.PushTokenRequest
//	    if err := c.BindJSON(&req, 1<<10); err != nil {
//	        return err
//	    }
//	    token := req.ToPushToken(c)
//	    saveToken(token)
//	    return c.JSON(200, token)
//	}
//
// # Header Conventions
//
// Standard headers used by this package:
//
//	Request Headers:
//	  X-Device-ID      - Unique device identifier
//	  X-App-Version    - Client app version (e.g., "1.0.0")
//	  X-App-Build      - Client app build number
//	  X-Device-Model   - Device model (e.g., "iPhone15,2")
//	  X-Timezone       - Device timezone (IANA format)
//	  X-Push-Token     - Push notification token
//	  X-API-Version    - API version (e.g., "v1", "v2")
//	  X-Sync-Token     - Offline sync checkpoint
//
//	Response Headers:
//	  ETag             - Content hash for caching
//	  X-API-Deprecated - Version deprecation warning
//
// # Mobile Client Examples
//
// iOS (Swift):
//
//	var request = URLRequest(url: URL(string: "https://api.example.com/profile")!)
//	request.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")
//	request.setValue(appVersion, forHTTPHeaderField: "X-App-Version")
//	request.setValue(UIDevice.current.model, forHTTPHeaderField: "X-Device-Model")
//
// Android (Kotlin):
//
//	val request = Request.Builder()
//	    .url("https://api.example.com/profile")
//	    .header("X-Device-ID", deviceId)
//	    .header("X-App-Version", BuildConfig.VERSION_NAME)
//	    .header("X-Device-Model", Build.MODEL)
//	    .build()
//
// React Native:
//
//	fetch('https://api.example.com/profile', {
//	  headers: {
//	    'X-Device-ID': deviceId,
//	    'X-App-Version': DeviceInfo.getVersion(),
//	    'X-Device-Model': DeviceInfo.getModel(),
//	  },
//	});
package mobile
