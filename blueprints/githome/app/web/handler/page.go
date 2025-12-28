package handler

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// pathParts extracts path segments from the request URL
func pathParts(c *mizu.Ctx) []string {
	path := strings.Trim(c.Request().URL.Path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// Page handles HTML page rendering
type Page struct {
	users     users.API
	repos     repos.API
	issues    issues.API
	getUser   func(*mizu.Ctx) *users.User
	templates map[string]*template.Template
}

// NewPage creates a new page handler
func NewPage(users users.API, repos repos.API, issues issues.API, getUser func(*mizu.Ctx) *users.User, templates map[string]*template.Template) *Page {
	return &Page{
		users:     users,
		repos:     repos,
		issues:    issues,
		getUser:   getUser,
		templates: templates,
	}
}

// ============================================
// Template Data Types
// ============================================

// LoginData is data for the login page
type LoginData struct {
	Title string
	Error string
}

// RegisterData is data for the register page
type RegisterData struct {
	Title string
	Error string
}

// HomeData is data for the home page
type HomeData struct {
	Title        string
	User         *users.User
	Repositories []*repos.Repository
}

// ExploreData is data for the explore page
type ExploreData struct {
	Title        string
	User         *users.User
	Repositories []*repos.Repository
	Query        string
}

// NewRepoData is data for the new repository page
type NewRepoData struct {
	Title string
	User  *users.User
	Error string
}

// UserProfileData is data for the user profile page
type UserProfileData struct {
	Title        string
	User         *users.User
	Profile      *users.User
	Repositories []*repos.Repository
	IsOwner      bool
}

// RepoHomeData is data for the repository home page
type RepoHomeData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
	IsStarred  bool
	CanEdit    bool
}

// RepoIssuesData is data for the repository issues page
type RepoIssuesData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
	Issues     []*issues.Issue
	Total      int
	State      string
}

// IssueViewData is data for the issue view page
type IssueViewData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
	Issue      *issues.Issue
	Author     *users.User
	Comments   []*issues.Comment
	CanEdit    bool
}

// NewIssueData is data for the new issue page
type NewIssueData struct {
	Title      string
	User       *users.User
	Owner      *users.User
	Repository *repos.Repository
}

// RepoSettingsData is data for the repository settings page
type RepoSettingsData struct {
	Title         string
	User          *users.User
	Owner         *users.User
	Repository    *repos.Repository
	Collaborators []*repos.Collaborator
}

// NotFoundData is data for the 404 page
type NotFoundData struct {
	Title   string
	User    *users.User
	Message string
}

// ============================================
// Page Handlers
// ============================================

// render executes a template with data
func (h *Page) render(c *mizu.Ctx, name string, data interface{}) error {
	tmpl, ok := h.templates[name]
	if !ok {
		// Fallback to JSON if template not found
		return c.JSON(http.StatusOK, data)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}

// Home renders the home page
func (h *Page) Home(c *mizu.Ctx) error {
	user := h.getUser(c)

	var repoList []*repos.Repository
	if user != nil {
		repoList, _ = h.repos.ListAccessible(c.Context(), user.ID, &repos.ListOpts{Limit: 10})
	} else {
		repoList, _ = h.repos.ListPublic(c.Context(), &repos.ListOpts{Limit: 10})
	}

	data := HomeData{
		Title:        "Home",
		User:         user,
		Repositories: repoList,
	}

	return h.render(c, "home", data)
}

// Login renders the login page
func (h *Page) Login(c *mizu.Ctx) error {
	data := LoginData{
		Title: "Sign in",
	}
	return h.render(c, "login", data)
}

// Register renders the register page
func (h *Page) Register(c *mizu.Ctx) error {
	data := RegisterData{
		Title: "Sign up",
	}
	return h.render(c, "register", data)
}

// Explore renders the explore page
func (h *Page) Explore(c *mizu.Ctx) error {
	user := h.getUser(c)
	repoList, _ := h.repos.ListPublic(c.Context(), &repos.ListOpts{Limit: 30})

	data := ExploreData{
		Title:        "Explore",
		User:         user,
		Repositories: repoList,
		Query:        c.Query("q"),
	}

	return h.render(c, "explore", data)
}

// NewRepo renders the new repository page
func (h *Page) NewRepo(c *mizu.Ctx) error {
	user := h.getUser(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	data := NewRepoData{
		Title: "Create a new repository",
		User:  user,
	}

	return h.render(c, "new_repo", data)
}

// UserProfile renders a user's profile page
func (h *Page) UserProfile(c *mizu.Ctx) error {
	parts := pathParts(c)
	if len(parts) < 1 {
		return h.notFound(c, nil, "User not found")
	}
	username := parts[0]
	currentUser := h.getUser(c)

	profileUser, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		data := NotFoundData{
			Title:   "Not Found",
			User:    currentUser,
			Message: "User not found",
		}
		c.Writer().WriteHeader(http.StatusNotFound)
		return h.render(c, "home", data)
	}

	repoList, _ := h.repos.ListByOwner(c.Context(), profileUser.ID, "user", &repos.ListOpts{Limit: 30})

	// Filter private repos if not the owner
	if currentUser == nil || currentUser.ID != profileUser.ID {
		var publicRepos []*repos.Repository
		for _, r := range repoList {
			if !r.IsPrivate {
				publicRepos = append(publicRepos, r)
			}
		}
		repoList = publicRepos
	}

	isOwner := currentUser != nil && currentUser.ID == profileUser.ID

	data := UserProfileData{
		Title:        profileUser.Username,
		User:         currentUser,
		Profile:      profileUser,
		Repositories: repoList,
		IsOwner:      isOwner,
	}

	return h.render(c, "user_profile", data)
}

// RepoHome renders a repository's home page
func (h *Page) RepoHome(c *mizu.Ctx) error {
	parts := pathParts(c)
	if len(parts) < 2 {
		return h.notFound(c, nil, "Repository not found")
	}
	owner := parts[0]
	repoName := parts[1]
	user := h.getUser(c)

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	// Check access
	if repo.IsPrivate {
		userID := ""
		if user != nil {
			userID = user.ID
		}
		if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionRead) {
			return h.notFound(c, user, "Repository not found")
		}
	}

	repo.OwnerName = ownerUser.Username

	// Check if starred
	isStarred := false
	if user != nil {
		isStarred, _ = h.repos.IsStarred(c.Context(), user.ID, repo.ID)
	}

	// Check if can edit
	canEdit := false
	if user != nil {
		canEdit = h.repos.CanAccess(c.Context(), repo.ID, user.ID, repos.PermissionWrite)
	}

	data := RepoHomeData{
		Title:      repo.Name,
		User:       user,
		Owner:      ownerUser,
		Repository: repo,
		IsStarred:  isStarred,
		CanEdit:    canEdit,
	}

	return h.render(c, "repo_home", data)
}

// RepoIssues renders a repository's issues page
func (h *Page) RepoIssues(c *mizu.Ctx) error {
	parts := pathParts(c)
	if len(parts) < 3 {
		return h.notFound(c, nil, "Repository not found")
	}
	owner := parts[0]
	repoName := parts[1]
	user := h.getUser(c)

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo.OwnerName = ownerUser.Username

	state := c.Query("state")
	if state == "" {
		state = "open"
	}

	issueList, total, _ := h.issues.List(c.Context(), repo.ID, &issues.ListOpts{
		State: state,
		Limit: 30,
	})

	data := RepoIssuesData{
		Title:      "Issues",
		User:       user,
		Owner:      ownerUser,
		Repository: repo,
		Issues:     issueList,
		Total:      total,
		State:      state,
	}

	return h.render(c, "repo_issues", data)
}

// IssueView renders a single issue page
func (h *Page) IssueView(c *mizu.Ctx) error {
	parts := pathParts(c)
	if len(parts) < 4 {
		return h.notFound(c, nil, "Issue not found")
	}
	owner := parts[0]
	repoName := parts[1]
	number, _ := strconv.Atoi(parts[3])
	user := h.getUser(c)

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo.OwnerName = ownerUser.Username

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return h.notFound(c, user, "Issue not found")
	}

	// Get author
	author, _ := h.users.GetByID(c.Context(), issue.AuthorID)

	comments, _ := h.issues.ListComments(c.Context(), issue.ID)

	// Check if can edit
	canEdit := false
	if user != nil {
		canEdit = h.repos.CanAccess(c.Context(), repo.ID, user.ID, repos.PermissionWrite)
	}

	data := IssueViewData{
		Title:      issue.Title,
		User:       user,
		Owner:      ownerUser,
		Repository: repo,
		Issue:      issue,
		Author:     author,
		Comments:   comments,
		CanEdit:    canEdit,
	}

	return h.render(c, "issue_view", data)
}

// NewIssue renders the new issue page
func (h *Page) NewIssue(c *mizu.Ctx) error {
	parts := pathParts(c)
	if len(parts) < 4 {
		return h.notFound(c, nil, "Repository not found")
	}
	owner := parts[0]
	repoName := parts[1]
	user := h.getUser(c)

	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo.OwnerName = ownerUser.Username

	data := NewIssueData{
		Title:      "New Issue",
		User:       user,
		Owner:      ownerUser,
		Repository: repo,
	}

	return h.render(c, "new_issue", data)
}

// RepoSettings renders the repository settings page
func (h *Page) RepoSettings(c *mizu.Ctx) error {
	parts := pathParts(c)
	if len(parts) < 3 {
		return h.notFound(c, nil, "Repository not found")
	}
	owner := parts[0]
	repoName := parts[1]
	user := h.getUser(c)

	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return h.notFound(c, user, "Repository not found")
	}

	repo.OwnerName = ownerUser.Username

	// Check admin access
	if !h.repos.CanAccess(c.Context(), repo.ID, user.ID, repos.PermissionAdmin) {
		return h.notFound(c, user, "You don't have permission to access this page")
	}

	collabs, _ := h.repos.ListCollaborators(c.Context(), repo.ID)

	data := RepoSettingsData{
		Title:         "Settings",
		User:          user,
		Owner:         ownerUser,
		Repository:    repo,
		Collaborators: collabs,
	}

	return h.render(c, "repo_settings", data)
}

// notFound renders a not found response
func (h *Page) notFound(c *mizu.Ctx, user *users.User, message string) error {
	c.Writer().WriteHeader(http.StatusNotFound)

	data := HomeData{
		Title: "Not Found",
		User:  user,
	}

	return h.render(c, "home", data)
}
