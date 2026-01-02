package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/votes"
)

// Vote handles vote endpoints.
type Vote struct {
	votes       votes.API
	getAccountID func(*mizu.Ctx) string
}

// NewVote creates a new vote handler.
func NewVote(votes votes.API, getAccountID func(*mizu.Ctx) string) *Vote {
	return &Vote{votes: votes, getAccountID: getAccountID}
}

// Cast casts a vote.
func (h *Vote) Cast(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in struct {
		TargetType votes.TargetType `json:"target_type"`
		TargetID   string           `json:"target_id"`
		Value      int              `json:"value"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	vote, err := h.votes.Cast(c.Request().Context(), accountID, in.TargetType, in.TargetID, in.Value)
	if err != nil {
		return BadRequest(c, err.Error())
	}
	return Success(c, vote)
}
