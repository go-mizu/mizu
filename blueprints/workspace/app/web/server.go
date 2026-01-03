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

	"github.com/go-mizu/blueprints/workspace/app/web/handler"
	"github.com/go-mizu/blueprints/workspace/app/web/handler/api"
	"github.com/go-mizu/blueprints/workspace/assets"
	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/comments"
	"github.com/go-mizu/blueprints/workspace/feature/databases"
	"github.com/go-mizu/blueprints/workspace/feature/favorites"
	"github.com/go-mizu/blueprints/workspace/feature/history"
	"github.com/go-mizu/blueprints/workspace/feature/members"
	"github.com/go-mizu/blueprints/workspace/feature/notifications"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/search"
	"github.com/go-mizu/blueprints/workspace/feature/sharing"
	"github.com/go-mizu/blueprints/workspace/feature/templates"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/feature/views"
	"github.com/go-mizu/blueprints/workspace/feature/workspaces"
	"github.com/go-mizu/blueprints/workspace/store/duckdb"
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
	users         users.API
	workspaces    workspaces.API
	members       members.API
	pages         pages.API
	blocks        blocks.API
	databases     databases.API
	views         views.API
	comments      comments.API
	sharing       sharing.API
	history       history.API
	notifications notifications.API
	favorites     favorites.API
	search        search.API
	templates     templates.API

	// Handlers
	authHandlers       *api.Auth
	workspaceHandlers  *api.Workspace
	pageHandlers       *api.Page
	blockHandlers      *api.Block
	databaseHandlers   *api.Database
	viewHandlers       *api.View
	commentHandlers    *api.Comment
	shareHandlers      *api.Share
	favoriteHandlers   *api.Favorite
	searchHandlers     *api.Search
	mediaHandlers      *api.Media
	uiHandlers         *handler.UI
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "workspace.duckdb")
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
	workspacesStore := duckdb.NewWorkspacesStore(db)
	membersStore := duckdb.NewMembersStore(db)
	pagesStore := duckdb.NewPagesStore(db)
	blocksStore := duckdb.NewBlocksStore(db)
	databasesStore := duckdb.NewDatabasesStore(db)
	viewsStore := duckdb.NewViewsStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	sharesStore := duckdb.NewSharesStore(db)
	historyStore := duckdb.NewHistoryStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)
	favoritesStore := duckdb.NewFavoritesStore(db)
	templatesStore := duckdb.NewTemplatesStore(db)

	// Create services
	usersSvc := users.NewService(usersStore)
	workspacesSvc := workspaces.NewService(workspacesStore)
	membersSvc := members.NewService(membersStore, usersSvc)
	pagesSvc := pages.NewService(pagesStore, usersSvc)
	blocksSvc := blocks.NewService(blocksStore)
	databasesSvc := databases.NewService(databasesStore)
	viewsSvc := views.NewService(viewsStore, pagesSvc)
	commentsSvc := comments.NewService(commentsStore, usersSvc)
	sharingSvc := sharing.NewService(sharesStore, usersSvc)
	historySvc := history.NewService(historyStore, usersSvc, pagesSvc, blocksSvc)
	notificationsSvc := notifications.NewService(notificationsStore, usersSvc, pagesSvc)
	favoritesSvc := favorites.NewService(favoritesStore, pagesSvc)
	templatesSvc := templates.NewService(templatesStore, pagesSvc)
	searchSvc := search.NewService(duckdb.NewSearchStore(db), pagesSvc, databasesSvc)

	s := &Server{
		app:           mizu.New(),
		cfg:           cfg,
		db:            db,
		users:         usersSvc,
		workspaces:    workspacesSvc,
		members:       membersSvc,
		pages:         pagesSvc,
		blocks:        blocksSvc,
		databases:     databasesSvc,
		views:         viewsSvc,
		comments:      commentsSvc,
		sharing:       sharingSvc,
		history:       historySvc,
		notifications: notificationsSvc,
		favorites:     favoritesSvc,
		search:        searchSvc,
		templates:     templatesSvc,
	}

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	// Create handlers
	s.authHandlers = api.NewAuth(usersSvc)
	s.workspaceHandlers = api.NewWorkspace(workspacesSvc, membersSvc, s.getUserID)
	s.pageHandlers = api.NewPage(pagesSvc, blocksSvc, s.getUserID)
	s.blockHandlers = api.NewBlock(blocksSvc, s.getUserID)
	s.databaseHandlers = api.NewDatabase(databasesSvc, s.getUserID)
	s.viewHandlers = api.NewView(viewsSvc, s.getUserID)
	s.commentHandlers = api.NewComment(commentsSvc, s.getUserID)
	s.shareHandlers = api.NewShare(sharingSvc, s.getUserID)
	s.favoriteHandlers = api.NewFavorite(favoritesSvc, s.getUserID)
	s.searchHandlers = api.NewSearch(searchSvc, s.getUserID)
	s.mediaHandlers = api.NewMedia(filepath.Join(cfg.DataDir, "uploads"), s.getUserID)
	s.uiHandlers = handler.NewUI(tmpl, usersSvc, workspacesSvc, membersSvc, pagesSvc, blocksSvc, databasesSvc, viewsSvc, favoritesSvc, s.getUserID)

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
	log.Printf("Starting Workspace server on %s", s.cfg.Addr)
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
func (s *Server) UserService() users.API           { return s.users }
func (s *Server) WorkspaceService() workspaces.API { return s.workspaces }
func (s *Server) MemberService() members.API       { return s.members }
func (s *Server) PageService() pages.API           { return s.pages }
func (s *Server) BlockService() blocks.API         { return s.blocks }
func (s *Server) DatabaseService() databases.API   { return s.databases }
func (s *Server) ViewService() views.API           { return s.views }

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
	s.app.Get("/w/{workspace}", s.authRequired(s.uiHandlers.Workspace))
	s.app.Get("/w/{workspace}/p/{pageID}", s.authRequired(s.uiHandlers.Page))
	s.app.Get("/w/{workspace}/d/{databaseID}", s.authRequired(s.uiHandlers.Database))
	s.app.Get("/w/{workspace}/search", s.authRequired(s.uiHandlers.Search))
	s.app.Get("/w/{workspace}/settings", s.authRequired(s.uiHandlers.Settings))

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Workspaces
		api.Get("/workspaces", s.authRequired(s.workspaceHandlers.List))
		api.Post("/workspaces", s.authRequired(s.workspaceHandlers.Create))
		api.Get("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Get))
		api.Patch("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Update))
		api.Delete("/workspaces/{id}", s.authRequired(s.workspaceHandlers.Delete))
		api.Get("/workspaces/{id}/members", s.authRequired(s.workspaceHandlers.ListMembers))
		api.Post("/workspaces/{id}/members", s.authRequired(s.workspaceHandlers.AddMember))

		// Pages
		api.Post("/pages", s.authRequired(s.pageHandlers.Create))
		api.Get("/pages/{id}", s.authRequired(s.pageHandlers.Get))
		api.Patch("/pages/{id}", s.authRequired(s.pageHandlers.Update))
		api.Delete("/pages/{id}", s.authRequired(s.pageHandlers.Delete))
		api.Get("/workspaces/{workspaceID}/pages", s.authRequired(s.pageHandlers.List))
		api.Get("/pages/{id}/blocks", s.authRequired(s.pageHandlers.GetBlocks))
		api.Put("/pages/{id}/blocks", s.authRequired(s.pageHandlers.UpdateBlocks))
		api.Post("/pages/{id}/archive", s.authRequired(s.pageHandlers.Archive))
		api.Post("/pages/{id}/restore", s.authRequired(s.pageHandlers.Restore))
		api.Post("/pages/{id}/duplicate", s.authRequired(s.pageHandlers.Duplicate))

		// Blocks
		api.Post("/blocks", s.authRequired(s.blockHandlers.Create))
		api.Patch("/blocks/{id}", s.authRequired(s.blockHandlers.Update))
		api.Delete("/blocks/{id}", s.authRequired(s.blockHandlers.Delete))
		api.Post("/blocks/{id}/move", s.authRequired(s.blockHandlers.Move))

		// Databases
		api.Post("/databases", s.authRequired(s.databaseHandlers.Create))
		api.Get("/databases/{id}", s.authRequired(s.databaseHandlers.Get))
		api.Patch("/databases/{id}", s.authRequired(s.databaseHandlers.Update))
		api.Delete("/databases/{id}", s.authRequired(s.databaseHandlers.Delete))
		api.Post("/databases/{id}/properties", s.authRequired(s.databaseHandlers.AddProperty))
		api.Patch("/databases/{id}/properties/{propID}", s.authRequired(s.databaseHandlers.UpdateProperty))
		api.Delete("/databases/{id}/properties/{propID}", s.authRequired(s.databaseHandlers.DeleteProperty))

		// Views
		api.Post("/views", s.authRequired(s.viewHandlers.Create))
		api.Get("/views/{id}", s.authRequired(s.viewHandlers.Get))
		api.Patch("/views/{id}", s.authRequired(s.viewHandlers.Update))
		api.Delete("/views/{id}", s.authRequired(s.viewHandlers.Delete))
		api.Get("/databases/{id}/views", s.authRequired(s.viewHandlers.List))
		api.Post("/views/{id}/query", s.authRequired(s.viewHandlers.Query))

		// Comments
		api.Post("/comments", s.authRequired(s.commentHandlers.Create))
		api.Patch("/comments/{id}", s.authRequired(s.commentHandlers.Update))
		api.Delete("/comments/{id}", s.authRequired(s.commentHandlers.Delete))
		api.Get("/pages/{id}/comments", s.authRequired(s.commentHandlers.List))
		api.Post("/comments/{id}/resolve", s.authRequired(s.commentHandlers.Resolve))

		// Sharing
		api.Post("/pages/{id}/shares", s.authRequired(s.shareHandlers.Create))
		api.Get("/pages/{id}/shares", s.authRequired(s.shareHandlers.List))
		api.Delete("/shares/{id}", s.authRequired(s.shareHandlers.Delete))

		// Favorites
		api.Post("/favorites", s.authRequired(s.favoriteHandlers.Add))
		api.Delete("/favorites/{pageID}", s.authRequired(s.favoriteHandlers.Remove))
		api.Get("/workspaces/{id}/favorites", s.authRequired(s.favoriteHandlers.List))

		// Search
		api.Get("/workspaces/{id}/search", s.authRequired(s.searchHandlers.Search))
		api.Get("/workspaces/{id}/quick-search", s.authRequired(s.searchHandlers.QuickSearch))
		api.Get("/workspaces/{id}/recent", s.authRequired(s.searchHandlers.Recent))

		// Media
		api.Post("/media/upload", s.authRequired(s.mediaHandlers.Upload))
	})
}
