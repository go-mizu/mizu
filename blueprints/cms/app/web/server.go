// Package web provides the HTTP server.
package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/cms/app/web/handler/rest"
	"github.com/go-mizu/blueprints/cms/app/web/middleware"
	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/config/collections"
	"github.com/go-mizu/blueprints/cms/config/globals"
	"github.com/go-mizu/blueprints/cms/feature/auth"
	collectionsSvc "github.com/go-mizu/blueprints/cms/feature/collections"
	globalsSvc "github.com/go-mizu/blueprints/cms/feature/globals"
	prefsSvc "github.com/go-mizu/blueprints/cms/feature/preferences"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
	"github.com/go-mizu/mizu"
)

// Config holds server configuration.
type Config struct {
	Port      int
	DBPath    string
	Secret    string
	Dev       bool
	UploadDir string
}

// Server is the HTTP server.
type Server struct {
	config      Config
	app         *mizu.App
	store       *duckdb.Store
	collections []config.CollectionConfig
	globals     []config.GlobalConfig

	// Services
	authService        auth.API
	collectionsService collectionsSvc.API
	globalsService     globalsSvc.API
	prefsService       prefsSvc.API
}

// New creates a new Server.
func New(cfg Config) (*Server, error) {
	// Default config
	if cfg.Port == 0 {
		cfg.Port = 3000
	}
	if cfg.DBPath == "" {
		home, _ := os.UserHomeDir()
		cfg.DBPath = filepath.Join(home, "data", "blueprint", "cms", "cms.db")
	}
	if cfg.Secret == "" {
		cfg.Secret = "change-me-in-production"
	}
	if cfg.UploadDir == "" {
		cfg.UploadDir = "./uploads"
	}

	// Open store
	store, err := duckdb.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// Default collections and globals
	allCollections := []config.CollectionConfig{
		collections.Users,
		collections.Media,
		collections.Pages,
		collections.Posts,
		collections.Categories,
		collections.Tags,
	}
	allGlobals := []config.GlobalConfig{
		globals.SiteSettings,
		globals.Navigation,
	}

	// Create services
	authSvc := auth.NewService(store.DB(), store.Sessions, cfg.Secret, allCollections)
	collSvc := collectionsSvc.NewService(store.Collections, store.Versions, allCollections)
	globSvc := globalsSvc.NewService(store.Globals, store.Versions, allGlobals)
	prefSvc := prefsSvc.NewService(store.Preferences)

	s := &Server{
		config:             cfg,
		store:              store,
		collections:        allCollections,
		globals:            allGlobals,
		authService:        authSvc,
		collectionsService: collSvc,
		globalsService:     globSvc,
		prefsService:       prefSvc,
	}

	s.setupRoutes()

	return s, nil
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	s.app = mizu.New()

	// Middleware
	authMiddleware := middleware.NewAuth(s.authService)

	// Handlers
	collectionsHandler := rest.NewCollections(s.collectionsService)
	authHandler := rest.NewAuth(s.authService)
	globalsHandler := rest.NewGlobals(s.globalsService)
	accessHandler := rest.NewAccess(s.collections, s.globals)
	prefsHandler := rest.NewPreferences(s.prefsService)

	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Access endpoint
	s.app.Get("/api/access", authMiddleware.OptionalAuth(accessHandler.GetAccess))

	// Auth endpoints for each auth-enabled collection
	for _, col := range s.collections {
		if col.Auth == nil {
			continue
		}
		slug := col.Slug
		prefix := "/api/" + slug

		s.app.Post(prefix+"/login", authHandler.LoginWithCollection(slug))
		s.app.Post(prefix+"/logout", authHandler.LogoutWithCollection(slug))
		s.app.Get(prefix+"/me", authHandler.MeWithCollection(slug))
		s.app.Post(prefix+"/refresh-token", authHandler.RefreshTokenWithCollection(slug))
		s.app.Post(prefix+"/forgot-password", authHandler.ForgotPasswordWithCollection(slug))
		s.app.Post(prefix+"/reset-password", authHandler.ResetPasswordWithCollection(slug))
		s.app.Post(prefix+"/verify/{token}", authHandler.VerifyEmailWithCollection(slug))
		s.app.Post(prefix+"/unlock", authMiddleware.RequireAuth(authHandler.UnlockWithCollection(slug)))
	}

	// Collection CRUD endpoints
	for _, col := range s.collections {
		slug := col.Slug
		prefix := "/api/" + slug

		// Read operations - optional auth (access control may allow public)
		s.app.Get(prefix, authMiddleware.OptionalAuth(collectionsHandler.FindWithCollection(slug)))
		s.app.Get(prefix+"/count", authMiddleware.OptionalAuth(collectionsHandler.CountWithCollection(slug)))
		s.app.Get(prefix+"/{id}", authMiddleware.OptionalAuth(collectionsHandler.FindByIDWithCollection(slug)))

		// Write operations - require auth
		s.app.Post(prefix, authMiddleware.RequireAuth(collectionsHandler.CreateWithCollection(slug)))
		s.app.Patch(prefix, authMiddleware.RequireAuth(collectionsHandler.UpdateWithCollection(slug)))
		s.app.Patch(prefix+"/{id}", authMiddleware.RequireAuth(collectionsHandler.UpdateByIDWithCollection(slug)))
		s.app.Delete(prefix, authMiddleware.RequireAuth(collectionsHandler.DeleteWithCollection(slug)))
		s.app.Delete(prefix+"/{id}", authMiddleware.RequireAuth(collectionsHandler.DeleteByIDWithCollection(slug)))
	}

	// Global endpoints
	for _, g := range s.globals {
		slug := g.Slug
		prefix := "/api/globals/" + slug

		s.app.Get(prefix, authMiddleware.OptionalAuth(globalsHandler.GetWithSlug(slug)))
		s.app.Post(prefix, authMiddleware.RequireAuth(globalsHandler.UpdateWithSlug(slug)))
	}

	// Preferences endpoints
	s.app.Get("/api/payload-preferences/{key}", authMiddleware.RequireAuth(prefsHandler.Get))
	s.app.Post("/api/payload-preferences/{key}", authMiddleware.RequireAuth(prefsHandler.Set))
	s.app.Delete("/api/payload-preferences/{key}", authMiddleware.RequireAuth(prefsHandler.Delete))

	// Static file serving for uploads
	s.app.Static("/uploads", http.Dir(s.config.UploadDir))
}

// Run starts the server.
func (s *Server) Run() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	fmt.Printf("CMS server listening on http://localhost%s\n", addr)
	fmt.Printf("REST API: http://localhost%s/api\n", addr)
	return s.app.Listen(addr)
}

// Close closes the server.
func (s *Server) Close() error {
	return s.store.Close()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Give in-flight requests time to complete
	<-time.After(100 * time.Millisecond)
	return s.Close()
}

// Getters for services (for seeding, testing, etc.)

func (s *Server) AuthService() auth.API {
	return s.authService
}

func (s *Server) CollectionsService() collectionsSvc.API {
	return s.collectionsService
}

func (s *Server) GlobalsService() globalsSvc.API {
	return s.globalsService
}

func (s *Server) Store() *duckdb.Store {
	return s.store
}
