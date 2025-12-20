package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/frontend"
)

// SolidOptions extends frontend.Options with Solid-specific settings.
type SolidOptions struct {
	frontend.Options

	// SolidStart indicates this is a SolidStart application.
	// Default: false (plain Solid with Vite)
	SolidStart bool
}

// Solid creates a frontend middleware optimized for Solid applications.
//
// Solid-specific optimizations:
//   - Vite manifest path: .vite/manifest.json
//   - Dev server: http://localhost:5173 (Vite)
//   - Proper handling of Solid's fine-grained reactivity output
//
// Example:
//
//	app.Use(adapters.Solid(frontend.Options{
//	    Root:      "./dist",
//	    DevServer: "http://localhost:5173",
//	}))
func Solid(opts frontend.Options) mizu.Middleware {
	return SolidWithOptions(SolidOptions{Options: opts})
}

// SolidWithOptions creates a Solid middleware with extended options.
func SolidWithOptions(opts SolidOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applySolidDefaults(opts)

	return frontend.WithOptions(opts.Options)
}

func applySolidDefaults(opts SolidOptions) frontend.Options {
	o := opts.Options

	// Default to Vite dev server
	if o.DevServer == "" {
		o.DevServer = "http://localhost:5173"
	}

	// SolidStart uses different build output
	if opts.SolidStart {
		if o.Root == "" {
			o.Root = ".output/public"
		}
		// SolidStart has its own routing
		o.IgnorePaths = append(o.IgnorePaths, "/_server")
	}

	return o
}

// SolidStart creates a frontend middleware optimized for SolidStart applications.
//
// SolidStart-specific optimizations:
//   - Dev server: http://localhost:3000
//   - Build output: .output/public
//   - Handles SolidStart-specific paths
func SolidStart(opts frontend.Options) mizu.Middleware {
	if opts.DevServer == "" {
		opts.DevServer = "http://localhost:3000"
	}

	return SolidWithOptions(SolidOptions{
		Options:    opts,
		SolidStart: true,
	})
}
