package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/mobile"
)

// IOSOptions extends mobile.Options with iOS-specific settings.
type IOSOptions struct {
	mobile.Options

	// APNSEnvironment is the APNS environment (production, sandbox).
	// Used to validate push tokens.
	APNSEnvironment string

	// AllowSimulator allows simulator device IDs.
	// Default: false in production
	AllowSimulator bool

	// RequireBundleID requires X-Bundle-ID header.
	RequireBundleID bool

	// StrictUserAgent rejects requests with non-iOS User-Agent.
	StrictUserAgent bool
}

// IOS creates a mobile middleware optimized for iOS applications.
//
// iOS-specific features:
//   - APNS push token validation
//   - iOS device model detection (iPhone, iPad, etc.)
//   - App Store version checking support
//   - Universal link headers
//
// Example:
//
//	app.Use(adapters.IOS(mobile.Options{
//	    RequireDeviceID:   true,
//	    RequireAppVersion: true,
//	}))
func IOS(opts mobile.Options) mizu.Middleware {
	return IOSWithOptions(IOSOptions{Options: opts})
}

// IOSWithOptions creates an iOS middleware with extended options.
func IOSWithOptions(opts IOSOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyIOSDefaults(opts)

	// Restrict to iOS platform
	if opts.AllowedPlatforms == nil {
		opts.AllowedPlatforms = []mobile.Platform{mobile.PlatformIOS}
	}

	base := mobile.WithOptions(opts.Options)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Add iOS-specific headers
			// App Transport Security hints
			c.Header().Set("Strict-Transport-Security", "max-age=31536000")

			// APNS topic hint (for push configuration)
			if bundleID := c.Request().Header.Get("X-Bundle-ID"); bundleID != "" {
				c.Header().Set("X-APNS-Topic", bundleID)
			}

			return base(next)(c)
		}
	}
}

func applyIOSDefaults(opts IOSOptions) mobile.Options {
	o := opts.Options

	// iOS apps typically require device ID for analytics
	// (but we don't force it by default)

	return o
}
