// File: lib/storage/driver.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// Driver opens a Storage from a DSN string.
//
// A DSN is typically a URL like:
//
//	"local:///var/data/storage"
//	"file:///tmp/storage"
//	"s3://bucket-name?region=us-east-1"
//	"sftp://user@host:22/path?key=/id_ed25519"
//
// The scheme (local, file, s3, sftp, rclone) selects which driver to use.
// The rest of the DSN is interpreted by that driver.
type Driver interface {
	Open(ctx context.Context, dsn string) (Storage, error)
}

// drivers holds registered drivers keyed by name or scheme.
var drivers sync.Map // map[string]Driver

// Register registers a Driver for a name or scheme.
//
// It panics if name is empty, d is nil, or name is already registered.
//
// Typical usage in a driver package init:
//
//	func init() {
//	    storage.Register("local", &driver{})
//	    storage.Register("file", &driver{})
//	    storage.Register("s3", &s3Driver{})
//	}
func Register(name string, d Driver) {
	if name == "" {
		panic("storage: empty driver name")
	}
	if d == nil {
		panic("storage: nil driver")
	}

	if _, loaded := drivers.LoadOrStore(name, d); loaded {
		panic("storage: driver already registered: " + name)
	}
}

// Open selects a Driver based on dsn and opens a Storage.
//
// Open returns Storage rather than Bucket so callers can:
//
//   - Work with multiple buckets under the same backend (for example S3 account,
//     local root, SFTP host).
//   - List and manage buckets via Storage methods.
//   - Decorate or wrap Storage in one place (logging, quotas, metrics).
//
// Examples:
//
//	st, err := storage.Open(ctx, "local:///var/data/storage")
//	st, err := storage.Open(ctx, "s3://my-bucket?region=us-east-1")
func Open(ctx context.Context, dsn string) (Storage, error) {
	if dsn == "" {
		return nil, errors.New("storage: empty dsn")
	}

	name, err := driverFromDSN(dsn)
	if err != nil {
		return nil, err
	}

	v, ok := drivers.Load(name)
	if !ok {
		return nil, fmt.Errorf("storage: unknown driver %q", name)
	}

	d, ok := v.(Driver)
	if !ok {
		return nil, errors.New("storage: invalid driver type stored")
	}
	return d.Open(ctx, dsn)
}

// driverFromDSN tries to parse dsn as URL and return its scheme.
//
// If URL parsing fails or there is no scheme, it falls back to splitting
// on the first colon and using the prefix as driver name:
//
//	"local:/var/data" -> "local"
func driverFromDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err == nil && u.Scheme != "" {
		return u.Scheme, nil
	}

	i := strings.IndexByte(dsn, ':')
	if i <= 0 {
		return "", fmt.Errorf("storage: missing driver in dsn %q", dsn)
	}
	return dsn[:i], nil
}
