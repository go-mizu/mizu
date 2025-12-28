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
	"github.com/go-mizu/blueprints/githome/app/web/handler/api"
	"github.com/go-mizu/blueprints/githome/assets"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
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
	users      users.API
	repos      repos.API
	issues     issues.API
	pulls      pulls.API
	orgs       orgs.API
	labels     labels.API
	milestones milestones.API
	releases   releases.API
	teams      teams.API
	webhooks   webhooks.API

	// API Handlers
	authHandler      *api.Auth
	userHandler      *api.User
	repoHandler      *api.Repo
	issueHandler     *api.Issue
	pullHandler      *api.Pull
	orgHandler       *api.Org
	labelHandler     *api.Label
	milestoneHandler *api.Milestone
	releaseHandler   *api.Release
	teamHandler      *api.Team
	webhookHandler   *api.Webhook

	// Page Handler
	pageHandler *handler.Page

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

	// Create stores
	usersStore := duckdb.NewUsersStore(db)
	reposStore := duckdb.NewReposStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	pullsStore := duckdb.NewPullsStore(db)
	orgsStore := duckdb.NewOrgsStore(db)
	labelsStore := duckdb.NewLabelsStore(db)
	milestonesStore := duckdb.NewMilestonesStore(db)
	releasesStore := duckdb.NewReleasesStore(db)
	teamsStore := duckdb.NewTeamsStore(db)
	webhooksStore := duckdb.NewWebhooksStore(db)
	actorsStore := duckdb.NewActorsStore(db)

	// Create services
	usersSvc := users.NewService(usersStore)
	reposSvc := repos.NewService(reposStore, cfg.ReposDir)
	issuesSvc := issues.NewService(issuesStore)
	pullsSvc := pulls.NewService(pullsStore)
	orgsSvc := orgs.NewService(orgsStore)
	labelsSvc := labels.NewService(labelsStore)
	milestonesSvc := milestones.NewService(milestonesStore)
	releasesSvc := releases.NewService(releasesStore)
	teamsSvc := teams.NewService(teamsStore)
	webhooksSvc := webhooks.NewService(webhooksStore)

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

	// Create API handlers
	authHandler := api.NewAuth(usersSvc, actorsStore)
	userHandler := api.NewUser(usersSvc, reposSvc, getUserID)
	repoHandler := api.NewRepo(reposSvc, usersSvc, getUserID)
	issueHandler := api.NewIssue(issuesSvc, reposSvc, usersSvc, getUserID)
	pullHandler := api.NewPull(pullsSvc, reposSvc, usersSvc, getUserID)
	orgHandler := api.NewOrg(orgsSvc, usersSvc, getUserID)
	labelHandler := api.NewLabel(labelsSvc, reposSvc, usersSvc, getUserID)
	milestoneHandler := api.NewMilestone(milestonesSvc, reposSvc, usersSvc, getUserID)
	releaseHandler := api.NewRelease(releasesSvc, reposSvc, usersSvc, getUserID)
	teamHandler := api.NewTeam(teamsSvc, orgsSvc, reposSvc, usersSvc, getUserID)
	webhookHandler := api.NewWebhook(webhooksSvc, reposSvc, orgsSvc, usersSvc, getUserID)

	// Create page handler
	pageHandler := handler.NewPage(usersSvc, reposSvc, issuesSvc, getUser, templates)

	// Create Mizu app
	app := mizu.New()

	srv := &Server{
		app:              app,
		cfg:              cfg,
		db:               db,
		users:            usersSvc,
		repos:            reposSvc,
		issues:           issuesSvc,
		pulls:            pullsSvc,
		orgs:             orgsSvc,
		labels:           labelsSvc,
		milestones:       milestonesSvc,
		releases:         releasesSvc,
		teams:            teamsSvc,
		webhooks:         webhooksSvc,
		authHandler:      authHandler,
		userHandler:      userHandler,
		repoHandler:      repoHandler,
		issueHandler:     issueHandler,
		pullHandler:      pullHandler,
		orgHandler:       orgHandler,
		labelHandler:     labelHandler,
		milestoneHandler: milestoneHandler,
		releaseHandler:   releaseHandler,
		teamHandler:      teamHandler,
		webhookHandler:   webhookHandler,
		pageHandler:      pageHandler,
		templates:        templates,
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
	s.app.Group("/api/v1", func(r *mizu.Router) {
		// Auth
		r.Post("/auth/register", s.authHandler.Register)
		r.Post("/auth/login", s.authHandler.Login)
		r.Post("/auth/logout", s.authHandler.Logout)
		r.Get("/auth/me", s.authHandler.Me)

		// Users
		r.Get("/user", s.userHandler.GetCurrent)
		r.Patch("/user", s.userHandler.UpdateCurrent)
		r.Delete("/user", s.userHandler.Delete)
		r.Put("/user/password", s.userHandler.ChangePassword)
		r.Get("/user/repos", s.userHandler.ListRepos)
		r.Get("/user/starred", s.userHandler.ListStarred)
		r.Get("/user/orgs", s.orgHandler.ListUserOrgs)
		r.Get("/users", s.userHandler.List)
		r.Get("/users/{username}", s.userHandler.GetByUsername)
		r.Get("/users/{username}/repos", s.userHandler.ListUserRepos)
		r.Get("/users/{username}/orgs", s.orgHandler.ListUserOrgs)

		// Repositories
		r.Get("/repos", s.repoHandler.ListPublic)
		r.Post("/repos", s.repoHandler.Create)
		r.Get("/repos/{owner}/{repo}", s.repoHandler.Get)
		r.Patch("/repos/{owner}/{repo}", s.repoHandler.Update)
		r.Delete("/repos/{owner}/{repo}", s.repoHandler.Delete)

		// Stars
		r.Get("/user/starred/{owner}/{repo}", s.repoHandler.CheckStarred)
		r.Put("/user/starred/{owner}/{repo}", s.repoHandler.Star)
		r.Delete("/user/starred/{owner}/{repo}", s.repoHandler.Unstar)
		r.Get("/repos/{owner}/{repo}/stargazers", s.repoHandler.ListStargazers)

		// Forks
		r.Get("/repos/{owner}/{repo}/forks", s.repoHandler.ListForks)
		r.Post("/repos/{owner}/{repo}/forks", s.repoHandler.Fork)

		// Collaborators
		r.Get("/repos/{owner}/{repo}/collaborators", s.repoHandler.ListCollaborators)
		r.Put("/repos/{owner}/{repo}/collaborators/{username}", s.repoHandler.AddCollaborator)
		r.Delete("/repos/{owner}/{repo}/collaborators/{username}", s.repoHandler.RemoveCollaborator)
		r.Get("/repos/{owner}/{repo}/permission/{username}", s.repoHandler.GetPermission)

		// Labels
		r.Get("/repos/{owner}/{repo}/labels", s.labelHandler.List)
		r.Post("/repos/{owner}/{repo}/labels", s.labelHandler.Create)
		r.Get("/repos/{owner}/{repo}/labels/{name}", s.labelHandler.Get)
		r.Patch("/repos/{owner}/{repo}/labels/{name}", s.labelHandler.Update)
		r.Delete("/repos/{owner}/{repo}/labels/{name}", s.labelHandler.Delete)

		// Milestones
		r.Get("/repos/{owner}/{repo}/milestones", s.milestoneHandler.List)
		r.Post("/repos/{owner}/{repo}/milestones", s.milestoneHandler.Create)
		r.Get("/repos/{owner}/{repo}/milestones/{number}", s.milestoneHandler.Get)
		r.Patch("/repos/{owner}/{repo}/milestones/{number}", s.milestoneHandler.Update)
		r.Delete("/repos/{owner}/{repo}/milestones/{number}", s.milestoneHandler.Delete)

		// Issues
		r.Get("/repos/{owner}/{repo}/issues", s.issueHandler.List)
		r.Post("/repos/{owner}/{repo}/issues", s.issueHandler.Create)
		r.Get("/repos/{owner}/{repo}/issues/{number}", s.issueHandler.Get)
		r.Patch("/repos/{owner}/{repo}/issues/{number}", s.issueHandler.Update)
		r.Delete("/repos/{owner}/{repo}/issues/{number}", s.issueHandler.Delete)

		// Issue state
		r.Put("/repos/{owner}/{repo}/issues/{number}/lock", s.issueHandler.Lock)
		r.Delete("/repos/{owner}/{repo}/issues/{number}/lock", s.issueHandler.Unlock)

		// Issue labels
		r.Get("/repos/{owner}/{repo}/issues/{number}/labels", s.issueHandler.ListLabels)
		r.Post("/repos/{owner}/{repo}/issues/{number}/labels", s.issueHandler.AddLabels)
		r.Put("/repos/{owner}/{repo}/issues/{number}/labels", s.issueHandler.SetLabels)
		r.Delete("/repos/{owner}/{repo}/issues/{number}/labels/{label}", s.issueHandler.RemoveLabel)

		// Issue assignees
		r.Post("/repos/{owner}/{repo}/issues/{number}/assignees", s.issueHandler.AddAssignees)
		r.Delete("/repos/{owner}/{repo}/issues/{number}/assignees", s.issueHandler.RemoveAssignees)

		// Issue comments
		r.Get("/repos/{owner}/{repo}/issues/{number}/comments", s.issueHandler.ListComments)
		r.Post("/repos/{owner}/{repo}/issues/{number}/comments", s.issueHandler.AddComment)
		r.Patch("/repos/{owner}/{repo}/issues/{number}/comments/{id}", s.issueHandler.UpdateComment)
		r.Delete("/repos/{owner}/{repo}/issues/{number}/comments/{id}", s.issueHandler.DeleteComment)

		// Pull Requests
		r.Get("/repos/{owner}/{repo}/pulls", s.pullHandler.List)
		r.Post("/repos/{owner}/{repo}/pulls", s.pullHandler.Create)
		r.Get("/repos/{owner}/{repo}/pulls/{number}", s.pullHandler.Get)
		r.Patch("/repos/{owner}/{repo}/pulls/{number}", s.pullHandler.Update)
		r.Put("/repos/{owner}/{repo}/pulls/{number}/merge", s.pullHandler.Merge)
		r.Put("/repos/{owner}/{repo}/pulls/{number}/ready", s.pullHandler.MarkReady)
		r.Put("/repos/{owner}/{repo}/pulls/{number}/lock", s.pullHandler.Lock)
		r.Delete("/repos/{owner}/{repo}/pulls/{number}/lock", s.pullHandler.Unlock)

		// PR labels
		r.Get("/repos/{owner}/{repo}/pulls/{number}/labels", s.pullHandler.ListLabels)
		r.Post("/repos/{owner}/{repo}/pulls/{number}/labels", s.pullHandler.AddLabels)
		r.Delete("/repos/{owner}/{repo}/pulls/{number}/labels/{label}", s.pullHandler.RemoveLabel)

		// PR assignees
		r.Post("/repos/{owner}/{repo}/pulls/{number}/assignees", s.pullHandler.AddAssignees)
		r.Delete("/repos/{owner}/{repo}/pulls/{number}/assignees", s.pullHandler.RemoveAssignees)

		// PR reviewers
		r.Post("/repos/{owner}/{repo}/pulls/{number}/reviewers", s.pullHandler.RequestReview)
		r.Delete("/repos/{owner}/{repo}/pulls/{number}/reviewers", s.pullHandler.RemoveReviewRequest)

		// PR reviews
		r.Get("/repos/{owner}/{repo}/pulls/{number}/reviews", s.pullHandler.ListReviews)
		r.Post("/repos/{owner}/{repo}/pulls/{number}/reviews", s.pullHandler.CreateReview)
		r.Get("/repos/{owner}/{repo}/pulls/{number}/reviews/{id}", s.pullHandler.GetReview)
		r.Put("/repos/{owner}/{repo}/pulls/{number}/reviews/{id}", s.pullHandler.SubmitReview)
		r.Delete("/repos/{owner}/{repo}/pulls/{number}/reviews/{id}", s.pullHandler.DismissReview)

		// PR review comments
		r.Get("/repos/{owner}/{repo}/pulls/{number}/comments", s.pullHandler.ListReviewComments)
		r.Post("/repos/{owner}/{repo}/pulls/{number}/comments", s.pullHandler.CreateReviewComment)
		r.Patch("/repos/{owner}/{repo}/pulls/{number}/comments/{id}", s.pullHandler.UpdateReviewComment)
		r.Delete("/repos/{owner}/{repo}/pulls/{number}/comments/{id}", s.pullHandler.DeleteReviewComment)

		// Releases
		r.Get("/repos/{owner}/{repo}/releases", s.releaseHandler.List)
		r.Post("/repos/{owner}/{repo}/releases", s.releaseHandler.Create)
		r.Get("/repos/{owner}/{repo}/releases/latest", s.releaseHandler.GetLatest)
		r.Get("/repos/{owner}/{repo}/releases/{id}", s.releaseHandler.Get)
		r.Patch("/repos/{owner}/{repo}/releases/{id}", s.releaseHandler.Update)
		r.Delete("/repos/{owner}/{repo}/releases/{id}", s.releaseHandler.Delete)
		r.Put("/repos/{owner}/{repo}/releases/{id}/publish", s.releaseHandler.Publish)

		// Release by tag (uses query param to avoid route conflict)
		r.Get("/repos/{owner}/{repo}/release-by-tag/{tag}", s.releaseHandler.GetByTag)

		// Release assets (using separate path to avoid route conflicts)
		r.Get("/repos/{owner}/{repo}/release/{id}/assets", s.releaseHandler.ListAssets)
		r.Post("/repos/{owner}/{repo}/release/{id}/assets", s.releaseHandler.UploadAsset)
		r.Get("/repos/{owner}/{repo}/release-assets/{assetId}", s.releaseHandler.GetAsset)
		r.Patch("/repos/{owner}/{repo}/release-assets/{assetId}", s.releaseHandler.UpdateAsset)
		r.Delete("/repos/{owner}/{repo}/release-assets/{assetId}", s.releaseHandler.DeleteAsset)

		// Repository webhooks
		r.Get("/repos/{owner}/{repo}/hooks", s.webhookHandler.ListByRepo)
		r.Post("/repos/{owner}/{repo}/hooks", s.webhookHandler.CreateForRepo)
		r.Get("/repos/{owner}/{repo}/hooks/{id}", s.webhookHandler.Get)
		r.Patch("/repos/{owner}/{repo}/hooks/{id}", s.webhookHandler.Update)
		r.Delete("/repos/{owner}/{repo}/hooks/{id}", s.webhookHandler.Delete)
		r.Post("/repos/{owner}/{repo}/hooks/{id}/pings", s.webhookHandler.Ping)
		r.Post("/repos/{owner}/{repo}/hooks/{id}/tests", s.webhookHandler.Test)
		r.Get("/repos/{owner}/{repo}/hooks/{id}/deliveries", s.webhookHandler.ListDeliveries)
		r.Get("/repos/{owner}/{repo}/hooks/{id}/deliveries/{did}", s.webhookHandler.GetDelivery)
		r.Post("/repos/{owner}/{repo}/hooks/{id}/deliveries/{did}", s.webhookHandler.Redeliver)

		// Organizations
		r.Get("/orgs", s.orgHandler.List)
		r.Post("/orgs", s.orgHandler.Create)
		r.Get("/orgs/{org}", s.orgHandler.Get)
		r.Patch("/orgs/{org}", s.orgHandler.Update)
		r.Delete("/orgs/{org}", s.orgHandler.Delete)

		// Org members
		r.Get("/orgs/{org}/members", s.orgHandler.ListMembers)
		r.Get("/orgs/{org}/members/{username}", s.orgHandler.GetMember)
		r.Put("/orgs/{org}/members/{username}", s.orgHandler.AddMember)
		r.Patch("/orgs/{org}/members/{username}", s.orgHandler.UpdateMemberRole)
		r.Delete("/orgs/{org}/members/{username}", s.orgHandler.RemoveMember)
		r.Get("/orgs/{org}/membership/{username}", s.orgHandler.CheckMembership)

		// Org webhooks
		r.Get("/orgs/{org}/hooks", s.webhookHandler.ListByOrg)
		r.Post("/orgs/{org}/hooks", s.webhookHandler.CreateForOrg)
		r.Get("/orgs/{org}/hooks/{id}", s.webhookHandler.Get)
		r.Patch("/orgs/{org}/hooks/{id}", s.webhookHandler.Update)
		r.Delete("/orgs/{org}/hooks/{id}", s.webhookHandler.Delete)

		// Teams
		r.Get("/orgs/{org}/teams", s.teamHandler.List)
		r.Post("/orgs/{org}/teams", s.teamHandler.Create)
		r.Get("/orgs/{org}/teams/{team}", s.teamHandler.Get)
		r.Patch("/orgs/{org}/teams/{team}", s.teamHandler.Update)
		r.Delete("/orgs/{org}/teams/{team}", s.teamHandler.Delete)

		// Team members
		r.Get("/orgs/{org}/teams/{team}/members", s.teamHandler.ListMembers)
		r.Get("/orgs/{org}/teams/{team}/members/{username}", s.teamHandler.GetMember)
		r.Put("/orgs/{org}/teams/{team}/members/{username}", s.teamHandler.AddMember)
		r.Patch("/orgs/{org}/teams/{team}/members/{username}", s.teamHandler.UpdateMemberRole)
		r.Delete("/orgs/{org}/teams/{team}/members/{username}", s.teamHandler.RemoveMember)

		// Team repos
		r.Get("/orgs/{org}/teams/{team}/repos", s.teamHandler.ListRepos)
		r.Get("/orgs/{org}/teams/{team}/repos/{owner}/{repo}", s.teamHandler.GetRepoAccess)
		r.Put("/orgs/{org}/teams/{team}/repos/{owner}/{repo}", s.teamHandler.AddRepo)
		r.Patch("/orgs/{org}/teams/{team}/repos/{owner}/{repo}", s.teamHandler.UpdateRepoPermission)
		r.Delete("/orgs/{org}/teams/{team}/repos/{owner}/{repo}", s.teamHandler.RemoveRepo)
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
