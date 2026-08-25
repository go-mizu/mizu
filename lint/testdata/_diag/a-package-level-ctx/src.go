package app

import "github.com/go-mizu/mizu/web"

// active is the request being served right now, so that the audit log can name
// it without every function taking one.
var active *web.Ctx

// Handle serves one.
func Handle(c *web.Ctx) error {
	active = c
	return c.Text("ok")
}
