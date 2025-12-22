package web

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/go-mizu/blueprints/forum/app/web/handler"
	"github.com/go-mizu/blueprints/forum/assets"
	"github.com/go-mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/blueprints/forum/feature/forums"
	"github.com/go-mizu/blueprints/forum/feature/posts"
	"github.com/go-mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/blueprints/forum/feature/votes"
	"github.com/go-mizu/blueprints/forum/store/duckdb"
	"github.com/go-mizu/mizu"
)

// Server is the web server.
type Server struct {
	app       *mizu.App
	cfg       Config
	db        *sql.DB
	store     *duckdb.Store
	templates *template.Template

	// Services
	accounts accounts.API
	forums   forums.API
	threads  threads.API
	posts    posts.API
	votes    votes.API
}

// New creates a new web server.
func New(cfg Config) (*Server, error) {
	// Open database
	db, err := sql.Open("duckdb", cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create store
	store, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Ensure schema
	if err := store.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create feature stores
	accountsStore := duckdb.NewAccountsStore(db)

	// Create services
	accountsSvc := accounts.NewService(accountsStore)

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	// Create app
	app := mizu.New()

	s := &Server{
		app:       app,
		cfg:       cfg,
		db:        db,
		store:     store,
		templates: tmpl,
		accounts:  accountsSvc,
	}

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	// Serve static files from embedded assets
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	// Create handler
	h := handler.New(s.templates, s.accounts, s.forums, s.threads, s.posts, s.votes)

	// Web pages
	s.app.Get("/", h.Home)
	s.app.Get("/login", h.LoginPage)
	s.app.Post("/login", h.Login)
	s.app.Get("/register", h.RegisterPage)
	s.app.Post("/register", h.Register)
	s.app.Post("/logout", h.Logout)

	// Forums
	s.app.Get("/f/{slug}", h.ForumPage)

	// Threads
	s.app.Get("/f/{slug}/t/{id}", h.ThreadPage)

	// User profiles
	s.app.Get("/u/{username}", h.ProfilePage)

	// API endpoints
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Authentication
		api.Post("/auth/register", h.APIRegister)
		api.Post("/auth/login", h.APILogin)
		api.Post("/auth/logout", h.APILogout)

		// Forums
		api.Get("/forums", h.APIListForums)
		api.Post("/forums", h.APICreateForum)
		api.Get("/forums/{id}", h.APIGetForum)

		// Threads
		api.Get("/forums/{id}/threads", h.APIListThreads)
		api.Post("/forums/{id}/threads", h.APICreateThread)
		api.Get("/threads/{id}", h.APIGetThread)

		// Posts
		api.Get("/threads/{id}/posts", h.APIListPosts)
		api.Post("/threads/{id}/posts", h.APICreatePost)

		// Voting
		api.Post("/threads/{id}/vote", h.APIVoteThread)
		api.Post("/posts/{id}/vote", h.APIVotePost)
	})
}

// Start starts the web server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	fmt.Printf("Starting server on http://%s\n", addr)
	return s.app.Listen(addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.app
}
