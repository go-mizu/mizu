package web

import (
	"context"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/app/web/handler"
	"github.com/go-mizu/mizu/blueprints/qa/assets"
	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/answers"
	"github.com/go-mizu/mizu/blueprints/qa/feature/badges"
	"github.com/go-mizu/mizu/blueprints/qa/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
	"github.com/go-mizu/mizu/blueprints/qa/feature/notifications"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/search"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
	"github.com/go-mizu/mizu/blueprints/qa/feature/votes"
	"github.com/go-mizu/mizu/blueprints/qa/store/duckdb"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Addr  string
	Dev   bool
	Theme string
}

// Server is the QA web server.
type Server struct {
	app       *mizu.App
	store     *duckdb.Store
	templates map[string]*template.Template
	config    ServerConfig

	accounts      accounts.API
	questions     questions.API
	answers       answers.API
	comments      comments.API
	votes         votes.API
	bookmarks     bookmarks.API
	tags          tags.API
	badges        badges.API
	notifications notifications.API
	search        search.API

	auth     *handler.Auth
	question *handler.Question
	answer   *handler.Answer
	comment  *handler.Comment
	vote     *handler.Vote
	page     *handler.Page
	user     *handler.User
	tag      *handler.Tag
	badge    *handler.Badge
}

// NewServer creates a new server with the given store and config.
func NewServer(store *duckdb.Store, cfg ServerConfig) (*Server, error) {
	accountsSvc := accounts.NewService(store.Accounts())
	tagsSvc := tags.NewService(store.Tags())
	questionsSvc := questions.NewService(store.Questions(), accountsSvc, tagsSvc)
	answersSvc := answers.NewService(store.Answers(), accountsSvc, questionsSvc)
	commentsSvc := comments.NewService(store.Comments(), accountsSvc, questionsSvc, answersSvc)
	votesSvc := votes.NewService(store.Votes(), accountsSvc, questionsSvc, answersSvc, commentsSvc)
	bookmarksSvc := bookmarks.NewService(store.Bookmarks())
	badgesSvc := badges.NewService(store.Badges())
	notificationsSvc := notifications.NewService(store.Notifications())
	searchSvc := search.NewService(questionsSvc, tagsSvc, accountsSvc)

	app := mizu.New()

	theme := cfg.Theme
	if theme == "" {
		theme = "default"
	}

	templates, err := assets.TemplatesForTheme(theme)
	if err != nil {
		return nil, err
	}

	s := &Server{
		app:           app,
		store:         store,
		templates:     templates,
		config:        cfg,
		accounts:      accountsSvc,
		questions:     questionsSvc,
		answers:       answersSvc,
		comments:      commentsSvc,
		votes:         votesSvc,
		bookmarks:     bookmarksSvc,
		tags:          tagsSvc,
		badges:        badgesSvc,
		notifications: notificationsSvc,
		search:        searchSvc,
	}

	s.auth = handler.NewAuth(accountsSvc, s.getAccountID)
	s.question = handler.NewQuestion(questionsSvc, tagsSvc, s.getAccountID)
	s.answer = handler.NewAnswer(answersSvc, questionsSvc, s.getAccountID)
	s.comment = handler.NewComment(commentsSvc, s.getAccountID)
	s.vote = handler.NewVote(votesSvc, s.getAccountID)
	s.user = handler.NewUser(accountsSvc, questionsSvc)
	s.tag = handler.NewTag(tagsSvc, questionsSvc)
	s.badge = handler.NewBadge(badgesSvc)
	s.page = handler.NewPage(
		s.templates,
		s.accounts,
		s.questions,
		s.answers,
		s.comments,
		s.tags,
		s.badges,
		s.search,
		s.getAccountID,
	)

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

func (s *Server) setupRoutes() {
	r := s.app.Router

	staticFS := assets.Static()
	r.Get("/static/{path...}", func(c *mizu.Ctx) error {
		path := c.Param("path")
		f, err := staticFS.Open(path)
		if err != nil {
			return c.Text(404, "Not found")
		}
		defer f.Close()

		if strings.HasSuffix(path, ".css") {
			c.Writer().Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.HasSuffix(path, ".js") {
			c.Writer().Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		c.Writer().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		_, err = io.Copy(c.Writer(), f)
		return err
	})

	r.Get("/", s.page.Home)
	r.Get("/questions", s.page.Questions)
	r.Get("/questions/{id}/{slug}", s.page.Question)
	r.Get("/questions/{id}", s.page.Question)
	r.Get("/ask", s.page.Ask)
	r.Get("/tags", s.page.Tags)
	r.Get("/tags/{name}", s.page.Tag)
	r.Get("/users", s.page.Users)
	r.Get("/users/{id}", s.page.User)
	r.Get("/badges", s.page.Badges)
	r.Get("/search", s.page.Search)
	r.Get("/login", s.page.Login)
	r.Get("/register", s.page.Register)

	// API endpoints (basic)
	r.Post("/api/questions", s.question.Create)
	r.Get("/api/questions/{id}", s.question.Get)
	r.Post("/api/questions/{id}/answers", s.answer.Create)
	r.Post("/api/questions/{id}/comments", s.comment.CreateForQuestion)
	r.Post("/api/answers/{id}/comments", s.comment.CreateForAnswer)
	r.Post("/api/votes", s.vote.Cast)
}

func (s *Server) getAccountID(c *mizu.Ctx) string {
	if cookie, err := c.Request().Cookie("session"); err == nil {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil {
			return session.AccountID
		}
	}
	return ""
}
