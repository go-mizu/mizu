// app/web/render.go
package web

import "github.com/go-mizu/mizu"

type Templates interface {
	Render(w any, name string, data any) error
}

func (s *Server) render(c *mizu.Ctx, name string, data any) {
	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Render(c.Writer(), name, data); err != nil {
		c.Text(500, err.Error())
	}
}
