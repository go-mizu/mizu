package web

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/spreadsheet/app/web/handler"
	"github.com/go-mizu/blueprints/spreadsheet/app/web/handler/api"
	"github.com/go-mizu/blueprints/spreadsheet/assets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/cells"
	"github.com/go-mizu/blueprints/spreadsheet/feature/sheets"
	"github.com/go-mizu/blueprints/spreadsheet/feature/users"
	"github.com/go-mizu/blueprints/spreadsheet/feature/workbooks"
	"github.com/go-mizu/blueprints/spreadsheet/store/duckdb"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app *mizu.App
	cfg Config
	db  *sql.DB

	// Services
	users     users.API
	workbooks workbooks.API
	sheets    sheets.API
	cells     cells.API

	// Handlers
	authHandlers     *api.Auth
	workbookHandlers *api.Workbook
	sheetHandlers    *api.Sheet
	cellHandlers     *api.Cell
	uiHandlers       *handler.UI
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "spreadsheet.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create core store for schema initialization
	coreStore, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Initialize schema
	if err := coreStore.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create feature stores
	usersStore := duckdb.NewUsersStore(db)
	workbooksStore := duckdb.NewWorkbooksStore(db)
	sheetsStore := duckdb.NewSheetsStore(db)
	cellsStore := duckdb.NewCellsStore(db)

	// Create services
	usersSvc := users.NewService(usersStore)
	workbooksSvc := workbooks.NewService(workbooksStore)
	sheetsSvc := sheets.NewService(sheetsStore)
	cellsSvc := cells.NewService(cellsStore, usersSvc.GetSecret())

	// Create dev user in dev mode
	if cfg.Dev {
		ctx := context.Background()
		// Check if dev user exists
		if _, err := usersStore.GetByID(ctx, devUserID); err != nil {
			// Create dev user
			now := time.Now()
			devUser := &users.User{
				ID:        devUserID,
				Email:     "dev@example.com",
				Name:      "Developer",
				Password:  "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "password"
				CreatedAt: now,
				UpdatedAt: now,
			}
			usersStore.Create(ctx, devUser)
			slog.Info("Created dev user", "id", devUserID, "email", "dev@example.com")
		}
	}

	s := &Server{
		app:       mizu.New(),
		cfg:       cfg,
		db:        db,
		users:     usersSvc,
		workbooks: workbooksSvc,
		sheets:    sheetsSvc,
		cells:     cellsSvc,
	}

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	// Create handlers
	s.authHandlers = api.NewAuth(usersSvc)
	s.workbookHandlers = api.NewWorkbook(workbooksSvc, sheetsSvc, s.getUserID)
	s.sheetHandlers = api.NewSheet(sheetsSvc, s.getUserID)
	s.cellHandlers = api.NewCell(cellsSvc, sheetsSvc, s.getUserID)
	s.uiHandlers = handler.NewUI(tmpl, usersSvc, workbooksSvc)

	s.setupRoutes()

	// Serve static files
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

	// Serve uploaded files
	uploadsDir := filepath.Join(cfg.DataDir, "uploads")
	os.MkdirAll(uploadsDir, 0755)
	uploadsHandler := http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadsDir)))
	s.app.Get("/uploads/{path...}", func(c *mizu.Ctx) error {
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		c.Writer().Header().Set("Cache-Control", "public, max-age=86400")
		uploadsHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	slog.Info("Starting Spreadsheet server", "addr", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Service accessors for CLI use
func (s *Server) UserService() users.API         { return s.users }
func (s *Server) WorkbookService() workbooks.API { return s.workbooks }
func (s *Server) SheetService() sheets.API       { return s.sheets }
func (s *Server) CellService() cells.API         { return s.cells }

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler { return s.app }

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// UI routes
	s.app.Get("/", func(c *mizu.Ctx) error {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	})
	s.app.Get("/login", s.uiHandlers.Login)
	s.app.Get("/register", s.uiHandlers.Register)
	s.app.Get("/app", s.uiHandlers.AppRedirect)
	s.app.Get("/s/{workbookID}", s.authRequired(s.uiHandlers.Spreadsheet))
	s.app.Get("/s/{workbookID}/{sheetID}", s.authRequired(s.uiHandlers.Spreadsheet))

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Workbooks
		api.Get("/workbooks", s.authRequired(s.workbookHandlers.List))
		api.Post("/workbooks", s.authRequired(s.workbookHandlers.Create))
		api.Get("/workbooks/{id}", s.authRequired(s.workbookHandlers.Get))
		api.Patch("/workbooks/{id}", s.authRequired(s.workbookHandlers.Update))
		api.Delete("/workbooks/{id}", s.authRequired(s.workbookHandlers.Delete))
		api.Get("/workbooks/{id}/sheets", s.authRequired(s.workbookHandlers.ListSheets))

		// Sheets
		api.Post("/sheets", s.authRequired(s.sheetHandlers.Create))
		api.Get("/sheets/{id}", s.authRequired(s.sheetHandlers.Get))
		api.Patch("/sheets/{id}", s.authRequired(s.sheetHandlers.Update))
		api.Delete("/sheets/{id}", s.authRequired(s.sheetHandlers.Delete))

		// Cells
		api.Get("/sheets/{sheetID}/cells", s.authRequired(s.cellHandlers.GetRange))
		api.Put("/sheets/{sheetID}/cells", s.authRequired(s.cellHandlers.BatchUpdate))
		api.Get("/sheets/{sheetID}/cells/{row}/{col}", s.authRequired(s.cellHandlers.Get))
		api.Put("/sheets/{sheetID}/cells/{row}/{col}", s.authRequired(s.cellHandlers.Set))
		api.Delete("/sheets/{sheetID}/cells/{row}/{col}", s.authRequired(s.cellHandlers.Delete))

		// Cell operations
		api.Post("/sheets/{sheetID}/rows/insert", s.authRequired(s.cellHandlers.InsertRows))
		api.Post("/sheets/{sheetID}/rows/delete", s.authRequired(s.cellHandlers.DeleteRows))
		api.Post("/sheets/{sheetID}/cols/insert", s.authRequired(s.cellHandlers.InsertCols))
		api.Post("/sheets/{sheetID}/cols/delete", s.authRequired(s.cellHandlers.DeleteCols))

		// Merged regions
		api.Get("/sheets/{sheetID}/merges", s.authRequired(s.cellHandlers.GetMerges))
		api.Post("/sheets/{sheetID}/merge", s.authRequired(s.cellHandlers.Merge))
		api.Post("/sheets/{sheetID}/unmerge", s.authRequired(s.cellHandlers.Unmerge))

		// Formula evaluation
		api.Post("/formula/evaluate", s.authRequired(s.cellHandlers.Evaluate))
	})
}
