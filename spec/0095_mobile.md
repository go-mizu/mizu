# Mobile Package Design Specification

**Status:** Draft
**Author:** Mizu Team
**Created:** 2025-12-20

## Overview

The `mobile` package provides comprehensive mobile backend integration for Mizu. It enables Go backends to seamlessly support mobile applications across iOS, Android, Windows, and cross-platform frameworks (Flutter, React Native, Capacitor).

### Design Philosophy

1. **Minimal Core** - Small, focused API surface inspired by Go standard library
2. **Zero Dependencies** - Uses only Go standard library
3. **Composable Middleware** - Each feature is an independent, composable middleware
4. **Context-Based State** - All state flows through request context
5. **HTTP-First** - Uses standard HTTP headers as the primary protocol
6. **Opt-In Features** - Nothing enabled by default; compose what you need

### Key Features

| Feature | Description |
|---------|-------------|
| Device Detection | Parse platform, version, device ID from headers/User-Agent |
| API Versioning | Semantic versioning with deprecation support |
| Response Helpers | Pagination, cursors, structured errors, ETags |
| Offline Sync | Delta sync with tokens and conflict resolution |
| Push Tokens | APNS/FCM token management |
| Deep Linking | Universal links and app link verification |
| App Store | Version checking and forced update support |

## Package Structure

```
mobile/
├── mobile.go         # Core middleware (New, WithOptions)
├── device.go         # Platform/device detection (Device, Platform)
├── version.go        # API versioning (Version, VersionMiddleware)
├── response.go       # Response helpers (Page, Error, Cursor)
├── sync.go           # Offline sync (SyncToken, Delta)
├── push.go           # Push tokens (Token, APNS, FCM)
├── deeplink.go       # Deep linking (Link, Verify)
├── appstore.go       # App store version checks
├── doc.go            # Package documentation
├── mobile_test.go    # Comprehensive tests
└── adapters/         # Framework-specific adapters
    ├── adapters.go   # Common utilities
    ├── ios.go        # iOS/Swift hints
    ├── android.go    # Android/Kotlin hints
    ├── flutter.go    # Flutter integration
    ├── reactnative.go # React Native integration
    └── capacitor.go  # Capacitor/Ionic integration
```

## Core Types

### Platform

```go
// Platform represents a mobile operating system.
type Platform string

const (
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformWindows Platform = "windows"
    PlatformMacOS   Platform = "macos"
    PlatformWeb     Platform = "web"
    PlatformUnknown Platform = "unknown"
)

// Device returns the device family (iPhone, Android, Windows, etc.)
func (p Platform) Device() string

// IsMobile returns true for iOS, Android, Windows mobile.
func (p Platform) IsMobile() bool

// IsDesktop returns true for macOS, Windows desktop.
func (p Platform) IsDesktop() bool
```

### Device

```go
// Device contains information about the client device.
type Device struct {
    // Platform is the operating system (ios, android, windows, etc.)
    Platform Platform

    // OSVersion is the OS version (e.g., "17.0", "14.0")
    OSVersion string

    // AppVersion is the client app version (e.g., "1.2.3")
    AppVersion string

    // AppBuild is the build number (e.g., "123", "2024.01.15")
    AppBuild string

    // DeviceID is a unique device identifier
    DeviceID string

    // DeviceModel is the device model (e.g., "iPhone15,2", "Pixel 8")
    DeviceModel string

    // Locale is the device locale (e.g., "en-US", "ja-JP")
    Locale string

    // Timezone is the IANA timezone (e.g., "America/New_York")
    Timezone string

    // PushToken is the push notification token (if provided)
    PushToken string

    // PushProvider is APNS, FCM, or WNS
    PushProvider PushProvider

    // UserAgent is the raw User-Agent header
    UserAgent string
}

// DeviceFromCtx extracts Device from request context.
func DeviceFromCtx(c *mizu.Ctx) *Device

// deviceKey is unexported to prevent external modification.
type deviceKey struct{}
```

### Version

```go
// Version represents a semantic API version.
type Version struct {
    Major int
    Minor int
}

// ParseVersion parses "v1", "v1.2", "1", "1.2" formats.
func ParseVersion(s string) (Version, error)

// String returns "v1" or "v1.2" format.
func (v Version) String() string

// Compare returns -1, 0, or 1.
func (v Version) Compare(other Version) int

// AtLeast returns true if v >= other.
func (v Version) AtLeast(major, minor int) bool

// VersionFromCtx extracts Version from request context.
func VersionFromCtx(c *mizu.Ctx) Version
```

## Middleware Design

### Core Middleware

```go
// Options configures the mobile middleware.
type Options struct {
    // RequireDeviceID requires X-Device-ID header.
    // Default: false
    RequireDeviceID bool

    // RequireAppVersion requires X-App-Version header.
    // Default: false
    RequireAppVersion bool

    // AllowedPlatforms restricts to specific platforms.
    // Empty means all platforms allowed.
    AllowedPlatforms []Platform

    // MinAppVersion is the minimum required app version.
    // Requests below this version receive 426 Upgrade Required.
    MinAppVersion string

    // OnMissingHeader is called when required headers are missing.
    OnMissingHeader func(c *mizu.Ctx, header string) error

    // OnUnsupportedPlatform is called for disallowed platforms.
    OnUnsupportedPlatform func(c *mizu.Ctx, platform Platform) error

    // OnOutdatedApp is called when app version is below minimum.
    OnOutdatedApp func(c *mizu.Ctx, version, minimum string) error

    // SkipUserAgent skips User-Agent parsing (performance opt).
    // Default: false
    SkipUserAgent bool

    // TrustProxy trusts X-Forwarded-* headers.
    // Default: false
    TrustProxy bool
}

// New creates mobile middleware with default options.
func New() mizu.Middleware

// WithOptions creates mobile middleware with custom options.
func WithOptions(opts Options) mizu.Middleware
```

### API Version Middleware

```go
// VersionOptions configures API version middleware.
type VersionOptions struct {
    // Header is the version header name.
    // Default: "X-API-Version"
    Header string

    // QueryParam is an alternative query parameter.
    // Default: "" (disabled)
    QueryParam string

    // PathPrefix enables /v1/... URL versioning.
    // Default: false
    PathPrefix bool

    // Default is the default version when none specified.
    // Default: v1
    Default Version

    // Supported lists all supported versions.
    // Empty means no validation.
    Supported []Version

    // Deprecated lists deprecated versions (still work but warn).
    Deprecated []Version

    // OnUnsupported handles unsupported version requests.
    OnUnsupported func(c *mizu.Ctx, v Version) error
}

// VersionMiddleware creates API versioning middleware.
func VersionMiddleware(opts VersionOptions) mizu.Middleware
```

## Response Helpers

### Pagination

```go
// Page represents a paginated response.
type Page[T any] struct {
    Data       []T    `json:"data"`
    Page       int    `json:"page,omitempty"`
    PerPage    int    `json:"per_page,omitempty"`
    Total      int    `json:"total,omitempty"`
    TotalPages int    `json:"total_pages,omitempty"`
    HasMore    bool   `json:"has_more"`
    NextCursor string `json:"next_cursor,omitempty"`
    PrevCursor string `json:"prev_cursor,omitempty"`
}

// PageRequest extracts pagination from query parameters.
type PageRequest struct {
    Page    int    // ?page=1 (1-indexed)
    PerPage int    // ?per_page=20
    Cursor  string // ?cursor=xxx (for cursor pagination)
    After   string // ?after=xxx (alias for cursor)
    Before  string // ?before=xxx (reverse cursor)
}

// ParsePageRequest extracts pagination from request.
func ParsePageRequest(c *mizu.Ctx) PageRequest

// NewPage creates a page response.
func NewPage[T any](data []T, req PageRequest, total int) Page[T]

// NewCursorPage creates a cursor-based page response.
func NewCursorPage[T any](data []T, nextCursor, prevCursor string, hasMore bool) Page[T]
```

### Structured Errors

```go
// Error is a structured API error for mobile clients.
type Error struct {
    Code    string         `json:"code"`              // Machine-readable code
    Message string         `json:"message"`           // Human-readable message
    Details map[string]any `json:"details,omitempty"` // Additional context
    TraceID string         `json:"trace_id,omitempty"`// Request trace ID
    DocURL  string         `json:"doc_url,omitempty"` // Documentation link
}

// Standard error codes
const (
    ErrInvalidRequest   = "invalid_request"
    ErrUnauthorized     = "unauthorized"
    ErrForbidden        = "forbidden"
    ErrNotFound         = "not_found"
    ErrConflict         = "conflict"
    ErrRateLimited      = "rate_limited"
    ErrValidation       = "validation_error"
    ErrInternal         = "internal_error"
    ErrServiceDown      = "service_unavailable"
    ErrUpgradeRequired  = "upgrade_required"
    ErrMaintenance      = "maintenance"
)

// Error implements error interface.
func (e *Error) Error() string

// SendError sends a structured error response.
func SendError(c *mizu.Ctx, code int, err *Error) error

// NewError creates a new Error.
func NewError(code, message string) *Error

// WithDetails adds details to error.
func (e *Error) WithDetails(key string, value any) *Error

// WithTraceID adds trace ID to error.
func (e *Error) WithTraceID(id string) *Error
```

### ETag Support

```go
// ETag generates an ETag from data.
// Uses SHA-256 truncated to 16 chars for brevity.
func ETag(data any) string

// WeakETag generates a weak ETag (W/"...").
func WeakETag(data any) string

// CheckETag checks If-None-Match and returns true if matched (304).
func CheckETag(c *mizu.Ctx, etag string) bool

// Conditional sends 304 if ETag matches, otherwise sends data.
func Conditional(c *mizu.Ctx, data any) error
```

### Cache Control

```go
// CacheControl configures response caching.
type CacheControl struct {
    MaxAge       time.Duration
    Private      bool
    NoCache      bool
    NoStore      bool
    MustRevalidate bool
    Immutable    bool
}

// Cache presets
var (
    CachePrivate   = CacheControl{Private: true, MaxAge: 0}
    CacheShort     = CacheControl{MaxAge: 5 * time.Minute}
    CacheMedium    = CacheControl{MaxAge: 1 * time.Hour}
    CacheLong      = CacheControl{MaxAge: 24 * time.Hour}
    CacheImmutable = CacheControl{MaxAge: 365 * 24 * time.Hour, Immutable: true}
    CacheNone      = CacheControl{NoStore: true, NoCache: true}
)

// Apply sets Cache-Control header.
func (cc CacheControl) Apply(c *mizu.Ctx)

// String returns Cache-Control header value.
func (cc CacheControl) String() string
```

## Offline Sync

```go
// SyncToken is an opaque token representing sync state.
type SyncToken string

// NewSyncToken creates a token from timestamp.
func NewSyncToken(t time.Time) SyncToken

// Time extracts timestamp from token.
func (t SyncToken) Time() time.Time

// SyncRequest represents a sync request from client.
type SyncRequest struct {
    Token     SyncToken  // Last sync token (empty for initial)
    Resources []string   // Resources to sync (empty for all)
    FullSync  bool       // Force full resync
}

// ParseSyncRequest extracts sync request from headers/query.
func ParseSyncRequest(c *mizu.Ctx) SyncRequest

// SyncResponse wraps data with sync metadata.
type SyncResponse[T any] struct {
    Data      T         `json:"data"`
    SyncToken SyncToken `json:"sync_token"`
    HasMore   bool      `json:"has_more"`
    FullSync  bool      `json:"full_sync,omitempty"`
}

// Delta represents changes since last sync.
type Delta[T any] struct {
    Created []T      `json:"created,omitempty"`
    Updated []T      `json:"updated,omitempty"`
    Deleted []string `json:"deleted,omitempty"` // IDs only
}

// SyncDelta wraps a delta with sync metadata.
type SyncDelta[T any] struct {
    Delta[T]
    SyncToken SyncToken `json:"sync_token"`
    HasMore   bool      `json:"has_more"`
}

// NewSyncDelta creates a sync delta response.
func NewSyncDelta[T any](delta Delta[T], token SyncToken, hasMore bool) SyncDelta[T]
```

## Push Notifications

```go
// PushProvider is the push notification service.
type PushProvider string

const (
    PushAPNS PushProvider = "apns" // Apple Push Notification Service
    PushFCM  PushProvider = "fcm"  // Firebase Cloud Messaging
    PushWNS  PushProvider = "wns"  // Windows Notification Service
)

// PushToken represents a device push token.
type PushToken struct {
    Token     string       `json:"token"`
    Provider  PushProvider `json:"provider"`
    DeviceID  string       `json:"device_id"`
    Sandbox   bool         `json:"sandbox,omitempty"`  // APNS sandbox
    CreatedAt time.Time    `json:"created_at"`
}

// ParsePushToken extracts push token from request.
func ParsePushToken(c *mizu.Ctx) *PushToken

// ValidateAPNS validates APNS token format.
func ValidateAPNS(token string) bool

// ValidateFCM validates FCM token format.
func ValidateFCM(token string) bool
```

## Deep Linking

```go
// DeepLink represents a deep link configuration.
type DeepLink struct {
    // Scheme is the custom URL scheme (e.g., "myapp")
    Scheme string

    // Host is the universal link domain (e.g., "example.com")
    Host string

    // Paths are allowed deep link paths
    Paths []string

    // Fallback is the web fallback URL
    Fallback string
}

// AppleAppSiteAssociation generates apple-app-site-association.
func (d DeepLink) AppleAppSiteAssociation(teamID, bundleID string) []byte

// AssetLinks generates .well-known/assetlinks.json for Android.
func (d DeepLink) AssetLinks(packageName, fingerprint string) []byte

// DeepLinkMiddleware serves deep link verification files.
func DeepLinkMiddleware(links ...DeepLink) mizu.Middleware
```

## App Store Version Check

```go
// AppInfo contains app store information.
type AppInfo struct {
    CurrentVersion  string    `json:"current_version"`  // Latest store version
    MinimumVersion  string    `json:"minimum_version"`  // Min required version
    UpdateURL       string    `json:"update_url"`       // Store URL
    ReleaseNotes    string    `json:"release_notes,omitempty"`
    ReleasedAt      time.Time `json:"released_at,omitempty"`
    ForceUpdate     bool      `json:"force_update"`
    MaintenanceMode bool      `json:"maintenance_mode"`
    MaintenanceMsg  string    `json:"maintenance_message,omitempty"`
}

// AppInfoProvider fetches app info (implement for your backend).
type AppInfoProvider interface {
    GetAppInfo(ctx context.Context, platform Platform, bundleID string) (*AppInfo, error)
}

// AppInfoHandler creates an endpoint for app version checking.
func AppInfoHandler(provider AppInfoProvider) mizu.Handler

// CheckUpdate compares client version against store version.
type UpdateStatus struct {
    Available   bool   `json:"update_available"`
    Required    bool   `json:"update_required"`
    CurrentVer  string `json:"current_version"`
    LatestVer   string `json:"latest_version"`
    UpdateURL   string `json:"update_url"`
}

func CheckUpdate(client, latest, minimum string) UpdateStatus
```

## HTTP Header Conventions

### Request Headers

| Header | Description | Example |
|--------|-------------|---------|
| `X-Device-ID` | Unique device identifier | `550e8400-e29b-41d4-a716-446655440000` |
| `X-App-Version` | Client app version | `1.2.3` |
| `X-App-Build` | Build number | `2024.01.15` |
| `X-Device-Model` | Device model | `iPhone15,2` |
| `X-Platform` | Platform override | `ios` |
| `X-OS-Version` | OS version | `17.0` |
| `X-Timezone` | IANA timezone | `America/New_York` |
| `X-Locale` | Device locale | `en-US` |
| `X-Push-Token` | Push notification token | `abc123...` |
| `X-API-Version` | API version | `v2` |
| `X-Sync-Token` | Sync state token | `dG9rZW4...` |
| `X-Idempotency-Key` | Idempotency key | `req-123-abc` |

### Response Headers

| Header | Description | Example |
|--------|-------------|---------|
| `X-API-Version` | API version used | `v2` |
| `X-API-Deprecated` | Deprecation warning | `true` |
| `X-Sync-Token` | New sync token | `dG9rZW4...` |
| `X-RateLimit-Limit` | Rate limit ceiling | `100` |
| `X-RateLimit-Remaining` | Requests remaining | `95` |
| `X-RateLimit-Reset` | Reset timestamp | `1703001600` |
| `X-Request-ID` | Request trace ID | `req-abc-123` |
| `X-Min-App-Version` | Minimum required version | `1.0.0` |

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

    // Basic mobile middleware
    app.Use(mobile.New())

    // API versioning
    app.Use(mobile.VersionMiddleware(mobile.VersionOptions{
        Supported:  []mobile.Version{{1, 0}, {2, 0}},
        Deprecated: []mobile.Version{{1, 0}},
    }))

    app.Get("/api/users", listUsers)
    app.Listen(":3000")
}

func listUsers(c *mizu.Ctx) error {
    device := mobile.DeviceFromCtx(c)
    version := mobile.VersionFromCtx(c)

    // Platform-specific response
    if device.Platform == mobile.PlatformIOS {
        // iOS-specific logic
    }

    // Version-specific response
    if version.AtLeast(2, 0) {
        return c.JSON(200, v2Response)
    }
    return c.JSON(200, v1Response)
}
```

### With Required Headers

```go
app.Use(mobile.WithOptions(mobile.Options{
    RequireDeviceID:   true,
    RequireAppVersion: true,
    MinAppVersion:     "1.5.0",
    OnOutdatedApp: func(c *mizu.Ctx, v, min string) error {
        return mobile.SendError(c, 426, mobile.NewError(
            mobile.ErrUpgradeRequired,
            "Please update to the latest version",
        ).WithDetails("minimum_version", min))
    },
}))
```

### Pagination Example

```go
func listItems(c *mizu.Ctx) error {
    page := mobile.ParsePageRequest(c)

    items, total := db.ListItems(page.Page, page.PerPage)

    resp := mobile.NewPage(items, page, total)

    // Add ETag for caching
    etag := mobile.ETag(resp)
    if mobile.CheckETag(c, etag) {
        return nil // 304 Not Modified
    }
    c.Header().Set("ETag", etag)

    return c.JSON(200, resp)
}
```

### Offline Sync Example

```go
func syncItems(c *mizu.Ctx) error {
    req := mobile.ParseSyncRequest(c)

    var delta mobile.Delta[Item]
    if req.Token == "" || req.FullSync {
        // Initial sync - send everything
        delta.Created = db.GetAllItems()
    } else {
        // Delta sync
        since := req.Token.Time()
        delta.Created = db.GetCreatedSince(since)
        delta.Updated = db.GetUpdatedSince(since)
        delta.Deleted = db.GetDeletedSince(since)
    }

    token := mobile.NewSyncToken(time.Now())
    resp := mobile.NewSyncDelta(delta, token, false)

    return c.JSON(200, resp)
}
```

### Deep Link Setup

```go
link := mobile.DeepLink{
    Scheme:   "myapp",
    Host:     "example.com",
    Paths:    []string{"/share/*", "/invite/*"},
    Fallback: "https://example.com",
}

// Serve verification files
app.Use(mobile.DeepLinkMiddleware(link))

// Generates:
// - /.well-known/apple-app-site-association
// - /.well-known/assetlinks.json
```

## Mobile Client Examples

### iOS (Swift)

```swift
import Foundation

class APIClient {
    static let shared = APIClient()

    private var deviceID: String {
        // Use Keychain for persistence
        return KeychainService.getOrCreateDeviceID()
    }

    func request(_ endpoint: String) async throws -> Data {
        var request = URLRequest(url: URL(string: baseURL + endpoint)!)

        // Standard mobile headers
        request.setValue(deviceID, forHTTPHeaderField: "X-Device-ID")
        request.setValue(Bundle.main.appVersion, forHTTPHeaderField: "X-App-Version")
        request.setValue(Bundle.main.buildNumber, forHTTPHeaderField: "X-App-Build")
        request.setValue(UIDevice.current.model, forHTTPHeaderField: "X-Device-Model")
        request.setValue(TimeZone.current.identifier, forHTTPHeaderField: "X-Timezone")
        request.setValue(Locale.current.identifier, forHTTPHeaderField: "X-Locale")
        request.setValue("v2", forHTTPHeaderField: "X-API-Version")

        let (data, response) = try await URLSession.shared.data(for: request)

        // Handle deprecation warnings
        if let deprecated = (response as? HTTPURLResponse)?.value(forHTTPHeaderField: "X-API-Deprecated") {
            print("Warning: API version deprecated")
        }

        return data
    }
}
```

### Android (Kotlin)

```kotlin
class ApiClient(private val context: Context) {
    private val deviceId: String by lazy {
        PreferenceManager.getDeviceId(context)
    }

    fun createRequest(endpoint: String): Request {
        return Request.Builder()
            .url("$baseUrl$endpoint")
            .header("X-Device-ID", deviceId)
            .header("X-App-Version", BuildConfig.VERSION_NAME)
            .header("X-App-Build", BuildConfig.VERSION_CODE.toString())
            .header("X-Device-Model", Build.MODEL)
            .header("X-Timezone", TimeZone.getDefault().id)
            .header("X-Locale", Locale.getDefault().toLanguageTag())
            .header("X-API-Version", "v2")
            .build()
    }
}
```

### React Native

```typescript
import { Platform } from 'react-native';
import DeviceInfo from 'react-native-device-info';
import AsyncStorage from '@react-native-async-storage/async-storage';

const getDeviceHeaders = async () => {
  let deviceId = await AsyncStorage.getItem('deviceId');
  if (!deviceId) {
    deviceId = DeviceInfo.getUniqueId();
    await AsyncStorage.setItem('deviceId', deviceId);
  }

  return {
    'X-Device-ID': deviceId,
    'X-App-Version': DeviceInfo.getVersion(),
    'X-App-Build': DeviceInfo.getBuildNumber(),
    'X-Device-Model': DeviceInfo.getModel(),
    'X-Platform': Platform.OS,
    'X-OS-Version': Platform.Version.toString(),
    'X-Timezone': Intl.DateTimeFormat().resolvedOptions().timeZone,
    'X-Locale': Intl.DateTimeFormat().resolvedOptions().locale,
    'X-API-Version': 'v2',
  };
};
```

### Flutter

```dart
class ApiClient {
  static Future<Map<String, String>> getDeviceHeaders() async {
    final deviceInfo = DeviceInfoPlugin();
    final prefs = await SharedPreferences.getInstance();

    String? deviceId = prefs.getString('deviceId');
    if (deviceId == null) {
      deviceId = const Uuid().v4();
      await prefs.setString('deviceId', deviceId);
    }

    final packageInfo = await PackageInfo.fromPlatform();

    return {
      'X-Device-ID': deviceId,
      'X-App-Version': packageInfo.version,
      'X-App-Build': packageInfo.buildNumber,
      'X-Platform': Platform.operatingSystem,
      'X-Timezone': DateTime.now().timeZoneName,
      'X-Locale': Platform.localeName,
      'X-API-Version': 'v2',
    };
  }
}
```

## Adapters

Framework-specific adapters provide optimized defaults and platform hints:

```go
// iOS adapter with APNS support
app.Use(adapters.IOS(mobile.Options{
    RequireDeviceID: true,
}))

// Android adapter with FCM support
app.Use(adapters.Android(mobile.Options{
    RequireDeviceID: true,
}))

// Flutter adapter (handles both platforms)
app.Use(adapters.Flutter(mobile.Options{}))

// React Native adapter
app.Use(adapters.ReactNative(mobile.Options{}))
```

## Security Considerations

1. **Device ID Privacy** - Device IDs are opaque tokens, not tied to advertising IDs
2. **Token Validation** - Push tokens validated before storage
3. **Version Headers** - Cannot be spoofed for security decisions (auth is separate)
4. **Rate Limiting** - Per-device rate limiting supported via device ID
5. **HTTPS Required** - All mobile APIs should use HTTPS

## Future Considerations

1. **Binary Protocols** - Optional protobuf/msgpack support
2. **Compression** - Mobile-optimized compression (brotli)
3. **Batch Requests** - Multiple operations in single request
4. **GraphQL Support** - GraphQL adapter with mobile optimizations
5. **WebSocket Support** - Real-time sync over WebSocket
6. **Offline Queue** - Server-side offline operation queue

## Implementation Priority

### Phase 1: Core (MVP)
- Device detection middleware
- API versioning
- Response helpers (Page, Error)
- ETag/caching support

### Phase 2: Sync & Push
- Offline sync (SyncToken, Delta)
- Push token management

### Phase 3: Advanced
- Deep linking
- App store version check
- Framework adapters

## References

- [Apple Human Interface Guidelines - Mobile](https://developer.apple.com/design/human-interface-guidelines/)
- [Android Developer Guidelines](https://developer.android.com/guide)
- [React Native Best Practices](https://reactnative.dev/docs/performance)
- [Flutter Performance Best Practices](https://docs.flutter.dev/perf/best-practices)
