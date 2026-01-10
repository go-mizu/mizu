package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/comments"
	"github.com/go-mizu/blueprints/table/feature/users"
)

// Comment handles comment endpoints.
type Comment struct {
	comments  *comments.Service
	users     *users.Service
	getUserID func(*mizu.Ctx) string
}

// NewComment creates a new comment handler.
func NewComment(comments *comments.Service, users *users.Service, getUserID func(*mizu.Ctx) string) *Comment {
	return &Comment{comments: comments, users: users, getUserID: getUserID}
}

// CommentView is the view model for a comment with user info.
type CommentView struct {
	ID         string       `json:"id"`
	RecordID   string       `json:"recordId"`
	ParentID   string       `json:"parentId,omitempty"`
	UserID     string       `json:"userId"`
	User       *users.User  `json:"user,omitempty"`
	Content    string       `json:"content"`
	IsResolved bool         `json:"isResolved"`
	CreatedAt  string       `json:"createdAt"`
	UpdatedAt  string       `json:"updatedAt"`
}

// ListByRecord returns all comments for a record.
func (h *Comment) ListByRecord(c *mizu.Ctx) error {
	recordID := c.Param("recordId")

	list, err := h.comments.ListByRecord(c.Context(), recordID)
	if err != nil {
		return InternalError(c, "failed to list comments")
	}

	// Enrich with user info
	result := make([]*CommentView, 0, len(list))
	for _, comment := range list {
		cv := &CommentView{
			ID:         comment.ID,
			RecordID:   comment.RecordID,
			ParentID:   comment.ParentID,
			UserID:     comment.UserID,
			Content:    comment.Content,
			IsResolved: comment.IsResolved,
			CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if user, err := h.users.GetByID(c.Context(), comment.UserID); err == nil {
			cv.User = user
		}

		result = append(result, cv)
	}

	return OK(c, map[string]any{"comments": result})
}

// CreateCommentRequest is the request body for creating a comment.
type CreateCommentRequest struct {
	RecordID string `json:"record_id"`
	Content  string `json:"content"`
	ParentID string `json:"parent_id,omitempty"`
}

// Create creates a new comment.
func (h *Comment) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var req CreateCommentRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.comments.Create(c.Context(), userID, comments.CreateIn{
		RecordID: req.RecordID,
		ParentID: req.ParentID,
		Content:  req.Content,
	})
	if err != nil {
		return InternalError(c, "failed to create comment")
	}

	cv := &CommentView{
		ID:         comment.ID,
		RecordID:   comment.RecordID,
		ParentID:   comment.ParentID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		IsResolved: comment.IsResolved,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if user, err := h.users.GetByID(c.Context(), userID); err == nil {
		cv.User = user
	}

	return Created(c, map[string]any{"comment": cv})
}

// Get returns a comment by ID.
func (h *Comment) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	comment, err := h.comments.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "comment not found")
	}

	cv := &CommentView{
		ID:         comment.ID,
		RecordID:   comment.RecordID,
		ParentID:   comment.ParentID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		IsResolved: comment.IsResolved,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if user, err := h.users.GetByID(c.Context(), comment.UserID); err == nil {
		cv.User = user
	}

	return OK(c, map[string]any{"comment": cv})
}

// UpdateCommentRequest is the request body for updating a comment.
type UpdateCommentRequest struct {
	Content    *string `json:"content,omitempty"`
	IsResolved *bool   `json:"is_resolved,omitempty"`
}

// Update updates a comment.
func (h *Comment) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var req UpdateCommentRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	var comment *comments.Comment
	var err error

	if req.Content != nil {
		comment, err = h.comments.Update(c.Context(), id, *req.Content)
	} else {
		comment, err = h.comments.GetByID(c.Context(), id)
	}

	if err != nil {
		return NotFound(c, "comment not found")
	}

	if req.IsResolved != nil {
		if *req.IsResolved {
			h.comments.Resolve(c.Context(), id)
			comment.IsResolved = true
		} else {
			h.comments.Unresolve(c.Context(), id)
			comment.IsResolved = false
		}
	}

	cv := &CommentView{
		ID:         comment.ID,
		RecordID:   comment.RecordID,
		ParentID:   comment.ParentID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		IsResolved: comment.IsResolved,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return OK(c, map[string]any{"comment": cv})
}

// Delete deletes a comment.
func (h *Comment) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete comment")
	}

	return NoContent(c)
}

// Resolve marks a comment as resolved.
func (h *Comment) Resolve(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Resolve(c.Context(), id); err != nil {
		return InternalError(c, "failed to resolve comment")
	}

	comment, _ := h.comments.GetByID(c.Context(), id)
	cv := &CommentView{
		ID:         comment.ID,
		RecordID:   comment.RecordID,
		ParentID:   comment.ParentID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		IsResolved: comment.IsResolved,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return OK(c, map[string]any{"comment": cv})
}

// Unresolve marks a comment as unresolved.
func (h *Comment) Unresolve(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Unresolve(c.Context(), id); err != nil {
		return InternalError(c, "failed to unresolve comment")
	}

	comment, _ := h.comments.GetByID(c.Context(), id)
	cv := &CommentView{
		ID:         comment.ID,
		RecordID:   comment.RecordID,
		ParentID:   comment.ParentID,
		UserID:     comment.UserID,
		Content:    comment.Content,
		IsResolved: comment.IsResolved,
		CreatedAt:  comment.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  comment.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return OK(c, map[string]any{"comment": cv})
}
