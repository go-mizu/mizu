package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/pulls"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Pull handles pull request endpoints
type Pull struct {
	pulls     pulls.API
	repos     repos.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewPull creates a new pull handler
func NewPull(pulls pulls.API, repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Pull {
	return &Pull{
		pulls:     pulls,
		repos:     repos,
		users:     users,
		getUserID: getUserID,
	}
}

func (h *Pull) getRepo(c *mizu.Ctx) (*repos.Repository, error) {
	owner := c.Param("owner")
	name := c.Param("repo")

	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return nil, repos.ErrNotFound
	}

	return h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
}

// List lists pull requests for a repository
func (h *Pull) List(c *mizu.Ctx) error {
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

	opts := &pulls.ListOpts{
		State:  c.Query("state"),
		Sort:   c.Query("sort"),
		Head:   c.Query("head"),
		Base:   c.Query("base"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	prList, total, err := h.pulls.List(c.Context(), repo.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list pull requests")
	}

	return OKList(c, prList, total, page, perPage)
}

// Create creates a new pull request
func (h *Pull) Create(c *mizu.Ctx) error {
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

	var in pulls.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	pr, err := h.pulls.Create(c.Context(), repo.ID, userID, &in)
	if err != nil {
		switch err {
		case pulls.ErrMissingTitle:
			return BadRequest(c, "pull request title is required")
		default:
			return InternalError(c, "failed to create pull request")
		}
	}

	return Created(c, pr)
}

// Get retrieves a pull request
func (h *Pull) Get(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	return OK(c, pr)
}

// Update updates a pull request
func (h *Pull) Update(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in pulls.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	pr, err = h.pulls.Update(c.Context(), pr.ID, &in)
	if err != nil {
		return InternalError(c, "failed to update pull request")
	}

	return OK(c, pr)
}

// Merge merges a pull request
func (h *Pull) Merge(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		MergeMethod   string `json:"merge_method"`
		CommitMessage string `json:"commit_message"`
	}
	c.BindJSON(&in, 1<<20)

	if in.MergeMethod == "" {
		in.MergeMethod = pulls.MergeMethodMerge
	}

	if err := h.pulls.Merge(c.Context(), pr.ID, userID, in.MergeMethod, in.CommitMessage); err != nil {
		switch err {
		case pulls.ErrAlreadyMerged:
			return Conflict(c, "pull request is already merged")
		case pulls.ErrNotMergeable:
			return Conflict(c, "pull request is not mergeable")
		default:
			return InternalError(c, "failed to merge pull request")
		}
	}

	return OK(c, map[string]bool{"merged": true})
}

// MarkReady marks a draft pull request as ready for review
func (h *Pull) MarkReady(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	if err := h.pulls.MarkReady(c.Context(), pr.ID); err != nil {
		return InternalError(c, "failed to mark pull request as ready")
	}

	pr, _ = h.pulls.GetByID(c.Context(), pr.ID)
	return OK(c, pr)
}

// Close closes a pull request
func (h *Pull) Close(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	if err := h.pulls.Close(c.Context(), pr.ID); err != nil {
		switch err {
		case pulls.ErrAlreadyClosed:
			return Conflict(c, "pull request is already closed")
		default:
			return InternalError(c, "failed to close pull request")
		}
	}

	pr, _ = h.pulls.GetByID(c.Context(), pr.ID)
	return OK(c, pr)
}

// Reopen reopens a closed pull request
func (h *Pull) Reopen(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	if err := h.pulls.Reopen(c.Context(), pr.ID); err != nil {
		switch err {
		case pulls.ErrAlreadyOpen:
			return Conflict(c, "pull request is already open")
		default:
			return InternalError(c, "failed to reopen pull request")
		}
	}

	pr, _ = h.pulls.GetByID(c.Context(), pr.ID)
	return OK(c, pr)
}

// Lock locks a pull request
func (h *Pull) Lock(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check maintain permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionMaintain) {
		return Forbidden(c, "insufficient permissions")
	}

	reason := c.Query("lock_reason")
	if err := h.pulls.Lock(c.Context(), pr.ID, reason); err != nil {
		return InternalError(c, "failed to lock pull request")
	}

	return NoContent(c)
}

// Unlock unlocks a pull request
func (h *Pull) Unlock(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check maintain permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionMaintain) {
		return Forbidden(c, "insufficient permissions")
	}

	if err := h.pulls.Unlock(c.Context(), pr.ID); err != nil {
		return InternalError(c, "failed to unlock pull request")
	}

	return NoContent(c)
}

// ListLabels lists labels for a pull request
func (h *Pull) ListLabels(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	return OK(c, pr.Labels)
}

// AddLabels adds labels to a pull request
func (h *Pull) AddLabels(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
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

	if err := h.pulls.AddLabels(c.Context(), pr.ID, in.Labels); err != nil {
		return InternalError(c, "failed to add labels")
	}

	return OK(c, in.Labels)
}

// RemoveLabel removes a label from a pull request
func (h *Pull) RemoveLabel(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	label := c.Param("label")
	if err := h.pulls.RemoveLabel(c.Context(), pr.ID, label); err != nil {
		return InternalError(c, "failed to remove label")
	}

	return NoContent(c)
}

// AddAssignees adds assignees to a pull request
func (h *Pull) AddAssignees(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
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

	if err := h.pulls.AddAssignees(c.Context(), pr.ID, in.Assignees); err != nil {
		return InternalError(c, "failed to add assignees")
	}

	pr, _ = h.pulls.GetByID(c.Context(), pr.ID)
	return OK(c, pr)
}

// RemoveAssignees removes assignees from a pull request
func (h *Pull) RemoveAssignees(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
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

	if err := h.pulls.RemoveAssignees(c.Context(), pr.ID, in.Assignees); err != nil {
		return InternalError(c, "failed to remove assignees")
	}

	pr, _ = h.pulls.GetByID(c.Context(), pr.ID)
	return OK(c, pr)
}

// RequestReview requests reviewers for a pull request
func (h *Pull) RequestReview(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		Reviewers []string `json:"reviewers"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.pulls.RequestReview(c.Context(), pr.ID, in.Reviewers); err != nil {
		return InternalError(c, "failed to request review")
	}

	pr, _ = h.pulls.GetByID(c.Context(), pr.ID)
	return OK(c, pr)
}

// RemoveReviewRequest removes review requests
func (h *Pull) RemoveReviewRequest(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in struct {
		Reviewers []string `json:"reviewers"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.pulls.RemoveReviewRequest(c.Context(), pr.ID, in.Reviewers); err != nil {
		return InternalError(c, "failed to remove review request")
	}

	return NoContent(c)
}

// ListReviews lists reviews for a pull request
func (h *Pull) ListReviews(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	reviews, err := h.pulls.ListReviews(c.Context(), pr.ID)
	if err != nil {
		return InternalError(c, "failed to list reviews")
	}

	return OK(c, reviews)
}

// CreateReview creates a review for a pull request
func (h *Pull) CreateReview(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	var in pulls.CreateReviewIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	review, err := h.pulls.CreateReview(c.Context(), pr.ID, userID, &in)
	if err != nil {
		return InternalError(c, "failed to create review")
	}

	return Created(c, review)
}

// GetReview retrieves a review
func (h *Pull) GetReview(c *mizu.Ctx) error {
	reviewID := c.Param("id")

	review, err := h.pulls.GetReview(c.Context(), reviewID)
	if err != nil {
		return NotFound(c, "review not found")
	}

	return OK(c, review)
}

// SubmitReview submits a pending review
func (h *Pull) SubmitReview(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	reviewID := c.Param("id")

	var in struct {
		Event string `json:"event"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	review, err := h.pulls.SubmitReview(c.Context(), reviewID, in.Event)
	if err != nil {
		return InternalError(c, "failed to submit review")
	}

	return OK(c, review)
}

// DismissReview dismisses a review
func (h *Pull) DismissReview(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check maintain permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionMaintain) {
		return Forbidden(c, "insufficient permissions")
	}

	reviewID := c.Param("id")

	var in struct {
		Message string `json:"message"`
	}
	c.BindJSON(&in, 1<<20)

	if err := h.pulls.DismissReview(c.Context(), reviewID, in.Message); err != nil {
		return InternalError(c, "failed to dismiss review")
	}

	return NoContent(c)
}

// ListReviewComments lists review comments for a pull request
func (h *Pull) ListReviewComments(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	number, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	comments, err := h.pulls.ListReviewComments(c.Context(), pr.ID)
	if err != nil {
		return InternalError(c, "failed to list review comments")
	}

	return OK(c, comments)
}

// CreateReviewComment creates a review comment
func (h *Pull) CreateReviewComment(c *mizu.Ctx) error {
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
		return BadRequest(c, "invalid pull request number")
	}

	pr, err := h.pulls.GetByNumber(c.Context(), repo.ID, number)
	if err != nil {
		return NotFound(c, "pull request not found")
	}

	var in pulls.CreateReviewCommentIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.pulls.CreateReviewComment(c.Context(), pr.ID, "", userID, &in)
	if err != nil {
		return InternalError(c, "failed to create review comment")
	}

	return Created(c, comment)
}

// UpdateReviewComment updates a review comment
func (h *Pull) UpdateReviewComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	commentID := c.Param("id")

	var in struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.pulls.UpdateReviewComment(c.Context(), commentID, in.Body)
	if err != nil {
		return InternalError(c, "failed to update review comment")
	}

	return OK(c, comment)
}

// DeleteReviewComment deletes a review comment
func (h *Pull) DeleteReviewComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	commentID := c.Param("id")

	if err := h.pulls.DeleteReviewComment(c.Context(), commentID); err != nil {
		return InternalError(c, "failed to delete review comment")
	}

	return NoContent(c)
}
