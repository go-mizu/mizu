package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/middlewares/frontend"
)

// AstroOptions extends frontend.Options with Astro-specific settings.
type AstroOptions struct {
	frontend.Options

	// SSR indicates this is an Astro SSR application.
	// Default: false (static output)
	SSR bool

	// Hybrid indicates this is an Astro hybrid rendering application.
	// Some pages are pre-rendered, some are server-rendered.
	// Default: false
	Hybrid bool
}

// Astro creates a frontend middleware optimized for Astro applications.
//
// Astro-specific optimizations:
//   - Dev server: http://localhost:4321 (Astro default)
//   - Build output: dist/
//   - Handles Astro's partial hydration patterns
//
// Example:
//
//	app.Use(adapters.Astro(frontend.Options{
//	    Root: "./dist",
//	}))
func Astro(opts frontend.Options) mizu.Middleware {
	return AstroWithOptions(AstroOptions{Options: opts})
}

// AstroWithOptions creates an Astro middleware with extended options.
func AstroWithOptions(opts AstroOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyAstroDefaults(opts)

	return frontend.WithOptions(opts.Options)
}

func applyAstroDefaults(opts AstroOptions) frontend.Options {
	o := opts.Options

	// Astro uses port 4321 by default
	if o.DevServer == "" {
		o.DevServer = "http://localhost:4321"
	}

	// Default build output
	if o.Root == "" {
		o.Root = "dist"
	}

	// Astro-specific paths
	o.IgnorePaths = append(o.IgnorePaths, "/_astro")

	return o
}

// AstroSSR creates a frontend middleware for Astro SSR applications.
//
// Note: For full Astro SSR features, consider using Astro's Node adapter
// and proxying to Mizu for API routes instead.
func AstroSSR(opts frontend.Options) mizu.Middleware {
	if opts.Root == "" {
		opts.Root = "dist/client"
	}

	return AstroWithOptions(AstroOptions{
		Options: opts,
		SSR:     true,
	})
}
