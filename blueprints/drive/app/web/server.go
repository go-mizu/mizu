package web

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/app/web/handler"
	"github.com/go-mizu/blueprints/drive/app/web/handler/api"
	"github.com/go-mizu/blueprints/drive/assets"
	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/activity"
	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
	"github.com/go-mizu/blueprints/drive/feature/shares"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

// Server is the HTTP server.
type Server struct {
	app *mizu.App
	cfg Config
	db  *sql.DB

	// Services
	accounts accounts.API
	files    files.API
	folders  folders.API
	shares   shares.API
	activity activity.API

	// Handlers
	authHandlers     *api.Auth
	fileHandlers     *api.Files
	folderHandlers   *api.Folders
	shareHandlers    *api.Shares
	activityHandlers *api.Activity
	pageHandlers     *handler.Page
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Create storage directory
	storageDir := filepath.Join(cfg.DataDir, "storage")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "drive.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create store
	store, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Initialize schema
	if err := store.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create services
	accountsSvc := accounts.NewService(store)
	filesSvc := files.NewService(store)
	foldersSvc := folders.NewService(store)
	sharesSvc := shares.NewService(store)
	activitySvc := activity.NewService(store)

	s := &Server{
		app:      mizu.New(),
		cfg:      cfg,
		db:       db,
		accounts: accountsSvc,
		files:    filesSvc,
		folders:  foldersSvc,
		shares:   sharesSvc,
		activity: activitySvc,
	}

	// Parse templates
	templates, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	// Create handlers
	s.authHandlers = api.NewAuth(accountsSvc, s.getUserID)
	s.fileHandlers = api.NewFiles(filesSvc, activitySvc, s.getUserID)
	s.folderHandlers = api.NewFolders(foldersSvc, activitySvc, s.getUserID)
	s.shareHandlers = api.NewShares(sharesSvc, s.getUserID)
	s.activityHandlers = api.NewActivity(activitySvc, s.getUserID)

	// Set storage root - defaults to user's home directory Downloads folder
	storageRoot := cfg.StorageRoot
	if storageRoot == "" {
		home, _ := os.UserHomeDir()
		storageRoot = filepath.Join(home, "Downloads")
	}

	// Create page handlers
	s.pageHandlers = handler.NewPage(
		templates,
		accountsSvc,
		filesSvc,
		foldersSvc,
		sharesSvc,
		activitySvc,
		s.getUserID,
		storageRoot,
	)

	s.setupRoutes()

	// Serve static files with caching
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		c.Writer().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	log.Printf("Starting Drive server on %s", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() *mizu.App {
	return s.app
}

// AccountService returns the accounts service.
func (s *Server) AccountService() accounts.API {
	return s.accounts
}

// FileService returns the files service.
func (s *Server) FileService() files.API {
	return s.files
}

// FolderService returns the folders service.
func (s *Server) FolderService() folders.API {
	return s.folders
}

// ShareService returns the shares service.
func (s *Server) ShareService() shares.API {
	return s.shares
}

// ActivityService returns the activity service.
func (s *Server) ActivityService() activity.API {
	return s.activity
}

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Page routes - local mode doesn't require auth, redirect to files directly
	s.app.Get("/", func(c *mizu.Ctx) error {
		http.Redirect(c.Writer(), c.Request(), "/files", http.StatusFound)
		return nil
	})

	s.app.Get("/login", s.pageHandlers.Login)
	s.app.Get("/register", s.pageHandlers.Register)
	s.app.Get("/files", s.pageHandlers.Files)
	s.app.Get("/files/{path...}", s.pageHandlers.Files)
	s.app.Get("/shared", s.pageHandlers.Shared)
	s.app.Get("/recent", s.pageHandlers.Recent)
	s.app.Get("/starred", s.pageHandlers.Starred)
	s.app.Get("/trash", s.pageHandlers.Trash)
	s.app.Get("/search", s.pageHandlers.Search)
	s.app.Get("/settings", s.pageHandlers.Settings)
	s.app.Get("/activity", s.pageHandlers.Activity)

	// Preview page
	s.app.Get("/preview/{id...}", s.pageHandlers.Preview)

	// File content and thumbnail (for local mode)
	s.app.Get("/api/v1/content/{id...}", s.pageHandlers.Content)
	s.app.Get("/api/v1/thumbnail/{id...}", s.pageHandlers.Thumbnail)
	s.app.Get("/api/v1/metadata/{id...}", s.pageHandlers.Metadata)
	s.app.Get("/api/v1/folder-children/{id...}", s.pageHandlers.FolderChildren)
	s.app.Get("/api/v1/folder-children", s.pageHandlers.FolderChildren)

	// Shared link access
	s.app.Get("/s/{token}", s.pageHandlers.ShareLink)

	// API routes
	s.app.Group("/api/v1", func(r *mizu.Router) {
		// Auth
		r.Post("/auth/register", s.authHandlers.Register)
		r.Post("/auth/login", s.authHandlers.Login)
		r.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		r.Get("/auth/me", s.authRequired(s.authHandlers.Me))
		r.Put("/auth/me", s.authRequired(s.authHandlers.Update))
		r.Post("/auth/password", s.authRequired(s.authHandlers.ChangePassword))
		r.Get("/auth/sessions", s.authRequired(s.authHandlers.ListSessions))
		r.Delete("/auth/sessions/{id}", s.authRequired(s.authHandlers.DeleteSession))

		// Files
		r.Get("/files", s.authRequired(s.fileHandlers.List))
		r.Post("/files", s.authRequired(s.fileHandlers.Create))
		r.Get("/files/{id}", s.authRequired(s.fileHandlers.Get))
		r.Put("/files/{id}", s.authRequired(s.fileHandlers.Update))
		r.Delete("/files/{id}", s.authRequired(s.fileHandlers.Delete))
		r.Get("/files/{id}/download", s.authRequired(s.fileHandlers.Download))
		r.Post("/files/{id}/copy", s.authRequired(s.fileHandlers.Copy))
		r.Post("/files/{id}/move", s.authRequired(s.fileHandlers.Move))
		r.Put("/files/{id}/star", s.authRequired(s.fileHandlers.Star))
		r.Delete("/files/{id}/star", s.authRequired(s.fileHandlers.Unstar))
		r.Post("/files/{id}/trash", s.authRequired(s.fileHandlers.Trash))
		r.Post("/files/{id}/restore", s.authRequired(s.fileHandlers.Restore))
		r.Get("/files/{id}/versions", s.authRequired(s.fileHandlers.ListVersions))
		r.Post("/files/{id}/versions/{version}/restore", s.authRequired(s.fileHandlers.RestoreVersion))
		r.Get("/files/{id}/preview", s.authRequired(s.fileHandlers.Preview))

		// Folders
		r.Get("/folders", s.authRequired(s.folderHandlers.List))
		r.Post("/folders", s.authRequired(s.folderHandlers.Create))
		r.Get("/folders/{id}", s.authRequired(s.folderHandlers.Get))
		r.Get("/folders/{id}/contents", s.authRequired(s.folderHandlers.Contents))
		r.Put("/folders/{id}", s.authRequired(s.folderHandlers.Update))
		r.Delete("/folders/{id}", s.authRequired(s.folderHandlers.Delete))
		r.Post("/folders/{id}/move", s.authRequired(s.folderHandlers.Move))
		r.Put("/folders/{id}/star", s.authRequired(s.folderHandlers.Star))
		r.Delete("/folders/{id}/star", s.authRequired(s.folderHandlers.Unstar))
		r.Post("/folders/{id}/trash", s.authRequired(s.folderHandlers.Trash))
		r.Post("/folders/{id}/restore", s.authRequired(s.folderHandlers.Restore))
		r.Get("/folders/{id}/path", s.authRequired(s.folderHandlers.Path))

		// Shares
		r.Get("/shares", s.authRequired(s.shareHandlers.List))
		r.Get("/shares/with-me", s.authRequired(s.shareHandlers.SharedWithMe))
		r.Post("/shares", s.authRequired(s.shareHandlers.Create))
		r.Post("/shares/link", s.authRequired(s.shareHandlers.CreateLink))
		r.Get("/shares/{id}", s.authRequired(s.shareHandlers.Get))
		r.Put("/shares/{id}", s.authRequired(s.shareHandlers.Update))
		r.Delete("/shares/{id}", s.authRequired(s.shareHandlers.Delete))
		r.Get("/shares/link/{token}", s.shareHandlers.GetByToken) // Public

		// Starred
		r.Get("/starred", s.authRequired(s.fileHandlers.ListStarred))

		// Recent
		r.Get("/recent", s.authRequired(s.fileHandlers.ListRecent))

		// Trash
		r.Get("/trash", s.authRequired(s.fileHandlers.ListTrashed))
		r.Delete("/trash", s.authRequired(s.fileHandlers.EmptyTrash))

		// Search
		r.Get("/search", s.authRequired(s.fileHandlers.Search))

		// Activity
		r.Get("/activity", s.authRequired(s.activityHandlers.List))
		r.Get("/activity/resource/{type}/{id}", s.authRequired(s.activityHandlers.ListForResource))

		// Storage
		r.Get("/storage", s.authRequired(s.authHandlers.StorageInfo))
	})
}
