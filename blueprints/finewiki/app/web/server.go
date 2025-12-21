// app/web/server.go
package web

import (
	"log"
	"time"

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

	r.Use(logging())

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

// logging returns a middleware that logs each request.
func logging() mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			start := time.Now()

			err := next(c)

			elapsed := time.Since(start)
			status := c.StatusCode()
			if status == 0 {
				status = 200
			}

			log.Printf("%s %s %d %s",
				c.Request().Method,
				c.Request().URL.Path,
				status,
				elapsed.Round(time.Microsecond),
			)

			return err
		}
	}
}
