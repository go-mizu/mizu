package web

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/app/web/handler"
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

	// Handler groups
	authHandlers         *handler.Auth
	accountHandlers      *handler.Account
	postHandlers         *handler.Post
	interactionHandlers  *handler.Interaction
	relationshipHandlers *handler.Relationship
	timelineHandlers     *handler.Timeline
	notificationHandlers *handler.Notification
	searchHandlers       *handler.Search
	pageHandlers         *handler.Page
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
	timelinesSvc := timelines.NewService(timelinesStore, accountsSvc)
	interactionsSvc := interactions.NewService(interactionsStore)
	relationshipsSvc := relationships.NewService(relationshipsStore)
	notificationsSvc := notifications.NewService(notificationsStore, accountsSvc)
	searchSvc := search.NewService(searchStore)
	trendingSvc := trending.NewService(trendingStore)

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
		timelines:     timelinesSvc,
		interactions:  interactionsSvc,
		relationships: relationshipsSvc,
		notifications: notificationsSvc,
		search:        searchSvc,
		trending:      trendingSvc,
	}

	// Create handler groups with dependencies
	s.authHandlers = handler.NewAuth(accountsSvc)
	s.accountHandlers = handler.NewAccount(accountsSvc, relationshipsSvc, timelinesSvc, s.getAccountID, s.optionalAuth)
	s.postHandlers = handler.NewPost(postsSvc, s.getAccountID, s.optionalAuth)
	s.interactionHandlers = handler.NewInteraction(interactionsSvc, postsSvc, accountsSvc, s.getAccountID)
	s.relationshipHandlers = handler.NewRelationship(relationshipsSvc, s.getAccountID)
	s.timelineHandlers = handler.NewTimeline(timelinesSvc, s.getAccountID, s.optionalAuth)
	s.notificationHandlers = handler.NewNotification(notificationsSvc, s.getAccountID)
	s.searchHandlers = handler.NewSearch(searchSvc, trendingSvc, postsSvc, s.optionalAuth)
	s.pageHandlers = handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relationshipsSvc, notificationsSvc, trendingSvc, s.optionalAuth, cfg.Dev)

	s.setupRoutes()

	// Serve static files from embedded assets
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		// Set correct Content-Type header based on file extension
		ext := filepath.Ext(c.Request().URL.Path)
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			c.Writer().Header().Set("Content-Type", contentType)
		}
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

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

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() *mizu.App {
	return s.app
}

func (s *Server) setupRoutes() {
	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))

		// Accounts
		api.Get("/accounts/verify_credentials", s.authRequired(s.accountHandlers.VerifyCredentials))
		api.Patch("/accounts/update_credentials", s.authRequired(s.accountHandlers.UpdateCredentials))
		api.Get("/accounts/{id}", s.accountHandlers.GetAccount)
		api.Get("/accounts/{id}/posts", s.accountHandlers.GetAccountPosts)
		api.Get("/accounts/{id}/followers", s.accountHandlers.GetAccountFollowers)
		api.Get("/accounts/{id}/following", s.accountHandlers.GetAccountFollowing)
		api.Post("/accounts/{id}/follow", s.authRequired(s.relationshipHandlers.Follow))
		api.Post("/accounts/{id}/unfollow", s.authRequired(s.relationshipHandlers.Unfollow))
		api.Post("/accounts/{id}/block", s.authRequired(s.relationshipHandlers.Block))
		api.Post("/accounts/{id}/unblock", s.authRequired(s.relationshipHandlers.Unblock))
		api.Post("/accounts/{id}/mute", s.authRequired(s.relationshipHandlers.Mute))
		api.Post("/accounts/{id}/unmute", s.authRequired(s.relationshipHandlers.Unmute))
		api.Get("/accounts/relationships", s.authRequired(s.relationshipHandlers.GetRelationships))

		// Posts
		api.Post("/posts", s.authRequired(s.postHandlers.Create))
		api.Get("/posts/{id}", s.postHandlers.Get)
		api.Put("/posts/{id}", s.authRequired(s.postHandlers.Update))
		api.Delete("/posts/{id}", s.authRequired(s.postHandlers.Delete))
		api.Get("/posts/{id}/context", s.postHandlers.GetContext)
		api.Post("/posts/{id}/like", s.authRequired(s.interactionHandlers.Like))
		api.Delete("/posts/{id}/like", s.authRequired(s.interactionHandlers.Unlike))
		api.Post("/posts/{id}/repost", s.authRequired(s.interactionHandlers.Repost))
		api.Delete("/posts/{id}/repost", s.authRequired(s.interactionHandlers.Unrepost))
		api.Post("/posts/{id}/bookmark", s.authRequired(s.interactionHandlers.Bookmark))
		api.Delete("/posts/{id}/bookmark", s.authRequired(s.interactionHandlers.Unbookmark))
		api.Get("/posts/{id}/liked_by", s.interactionHandlers.LikedBy)
		api.Get("/posts/{id}/reposted_by", s.interactionHandlers.RepostedBy)

		// Timelines
		api.Get("/timelines/home", s.authRequired(s.timelineHandlers.Home))
		api.Get("/timelines/local", s.timelineHandlers.Local)
		api.Get("/timelines/tag/{tag}", s.timelineHandlers.Hashtag)

		// Notifications
		api.Get("/notifications", s.authRequired(s.notificationHandlers.List))
		api.Post("/notifications/clear", s.authRequired(s.notificationHandlers.Clear))
		api.Post("/notifications/{id}/dismiss", s.authRequired(s.notificationHandlers.Dismiss))

		// Search
		api.Get("/search", s.searchHandlers.Search)

		// Trends
		api.Get("/trends/tags", s.searchHandlers.TrendingTags)
		api.Get("/trends/posts", s.searchHandlers.TrendingPosts)

		// Bookmarks
		api.Get("/bookmarks", s.authRequired(s.timelineHandlers.Bookmarks))
	})

	// Web routes (pages)
	s.app.Get("/", s.pageHandlers.Home)
	s.app.Get("/login", s.pageHandlers.Login)
	s.app.Get("/register", s.pageHandlers.Register)
	s.app.Get("/explore", s.pageHandlers.Explore)
	s.app.Get("/search", s.pageHandlers.Search)
	s.app.Get("/notifications", s.pageHandlers.Notifications)
	s.app.Get("/bookmarks", s.pageHandlers.Bookmarks)
	s.app.Get("/settings", s.pageHandlers.Settings)
	s.app.Get("/settings/{page}", s.pageHandlers.Settings)
	s.app.Get("/u/{username}", s.pageHandlers.Profile)
	s.app.Get("/u/{username}/post/{id}", s.pageHandlers.Post)
	s.app.Get("/u/{username}/followers", s.pageHandlers.FollowList)
	s.app.Get("/u/{username}/following", s.pageHandlers.FollowList)
	s.app.Get("/tags/{tag}", s.pageHandlers.Tag)
}
