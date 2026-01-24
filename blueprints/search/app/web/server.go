package web

import (
	"io/fs"
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/search/assets"
	"github.com/go-mizu/mizu/blueprints/search/store/postgres"
)

// NewServer creates a new HTTP server
func NewServer(store *postgres.Store, devMode bool) (http.Handler, error) {
	app := mizu.New()

	// Create handlers
	searchHandler := api.NewSearchHandler(store)
	suggestHandler := api.NewSuggestHandler(store)
	instantHandler := api.NewInstantHandler()
	knowledgeHandler := api.NewKnowledgeHandler(store)
	prefsHandler := api.NewPreferencesHandler(store)
	lensHandler := api.NewLensHandler(store)
	historyHandler := api.NewHistoryHandler(store)
	settingsHandler := api.NewSettingsHandler(store)
	indexHandler := api.NewIndexHandler(store)

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// API routes
	app.Group("/api", func(apiGroup *mizu.Router) {
		// Search
		apiGroup.Get("/search", searchHandler.Search)
		apiGroup.Get("/search/images", searchHandler.SearchImages)
		apiGroup.Get("/search/videos", searchHandler.SearchVideos)
		apiGroup.Get("/search/news", searchHandler.SearchNews)

		// Suggest
		apiGroup.Get("/suggest", suggestHandler.Suggest)
		apiGroup.Get("/suggest/trending", suggestHandler.Trending)

		// Instant answers
		apiGroup.Get("/instant/calculate", instantHandler.Calculate)
		apiGroup.Get("/instant/convert", instantHandler.Convert)
		apiGroup.Get("/instant/currency", instantHandler.Currency)
		apiGroup.Get("/instant/weather", instantHandler.Weather)
		apiGroup.Get("/instant/define", instantHandler.Define)
		apiGroup.Get("/instant/time", instantHandler.Time)

		// Knowledge
		apiGroup.Get("/knowledge/{query}", knowledgeHandler.GetEntity)

		// Preferences
		apiGroup.Get("/preferences", prefsHandler.List)
		apiGroup.Post("/preferences", prefsHandler.Set)
		apiGroup.Delete("/preferences/{domain}", prefsHandler.Delete)

		// Lenses
		apiGroup.Get("/lenses", lensHandler.List)
		apiGroup.Post("/lenses", lensHandler.Create)
		apiGroup.Get("/lenses/{id}", lensHandler.Get)
		apiGroup.Put("/lenses/{id}", lensHandler.Update)
		apiGroup.Delete("/lenses/{id}", lensHandler.Delete)

		// History
		apiGroup.Get("/history", historyHandler.List)
		apiGroup.Delete("/history", historyHandler.Clear)
		apiGroup.Delete("/history/{id}", historyHandler.Delete)

		// Settings
		apiGroup.Get("/settings", settingsHandler.Get)
		apiGroup.Put("/settings", settingsHandler.Update)

		// Index admin
		apiGroup.Get("/admin/index/stats", indexHandler.Stats)
		apiGroup.Post("/admin/index/rebuild", indexHandler.Rebuild)
	})

	// Serve frontend
	if devMode {
		// In dev mode, proxy to Vite dev server
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			return c.Text(200, "Frontend running on http://localhost:5173")
		})
	} else {
		// In production, serve embedded static files
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

			// Check if file exists
			if info, err := fs.Stat(staticContent, path[1:]); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer(), c.Request())
				return nil
			}

			// SPA fallback
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(200, string(indexHTML))
		})
	}

	return app, nil
}
