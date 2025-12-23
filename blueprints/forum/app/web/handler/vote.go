package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
)

// Vote handles vote endpoints.
type Vote struct {
	votes        votes.API
	getAccountID func(*mizu.Ctx) string
}

// NewVote creates a new vote handler.
func NewVote(votes votes.API, getAccountID func(*mizu.Ctx) string) *Vote {
	return &Vote{votes: votes, getAccountID: getAccountID}
}

// VoteThread votes on a thread.
func (h *Vote) VoteThread(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in struct {
		Value int `json:"value"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Value != -1 && in.Value != 1 {
		return BadRequest(c, "Value must be -1 or 1")
	}

	if err := h.votes.Vote(c.Request().Context(), accountID, votes.TargetThread, id, in.Value); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Voted"})
}

// UnvoteThread removes a vote from a thread.
func (h *Vote) UnvoteThread(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.votes.Unvote(c.Request().Context(), accountID, votes.TargetThread, id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Unvoted"})
}

// VoteComment votes on a comment.
func (h *Vote) VoteComment(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in struct {
		Value int `json:"value"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	if in.Value != -1 && in.Value != 1 {
		return BadRequest(c, "Value must be -1 or 1")
	}

	if err := h.votes.Vote(c.Request().Context(), accountID, votes.TargetComment, id, in.Value); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Voted"})
}

// UnvoteComment removes a vote from a comment.
func (h *Vote) UnvoteComment(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.votes.Unvote(c.Request().Context(), accountID, votes.TargetComment, id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Unvoted"})
}
