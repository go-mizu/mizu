package web

import (
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/lingo/assets"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/achievements"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/auth"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/courses"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/gamification"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/lessons"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/progress"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/shop"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/social"
	"github.com/go-mizu/mizu/blueprints/lingo/feature/users"
	"github.com/go-mizu/mizu/blueprints/lingo/store"
)

// NewServer creates a new HTTP server
func NewServer(st store.Store, devMode bool) (http.Handler, error) {
	app := mizu.New()

	// Create services
	authService := auth.NewService(st)
	userService := users.NewService(st)
	courseService := courses.NewService(st)
	lessonService := lessons.NewService(st)
	progressService := progress.NewService(st)
	gamificationService := gamification.NewService(st)
	socialService := social.NewService(st)
	achievementService := achievements.NewService(st)
	shopService := shop.NewService(st)

	// Create handlers
	authHandler := auth.NewHandler(authService)
	userHandler := users.NewHandler(userService)
	courseHandler := courses.NewHandler(courseService)
	lessonHandler := lessons.NewHandler(lessonService)
	progressHandler := progress.NewHandler(progressService)
	gamificationHandler := gamification.NewHandler(gamificationService)
	socialHandler := social.NewHandler(socialService)
	achievementHandler := achievements.NewHandler(achievementService)
	shopHandler := shop.NewHandler(shopService)

	// API routes
	app.Group("/api/v1", func(apiGroup *mizu.Router) {
		// Register all feature routes
		authHandler.RegisterRoutes(apiGroup)
		userHandler.RegisterRoutes(apiGroup)
		courseHandler.RegisterRoutes(apiGroup)
		lessonHandler.RegisterRoutes(apiGroup)
		progressHandler.RegisterRoutes(apiGroup)
		gamificationHandler.RegisterRoutes(apiGroup)
		socialHandler.RegisterRoutes(apiGroup)
		achievementHandler.RegisterRoutes(apiGroup)
		shopHandler.RegisterRoutes(apiGroup)
	})

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Serve frontend
	if devMode {
		// Proxy to Vite dev server
		viteURL, _ := url.Parse("http://localhost:5173")
		proxy := httputil.NewSingleHostReverseProxy(viteURL)
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			proxy.ServeHTTP(c.Writer(), c.Request())
			return nil
		})
	} else {
		// Serve embedded static files
		staticContent, err := fs.Sub(assets.StaticFS, "static")
		if err != nil {
			return nil, err
		}

		// Read index.html content for SPA fallback
		indexHTML, err := fs.ReadFile(staticContent, "index.html")
		if err != nil {
			return nil, err
		}

		fileServer := http.FileServer(http.FS(staticContent))
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			// Try to serve static file
			path := c.Request().URL.Path
			if path == "/" {
				path = "/index.html"
			}

			// Check if file exists (must be a file, not directory)
			if info, err := fs.Stat(staticContent, path[1:]); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer(), c.Request())
				return nil
			}

			// SPA fallback - serve index.html content directly
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(200, string(indexHTML))
		})
	}

	return app, nil
}
