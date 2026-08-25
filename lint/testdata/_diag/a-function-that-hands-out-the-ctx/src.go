package app

import (
	"context"

	"github.com/go-mizu/mizu/web"
)

// Current is the request being served, for code too far from the handler to
// have been passed it.
func Current(ctx context.Context) *web.Ctx {
	c, _ := web.FromContext(ctx)
	return c
}
