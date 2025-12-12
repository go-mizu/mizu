// Package spa provides single-page application middleware for Mizu.
package spa

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the SPA middleware.
type Options struct {
	// Root is the root directory to serve files from.
	// Required if FS is not set.
	Root string

	// FS is the file system to serve files from.
	// Takes precedence over Root.
	FS fs.FS

	// Index is the fallback file for SPA routing.
	// Default: "index.html".
	Index string

	// Prefix is the URL prefix for static assets.
	// Default: "".
	Prefix string

	// IgnorePaths are paths that should not fallback to index.
	// Useful for API routes.
	// Default: []string{"/api"}.
	IgnorePaths []string

	// MaxAge is the Cache-Control max-age for static assets.
	// Default: 0 (no cache).
	MaxAge int

	// IndexMaxAge is the Cache-Control max-age for index.html.
	// Default: 0 (no cache).
	IndexMaxAge int
}

// New creates SPA middleware with root directory.
func New(root string) mizu.Middleware {
	return WithOptions(Options{Root: root})
}

// WithFS creates SPA middleware with fs.FS.
func WithFS(fsys fs.FS) mizu.Middleware {
	return WithOptions(Options{FS: fsys})
}

// WithOptions creates SPA middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.FS == nil && opts.Root == "" {
		panic("spa: either FS or Root is required")
	}
	if opts.Index == "" {
		opts.Index = "index.html"
	}
	if opts.IgnorePaths == nil {
		opts.IgnorePaths = []string{"/api"}
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

			// Strip prefix
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

			// Check if file exists
			cleanPath := strings.TrimPrefix(servePath, "/")
			if cleanPath == "" {
				cleanPath = "."
			}

			var exists bool
			var isDir bool

			if opts.FS != nil {
				if info, err := fs.Stat(opts.FS, cleanPath); err == nil {
					exists = true
					isDir = info.IsDir()
				}
			} else {
				fullPath := filepath.Join(opts.Root, filepath.Clean(servePath))
				if info, err := os.Stat(fullPath); err == nil {
					exists = true
					isDir = info.IsDir()
				}
			}

			// Serve file if it exists and is not a directory
			if exists && !isDir {
				// Set cache headers for static assets
				if opts.MaxAge > 0 {
					c.Writer().Header().Set("Cache-Control", "public, max-age="+itoa(opts.MaxAge))
				}
				return serveFile(c, opts, cleanPath)
			}

			// Fallback to index.html for SPA routing
			if opts.IndexMaxAge > 0 {
				c.Writer().Header().Set("Cache-Control", "public, max-age="+itoa(opts.IndexMaxAge))
			} else {
				// No cache for index.html by default
				c.Writer().Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			}

			return serveFile(c, opts, opts.Index)
		}
	}
}

// serveFile serves a file directly without http.FileServer redirect behavior.
func serveFile(c *mizu.Ctx, opts Options, cleanPath string) error {
	var file fs.File
	var err error

	if opts.FS != nil {
		file, err = opts.FS.Open(cleanPath)
	} else {
		file, err = os.Open(filepath.Join(opts.Root, cleanPath))
	}

	if err != nil {
		return c.Text(http.StatusNotFound, "Not Found")
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return c.Text(http.StatusInternalServerError, "Internal Server Error")
	}

	// Use http.ServeContent for proper content-type detection and range support
	http.ServeContent(c.Writer(), c.Request(), cleanPath, stat.ModTime(), file.(io.ReadSeeker))
	return nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
