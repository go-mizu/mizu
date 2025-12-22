package handler

import (
	"github.com/go-mizu/blueprints/forum/feature/votes"
	"github.com/go-mizu/mizu"
)

// Vote contains vote-related handlers.
type Vote struct {
	votes        votes.API
	getAccountID func(*mizu.Ctx) string
}

// NewVote creates new vote handlers.
func NewVote(
	votes votes.API,
	getAccountID func(*mizu.Ctx) string,
) *Vote {
	return &Vote{
		votes:        votes,
		getAccountID: getAccountID,
	}
}

// VoteThread handles voting on a thread.
func (h *Vote) VoteThread(c *mizu.Ctx) error {
	threadID := c.Param("id")
	accountID := h.getAccountID(c)

	var in struct {
		Value int `json:"value"`
	}
	if err := c.BindJSON(&in, 1<<10); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	if in.Value < -1 || in.Value > 1 {
		return c.JSON(400, ErrorResponse("Vote value must be -1, 0, or 1"))
	}

	if err := h.votes.Vote(c.Request().Context(), accountID, votes.TargetThread, threadID, in.Value); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message":   "Vote recorded",
		"user_vote": in.Value,
	}))
}

// VotePost handles voting on a post.
func (h *Vote) VotePost(c *mizu.Ctx) error {
	postID := c.Param("id")
	accountID := h.getAccountID(c)

	var in struct {
		Value int `json:"value"`
	}
	if err := c.BindJSON(&in, 1<<10); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	if in.Value < -1 || in.Value > 1 {
		return c.JSON(400, ErrorResponse("Vote value must be -1, 0, or 1"))
	}

	if err := h.votes.Vote(c.Request().Context(), accountID, votes.TargetPost, postID, in.Value); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message":   "Vote recorded",
		"user_vote": in.Value,
	}))
}
