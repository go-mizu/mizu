package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Issue handles issue endpoints
type Issue struct {
	issues    issues.API
	repos     repos.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewIssue creates a new issue handler
func NewIssue(issues issues.API, repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Issue {
	return &Issue{
		issues:    issues,
		repos:     repos,
		users:     users,
		getUserID: getUserID,
	}
}

func (h *Issue) getRepo(c *mizu.Ctx) (*repos.Repository, error) {
	owner := c.Param("owner")
	name := c.Param("repo")

	// Get owner user
	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return nil, repos.ErrNotFound
	}

	return h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
}

// List lists issues for a repository
func (h *Issue) List(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &issues.ListOpts{
		State:  c.Query("state"),
		Sort:   c.Query("sort"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	issueList, total, err := h.issues.List(c.Context(), repo.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list issues")
	}

	return OKList(c, issueList, total, page, perPage)
}

// Create creates a new issue
func (h *Issue) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in issues.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	issue, err := h.issues.Create(c.Context(), repo.ID, userID, &in)
	if err != nil {
		switch err {
		case issues.ErrMissingTitle:
			return BadRequest(c, "issue title is required")
		default:
			return InternalError(c, "failed to create issue")
		}
	}

	return Created(c, issue)
}

// Get retrieves an issue
func (h *Issue) Get(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	return OK(c, issue)
}

// Update updates an issue
func (h *Issue) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in issues.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	issue, err = h.issues.Update(c.Context(), issue.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update issue")
	}

	return OK(c, issue)
}

// Delete deletes an issue
func (h *Issue) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	if err := h.issues.Delete(c.Context(), issue.ID); err != nil {
		return InternalError(c, "failed to delete issue")
	}

	return NoContent(c)
}

// Close closes an issue
func (h *Issue) Close(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		StateReason string `json:"state_reason"`
	}
	c.BindJSON(&in, 1<<20) // Optional body

	if err := h.issues.Close(c.Context(), issue.ID, userID, in.StateReason); err != nil {
		return InternalError(c, "failed to close issue")
	}

	// Fetch updated issue
	issue, _ = h.issues.GetByID(c.Context(), issue.ID)
	return OK(c, issue)
}

// Reopen reopens an issue
func (h *Issue) Reopen(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	if err := h.issues.Reopen(c.Context(), issue.ID); err != nil {
		return InternalError(c, "failed to reopen issue")
	}

	// Fetch updated issue
	issue, _ = h.issues.GetByID(c.Context(), issue.ID)
	return OK(c, issue)
}

// Lock locks an issue
func (h *Issue) Lock(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check maintain permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionMaintain) {
		return Forbidden(c, "insufficient permissions")
	}

	reason := c.Query("lock_reason")
	if err := h.issues.Lock(c.Context(), issue.ID, reason); err != nil {
		return InternalError(c, "failed to lock issue")
	}

	return NoContent(c)
}

// Unlock unlocks an issue
func (h *Issue) Unlock(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check maintain permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionMaintain) {
		return Forbidden(c, "insufficient permissions")
	}

	if err := h.issues.Unlock(c.Context(), issue.ID); err != nil {
		return InternalError(c, "failed to unlock issue")
	}

	return NoContent(c)
}

// ListLabels lists labels for an issue
func (h *Issue) ListLabels(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	return OK(c, issue.Labels)
}

// AddLabels adds labels to an issue
func (h *Issue) AddLabels(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		Labels []string `json:"labels"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.issues.AddLabels(c.Context(), issue.ID, in.Labels); err != nil {
		return InternalError(c, "failed to add labels")
	}

	return OK(c, in.Labels)
}

// SetLabels replaces all labels on an issue
func (h *Issue) SetLabels(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		Labels []string `json:"labels"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.issues.SetLabels(c.Context(), issue.ID, in.Labels); err != nil {
		return InternalError(c, "failed to set labels")
	}

	return OK(c, in.Labels)
}

// RemoveLabel removes a label from an issue
func (h *Issue) RemoveLabel(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	label := c.Param("label")
	if err := h.issues.RemoveLabel(c.Context(), issue.ID, label); err != nil {
		return InternalError(c, "failed to remove label")
	}

	return NoContent(c)
}

// AddAssignees adds assignees to an issue
func (h *Issue) AddAssignees(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		Assignees []string `json:"assignees"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.issues.AddAssignees(c.Context(), issue.ID, in.Assignees); err != nil {
		return InternalError(c, "failed to add assignees")
	}

	// Fetch updated issue
	issue, _ = h.issues.GetByID(c.Context(), issue.ID)
	return OK(c, issue)
}

// RemoveAssignees removes assignees from an issue
func (h *Issue) RemoveAssignees(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		Assignees []string `json:"assignees"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.issues.RemoveAssignees(c.Context(), issue.ID, in.Assignees); err != nil {
		return InternalError(c, "failed to remove assignees")
	}

	// Fetch updated issue
	issue, _ = h.issues.GetByID(c.Context(), issue.ID)
	return OK(c, issue)
}

// ListComments lists comments for an issue
func (h *Issue) ListComments(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	comments, err := h.issues.ListComments(c.Context(), issue.ID)
	if err != nil {
		return InternalError(c, "failed to list comments")
	}

	return OK(c, comments)
}

// AddComment adds a comment to an issue
func (h *Issue) AddComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	issue, err := h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.issues.AddComment(c.Context(), issue.ID, userID, in.Body)
	if err != nil {
		if err == issues.ErrLocked {
			return Forbidden(c, "issue is locked")
		}
		return InternalError(c, "failed to add comment")
	}

	return Created(c, comment)
}

// UpdateComment updates a comment on an issue
func (h *Issue) UpdateComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	_, err = h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	commentID := c.Param("id")

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.issues.UpdateComment(c.Context(), commentID, in.Body)
	if err != nil {
		return InternalError(c, "failed to update comment")
	}

	return OK(c, comment)
}

// DeleteComment deletes a comment on an issue
func (h *Issue) DeleteComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid issue number")
	}

	_, err = h.issues.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "issue not found")
	}

	// Check admin permission for deleting comments
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	commentID := c.Param("id")

	if err := h.issues.DeleteComment(c.Context(), commentID); err != nil {
		return InternalError(c, "failed to delete comment")
	}

	return NoContent(c)
}
