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
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/local"
	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/searxng"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/search/pkg/llm/claude"
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
	bangHandler := api.NewBangHandler(st)
	widgetHandler := api.NewWidgetHandler(st)
	enrichHandler := api.NewEnrichHandler(st)

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

		// Bangs
		apiGroup.Get("/bangs", bangHandler.List)
		apiGroup.Get("/bangs/parse", bangHandler.Parse)
		apiGroup.Post("/bangs", bangHandler.Create)
		apiGroup.Delete("/bangs/{id}", bangHandler.Delete)

		// Widgets
		apiGroup.Get("/widgets", widgetHandler.GetSettings)
		apiGroup.Put("/widgets", widgetHandler.UpdateSettings)
		apiGroup.Get("/cheatsheet/{language}", widgetHandler.GetCheatSheet)
		apiGroup.Get("/cheatsheets", widgetHandler.ListCheatSheets)
		apiGroup.Get("/related", widgetHandler.GetRelated)

		// Enrichment (Teclis/TinyGem style)
		apiGroup.Get("/enrich/web", enrichHandler.SearchWeb)
		apiGroup.Get("/enrich/news", enrichHandler.SearchNews)

		// AI Mode and Summarizer (only if services are available)
		if sqliteStore, ok := st.(*sqlite.Store); ok {
			// Create summarize handler (uses first available LLM provider)
			summarizeHandler := createSummarizeHandler(sqliteStore)
			if summarizeHandler != nil {
				apiGroup.Get("/summarize", summarizeHandler.Summarize)
				apiGroup.Post("/summarize", summarizeHandler.Summarize)
			}

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

// createSummarizeHandler creates the summarization handler with an LLM provider.
func createSummarizeHandler(st *sqlite.Store) *api.SummarizeHandler {
	// Try to find an available LLM provider
	quickURL := os.Getenv("LLAMACPP_QUICK_URL")
	if quickURL == "" {
		quickURL = "http://localhost:8082"
	}

	var provider llm.Provider
	if client, err := llamacpp.New(llamacpp.Config{BaseURL: quickURL, Timeout: 120 * time.Second}); err == nil {
		if err := client.Ping(context.Background()); err == nil {
			provider = client
		}
	}

	// Return handler even if no provider (will use simple extraction)
	return api.NewSummarizeHandler(st, provider)
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

	// Check for Claude API key
	claudeAPIKey := os.Getenv("ANTHROPIC_API_KEY")

	// Try to create LLM providers
	var quickProvider, deepProvider, researchProvider llm.Provider

	// Try local llama.cpp providers first
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

	// Try Claude providers as fallback or supplement
	var claudeQuickProvider, claudeDeepProvider, claudeResearchProvider llm.Provider
	if claudeAPIKey != "" {
		// Create Claude providers for each tier with appropriate models
		if client, err := claude.New(claude.Config{
			APIKey:  claudeAPIKey,
			Timeout: 120 * time.Second,
			Model:   "claude-haiku-4.5", // Quick tier
		}); err == nil {
			claudeQuickProvider = client
		}
		if client, err := claude.New(claude.Config{
			APIKey:  claudeAPIKey,
			Timeout: 300 * time.Second,
			Model:   "claude-sonnet-4.5", // Deep tier
		}); err == nil {
			claudeDeepProvider = client
		}
		if client, err := claude.New(claude.Config{
			APIKey:  claudeAPIKey,
			Timeout: 600 * time.Second,
			Model:   "claude-opus-4.5", // Research tier
		}); err == nil {
			claudeResearchProvider = client
		}
	}

	// Use Claude as fallback if local providers unavailable
	if quickProvider == nil && claudeQuickProvider != nil {
		quickProvider = claudeQuickProvider
	}
	if deepProvider == nil && claudeDeepProvider != nil {
		deepProvider = claudeDeepProvider
	}
	if researchProvider == nil && claudeResearchProvider != nil {
		researchProvider = claudeResearchProvider
	}

	// If no providers available, return nil (AI mode disabled)
	if quickProvider == nil && deepProvider == nil && researchProvider == nil {
		return nil
	}

	// Create session service
	sessionSvc := session.New(st.Session())

	// Create canvas service
	canvasSvc := canvas.New(st.Canvas())

	// Create chunker service (use quick provider for embeddings, prefer llama.cpp)
	var chunkerSvc *chunker.Service
	var embeddingProvider llm.Provider
	// Prefer llama.cpp for embeddings since Claude doesn't support them
	if client, err := llamacpp.New(llamacpp.Config{BaseURL: quickURL, Timeout: 120 * time.Second}); err == nil {
		if err := client.Ping(context.Background()); err == nil {
			embeddingProvider = client
		}
	}
	if embeddingProvider != nil {
		chunkerSvc = chunker.New(st.Chunker(), embeddingProvider, chunker.Config{
			ChunkSize:    1000,
			ChunkOverlap: 200,
			MaxChunks:    100,
		})
	}

	// Use the same search service from the search handler (has SearXNG if available)
	searchSvc := searchHandler.Service()

	// Create AI service with caching and logging
	aiSvc := ai.New(ai.Config{
		QuickProvider:    quickProvider,
		DeepProvider:     deepProvider,
		ResearchProvider: researchProvider,
		MaxIterations:    10,
		MaxSources:       10,
		CacheStore:       &llmCacheAdapter{st.LLMCache()},
		LogStore:         &llmLogAdapter{st.LLMLog()},
		CacheTTL:         24 * time.Hour,
	}, searchSvc, chunkerSvc, sessionSvc)

	// Create model registry
	registry := ai.NewModelRegistry()

	// Register llama.cpp models if available
	if llamaQuick, err := llamacpp.New(llamacpp.Config{BaseURL: quickURL, Timeout: 120 * time.Second}); err == nil {
		if llamaQuick.Ping(context.Background()) == nil {
			registry.RegisterModel(ai.ModelInfo{
				ID:           "gemma-3-270m",
				Provider:     "llamacpp",
				Name:         "Gemma 3 270M",
				Description:  "Fast, lightweight model for quick answers",
				Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityEmbeddings},
				ContextSize:  4096,
				Speed:        "fast",
				Available:    true,
			}, llamaQuick)
		}
	}

	if llamaDeep, err := llamacpp.New(llamacpp.Config{BaseURL: deepURL, Timeout: 300 * time.Second}); err == nil {
		if llamaDeep.Ping(context.Background()) == nil {
			registry.RegisterModel(ai.ModelInfo{
				ID:           "gemma-3-1b",
				Provider:     "llamacpp",
				Name:         "Gemma 3 1B",
				Description:  "Balanced model for detailed analysis",
				Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityEmbeddings},
				ContextSize:  8192,
				Speed:        "balanced",
				Available:    true,
			}, llamaDeep)
		}
	}

	if llamaResearch, err := llamacpp.New(llamacpp.Config{BaseURL: researchURL, Timeout: 600 * time.Second}); err == nil {
		if llamaResearch.Ping(context.Background()) == nil {
			registry.RegisterModel(ai.ModelInfo{
				ID:           "gemma-3-4b",
				Provider:     "llamacpp",
				Name:         "Gemma 3 4B",
				Description:  "Comprehensive model for in-depth research",
				Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityEmbeddings},
				ContextSize:  16384,
				Speed:        "thorough",
				Available:    true,
			}, llamaResearch)
		}
	}

	// Register Claude models if API key is available
	if claudeAPIKey != "" {
		if claudeQuickProvider != nil {
			registry.RegisterModel(ai.ModelInfo{
				ID:           "claude-haiku-4.5",
				Provider:     "claude",
				Name:         "Claude Haiku 4.5",
				Description:  "Fast Claude model for quick answers",
				Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityVision},
				ContextSize:  200000,
				Speed:        "fast",
				Available:    true,
			}, claudeQuickProvider)
		}

		if claudeDeepProvider != nil {
			registry.RegisterModel(ai.ModelInfo{
				ID:           "claude-sonnet-4.5",
				Provider:     "claude",
				Name:         "Claude Sonnet 4.5",
				Description:  "Balanced Claude model for detailed analysis",
				Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityVision},
				ContextSize:  200000,
				Speed:        "balanced",
				Available:    true,
			}, claudeDeepProvider)
		}

		if claudeResearchProvider != nil {
			registry.RegisterModel(ai.ModelInfo{
				ID:           "claude-opus-4.5",
				Provider:     "claude",
				Name:         "Claude Opus 4.5",
				Description:  "Most capable Claude model for comprehensive research",
				Capabilities: []ai.Capability{ai.CapabilityText, ai.CapabilityVision},
				ContextSize:  200000,
				Speed:        "thorough",
				Available:    true,
			}, claudeResearchProvider)
		}
	}

	return api.NewAIHandler(aiSvc, sessionSvc, canvasSvc, registry)
}

// createSearchHandler creates a search handler with local engine or SearXNG.
func createSearchHandler(st store.Store) *api.SearchHandler {
	var cache *search.Cache
	if sqliteStore, ok := st.(*sqlite.Store); ok {
		cacheStore := sqliteStore.Cache()
		cache = search.NewCacheWithDefaults(cacheStore)
	}

	// Check if we should use SearXNG (optional external instance)
	searxngURL := os.Getenv("SEARXNG_URL")
	if searxngURL != "" {
		// Try to connect to SearXNG
		eng := searxng.New(searxngURL)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := eng.Healthz(ctx); err == nil {
			// SearXNG is available, use it
			return api.NewSearchHandlerWithConfig(search.ServiceConfig{
				Engine: eng,
				Cache:  cache,
				Store:  st,
			}, st)
		}
	}

	// Use local metasearch engine (built-in, always available)
	localEngine := local.NewAdapterWithDefaults()

	return api.NewSearchHandlerWithConfig(search.ServiceConfig{
		Engine: localEngine,
		Cache:  cache,
		Store:  st,
	}, st)
}

// ========== LLM Store Adapters ==========

// llmCacheAdapter adapts sqlite.LLMCacheStore to ai.CacheStore.
type llmCacheAdapter struct {
	store *sqlite.LLMCacheStore
}

func (a *llmCacheAdapter) Get(ctx context.Context, queryHash, mode, model string) (*ai.CacheEntry, error) {
	entry, err := a.store.Get(ctx, queryHash, mode, model)
	if err != nil || entry == nil {
		return nil, err
	}
	return &ai.CacheEntry{
		QueryHash:        entry.QueryHash,
		Query:            entry.Query,
		Mode:             entry.Mode,
		Model:            entry.Model,
		ResponseText:     entry.ResponseText,
		Citations:        entry.Citations,
		FollowUps:        entry.FollowUps,
		RelatedQuestions: entry.RelatedQuestions,
		InputTokens:      entry.InputTokens,
		OutputTokens:     entry.OutputTokens,
		ExpiresAt:        entry.ExpiresAt,
	}, nil
}

func (a *llmCacheAdapter) Set(ctx context.Context, entry *ai.CacheEntry) error {
	return a.store.Set(ctx, &sqlite.LLMCacheEntry{
		QueryHash:        entry.QueryHash,
		Query:            entry.Query,
		Mode:             entry.Mode,
		Model:            entry.Model,
		ResponseText:     entry.ResponseText,
		Citations:        entry.Citations,
		FollowUps:        entry.FollowUps,
		RelatedQuestions: entry.RelatedQuestions,
		InputTokens:      entry.InputTokens,
		OutputTokens:     entry.OutputTokens,
		ExpiresAt:        entry.ExpiresAt,
	})
}

func (a *llmCacheAdapter) Delete(ctx context.Context, queryHash, mode, model string) error {
	return a.store.Delete(ctx, queryHash, mode, model)
}

func (a *llmCacheAdapter) GetStats(ctx context.Context) (map[string]any, error) {
	return a.store.GetStats(ctx)
}

// llmLogAdapter adapts sqlite.LLMLogStore to ai.LogStore.
type llmLogAdapter struct {
	store *sqlite.LLMLogStore
}

func (a *llmLogAdapter) Log(ctx context.Context, entry *ai.LogEntry) error {
	return a.store.Log(ctx, &sqlite.LLMLogEntry{
		RequestID:    entry.RequestID,
		Provider:     entry.Provider,
		Model:        entry.Model,
		Mode:         entry.Mode,
		Query:        entry.Query,
		RequestJSON:  entry.RequestJSON,
		ResponseJSON: entry.ResponseJSON,
		Status:       entry.Status,
		ErrorMessage: entry.ErrorMessage,
		InputTokens:  entry.InputTokens,
		OutputTokens: entry.OutputTokens,
		DurationMs:   entry.DurationMs,
		CostUSD:      entry.CostUSD,
	})
}

func (a *llmLogAdapter) GetStats(ctx context.Context) (map[string]any, error) {
	return a.store.GetStats(ctx)
}
