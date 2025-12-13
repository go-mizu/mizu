// Package static provides static file serving middleware for Mizu.
package static

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the static middleware.
type Options struct {
	// Root is the root directory to serve files from.
	// Required if FS is not set.
	Root string

	// FS is the file system to serve files from.
	// Takes precedence over Root.
	FS fs.FS

	// Index is the index file name.
	// Default: "index.html".
	Index string

	// Browse enables directory browsing.
	// Default: false.
	Browse bool

	// Prefix is the URL prefix to strip.
	// Default: "".
	Prefix string

	// MaxAge is the Cache-Control max-age in seconds.
	// Default: 0 (no cache).
	MaxAge int

	// NotFoundHandler is called when file is not found.
	// Default: calls next handler.
	NotFoundHandler mizu.Handler

	// Compress enables gzip compression.
	// Default: false.
	Compress bool
}

// New creates static file middleware with root directory.
func New(root string) mizu.Middleware {
	return WithOptions(Options{Root: root})
}

// WithFS creates static file middleware with fs.FS.
func WithFS(fsys fs.FS) mizu.Middleware {
	return WithOptions(Options{FS: fsys})
}

// WithOptions creates static file middleware with custom options.
//
//nolint:cyclop // Static file serving requires multiple path and directory checks
func WithOptions(opts Options) mizu.Middleware {
	if opts.FS == nil && opts.Root == "" {
		panic("static: either FS or Root is required")
	}
	if opts.Index == "" {
		opts.Index = "index.html"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			// Strip prefix
			if opts.Prefix != "" {
				if !strings.HasPrefix(path, opts.Prefix) {
					return next(c)
				}
				path = strings.TrimPrefix(path, opts.Prefix)
				if path == "" {
					path = "/"
				}
			}

			// Check if file exists
			var exists bool
			var isDir bool
			cleanPath := strings.TrimPrefix(path, "/")
			if cleanPath == "" {
				cleanPath = "."
			}

			if opts.FS != nil {
				if info, err := fs.Stat(opts.FS, cleanPath); err == nil {
					exists = true
					isDir = info.IsDir()
				}
			} else {
				fullPath := filepath.Join(opts.Root, filepath.Clean(path))
				if info, err := os.Stat(fullPath); err == nil {
					exists = true
					isDir = info.IsDir()
				}
			}

			if !exists {
				if opts.NotFoundHandler != nil {
					return opts.NotFoundHandler(c)
				}
				return next(c)
			}

			// Handle directory
			if isDir {
				// Check for index file
				indexPath := cleanPath
				if indexPath != "." && !strings.HasSuffix(indexPath, "/") {
					indexPath += "/"
				}
				if indexPath == "." {
					indexPath = opts.Index
				} else {
					indexPath += opts.Index
				}

				var indexExists bool
				if opts.FS != nil {
					_, err := fs.Stat(opts.FS, indexPath)
					indexExists = err == nil
				} else {
					fullIndex := filepath.Join(opts.Root, filepath.Clean(indexPath))
					_, err := os.Stat(fullIndex)
					indexExists = err == nil
				}

				if indexExists {
					cleanPath = indexPath
				} else if !opts.Browse {
					if opts.NotFoundHandler != nil {
						return opts.NotFoundHandler(c)
					}
					return next(c)
				} else {
					// Directory browsing - use file server
					return serveWithFileServer(c, opts, path)
				}
			}

			// Set cache headers
			if opts.MaxAge > 0 {
				c.Writer().Header().Set("Cache-Control", "public, max-age="+itoa(opts.MaxAge))
			}

			// Serve file directly
			return serveFile(c, opts, cleanPath)
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
		file, err = os.Open(filepath.Join(opts.Root, cleanPath)) //nolint:gosec // G304: Path is cleaned via filepath.Clean
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

// serveWithFileServer uses http.FileServer for directory browsing.
func serveWithFileServer(c *mizu.Ctx, opts Options, path string) error {
	var fileServer http.Handler
	if opts.FS != nil {
		fileServer = http.FileServer(http.FS(opts.FS))
	} else {
		fileServer = http.FileServer(http.Dir(opts.Root))
	}
	// Ensure directory paths end with / to prevent redirect
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	c.Request().URL.Path = path
	fileServer.ServeHTTP(c.Writer(), c.Request())
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
