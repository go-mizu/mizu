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
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/app/web/handler"
	"github.com/go-mizu/blueprints/social/assets"
	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/interactions"
	"github.com/go-mizu/blueprints/social/feature/lists"
	"github.com/go-mizu/blueprints/social/feature/notifications"
	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/relationships"
	"github.com/go-mizu/blueprints/social/feature/search"
	"github.com/go-mizu/blueprints/social/feature/timelines"
	"github.com/go-mizu/blueprints/social/feature/trending"
	"github.com/go-mizu/blueprints/social/store/duckdb"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app       *mizu.App
	cfg       Config
	db        *sql.DB
	templates *template.Template

	// Services
	accounts      accounts.API
	posts         posts.API
	timelines     timelines.API
	interactions  interactions.API
	relationships relationships.API
	notifications notifications.API
	search        search.API
	trending      trending.API
	lists         lists.API

	// Handlers
	authHandlers         *handler.Auth
	accountHandlers      *handler.Account
	postHandlers         *handler.Post
	interactionHandlers  *handler.Interaction
	relationshipHandlers *handler.Relationship
	timelineHandlers     *handler.Timeline
	notificationHandlers *handler.Notification
	searchHandlers       *handler.Search
	listHandlers         *handler.List
	pageHandlers         *handler.Page
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "social.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	coreStore, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	if err := coreStore.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create stores
	accountsStore := duckdb.NewAccountsStore(db)
	postsStore := duckdb.NewPostsStore(db)
	interactionsStore := duckdb.NewInteractionsStore(db)
	relationshipsStore := duckdb.NewRelationshipsStore(db)
	timelinesStore := duckdb.NewTimelinesStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)
	searchStore := duckdb.NewSearchStore(db)
	trendingStore := duckdb.NewTrendingStore(db)
	listsStore := duckdb.NewListsStore(db)

	// Create services
	isPrivate := func(ctx context.Context, id string) (bool, error) {
		acc, err := accountsStore.GetByID(ctx, id)
		if err != nil {
			return false, err
		}
		return acc.Private, nil
	}

	accountsSvc := accounts.NewService(accountsStore, relationshipsStore)
	postsSvc := posts.NewService(postsStore, accountsSvc, interactionsStore)
	timelinesSvc := timelines.NewService(timelinesStore, accountsSvc, postsSvc)
	interactionsSvc := interactions.NewService(interactionsStore)
	relationshipsSvc := relationships.NewService(relationshipsStore, isPrivate)
	notificationsSvc := notifications.NewService(notificationsStore, accountsSvc, postsSvc)
	searchSvc := search.NewService(searchStore, accountsSvc, postsSvc)
	trendingSvc := trending.NewService(trendingStore, postsSvc)
	listsSvc := lists.NewService(listsStore, accountsSvc)

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
		lists:         listsSvc,
	}

	// Create handlers
	s.authHandlers = handler.NewAuth(accountsSvc)
	s.accountHandlers = handler.NewAccount(accountsSvc, relationshipsSvc, timelinesSvc, s.getAccountID, s.optionalAuth)
	s.postHandlers = handler.NewPost(postsSvc, s.getAccountID, s.optionalAuth)
	s.interactionHandlers = handler.NewInteraction(interactionsSvc, postsSvc, notificationsSvc, s.getAccountID)
	s.relationshipHandlers = handler.NewRelationship(relationshipsSvc, notificationsSvc, s.getAccountID)
	s.timelineHandlers = handler.NewTimeline(timelinesSvc, s.getAccountID, s.optionalAuth)
	s.notificationHandlers = handler.NewNotification(notificationsSvc, s.getAccountID)
	s.searchHandlers = handler.NewSearch(searchSvc, trendingSvc, postsSvc, s.optionalAuth)
	s.listHandlers = handler.NewList(listsSvc, s.getAccountID)
	s.pageHandlers = handler.NewPage(tmpl, accountsSvc, postsSvc, timelinesSvc, relationshipsSvc, notificationsSvc, trendingSvc, s.optionalAuth, cfg.Dev)

	s.setupRoutes()

	// Static files
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(assets.Static())))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
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
		api.Get("/accounts/search", s.accountHandlers.Search)
		api.Get("/accounts/relationships", s.authRequired(s.relationshipHandlers.GetRelationships))
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
		api.Get("/timelines/public", s.timelineHandlers.Public)
		api.Get("/timelines/tag/{tag}", s.timelineHandlers.Hashtag)
		api.Get("/timelines/list/{id}", s.authRequired(s.timelineHandlers.List))

		// Notifications
		api.Get("/notifications", s.authRequired(s.notificationHandlers.List))
		api.Get("/notifications/unread_count", s.authRequired(s.notificationHandlers.UnreadCount))
		api.Post("/notifications/clear", s.authRequired(s.notificationHandlers.Clear))
		api.Post("/notifications/{id}/dismiss", s.authRequired(s.notificationHandlers.Dismiss))

		// Search
		api.Get("/search", s.searchHandlers.Search)

		// Trends
		api.Get("/trends/tags", s.searchHandlers.TrendingTags)
		api.Get("/trends/posts", s.searchHandlers.TrendingPosts)

		// Bookmarks
		api.Get("/bookmarks", s.authRequired(s.timelineHandlers.Bookmarks))

		// Lists
		api.Get("/lists", s.authRequired(s.listHandlers.List))
		api.Post("/lists", s.authRequired(s.listHandlers.Create))
		api.Get("/lists/{id}", s.listHandlers.Get)
		api.Put("/lists/{id}", s.authRequired(s.listHandlers.Update))
		api.Delete("/lists/{id}", s.authRequired(s.listHandlers.Delete))
		api.Get("/lists/{id}/accounts", s.listHandlers.GetMembers)
		api.Post("/lists/{id}/accounts", s.authRequired(s.listHandlers.AddMember))
		api.Delete("/lists/{id}/accounts", s.authRequired(s.listHandlers.RemoveMember))

		// Follow requests
		api.Get("/follow_requests", s.authRequired(s.relationshipHandlers.GetPendingFollowers))
		api.Post("/follow_requests/{id}/authorize", s.authRequired(s.relationshipHandlers.AcceptFollow))
		api.Post("/follow_requests/{id}/reject", s.authRequired(s.relationshipHandlers.RejectFollow))
	})

	// Web routes
	s.app.Get("/", s.pageHandlers.Home)
	s.app.Get("/login", s.pageHandlers.Login)
	s.app.Get("/register", s.pageHandlers.Register)
	s.app.Get("/explore", s.pageHandlers.Explore)
	s.app.Get("/search", s.pageHandlers.Search)
	s.app.Get("/notifications", s.pageHandlers.Notifications)
	s.app.Get("/bookmarks", s.pageHandlers.Bookmarks)
	s.app.Get("/lists", s.pageHandlers.Lists)
	s.app.Get("/lists/{id}", s.pageHandlers.ListView)
	s.app.Get("/settings", s.pageHandlers.Settings)
	s.app.Get("/settings/{page}", s.pageHandlers.Settings)
	s.app.Get("/@{username}", s.pageHandlers.Profile)
	s.app.Get("/@{username}/post/{id}", s.pageHandlers.Post)
	s.app.Get("/@{username}/followers", s.pageHandlers.FollowList)
	s.app.Get("/@{username}/following", s.pageHandlers.FollowList)
	s.app.Get("/tags/{tag}", s.pageHandlers.Tag)
}

// getAccountID extracts the account ID from the request.
func (s *Server) getAccountID(c *mizu.Ctx) string {
	// Try Authorization header first
	auth := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.accounts.GetSession(c.Request().Context(), token)
		if err == nil {
			return session.AccountID
		}
	}

	// Try cookie
	cookie, err := c.Cookie("session")
	if err == nil && cookie.Value != "" {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil {
			return session.AccountID
		}
	}

	return ""
}

// optionalAuth extracts account ID if present but doesn't require it.
func (s *Server) optionalAuth(c *mizu.Ctx) string {
	return s.getAccountID(c)
}

// authRequired requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		accountID := s.getAccountID(c)
		if accountID == "" {
			return c.JSON(401, map[string]any{
				"error": map[string]any{
					"code":    "UNAUTHORIZED",
					"message": "Authentication required",
				},
			})
		}
		return next(c)
	}
}
