package usagi

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage"
)

func init() {
	storage.Register("usagi", &Driver{})
}

// Driver implements storage.Driver for the usagi backend.
type Driver struct{}

// Open creates a new usagi storage instance.
// DSN format: usagi:///path/to/root?bucket=default&nofsync=true
func (d *Driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	_ = ctx
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("usagi: empty dsn")
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("usagi: parse dsn: %w", err)
	}
	if u.Scheme != "usagi" && u.Scheme != "" {
		return nil, fmt.Errorf("usagi: unsupported scheme %q", u.Scheme)
	}

	root := strings.TrimSpace(u.Path)
	if root == "" && u.Host != "" {
		root = "/" + u.Host + u.Path
	}
	if root == "" {
		return nil, fmt.Errorf("usagi: missing root path")
	}
	root = filepath.Clean(root)

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("usagi: create root %q: %w", root, err)
	}

	q := u.Query()
	defaultBucket := strings.TrimSpace(q.Get("bucket"))
	nofsync := strings.EqualFold(strings.TrimSpace(q.Get("nofsync")), "true")
	segmentSize := int64(defaultSegSizeMB) * 1024 * 1024
	if v := strings.TrimSpace(q.Get("segment_size_mb")); v != "" {
		if n, err := parseInt64(v); err == nil && n > 0 {
			segmentSize = n * 1024 * 1024
		}
	}
	manifestEvery := 30
	if v := strings.TrimSpace(q.Get("manifest_interval_s")); v != "" {
		if n, err := parseInt64(v); err == nil && n >= 0 {
			manifestEvery = int(n)
		}
	}

	st := &store{
		root:          root,
		defaultBucket: defaultBucket,
		nofsync:       nofsync,
		segmentSize:   segmentSize,
		manifestEvery: time.Duration(manifestEvery) * time.Second,
		buckets:       make(map[string]*bucket),
		features: storage.Features{
			"move": true,
		},
	}
	return st, nil
}
