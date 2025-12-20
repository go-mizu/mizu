// Package frontend provides comprehensive frontend/SPA integration for Mizu.
//
// It supports both development (with hot reload proxy to Vite/webpack dev servers)
// and production (with optimized static serving, caching, and security headers).
//
// Basic usage:
//
//	// Production: serve from dist directory
//	app.Use(frontend.New("./dist"))
//
//	// Production with embedded filesystem
//	//go:embed dist/*
//	var distFS embed.FS
//	app.Use(frontend.WithFS(distFS))
//
//	// Development: proxy to Vite dev server
//	app.Use(frontend.Dev("http://localhost:5173"))
//
//	// Auto-detect mode based on MIZU_ENV
//	app.Use(frontend.WithOptions(frontend.Options{
//	    Mode:      frontend.ModeAuto,
//	    Root:      "./dist",
//	    DevServer: "http://localhost:5173",
//	}))
package frontend

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Mode determines how the frontend is served.
type Mode string

const (
	// ModeDev proxies requests to a development server.
	ModeDev Mode = "dev"

	// ModeProduction serves static files from disk or embedded FS.
	ModeProduction Mode = "production"

	// ModeAuto auto-detects based on MIZU_ENV environment variable.
	// Uses ModeDev if MIZU_ENV is not "production".
	ModeAuto Mode = "auto"
)

// Options configures the frontend middleware.
type Options struct {
	// Mode determines serving behavior.
	// Default: ModeAuto
	Mode Mode

	// --- Production Options ---

	// Root is the directory containing built assets.
	// Default: "dist"
	Root string

	// FS is an embedded filesystem for production builds.
	// Takes precedence over Root.
	FS fs.FS

	// Index is the fallback file for SPA routing.
	// Default: "index.html"
	Index string

	// Prefix is the URL prefix for serving (e.g., "/app").
	// Default: ""
	Prefix string

	// IgnorePaths are paths that bypass SPA fallback.
	// Default: []string{"/api", "/health", "/metrics"}
	IgnorePaths []string

	// CacheControl configures caching strategy.
	CacheControl CacheConfig

	// --- Development Options ---

	// DevServer is the development server URL.
	// Default: "http://localhost:5173" (Vite default)
	DevServer string

	// DevServerTimeout is the timeout for dev server requests.
	// Default: 30s
	DevServerTimeout time.Duration

	// ProxyWebSocket enables WebSocket proxying for HMR.
	// Default: true
	ProxyWebSocket bool

	// --- Advanced Options ---

	// Manifest is the path to build manifest (vite/webpack).
	// Used for asset fingerprinting and preloading.
	Manifest string

	// InjectEnv injects environment variables into index.html.
	// Variables are exposed as window.__ENV__
	InjectEnv []string

	// InjectMeta adds custom meta tags to index.html.
	InjectMeta map[string]string

	// SecurityHeaders adds recommended security headers.
	// Default: true
	SecurityHeaders bool

	// Compression enables gzip/brotli compression.
	// Default: true in production
	Compression bool

	// SourceMaps controls source map serving in production.
	// Default: false in production
	SourceMaps bool

	// ServiceWorker is the path to service worker file.
	// If set, proper headers are added for SW scope.
	ServiceWorker string

	// ErrorHandler handles errors.
	ErrorHandler func(*mizu.Ctx, error) error

	// NotFoundHandler handles 404s before SPA fallback.
	// Return nil to continue to SPA fallback.
	NotFoundHandler func(*mizu.Ctx) error
}

// CacheConfig configures caching behavior.
type CacheConfig struct {
	// HashedAssets is the cache duration for assets with content hash.
	// Default: 1 year (immutable)
	HashedAssets time.Duration

	// UnhashedAssets is the cache duration for assets without hash.
	// Default: 1 week
	UnhashedAssets time.Duration

	// HTML is the cache duration for HTML files.
	// Default: 0 (no-cache)
	HTML time.Duration

	// Patterns maps glob patterns to cache durations.
	// Example: {"*.woff2": 30 * 24 * time.Hour}
	Patterns map[string]time.Duration
}

// New creates frontend middleware with sensible defaults.
// Auto-detects mode based on MIZU_ENV environment variable.
func New(root string) mizu.Middleware {
	return WithOptions(Options{Root: root})
}

// Dev creates development-only middleware that proxies to dev server.
func Dev(devServerURL string) mizu.Middleware {
	return WithOptions(Options{
		Mode:      ModeDev,
		DevServer: devServerURL,
	})
}

// WithFS creates production middleware with embedded filesystem.
func WithFS(fsys fs.FS) mizu.Middleware {
	return WithOptions(Options{FS: fsys})
}

// WithOptions creates middleware with full configuration.
func WithOptions(opts Options) mizu.Middleware {
	opts = applyDefaults(opts)

	// Determine effective mode
	mode := opts.Mode
	if mode == ModeAuto {
		mode = detectMode()
	}

	if mode == ModeDev {
		return newDevProxy(opts)
	}
	return newStaticServer(opts)
}

//nolint:cyclop // Configuration defaults require multiple checks
func applyDefaults(opts Options) Options {
	if opts.Mode == "" {
		opts.Mode = ModeAuto
	}
	if opts.Root == "" && opts.FS == nil {
		opts.Root = "dist"
	}
	if opts.Index == "" {
		opts.Index = "index.html"
	}
	if opts.IgnorePaths == nil {
		opts.IgnorePaths = []string{"/api", "/health", "/metrics"}
	}
	if opts.DevServer == "" {
		opts.DevServer = "http://localhost:5173"
	}
	if opts.DevServerTimeout == 0 {
		opts.DevServerTimeout = 30 * time.Second
	}
	if !opts.ProxyWebSocket {
		opts.ProxyWebSocket = true
	}

	// Cache defaults
	if opts.CacheControl.HashedAssets == 0 {
		opts.CacheControl.HashedAssets = 365 * 24 * time.Hour // 1 year
	}
	if opts.CacheControl.UnhashedAssets == 0 {
		opts.CacheControl.UnhashedAssets = 7 * 24 * time.Hour // 1 week
	}

	return opts
}

func detectMode() Mode {
	env := os.Getenv("MIZU_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = os.Getenv("ENV")
	}

	if strings.ToLower(env) == "production" || strings.ToLower(env) == "prod" {
		return ModeProduction
	}

	return ModeDev
}

// newStaticServer creates the production static file server.
//
//nolint:cyclop // Static serving requires multiple path and cache checks
func newStaticServer(opts Options) mizu.Middleware {
	var fsys fs.FS
	if opts.FS != nil {
		fsys = opts.FS
	} else {
		fsys = os.DirFS(opts.Root)
	}

	// Pre-load index.html for injection
	var indexContent []byte
	if len(opts.InjectEnv) > 0 || len(opts.InjectMeta) > 0 {
		if data, err := fs.ReadFile(fsys, opts.Index); err == nil {
			indexContent = data
			if len(opts.InjectEnv) > 0 {
				indexContent = InjectEnv(indexContent, opts.InjectEnv)
			}
			if len(opts.InjectMeta) > 0 {
				indexContent = InjectMeta(indexContent, opts.InjectMeta)
			}
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			// Check if path should be ignored
			for _, ignorePath := range opts.IgnorePaths {
				if strings.HasPrefix(path, ignorePath) {
					return next(c)
				}
			}

			// Strip prefix if configured
			servePath := path
			if opts.Prefix != "" {
				if !strings.HasPrefix(path, opts.Prefix) {
					return next(c)
				}
				servePath = strings.TrimPrefix(path, opts.Prefix)
				if servePath == "" {
					servePath = "/"
				}
			}

			// Clean path for filesystem access
			cleanPath := strings.TrimPrefix(servePath, "/")
			if cleanPath == "" {
				cleanPath = "."
			}

			// Block source maps in production if disabled
			if !opts.SourceMaps && strings.HasSuffix(cleanPath, ".map") {
				return c.Text(http.StatusNotFound, "Not Found")
			}

			// Check if file exists
			info, err := fs.Stat(fsys, cleanPath)
			exists := err == nil
			isDir := exists && info.IsDir()

			// Apply security headers
			if opts.SecurityHeaders {
				applySecurityHeaders(c.Writer())
			}

			// Helper to serve index with injections
			serveIndex := func() error {
				setHTMLCacheHeaders(c.Writer(), opts.CacheControl)
				if len(indexContent) > 0 {
					c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
					c.Writer().WriteHeader(http.StatusOK)
					_, err := c.Writer().Write(indexContent)
					return err
				}
				return serveFile(c, fsys, opts.Index)
			}

			// Serve file if it exists and is not a directory
			if exists && !isDir {
				// If serving index.html directly, use injected content
				if cleanPath == opts.Index {
					return serveIndex()
				}
				setCacheHeaders(c.Writer(), cleanPath, opts.CacheControl)
				return serveFile(c, fsys, cleanPath)
			}

			// Check for index in directory
			if isDir {
				indexPath := filepath.Join(cleanPath, opts.Index)
				if _, err := fs.Stat(fsys, indexPath); err == nil {
					// Use injected content for index files
					return serveIndex()
				}
			}

			// SPA fallback: serve index.html
			if opts.NotFoundHandler != nil {
				if err := opts.NotFoundHandler(c); err != nil {
					return err
				}
			}

			return serveIndex()
		}
	}
}

// serveFile serves a file from the filesystem.
func serveFile(c *mizu.Ctx, fsys fs.FS, path string) error {
	file, err := fsys.Open(path)
	if err != nil {
		return c.Text(http.StatusNotFound, "Not Found")
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return c.Text(http.StatusInternalServerError, "Internal Server Error")
	}

	// Use http.ServeContent for proper content-type detection and range support
	http.ServeContent(c.Writer(), c.Request(), path, stat.ModTime(), file.(io.ReadSeeker))
	return nil
}

// Default security headers for frontend assets.
var defaultSecurityHeaders = map[string]string{
	"X-Content-Type-Options": "nosniff",
	"X-Frame-Options":        "SAMEORIGIN",
	"X-XSS-Protection":       "1; mode=block",
	"Referrer-Policy":        "strict-origin-when-cross-origin",
}

func applySecurityHeaders(w http.ResponseWriter) {
	for k, v := range defaultSecurityHeaders {
		w.Header().Set(k, v)
	}
}
