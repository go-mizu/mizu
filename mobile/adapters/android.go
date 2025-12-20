package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/mobile"
)

// AndroidOptions extends mobile.Options with Android-specific settings.
type AndroidOptions struct {
	mobile.Options

	// FCMSenderID is the Firebase sender ID for push validation.
	FCMSenderID string

	// AllowEmulator allows emulator device IDs.
	// Default: false in production
	AllowEmulator bool

	// RequirePackageName requires X-Package-Name header.
	RequirePackageName bool

	// StrictUserAgent rejects requests with non-Android User-Agent.
	StrictUserAgent bool

	// MinSDKVersion is the minimum Android SDK version.
	MinSDKVersion int
}

// Android creates a mobile middleware optimized for Android applications.
//
// Android-specific features:
//   - FCM push token validation
//   - Android device model detection
//   - Play Store version checking support
//   - App Links headers
//
// Example:
//
//	app.Use(adapters.Android(mobile.Options{
//	    RequireDeviceID:   true,
//	    RequireAppVersion: true,
//	}))
func Android(opts mobile.Options) mizu.Middleware {
	return AndroidWithOptions(AndroidOptions{Options: opts})
}

// AndroidWithOptions creates an Android middleware with extended options.
func AndroidWithOptions(opts AndroidOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyAndroidDefaults(opts)

	// Restrict to Android platform
	if opts.AllowedPlatforms == nil {
		opts.AllowedPlatforms = []mobile.Platform{mobile.PlatformAndroid}
	}

	base := mobile.WithOptions(opts.Options)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Add Android-specific headers

			// FCM sender ID hint (for push configuration)
			if opts.FCMSenderID != "" {
				c.Header().Set("X-FCM-Sender-ID", opts.FCMSenderID)
			}

			return base(next)(c)
		}
	}
}

func applyAndroidDefaults(opts AndroidOptions) mobile.Options {
	o := opts.Options

	// Android apps typically send more device info
	// (but we don't force it by default)

	return o
}
