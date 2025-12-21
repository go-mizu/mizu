// app/web/server.go
package web

import (
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/mizu"
)

type Server struct {
	app *mizu.App

	view   view.API
	search search.API
	tmpl   Templates
}

func New(viewAPI view.API, searchAPI search.API, tmpl Templates) *Server {
	s := &Server{
		app:    mizu.New(),
		view:   viewAPI,
		search: searchAPI,
		tmpl:   tmpl,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	r := s.app

	r.Use(Logging())

	r.Get("/", s.home)
	r.Get("/page", s.page)
	r.Get("/search", s.searchPage)

	r.Get("/healthz", func(c *mizu.Ctx) error {
		return c.Text(200, "ok")
	})
}

func (s *Server) Handler() *mizu.App {
	return s.app
}
