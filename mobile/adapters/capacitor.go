package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/mobile"
)

// CapacitorOptions extends mobile.Options with Capacitor-specific settings.
type CapacitorOptions struct {
	mobile.Options

	// AllowWeb allows Capacitor Web/PWA mode.
	// Default: true
	AllowWeb bool

	// AllowElectron allows Capacitor Electron apps.
	// Default: false
	AllowElectron bool

	// CORSOrigins are allowed CORS origins for web/hybrid apps.
	CORSOrigins []string
}

// Capacitor creates a mobile middleware optimized for Capacitor/Ionic applications.
//
// Capacitor-specific features:
//   - iOS and Android native support
//   - Web/PWA support
//   - Ionic conventions
//   - CORS handling for hybrid apps
//
// Example:
//
//	app.Use(adapters.Capacitor(mobile.Options{
//	    RequireDeviceID: true,
//	}))
func Capacitor(opts mobile.Options) mizu.Middleware {
	return CapacitorWithOptions(CapacitorOptions{
		Options:  opts,
		AllowWeb: true,
	})
}

// CapacitorWithOptions creates a Capacitor middleware with extended options.
func CapacitorWithOptions(opts CapacitorOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyCapacitorDefaults(opts)

	// Capacitor supports iOS, Android, Web, and optionally Electron
	if opts.AllowedPlatforms == nil {
		platforms := []mobile.Platform{
			mobile.PlatformIOS,
			mobile.PlatformAndroid,
		}
		if opts.AllowWeb {
			platforms = append(platforms, mobile.PlatformWeb)
		}
		if opts.AllowElectron {
			platforms = append(platforms, mobile.PlatformWindows, mobile.PlatformMacOS)
		}
		opts.AllowedPlatforms = platforms
	}

	base := mobile.WithOptions(opts.Options)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Add Capacitor-specific headers

			// Indicate server supports Capacitor conventions
			c.Header().Set("X-Capacitor-Compatible", "true")

			return base(next)(c)
		}
	}
}

func applyCapacitorDefaults(opts CapacitorOptions) mobile.Options {
	o := opts.Options

	// Capacitor hybrid apps may send web-like requests
	// We're lenient with User-Agent parsing by default

	return o
}
