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

	"github.com/go-mizu/blueprints/kanban/app/web/handler"
	"github.com/go-mizu/blueprints/kanban/assets"
	"github.com/go-mizu/blueprints/kanban/feature/comments"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/labels"
	"github.com/go-mizu/blueprints/kanban/feature/notifications"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/search"
	"github.com/go-mizu/blueprints/kanban/feature/sprints"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/workspaces"
	"github.com/go-mizu/blueprints/kanban/store/duckdb"
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
	users         users.API
	workspaces    workspaces.API
	projects      projects.API
	issues        issues.API
	comments      comments.API
	labels        labels.API
	sprints       sprints.API
	notifications notifications.API
	search        search.API

	// Handlers
	authHandlers         *handler.Auth
	workspaceHandlers    *handler.Workspace
	projectHandlers      *handler.Project
	issueHandlers        *handler.Issue
	commentHandlers      *handler.Comment
	labelHandlers        *handler.Label
	sprintHandlers       *handler.Sprint
	notificationHandlers *handler.Notification
	pageHandlers         *handler.Page
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "kanban.duckdb")
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
	usersStore := duckdb.NewUsersStore(db)
	workspacesStore := duckdb.NewWorkspacesStore(db)
	projectsStore := duckdb.NewProjectsStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	labelsStore := duckdb.NewLabelsStore(db)
	sprintsStore := duckdb.NewSprintsStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)

	// Create services with stores
	usersSvc := users.NewService(usersStore)
	workspacesSvc := workspaces.NewService(workspacesStore)
	projectsSvc := projects.NewService(projectsStore)
	labelsSvc := labels.NewService(labelsStore)
	sprintsSvc := sprints.NewService(sprintsStore)
	issuesSvc := issues.NewService(issuesStore, projectsSvc)
	commentsSvc := comments.NewService(commentsStore)
	notificationsSvc := notifications.NewService(notificationsStore)
	searchSvc := search.NewService(issuesSvc)

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
		users:         usersSvc,
		workspaces:    workspacesSvc,
		projects:      projectsSvc,
		issues:        issuesSvc,
		comments:      commentsSvc,
		labels:        labelsSvc,
		sprints:       sprintsSvc,
		notifications: notificationsSvc,
		search:        searchSvc,
	}

	// Create handlers
	s.authHandlers = handler.NewAuth(usersSvc)
	s.workspaceHandlers = handler.NewWorkspace(workspacesSvc, s.getUserID)
	s.projectHandlers = handler.NewProject(projectsSvc, workspacesSvc, s.getUserID)
	s.issueHandlers = handler.NewIssue(issuesSvc, projectsSvc, workspacesSvc, s.getUserID)
	s.commentHandlers = handler.NewComment(commentsSvc, s.getUserID)
	s.labelHandlers = handler.NewLabel(labelsSvc)
	s.sprintHandlers = handler.NewSprint(sprintsSvc)
	s.notificationHandlers = handler.NewNotification(notificationsSvc, s.getUserID)
	s.pageHandlers = handler.NewPage(tmpl, usersSvc, workspacesSvc, projectsSvc, issuesSvc, labelsSvc, sprintsSvc, notificationsSvc, s.optionalAuth, cfg.Dev)

	s.setupRoutes()

	// Serve static files
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
	log.Printf("Starting Kanban server on %s", s.cfg.Addr)
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

// NewServer creates a server with just a database path (for CLI use).
func NewServer(dbPath string) (*Server, error) {
	dir := filepath.Dir(dbPath)
	return New(Config{
		Addr:    ":8080",
		DataDir: dir,
		Dev:     false,
	})
}

// Init initializes the database schema.
func (s *Server) Init(ctx context.Context) error {
	// Schema is already initialized in New()
	return nil
}

// UserService returns the users service.
func (s *Server) UserService() users.API {
	return s.users
}

// WorkspaceService returns the workspaces service.
func (s *Server) WorkspaceService() workspaces.API {
	return s.workspaces
}

// ProjectService returns the projects service.
func (s *Server) ProjectService() projects.API {
	return s.projects
}

// IssueService returns the issues service.
func (s *Server) IssueService() issues.API {
	return s.issues
}

// LabelService returns the labels service.
func (s *Server) LabelService() labels.API {
	return s.labels
}

// SprintService returns the sprints service.
func (s *Server) SprintService() sprints.API {
	return s.sprints
}

func (s *Server) setupRoutes() {
	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Workspaces
		api.Get("/workspaces", s.authRequired(s.workspaceHandlers.List))
		api.Post("/workspaces", s.authRequired(s.workspaceHandlers.Create))
		api.Get("/workspaces/{slug}", s.authRequired(s.workspaceHandlers.Get))
		api.Patch("/workspaces/{slug}", s.authRequired(s.workspaceHandlers.Update))
		api.Delete("/workspaces/{slug}", s.authRequired(s.workspaceHandlers.Delete))
		api.Get("/workspaces/{slug}/members", s.authRequired(s.workspaceHandlers.ListMembers))
		api.Post("/workspaces/{slug}/members", s.authRequired(s.workspaceHandlers.AddMember))

		// Projects
		api.Get("/workspaces/{slug}/projects", s.authRequired(s.projectHandlers.List))
		api.Post("/workspaces/{slug}/projects", s.authRequired(s.projectHandlers.Create))
		api.Get("/workspaces/{slug}/projects/{key}", s.authRequired(s.projectHandlers.Get))
		api.Patch("/workspaces/{slug}/projects/{key}", s.authRequired(s.projectHandlers.Update))
		api.Delete("/workspaces/{slug}/projects/{key}", s.authRequired(s.projectHandlers.Delete))

		// Issues
		api.Get("/projects/{projectID}/issues", s.authRequired(s.issueHandlers.List))
		api.Post("/projects/{projectID}/issues", s.authRequired(s.issueHandlers.Create))
		api.Get("/issues/{key}", s.authRequired(s.issueHandlers.Get))
		api.Patch("/issues/{key}", s.authRequired(s.issueHandlers.Update))
		api.Delete("/issues/{key}", s.authRequired(s.issueHandlers.Delete))
		api.Post("/issues/{key}/move", s.authRequired(s.issueHandlers.Move))
		api.Post("/issues/{key}/assignees", s.authRequired(s.issueHandlers.AddAssignee))
		api.Delete("/issues/{key}/assignees/{userID}", s.authRequired(s.issueHandlers.RemoveAssignee))
		api.Post("/issues/{key}/labels", s.authRequired(s.issueHandlers.AddLabel))
		api.Delete("/issues/{key}/labels/{labelID}", s.authRequired(s.issueHandlers.RemoveLabel))

		// Comments
		api.Get("/issues/{key}/comments", s.authRequired(s.commentHandlers.List))
		api.Post("/issues/{key}/comments", s.authRequired(s.commentHandlers.Create))
		api.Patch("/comments/{id}", s.authRequired(s.commentHandlers.Update))
		api.Delete("/comments/{id}", s.authRequired(s.commentHandlers.Delete))

		// Labels
		api.Get("/projects/{projectID}/labels", s.authRequired(s.labelHandlers.List))
		api.Post("/projects/{projectID}/labels", s.authRequired(s.labelHandlers.Create))
		api.Patch("/labels/{id}", s.authRequired(s.labelHandlers.Update))
		api.Delete("/labels/{id}", s.authRequired(s.labelHandlers.Delete))

		// Sprints
		api.Get("/projects/{projectID}/sprints", s.authRequired(s.sprintHandlers.List))
		api.Post("/projects/{projectID}/sprints", s.authRequired(s.sprintHandlers.Create))
		api.Patch("/sprints/{id}", s.authRequired(s.sprintHandlers.Update))
		api.Delete("/sprints/{id}", s.authRequired(s.sprintHandlers.Delete))
		api.Post("/sprints/{id}/start", s.authRequired(s.sprintHandlers.Start))
		api.Post("/sprints/{id}/complete", s.authRequired(s.sprintHandlers.Complete))

		// Notifications
		api.Get("/notifications", s.authRequired(s.notificationHandlers.List))
		api.Post("/notifications/{id}/read", s.authRequired(s.notificationHandlers.MarkRead))
		api.Post("/notifications/read-all", s.authRequired(s.notificationHandlers.MarkAllRead))
	})

	// Web routes (pages)
	s.app.Get("/", s.pageHandlers.Home)
	s.app.Get("/login", s.pageHandlers.Login)
	s.app.Get("/register", s.pageHandlers.Register)
	s.app.Get("/notifications", s.pageHandlers.Notifications)
	s.app.Get("/settings", s.pageHandlers.Settings)
	s.app.Get("/{workspace}", s.pageHandlers.Workspace)
	s.app.Get("/{workspace}/projects", s.pageHandlers.Projects)
	s.app.Get("/{workspace}/projects/{key}", s.pageHandlers.Board)
	s.app.Get("/{workspace}/projects/{key}/list", s.pageHandlers.List)
	s.app.Get("/{workspace}/projects/{key}/backlog", s.pageHandlers.Backlog)
	s.app.Get("/{workspace}/projects/{key}/sprints", s.pageHandlers.Sprints)
	s.app.Get("/{workspace}/projects/{key}/settings", s.pageHandlers.ProjectSettings)
	s.app.Get("/{workspace}/issue/{key}", s.pageHandlers.Issue)
	s.app.Get("/{workspace}/settings", s.pageHandlers.WorkspaceSettings)
	s.app.Get("/{workspace}/members", s.pageHandlers.Members)
}
