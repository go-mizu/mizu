package web

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/app/web/handler/api"
	"github.com/go-mizu/blueprints/table/assets"
	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/comments"
	"github.com/go-mizu/blueprints/table/feature/dashboard"
	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/importexport"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/shares"
	"github.com/go-mizu/blueprints/table/feature/tables"
	"github.com/go-mizu/blueprints/table/feature/users"
	"github.com/go-mizu/blueprints/table/feature/views"
	"github.com/go-mizu/blueprints/table/feature/workspaces"
	"github.com/go-mizu/blueprints/table/store/sqlite"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app   *mizu.App
	cfg   Config
	store *sqlite.Store

	// Services
	users        *users.Service
	workspaces   *workspaces.Service
	bases        *bases.Service
	tables       *tables.Service
	fields       *fields.Service
	records      *records.Service
	views        *views.Service
	comments     *comments.Service
	shares       *shares.Service
	importexport *importexport.Service
	dashboard    *dashboard.Service

	// Handlers
	authHandlers       *api.Auth
	workspaceHandlers  *api.Workspace
	baseHandlers       *api.Base
	tableHandlers      *api.Table
	fieldHandlers      *api.Field
	recordHandlers     *api.Record
	viewHandlers       *api.View
	commentHandlers    *api.Comment
	shareHandlers      *api.Share
	publicFormHandlers *api.PublicForm
	dashboardHandlers  *api.Dashboard
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open store
	store, err := sqlite.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	// Create services
	usersSvc := users.NewService(store.Users(), store.DB())
	workspacesSvc := workspaces.NewService(store.Workspaces())
	basesSvc := bases.NewService(store.Bases())
	tablesSvc := tables.NewService(store.Tables())
	fieldsSvc := fields.NewService(store.Fields())
	recordsSvc := records.NewService(store.Records())
	viewsSvc := views.NewService(store.Views())
	commentsSvc := comments.NewService(store.Comments())
	sharesSvc := shares.NewService(store.Shares())
	importexportSvc := importexport.NewService(basesSvc, tablesSvc, fieldsSvc, recordsSvc, viewsSvc)
	dashboardSvc := dashboard.NewService(viewsSvc, recordsSvc, fieldsSvc)

	s := &Server{
		app:          mizu.New(),
		cfg:          cfg,
		store:        store,
		users:        usersSvc,
		workspaces:   workspacesSvc,
		bases:        basesSvc,
		tables:       tablesSvc,
		fields:       fieldsSvc,
		records:      recordsSvc,
		views:        viewsSvc,
		comments:     commentsSvc,
		shares:       sharesSvc,
		importexport: importexportSvc,
		dashboard:    dashboardSvc,
	}

	// Create handlers
	s.authHandlers = api.NewAuth(usersSvc, s.getUserID)
	s.workspaceHandlers = api.NewWorkspace(workspacesSvc, basesSvc, s.getUserID)
	s.baseHandlers = api.NewBase(basesSvc, tablesSvc, s.getUserID)
	s.tableHandlers = api.NewTable(tablesSvc, fieldsSvc, viewsSvc, s.getUserID)
	s.fieldHandlers = api.NewField(fieldsSvc, s.getUserID)
	s.recordHandlers = api.NewRecord(recordsSvc, s.getUserID)
	s.viewHandlers = api.NewView(viewsSvc, s.getUserID)
	s.commentHandlers = api.NewComment(commentsSvc, usersSvc, s.getUserID)
	s.shareHandlers = api.NewShare(sharesSvc, basesSvc, tablesSvc, s.getUserID)
	s.publicFormHandlers = api.NewPublicForm(viewsSvc, tablesSvc, fieldsSvc, recordsSvc)
	s.dashboardHandlers = api.NewDashboard(dashboardSvc, viewsSvc, s.getUserID)

	s.setupRoutes()

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	slog.Info("Starting Table server", "addr", s.cfg.Addr, "data", s.cfg.DataDir)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() *mizu.App {
	return s.app
}

// Service accessors for CLI use

// UserService returns the users service.
func (s *Server) UserService() *users.Service {
	return s.users
}

// WorkspaceService returns the workspaces service.
func (s *Server) WorkspaceService() *workspaces.Service {
	return s.workspaces
}

// BaseService returns the bases service.
func (s *Server) BaseService() *bases.Service {
	return s.bases
}

// TableService returns the tables service.
func (s *Server) TableService() *tables.Service {
	return s.tables
}

// FieldService returns the fields service.
func (s *Server) FieldService() *fields.Service {
	return s.fields
}

// RecordService returns the records service.
func (s *Server) RecordService() *records.Service {
	return s.records
}

// ViewService returns the views service.
func (s *Server) ViewService() *views.Service {
	return s.views
}

// ImportExportService returns the import/export service.
func (s *Server) ImportExportService() *importexport.Service {
	return s.importexport
}

// Init initializes the database schema (already done in New).
func (s *Server) Init(ctx context.Context) error {
	return nil
}

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// API routes
	s.app.Group("/api/v1", func(r *mizu.Router) {
		// Auth
		r.Post("/auth/register", s.authHandlers.Register)
		r.Post("/auth/login", s.authHandlers.Login)
		r.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		r.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Workspaces
		r.Get("/workspaces", s.authRequired(s.workspaceHandlers.List))
		r.Post("/workspaces", s.authRequired(s.workspaceHandlers.Create))
		r.Get("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Get))
		r.Patch("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Update))
		r.Delete("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Delete))
		r.Get("/workspaces/{id}/bases", s.authRequired(s.workspaceHandlers.ListBases))

		// Bases
		r.Post("/bases", s.authRequired(s.baseHandlers.Create))
		r.Get("/bases/{id}", s.authRequired(s.baseHandlers.Get))
		r.Patch("/bases/{id}", s.authRequired(s.baseHandlers.Update))
		r.Delete("/bases/{id}", s.authRequired(s.baseHandlers.Delete))
		r.Get("/bases/{id}/tables", s.authRequired(s.baseHandlers.ListTables))

		// Tables
		r.Post("/tables", s.authRequired(s.tableHandlers.Create))
		r.Get("/tables/{id}", s.authRequired(s.tableHandlers.Get))
		r.Patch("/tables/{id}", s.authRequired(s.tableHandlers.Update))
		r.Delete("/tables/{id}", s.authRequired(s.tableHandlers.Delete))
		r.Get("/tables/{id}/fields", s.authRequired(s.tableHandlers.ListFields))
		r.Get("/tables/{id}/views", s.authRequired(s.tableHandlers.ListViews))

		// Fields
		r.Post("/fields", s.authRequired(s.fieldHandlers.Create))
		r.Get("/fields/{id}", s.authRequired(s.fieldHandlers.Get))
		r.Patch("/fields/{id}", s.authRequired(s.fieldHandlers.Update))
		r.Delete("/fields/{id}", s.authRequired(s.fieldHandlers.Delete))
		r.Post("/fields/{tableId}/reorder", s.authRequired(s.fieldHandlers.Reorder))
		r.Get("/fields/{id}/options", s.authRequired(s.fieldHandlers.ListOptions))
		r.Post("/fields/{id}/options", s.authRequired(s.fieldHandlers.CreateOption))
		r.Patch("/fields/{id}/options/{optionId}", s.authRequired(s.fieldHandlers.UpdateOption))
		r.Delete("/fields/{id}/options/{optionId}", s.authRequired(s.fieldHandlers.DeleteOption))

		// Records
		r.Get("/records", s.authRequired(s.recordHandlers.List))
		r.Post("/records", s.authRequired(s.recordHandlers.Create))
		r.Get("/records/{id}", s.authRequired(s.recordHandlers.Get))
		r.Patch("/records/{id}", s.authRequired(s.recordHandlers.Update))
		r.Delete("/records/{id}", s.authRequired(s.recordHandlers.Delete))
		r.Post("/records/batch", s.authRequired(s.recordHandlers.BatchCreate))
		r.Patch("/records/batch", s.authRequired(s.recordHandlers.BatchUpdate))
		r.Delete("/records/batch", s.authRequired(s.recordHandlers.BatchDelete))

		// Views
		r.Post("/views", s.authRequired(s.viewHandlers.Create))
		r.Get("/views/{id}", s.authRequired(s.viewHandlers.Get))
		r.Patch("/views/{id}", s.authRequired(s.viewHandlers.Update))
		r.Delete("/views/{id}", s.authRequired(s.viewHandlers.Delete))
		r.Post("/views/{id}/duplicate", s.authRequired(s.viewHandlers.Duplicate))
		r.Post("/views/{tableId}/reorder", s.authRequired(s.viewHandlers.Reorder))
		r.Patch("/views/{id}/filters", s.authRequired(s.viewHandlers.SetFilters))
		r.Patch("/views/{id}/sorts", s.authRequired(s.viewHandlers.SetSorts))
		r.Patch("/views/{id}/groups", s.authRequired(s.viewHandlers.SetGroups))
		r.Patch("/views/{id}/field-config", s.authRequired(s.viewHandlers.SetFieldConfig))
		r.Patch("/views/{id}/config", s.authRequired(s.viewHandlers.SetConfig))

		// Comments
		r.Get("/comments/record/{recordId}", s.authRequired(s.commentHandlers.ListByRecord))
		r.Post("/comments", s.authRequired(s.commentHandlers.Create))
		r.Get("/comments/{id}", s.authRequired(s.commentHandlers.Get))
		r.Patch("/comments/{id}", s.authRequired(s.commentHandlers.Update))
		r.Delete("/comments/{id}", s.authRequired(s.commentHandlers.Delete))
		r.Post("/comments/{id}/resolve", s.authRequired(s.commentHandlers.Resolve))
		r.Post("/comments/{id}/unresolve", s.authRequired(s.commentHandlers.Unresolve))

		// Shares
		r.Get("/shares/base/{baseId}", s.authRequired(s.shareHandlers.ListByBase))
		r.Post("/shares", s.authRequired(s.shareHandlers.Create))
		r.Delete("/shares/{id}", s.authRequired(s.shareHandlers.Delete))
		r.Get("/shares/token/{token}", s.shareHandlers.GetByToken) // Public endpoint

		// Public forms (no auth required)
		r.Get("/public/forms/{viewId}", s.publicFormHandlers.GetForm)
		r.Post("/public/forms/{viewId}/submit", s.publicFormHandlers.SubmitForm)

		// Dashboard
		r.Get("/views/{id}/dashboard/data", s.authRequired(s.dashboardHandlers.GetData))
		r.Post("/views/{id}/dashboard/widgets", s.authRequired(s.dashboardHandlers.AddWidget))
		r.Patch("/views/{id}/dashboard/widgets/{widgetId}", s.authRequired(s.dashboardHandlers.UpdateWidget))
		r.Delete("/views/{id}/dashboard/widgets/{widgetId}", s.authRequired(s.dashboardHandlers.DeleteWidget))
	})

	// Serve static files
	s.serveStatic()
}

func (s *Server) serveStatic() {
	staticFS := assets.Static()

	// Serve static files from embedded filesystem
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))

	// Handle /static/* paths
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		// Cache static assets for 1 year (immutable since embedded)
		c.Writer().Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	// SPA fallback - serve index.html for non-API routes
	s.app.Get("/{path...}", func(c *mizu.Ctx) error {
		reqPath := c.Request().URL.Path

		// Don't handle API routes
		if strings.HasPrefix(reqPath, "/api/") {
			return c.JSON(404, map[string]string{"message": "not found"})
		}
		if strings.HasPrefix(reqPath, "/static/") {
			return c.JSON(404, map[string]string{"message": "not found"})
		}

		// Try to serve the file directly first
		if reqPath != "/" && reqPath != "" {
			cleanPath := strings.TrimPrefix(reqPath, "/")
			distPath := pathpkg.Join("dist", cleanPath)
			if info, err := fs.Stat(staticFS, distPath); err == nil && !info.IsDir() {
				ext := filepath.Ext(cleanPath)
				if contentType := mime.TypeByExtension(ext); contentType != "" {
					c.Writer().Header().Set("Content-Type", contentType)
				}
				http.ServeFileFS(c.Writer(), c.Request(), staticFS, distPath)
				return nil
			}
		}

		// Serve index.html for SPA routes
		indexContent, err := fs.ReadFile(staticFS, "dist/index.html")
		if err != nil {
			// If no index.html, show a placeholder
			c.Writer().Header().Set("Content-Type", "text/html")
			c.Writer().Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Table</title></head>
<body>
<h1>Table Blueprint</h1>
<p>Frontend not built. Run <code>make frontend-build</code> first.</p>
</body>
</html>`))
			return nil
		}

		c.Writer().Header().Set("Content-Type", "text/html")
		c.Writer().Header().Set("Cache-Control", "no-cache")
		c.Writer().Write(indexContent)
		return nil
	})
}
