package web

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/app/web/handler"
	"github.com/go-mizu/blueprints/kanban/assets"
	"github.com/go-mizu/blueprints/kanban/feature/assignees"
	"github.com/go-mizu/blueprints/kanban/feature/columns"
	"github.com/go-mizu/blueprints/kanban/feature/comments"
	"github.com/go-mizu/blueprints/kanban/feature/cycles"
	"github.com/go-mizu/blueprints/kanban/feature/fields"
	"github.com/go-mizu/blueprints/kanban/feature/issues"
	"github.com/go-mizu/blueprints/kanban/feature/projects"
	"github.com/go-mizu/blueprints/kanban/feature/teams"
	"github.com/go-mizu/blueprints/kanban/feature/users"
	"github.com/go-mizu/blueprints/kanban/feature/values"
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
	app *mizu.App
	cfg Config
	db  *sql.DB

	// Services
	users      users.API
	workspaces workspaces.API
	teams      teams.API
	projects   projects.API
	columns    columns.API
	issues     issues.API
	cycles     cycles.API
	comments   comments.API
	fields     fields.API
	values     values.API
	assignees  assignees.API

	// Handlers
	authHandlers      *handler.Auth
	workspaceHandlers *handler.Workspace
	teamHandlers      *handler.Team
	projectHandlers   *handler.Project
	columnHandlers    *handler.Column
	issueHandlers     *handler.Issue
	cycleHandlers     *handler.Cycle
	commentHandlers   *handler.Comment
	fieldHandlers     *handler.Field
	valueHandlers     *handler.Value
	assigneeHandlers  *handler.Assignee
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
	teamsStore := duckdb.NewTeamsStore(db)
	projectsStore := duckdb.NewProjectsStore(db)
	columnsStore := duckdb.NewColumnsStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	cyclesStore := duckdb.NewCyclesStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	fieldsStore := duckdb.NewFieldsStore(db)
	valuesStore := duckdb.NewValuesStore(db)
	assigneesStore := duckdb.NewAssigneesStore(db)

	// Create services with stores
	usersSvc := users.NewService(usersStore)
	workspacesSvc := workspaces.NewService(workspacesStore)
	teamsSvc := teams.NewService(teamsStore)
	projectsSvc := projects.NewService(projectsStore)
	columnsSvc := columns.NewService(columnsStore)
	cyclesSvc := cycles.NewService(cyclesStore)
	issuesSvc := issues.NewService(issuesStore, projectsSvc)
	commentsSvc := comments.NewService(commentsStore)
	fieldsSvc := fields.NewService(fieldsStore)
	valuesSvc := values.NewService(valuesStore)
	assigneesSvc := assignees.NewService(assigneesStore)

	s := &Server{
		app:        mizu.New(),
		cfg:        cfg,
		db:         db,
		users:      usersSvc,
		workspaces: workspacesSvc,
		teams:      teamsSvc,
		projects:   projectsSvc,
		columns:    columnsSvc,
		issues:     issuesSvc,
		cycles:     cyclesSvc,
		comments:   commentsSvc,
		fields:     fieldsSvc,
		values:     valuesSvc,
		assignees:  assigneesSvc,
	}

	// Create handlers
	s.authHandlers = handler.NewAuth(usersSvc)
	s.workspaceHandlers = handler.NewWorkspace(workspacesSvc, s.getUserID)
	s.teamHandlers = handler.NewTeam(teamsSvc, s.getUserID)
	s.projectHandlers = handler.NewProject(projectsSvc, s.getUserID)
	s.columnHandlers = handler.NewColumn(columnsSvc)
	s.issueHandlers = handler.NewIssue(issuesSvc, projectsSvc, s.getUserID)
	s.cycleHandlers = handler.NewCycle(cyclesSvc)
	s.commentHandlers = handler.NewComment(commentsSvc, s.getUserID)
	s.fieldHandlers = handler.NewField(fieldsSvc)
	s.valueHandlers = handler.NewValue(valuesSvc)
	s.assigneeHandlers = handler.NewAssignee(assigneesSvc)

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

// TeamService returns the teams service.
func (s *Server) TeamService() teams.API {
	return s.teams
}

// ProjectService returns the projects service.
func (s *Server) ProjectService() projects.API {
	return s.projects
}

// ColumnService returns the columns service.
func (s *Server) ColumnService() columns.API {
	return s.columns
}

// IssueService returns the issues service.
func (s *Server) IssueService() issues.API {
	return s.issues
}

// CycleService returns the cycles service.
func (s *Server) CycleService() cycles.API {
	return s.cycles
}

// CommentService returns the comments service.
func (s *Server) CommentService() comments.API {
	return s.comments
}

// FieldService returns the fields service.
func (s *Server) FieldService() fields.API {
	return s.fields
}

// ValueService returns the values service.
func (s *Server) ValueService() values.API {
	return s.values
}

// AssigneeService returns the assignees service.
func (s *Server) AssigneeService() assignees.API {
	return s.assignees
}

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

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

		// Teams (workspace-scoped)
		api.Get("/workspaces/{workspaceID}/teams", s.authRequired(s.teamHandlers.List))
		api.Post("/workspaces/{workspaceID}/teams", s.authRequired(s.teamHandlers.Create))

		// Teams (by ID)
		api.Get("/teams/{id}", s.authRequired(s.teamHandlers.Get))
		api.Patch("/teams/{id}", s.authRequired(s.teamHandlers.Update))
		api.Delete("/teams/{id}", s.authRequired(s.teamHandlers.Delete))
		api.Get("/teams/{id}/members", s.authRequired(s.teamHandlers.ListMembers))
		api.Post("/teams/{id}/members", s.authRequired(s.teamHandlers.AddMember))
		api.Patch("/teams/{id}/members/{userID}", s.authRequired(s.teamHandlers.UpdateMemberRole))
		api.Delete("/teams/{id}/members/{userID}", s.authRequired(s.teamHandlers.RemoveMember))

		// Projects (team-scoped)
		api.Get("/teams/{teamID}/projects", s.authRequired(s.projectHandlers.List))
		api.Post("/teams/{teamID}/projects", s.authRequired(s.projectHandlers.Create))

		// Projects (by ID)
		api.Get("/projects/{id}", s.authRequired(s.projectHandlers.Get))
		api.Patch("/projects/{id}", s.authRequired(s.projectHandlers.Update))
		api.Delete("/projects/{id}", s.authRequired(s.projectHandlers.Delete))

		// Columns (project-scoped)
		api.Get("/projects/{projectID}/columns", s.authRequired(s.columnHandlers.List))
		api.Post("/projects/{projectID}/columns", s.authRequired(s.columnHandlers.Create))
		api.Post("/projects/{projectID}/columns/{id}/default", s.authRequired(s.columnHandlers.SetDefault))

		// Columns (by ID)
		api.Patch("/columns/{id}", s.authRequired(s.columnHandlers.Update))
		api.Post("/columns/{id}/position", s.authRequired(s.columnHandlers.UpdatePosition))
		api.Post("/columns/{id}/archive", s.authRequired(s.columnHandlers.Archive))
		api.Delete("/columns/{id}/archive", s.authRequired(s.columnHandlers.Unarchive))
		api.Delete("/columns/{id}", s.authRequired(s.columnHandlers.Delete))

		// Issues (project-scoped)
		api.Get("/projects/{projectID}/issues", s.authRequired(s.issueHandlers.List))
		api.Post("/projects/{projectID}/issues", s.authRequired(s.issueHandlers.Create))
		api.Get("/projects/{projectID}/issues/search", s.authRequired(s.issueHandlers.Search))

		// Issues (column-scoped)
		api.Get("/columns/{columnID}/issues", s.authRequired(s.issueHandlers.ListByColumn))

		// Issues (cycle-scoped)
		api.Get("/cycles/{cycleID}/issues", s.authRequired(s.issueHandlers.ListByCycle))

		// Issues (by key)
		api.Get("/issues/{key}", s.authRequired(s.issueHandlers.Get))
		api.Patch("/issues/{key}", s.authRequired(s.issueHandlers.Update))
		api.Delete("/issues/{key}", s.authRequired(s.issueHandlers.Delete))
		api.Post("/issues/{key}/move", s.authRequired(s.issueHandlers.Move))
		api.Post("/issues/{key}/cycle", s.authRequired(s.issueHandlers.AttachCycle))
		api.Delete("/issues/{key}/cycle", s.authRequired(s.issueHandlers.DetachCycle))

		// Cycles (team-scoped)
		api.Get("/teams/{teamID}/cycles", s.authRequired(s.cycleHandlers.List))
		api.Post("/teams/{teamID}/cycles", s.authRequired(s.cycleHandlers.Create))
		api.Get("/teams/{teamID}/cycles/active", s.authRequired(s.cycleHandlers.GetActive))

		// Cycles (by ID)
		api.Get("/cycles/{id}", s.authRequired(s.cycleHandlers.Get))
		api.Patch("/cycles/{id}", s.authRequired(s.cycleHandlers.Update))
		api.Post("/cycles/{id}/status", s.authRequired(s.cycleHandlers.UpdateStatus))
		api.Delete("/cycles/{id}", s.authRequired(s.cycleHandlers.Delete))

		// Comments (issue-scoped)
		api.Get("/issues/{issueID}/comments", s.authRequired(s.commentHandlers.List))
		api.Post("/issues/{issueID}/comments", s.authRequired(s.commentHandlers.Create))

		// Comments (by ID)
		api.Patch("/comments/{id}", s.authRequired(s.commentHandlers.Update))
		api.Delete("/comments/{id}", s.authRequired(s.commentHandlers.Delete))

		// Fields (project-scoped)
		api.Get("/projects/{projectID}/fields", s.authRequired(s.fieldHandlers.List))
		api.Post("/projects/{projectID}/fields", s.authRequired(s.fieldHandlers.Create))

		// Fields (by ID)
		api.Get("/fields/{id}", s.authRequired(s.fieldHandlers.Get))
		api.Patch("/fields/{id}", s.authRequired(s.fieldHandlers.Update))
		api.Post("/fields/{id}/position", s.authRequired(s.fieldHandlers.UpdatePosition))
		api.Post("/fields/{id}/archive", s.authRequired(s.fieldHandlers.Archive))
		api.Delete("/fields/{id}/archive", s.authRequired(s.fieldHandlers.Unarchive))
		api.Delete("/fields/{id}", s.authRequired(s.fieldHandlers.Delete))

		// Values (issue-scoped)
		api.Get("/issues/{issueID}/values", s.authRequired(s.valueHandlers.ListByIssue))
		api.Post("/issues/{issueID}/values", s.authRequired(s.valueHandlers.BulkSet))
		api.Put("/issues/{issueID}/values/{fieldID}", s.authRequired(s.valueHandlers.Set))
		api.Get("/issues/{issueID}/values/{fieldID}", s.authRequired(s.valueHandlers.Get))
		api.Delete("/issues/{issueID}/values/{fieldID}", s.authRequired(s.valueHandlers.Delete))

		// Assignees (issue-scoped)
		api.Get("/issues/{issueID}/assignees", s.authRequired(s.assigneeHandlers.List))
		api.Post("/issues/{issueID}/assignees", s.authRequired(s.assigneeHandlers.Add))
		api.Delete("/issues/{issueID}/assignees/{userID}", s.authRequired(s.assigneeHandlers.Remove))
	})
}
