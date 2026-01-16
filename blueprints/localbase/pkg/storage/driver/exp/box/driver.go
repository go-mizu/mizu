package box

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage"
	"github.com/go-mizu/mizu/blueprints/localbase/pkg/storage/driver/local"
)

// driver implements storage.Driver for the box backend.
type driver struct{}

// Open interprets a box:// DSN and delegates to the local backend using the
// resolved absolute path. This lets the box driver reuse the well-tested local
// implementation while keeping its own scheme.
func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	root, err := parseRoot(dsn)
	if err != nil {
		return nil, err
	}
	return local.Open(ctx, root)
}

// parseRoot extracts an absolute filesystem path from a box DSN.
//
// Supported formats:
//
//	box:/abs/path
//	box:///abs/path
//	box:C:\windows\path
func parseRoot(dsn string) (string, error) {
	if dsn == "" {
		return "", errors.New("box: empty dsn")
	}

	if !strings.HasPrefix(dsn, "box:") {
		return "", fmt.Errorf("box: unsupported dsn %q", dsn)
	}

	rest := strings.TrimPrefix(dsn, "box:")
	if rest == "" {
		return "", errors.New("box: missing path")
	}

	if strings.HasPrefix(rest, "//") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", fmt.Errorf("box: parse dsn: %w", err)
		}

		// Treat a host segment as part of the path so both box:/path and
		// box://host/path map to absolute filesystem paths.
		path := u.Path
		if u.Host != "" {
			path = "/" + u.Host + u.Path
		}

		if path == "" {
			return "", errors.New("box: empty path")
		}
		return filepath.Clean(path), nil
	}

	// Check for Windows absolute path (e.g., box:C:\path)
	if isWindowsAbsPath(rest) {
		return filepath.Clean(rest), nil
	}

	// Unix absolute path must start with /
	if !strings.HasPrefix(rest, "/") {
		return "", errors.New("box: path must be absolute")
	}

	return filepath.Clean(rest), nil
}

// isWindowsAbsPath checks if a path is a Windows absolute path.
// Matches patterns like C:, C:\, C:/, D:\, etc.
func isWindowsAbsPath(p string) bool {
	if len(p) < 2 {
		return false
	}
	// Check for drive letter followed by colon
	if (p[0] >= 'A' && p[0] <= 'Z' || p[0] >= 'a' && p[0] <= 'z') && p[1] == ':' {
		return true
	}
	return false
}

func init() {
	storage.Register("box", &driver{})
}
