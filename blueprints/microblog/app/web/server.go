package web

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

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
	app    *mizu.App
	cfg    Config
	store  *duckdb.Store

	// Services
	accounts      *accounts.Service
	posts         *posts.Service
	timelines     *timelines.Service
	interactions  *interactions.Service
	relationships *relationships.Service
	notifications *notifications.Service
	search        *search.Service
	trending      *trending.Service
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
	postsSvc := posts.NewService(store, accountsSvc)

	s := &Server{
		app:           mizu.New(),
		cfg:           cfg,
		store:         store,
		accounts:      accountsSvc,
		posts:         postsSvc,
		timelines:     timelines.NewService(store, accountsSvc, postsSvc),
		interactions:  interactions.NewService(store),
		relationships: relationships.NewService(store),
		notifications: notifications.NewService(store, accountsSvc),
		search:        search.NewService(store),
		trending:      trending.NewService(store),
	}

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
	if s.store != nil {
		return s.store.Close()
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
