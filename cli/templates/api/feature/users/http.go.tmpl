package users

import (
	"strconv"

	"github.com/go-mizu/mizu"
)

func List(c *mizu.Ctx) error {
	return c.JSON(200, []map[string]any{
		{"id": 1, "name": "Ada"},
		{"id": 2, "name": "Linus"},
	})
}

func Get(c *mizu.Ctx) error {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.JSON(400, map[string]any{
			"error": "invalid_id",
		})
	}

	return c.JSON(200, map[string]any{
		"id":   id,
		"name": "User " + idStr,
	})
}
