package web

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/app/web/handler"
	"github.com/go-mizu/mizu/blueprints/forum/assets"
	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/notifications"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
	"github.com/go-mizu/mizu/blueprints/forum/store/duckdb"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Addr string
	Dev  bool
}

// Server is the forum web server.
type Server struct {
	app       *mizu.App
	store     *duckdb.Store
	templates map[string]*template.Template
	config    ServerConfig

	// Services
	accounts      accounts.API
	boards        boards.API
	threads       threads.API
	comments      comments.API
	votes         votes.API
	bookmarks     bookmarks.API
	notifications notifications.API

	// Handlers
	auth         *handler.Auth
	board        *handler.Board
	thread       *handler.Thread
	comment      *handler.Comment
	vote         *handler.Vote
	user         *handler.User
	notification *handler.Notification
	page         *handler.Page
}

// NewServer creates a new server with the given store and config.
func NewServer(store *duckdb.Store, cfg ServerConfig) (*Server, error) {
	// Create services
	accountsSvc := accounts.NewService(store.Accounts())
	boardsSvc := boards.NewService(store.Boards())

	// Create threads and comments services
	threadsSvc := threads.NewService(store.Threads(), accountsSvc, boardsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, threadsSvc)

	votesSvc := votes.NewService(store.Votes(), threadsSvc, commentsSvc)
	bookmarksSvc := bookmarks.NewService(store.Bookmarks())
	notificationsSvc := notifications.NewService(store.Notifications(), accountsSvc, boardsSvc, threadsSvc, commentsSvc)

	// Create Mizu app
	app := mizu.New()

	// Load templates
	templates, err := assets.Templates()
	if err != nil {
		return nil, err
	}

	s := &Server{
		app:           app,
		store:         store,
		templates:     templates,
		config:        cfg,
		accounts:      accountsSvc,
		boards:        boardsSvc,
		threads:       threadsSvc,
		comments:      commentsSvc,
		votes:         votesSvc,
		bookmarks:     bookmarksSvc,
		notifications: notificationsSvc,
	}

	// Create handlers
	s.auth = handler.NewAuth(accountsSvc, s.getAccountID)
	s.board = handler.NewBoard(boardsSvc, s.getAccountID)
	s.thread = handler.NewThread(threadsSvc, boardsSvc, votesSvc, bookmarksSvc, s.getAccountID)
	s.comment = handler.NewComment(commentsSvc, threadsSvc, votesSvc, bookmarksSvc, s.getAccountID)
	s.vote = handler.NewVote(votesSvc, s.getAccountID)
	s.user = handler.NewUser(accountsSvc, threadsSvc, commentsSvc)
	s.notification = handler.NewNotification(notificationsSvc, s.getAccountID)
	s.page = handler.NewPage(
		s.templates, s.accounts, s.boards, s.threads, s.comments,
		s.votes, s.bookmarks, s.notifications, s.getAccountID,
	)

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// Start starts the server and blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	// Run in background, wait for context cancellation
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.app.Listen(s.config.Addr)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// Run starts the server.
func (s *Server) Run() error {
	return s.app.Listen(s.config.Addr)
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.app.Router
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.app.Router.ServeHTTP(w, r)
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	r := s.app.Router

	// Static files
	staticFS := assets.Static()
	r.Get("/static/{path...}", func(c *mizu.Ctx) error {
		path := c.Param("path")
		f, err := staticFS.Open(path)
		if err != nil {
			return c.Text(404, "Not found")
		}
		defer f.Close()

		stat, _ := f.Stat()
		http.ServeContent(c.Writer(), c.Request(), path, stat.ModTime(), f.(io.ReadSeeker))
		return nil
	})

	// API routes
	r.Group("/api", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.auth.Register)
		api.Post("/auth/login", s.auth.Login)
		api.Post("/auth/logout", s.authRequired(s.auth.Logout))
		api.Get("/auth/me", s.authRequired(s.auth.Me))

		// Boards
		api.Get("/boards", s.board.List)
		api.Post("/boards", s.authRequired(s.board.Create))
		api.Get("/boards/{name}", s.board.Get)
		api.Put("/boards/{name}", s.authRequired(s.modRequired(s.board.Update)))
		api.Post("/boards/{name}/join", s.authRequired(s.board.Join))
		api.Delete("/boards/{name}/join", s.authRequired(s.board.Leave))
		api.Get("/boards/{name}/moderators", s.board.ListModerators)

		// Threads
		api.Get("/threads", s.thread.List)
		api.Get("/threads/{id}", s.thread.Get)
		api.Put("/threads/{id}", s.authRequired(s.thread.Update))
		api.Delete("/threads/{id}", s.authRequired(s.thread.Delete))
		api.Post("/boards/{name}/threads", s.authRequired(s.thread.Create))

		// Voting
		api.Post("/threads/{id}/vote", s.authRequired(s.vote.VoteThread))
		api.Delete("/threads/{id}/vote", s.authRequired(s.vote.UnvoteThread))
		api.Post("/comments/{id}/vote", s.authRequired(s.vote.VoteComment))
		api.Delete("/comments/{id}/vote", s.authRequired(s.vote.UnvoteComment))

		// Bookmarks
		api.Post("/threads/{id}/bookmark", s.authRequired(s.thread.Bookmark))
		api.Delete("/threads/{id}/bookmark", s.authRequired(s.thread.Unbookmark))
		api.Post("/comments/{id}/bookmark", s.authRequired(s.comment.Bookmark))
		api.Delete("/comments/{id}/bookmark", s.authRequired(s.comment.Unbookmark))

		// Comments
		api.Get("/threads/{id}/comments", s.comment.List)
		api.Post("/threads/{id}/comments", s.authRequired(s.comment.Create))
		api.Get("/comments/{id}", s.comment.Get)
		api.Put("/comments/{id}", s.authRequired(s.comment.Update))
		api.Delete("/comments/{id}", s.authRequired(s.comment.Delete))

		// Users
		api.Get("/users/{username}", s.user.Get)
		api.Get("/users/{username}/threads", s.user.ListThreads)
		api.Get("/users/{username}/comments", s.user.ListComments)

		// Notifications
		api.Get("/notifications", s.authRequired(s.notification.List))
		api.Post("/notifications/read", s.authRequired(s.notification.MarkRead))
		api.Post("/notifications/read-all", s.authRequired(s.notification.MarkAllRead))
	})

	// HTML pages
	r.Get("/", s.page.Home)
	r.Get("/all", s.page.All)
	r.Get("/b/{name}", s.page.Board)
	r.Get("/b/{name}/submit", s.authRequired(s.page.Submit))
	r.Get("/b/{name}/{id}/{slug}", s.page.Thread)
	r.Get("/u/{username}", s.page.User)
	r.Get("/search", s.page.Search)
	r.Get("/login", s.page.Login)
	r.Get("/register", s.page.Register)
	r.Get("/settings", s.authRequired(s.page.Settings))
	r.Get("/bookmarks", s.authRequired(s.page.Bookmarks))
	r.Get("/notifications", s.authRequired(s.page.Notifications))
}

// getAccountID extracts the account ID from the request.
func (s *Server) getAccountID(c *mizu.Ctx) string {
	// Check Authorization header
	auth := c.Header().Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.accounts.GetSession(c.Request().Context(), token)
		if err == nil && session != nil {
			return session.AccountID
		}
	}

	// Check cookie
	cookie, err := c.Request().Cookie("session")
	if err == nil && cookie != nil {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil && session != nil {
			return session.AccountID
		}
	}

	return ""
}

// authRequired middleware requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		accountID := s.getAccountID(c)
		if accountID == "" {
			return handler.Unauthorized(c)
		}
		return next(c)
	}
}

// modRequired middleware requires moderator permissions.
func (s *Server) modRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		accountID := s.getAccountID(c)
		boardName := c.Param("name")
		if boardName == "" {
			return next(c)
		}

		board, err := s.boards.GetByName(c.Request().Context(), boardName)
		if err != nil {
			return handler.NotFound(c, "board")
		}

		isMod, _ := s.boards.IsModerator(c.Request().Context(), board.ID, accountID)
		if !isMod {
			// Check if admin
			account, err := s.accounts.GetByID(c.Request().Context(), accountID)
			if err != nil || !account.IsAdmin {
				return handler.Forbidden(c)
			}
		}

		return next(c)
	}
}
