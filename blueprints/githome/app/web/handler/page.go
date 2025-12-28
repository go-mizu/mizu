package handler

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Page handles HTML page rendering
type Page struct {
	users   users.API
	repos   repos.API
	issues  issues.API
	getUser func(*mizu.Ctx) *users.User
}

// NewPage creates a new page handler
func NewPage(users users.API, repos repos.API, issues issues.API, getUser func(*mizu.Ctx) *users.User) *Page {
	return &Page{
		users:   users,
		repos:   repos,
		issues:  issues,
		getUser: getUser,
	}
}

// Home renders the home page
func (h *Page) Home(c *mizu.Ctx) error {
	user := h.getUser(c)

	if user != nil {
		// Authenticated: show dashboard
		repoList, _ := h.repos.ListAccessible(c.Context(), user.ID, &repos.ListOpts{Limit: 10})
		return c.JSON(http.StatusOK, map[string]any{
			"page":         "dashboard",
			"user":         user,
			"repositories": repoList,
		})
	}

	// Not authenticated: show public repos
	repoList, _ := h.repos.ListPublic(c.Context(), &repos.ListOpts{Limit: 10})
	return c.JSON(http.StatusOK, map[string]any{
		"page":         "home",
		"repositories": repoList,
	})
}

// Login renders the login page
func (h *Page) Login(c *mizu.Ctx) error {
	return c.JSON(http.StatusOK, map[string]any{
		"page": "login",
	})
}

// Register renders the register page
func (h *Page) Register(c *mizu.Ctx) error {
	return c.JSON(http.StatusOK, map[string]any{
		"page": "register",
	})
}

// Explore renders the explore page
func (h *Page) Explore(c *mizu.Ctx) error {
	user := h.getUser(c)
	repoList, _ := h.repos.ListPublic(c.Context(), &repos.ListOpts{Limit: 30})

	return c.JSON(http.StatusOK, map[string]any{
		"page":         "explore",
		"user":         user,
		"repositories": repoList,
	})
}

// NewRepo renders the new repository page
func (h *Page) NewRepo(c *mizu.Ctx) error {
	user := h.getUser(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	return c.JSON(http.StatusOK, map[string]any{
		"page": "new_repo",
		"user": user,
	})
}

// UserProfile renders a user's profile page
func (h *Page) UserProfile(c *mizu.Ctx) error {
	username := c.Param("username")
	currentUser := h.getUser(c)

	profileUser, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "User not found",
		})
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

	return c.JSON(http.StatusOK, map[string]any{
		"page":         "user_profile",
		"user":         currentUser,
		"profile":      profileUser,
		"repositories": repoList,
	})
}

// RepoHome renders a repository's home page
func (h *Page) RepoHome(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	user := h.getUser(c)

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	// Check access
	if repo.IsPrivate {
		userID := ""
		if user != nil {
			userID = user.ID
		}
		if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionRead) {
			return c.JSON(http.StatusNotFound, map[string]any{
				"page":  "not_found",
				"error": "Repository not found",
			})
		}
	}

	repo.OwnerName = ownerUser.Username

	// Check if starred
	isStarred := false
	if user != nil {
		isStarred, _ = h.repos.IsStarred(c.Context(), user.ID, repo.ID)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"page":       "repo_home",
		"user":       user,
		"owner":      ownerUser,
		"repository": repo,
		"is_starred": isStarred,
	})
}

// RepoIssues renders a repository's issues page
func (h *Page) RepoIssues(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	user := h.getUser(c)

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
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

	return c.JSON(http.StatusOK, map[string]any{
		"page":       "repo_issues",
		"user":       user,
		"owner":      ownerUser,
		"repository": repo,
		"issues":     issueList,
		"total":      total,
		"state":      state,
	})
}

// IssueView renders a single issue page
func (h *Page) IssueView(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	number, _ := strconv.Atoi(c.Param("number"))
	user := h.getUser(c)

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo.OwnerName = ownerUser.Username

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Issue not found",
		})
	}

	// Get author
	author, _ := h.users.GetByID(c.Context(), issue.AuthorID)

	comments, _ := h.issues.ListComments(c.Context(), issue.ID)

	return c.JSON(http.StatusOK, map[string]any{
		"page":       "issue_view",
		"user":       user,
		"owner":      ownerUser,
		"repository": repo,
		"issue":      issue,
		"author":     author,
		"comments":   comments,
	})
}

// NewIssue renders the new issue page
func (h *Page) NewIssue(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	user := h.getUser(c)

	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo.OwnerName = ownerUser.Username

	return c.JSON(http.StatusOK, map[string]any{
		"page":       "new_issue",
		"user":       user,
		"owner":      ownerUser,
		"repository": repo,
	})
}

// RepoSettings renders the repository settings page
func (h *Page) RepoSettings(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	user := h.getUser(c)

	if user == nil {
		return c.Redirect(http.StatusFound, "/login")
	}

	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", repoName)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"page":  "not_found",
			"error": "Repository not found",
		})
	}

	repo.OwnerName = ownerUser.Username

	// Check admin access
	if !h.repos.CanAccess(c.Context(), repo.ID, user.ID, repos.PermissionAdmin) {
		return c.JSON(http.StatusForbidden, map[string]any{
			"page":  "forbidden",
			"error": "You don't have permission to access this page",
		})
	}

	collabs, _ := h.repos.ListCollaborators(c.Context(), repo.ID)

	return c.JSON(http.StatusOK, map[string]any{
		"page":          "repo_settings",
		"user":          user,
		"owner":         ownerUser,
		"repository":    repo,
		"collaborators": collabs,
	})
}
