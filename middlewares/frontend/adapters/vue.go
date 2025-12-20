package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/middlewares/frontend"
)

// VueOptions extends frontend.Options with Vue-specific settings.
type VueOptions struct {
	frontend.Options

	// VueDevtools enables Vue DevTools hints.
	// Default: true in development
	VueDevtools bool
}

// Vue creates a frontend middleware optimized for Vue applications.
//
// Vue-specific optimizations:
//   - Vite manifest path: .vite/manifest.json
//   - Dev server: http://localhost:5173 (Vite)
//   - Vue DevTools hints in development
//
// Example:
//
//	app.Use(adapters.Vue(frontend.Options{
//	    Root:      "./dist",
//	    DevServer: "http://localhost:5173",
//	}))
func Vue(opts frontend.Options) mizu.Middleware {
	return VueWithOptions(VueOptions{Options: opts})
}

// VueWithOptions creates a Vue middleware with extended options.
func VueWithOptions(opts VueOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyVueDefaults(opts)

	return frontend.WithOptions(opts.Options)
}

func applyVueDefaults(opts VueOptions) frontend.Options {
	o := opts.Options

	// Default to Vite dev server
	if o.DevServer == "" {
		o.DevServer = "http://localhost:5173"
	}

	// Add Vue DevTools meta in development
	if o.Mode != frontend.ModeProduction && opts.VueDevtools {
		if o.InjectMeta == nil {
			o.InjectMeta = make(map[string]string)
		}
		o.InjectMeta["vue-devtools-enabled"] = "true"
	}

	return o
}

// Nuxt creates a frontend middleware optimized for Nuxt applications.
//
// Nuxt-specific optimizations:
//   - Dev server: http://localhost:3000
//   - Handles Nuxt-specific paths (_nuxt)
//   - SSR support hints
//
// Note: For full Nuxt features, consider running Nuxt as the primary server
// and proxying API routes to Mizu instead.
func Nuxt(opts frontend.Options) mizu.Middleware {
	// Nuxt defaults
	if opts.DevServer == "" {
		opts.DevServer = "http://localhost:3000"
	}
	if opts.Root == "" {
		opts.Root = ".output/public"
	}

	// Nuxt uses its own routing
	opts.IgnorePaths = append(opts.IgnorePaths, "/_nuxt")

	return frontend.WithOptions(opts)
}
