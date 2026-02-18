//go:build !s3debug

// File: lib/storage/transport/s3/debug.go
package s3

import "github.com/go-mizu/mizu"

// debugLogRequest is a no-op in release builds for maximum performance.
// Build with -tags=s3debug to enable debug logging.
func debugLogRequest(_ *mizu.Ctx, _ *Request) {}
