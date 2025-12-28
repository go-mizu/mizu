package web

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/app/web/handler"
	"github.com/go-mizu/blueprints/githome/assets"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/go-mizu/mizu"
)

// Config is the server configuration
type Config struct {
	Addr     string
	DataDir  string
	ReposDir string
	Dev      bool
}

// Server is the GitHome HTTP server
type Server struct {
	app *mizu.App
	cfg Config
	db  *sql.DB

	// Services
	users  users.API
	repos  repos.API
	issues issues.API

	// Handlers
	authHandler  *handler.Auth
	userHandler  *handler.User
	repoHandler  *handler.Repo
	issueHandler *handler.Issue
	pageHandler  *handler.Page

	// Templates
	templates map[string]*template.Template
}

// New creates a new server
func New(cfg Config) (*Server, error) {
	// Open database
	dbPath := cfg.DataDir + "/githome.db"
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create stores
	store, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Ensure schema
	if err := store.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	usersStore := duckdb.NewUsersStore(db)
	reposStore := duckdb.NewReposStore(db)
	issuesStore := duckdb.NewIssuesStore(db)

	// Create services
	usersSvc := users.NewService(usersStore)
	reposSvc := repos.NewService(reposStore, cfg.ReposDir)
	issuesSvc := issues.NewService(issuesStore)

	// Create user getter function
	getUserID := func(c *mizu.Ctx) string {
		cookie, err := c.Cookie("session")
		if err != nil || cookie.Value == "" {
			return ""
		}
		user, err := usersSvc.ValidateSession(c.Context(), cookie.Value)
		if err != nil || user == nil {
			return ""
		}
		return user.ID
	}

	getUser := func(c *mizu.Ctx) *users.User {
		cookie, err := c.Cookie("session")
		if err != nil || cookie.Value == "" {
			return nil
		}
		user, err := usersSvc.ValidateSession(c.Context(), cookie.Value)
		if err != nil {
			return nil
		}
		return user
	}

	// Parse templates
	templates, err := assets.Templates()
	if err != nil {
		slog.Warn("failed to parse templates", "error", err)
		templates = make(map[string]*template.Template)
	}

	// Create handlers
	authHandler := handler.NewAuth(usersSvc)
	userHandler := handler.NewUser(usersSvc, reposSvc, getUserID)
	repoHandler := handler.NewRepo(reposSvc, usersSvc, getUserID)
	issueHandler := handler.NewIssue(issuesSvc, reposSvc, getUserID)
	pageHandler := handler.NewPage(usersSvc, reposSvc, issuesSvc, getUser, templates)

	// Create Mizu app
	app := mizu.New()

	srv := &Server{
		app:          app,
		cfg:          cfg,
		db:           db,
		users:        usersSvc,
		repos:        reposSvc,
		issues:       issuesSvc,
		authHandler:  authHandler,
		userHandler:  userHandler,
		repoHandler:  repoHandler,
		issueHandler: issueHandler,
		pageHandler:  pageHandler,
		templates:    templates,
	}

	srv.setupRoutes()

	return srv, nil
}

// reservedPaths are paths that cannot be used as usernames
var reservedPaths = map[string]bool{
	"api":      true,
	"static":   true,
	"login":    true,
	"register": true,
	"explore":  true,
	"new":      true,
	"livez":    true,
	"readyz":   true,
	"settings": true,
}

func (s *Server) setupRoutes() {
	// Static files - served via custom handler to avoid route conflicts
	staticFS := assets.Static()
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandler.Register)
		api.Post("/auth/login", s.authHandler.Login)
		api.Post("/auth/logout", s.authHandler.Logout)
		api.Get("/auth/me", s.authHandler.Me)

		// Users
		api.Get("/user", s.userHandler.GetCurrent)
		api.Patch("/user", s.userHandler.UpdateCurrent)
		api.Get("/user/repos", s.userHandler.ListRepos)
		api.Get("/user/starred", s.userHandler.ListStarred)
		api.Get("/users/{username}", s.userHandler.GetByUsername)
		api.Get("/users/{username}/repos", s.userHandler.ListUserRepos)

		// Repositories
		api.Get("/repos", s.repoHandler.ListPublic)
		api.Post("/repos", s.repoHandler.Create)
		api.Get("/repos/{owner}/{repo}", s.repoHandler.Get)
		api.Patch("/repos/{owner}/{repo}", s.repoHandler.Update)
		api.Delete("/repos/{owner}/{repo}", s.repoHandler.Delete)

		// Stars
		api.Put("/user/starred/{owner}/{repo}", s.repoHandler.Star)
		api.Delete("/user/starred/{owner}/{repo}", s.repoHandler.Unstar)
		api.Get("/repos/{owner}/{repo}/stargazers", s.repoHandler.ListStargazers)

		// Collaborators
		api.Get("/repos/{owner}/{repo}/collaborators", s.repoHandler.ListCollaborators)
		api.Put("/repos/{owner}/{repo}/collaborators/{username}", s.repoHandler.AddCollaborator)
		api.Delete("/repos/{owner}/{repo}/collaborators/{username}", s.repoHandler.RemoveCollaborator)

		// Issues
		api.Get("/repos/{owner}/{repo}/issues", s.issueHandler.List)
		api.Post("/repos/{owner}/{repo}/issues", s.issueHandler.Create)
		api.Get("/repos/{owner}/{repo}/issues/{number}", s.issueHandler.Get)
		api.Patch("/repos/{owner}/{repo}/issues/{number}", s.issueHandler.Update)
		api.Delete("/repos/{owner}/{repo}/issues/{number}", s.issueHandler.Delete)

		// Issue state
		api.Put("/repos/{owner}/{repo}/issues/{number}/lock", s.issueHandler.Lock)
		api.Delete("/repos/{owner}/{repo}/issues/{number}/lock", s.issueHandler.Unlock)

		// Issue labels
		api.Get("/repos/{owner}/{repo}/issues/{number}/labels", s.issueHandler.ListLabels)
		api.Post("/repos/{owner}/{repo}/issues/{number}/labels", s.issueHandler.AddLabels)
		api.Delete("/repos/{owner}/{repo}/issues/{number}/labels/{label}", s.issueHandler.RemoveLabel)

		// Issue comments
		api.Get("/repos/{owner}/{repo}/issues/{number}/comments", s.issueHandler.ListComments)
		api.Post("/repos/{owner}/{repo}/issues/{number}/comments", s.issueHandler.AddComment)
	})

	// Health checks
	s.app.Get("/livez", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "ok")
	})
	s.app.Get("/readyz", func(c *mizu.Ctx) error {
		if err := s.db.Ping(); err != nil {
			return c.Text(http.StatusServiceUnavailable, "database unavailable")
		}
		return c.Text(http.StatusOK, "ok")
	})

	// Catch-all handler for all HTML pages
	// This handles: /, /static/*, /login, /register, /explore, /new, /{username}, /{owner}/{repo}, etc.
	s.app.Get("/{path...}", func(c *mizu.Ctx) error {
		path := c.Request().URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		// Handle root path
		if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
			return s.pageHandler.Home(c)
		}

		// Handle static files
		if parts[0] == "static" {
			staticHandler.ServeHTTP(c.Writer(), c.Request())
			return nil
		}

		// Handle reserved/static HTML pages
		if len(parts) == 1 {
			switch parts[0] {
			case "login":
				return s.pageHandler.Login(c)
			case "register":
				return s.pageHandler.Register(c)
			case "explore":
				return s.pageHandler.Explore(c)
			case "new":
				return s.pageHandler.NewRepo(c)
			default:
				// /{username}
				return s.pageHandler.UserProfile(c)
			}
		}

		// Route based on path structure
		switch len(parts) {
		case 2:
			// /{owner}/{repo}
			return s.pageHandler.RepoHome(c)
		case 3:
			// /{owner}/{repo}/issues or /{owner}/{repo}/settings
			switch parts[2] {
			case "issues":
				return s.pageHandler.RepoIssues(c)
			case "settings":
				return s.pageHandler.RepoSettings(c)
			}
		case 4:
			// /{owner}/{repo}/issues/new or /{owner}/{repo}/issues/{number}
			if parts[2] == "issues" {
				if parts[3] == "new" {
					return s.pageHandler.NewIssue(c)
				}
				return s.pageHandler.IssueView(c)
			}
		}

		// Not found
		c.Writer().WriteHeader(http.StatusNotFound)
		return c.Text(http.StatusNotFound, "Not Found")
	})
}

// Run starts the server
func (s *Server) Run() error {
	return s.app.Listen(s.cfg.Addr)
}

// Close closes the server
func (s *Server) Close() error {
	return s.db.Close()
}

// Unused imports placeholder to avoid compile errors
var _ = time.Now
var _ = fmt.Sprintf
