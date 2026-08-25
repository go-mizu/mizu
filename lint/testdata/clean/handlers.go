// Package clean uses a *web.Ctx the way the web package asks for, and the ctx
// check has nothing to say about any of it.
//
// Every shape here is one somebody reaches for when the shape the check
// reports is what they wanted, so a change that makes the check noisier fails
// on this package first.
package clean

import (
	"context"

	"github.com/go-mizu/mizu/web"
)

// A record is what the handler took out of the request, which is what outlives
// it.
type record struct {
	id string
	ip string
}

// Handle answers the request and hands the work off to something slower.
func Handle(c *web.Ctx) error {
	// A local Ctx belongs to the call, and the call is the handler.
	req := c

	r := record{id: req.Param("id"), ip: req.IP().String()}
	ctx := c.Detach()
	go audit(ctx, r)

	return c.Text("ok")
}

// audit runs after the handler has returned and reads nothing that went back
// in the pool.
func audit(ctx context.Context, r record) {}

// Wrap takes a Ctx and gives one back, which is what a middleware written by
// hand looks like, and is not a Ctx escaping anything.
func Wrap(next web.Handler) web.Handler {
	return func(c *web.Ctx) error {
		c.SetHeader("X-Trace", c.RequestID())
		return next(c)
	}
}

// records is a package level map of what was taken out of requests, which is
// the answer to keeping the requests themselves.
var records = map[string]record{}

// queue carries what a worker needs rather than the request it came from.
func queue(c *web.Ctx) {
	work := make(chan record, 1)
	work <- record{id: c.Param("id")}
	close(work)
}
