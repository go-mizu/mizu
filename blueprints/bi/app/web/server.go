package web

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/bi/app/web/handler/api"
	"github.com/go-mizu/blueprints/bi/assets"
	"github.com/go-mizu/blueprints/bi/store"
	"github.com/go-mizu/blueprints/bi/store/sqlite"
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

	// Handlers
	datasourcesHandler   *api.DataSources
	questionsHandler     *api.Questions
	queryHandler         *api.Query
	dashboardsHandler    *api.Dashboards
	collectionsHandler   *api.Collections
	modelsHandler        *api.Models
	metricsHandler       *api.Metrics
	alertsHandler        *api.Alerts
	subscriptionsHandler *api.Subscriptions
	usersHandler         *api.Users
	settingsHandler      *api.Settings
	xrayHandler          *api.XRay
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Create store
	st, err := sqlite.New(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Initialize schema
	if err := st.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	s := &Server{
		app:   mizu.New(),
		cfg:   cfg,
		store: st,
	}

	// Create handlers
	s.datasourcesHandler = api.NewDataSources(st)
	s.questionsHandler = api.NewQuestions(st)
	s.queryHandler = api.NewQuery(st)
	s.dashboardsHandler = api.NewDashboards(st)
	s.collectionsHandler = api.NewCollections(st)
	s.modelsHandler = api.NewModels(st)
	s.metricsHandler = api.NewMetrics(st)
	s.alertsHandler = api.NewAlerts(st)
	s.subscriptionsHandler = api.NewSubscriptions(st)
	s.usersHandler = api.NewUsers(st)
	s.settingsHandler = api.NewSettings(st)
	s.xrayHandler = api.NewXRay(st)

	s.setupRoutes()

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	slog.Info("Starting BI server", "addr", s.cfg.Addr)
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
func (s *Server) Handler() http.Handler { return s.app }

// Store returns the underlying store.
func (s *Server) Store() store.Store { return s.store }

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// API routes
	s.app.Group("/api", func(apiGroup *mizu.Router) {
		// Auth
		apiGroup.Post("/auth/login", s.usersHandler.Login)
		apiGroup.Post("/auth/logout", s.usersHandler.Logout)
		apiGroup.Get("/auth/me", s.usersHandler.Me)
		apiGroup.Put("/auth/me", s.usersHandler.UpdateProfile)
		apiGroup.Post("/auth/me/password", s.usersHandler.ChangePassword)

		// Data Sources
		apiGroup.Get("/datasources", s.datasourcesHandler.List)
		apiGroup.Post("/datasources", s.datasourcesHandler.Create)
		apiGroup.Post("/datasources/test-connection", s.datasourcesHandler.TestConnection)
		apiGroup.Get("/datasources/{id}", s.datasourcesHandler.Get)
		apiGroup.Put("/datasources/{id}", s.datasourcesHandler.Update)
		apiGroup.Delete("/datasources/{id}", s.datasourcesHandler.Delete)
		apiGroup.Post("/datasources/{id}/test", s.datasourcesHandler.Test)
		apiGroup.Get("/datasources/{id}/status", s.datasourcesHandler.GetStatus)
		apiGroup.Post("/datasources/{id}/sync", s.datasourcesHandler.Sync)
		apiGroup.Post("/datasources/{id}/scan", s.datasourcesHandler.Scan)
		apiGroup.Post("/datasources/{id}/fingerprint", s.datasourcesHandler.Fingerprint)
		apiGroup.Get("/datasources/{id}/sync-log", s.datasourcesHandler.GetSyncLog)
		apiGroup.Get("/datasources/{id}/schemas", s.datasourcesHandler.ListSchemas)
		apiGroup.Get("/datasources/{id}/cache/stats", s.datasourcesHandler.GetCacheStats)
		apiGroup.Post("/datasources/{id}/cache/clear", s.datasourcesHandler.ClearCache)
		apiGroup.Get("/datasources/{id}/tables", s.datasourcesHandler.ListTables)
		apiGroup.Get("/datasources/{id}/tables/{table}", s.datasourcesHandler.GetTable)
		apiGroup.Put("/datasources/{id}/tables/{table}", s.datasourcesHandler.UpdateTable)
		apiGroup.Post("/datasources/{id}/tables/{table}/sync", s.datasourcesHandler.SyncTable)
		apiGroup.Post("/datasources/{id}/tables/{table}/scan", s.datasourcesHandler.ScanTable)
		apiGroup.Post("/datasources/{id}/tables/{table}/discard-values", s.datasourcesHandler.DiscardCachedValues)
		apiGroup.Get("/datasources/{id}/tables/{table}/columns", s.datasourcesHandler.ListColumns)
		apiGroup.Put("/datasources/tables/{tableId}/columns/{columnId}", s.datasourcesHandler.UpdateColumn)
		apiGroup.Post("/datasources/{id}/tables/{table}/columns/{column}/scan", s.datasourcesHandler.ScanColumn)
		apiGroup.Post("/datasources/{id}/tables/{table}/preview", s.datasourcesHandler.TablePreview)
		apiGroup.Get("/datasources/{id}/tables/{table}/preview", s.datasourcesHandler.TablePreview)
		apiGroup.Get("/datasources/{id}/search-tables", s.datasourcesHandler.SearchTables)

		// Questions
		apiGroup.Get("/questions", s.questionsHandler.List)
		apiGroup.Post("/questions", s.questionsHandler.Create)
		apiGroup.Get("/questions/{id}", s.questionsHandler.Get)
		apiGroup.Put("/questions/{id}", s.questionsHandler.Update)
		apiGroup.Delete("/questions/{id}", s.questionsHandler.Delete)
		apiGroup.Post("/questions/{id}/query", s.questionsHandler.Execute)

		// Query
		apiGroup.Post("/query", s.queryHandler.Execute)
		apiGroup.Post("/query/native", s.queryHandler.ExecuteNative)
		apiGroup.Get("/query/history", s.queryHandler.History)

		// Dashboards
		apiGroup.Get("/dashboards", s.dashboardsHandler.List)
		apiGroup.Post("/dashboards", s.dashboardsHandler.Create)
		apiGroup.Get("/dashboards/{id}", s.dashboardsHandler.Get)
		apiGroup.Put("/dashboards/{id}", s.dashboardsHandler.Update)
		apiGroup.Delete("/dashboards/{id}", s.dashboardsHandler.Delete)
		apiGroup.Get("/dashboards/{id}/cards", s.dashboardsHandler.ListCards)
		apiGroup.Post("/dashboards/{id}/cards", s.dashboardsHandler.AddCard)
		apiGroup.Put("/dashboards/{id}/cards/{card}", s.dashboardsHandler.UpdateCard)
		apiGroup.Delete("/dashboards/{id}/cards/{card}", s.dashboardsHandler.RemoveCard)

		// Collections
		apiGroup.Get("/collections", s.collectionsHandler.List)
		apiGroup.Post("/collections", s.collectionsHandler.Create)
		// Special collections (must be before {id} routes)
		apiGroup.Get("/collections/root", s.collectionsHandler.GetRoot)
		apiGroup.Get("/collections/root/items", s.collectionsHandler.ListItems)
		apiGroup.Get("/collections/personal", s.collectionsHandler.GetPersonal)
		apiGroup.Get("/collections/personal/items", s.collectionsHandler.GetPersonalItems)
		// Standard collection routes
		apiGroup.Get("/collections/{id}", s.collectionsHandler.Get)
		apiGroup.Put("/collections/{id}", s.collectionsHandler.Update)
		apiGroup.Delete("/collections/{id}", s.collectionsHandler.Delete)
		apiGroup.Get("/collections/{id}/items", s.collectionsHandler.ListItems)

		// Models
		apiGroup.Get("/models", s.modelsHandler.List)
		apiGroup.Post("/models", s.modelsHandler.Create)
		apiGroup.Get("/models/{id}", s.modelsHandler.Get)
		apiGroup.Put("/models/{id}", s.modelsHandler.Update)
		apiGroup.Delete("/models/{id}", s.modelsHandler.Delete)

		// Metrics
		apiGroup.Get("/metrics", s.metricsHandler.List)
		apiGroup.Post("/metrics", s.metricsHandler.Create)
		apiGroup.Get("/metrics/{id}", s.metricsHandler.Get)
		apiGroup.Put("/metrics/{id}", s.metricsHandler.Update)
		apiGroup.Delete("/metrics/{id}", s.metricsHandler.Delete)

		// Alerts
		apiGroup.Get("/alerts", s.alertsHandler.List)
		apiGroup.Post("/alerts", s.alertsHandler.Create)
		apiGroup.Get("/alerts/{id}", s.alertsHandler.Get)
		apiGroup.Put("/alerts/{id}", s.alertsHandler.Update)
		apiGroup.Delete("/alerts/{id}", s.alertsHandler.Delete)

		// Subscriptions
		apiGroup.Get("/subscriptions", s.subscriptionsHandler.List)
		apiGroup.Post("/subscriptions", s.subscriptionsHandler.Create)
		apiGroup.Get("/subscriptions/{id}", s.subscriptionsHandler.Get)
		apiGroup.Put("/subscriptions/{id}", s.subscriptionsHandler.Update)
		apiGroup.Delete("/subscriptions/{id}", s.subscriptionsHandler.Delete)

		// Users (admin)
		apiGroup.Get("/users", s.usersHandler.List)
		apiGroup.Post("/users", s.usersHandler.Create)
		apiGroup.Get("/users/{id}", s.usersHandler.Get)
		apiGroup.Put("/users/{id}", s.usersHandler.Update)
		apiGroup.Delete("/users/{id}", s.usersHandler.Delete)
		apiGroup.Post("/users/{id}/deactivate", s.usersHandler.Deactivate)
		apiGroup.Post("/users/{id}/reset-password", s.usersHandler.ResetPassword)

		// Admin
		apiGroup.Get("/admin/activity", s.settingsHandler.ActivityLog)

		// Settings
		apiGroup.Get("/settings", s.settingsHandler.List)
		apiGroup.Put("/settings", s.settingsHandler.Update)
		apiGroup.Get("/settings/audit", s.settingsHandler.AuditLogs)

		// X-Ray (automatic insights)
		apiGroup.Post("/xray/{datasourceId}/table/{tableId}", s.xrayHandler.XRayTable)
		apiGroup.Post("/xray/{datasourceId}/field/{columnId}", s.xrayHandler.XRayField)
		apiGroup.Post("/xray/{datasourceId}/table/{tableId}/compare", s.xrayHandler.XRayCompare)
		apiGroup.Post("/xray/save", s.xrayHandler.SaveXRay)
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
			distPath := "dist/" + cleanPath
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
			c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
			c.Writer().Write([]byte(defaultIndexHTML))
			return nil
		}

		c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
		c.Writer().Write(indexContent)
		return nil
	})
}

const defaultIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>BI - Business Intelligence</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
      background: #f9fbfc;
      color: #2e353b;
      min-height: 100vh;
      display: flex;
    }
    .sidebar {
      width: 260px;
      background: #2e353b;
      color: white;
      padding: 20px 0;
      display: flex;
      flex-direction: column;
    }
    .logo {
      padding: 0 20px 20px;
      font-size: 24px;
      font-weight: bold;
      color: #509ee3;
      border-bottom: 1px solid rgba(255,255,255,0.1);
      margin-bottom: 20px;
    }
    .nav-item {
      padding: 12px 20px;
      display: flex;
      align-items: center;
      gap: 12px;
      color: rgba(255,255,255,0.7);
      text-decoration: none;
      transition: all 0.2s;
    }
    .nav-item:hover { background: rgba(255,255,255,0.1); color: white; }
    .nav-item.active { background: #509ee3; color: white; }
    .nav-icon { width: 20px; text-align: center; }
    .main {
      flex: 1;
      padding: 24px;
      overflow: auto;
    }
    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
    }
    .search {
      display: flex;
      align-items: center;
      background: white;
      border: 1px solid #e0e4e8;
      border-radius: 8px;
      padding: 8px 16px;
      width: 400px;
    }
    .search input {
      border: none;
      outline: none;
      flex: 1;
      font-size: 14px;
    }
    .btn {
      background: #509ee3;
      color: white;
      border: none;
      padding: 10px 20px;
      border-radius: 8px;
      font-size: 14px;
      font-weight: 500;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .btn:hover { background: #4285c9; }
    .cards {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
      gap: 20px;
    }
    .card {
      background: white;
      border-radius: 8px;
      padding: 20px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    }
    .card-title {
      font-size: 16px;
      font-weight: 600;
      margin-bottom: 8px;
    }
    .card-desc {
      color: #949aab;
      font-size: 14px;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 20px;
      margin-bottom: 24px;
    }
    .stat {
      background: white;
      border-radius: 8px;
      padding: 20px;
      box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    }
    .stat-value {
      font-size: 32px;
      font-weight: 700;
      color: #509ee3;
    }
    .stat-label {
      color: #949aab;
      font-size: 14px;
      margin-top: 4px;
    }
    .empty {
      text-align: center;
      padding: 60px 20px;
      color: #949aab;
    }
    .empty h2 { margin-bottom: 12px; color: #2e353b; }
  </style>
</head>
<body>
  <aside class="sidebar">
    <div class="logo">BI</div>
    <a href="/" class="nav-item active">
      <span class="nav-icon">&#127968;</span>
      <span>Home</span>
    </a>
    <a href="/browse" class="nav-item">
      <span class="nav-icon">&#128193;</span>
      <span>Browse</span>
    </a>
    <a href="/question/new" class="nav-item">
      <span class="nav-icon">&#10010;</span>
      <span>New Question</span>
    </a>
    <a href="/dashboard/new" class="nav-item">
      <span class="nav-icon">&#128202;</span>
      <span>New Dashboard</span>
    </a>
  </aside>
  <main class="main">
    <div class="header">
      <div class="search">
        <span style="margin-right: 8px;">&#128269;</span>
        <input type="text" placeholder="Search for anything...">
      </div>
      <button class="btn">
        <span>+</span>
        <span>New</span>
      </button>
    </div>
    <div class="stats">
      <div class="stat">
        <div class="stat-value">0</div>
        <div class="stat-label">Questions</div>
      </div>
      <div class="stat">
        <div class="stat-value">0</div>
        <div class="stat-label">Dashboards</div>
      </div>
      <div class="stat">
        <div class="stat-value">0</div>
        <div class="stat-label">Collections</div>
      </div>
      <div class="stat">
        <div class="stat-value">0</div>
        <div class="stat-label">Data Sources</div>
      </div>
    </div>
    <div class="empty">
      <h2>Welcome to BI</h2>
      <p>Run 'bi seed' to add sample data, then refresh this page.</p>
      <p style="margin-top: 20px; font-size: 12px;">
        Or build the frontend with 'make frontend-build' for the full experience.
      </p>
    </div>
  </main>
</body>
</html>`
