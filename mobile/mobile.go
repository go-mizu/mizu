package mobile

import (
	"context"
	"net/http"

	"github.com/go-mizu/mizu"
)

// Standard mobile headers.
const (
	HeaderDeviceID    = "X-Device-ID"
	HeaderAppVersion  = "X-App-Version"
	HeaderAppBuild    = "X-App-Build"
	HeaderDeviceModel = "X-Device-Model"
	HeaderPlatform    = "X-Platform"
	HeaderOSVersion   = "X-OS-Version"
	HeaderTimezone    = "X-Timezone"
	HeaderLocale      = "X-Locale"
	HeaderPushToken   = "X-Push-Token"
	HeaderAPIVersion  = "X-API-Version"
	HeaderSyncToken   = "X-Sync-Token"
	HeaderIdempotency = "X-Idempotency-Key"
	HeaderRequestID   = "X-Request-ID"
	HeaderMinVersion  = "X-Min-App-Version"
	HeaderDeprecated  = "X-API-Deprecated"
)

// Options configures the mobile middleware.
type Options struct {
	// RequireDeviceID requires X-Device-ID header.
	RequireDeviceID bool

	// RequireAppVersion requires X-App-Version header.
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

	// SkipUserAgent skips User-Agent parsing (performance optimization).
	SkipUserAgent bool

	// SkipPaths bypasses mobile middleware for these path prefixes.
	SkipPaths []string
}

// deviceKey is unexported to prevent external modification.
type deviceKey struct{}

// DeviceFromCtx extracts Device from request context.
// Returns nil if middleware was not applied.
func DeviceFromCtx(c *mizu.Ctx) *Device {
	if d, ok := c.Context().Value(deviceKey{}).(*Device); ok {
		return d
	}
	return nil
}

// New creates mobile middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates mobile middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Check skip paths
			path := c.Request().URL.Path
			for _, skip := range opts.SkipPaths {
				if len(path) >= len(skip) && path[:len(skip)] == skip {
					return next(c)
				}
			}

			// Parse device information
			device := parseDevice(c.Request(), opts)

			// Validate required headers
			if opts.RequireDeviceID && device.DeviceID == "" {
				if opts.OnMissingHeader != nil {
					return opts.OnMissingHeader(c, HeaderDeviceID)
				}
				return defaultMissingHeader(c, HeaderDeviceID)
			}

			if opts.RequireAppVersion && device.AppVersion == "" {
				if opts.OnMissingHeader != nil {
					return opts.OnMissingHeader(c, HeaderAppVersion)
				}
				return defaultMissingHeader(c, HeaderAppVersion)
			}

			// Validate platform
			if len(opts.AllowedPlatforms) > 0 {
				allowed := false
				for _, p := range opts.AllowedPlatforms {
					if device.Platform == p {
						allowed = true
						break
					}
				}
				if !allowed {
					if opts.OnUnsupportedPlatform != nil {
						return opts.OnUnsupportedPlatform(c, device.Platform)
					}
					return defaultUnsupportedPlatform(c, device.Platform)
				}
			}

			// Validate minimum version
			if opts.MinAppVersion != "" && device.AppVersion != "" {
				if compareVersions(device.AppVersion, opts.MinAppVersion) < 0 {
					// Set response header for client
					c.Header().Set(HeaderMinVersion, opts.MinAppVersion)

					if opts.OnOutdatedApp != nil {
						return opts.OnOutdatedApp(c, device.AppVersion, opts.MinAppVersion)
					}
					return defaultOutdatedApp(c, device.AppVersion, opts.MinAppVersion)
				}
			}

			// Store device in context
			ctx := context.WithValue(c.Context(), deviceKey{}, device)
			*c.Request() = *c.Request().WithContext(ctx)

			return next(c)
		}
	}
}

// Default handlers for validation failures.

func defaultMissingHeader(c *mizu.Ctx, header string) error {
	return SendError(c, http.StatusBadRequest, NewError(
		ErrInvalidRequest,
		"Missing required header: "+header,
	).WithDetails("header", header))
}

func defaultUnsupportedPlatform(c *mizu.Ctx, platform Platform) error {
	return SendError(c, http.StatusBadRequest, NewError(
		ErrInvalidRequest,
		"Unsupported platform: "+platform.String(),
	).WithDetails("platform", platform.String()))
}

func defaultOutdatedApp(c *mizu.Ctx, version, minimum string) error {
	return SendError(c, http.StatusUpgradeRequired, NewError(
		ErrUpgradeRequired,
		"Please update to the latest version",
	).WithDetails("current_version", version).
		WithDetails("minimum_version", minimum))
}

// compareVersions compares semantic version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	av := parseVersionParts(a)
	bv := parseVersionParts(b)

	for i := 0; i < len(av) || i < len(bv); i++ {
		var ai, bi int
		if i < len(av) {
			ai = av[i]
		}
		if i < len(bv) {
			bi = bv[i]
		}

		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}

// parseVersionParts splits version string into integer parts.
func parseVersionParts(v string) []int {
	var parts []int
	var current int
	for _, c := range v {
		if c >= '0' && c <= '9' {
			current = current*10 + int(c-'0')
		} else if c == '.' || c == '-' || c == '+' {
			parts = append(parts, current)
			current = 0
			// Stop at prerelease/build metadata
			if c == '-' || c == '+' {
				break
			}
		}
	}
	parts = append(parts, current)
	return parts
}
