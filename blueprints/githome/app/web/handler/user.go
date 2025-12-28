package handler

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// User handles user endpoints
type User struct {
	users     users.API
	repos     repos.API
	getUserID func(*mizu.Ctx) string
}

// NewUser creates a new user handler
func NewUser(users users.API, repos repos.API, getUserID func(*mizu.Ctx) string) *User {
	return &User{
		users:     users,
		repos:     repos,
		getUserID: getUserID,
	}
}

// GetCurrent returns the current authenticated user
func (h *User) GetCurrent(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return NotFound(c, "user not found")
	}

	return OK(c, user)
}

// UpdateCurrent updates the current user's profile
func (h *User) UpdateCurrent(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	var in users.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	user, err := h.users.Update(c.Context(), userID, &in)
	if err != nil {
		return InternalError(c, "failed to update user")
	}

	return OK(c, user)
}

// GetByUsername returns a user by username
func (h *User) GetByUsername(c *mizu.Ctx) error {
	username := c.Param("username")
	if username == "" {
		return BadRequest(c, "username is required")
	}

	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	return OK(c, user)
}

// ListRepos lists the current user's repositories
func (h *User) ListRepos(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &repos.ListOpts{
		Type:   c.Query("type"),
		Sort:   c.Query("sort"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	repoList, err := h.repos.ListByOwner(c.Context(), userID, "user", opts)
	if err != nil {
		return InternalError(c, "failed to list repositories")
	}

	return OK(c, repoList)
}

// ListUserRepos lists a user's public repositories
func (h *User) ListUserRepos(c *mizu.Ctx) error {
	username := c.Param("username")
	if username == "" {
		return BadRequest(c, "username is required")
	}

	user, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &repos.ListOpts{
		Type:   c.Query("type"),
		Sort:   c.Query("sort"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	repoList, err := h.repos.ListByOwner(c.Context(), user.ID, "user", opts)
	if err != nil {
		return InternalError(c, "failed to list repositories")
	}

	// Filter out private repos if not the owner
	currentUserID := h.getUserID(c)
	if currentUserID != user.ID {
		var publicRepos []*repos.Repository
		for _, repo := range repoList {
			if !repo.IsPrivate {
				publicRepos = append(publicRepos, repo)
			}
		}
		repoList = publicRepos
	}

	return OK(c, repoList)
}

// ListStarred lists repositories starred by the current user
func (h *User) ListStarred(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &repos.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	repoList, err := h.repos.ListStarred(c.Context(), userID, opts)
	if err != nil {
		return InternalError(c, "failed to list starred repositories")
	}

	return OK(c, repoList)
}
