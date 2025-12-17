package hello

import (
	"github.com/go-mizu/mizu"
)

func Get(c *mizu.Ctx) error {
	name := c.Query("name")
	if name == "" {
		name = "world"
	}
	return c.JSON(200, map[string]any{
		"message": "hello " + name,
	})
}
