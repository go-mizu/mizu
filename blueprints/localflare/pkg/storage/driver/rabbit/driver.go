// Package rabbit implements a high-performance in-process storage driver.
// It aims to achieve 10x performance improvement over MinIO by eliminating
// network overhead and leveraging modern I/O techniques.
package rabbit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

func init() {
	storage.Register("rabbit", &Driver{})
}

// Driver implements storage.Driver for the rabbit backend.
type Driver struct{}

// Open creates a new rabbit storage instance.
// DSN format: rabbit:///path/to/root?nofsync=true
// Options:
//   - nofsync=true: Disable fsync for maximum write performance (data loss risk on crash)
func (d *Driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root, opts, err := parseRootAndOptions(dsn)
	if err != nil {
		return nil, err
	}

	// Apply global options
	if opts.Get("nofsync") == "true" {
		NoFsync = true
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Create root directory if it doesn't exist
			if err := os.MkdirAll(root, 0750); err != nil {
				return nil, fmt.Errorf("rabbit: create root %q: %w", root, err)
			}
		} else {
			return nil, fmt.Errorf("rabbit: stat root %q: %w", root, err)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf("rabbit: root %q is not a directory", root)
	}

	return NewStore(root)
}

// parseRootAndOptions extracts the root path and query options from DSN.
func parseRootAndOptions(dsn string) (string, url.Values, error) {
	if dsn == "" {
		return "", nil, errors.New("rabbit: empty dsn")
	}

	// Split off query string
	var queryStr string
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		queryStr = dsn[idx+1:]
		dsn = dsn[:idx]
	}

	// Parse query options
	opts, _ := url.ParseQuery(queryStr)

	// Handle bare path
	if strings.HasPrefix(dsn, "/") {
		return filepath.Clean(dsn), opts, nil
	}

	// Handle rabbit:/ prefix
	if strings.HasPrefix(dsn, "rabbit:") {
		rest := strings.TrimPrefix(dsn, "rabbit:")
		if strings.HasPrefix(rest, "//") {
			// rabbit:///path or rabbit://./path
			rest = strings.TrimPrefix(rest, "//")
		}
		if rest == "" {
			return "", nil, errors.New("rabbit: missing path")
		}
		return filepath.Clean(rest), opts, nil
	}

	// Try URL parsing
	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("rabbit: parse dsn: %w", err)
	}

	if u.Scheme != "rabbit" && u.Scheme != "" {
		return "", nil, fmt.Errorf("rabbit: unsupported scheme %q", u.Scheme)
	}

	path := u.Path
	if u.Host == "." {
		path = "./" + path
	}
	if path == "" {
		return "", nil, errors.New("rabbit: missing path in dsn")
	}

	return filepath.Clean(path), opts, nil
}
