package web

import (
	"context"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/app/web/handler"
	"github.com/go-mizu/mizu/blueprints/news/assets"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/tags"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
	"github.com/go-mizu/mizu/blueprints/news/store/duckdb"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Addr string
	Dev  bool
}

// Server is the news web server.
type Server struct {
	app       *mizu.App
	store     *duckdb.Store
	templates *template.Template
	config    ServerConfig

	// Services
	users    *users.Service
	stories  *stories.Service
	comments *comments.Service
	votes    *votes.Service
	tags     *tags.Service

	// Handlers
	auth    *handler.Auth
	story   *handler.Story
	comment *handler.Comment
	vote    *handler.Vote
	user    *handler.User
	page    *handler.Page
}

// NewServer creates a new server with the given store and config.
func NewServer(store *duckdb.Store, cfg ServerConfig) (*Server, error) {
	// Create services
	usersStore := store.Users()
	storiesStore := store.Stories()
	commentsStore := store.Comments()
	votesStore := store.Votes()
	tagsStore := store.Tags()

	usersSvc := users.NewService(usersStore)
	tagsSvc := tags.NewService(tagsStore)
	storiesSvc := stories.NewService(storiesStore, usersStore, votesStore, tagsStore)
	commentsSvc := comments.NewService(commentsStore, storiesStore, usersStore, votesStore)
	votesSvc := votes.NewService(votesStore)

	// Create Mizu app
	app := mizu.New()

	// Load templates
	templates, err := assets.LoadTemplates()
	if err != nil {
		return nil, err
	}

	s := &Server{
		app:       app,
		store:     store,
		templates: templates,
		config:    cfg,
		users:     usersSvc,
		stories:   storiesSvc,
		comments:  commentsSvc,
		votes:     votesSvc,
		tags:      tagsSvc,
	}

	// Create handlers
	s.auth = handler.NewAuth(usersSvc, s.getUserID)
	s.story = handler.NewStory(storiesSvc, s.getUserID)
	s.comment = handler.NewComment(commentsSvc, s.getUserID)
	s.vote = handler.NewVote(storiesSvc, commentsSvc, s.getUserID)
	s.user = handler.NewUser(usersSvc, storiesSvc, commentsSvc)
	s.page = handler.NewPage(templates, usersSvc, storiesSvc, commentsSvc, tagsSvc, s.getUserID)

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// Start starts the server and blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
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

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.app.Router
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	r := s.app.Router

	// Static files
	r.Get("/static/{path...}", s.serveStatic)

	// API routes
	r.Group("/api", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.auth.Register)
		api.Post("/auth/login", s.auth.Login)
		api.Post("/auth/logout", s.authRequired(s.auth.Logout))

		// Stories
		api.Get("/stories", s.story.List)
		api.Get("/stories/{id}", s.story.Get)
		api.Post("/stories", s.authRequired(s.story.Create))

		// Story voting
		api.Post("/stories/{id}/vote", s.authRequired(s.vote.VoteStory))
		api.Delete("/stories/{id}/vote", s.authRequired(s.vote.UnvoteStory))

		// Comments
		api.Post("/comments", s.authRequired(s.comment.Create))
		api.Post("/comments/{id}/vote", s.authRequired(s.vote.VoteComment))
		api.Delete("/comments/{id}/vote", s.authRequired(s.vote.UnvoteComment))

		// Users
		api.Get("/users/{username}", s.user.Get)
	})

	// HTML pages
	r.Get("/", s.page.Home)
	r.Get("/newest", s.page.Newest)
	r.Get("/top", s.page.Top)
	r.Get("/story/{id}", s.page.Story)
	r.Get("/submit", s.authRequired(s.page.Submit))
	r.Get("/user/{username}", s.page.User)
	r.Get("/tag/{name}", s.page.Tag)
	r.Get("/login", s.page.Login)
	r.Get("/register", s.page.Register)

	// Form submissions (for HTML forms)
	r.Post("/submit", s.authRequired(s.page.SubmitPost))
	r.Post("/comment", s.authRequired(s.page.CommentPost))
	r.Post("/login", s.page.LoginPost)
	r.Post("/register", s.page.RegisterPost)
	r.Post("/logout", s.authRequired(s.page.LogoutPost))
}

func (s *Server) serveStatic(c *mizu.Ctx) error {
	path := c.Param("path")
	data, contentType, err := assets.GetStatic(path)
	if err != nil {
		return c.Text(404, "Not found")
	}
	c.Header().Set("Content-Type", contentType)
	c.Header().Set("Cache-Control", "public, max-age=86400")
	return c.Bytes(200, data, contentType)
}

// getUserID extracts the user ID from the request.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// Check Authorization header
	auth := c.Header().Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.users.GetSession(c.Request().Context(), token)
		if err == nil && session != nil {
			return session.UserID
		}
	}

	// Check cookie
	cookie, err := c.Request().Cookie("session")
	if err == nil && cookie != nil {
		session, err := s.users.GetSession(c.Request().Context(), cookie.Value)
		if err == nil && session != nil {
			return session.UserID
		}
	}

	return ""
}

// authRequired middleware requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		userID := s.getUserID(c)
		if userID == "" {
			// Check if this is an API request or page request
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/api/") {
				return handler.Unauthorized(c)
			}
			// Redirect to login for page requests
			return c.Redirect(302, "/login?next="+path)
		}
		return next(c)
	}
}
