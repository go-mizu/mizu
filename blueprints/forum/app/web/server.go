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

	// Services (as interfaces)
	accounts accounts.API
	forums   forums.API
	threads  threads.API
	posts    posts.API
	votes    votes.API

	// Handler groups
	authHandlers    *handler.Auth
	accountHandlers *handler.Account
	forumHandlers   *handler.Forum
	threadHandlers  *handler.Thread
	postHandlers    *handler.Post
	voteHandlers    *handler.Vote
	pageHandlers    *handler.Handler // Legacy page handler
}

// New creates a new web server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	dataDir := filepath.Dir(cfg.DatabasePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	db, err := sql.Open("duckdb", cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create core store for schema initialization
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
	// TODO: Implement these stores in store/duckdb/
	// forumsStore := duckdb.NewForumsStore(db)
	// threadsStore := duckdb.NewThreadsStore(db)
	// postsStore := duckdb.NewPostsStore(db)
	// votesStore := duckdb.NewVotesStore(db)

	// Create services
	accountsSvc := accounts.NewService(accountsStore)
	// TODO: Implement these services in feature/*/service.go
	// forumsSvc := forums.NewService(forumsStore)
	// threadsSvc := threads.NewService(threadsStore, forumsSvc, accountsSvc)
	// postsSvc := posts.NewService(postsStore, threadsSvc, accountsSvc)
	// votesSvc := votes.NewService(votesStore, threadsSvc, postsSvc, accountsSvc)

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
		// TODO: Assign services when implemented
		// forums:    forumsSvc,
		// threads:   threadsSvc,
		// posts:     postsSvc,
		// votes:     votesSvc,
	}

	// Create handler groups with dependencies
	s.authHandlers = handler.NewAuth(accountsSvc)
	s.accountHandlers = handler.NewAccount(accountsSvc, s.getAccountID, s.optionalAuth)

	// TODO: Uncomment when services are implemented
	// s.forumHandlers = handler.NewForum(forumsSvc, s.getAccountID, s.optionalAuth)
	// s.threadHandlers = handler.NewThread(threadsSvc, s.getAccountID, s.optionalAuth)
	// s.postHandlers = handler.NewPost(postsSvc, s.getAccountID, s.optionalAuth)
	// s.voteHandlers = handler.NewVote(votesSvc, s.getAccountID)

	// Legacy page handler (for simple HTML pages)
	s.pageHandlers = handler.New(tmpl, accountsSvc, s.forums, s.threads, s.posts, s.votes)

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

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// ========== Authentication ==========
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/verify", s.authRequired(func(c *mizu.Ctx) error {
			return s.authHandlers.VerifyCredentials(c, s.getAccountID)
		}))

		// ========== Accounts ==========
		api.Get("/accounts/search", s.accountHandlers.Search)
		api.Get("/accounts/{id}", s.accountHandlers.Get)
		api.Patch("/accounts/{id}", s.authRequired(s.accountHandlers.Update))

		// TODO: Uncomment when thread/post services are implemented
		// api.Get("/accounts/{id}/threads", s.threadHandlers.ListByAccount)
		// api.Get("/accounts/{id}/posts", s.postHandlers.ListByAccount)

		// ========== Forums ==========
		// TODO: Uncomment when forum service is implemented
		/*
		api.Get("/forums", s.forumHandlers.List)
		api.Post("/forums", s.adminRequired(s.forumHandlers.Create))
		api.Get("/forums/{id}", s.forumHandlers.Get)
		api.Patch("/forums/{id}", s.moderatorRequired(s.forumHandlers.Update))
		api.Delete("/forums/{id}", s.adminRequired(s.forumHandlers.Delete))
		api.Post("/forums/{id}/join", s.authRequired(s.forumHandlers.Join))
		api.Post("/forums/{id}/leave", s.authRequired(s.forumHandlers.Leave))

		// Forum moderators
		api.Post("/forums/{id}/moderators", s.authRequired(s.forumHandlers.AddModerator))
		api.Delete("/forums/{id}/moderators/{account_id}", s.authRequired(s.forumHandlers.RemoveModerator))
		*/

		// ========== Threads ==========
		// TODO: Uncomment when thread service is implemented
		/*
		api.Get("/forums/{id}/threads", s.threadHandlers.ListByForum)
		api.Post("/forums/{id}/threads", s.authRequired(s.threadHandlers.Create))
		api.Get("/threads/{id}", s.threadHandlers.Get)
		api.Patch("/threads/{id}", s.authRequired(s.threadHandlers.Update))
		api.Delete("/threads/{id}", s.authRequired(s.threadHandlers.Delete))

		// Thread voting
		api.Post("/threads/{id}/vote", s.authRequired(s.voteHandlers.VoteThread))

		// Thread moderation
		api.Post("/threads/{id}/lock", s.moderatorRequired(s.threadHandlers.Lock))
		api.Post("/threads/{id}/unlock", s.moderatorRequired(s.threadHandlers.Unlock))
		api.Post("/threads/{id}/sticky", s.moderatorRequired(s.threadHandlers.Sticky))
		api.Post("/threads/{id}/unsticky", s.moderatorRequired(s.threadHandlers.Unsticky))

		// Thread subscriptions
		// TODO: Implement subscription handlers
		// api.Post("/threads/{id}/subscribe", s.authRequired(s.subscriptionHandlers.Subscribe))
		// api.Delete("/threads/{id}/subscribe", s.authRequired(s.subscriptionHandlers.Unsubscribe))
		*/

		// ========== Posts (Comments) ==========
		// TODO: Uncomment when post service is implemented
		/*
		api.Get("/threads/{id}/posts", s.postHandlers.ListByThread)
		api.Get("/threads/{id}/posts/tree", s.postHandlers.GetTree)
		api.Post("/threads/{id}/posts", s.authRequired(s.postHandlers.Create))
		api.Get("/posts/{id}", s.postHandlers.Get)
		api.Patch("/posts/{id}", s.authRequired(s.postHandlers.Update))
		api.Delete("/posts/{id}", s.authRequired(s.postHandlers.Delete))

		// Post voting
		api.Post("/posts/{id}/vote", s.authRequired(s.voteHandlers.VotePost))

		// Best answer (for Q&A threads)
		// TODO: Implement in thread/post handlers
		// api.Post("/posts/{id}/best", s.authRequired(s.postHandlers.MarkBestAnswer))
		*/

		// ========== Search ==========
		// TODO: Implement search service and handlers
		/*
		api.Get("/search", s.searchHandlers.Search)
		*/

		// ========== Trending ==========
		// TODO: Implement trending service and handlers
		/*
		api.Get("/trending/forums", s.trendingHandlers.Forums)
		api.Get("/trending/tags", s.trendingHandlers.Tags)
		api.Get("/trending/threads", s.trendingHandlers.Threads)
		*/

		// ========== Moderation ==========
		// TODO: Implement moderation service and handlers
		/*
		api.Get("/forums/{id}/queue", s.moderatorRequired(s.moderationHandlers.Queue))
		api.Post("/posts/{id}/approve", s.moderatorRequired(s.moderationHandlers.Approve))
		api.Post("/posts/{id}/remove", s.moderatorRequired(s.moderationHandlers.Remove))
		api.Get("/forums/{id}/reports", s.moderatorRequired(s.moderationHandlers.ListReports))
		api.Post("/reports", s.authRequired(s.moderationHandlers.CreateReport))
		api.Post("/forums/{id}/ban", s.moderatorRequired(s.moderationHandlers.BanUser))
		api.Delete("/forums/{id}/ban/{account_id}", s.moderatorRequired(s.moderationHandlers.UnbanUser))
		api.Get("/forums/{id}/logs", s.moderatorRequired(s.moderationHandlers.Logs))
		*/
	})

	// Web routes (pages) - using legacy page handler
	s.app.Get("/", s.pageHandlers.Home)
	s.app.Get("/login", s.pageHandlers.LoginPage)
	s.app.Post("/login", s.pageHandlers.Login)
	s.app.Get("/register", s.pageHandlers.RegisterPage)
	s.app.Post("/register", s.pageHandlers.Register)
	s.app.Post("/logout", s.pageHandlers.Logout)

	// Forum pages
	s.app.Get("/f/{slug}", s.pageHandlers.ForumPage)

	// Thread pages
	s.app.Get("/f/{slug}/t/{id}", s.pageHandlers.ThreadPage)

	// User profiles
	s.app.Get("/u/{username}", s.pageHandlers.ProfilePage)
}

// Start starts the web server.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	log.Printf("Starting server on http://%s", addr)
	return s.app.Listen(addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler {
	return s.app
}
