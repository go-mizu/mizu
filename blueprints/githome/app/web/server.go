package web

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
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

	// Create handlers
	authHandler := handler.NewAuth(usersSvc)
	userHandler := handler.NewUser(usersSvc, reposSvc, getUserID)
	repoHandler := handler.NewRepo(reposSvc, usersSvc, getUserID)
	issueHandler := handler.NewIssue(issuesSvc, reposSvc, getUserID)
	pageHandler := handler.NewPage(usersSvc, reposSvc, issuesSvc, getUser)

	// Parse templates
	templates, err := parseTemplates(cfg.Dev)
	if err != nil {
		slog.Warn("failed to parse templates", "error", err)
	}

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

func (s *Server) setupRoutes() {
	// Static files
	staticSub, _ := assets.StaticFS()
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticSub)))
	s.app.Get("/static/{path...}", func(c *mizu.Ctx) error {
		staticHandler.ServeHTTP(c.Writer(), c.Request())
		return nil
	})

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

	// HTML pages
	s.app.Get("/", s.pageHandler.Home)
	s.app.Get("/login", s.pageHandler.Login)
	s.app.Get("/register", s.pageHandler.Register)
	s.app.Get("/explore", s.pageHandler.Explore)
	s.app.Get("/new", s.pageHandler.NewRepo)
	s.app.Get("/{username}", s.pageHandler.UserProfile)
	s.app.Get("/{owner}/{repo}", s.pageHandler.RepoHome)
	s.app.Get("/{owner}/{repo}/issues", s.pageHandler.RepoIssues)
	s.app.Get("/{owner}/{repo}/issues/new", s.pageHandler.NewIssue)
	s.app.Get("/{owner}/{repo}/issues/{number}", s.pageHandler.IssueView)
	s.app.Get("/{owner}/{repo}/settings", s.pageHandler.RepoSettings)

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
}

// Run starts the server
func (s *Server) Run() error {
	return s.app.Listen(s.cfg.Addr)
}

// Close closes the server
func (s *Server) Close() error {
	return s.db.Close()
}

func parseTemplates(dev bool) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006")
		},
		"timeAgo": func(t time.Time) string {
			d := time.Since(t)
			if d < time.Minute {
				return "just now"
			}
			if d < time.Hour {
				return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
			}
			if d < 24*time.Hour {
				return fmt.Sprintf("%d hours ago", int(d.Hours()))
			}
			if d < 7*24*time.Hour {
				return fmt.Sprintf("%d days ago", int(d.Hours()/24))
			}
			return t.Format("Jan 2, 2006")
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	// In development, load from filesystem
	if dev {
		viewsDir := "./assets/views"
		if _, err := os.Stat(viewsDir); os.IsNotExist(err) {
			return templates, nil
		}

		// Load each page template
		pages := []string{
			"home", "login", "register", "explore", "new_repo",
			"user_profile", "repo_home", "repo_issues", "issue_view", "new_issue", "repo_settings",
		}

		for _, page := range pages {
			tmpl := template.New(page).Funcs(funcMap)
			// Parse base layout
			tmpl, err := tmpl.ParseFiles(
				viewsDir+"/layouts/base.html",
				viewsDir+"/"+page+".html",
			)
			if err != nil {
				slog.Warn("failed to parse template", "page", page, "error", err)
				continue
			}
			templates[page] = tmpl
		}
	}

	return templates, nil
}
