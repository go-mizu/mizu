package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/mobile"
)

// ReactNativeOptions extends mobile.Options with React Native-specific settings.
type ReactNativeOptions struct {
	mobile.Options

	// AllowExpo allows Expo-managed apps.
	// Default: true
	AllowExpo bool

	// AllowWeb allows React Native Web.
	// Default: false
	AllowWeb bool

	// RequireRNHeaders requires React Native-specific headers.
	RequireRNHeaders bool
}

// ReactNative creates a mobile middleware optimized for React Native applications.
//
// React Native-specific features:
//   - iOS and Android support
//   - Expo compatibility
//   - React Native header conventions
//
// Example:
//
//	app.Use(adapters.ReactNative(mobile.Options{
//	    RequireDeviceID: true,
//	}))
func ReactNative(opts mobile.Options) mizu.Middleware {
	return ReactNativeWithOptions(ReactNativeOptions{
		Options:   opts,
		AllowExpo: true,
	})
}

// ReactNativeWithOptions creates a React Native middleware with extended options.
func ReactNativeWithOptions(opts ReactNativeOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyReactNativeDefaults(opts)

	// React Native supports iOS, Android, and optionally Web
	if opts.AllowedPlatforms == nil {
		platforms := []mobile.Platform{
			mobile.PlatformIOS,
			mobile.PlatformAndroid,
		}
		if opts.AllowWeb {
			platforms = append(platforms, mobile.PlatformWeb)
		}
		opts.AllowedPlatforms = platforms
	}

	base := mobile.WithOptions(opts.Options)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Add React Native-specific headers

			// Indicate server supports RN conventions
			c.Header().Set("X-RN-Compatible", "true")

			return base(next)(c)
		}
	}
}

func applyReactNativeDefaults(opts ReactNativeOptions) mobile.Options {
	o := opts.Options

	// React Native uses consistent headers across platforms
	// The mobile package already handles this

	return o
}
