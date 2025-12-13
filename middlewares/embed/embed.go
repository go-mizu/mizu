// Package embed provides embedded filesystem serving middleware for Mizu.
package embed

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the embed middleware.
type Options struct {
	// Root is the root directory within the filesystem.
	// Default: ".".
	Root string

	// Index is the index file name.
	// Default: "index.html".
	Index string

	// Browse enables directory browsing.
	// Default: false.
	Browse bool

	// MaxAge sets Cache-Control max-age.
	// Default: 0 (no caching).
	MaxAge int

	// NotFoundHandler handles missing files.
	NotFoundHandler func(c *mizu.Ctx) error
}

// New creates embed middleware for an embedded filesystem.
func New(fsys fs.FS) mizu.Middleware {
	return WithOptions(fsys, Options{})
}

// WithOptions creates embed middleware with custom options.
//
//nolint:cyclop // Embedded file serving requires multiple path and option checks
func WithOptions(fsys fs.FS, opts Options) mizu.Middleware {
	if opts.Root == "" {
		opts.Root = "."
	}
	if opts.Index == "" {
		opts.Index = "index.html"
	}

	var root fs.FS
	if opts.Root != "." {
		var err error
		root, err = fs.Sub(fsys, opts.Root)
		if err != nil {
			root = fsys
		}
	} else {
		root = fsys
	}

	fileServer := http.FileServer(http.FS(root))

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Clean path
			urlPath := c.Request().URL.Path
			if !strings.HasPrefix(urlPath, "/") {
				urlPath = "/" + urlPath
			}
			urlPath = path.Clean(urlPath)

			// Try to open file
			filePath := strings.TrimPrefix(urlPath, "/")
			if filePath == "" {
				filePath = opts.Index
			}

			file, err := root.Open(filePath)
			if err != nil {
				// Try index file for directories
				indexPath := path.Join(filePath, opts.Index)
				file, err = root.Open(indexPath)
				if err != nil {
					if opts.NotFoundHandler != nil {
						return opts.NotFoundHandler(c)
					}
					return next(c)
				}
			}
			_ = file.Close()

			// Set cache headers
			if opts.MaxAge > 0 {
				c.Header().Set("Cache-Control", "max-age="+itoa(opts.MaxAge))
			}

			// Serve file
			fileServer.ServeHTTP(c.Writer(), c.Request())
			return nil
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Handler creates a handler for embedded files (not middleware).
func Handler(fsys fs.FS) mizu.Handler {
	return HandlerWithOptions(fsys, Options{})
}

// HandlerWithOptions creates a handler with options.
func HandlerWithOptions(fsys fs.FS, opts Options) mizu.Handler {
	if opts.Root == "" {
		opts.Root = "."
	}
	if opts.Index == "" {
		opts.Index = "index.html"
	}

	var root fs.FS
	if opts.Root != "." {
		var err error
		root, err = fs.Sub(fsys, opts.Root)
		if err != nil {
			root = fsys
		}
	} else {
		root = fsys
	}

	fileServer := http.FileServer(http.FS(root))

	return func(c *mizu.Ctx) error {
		// Set cache headers
		if opts.MaxAge > 0 {
			c.Header().Set("Cache-Control", "max-age="+itoa(opts.MaxAge))
		}

		fileServer.ServeHTTP(c.Writer(), c.Request())
		return nil
	}
}

// Static creates middleware that serves from a subdirectory.
func Static(fsys fs.FS, subdir string) mizu.Middleware {
	return WithOptions(fsys, Options{Root: subdir})
}

// WithCaching creates middleware with caching enabled.
func WithCaching(fsys fs.FS, maxAge int) mizu.Middleware {
	return WithOptions(fsys, Options{MaxAge: maxAge})
}

// SPA creates middleware for single-page applications.
func SPA(fsys fs.FS, index string) mizu.Middleware {
	if index == "" {
		index = "index.html"
	}

	return WithOptions(fsys, Options{
		Index: index,
		NotFoundHandler: func(c *mizu.Ctx) error {
			// Serve index.html for SPA routing
			file, err := fsys.Open(index)
			if err != nil {
				return c.Text(http.StatusNotFound, "Not Found")
			}
			defer func() { _ = file.Close() }()

			stat, err := file.Stat()
			if err != nil {
				return c.Text(http.StatusInternalServerError, "Error")
			}

			if seeker, ok := file.(ReadSeekFile); ok {
				http.ServeContent(c.Writer(), c.Request(), index, stat.ModTime(), seeker)
			}
			return nil
		},
	})
}

// ReadSeekFile is implemented by files that support seeking.
type ReadSeekFile interface {
	fs.File
	Seek(offset int64, whence int) (int64, error)
}
