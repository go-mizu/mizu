package app

import "github.com/go-mizu/mizu/web"

// Handle is fine. What is wrong with this case is the name on the command
// line, which is in the checks file beside this one.
func Handle(c *web.Ctx) error {
	return c.Text("ok")
}
