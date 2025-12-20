// Package adapters provides framework-specific mobile adapters for Mizu.
//
// Each adapter applies framework-specific defaults and optimizations
// while maintaining compatibility with the core mobile package.
//
// Usage:
//
//	// iOS app with APNS support
//	app.Use(adapters.IOS(mobile.Options{
//	    RequireDeviceID: true,
//	}))
//
//	// Android app with FCM support
//	app.Use(adapters.Android(mobile.Options{
//	    RequireDeviceID: true,
//	}))
//
//	// Flutter app (handles both platforms)
//	app.Use(adapters.Flutter(mobile.Options{}))
//
//	// React Native app
//	app.Use(adapters.ReactNative(mobile.Options{}))
package adapters

import (
	"github.com/go-mizu/mizu/mobile"
)

// applyCommonDefaults applies defaults common to all mobile frameworks.
func applyCommonDefaults(opts mobile.Options) mobile.Options {
	// Common skip paths for API routes
	if opts.SkipPaths == nil {
		opts.SkipPaths = []string{"/health", "/metrics", "/version"}
	}
	return opts
}
