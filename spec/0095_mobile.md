# Mobile Package Design

This document specifies the design for `github.com/go-mizu/mizu/mobile`, a package for building mobile-optimized APIs with Mizu.

## Overview

The mobile package provides primitives for building APIs that serve mobile clients (iOS, Android, React Native, Flutter, Expo). It focuses on:

1. **Device Context** - Parse and expose device information from request headers
2. **API Versioning** - Version negotiation via headers or paths
3. **Response Helpers** - Mobile-optimized JSON responses with pagination, caching hints
4. **Offline Support** - ETags, conditional requests, sync timestamps
5. **Push Notifications** - Device token registration patterns

## Design Principles

- **Minimal Core** - Small API surface, composable primitives
- **Zero Dependencies** - Standard library only
- **Go Idioms** - Follow net/http and Go stdlib patterns
- **Opt-in Features** - Each feature is independently usable
- **Safe Defaults** - Secure by default, explicit overrides

## Package Structure

```
mobile/
├── mobile.go       # Core types and context middleware
├── device.go       # Device detection and parsing
├── version.go      # API versioning middleware
├── response.go     # Response helpers (pagination, errors)
├── sync.go         # Offline sync primitives
├── push.go         # Push notification token helpers
├── doc.go          # Package documentation
└── mobile_test.go  # Comprehensive tests
```

## Core Types

### Device

```go
// Device represents a mobile device making a request.
// Extracted from User-Agent and custom headers.
type Device struct {
    // Platform is the OS: "ios", "android", "web", or "unknown"
    Platform Platform

    // Version is the OS version (e.g., "17.2", "14.0")
    Version string

    // AppVersion is the client app version from X-App-Version header
    AppVersion string

    // AppBuild is the client app build number from X-App-Build header
    AppBuild string

    // DeviceID is a unique device identifier from X-Device-ID header
    DeviceID string

    // DeviceModel is the device model from X-Device-Model header (e.g., "iPhone15,2")
    DeviceModel string

    // Locale is the device locale from Accept-Language header
    Locale string

    // Timezone is the device timezone from X-Timezone header
    Timezone string

    // PushToken is the push notification token from X-Push-Token header
    PushToken string
}

// Platform represents the mobile operating system.
type Platform string

const (
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformWeb     Platform = "web"
    PlatformUnknown Platform = "unknown"
)

// Is checks if platform matches.
func (p Platform) Is(other Platform) bool { return p == other }

// IsMobile returns true if platform is iOS or Android.
func (p Platform) IsMobile() bool { return p == PlatformIOS || p == PlatformAndroid }
```

### Context Key

```go
// deviceKey is the context key for device information.
type deviceKey struct{}

// DeviceFromContext extracts Device from request context.
// Returns zero Device if not present.
func DeviceFromContext(ctx context.Context) Device {
    if d, ok := ctx.Value(deviceKey{}).(Device); ok {
        return d
    }
    return Device{}
}

// DeviceFromCtx is a convenience wrapper for Mizu handlers.
func DeviceFromCtx(c *mizu.Ctx) Device {
    return DeviceFromContext(c.Context())
}
```

## Middleware

### Device Detection

```go
// New creates middleware that parses device information from headers.
// Device info is stored in request context and accessible via DeviceFromCtx.
func New() mizu.Middleware {
    return WithOptions(Options{})
}

// Options configures device detection middleware.
type Options struct {
    // RequireDeviceID rejects requests without X-Device-ID header.
    // Default: false
    RequireDeviceID bool

    // RequireAppVersion rejects requests without X-App-Version header.
    // Default: false
    RequireAppVersion bool

    // OnMissingHeader is called when required headers are missing.
    // Default: returns 400 Bad Request
    OnMissingHeader func(c *mizu.Ctx, header string) error

    // SkipUserAgent disables User-Agent parsing for platform detection.
    // Default: false (User-Agent parsing is enabled by default)
    SkipUserAgent bool
}

// WithOptions creates middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
    return func(next mizu.Handler) mizu.Handler {
        return func(c *mizu.Ctx) error {
            device := parseDevice(c.Request(), opts)

            if opts.RequireDeviceID && device.DeviceID == "" {
                return handleMissing(c, opts, HeaderDeviceID)
            }
            if opts.RequireAppVersion && device.AppVersion == "" {
                return handleMissing(c, opts, HeaderAppVersion)
            }

            ctx := context.WithValue(c.Context(), deviceKey{}, device)
            *c.Request() = *c.Request().WithContext(ctx)

            return next(c)
        }
    }
}
```

### API Versioning

```go
// Version represents an API version.
type Version struct {
    Major int
    Minor int
}

// String returns "vN" or "vN.M" format.
func (v Version) String() string {
    if v.Minor == 0 {
        return fmt.Sprintf("v%d", v.Major)
    }
    return fmt.Sprintf("v%d.%d", v.Major, v.Minor)
}

// VersionOptions configures version detection.
type VersionOptions struct {
    // Header is the header name for version negotiation.
    // Default: "X-API-Version" or "Accept-Version"
    Header string

    // Default is the default version when none specified.
    // Default: Version{Major: 1}
    Default Version

    // Supported lists supported versions.
    // If empty, all versions are accepted.
    Supported []Version

    // Deprecated lists deprecated versions.
    // Responses include X-API-Deprecated header.
    Deprecated []Version

    // OnUnsupported handles unsupported version requests.
    // Default: returns 400 with error message
    OnUnsupported func(c *mizu.Ctx, v Version) error
}

// VersionMiddleware parses API version from headers.
func VersionMiddleware(opts VersionOptions) mizu.Middleware

// VersionFromCtx extracts Version from request context.
func VersionFromCtx(c *mizu.Ctx) Version
```

## Response Helpers

### Page

```go
// Page represents a paginated response.
type Page[T any] struct {
    Items      []T    `json:"items"`
    Total      int    `json:"total,omitempty"`
    Page       int    `json:"page"`
    PerPage    int    `json:"per_page"`
    HasMore    bool   `json:"has_more"`
    NextCursor string `json:"next_cursor,omitempty"`
}

// NewPage creates a paginated response.
func NewPage[T any](items []T, page, perPage, total int) Page[T] {
    hasMore := page*perPage < total
    return Page[T]{
        Items:   items,
        Total:   total,
        Page:    page,
        PerPage: perPage,
        HasMore: hasMore,
    }
}

// Paginate parses pagination params from query string.
func Paginate(c *mizu.Ctx) (page, perPage int) {
    page, _ = strconv.Atoi(c.Query("page"))
    perPage, _ = strconv.Atoi(c.Query("per_page"))

    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }
    return
}
```

### API Error

```go
// Error represents a structured API error.
type Error struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
    TraceID string            `json:"trace_id,omitempty"`
}

// Error implements the error interface.
func (e Error) Error() string { return e.Message }

// Common error codes
const (
    ErrCodeInvalidRequest   = "invalid_request"
    ErrCodeUnauthorized     = "unauthorized"
    ErrCodeForbidden        = "forbidden"
    ErrCodeNotFound         = "not_found"
    ErrCodeConflict         = "conflict"
    ErrCodeRateLimited      = "rate_limited"
    ErrCodeServerError      = "server_error"
    ErrCodeMaintenance      = "maintenance"
    ErrCodeUpgradeRequired  = "upgrade_required"
)

// NewError creates an API error.
func NewError(code, message string) Error {
    return Error{Code: code, Message: message}
}

// WithDetails adds details to an error.
func (e Error) WithDetails(details map[string]string) Error {
    e.Details = details
    return e
}

// WithTraceID adds trace ID to an error.
func (e Error) WithTraceID(traceID string) Error {
    e.TraceID = traceID
    return e
}

// SendError sends an error response with appropriate status code.
func SendError(c *mizu.Ctx, status int, err Error) error {
    return c.JSON(status, map[string]Error{"error": err})
}
```

### Conditional Responses

```go
// ETag generates an ETag from data.
func ETag(data any) string {
    h := sha256.New()
    json.NewEncoder(h).Encode(data)
    return fmt.Sprintf(`"%x"`, h.Sum(nil)[:8])
}

// Conditional checks If-None-Match and returns 304 if matched.
// Returns true if 304 was sent (caller should return early).
func Conditional(c *mizu.Ctx, etag string) bool {
    c.Header().Set("ETag", etag)

    if match := c.Request().Header.Get("If-None-Match"); match != "" {
        if match == etag || match == "*" {
            c.Writer().WriteHeader(http.StatusNotModified)
            return true
        }
    }
    return false
}

// CacheControl sets Cache-Control header for mobile responses.
type CacheControl struct {
    MaxAge   time.Duration
    Private  bool
    NoStore  bool
    MustRevalidate bool
}

// Set applies cache control header to response.
func (cc CacheControl) Set(c *mizu.Ctx) {
    var parts []string

    if cc.NoStore {
        parts = append(parts, "no-store")
    } else {
        if cc.Private {
            parts = append(parts, "private")
        } else {
            parts = append(parts, "public")
        }
        if cc.MaxAge > 0 {
            parts = append(parts, fmt.Sprintf("max-age=%d", int(cc.MaxAge.Seconds())))
        }
        if cc.MustRevalidate {
            parts = append(parts, "must-revalidate")
        }
    }

    c.Header().Set("Cache-Control", strings.Join(parts, ", "))
}
```

## Sync Primitives

```go
// SyncState represents offline sync state.
type SyncState struct {
    LastSync  time.Time `json:"last_sync"`
    SyncToken string    `json:"sync_token"`
    HasMore   bool      `json:"has_more"`
}

// SyncResponse wraps data with sync metadata.
type SyncResponse[T any] struct {
    Data      T         `json:"data"`
    SyncState SyncState `json:"sync_state"`
    Deleted   []string  `json:"deleted,omitempty"`
}

// ParseSyncToken extracts sync token from header or query.
func ParseSyncToken(c *mizu.Ctx) string {
    if token := c.Request().Header.Get("X-Sync-Token"); token != "" {
        return token
    }
    return c.Query("sync_token")
}

// NewSyncToken generates a sync token from timestamp.
func NewSyncToken(t time.Time) string {
    return base64.RawURLEncoding.EncodeToString(
        []byte(strconv.FormatInt(t.UnixNano(), 36)),
    )
}

// ParseSyncTokenTime decodes a sync token to timestamp.
func ParseSyncTokenTime(token string) (time.Time, error) {
    b, err := base64.RawURLEncoding.DecodeString(token)
    if err != nil {
        return time.Time{}, err
    }
    ns, err := strconv.ParseInt(string(b), 36, 64)
    if err != nil {
        return time.Time{}, err
    }
    return time.Unix(0, ns), nil
}
```

## Push Notification Helpers

```go
// TokenType represents push notification service.
type TokenType string

const (
    TokenAPNS TokenType = "apns"  // Apple Push Notification Service
    TokenFCM  TokenType = "fcm"   // Firebase Cloud Messaging
)

// PushToken represents a registered push token.
type PushToken struct {
    Token     string    `json:"token"`
    Type      TokenType `json:"type"`
    DeviceID  string    `json:"device_id"`
    CreatedAt time.Time `json:"created_at"`
}

// ParsePushToken extracts push token from request.
func ParsePushToken(c *mizu.Ctx) PushToken {
    device := DeviceFromCtx(c)

    token := c.Request().Header.Get("X-Push-Token")
    if token == "" {
        return PushToken{}
    }

    tokenType := TokenFCM
    if device.Platform == PlatformIOS {
        tokenType = TokenAPNS
    }

    return PushToken{
        Token:     token,
        Type:      tokenType,
        DeviceID:  device.DeviceID,
        CreatedAt: time.Now(),
    }
}
```

## Usage Examples

### Basic Setup

```go
package main

import (
    "github.com/go-mizu/mizu"
    "github.com/go-mizu/mizu/mobile"
)

func main() {
    app := mizu.New()

    // Add device detection middleware
    app.Use(mobile.New())

    // API routes
    app.Get("/api/v1/profile", getProfile)
    app.Get("/api/v1/items", listItems)

    app.Listen(":3000")
}

func getProfile(c *mizu.Ctx) error {
    device := mobile.DeviceFromCtx(c)

    profile := getProfileData(c)

    // ETag for conditional responses
    etag := mobile.ETag(profile)
    if mobile.Conditional(c, etag) {
        return nil // 304 sent
    }

    // Cache for 5 minutes on mobile
    if device.Platform.IsMobile() {
        mobile.CacheControl{
            MaxAge:  5 * time.Minute,
            Private: true,
        }.Set(c)
    }

    return c.JSON(200, profile)
}

func listItems(c *mizu.Ctx) error {
    page, perPage := mobile.Paginate(c)

    items, total := fetchItems(page, perPage)

    return c.JSON(200, mobile.NewPage(items, page, perPage, total))
}
```

### API Versioning

```go
app := mizu.New()

app.Use(mobile.VersionMiddleware(mobile.VersionOptions{
    Default: mobile.Version{Major: 1},
    Supported: []mobile.Version{
        {Major: 1},
        {Major: 2},
    },
    Deprecated: []mobile.Version{
        {Major: 1},
    },
}))

app.Get("/api/users", func(c *mizu.Ctx) error {
    v := mobile.VersionFromCtx(c)

    switch v.Major {
    case 2:
        return c.JSON(200, getUsersV2())
    default:
        return c.JSON(200, getUsersV1())
    }
})
```

### Offline Sync

```go
func syncItems(c *mizu.Ctx) error {
    // Get last sync time from token
    syncToken := mobile.ParseSyncToken(c)
    lastSync, _ := mobile.ParseSyncTokenTime(syncToken)

    // Get changes since last sync
    items, deleted := getChangesSince(lastSync)

    now := time.Now()
    return c.JSON(200, mobile.SyncResponse[[]Item]{
        Data: items,
        Deleted: deleted,
        SyncState: mobile.SyncState{
            LastSync:  now,
            SyncToken: mobile.NewSyncToken(now),
            HasMore:   false,
        },
    })
}
```

### Error Handling

```go
func updateUser(c *mizu.Ctx) error {
    var input UpdateUserInput
    if err := c.BindJSON(&input, 1<<20); err != nil {
        return mobile.SendError(c, 400, mobile.NewError(
            mobile.ErrCodeInvalidRequest,
            "Invalid request body",
        ).WithDetails(map[string]string{
            "field": "body",
            "error": err.Error(),
        }))
    }

    // Validate
    if input.Email == "" {
        return mobile.SendError(c, 400, mobile.NewError(
            mobile.ErrCodeInvalidRequest,
            "Email is required",
        ).WithDetails(map[string]string{
            "field": "email",
        }))
    }

    // Update...
    return c.JSON(200, user)
}
```

### Minimum Version Check

```go
// RequireMinVersion creates middleware that rejects old app versions.
func RequireMinVersion(minVersion string) mizu.Middleware {
    return func(next mizu.Handler) mizu.Handler {
        return func(c *mizu.Ctx) error {
            device := mobile.DeviceFromCtx(c)

            if device.AppVersion != "" && !isVersionSupported(device.AppVersion, minVersion) {
                return mobile.SendError(c, 426, mobile.NewError(
                    mobile.ErrCodeUpgradeRequired,
                    "Please update your app to continue",
                ).WithDetails(map[string]string{
                    "min_version": minVersion,
                    "current_version": device.AppVersion,
                }))
            }

            return next(c)
        }
    }
}
```

## Header Conventions

The package uses standard and custom headers:

| Header | Direction | Description |
|--------|-----------|-------------|
| `User-Agent` | Request | Platform detection fallback |
| `Accept-Language` | Request | Locale preference |
| `X-Device-ID` | Request | Unique device identifier |
| `X-App-Version` | Request | Client app version |
| `X-App-Build` | Request | Client app build number |
| `X-Device-Model` | Request | Device model identifier |
| `X-Timezone` | Request | Device timezone (IANA) |
| `X-Push-Token` | Request | Push notification token |
| `X-API-Version` | Request | API version negotiation |
| `X-Sync-Token` | Both | Offline sync checkpoint |
| `ETag` | Response | Content hash for caching |
| `X-API-Deprecated` | Response | Deprecation warning |
| `X-Request-ID` | Response | Request trace ID |

## Mobile Client Examples

### iOS (Swift)

```swift
class APIClient {
    var baseURL = "https://api.example.com"
    var deviceID: String
    var appVersion: String

    func request(_ path: String) async throws -> Data {
        var request = URLRequest(url: URL(string: baseURL + path)!)
        request.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")
        request.setValue(appVersion, forHTTPHeaderField: "X-App-Version")
        request.setValue(UIDevice.current.model, forHTTPHeaderField: "X-Device-Model")
        request.setValue(TimeZone.current.identifier, forHTTPHeaderField: "X-Timezone")

        let (data, _) = try await URLSession.shared.data(for: request)
        return data
    }
}
```

### Android (Kotlin)

```kotlin
class APIClient(
    private val baseUrl: String,
    private val deviceId: String,
    private val appVersion: String
) {
    private val client = OkHttpClient.Builder()
        .addInterceptor { chain ->
            val request = chain.request().newBuilder()
                .header("X-Device-ID", deviceId)
                .header("X-App-Version", appVersion)
                .header("X-Device-Model", Build.MODEL)
                .header("X-Timezone", TimeZone.getDefault().id)
                .build()
            chain.proceed(request)
        }
        .build()
}
```

### React Native

```typescript
const api = {
  baseURL: 'https://api.example.com',
  deviceId: await DeviceInfo.getUniqueId(),
  appVersion: DeviceInfo.getVersion(),

  async fetch(path: string, options: RequestInit = {}) {
    return fetch(this.baseURL + path, {
      ...options,
      headers: {
        'X-Device-ID': this.deviceId,
        'X-App-Version': this.appVersion,
        'X-Device-Model': DeviceInfo.getModel(),
        'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone,
        ...options.headers,
      },
    });
  },
};
```

## Testing

The package should be fully testable without external dependencies:

```go
func TestDeviceDetection(t *testing.T) {
    app := mizu.NewRouter()
    app.Use(mobile.New())
    app.Get("/test", func(c *mizu.Ctx) error {
        device := mobile.DeviceFromCtx(c)
        return c.JSON(200, device)
    })

    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("User-Agent", "MyApp/1.0 (iPhone; iOS 17.2)")
    req.Header.Set("X-Device-ID", "abc123")
    req.Header.Set("X-App-Version", "1.0.0")

    rec := httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    var device mobile.Device
    json.Unmarshal(rec.Body.Bytes(), &device)

    assert.Equal(t, mobile.PlatformIOS, device.Platform)
    assert.Equal(t, "abc123", device.DeviceID)
    assert.Equal(t, "1.0.0", device.AppVersion)
}
```

## Future Considerations

Not in initial scope but possible additions:

1. **Biometric Auth Hints** - Headers for Face ID/Touch ID state
2. **Network Quality** - Headers for connection type (wifi, cellular, quality)
3. **Feature Flags** - Device-aware feature flag evaluation
4. **Analytics Batching** - Helpers for batched event uploads
5. **Deep Link Handling** - Universal/App link helpers

## Implementation Notes

1. All parsing should be lenient (don't fail on malformed headers)
2. User-Agent parsing should handle common mobile patterns
3. Timezone should be validated against IANA database
4. Version comparison should handle semver and simple numbers
5. All context values should use unexported key types
6. No globals, all state passed explicitly
