package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
)

// Comment handles comment endpoints.
type Comment struct {
	comments  *comments.Service
	getUserID func(*mizu.Ctx) string
}

// NewComment creates a new comment handler.
func NewComment(comments *comments.Service, getUserID func(*mizu.Ctx) string) *Comment {
	return &Comment{
		comments:  comments,
		getUserID: getUserID,
	}
}

// CreateCommentInput is the input for creating a comment.
type CreateCommentInput struct {
	StoryID  string `json:"story_id"`
	ParentID string `json:"parent_id,omitempty"`
	Text     string `json:"text"`
}

// Create creates a new comment.
func (h *Comment) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c)
	}

	var in CreateCommentInput
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid input")
	}

	comment, err := h.comments.Create(c.Request().Context(), userID, comments.CreateIn{
		StoryID:  in.StoryID,
		ParentID: in.ParentID,
		Text:     in.Text,
	})
	if err != nil {
		switch err {
		case comments.ErrInvalidText:
			return BadRequest(c, "comment text is required")
		case comments.ErrTooDeep:
			return BadRequest(c, "comment thread too deep")
		default:
			return InternalError(c, err)
		}
	}

	return Created(c, comment)
}
