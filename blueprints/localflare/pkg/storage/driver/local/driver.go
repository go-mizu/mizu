package local

import (
	"context"

	"github.com/go-mizu/blueprints/drive/lib/storage"
)

// driver implements storage.Driver for the local backend.
type driver struct{}

// Open satisfies the storage.Driver interface by delegating to local.Open.
// The DSN is interpreted by parseRoot inside Open.
func (d *driver) Open(ctx context.Context, dsn string) (storage.Storage, error) {
	return Open(ctx, dsn)
}

func init() {
	// Register this backend for both "local" and "file" schemes so the user can write:
	//
	//   storage.Open(ctx, "local:/abs/path")
	//   storage.Open(ctx, "file:///abs/path")
	//   storage.Open(ctx, "/abs/path")  // if driverFromDSN falls back correctly
	//
	// Panics if already registered (as intended).
	storage.Register("local", &driver{})
	storage.Register("file", &driver{})
}
