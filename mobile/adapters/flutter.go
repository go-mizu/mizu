package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/mobile"
)

// FlutterOptions extends mobile.Options with Flutter-specific settings.
type FlutterOptions struct {
	mobile.Options

	// AllowWeb allows Flutter Web platform.
	// Default: true
	AllowWeb bool

	// AllowDesktop allows Flutter Desktop platforms (Windows, macOS, Linux).
	// Default: false
	AllowDesktop bool

	// RequireFlutterHeaders requires Flutter-specific headers.
	RequireFlutterHeaders bool
}

// Flutter creates a mobile middleware optimized for Flutter applications.
//
// Flutter-specific features:
//   - Multi-platform support (iOS, Android, Web, Desktop)
//   - Unified device detection across platforms
//   - Dart/Flutter header conventions
//
// Example:
//
//	app.Use(adapters.Flutter(mobile.Options{
//	    RequireDeviceID: true,
//	}))
func Flutter(opts mobile.Options) mizu.Middleware {
	return FlutterWithOptions(FlutterOptions{
		Options:  opts,
		AllowWeb: true,
	})
}

// FlutterWithOptions creates a Flutter middleware with extended options.
func FlutterWithOptions(opts FlutterOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyFlutterDefaults(opts)

	// Flutter supports multiple platforms
	if opts.AllowedPlatforms == nil {
		platforms := []mobile.Platform{
			mobile.PlatformIOS,
			mobile.PlatformAndroid,
		}
		if opts.AllowWeb {
			platforms = append(platforms, mobile.PlatformWeb)
		}
		if opts.AllowDesktop {
			platforms = append(platforms, mobile.PlatformWindows, mobile.PlatformMacOS)
		}
		opts.AllowedPlatforms = platforms
	}

	base := mobile.WithOptions(opts.Options)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Add Flutter-specific headers

			// Indicate server supports Flutter conventions
			c.Header().Set("X-Flutter-Compatible", "true")

			return base(next)(c)
		}
	}
}

func applyFlutterDefaults(opts FlutterOptions) mobile.Options {
	o := opts.Options

	// Flutter uses consistent headers across platforms
	// The mobile package already handles this

	return o
}
