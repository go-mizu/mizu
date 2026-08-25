package app

import "github.com/go-mizu/mizu/web"

// Fanout hands the request to a worker and answers straight away.
func Fanout(c *web.Ctx) error {
	work := make(chan *web.Ctx, 1)
	work <- c
	close(work)
	return c.Text("queued")
}
