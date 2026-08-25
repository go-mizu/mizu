package app

import "github.com/go-mizu/mizu/web"

// Handle answers now and tells the audit log about it afterwards.
func Handle(c *web.Ctx) error {
	go func() {
		audit(c.Param("id"), c.IP().String())
	}()
	return c.NoContent()
}

// audit writes one line somewhere slow.
func audit(id, ip string) {}
