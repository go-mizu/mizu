// app/web/server.go
package web

import (
	"log"
	"time"

	"github.com/go-mizu/blueprints/finewiki/feature/api"
	"github.com/go-mizu/blueprints/finewiki/feature/search"
	"github.com/go-mizu/blueprints/finewiki/feature/view"
	"github.com/go-mizu/mizu"
	contract "github.com/go-mizu/mizu/contract/v2"
	"github.com/go-mizu/mizu/contract/v2/transport/rest"
	"github.com/go-mizu/mizu/openapi"
)

type Server struct {
	app *mizu.App

	view     view.API
	search   search.API
	tmpl     Templates
	contract contract.Invoker
}

func New(viewAPI view.API, searchAPI search.API, tmpl Templates) *Server {
	// Create the API service that implements the WikiAPI contract
	apiSvc := api.New(viewAPI, searchAPI)

	// Register the contract with HTTP bindings
	inv := contract.Register[api.WikiAPI](apiSvc,
		contract.WithName("FineWiki API"),
		contract.WithDescription("Read-only wiki API powered by DuckDB"),
		contract.WithDefaultResource("wiki"),
		contract.WithHTTP(map[string]contract.HTTPBinding{
			"GetPage": {Method: "GET", Path: "/pages"},
			"Search":  {Method: "GET", Path: "/search"},
		}),
	)

	s := &Server{
		app:      mizu.New(),
		view:     viewAPI,
		search:   searchAPI,
		tmpl:     tmpl,
		contract: inv,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	r := s.app

	r.Use(logging())

	// HTML pages (/ handles both home and search)
	r.Get("/", s.searchPage)
	r.Get("/page", s.page)

	r.Get("/healthz", func(c *mizu.Ctx) error {
		return c.Text(200, "ok")
	})

	// Mount REST API at /api
	if err := rest.MountAt(r.Router, "/api", s.contract); err != nil {
		log.Printf("warning: failed to mount API: %v", err)
	}

	// OpenAPI spec
	r.Get("/openapi.json", func(c *mizu.Ctx) error {
		spec, err := rest.OpenAPI(s.contract.Descriptor())
		if err != nil {
			return c.JSON(500, map[string]string{"error": err.Error()})
		}
		c.Header().Set("Content-Type", "application/json")
		_, err = c.Write(spec)
		return err
	})

	// API documentation UI
	docsHandler, err := openapi.NewHandler(openapi.Config{
		SpecURL:   "/openapi.json",
		DefaultUI: "scalar",
	})
	if err != nil {
		log.Printf("warning: failed to create docs handler: %v", err)
	} else {
		r.Get("/docs", func(c *mizu.Ctx) error {
			docsHandler.ServeHTTP(c.Writer(), c.Request())
			return nil
		})
	}
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
