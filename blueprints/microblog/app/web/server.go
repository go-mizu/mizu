package web

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/assets"
	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/interactions"
	"github.com/go-mizu/blueprints/microblog/feature/notifications"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/search"
	"github.com/go-mizu/blueprints/microblog/feature/timelines"
	"github.com/go-mizu/blueprints/microblog/feature/trending"
	"github.com/go-mizu/blueprints/microblog/store/duckdb"
)

// Server is the HTTP server.
type Server struct {
	app       *mizu.App
	cfg       Config
	db        *sql.DB
	templates *template.Template

	// Services (as interfaces)
	accounts      accounts.API
	posts         posts.API
	timelines     timelines.API
	interactions  interactions.API
	relationships relationships.API
	notifications notifications.API
	search        search.API
	trending      trending.API
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "microblog.duckdb")
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
	accountsStore := duckdb.NewAccountsStore(db)
	postsStore := duckdb.NewPostsStore(db)
	interactionsStore := duckdb.NewInteractionsStore(db)
	relationshipsStore := duckdb.NewRelationshipsStore(db)
	timelinesStore := duckdb.NewTimelinesStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)
	searchStore := duckdb.NewSearchStore(db)
	trendingStore := duckdb.NewTrendingStore(db)

	// Create services with stores
	accountsSvc := accounts.NewService(accountsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc)

	// Parse templates
	tmpl, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	s := &Server{
		app:           mizu.New(),
		cfg:           cfg,
		db:            db,
		templates:     tmpl,
		accounts:      accountsSvc,
		posts:         postsSvc,
		timelines:     timelines.NewService(timelinesStore, accountsSvc),
		interactions:  interactions.NewService(interactionsStore),
		relationships: relationships.NewService(relationshipsStore),
		notifications: notifications.NewService(notificationsStore, accountsSvc),
		search:        search.NewService(searchStore),
		trending:      trending.NewService(trendingStore),
	}

	// Serve static files from embedded assets
	s.app.Get("/static/*", func(c *mizu.Ctx) error {
		http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static()))).ServeHTTP(c.Writer(), c.Request())
		return nil
	})

	s.setupRoutes()
	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	log.Printf("Starting server on %s", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Server) setupRoutes() {
	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.handleRegister)
		api.Post("/auth/login", s.handleLogin)
		api.Post("/auth/logout", s.authRequired(s.handleLogout))

		// Accounts
		api.Get("/accounts/verify_credentials", s.authRequired(s.handleVerifyCredentials))
		api.Patch("/accounts/update_credentials", s.authRequired(s.handleUpdateCredentials))
		api.Get("/accounts/{id}", s.handleGetAccount)
		api.Get("/accounts/{id}/posts", s.handleAccountPosts)
		api.Get("/accounts/{id}/followers", s.handleAccountFollowers)
		api.Get("/accounts/{id}/following", s.handleAccountFollowing)
		api.Post("/accounts/{id}/follow", s.authRequired(s.handleFollow))
		api.Post("/accounts/{id}/unfollow", s.authRequired(s.handleUnfollow))
		api.Post("/accounts/{id}/block", s.authRequired(s.handleBlock))
		api.Post("/accounts/{id}/unblock", s.authRequired(s.handleUnblock))
		api.Post("/accounts/{id}/mute", s.authRequired(s.handleMute))
		api.Post("/accounts/{id}/unmute", s.authRequired(s.handleUnmute))
		api.Get("/accounts/relationships", s.authRequired(s.handleRelationships))

		// Posts
		api.Post("/posts", s.authRequired(s.handleCreatePost))
		api.Get("/posts/{id}", s.handleGetPost)
		api.Put("/posts/{id}", s.authRequired(s.handleUpdatePost))
		api.Delete("/posts/{id}", s.authRequired(s.handleDeletePost))
		api.Get("/posts/{id}/context", s.handlePostContext)
		api.Post("/posts/{id}/like", s.authRequired(s.handleLike))
		api.Delete("/posts/{id}/like", s.authRequired(s.handleUnlike))
		api.Post("/posts/{id}/repost", s.authRequired(s.handleRepost))
		api.Delete("/posts/{id}/repost", s.authRequired(s.handleUnrepost))
		api.Post("/posts/{id}/bookmark", s.authRequired(s.handleBookmark))
		api.Delete("/posts/{id}/bookmark", s.authRequired(s.handleUnbookmark))
		api.Get("/posts/{id}/liked_by", s.handleLikedBy)
		api.Get("/posts/{id}/reposted_by", s.handleRepostedBy)

		// Timelines
		api.Get("/timelines/home", s.authRequired(s.handleHomeTimeline))
		api.Get("/timelines/local", s.handleLocalTimeline)
		api.Get("/timelines/tag/{tag}", s.handleHashtagTimeline)

		// Notifications
		api.Get("/notifications", s.authRequired(s.handleNotifications))
		api.Post("/notifications/clear", s.authRequired(s.handleClearNotifications))
		api.Post("/notifications/{id}/dismiss", s.authRequired(s.handleDismissNotification))

		// Search
		api.Get("/search", s.handleSearch)

		// Trends
		api.Get("/trends/tags", s.handleTrendingTags)
		api.Get("/trends/posts", s.handleTrendingPosts)

		// Bookmarks
		api.Get("/bookmarks", s.authRequired(s.handleBookmarks))
	})

	// Web routes (pages)
	s.app.Get("/", s.handleHomePage)
	s.app.Get("/login", s.handleLoginPage)
	s.app.Get("/register", s.handleRegisterPage)
	s.app.Get("/@{username}", s.handleProfilePage)
	s.app.Get("/@{username}/{id}", s.handlePostPage)
	s.app.Get("/tags/{tag}", s.handleTagPage)
	s.app.Get("/explore", s.handleExplorePage)
	s.app.Get("/notifications", s.handleNotificationsPage)
	s.app.Get("/settings", s.handleSettingsPage)
}

// render renders a template with the given data.
func (s *Server) render(c *mizu.Ctx, name string, data map[string]any) error {
	if data == nil {
		data = make(map[string]any)
	}
	data["Dev"] = s.cfg.Dev

	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	return s.templates.ExecuteTemplate(c.Writer(), name, data)
}
