package web

import (
	"context"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/search/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/search/assets"
	"github.com/go-mizu/mizu/blueprints/search/feature/ai"
	"github.com/go-mizu/mizu/blueprints/search/feature/canvas"
	"github.com/go-mizu/mizu/blueprints/search/feature/chunker"
	"github.com/go-mizu/mizu/blueprints/search/feature/search"
	"github.com/go-mizu/mizu/blueprints/search/feature/session"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/searxng"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm/llamacpp"
	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/store/sqlite"
)

// NewServer creates a new HTTP server
func NewServer(st store.Store, devMode bool) (http.Handler, error) {
	app := mizu.New()

	// Create search handler with SearXNG if available
	searchHandler := createSearchHandler(st)
	suggestHandler := api.NewSuggestHandler(st)
	instantHandler := api.NewInstantHandler()
	knowledgeHandler := api.NewKnowledgeHandler(st)
	prefsHandler := api.NewPreferencesHandler(st)
	lensHandler := api.NewLensHandler(st)
	historyHandler := api.NewHistoryHandler(st)
	settingsHandler := api.NewSettingsHandler(st)
	indexHandler := api.NewIndexHandler(st)

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

		// AI Mode (only if services are available)
		if sqliteStore, ok := st.(*sqlite.Store); ok {
			aiHandler := createAIHandler(sqliteStore, searchHandler)
			if aiHandler != nil {
				apiGroup.Group("/ai", aiHandler.Register)
			}
		}
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

// createAIHandler creates the AI handler with LLM providers.
func createAIHandler(st *sqlite.Store, searchHandler *api.SearchHandler) *api.AIHandler {
	// Get LLM URLs from environment
	quickURL := os.Getenv("LLAMACPP_QUICK_URL")
	if quickURL == "" {
		quickURL = "http://localhost:8082"
	}
	deepURL := os.Getenv("LLAMACPP_DEEP_URL")
	if deepURL == "" {
		deepURL = "http://localhost:8083"
	}
	researchURL := os.Getenv("LLAMACPP_RESEARCH_URL")
	if researchURL == "" {
		researchURL = "http://localhost:8084"
	}

	// Try to create LLM providers
	var quickProvider, deepProvider, researchProvider llm.Provider

	if client, err := llamacpp.New(llamacpp.Config{BaseURL: quickURL, Timeout: 120 * time.Second}); err == nil {
		if err := client.Ping(context.Background()); err == nil {
			quickProvider = client
		}
	}
	if client, err := llamacpp.New(llamacpp.Config{BaseURL: deepURL, Timeout: 300 * time.Second}); err == nil {
		if err := client.Ping(context.Background()); err == nil {
			deepProvider = client
		}
	}
	if client, err := llamacpp.New(llamacpp.Config{BaseURL: researchURL, Timeout: 600 * time.Second}); err == nil {
		if err := client.Ping(context.Background()); err == nil {
			researchProvider = client
		}
	}

	// If no providers available, return nil (AI mode disabled)
	if quickProvider == nil && deepProvider == nil && researchProvider == nil {
		return nil
	}

	// Create session service
	sessionSvc := session.New(st.Session())

	// Create canvas service
	canvasSvc := canvas.New(st.Canvas())

	// Create chunker service (use quick provider for embeddings)
	var chunkerSvc *chunker.Service
	if quickProvider != nil {
		chunkerSvc = chunker.New(st.Chunker(), quickProvider, chunker.Config{
			ChunkSize:    1000,
			ChunkOverlap: 200,
			MaxChunks:    100,
		})
	}

	// Use the same search service from the search handler (has SearXNG if available)
	searchSvc := searchHandler.Service()

	// Create AI service
	aiSvc := ai.New(ai.Config{
		QuickProvider:    quickProvider,
		DeepProvider:     deepProvider,
		ResearchProvider: researchProvider,
		MaxIterations:    10,
		MaxSources:       10,
	}, searchSvc, chunkerSvc, sessionSvc)

	// Create model registry
	registry := ai.NewModelRegistry()

	// Register models
	if quickProvider != nil {
		registry.RegisterModel(ai.ModelInfo{
			ID:           "gemma-3-270m",
			Provider:     "llamacpp",
			Name:         "Gemma 3 270M",
			Description:  "Fast, lightweight model for quick answers",
			Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityEmbeddings},
			ContextSize:  4096,
			Speed:        "fast",
			Available:    true,
		}, quickProvider)
	}

	if deepProvider != nil {
		registry.RegisterModel(ai.ModelInfo{
			ID:           "gemma-3-1b",
			Provider:     "llamacpp",
			Name:         "Gemma 3 1B",
			Description:  "Balanced model for detailed analysis",
			Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityEmbeddings},
			ContextSize:  8192,
			Speed:        "balanced",
			Available:    true,
		}, deepProvider)
	}

	if researchProvider != nil {
		registry.RegisterModel(ai.ModelInfo{
			ID:           "gemma-3-4b",
			Provider:     "llamacpp",
			Name:         "Gemma 3 4B",
			Description:  "Comprehensive model for in-depth research",
			Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityEmbeddings},
			ContextSize:  16384,
			Speed:        "thorough",
			Available:    true,
		}, researchProvider)
	}

	return api.NewAIHandler(aiSvc, sessionSvc, canvasSvc, registry)
}

// createSearchHandler creates a search handler with SearXNG if available.
func createSearchHandler(st store.Store) *api.SearchHandler {
	// Get SearXNG URL from environment or use default
	searxngURL := os.Getenv("SEARXNG_URL")
	if searxngURL == "" {
		searxngURL = "http://localhost:8888"
	}

	// Try to connect to SearXNG
	eng := searxng.New(searxngURL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := eng.Healthz(ctx); err != nil {
		// SearXNG not available, use store fallback
		return api.NewSearchHandler(st)
	}

	// SearXNG is available, create cache and use engine
	var cache *search.Cache
	if sqliteStore, ok := st.(*sqlite.Store); ok {
		cacheStore := sqliteStore.Cache()
		cache = search.NewCacheWithDefaults(cacheStore)
	}

	return api.NewSearchHandlerWithConfig(search.ServiceConfig{
		Engine: eng,
		Cache:  cache,
		Store:  st,
	})
}
