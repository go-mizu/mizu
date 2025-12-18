package echo

import "github.com/go-mizu/mizu"

type Request struct {
	Message string `json:"message"`
}

func Post(c *mizu.Ctx) error {
	var req Request
	if err := c.BindJSON(&req, 1<<20); err != nil { // 1MB max
		return c.JSON(400, map[string]any{
			"error": "invalid_json",
		})
	}
	return c.JSON(200, map[string]any{
		"message": req.Message,
	})
}
