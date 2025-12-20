// Package frontend provides comprehensive frontend/SPA integration for Mizu.
//
// It supports both development (with hot reload proxy to Vite/webpack dev servers)
// and production (with optimized static serving, caching, and security headers).
//
// # Quick Start
//
// Basic production serving with embedded filesystem:
//
//	//go:embed dist/*
//	var distFS embed.FS
//
//	func main() {
//	    app := mizu.New()
//
//	    // API routes first
//	    app.Get("/api/users", listUsers)
//
//	    // Frontend middleware catches all other routes
//	    app.Use(frontend.WithFS(distFS))
//
//	    app.Listen(":3000")
//	}
//
// # Development Mode
//
// During development, requests are proxied to your frontend dev server (Vite, webpack, etc.):
//
//	app.Use(frontend.Dev("http://localhost:5173"))
//
// The middleware automatically:
//   - Proxies HTTP requests to the dev server
//   - Proxies WebSocket connections for Hot Module Replacement (HMR)
//   - Shows friendly error pages when the dev server is down
//   - Auto-retries when the dev server restarts
//
// # Auto Mode
//
// Use ModeAuto to automatically switch between dev and production based on environment:
//
//	app.Use(frontend.WithOptions(frontend.Options{
//	    Mode:      frontend.ModeAuto,
//	    Root:      "./dist",
//	    DevServer: "http://localhost:5173",
//	}))
//
// ModeAuto uses production mode when MIZU_ENV, GO_ENV, or ENV is set to "production".
//
// # Caching Strategy
//
// The middleware applies smart caching based on asset type:
//
//   - Hashed assets (app.a1b2c3.js): 1 year, immutable
//   - Unhashed assets (logo.png): 1 week
//   - HTML files (index.html): no-cache
//   - Source maps: no-cache (blocked in production by default)
//
// Custom caching:
//
//	app.Use(frontend.WithOptions(frontend.Options{
//	    Root: "./dist",
//	    CacheControl: frontend.CacheConfig{
//	        HashedAssets:   365 * 24 * time.Hour,
//	        UnhashedAssets: 7 * 24 * time.Hour,
//	        Patterns: map[string]time.Duration{
//	            "*.woff2": 30 * 24 * time.Hour,
//	        },
//	    },
//	}))
//
// # Environment Variable Injection
//
// Inject server-side environment variables into the frontend:
//
//	app.Use(frontend.WithOptions(frontend.Options{
//	    Root: "./dist",
//	    InjectEnv: []string{"API_URL", "ANALYTICS_ID"},
//	}))
//
// Variables are available in your frontend as:
//
//	const apiUrl = window.__ENV__.API_URL;
//
// # Security Headers
//
// By default, the middleware adds security headers:
//
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: SAMEORIGIN
//   - X-XSS-Protection: 1; mode=block
//   - Referrer-Policy: strict-origin-when-cross-origin
//
// Disable with SecurityHeaders: false.
//
// # Build Manifest Integration
//
// For optimal caching and preloading, use the Manifest API with Vite or Webpack:
//
//	manifest, _ := frontend.LoadManifest(distFS, ".vite/manifest.json")
//
//	// In templates
//	{{ vite_entry "src/main.tsx" }}
//
// # Framework Adapters
//
// Framework-specific adapters apply optimal defaults:
//
//	import "github.com/go-mizu/mizu/frontend/adapters"
//
//	// React (Vite)
//	app.Use(adapters.React(frontend.Options{Root: "./dist"}))
//
//	// Vue (Vite)
//	app.Use(adapters.Vue(frontend.Options{Root: "./dist"}))
//
//	// Svelte (Vite)
//	app.Use(adapters.Svelte(frontend.Options{Root: "./dist"}))
//
//	// SvelteKit
//	app.Use(adapters.SvelteKit(frontend.Options{Root: "./build"}))
//
//	// Solid (Vite)
//	app.Use(adapters.Solid(frontend.Options{Root: "./dist"}))
//
//	// Astro
//	app.Use(adapters.Astro(frontend.Options{Root: "./dist"}))
//
//	// Angular
//	app.Use(adapters.Angular(frontend.Options{Root: "./dist/my-app/browser"}))
//
// # Ignored Paths
//
// By default, these paths bypass the SPA fallback:
//
//   - /api
//   - /health
//   - /metrics
//
// Add custom paths:
//
//	app.Use(frontend.WithOptions(frontend.Options{
//	    Root:        "./dist",
//	    IgnorePaths: []string{"/api", "/auth", "/webhooks"},
//	}))
package frontend
