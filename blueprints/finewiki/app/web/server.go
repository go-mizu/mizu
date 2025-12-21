// app/web/server.go
package web

import (
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/mizu"
)

type Server struct {
	r *mizu.Router

	view   view.API
	search search.API
	tmpl   Templates
}

func New(viewAPI view.API, searchAPI search.API, tmpl Templates) *Server {
	s := &Server{
		r:      mizu.New(),
		view:   viewAPI,
		search: searchAPI,
		tmpl:   tmpl,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	r := s.r

	r.Use(Logging())

	r.Get("/", s.home)
	r.Get("/page", s.page)
	r.Get("/search", s.searchPage)

	r.Get("/healthz", func(c *mizu.Ctx) {
		c.Text(200, "ok")
	})
}

func (s *Server) Handler() *mizu.Router {
	return s.r
}
