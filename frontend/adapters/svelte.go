package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/frontend"
)

// SvelteOptions extends frontend.Options with Svelte-specific settings.
type SvelteOptions struct {
	frontend.Options

	// SvelteKit indicates this is a SvelteKit application.
	// Default: false (plain Svelte with Vite)
	SvelteKit bool
}

// Svelte creates a frontend middleware optimized for Svelte applications.
//
// Svelte-specific optimizations:
//   - Vite manifest path: .vite/manifest.json
//   - Dev server: http://localhost:5173 (Vite)
//   - Proper handling of Svelte's compiled output
//
// Example:
//
//	app.Use(adapters.Svelte(frontend.Options{
//	    Root:      "./dist",
//	    DevServer: "http://localhost:5173",
//	}))
func Svelte(opts frontend.Options) mizu.Middleware {
	return SvelteWithOptions(SvelteOptions{Options: opts})
}

// SvelteWithOptions creates a Svelte middleware with extended options.
func SvelteWithOptions(opts SvelteOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applySvelteDefaults(opts)

	return frontend.WithOptions(opts.Options)
}

func applySvelteDefaults(opts SvelteOptions) frontend.Options {
	o := opts.Options

	// Default to Vite dev server
	if o.DevServer == "" {
		o.DevServer = "http://localhost:5173"
	}

	// SvelteKit uses different build output
	if opts.SvelteKit {
		if o.Root == "" {
			o.Root = "build"
		}
		// SvelteKit has its own routing
		o.IgnorePaths = append(o.IgnorePaths, "/_app")
	}

	return o
}

// SvelteKit creates a frontend middleware optimized for SvelteKit applications.
//
// SvelteKit-specific optimizations:
//   - Dev server: http://localhost:5173
//   - Build output: build/
//   - Handles SvelteKit-specific paths (_app)
//
// Note: For full SvelteKit features with SSR, consider running SvelteKit
// as the primary server and proxying API routes to Mizu instead.
func SvelteKit(opts frontend.Options) mizu.Middleware {
	return SvelteWithOptions(SvelteOptions{
		Options:   opts,
		SvelteKit: true,
	})
}
