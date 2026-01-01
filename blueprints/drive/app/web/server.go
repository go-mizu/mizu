package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/app/web/handler"
	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
	"github.com/go-mizu/blueprints/drive/feature/shares"
	"github.com/go-mizu/blueprints/drive/storage/local"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

// Server is the HTTP server.
type Server struct {
	app    *mizu.App
	store  *duckdb.Store
	config Config
}

// NewServer creates a new server.
func NewServer(cfg Config) (*Server, error) {
	// Open database
	store, err := duckdb.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Ensure schema
	if err := store.Ensure(context.Background()); err != nil {
		store.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create storage
	storage, err := local.New(cfg.DataDir)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("create storage: %w", err)
	}

	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	foldersSvc := folders.NewService(store.Folders())
	filesSvc := files.NewService(
		store.Files(),
		store.FileVersions(),
		store.ChunkedUploads(),
		foldersSvc,
		accountsSvc,
		storage,
	)

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost" + cfg.Addr
	}
	sharesSvc := shares.NewService(store.Shares(), store.ShareLinks(), baseURL)

	// Create handlers
	authHandler := handler.NewAuth(accountsSvc)
	filesHandler := handler.NewFiles(filesSvc, accountsSvc)
	foldersHandler := handler.NewFolders(foldersSvc, filesSvc)
	sharesHandler := handler.NewShares(sharesSvc, filesSvc, foldersSvc)

	// Create app
	app := mizu.New()

	// Middleware
	app.Use(handler.Logger)
	app.Use(handler.Recover)

	// Create router
	router := app.Router

	// Auth routes
	router.Post("/api/v1/auth/register", authHandler.Register)
	router.Post("/api/v1/auth/login", authHandler.Login)
	router.Post("/api/v1/auth/logout", authHandler.Logout)
	router.Get("/api/v1/auth/me", handler.RequireAuth(accountsSvc, authHandler.Me))
	router.Patch("/api/v1/auth/password", handler.RequireAuth(accountsSvc, authHandler.ChangePassword))

	// Account routes
	router.Get("/api/v1/accounts/me/storage", handler.RequireAuth(accountsSvc, authHandler.StorageUsage))

	// File routes
	router.Post("/api/v1/files", handler.RequireAuth(accountsSvc, filesHandler.Upload))
	router.Get("/api/v1/files/{id}", handler.RequireAuth(accountsSvc, filesHandler.Get))
	router.Get("/api/v1/files/{id}/download", handler.RequireAuth(accountsSvc, filesHandler.Download))
	router.Patch("/api/v1/files/{id}", handler.RequireAuth(accountsSvc, filesHandler.Update))
	router.Post("/api/v1/files/{id}/move", handler.RequireAuth(accountsSvc, filesHandler.Move))
	router.Post("/api/v1/files/{id}/copy", handler.RequireAuth(accountsSvc, filesHandler.Copy))
	router.Delete("/api/v1/files/{id}", handler.RequireAuth(accountsSvc, filesHandler.Delete))
	router.Post("/api/v1/files/{id}/star", handler.RequireAuth(accountsSvc, filesHandler.Star))
	router.Delete("/api/v1/files/{id}/star", handler.RequireAuth(accountsSvc, filesHandler.Unstar))
	router.Post("/api/v1/files/{id}/lock", handler.RequireAuth(accountsSvc, filesHandler.Lock))
	router.Delete("/api/v1/files/{id}/lock", handler.RequireAuth(accountsSvc, filesHandler.Unlock))

	// Chunked upload routes (using /uploads prefix to avoid route conflicts)
	router.Post("/api/v1/uploads", handler.RequireAuth(accountsSvc, filesHandler.CreateChunkedUpload))
	router.Put("/api/v1/uploads/{id}/chunk/{index}", handler.RequireAuth(accountsSvc, filesHandler.UploadChunk))
	router.Get("/api/v1/uploads/{id}", handler.RequireAuth(accountsSvc, filesHandler.GetUploadProgress))
	router.Post("/api/v1/uploads/{id}/complete", handler.RequireAuth(accountsSvc, filesHandler.CompleteUpload))

	// Version routes
	router.Get("/api/v1/files/{id}/versions", handler.RequireAuth(accountsSvc, filesHandler.ListVersions))
	router.Post("/api/v1/files/{id}/versions", handler.RequireAuth(accountsSvc, filesHandler.UploadVersion))
	router.Get("/api/v1/files/{id}/versions/{version}", handler.RequireAuth(accountsSvc, filesHandler.DownloadVersion))

	// Folder routes
	router.Post("/api/v1/folders", handler.RequireAuth(accountsSvc, foldersHandler.Create))
	router.Get("/api/v1/folders/{id}", handler.RequireAuth(accountsSvc, foldersHandler.Get))
	router.Get("/api/v1/folders/{id}/contents", handler.RequireAuth(accountsSvc, foldersHandler.Contents))
	router.Get("/api/v1/folders/{id}/tree", handler.RequireAuth(accountsSvc, foldersHandler.Tree))
	router.Patch("/api/v1/folders/{id}", handler.RequireAuth(accountsSvc, foldersHandler.Update))
	router.Post("/api/v1/folders/{id}/move", handler.RequireAuth(accountsSvc, foldersHandler.Move))
	router.Delete("/api/v1/folders/{id}", handler.RequireAuth(accountsSvc, foldersHandler.Delete))
	router.Post("/api/v1/folders/{id}/star", handler.RequireAuth(accountsSvc, foldersHandler.Star))
	router.Delete("/api/v1/folders/{id}/star", handler.RequireAuth(accountsSvc, foldersHandler.Unstar))

	// Share routes
	router.Post("/api/v1/shares", handler.RequireAuth(accountsSvc, sharesHandler.Create))
	router.Get("/api/v1/shares", handler.RequireAuth(accountsSvc, sharesHandler.ListByOwner))
	router.Get("/api/v1/shares/with-me", handler.RequireAuth(accountsSvc, sharesHandler.ListSharedWithMe))
	router.Delete("/api/v1/shares/{id}", handler.RequireAuth(accountsSvc, sharesHandler.Delete))

	// Share link routes
	router.Post("/api/v1/share-links", handler.RequireAuth(accountsSvc, sharesHandler.CreateLink))
	router.Get("/api/v1/share-links/{type}/{id}", handler.RequireAuth(accountsSvc, sharesHandler.ListLinksForItem))
	router.Patch("/api/v1/share-links/{id}", handler.RequireAuth(accountsSvc, sharesHandler.UpdateLink))
	router.Delete("/api/v1/share-links/{id}", handler.RequireAuth(accountsSvc, sharesHandler.DeleteLink))

	// Public share link access
	router.Get("/s/{token}", sharesHandler.AccessLink)
	router.Get("/s/{token}/download", sharesHandler.DownloadLink)
	router.Post("/s/{token}/verify", sharesHandler.VerifyLinkPassword)

	// View routes
	router.Get("/api/v1/recent", handler.RequireAuth(accountsSvc, filesHandler.ListRecent))
	router.Get("/api/v1/starred", handler.RequireAuth(accountsSvc, foldersHandler.ListStarred))
	router.Get("/api/v1/root", handler.RequireAuth(accountsSvc, foldersHandler.Root))

	// Health check
	router.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return &Server{
		app:    app,
		store:  store,
		config: cfg,
	}, nil
}

// ListenAndServe starts the server.
func (s *Server) ListenAndServe() error {
	return s.app.Listen(s.config.Addr)
}

// Close closes the server.
func (s *Server) Close() error {
	return s.store.Close()
}
