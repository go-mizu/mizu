// Package favicon provides favicon serving middleware for Mizu.
package favicon

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"

	"github.com/go-mizu/mizu"
)

// Options configures the favicon middleware.
type Options struct {
	// File is the path to the favicon file.
	// Required if Data and FS are not set.
	File string

	// Data is the favicon data bytes.
	// Takes precedence over File.
	Data []byte

	// FS is the file system containing the favicon.
	// Used with File when FS is set.
	FS fs.FS

	// URL is the URL path to serve favicon.
	// Default: "/favicon.ico".
	URL string

	// MaxAge is the Cache-Control max-age.
	// Default: 86400 (24 hours).
	MaxAge int
}

// New creates favicon middleware from file path.
func New(file string) mizu.Middleware {
	return WithOptions(Options{File: file})
}

// FromData creates favicon middleware from bytes.
func FromData(data []byte) mizu.Middleware {
	return WithOptions(Options{Data: data})
}

// WithOptions creates favicon middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.URL == "" {
		opts.URL = "/favicon.ico"
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 86400
	}

	// Load favicon data
	var iconData []byte
	if opts.Data != nil {
		iconData = opts.Data
	} else if opts.File != "" {
		var err error
		if opts.FS != nil {
			f, err := opts.FS.Open(opts.File)
			if err != nil {
				panic("favicon: cannot open file: " + err.Error())
			}
			defer func() { _ = f.Close() }()
			iconData, err = io.ReadAll(f)
			if err != nil {
				panic("favicon: cannot read file: " + err.Error())
			}
		} else {
			iconData, err = os.ReadFile(opts.File)
			if err != nil {
				panic("favicon: cannot read file: " + err.Error())
			}
		}
	}

	// Detect content type
	contentType := http.DetectContentType(iconData)
	if contentType == "application/octet-stream" || contentType == "text/plain; charset=utf-8" {
		// Default to ico if detection fails
		contentType = "image/x-icon"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().URL.Path != opts.URL {
				return next(c)
			}

			// Only allow GET and HEAD
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodHead {
				return next(c)
			}

			if len(iconData) == 0 {
				// No favicon configured, return 204
				c.Writer().WriteHeader(http.StatusNoContent)
				return nil
			}

			// Set headers
			c.Writer().Header().Set("Content-Type", contentType)
			c.Writer().Header().Set("Content-Length", strconv.Itoa(len(iconData)))
			c.Writer().Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(opts.MaxAge))

			if method == http.MethodHead {
				c.Writer().WriteHeader(http.StatusOK)
				return nil
			}

			c.Writer().WriteHeader(http.StatusOK)
			_, err := c.Writer().Write(iconData)
			return err
		}
	}
}

// Empty returns middleware that returns 204 for favicon requests.
func Empty() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().URL.Path == "/favicon.ico" {
				c.Writer().WriteHeader(http.StatusNoContent)
				return nil
			}
			return next(c)
		}
	}
}

// Redirect returns middleware that redirects favicon requests.
func Redirect(url string) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().URL.Path == "/favicon.ico" {
				return c.Redirect(http.StatusMovedPermanently, url)
			}
			return next(c)
		}
	}
}

// SVG creates favicon middleware from SVG data.
func SVG(data []byte) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().URL.Path != "/favicon.ico" && c.Request().URL.Path != "/favicon.svg" {
				return next(c)
			}

			c.Writer().Header().Set("Content-Type", "image/svg+xml")
			c.Writer().Header().Set("Content-Length", strconv.Itoa(len(data)))
			c.Writer().Header().Set("Cache-Control", "public, max-age=86400")

			if c.Request().Method == http.MethodHead {
				c.Writer().WriteHeader(http.StatusOK)
				return nil
			}

			_, err := io.Copy(c.Writer(), bytes.NewReader(data))
			return err
		}
	}
}
