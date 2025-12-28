package handler

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Repo handles repository endpoints
type Repo struct {
	repos     repos.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewRepo creates a new repo handler
func NewRepo(repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Repo {
	return &Repo{
		repos:     repos,
		users:     users,
		getUserID: getUserID,
	}
}

// ListPublic lists public repositories
func (h *Repo) ListPublic(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &repos.ListOpts{
		Sort:   c.Query("sort"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	repoList, err := h.repos.ListPublic(c.Context(), opts)
	if err != nil {
		return InternalError(c, "failed to list repositories")
	}

	return OK(c, repoList)
}

// Create creates a new repository
func (h *Repo) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	var in repos.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	repo, err := h.repos.Create(c.Context(), userID, &in)
	if err != nil {
		switch err {
		case repos.ErrExists:
			return Conflict(c, "repository already exists")
		case repos.ErrMissingName:
			return BadRequest(c, "repository name is required")
		case repos.ErrInvalidInput:
			return BadRequest(c, "invalid repository name")
		default:
			return InternalError(c, "failed to create repository")
		}
	}

	return Created(c, repo)
}

// Get retrieves a repository
func (h *Repo) Get(c *mizu.Ctx) error {
	owner := c.Param("owner")
	name := c.Param("repo")

	if owner == "" || name == "" {
		return BadRequest(c, "owner and repo are required")
	}

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check access for private repos
	if repo.IsPrivate {
		currentUserID := h.getUserID(c)
		if !h.repos.CanAccess(c.Context(), repo.ID, currentUserID, repos.PermissionRead) {
			return NotFound(c, "repository not found")
		}
	}

	repo.OwnerName = user.Username
	return OK(c, repo)
}

// Update updates a repository
func (h *Repo) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	var in repos.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	repo, err = h.repos.Update(c.Context(), repo.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update repository")
	}

	return OK(c, repo)
}

// Delete deletes a repository
func (h *Repo) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Only owner can delete
	if repo.OwnerID != userID {
		return Forbidden(c, "only the owner can delete a repository")
	}

	if err := h.repos.Delete(c.Context(), repo.ID); err != nil {
		return InternalError(c, "failed to delete repository")
	}

	return NoContent(c)
}

// Star stars a repository
func (h *Repo) Star(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	if err := h.repos.Star(c.Context(), userID, repo.ID); err != nil {
		return InternalError(c, "failed to star repository")
	}

	return NoContent(c)
}

// Unstar removes a star from a repository
func (h *Repo) Unstar(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	if err := h.repos.Unstar(c.Context(), userID, repo.ID); err != nil {
		return InternalError(c, "failed to unstar repository")
	}

	return NoContent(c)
}

// ListStargazers lists users who starred a repository
func (h *Repo) ListStargazers(c *mizu.Ctx) error {
	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// TODO: List stargazers
	_ = repo
	return OK(c, []any{})
}

// ListCollaborators lists repository collaborators
func (h *Repo) ListCollaborators(c *mizu.Ctx) error {
	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	collabs, err := h.repos.ListCollaborators(c.Context(), repo.ID)
	if err != nil {
		return InternalError(c, "failed to list collaborators")
	}

	return OK(c, collabs)
}

// AddCollaborator adds a collaborator to a repository
func (h *Repo) AddCollaborator(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	owner := c.Param("owner")
	name := c.Param("repo")
	username := c.Param("username")

	// Get owner user
	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	// Get collaborator user
	collabUser, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	// Get permission from query or body
	perm := repos.Permission(c.Query("permission"))
	if perm == "" {
		perm = repos.PermissionWrite
	}

	if err := h.repos.AddCollaborator(c.Context(), repo.ID, collabUser.ID, perm); err != nil {
		return InternalError(c, "failed to add collaborator")
	}

	return NoContent(c)
}

// RemoveCollaborator removes a collaborator from a repository
func (h *Repo) RemoveCollaborator(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	owner := c.Param("owner")
	name := c.Param("repo")
	username := c.Param("username")

	// Get owner user
	ownerUser, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	repo, err := h.repos.GetByOwnerAndName(c.Context(), ownerUser.ID, "user", name)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	// Get collaborator user
	collabUser, err := h.users.GetByUsername(c.Context(), username)
	if err != nil {
		return NotFound(c, "user not found")
	}

	if err := h.repos.RemoveCollaborator(c.Context(), repo.ID, collabUser.ID); err != nil {
		return InternalError(c, "failed to remove collaborator")
	}

	return NoContent(c)
}
