package dashboard

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/app/web/dashboard"
)

// Page serves the dashboard HTML page.
func Page(c *mizu.Ctx) error {
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().Header().Set("Cache-Control", "no-cache")
	_, err := c.Writer().Write([]byte(dashboard.HTML))
	return err
}
