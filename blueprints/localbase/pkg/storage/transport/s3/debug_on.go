//go:build s3debug

// File: lib/storage/transport/s3/debug_on.go
package s3

import "github.com/go-mizu/mizu"

// debugLogRequest logs S3 request details when built with -tags=s3debug.
func debugLogRequest(c *mizu.Ctx, req *Request) {
	c.Logger().Info("s3 request",
		"op", req.Op,
		"bucket", req.Bucket,
		"key", req.Key,
	)
}
