package web

import (
	"io/fs"
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/book/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/book/assets"
	"github.com/go-mizu/mizu/blueprints/book/pkg/openlibrary"
	"github.com/go-mizu/mizu/blueprints/book/store"
)

// NewServer creates a new HTTP server
func NewServer(st store.Store, devMode bool) (http.Handler, error) {
	app := mizu.New()

	olClient := openlibrary.NewClient()

	bookHandler := api.NewBookHandler(st, olClient)
	authorHandler := api.NewAuthorHandler(st, olClient)
	shelfHandler := api.NewShelfHandler(st)
	reviewHandler := api.NewReviewHandler(st)
	browseHandler := api.NewBrowseHandler(st)
	challengeHandler := api.NewChallengeHandler(st)
	listHandler := api.NewListHandler(st)
	quoteHandler := api.NewQuoteHandler(st)
	statsHandler := api.NewStatsHandler(st)
	importHandler := api.NewImportExportHandler(st)
	feedHandler := api.NewFeedHandler(st)
	goodreadsHandler := api.NewGoodreadsHandler(st)

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// API routes
	app.Group("/api", func(r *mizu.Router) {
		// Books
		r.Get("/books/search", bookHandler.Search)
		r.Get("/books/trending", bookHandler.Trending)
		r.Get("/books/{id}", bookHandler.Get)
		r.Post("/books", bookHandler.Create)
		r.Get("/books/{id}/similar", bookHandler.Similar)

		// Authors
		r.Get("/authors/search", authorHandler.Search)
		r.Get("/authors/{id}", authorHandler.Get)
		r.Get("/authors/{id}/books", authorHandler.Books)

		// Shelves
		r.Get("/shelves", shelfHandler.List)
		r.Post("/shelves", shelfHandler.Create)
		r.Put("/shelves/{id}", shelfHandler.Update)
		r.Delete("/shelves/{id}", shelfHandler.Delete)
		r.Get("/shelves/{id}/books", shelfHandler.GetBooks)
		r.Post("/shelves/{id}/books", shelfHandler.AddBook)
		r.Delete("/shelves/{id}/books/{bookId}", shelfHandler.RemoveBook)

		// Reviews
		r.Get("/books/{id}/reviews", reviewHandler.GetByBook)
		r.Post("/books/{id}/reviews", reviewHandler.Create)
		r.Put("/reviews/{id}", reviewHandler.Update)
		r.Delete("/reviews/{id}", reviewHandler.Delete)

		// Reading Progress
		r.Get("/books/{id}/progress", reviewHandler.GetProgress)
		r.Post("/books/{id}/progress", reviewHandler.UpdateProgress)

		// Reading Challenge
		r.Get("/challenge/{year}", challengeHandler.Get)
		r.Post("/challenge", challengeHandler.Set)

		// Browse
		r.Get("/genres", browseHandler.ListGenres)
		r.Get("/genres/{genre}/books", browseHandler.BooksByGenre)
		r.Get("/browse/new-releases", browseHandler.NewReleases)
		r.Get("/browse/popular", browseHandler.Popular)

		// Lists
		r.Get("/lists", listHandler.GetAll)
		r.Post("/lists", listHandler.Create)
		r.Get("/lists/{id}", listHandler.Get)
		r.Post("/lists/{id}/books", listHandler.AddBook)
		r.Post("/lists/{id}/vote/{bookId}", listHandler.Vote)

		// Quotes
		r.Get("/quotes", quoteHandler.GetAll)
		r.Post("/quotes", quoteHandler.Create)
		r.Get("/books/{id}/quotes", quoteHandler.GetByBook)

		// Stats
		r.Get("/stats", statsHandler.Overall)
		r.Get("/stats/{year}", statsHandler.ByYear)

		// Import/Export
		r.Post("/import/csv", importHandler.ImportCSV)
		r.Get("/export/csv", importHandler.ExportCSV)

		// Goodreads
		r.Get("/goodreads/{id}", goodreadsHandler.GetByGoodreadsID)
		r.Post("/import-goodreads", goodreadsHandler.ImportFromURL)

		// Feed
		r.Get("/feed", feedHandler.Recent)
	})

	// Serve frontend
	if devMode {
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			return c.Text(200, "Frontend running on http://localhost:5173")
		})
	} else {
		staticContent, err := fs.Sub(assets.StaticFS, "static")
		if err != nil {
			return nil, err
		}

		indexHTML, err := fs.ReadFile(staticContent, "index.html")
		if err != nil {
			return nil, err
		}

		fileServer := http.FileServer(http.FS(staticContent))
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			if path == "/" {
				path = "/index.html"
			}

			if info, err := fs.Stat(staticContent, path[1:]); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer(), c.Request())
				return nil
			}

			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(200, string(indexHTML))
		})
	}

	return app, nil
}
