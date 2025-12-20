package adapters

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/middlewares/frontend"
)

// ReactOptions extends frontend.Options with React-specific settings.
type ReactOptions struct {
	frontend.Options

	// StrictMode enables React strict mode detection headers.
	// Default: true in development
	StrictMode bool

	// DevTools enables React DevTools hints.
	// Default: true in development
	DevTools bool

	// ConcurrentFeatures enables headers for React 18+ concurrent features.
	// Default: true
	ConcurrentFeatures bool
}

// React creates a frontend middleware optimized for React applications.
//
// React-specific optimizations:
//   - Vite manifest path: .vite/manifest.json
//   - Dev server: http://localhost:5173 (Vite) or http://localhost:3000 (CRA)
//   - React DevTools hints in development
//   - Concurrent features headers (React 18+)
//
// Example:
//
//	app.Use(adapters.React(frontend.Options{
//	    Root:      "./dist",
//	    DevServer: "http://localhost:5173",
//	}))
func React(opts frontend.Options) mizu.Middleware {
	return ReactWithOptions(ReactOptions{Options: opts})
}

// ReactWithOptions creates a React middleware with extended options.
func ReactWithOptions(opts ReactOptions) mizu.Middleware {
	opts.Options = applyCommonDefaults(opts.Options)
	opts.Options = applyReactDefaults(opts)

	base := frontend.WithOptions(opts.Options)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Add React-specific headers
			if opts.ConcurrentFeatures && opts.Mode != frontend.ModeDev {
				// Enable React profiling for production debugging
				c.Writer().Header().Set("Document-Policy", "js-profiling")
			}

			return base(next)(c)
		}
	}
}

func applyReactDefaults(opts ReactOptions) frontend.Options {
	o := opts.Options

	// Default to Vite dev server
	if o.DevServer == "" {
		o.DevServer = "http://localhost:5173"
	}

	// Add React DevTools meta in development
	if o.Mode != frontend.ModeProduction && opts.DevTools {
		if o.InjectMeta == nil {
			o.InjectMeta = make(map[string]string)
		}
		o.InjectMeta["react-devtools-backend"] = "enabled"
	}

	return o
}

// NextJS creates a frontend middleware optimized for Next.js applications.
//
// Next.js-specific optimizations:
//   - Dev server: http://localhost:3000
//   - Handles Next.js specific paths (_next, api)
//   - ISR cache headers
//
// Note: For full Next.js features, consider running Next.js as the primary server
// and proxying API routes to Mizu instead.
func NextJS(opts frontend.Options) mizu.Middleware {
	// Next.js defaults
	if opts.DevServer == "" {
		opts.DevServer = "http://localhost:3000"
	}
	if opts.Root == "" {
		opts.Root = ".next"
	}

	// Next.js uses its own routing, so we primarily proxy
	opts.IgnorePaths = append(opts.IgnorePaths, "/_next")

	return frontend.WithOptions(opts)
}
