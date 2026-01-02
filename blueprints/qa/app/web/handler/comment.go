package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/comments"
)

// Comment handles comment endpoints.
type Comment struct {
	comments     comments.API
	getAccountID func(*mizu.Ctx) string
}

// NewComment creates a new comment handler.
func NewComment(comments comments.API, getAccountID func(*mizu.Ctx) string) *Comment {
	return &Comment{comments: comments, getAccountID: getAccountID}
}

// CreateForQuestion creates a comment on a question.
func (h *Comment) CreateForQuestion(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	in.TargetType = comments.TargetQuestion
	in.TargetID = c.Param("id")

	comment, err := h.comments.Create(c.Request().Context(), accountID, in)
	if err != nil {
		return BadRequest(c, err.Error())
	}
	return Created(c, comment)
}

// CreateForAnswer creates a comment on an answer.
func (h *Comment) CreateForAnswer(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	in.TargetType = comments.TargetAnswer
	in.TargetID = c.Param("id")

	comment, err := h.comments.Create(c.Request().Context(), accountID, in)
	if err != nil {
		return BadRequest(c, err.Error())
	}
	return Created(c, comment)
}
