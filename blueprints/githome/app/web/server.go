package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"github.com/go-mizu/blueprints/githome/app/web/handler"
	"github.com/go-mizu/blueprints/githome/app/web/handler/api"
	"github.com/go-mizu/blueprints/githome/assets"
	"github.com/go-mizu/blueprints/githome/feature/activities"
	"github.com/go-mizu/blueprints/githome/feature/branches"
	"github.com/go-mizu/blueprints/githome/feature/collaborators"
	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/commits"
	"github.com/go-mizu/blueprints/githome/feature/git"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/labels"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/notifications"
	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/reactions"
	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/search"
	"github.com/go-mizu/blueprints/githome/feature/stars"
	"github.com/go-mizu/blueprints/githome/feature/teams"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/watches"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
	"github.com/go-mizu/blueprints/githome/store/duckdb"
	"github.com/go-mizu/mizu"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Config contains server configuration.
type Config struct {
	Addr     string
	DataDir  string
	ReposDir string
	Dev      bool
}

// App wraps the server and database.
type App struct {
	server *Server
	db     *sql.DB
	config Config
}

// New creates a new App with all dependencies wired up.
func New(cfg Config) (*App, error) {
	// Open database
	dbPath := cfg.DataDir + "/githome.db"
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create stores
	usersStore := duckdb.NewUsersStore(db)
	orgsStore := duckdb.NewOrgsStore(db)
	reposStore := duckdb.NewReposStore(db)
	issuesStore := duckdb.NewIssuesStore(db)
	pullsStore := duckdb.NewPullsStore(db)
	labelsStore := duckdb.NewLabelsStore(db)
	milestonesStore := duckdb.NewMilestonesStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	teamsStore := duckdb.NewTeamsStore(db)
	releasesStore := duckdb.NewReleasesStore(db)
	starsStore := duckdb.NewStarsStore(db)
	watchesStore := duckdb.NewWatchesStore(db)
	webhooksStore := duckdb.NewWebhooksStore(db)
	notificationsStore := duckdb.NewNotificationsStore(db)
	reactionsStore := duckdb.NewReactionsStore(db)
	collaboratorsStore := duckdb.NewCollaboratorsStore(db)
	branchesStore := duckdb.NewBranchesStore(db)
	commitsStore := duckdb.NewCommitsStore(db)
	gitStore := duckdb.NewGitStore(db)
	searchStore := duckdb.NewSearchStore(db)
	activitiesStore := duckdb.NewActivitiesStore(db)

	// Base URL
	baseURL := fmt.Sprintf("http://localhost%s", cfg.Addr)

	// Create services
	usersSvc := users.NewService(usersStore, baseURL)
	orgsSvc := orgs.NewService(orgsStore, usersStore, baseURL)
	reposSvc := repos.NewService(reposStore, usersStore, orgsStore, baseURL, cfg.ReposDir)
	issuesSvc := issues.NewService(issuesStore, reposStore, usersStore, orgsStore, collaboratorsStore, baseURL)
	pullsSvc := pulls.NewService(pullsStore, reposStore, usersStore, baseURL, cfg.ReposDir)
	labelsSvc := labels.NewService(labelsStore, reposStore, issuesStore, milestonesStore, baseURL)
	milestonesSvc := milestones.NewService(milestonesStore, reposStore, usersStore, baseURL)
	commentsSvc := comments.NewService(commentsStore, reposStore, issuesStore, usersStore, baseURL)
	teamsSvc := teams.NewService(teamsStore, orgsStore, reposStore, usersStore, baseURL)
	releasesSvc := releases.NewService(releasesStore, reposStore, usersStore, baseURL, cfg.ReposDir)
	starsSvc := stars.NewService(starsStore, reposStore, usersStore, baseURL)
	watchesSvc := watches.NewService(watchesStore, reposStore, usersStore, baseURL)
	webhooksSvc := webhooks.NewService(webhooksStore, reposStore, orgsStore, baseURL)
	notificationsSvc := notifications.NewService(notificationsStore, reposStore, baseURL)
	reactionsSvc := reactions.NewService(reactionsStore, reposStore, issuesStore, commentsStore, baseURL)
	collaboratorsSvc := collaborators.NewService(collaboratorsStore, reposStore, usersStore, baseURL)
	branchesSvc := branches.NewService(branchesStore, reposStore, baseURL, cfg.ReposDir)
	commitsSvc := commits.NewService(commitsStore, reposStore, usersStore, baseURL, cfg.ReposDir)
	gitSvc := git.NewService(gitStore, reposStore, baseURL, cfg.ReposDir)
	searchSvc := search.NewService(searchStore, baseURL)
	activitiesSvc := activities.NewService(activitiesStore, reposStore, orgsStore, usersStore, baseURL)

	services := &Services{
		Users:         usersSvc,
		Orgs:          orgsSvc,
		Repos:         reposSvc,
		Issues:        issuesSvc,
		Pulls:         pullsSvc,
		Labels:        labelsSvc,
		Milestones:    milestonesSvc,
		Comments:      commentsSvc,
		Teams:         teamsSvc,
		Releases:      releasesSvc,
		Stars:         starsSvc,
		Watches:       watchesSvc,
		Webhooks:      webhooksSvc,
		Notifications: notificationsSvc,
		Reactions:     reactionsSvc,
		Collaborators: collaboratorsSvc,
		Branches:      branchesSvc,
		Commits:       commitsSvc,
		Git:           gitSvc,
		Search:        searchSvc,
		Activities:    activitiesSvc,
	}

	// Load templates
	templates, err := assets.Templates()
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	server := NewServer(services, templates)

	// Serve static files with explicit paths to avoid conflicts with /{owner}/{repo} patterns
	// Literal paths always take precedence over wildcard patterns in ServeMux
	staticFS := http.FS(assets.Static())
	serveFile := func(path string) mizu.Handler {
		return func(c *mizu.Ctx) error {
			c.Writer().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			c.Request().URL.Path = path
			http.FileServer(staticFS).ServeHTTP(c.Writer(), c.Request())
			return nil
		}
	}

	server.app.Get("/_assets/css/main.css", serveFile("/css/main.css"))
	server.app.Get("/_assets/js/app.js", serveFile("/js/app.js"))

	return &App{
		server: server,
		db:     db,
		config: cfg,
	}, nil
}

// Run starts the HTTP server.
func (a *App) Run() error {
	return http.ListenAndServe(a.config.Addr, a.server.Handler())
}

// Close closes the database connection.
func (a *App) Close() error {
	return a.db.Close()
}

// Services contains all service dependencies
type Services struct {
	Users         users.API
	Orgs          orgs.API
	Repos         repos.API
	Issues        issues.API
	Pulls         pulls.API
	Labels        labels.API
	Milestones    milestones.API
	Comments      comments.API
	Teams         teams.API
	Releases      releases.API
	Stars         stars.API
	Watches       watches.API
	Webhooks      webhooks.API
	Notifications notifications.API
	Reactions     reactions.API
	Collaborators collaborators.API
	Branches      branches.API
	Commits       commits.API
	Git           git.API
	Search        search.API
	Activities    activities.API
}

// Server represents the HTTP server
type Server struct {
	app       *mizu.App
	services  *Services
	templates map[string]*template.Template
}

// NewServer creates a new server with all routes configured
func NewServer(services *Services, templates map[string]*template.Template) *Server {
	s := &Server{
		app:       mizu.New(),
		services:  services,
		templates: templates,
	}
	s.setupRoutes()
	return s
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.app
}

// App returns the mizu application
func (s *Server) App() *mizu.App {
	return s.app
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Create handlers
	authHandler := api.NewAuthHandler(s.services.Users)
	userHandler := api.NewUserHandler(s.services.Users)
	orgHandler := api.NewOrgHandler(s.services.Orgs)
	repoHandler := api.NewRepoHandler(s.services.Repos)
	issueHandler := api.NewIssueHandler(s.services.Issues, s.services.Repos)
	pullHandler := api.NewPullHandler(s.services.Pulls, s.services.Repos)
	labelHandler := api.NewLabelHandler(s.services.Labels, s.services.Repos)
	milestoneHandler := api.NewMilestoneHandler(s.services.Milestones, s.services.Repos)
	commentHandler := api.NewCommentHandler(s.services.Comments, s.services.Repos)
	teamHandler := api.NewTeamHandler(s.services.Teams)
	releaseHandler := api.NewReleaseHandler(s.services.Releases, s.services.Repos)
	starHandler := api.NewStarHandler(s.services.Stars, s.services.Repos)
	watchHandler := api.NewWatchHandler(s.services.Watches, s.services.Repos)
	webhookHandler := api.NewWebhookHandler(s.services.Webhooks, s.services.Repos)
	notificationHandler := api.NewNotificationHandler(s.services.Notifications, s.services.Repos)
	reactionHandler := api.NewReactionHandler(s.services.Reactions, s.services.Repos)
	collaboratorHandler := api.NewCollaboratorHandler(s.services.Collaborators, s.services.Repos)
	branchHandler := api.NewBranchHandler(s.services.Branches, s.services.Repos)
	commitHandler := api.NewCommitHandler(s.services.Commits, s.services.Repos)
	gitHandler := api.NewGitHandler(s.services.Git, s.services.Repos)
	searchHandler := api.NewSearchHandler(s.services.Search)
	activityHandler := api.NewActivityHandler(s.services.Activities, s.services.Repos)

	// Auth middleware
	requireAuth := api.RequireAuth(s.services.Users)
	optionalAuth := api.OptionalAuth(s.services.Users)

	r := s.app.Router

	// Helper to get user ID from context
	getUserID := func(c *mizu.Ctx) int64 {
		if user, ok := c.Request().Context().Value(api.UserContextKey).(*users.User); ok && user != nil {
			return user.ID
		}
		return 0
	}

	// ==========================================================================
	// Page Routes (HTML)
	// ==========================================================================
	pageHandler := handler.NewPage(
		s.templates,
		s.services.Users,
		s.services.Repos,
		s.services.Issues,
		s.services.Pulls,
		s.services.Comments,
		s.services.Orgs,
		s.services.Notifications,
		s.services.Stars,
		s.services.Watches,
		s.services.Branches,
		s.services.Releases,
		s.services.Labels,
		s.services.Milestones,
		getUserID,
	)

	// Health check endpoint
	r.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Auth pages
	r.Get("/login", pageHandler.Login)
	r.Get("/register", pageHandler.Register)

	// Main pages (with optional auth for user context)
	pages := r.With(optionalAuth)
	pages.Get("/", pageHandler.Home)
	pages.Get("/explore", pageHandler.Explore)
	pages.Get("/notifications", pageHandler.Notifications)
	pages.Get("/new", pageHandler.NewRepo)
	pages.Get("/{owner}", pageHandler.UserProfile)
	pages.Get("/{owner}/{repo}", pageHandler.RepoHome)
	pages.Get("/{owner}/{repo}/tree/{ref}", pageHandler.RepoTree)
	pages.Get("/{owner}/{repo}/tree/{ref}/{path...}", pageHandler.RepoTree)
	pages.Get("/{owner}/{repo}/blob/{ref}/{path...}", pageHandler.RepoBlob)
	pages.Get("/{owner}/{repo}/raw/{ref}/{path...}", pageHandler.RepoRaw)
	pages.Get("/{owner}/{repo}/issues", pageHandler.RepoIssues)
	pages.Get("/{owner}/{repo}/issues/new", pageHandler.NewIssue)
	pages.Get("/{owner}/{repo}/issues/{number}", pageHandler.IssueDetail)
	pages.Get("/{owner}/{repo}/settings", pageHandler.RepoSettings)

	// ==========================================================================
	// API Routes (prefixed with /api/v3 to avoid conflicts with page routes)
	// ==========================================================================
	// Create prefixed routers for API routes
	r.Group("/api/v3", func(api *mizu.Router) {
		apiAuth := api.With(requireAuth)
		apiOptAuth := api.With(optionalAuth)
		_ = apiOptAuth // silence unused warning

		// API: Authentication
		api.Post("/login", authHandler.Login)
		api.Post("/register", authHandler.Register)

		// Users
		apiAuth.Get("/user", userHandler.GetAuthenticatedUser)
		apiAuth.Patch("/user", userHandler.UpdateAuthenticatedUser)
		api.Get("/users", userHandler.ListUsers)
		api.Get("/users/{username}", userHandler.GetUser)
		api.Get("/users/{username}/followers", userHandler.ListFollowers)
		api.Get("/users/{username}/following", userHandler.ListFollowing)
		api.Get("/users/{username}/following/{target_user}", userHandler.CheckFollowing)
		apiAuth.Get("/user/followers", userHandler.ListAuthenticatedUserFollowers)
		apiAuth.Get("/user/following", userHandler.ListAuthenticatedUserFollowing)
		apiAuth.Get("/user/following/{username}", userHandler.CheckAuthenticatedUserFollowing)
		apiAuth.Put("/user/following/{username}", userHandler.FollowUser)
		apiAuth.Delete("/user/following/{username}", userHandler.UnfollowUser)

		// Organizations
		api.Get("/organizations", orgHandler.ListOrgs)
		api.Get("/orgs/{org}", orgHandler.GetOrg)
		apiAuth.Patch("/orgs/{org}", orgHandler.UpdateOrg)
		apiAuth.Get("/user/orgs", orgHandler.ListAuthenticatedUserOrgs)
		api.Get("/users/{username}/orgs", orgHandler.ListUserOrgs)
		api.Get("/orgs/{org}/members", orgHandler.ListOrgMembers)
		api.Get("/orgs/{org}/members/{username}", orgHandler.CheckOrgMember)
		apiAuth.Delete("/orgs/{org}/members/{username}", orgHandler.RemoveOrgMember)
		apiAuth.Get("/orgs/{org}/memberships/{username}", orgHandler.GetOrgMembership)
		apiAuth.Put("/orgs/{org}/memberships/{username}", orgHandler.SetOrgMembership)
		apiAuth.Delete("/orgs/{org}/memberships/{username}", orgHandler.RemoveOrgMembership)
		apiAuth.Get("/orgs/{org}/outside_collaborators", orgHandler.ListOutsideCollaborators)
		api.Get("/orgs/{org}/public_members", orgHandler.ListPublicOrgMembers)
		api.Get("/orgs/{org}/public_members/{username}", orgHandler.CheckPublicOrgMember)
		apiAuth.Put("/orgs/{org}/public_members/{username}", orgHandler.PublicizeMembership)
		apiAuth.Delete("/orgs/{org}/public_members/{username}", orgHandler.ConcealMembership)
		apiAuth.Get("/user/memberships/orgs/{org}", orgHandler.GetAuthenticatedUserOrgMembership)
		apiAuth.Patch("/user/memberships/orgs/{org}", orgHandler.UpdateAuthenticatedUserOrgMembership)

		// Repositories
		api.Get("/repositories", repoHandler.ListPublicRepos)
		apiAuth.Get("/user/repos", repoHandler.ListAuthenticatedUserRepos)
		apiAuth.Post("/user/repos", repoHandler.CreateAuthenticatedUserRepo)
		api.Get("/users/{username}/repos", repoHandler.ListUserRepos)
		api.Get("/orgs/{org}/repos", repoHandler.ListOrgRepos)
		apiAuth.Post("/orgs/{org}/repos", repoHandler.CreateOrgRepo)
		apiOptAuth.Get("/repos/{owner}/{repo}", repoHandler.GetRepo)
		apiAuth.Patch("/repos/{owner}/{repo}", repoHandler.UpdateRepo)
		apiAuth.Delete("/repos/{owner}/{repo}", repoHandler.DeleteRepo)
		api.Get("/repos/{owner}/{repo}/topics", repoHandler.ListRepoTopics)
		apiAuth.Put("/repos/{owner}/{repo}/topics", repoHandler.ReplaceRepoTopics)
		api.Get("/repos/{owner}/{repo}/languages", repoHandler.ListRepoLanguages)
		api.Get("/repos/{owner}/{repo}/contributors", repoHandler.ListRepoContributors)
		api.Get("/repos/{owner}/{repo}/tags", repoHandler.ListRepoTags)
		apiAuth.Post("/repos/{owner}/{repo}/transfer", repoHandler.TransferRepo)
		api.Get("/repos/{owner}/{repo}/readme", repoHandler.GetRepoReadme)
		api.Get("/repos/{owner}/{repo}/contents/{path...}", repoHandler.GetRepoContent)
		apiAuth.Put("/repos/{owner}/{repo}/contents/{path...}", repoHandler.CreateOrUpdateFileContent)
		apiAuth.Delete("/repos/{owner}/{repo}/contents/{path...}", repoHandler.DeleteFileContent)
		apiAuth.Post("/repos/{owner}/{repo}/forks", repoHandler.ForkRepo)
		api.Get("/repos/{owner}/{repo}/forks", repoHandler.ListForks)

		// Issues
		apiAuth.Get("/issues", issueHandler.ListIssues)
		apiAuth.Get("/user/issues", issueHandler.ListAuthenticatedUserIssues)
		apiAuth.Get("/orgs/{org}/issues", issueHandler.ListOrgIssues)
		api.Get("/repos/{owner}/{repo}/issues", issueHandler.ListRepoIssues)
		api.Get("/repos/{owner}/{repo}/issues/{issue_number}", issueHandler.GetIssue)
		apiAuth.Post("/repos/{owner}/{repo}/issues", issueHandler.CreateIssue)
		apiAuth.Patch("/repos/{owner}/{repo}/issues/{issue_number}", issueHandler.UpdateIssue)
		apiAuth.Put("/repos/{owner}/{repo}/issues/{issue_number}/lock", issueHandler.LockIssue)
		apiAuth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/lock", issueHandler.UnlockIssue)
		api.Get("/repos/{owner}/{repo}/assignees", issueHandler.ListIssueAssignees)
		api.Get("/repos/{owner}/{repo}/assignees/{assignee}", issueHandler.CheckAssignee)
		apiAuth.Post("/repos/{owner}/{repo}/issues/{issue_number}/assignees", issueHandler.AddAssignees)
		apiAuth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/assignees", issueHandler.RemoveAssignees)
		api.Get("/repos/{owner}/{repo}/issues/{issue_number}/events", issueHandler.ListIssueEvents)
		api.Get("/repos/{owner}/{repo}/issues/events", issueHandler.ListRepoIssueEvents)
		api.Get("/repos/{owner}/{repo}/issues/{issue_number}/timeline", issueHandler.ListIssueTimeline)

		// Pull Requests
		api.Get("/repos/{owner}/{repo}/pulls", pullHandler.ListPulls)
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}", pullHandler.GetPull)
		apiAuth.Post("/repos/{owner}/{repo}/pulls", pullHandler.CreatePull)
		apiAuth.Patch("/repos/{owner}/{repo}/pulls/{pull_number}", pullHandler.UpdatePull)
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/commits", pullHandler.ListPullCommits)
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/files", pullHandler.ListPullFiles)
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/merge", pullHandler.CheckPullMerged)
		apiAuth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/merge", pullHandler.MergePull)
		apiAuth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/update-branch", pullHandler.UpdatePullBranch)

		// Pull Request Reviews
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/reviews", pullHandler.ListPullReviews)
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.GetPullReview)
		apiAuth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/reviews", pullHandler.CreatePullReview)
		apiAuth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.UpdatePullReview)
		apiAuth.Delete("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}", pullHandler.DeletePullReview)
		apiAuth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/events", pullHandler.SubmitPullReview)
		apiAuth.Put("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/dismissals", pullHandler.DismissPullReview)
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/comments", pullHandler.ListReviewComments)

		// Pull Request Review Comments
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/comments", pullHandler.ListPullReviewComments)
		apiAuth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/comments", pullHandler.CreatePullReviewComment)

		// Requested Reviewers
		api.Get("/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.ListRequestedReviewers)
		apiAuth.Post("/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.RequestReviewers)
		apiAuth.Delete("/repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers", pullHandler.RemoveRequestedReviewers)

		// Labels
		api.Get("/repos/{owner}/{repo}/labels", labelHandler.ListRepoLabels)
		api.Get("/repos/{owner}/{repo}/labels/{name}", labelHandler.GetLabel)
		apiAuth.Post("/repos/{owner}/{repo}/labels", labelHandler.CreateLabel)
		apiAuth.Patch("/repos/{owner}/{repo}/labels/{name}", labelHandler.UpdateLabel)
		apiAuth.Delete("/repos/{owner}/{repo}/labels/{name}", labelHandler.DeleteLabel)
		api.Get("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.ListIssueLabels)
		apiAuth.Post("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.AddIssueLabels)
		apiAuth.Put("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.SetIssueLabels)
		apiAuth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/labels", labelHandler.RemoveAllIssueLabels)
		apiAuth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/labels/{name}", labelHandler.RemoveIssueLabel)
		api.Get("/repos/{owner}/{repo}/milestones/{milestone_number}/labels", labelHandler.ListLabelsForMilestone)

		// Milestones
		api.Get("/repos/{owner}/{repo}/milestones", milestoneHandler.ListMilestones)
		api.Get("/repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.GetMilestone)
		apiAuth.Post("/repos/{owner}/{repo}/milestones", milestoneHandler.CreateMilestone)
		apiAuth.Patch("/repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.UpdateMilestone)
		apiAuth.Delete("/repos/{owner}/{repo}/milestones/{milestone_number}", milestoneHandler.DeleteMilestone)

		// Comments (Issue & Commit)
		api.Get("/repos/{owner}/{repo}/issues/{issue_number}/comments", commentHandler.ListIssueComments)
		apiAuth.Post("/repos/{owner}/{repo}/issues/{issue_number}/comments", commentHandler.CreateIssueComment)
		api.Get("/repos/{owner}/{repo}/issues/comments", commentHandler.ListRepoComments)
		api.Get("/repos/{owner}/{repo}/commits/{commit_sha}/comments", commentHandler.ListCommitComments)
		apiAuth.Post("/repos/{owner}/{repo}/commits/{commit_sha}/comments", commentHandler.CreateCommitComment)
		api.Get("/repos/{owner}/{repo}/comments/{comment_id}", commentHandler.GetCommitComment)
		apiAuth.Patch("/repos/{owner}/{repo}/comments/{comment_id}", commentHandler.UpdateCommitComment)
		apiAuth.Delete("/repos/{owner}/{repo}/comments/{comment_id}", commentHandler.DeleteCommitComment)
		api.Get("/repos/{owner}/{repo}/comments", commentHandler.ListRepoCommitComments)

		// Teams
		api.Get("/orgs/{org}/teams", teamHandler.ListOrgTeams)
		api.Get("/orgs/{org}/teams/{team_slug}", teamHandler.GetOrgTeam)
		apiAuth.Post("/orgs/{org}/teams", teamHandler.CreateTeam)
		apiAuth.Patch("/orgs/{org}/teams/{team_slug}", teamHandler.UpdateTeam)
		apiAuth.Delete("/orgs/{org}/teams/{team_slug}", teamHandler.DeleteTeam)
		api.Get("/orgs/{org}/teams/{team_slug}/members", teamHandler.ListTeamMembers)
		apiAuth.Get("/orgs/{org}/teams/{team_slug}/memberships/{username}", teamHandler.GetTeamMembership)
		apiAuth.Put("/orgs/{org}/teams/{team_slug}/memberships/{username}", teamHandler.AddTeamMember)
		apiAuth.Delete("/orgs/{org}/teams/{team_slug}/memberships/{username}", teamHandler.RemoveTeamMember)
		api.Get("/orgs/{org}/teams/{team_slug}/repos", teamHandler.ListTeamRepos)
		api.Get("/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.CheckTeamRepoPermission)
		apiAuth.Put("/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.AddTeamRepo)
		apiAuth.Delete("/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}", teamHandler.RemoveTeamRepo)
		api.Get("/orgs/{org}/teams/{team_slug}/teams", teamHandler.ListChildTeams)
		apiAuth.Get("/user/teams", teamHandler.ListAuthenticatedUserTeams)

		// Releases
		api.Get("/repos/{owner}/{repo}/releases", releaseHandler.ListReleases)
		api.Get("/repos/{owner}/{repo}/releases/latest", releaseHandler.GetLatestRelease)
		apiAuth.Post("/repos/{owner}/{repo}/releases", releaseHandler.CreateRelease)
		apiAuth.Post("/repos/{owner}/{repo}/releases/generate-notes", releaseHandler.GenerateReleaseNotes)

		// Stars
		api.Get("/repos/{owner}/{repo}/stargazers", starHandler.ListStargazers)
		api.Get("/users/{username}/starred", starHandler.ListStarredRepos)
		apiAuth.Get("/user/starred", starHandler.ListAuthenticatedUserStarredRepos)
		apiAuth.Get("/user/starred/{owner}/{repo}", starHandler.CheckRepoStarred)
		apiAuth.Put("/user/starred/{owner}/{repo}", starHandler.StarRepo)
		apiAuth.Delete("/user/starred/{owner}/{repo}", starHandler.UnstarRepo)

		// Watches (Subscriptions)
		api.Get("/repos/{owner}/{repo}/subscribers", watchHandler.ListWatchers)
		apiAuth.Get("/repos/{owner}/{repo}/subscription", watchHandler.GetSubscription)
		apiAuth.Put("/repos/{owner}/{repo}/subscription", watchHandler.SetSubscription)
		apiAuth.Delete("/repos/{owner}/{repo}/subscription", watchHandler.DeleteSubscription)
		api.Get("/users/{username}/subscriptions", watchHandler.ListWatchedRepos)
		apiAuth.Get("/user/subscriptions", watchHandler.ListAuthenticatedUserWatchedRepos)

		// Webhooks
		apiAuth.Get("/repos/{owner}/{repo}/hooks", webhookHandler.ListRepoWebhooks)
		apiAuth.Get("/repos/{owner}/{repo}/hooks/{hook_id}", webhookHandler.GetRepoWebhook)
		apiAuth.Post("/repos/{owner}/{repo}/hooks", webhookHandler.CreateRepoWebhook)
		apiAuth.Patch("/repos/{owner}/{repo}/hooks/{hook_id}", webhookHandler.UpdateRepoWebhook)
		apiAuth.Delete("/repos/{owner}/{repo}/hooks/{hook_id}", webhookHandler.DeleteRepoWebhook)
		apiAuth.Post("/repos/{owner}/{repo}/hooks/{hook_id}/pings", webhookHandler.PingRepoWebhook)
		apiAuth.Post("/repos/{owner}/{repo}/hooks/{hook_id}/tests", webhookHandler.TestRepoWebhook)
		apiAuth.Get("/repos/{owner}/{repo}/hooks/{hook_id}/deliveries", webhookHandler.ListWebhookDeliveries)
		apiAuth.Get("/repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}", webhookHandler.GetWebhookDelivery)
		apiAuth.Post("/repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}/attempts", webhookHandler.RedeliverWebhook)
		apiAuth.Get("/orgs/{org}/hooks", webhookHandler.ListOrgWebhooks)
		apiAuth.Get("/orgs/{org}/hooks/{hook_id}", webhookHandler.GetOrgWebhook)
		apiAuth.Post("/orgs/{org}/hooks", webhookHandler.CreateOrgWebhook)
		apiAuth.Patch("/orgs/{org}/hooks/{hook_id}", webhookHandler.UpdateOrgWebhook)
		apiAuth.Delete("/orgs/{org}/hooks/{hook_id}", webhookHandler.DeleteOrgWebhook)
		apiAuth.Post("/orgs/{org}/hooks/{hook_id}/pings", webhookHandler.PingOrgWebhook)
		apiAuth.Get("/orgs/{org}/hooks/{hook_id}/deliveries", webhookHandler.ListOrgWebhookDeliveries)
		apiAuth.Get("/orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}", webhookHandler.GetOrgWebhookDelivery)
		apiAuth.Post("/orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}/attempts", webhookHandler.RedeliverOrgWebhook)

		// Notifications
		apiAuth.Get("/notifications", notificationHandler.ListNotifications)
		apiAuth.Put("/notifications", notificationHandler.MarkAllAsRead)
		apiAuth.Get("/notifications/threads/{thread_id}", notificationHandler.GetThread)
		apiAuth.Patch("/notifications/threads/{thread_id}", notificationHandler.MarkThreadAsRead)
		apiAuth.Delete("/notifications/threads/{thread_id}", notificationHandler.MarkThreadAsDone)
		apiAuth.Get("/notifications/threads/{thread_id}/subscription", notificationHandler.GetThreadSubscription)
		apiAuth.Put("/notifications/threads/{thread_id}/subscription", notificationHandler.SetThreadSubscription)
		apiAuth.Delete("/notifications/threads/{thread_id}/subscription", notificationHandler.DeleteThreadSubscription)
		apiAuth.Get("/repos/{owner}/{repo}/notifications", notificationHandler.ListRepoNotifications)
		apiAuth.Put("/repos/{owner}/{repo}/notifications", notificationHandler.MarkRepoNotificationsAsRead)

		// Reactions
		api.Get("/repos/{owner}/{repo}/issues/{issue_number}/reactions", reactionHandler.ListIssueReactions)
		apiAuth.Post("/repos/{owner}/{repo}/issues/{issue_number}/reactions", reactionHandler.CreateIssueReaction)
		apiAuth.Delete("/repos/{owner}/{repo}/issues/{issue_number}/reactions/{reaction_id}", reactionHandler.DeleteIssueReaction)
		api.Get("/repos/{owner}/{repo}/comments/{comment_id}/reactions", reactionHandler.ListCommitCommentReactions)
		apiAuth.Post("/repos/{owner}/{repo}/comments/{comment_id}/reactions", reactionHandler.CreateCommitCommentReaction)
		apiAuth.Delete("/repos/{owner}/{repo}/comments/{comment_id}/reactions/{reaction_id}", reactionHandler.DeleteCommitCommentReaction)

		// Collaborators
		apiAuth.Get("/repos/{owner}/{repo}/collaborators", collaboratorHandler.ListCollaborators)
		api.Get("/repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.CheckCollaborator)
		apiAuth.Put("/repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.AddCollaborator)
		apiAuth.Delete("/repos/{owner}/{repo}/collaborators/{username}", collaboratorHandler.RemoveCollaborator)
		apiAuth.Get("/repos/{owner}/{repo}/collaborators/{username}/permission", collaboratorHandler.GetCollaboratorPermission)
		apiAuth.Get("/repos/{owner}/{repo}/invitations", collaboratorHandler.ListInvitations)
		apiAuth.Patch("/repos/{owner}/{repo}/invitations/{invitation_id}", collaboratorHandler.UpdateInvitation)
		apiAuth.Delete("/repos/{owner}/{repo}/invitations/{invitation_id}", collaboratorHandler.DeleteInvitation)
		apiAuth.Get("/user/repository_invitations", collaboratorHandler.ListUserInvitations)
		apiAuth.Patch("/user/repository_invitations/{invitation_id}", collaboratorHandler.AcceptInvitation)
		apiAuth.Delete("/user/repository_invitations/{invitation_id}", collaboratorHandler.DeclineInvitation)

		// Branches
		api.Get("/repos/{owner}/{repo}/branches", branchHandler.ListBranches)
		api.Get("/repos/{owner}/{repo}/branches/{branch}", branchHandler.GetBranch)
		apiAuth.Post("/repos/{owner}/{repo}/branches/{branch}/rename", branchHandler.RenameBranch)
		apiAuth.Get("/repos/{owner}/{repo}/branches/{branch}/protection", branchHandler.GetBranchProtection)
		apiAuth.Put("/repos/{owner}/{repo}/branches/{branch}/protection", branchHandler.UpdateBranchProtection)
		apiAuth.Delete("/repos/{owner}/{repo}/branches/{branch}/protection", branchHandler.DeleteBranchProtection)
		apiAuth.Get("/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", branchHandler.GetRequiredStatusChecks)
		apiAuth.Patch("/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", branchHandler.UpdateRequiredStatusChecks)
		apiAuth.Delete("/repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks", branchHandler.DeleteRequiredStatusChecks)
		apiAuth.Get("/repos/{owner}/{repo}/branches/{branch}/protection/required_signatures", branchHandler.GetRequiredSignatures)
		apiAuth.Post("/repos/{owner}/{repo}/branches/{branch}/protection/required_signatures", branchHandler.CreateRequiredSignatures)
		apiAuth.Delete("/repos/{owner}/{repo}/branches/{branch}/protection/required_signatures", branchHandler.DeleteRequiredSignatures)

		// Commits
		api.Get("/repos/{owner}/{repo}/commits", commitHandler.ListCommits)
		api.Get("/repos/{owner}/{repo}/commits/{ref}", commitHandler.GetCommit)
		api.Get("/repos/{owner}/{repo}/compare/{basehead}", commitHandler.CompareCommits)
		api.Get("/repos/{owner}/{repo}/commits/{commit_sha}/branches-where-head", commitHandler.ListBranchesForHead)
		api.Get("/repos/{owner}/{repo}/commits/{commit_sha}/pulls", commitHandler.ListPullsForCommit)
		api.Get("/repos/{owner}/{repo}/commits/{ref}/status", commitHandler.GetCombinedStatus)
		api.Get("/repos/{owner}/{repo}/commits/{ref}/statuses", commitHandler.ListStatuses)
		apiAuth.Post("/repos/{owner}/{repo}/statuses/{sha}", commitHandler.CreateStatus)

		// Git Data (Low-level)
		api.Get("/repos/{owner}/{repo}/git/blobs/{file_sha}", gitHandler.GetBlob)
		apiAuth.Post("/repos/{owner}/{repo}/git/blobs", gitHandler.CreateBlob)
		api.Get("/repos/{owner}/{repo}/git/commits/{commit_sha}", gitHandler.GetGitCommit)
		apiAuth.Post("/repos/{owner}/{repo}/git/commits", gitHandler.CreateGitCommit)
		api.Get("/repos/{owner}/{repo}/git/ref/{ref...}", gitHandler.GetRef)
		api.Get("/repos/{owner}/{repo}/git/matching-refs/{ref...}", gitHandler.ListMatchingRefs)
		apiAuth.Post("/repos/{owner}/{repo}/git/refs", gitHandler.CreateRef)
		apiAuth.Patch("/repos/{owner}/{repo}/git/refs/{ref...}", gitHandler.UpdateRef)
		apiAuth.Delete("/repos/{owner}/{repo}/git/refs/{ref...}", gitHandler.DeleteRef)
		api.Get("/repos/{owner}/{repo}/git/trees/{tree_sha}", gitHandler.GetTree)
		apiAuth.Post("/repos/{owner}/{repo}/git/trees", gitHandler.CreateTree)
		api.Get("/repos/{owner}/{repo}/git/tags/{tag_sha}", gitHandler.GetTag)
		apiAuth.Post("/repos/{owner}/{repo}/git/tags", gitHandler.CreateTag)
		api.Get("/repos/{owner}/{repo}/git/tags", gitHandler.ListTags)

		// Search
		api.Get("/search/repositories", searchHandler.SearchRepositories)
		api.Get("/search/code", searchHandler.SearchCode)
		api.Get("/search/commits", searchHandler.SearchCommits)
		api.Get("/search/issues", searchHandler.SearchIssues)
		api.Get("/search/users", searchHandler.SearchUsers)
		api.Get("/search/labels", searchHandler.SearchLabels)
		api.Get("/search/topics", searchHandler.SearchTopics)

		// Activity (Events & Feeds)
		api.Get("/events", activityHandler.ListPublicEvents)
		api.Get("/repos/{owner}/{repo}/events", activityHandler.ListRepoEvents)
		api.Get("/networks/{owner}/{repo}/events", activityHandler.ListRepoNetworkEvents)
		api.Get("/orgs/{org}/events", activityHandler.ListOrgEvents)
		api.Get("/users/{username}/received_events", activityHandler.ListUserReceivedEvents)
		api.Get("/users/{username}/received_events/public", activityHandler.ListUserReceivedPublicEvents)
		api.Get("/users/{username}/events", activityHandler.ListUserEvents)
		api.Get("/users/{username}/events/public", activityHandler.ListUserPublicEvents)
		apiAuth.Get("/users/{username}/events/orgs/{org}", activityHandler.ListUserOrgEvents)
		apiOptAuth.Get("/feeds", activityHandler.ListFeeds)
	})
}
