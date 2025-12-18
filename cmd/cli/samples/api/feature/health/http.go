package health

import "github.com/go-mizu/mizu"

func Get(c *mizu.Ctx) error {
	return c.Text(200, "ok\n")
}
