// Package adapters provides framework-specific frontend adapters for Mizu.
//
// Each adapter applies framework-specific defaults and optimizations
// while maintaining compatibility with the core frontend package.
//
// Usage:
//
//	// React with Vite
//	app.Use(adapters.React(frontend.Options{
//	    Root: "./dist",
//	}))
//
//	// Vue with default Vite settings
//	app.Use(adapters.Vue(frontend.Options{
//	    Root: "./dist",
//	}))
//
//	// Svelte with SvelteKit
//	app.Use(adapters.Svelte(frontend.Options{
//	    Root: "./build",
//	}))
package adapters

import (
	"github.com/go-mizu/mizu/frontend"
)

// applyCommonDefaults applies defaults common to all frameworks.
func applyCommonDefaults(opts frontend.Options) frontend.Options {
	// Common manifest location for Vite-based frameworks
	if opts.Manifest == "" {
		opts.Manifest = ".vite/manifest.json"
	}
	return opts
}
