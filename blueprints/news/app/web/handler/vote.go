package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/votes"
)

// Vote handles vote endpoints.
type Vote struct {
	stories   *stories.Service
	comments  *comments.Service
	getUserID func(*mizu.Ctx) string
}

// NewVote creates a new vote handler.
func NewVote(stories *stories.Service, comments *comments.Service, getUserID func(*mizu.Ctx) string) *Vote {
	return &Vote{
		stories:   stories,
		comments:  comments,
		getUserID: getUserID,
	}
}

// VoteStory upvotes a story.
func (h *Vote) VoteStory(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c)
	}

	storyID := c.Param("id")

	err := h.stories.Vote(c.Request().Context(), storyID, userID, 1)
	if err != nil {
		if err == votes.ErrAlreadyVoted {
			return Conflict(c, "already voted")
		}
		return InternalError(c, err)
	}

	return NoContent(c)
}

// UnvoteStory removes a vote from a story.
func (h *Vote) UnvoteStory(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c)
	}

	storyID := c.Param("id")

	err := h.stories.Unvote(c.Request().Context(), storyID, userID)
	if err != nil {
		return InternalError(c, err)
	}

	return NoContent(c)
}

// VoteComment upvotes a comment.
func (h *Vote) VoteComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c)
	}

	commentID := c.Param("id")

	err := h.comments.Vote(c.Request().Context(), commentID, userID, 1)
	if err != nil {
		if err == votes.ErrAlreadyVoted {
			return Conflict(c, "already voted")
		}
		return InternalError(c, err)
	}

	return NoContent(c)
}

// UnvoteComment removes a vote from a comment.
func (h *Vote) UnvoteComment(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c)
	}

	commentID := c.Param("id")

	err := h.comments.Unvote(c.Request().Context(), commentID, userID)
	if err != nil {
		return InternalError(c, err)
	}

	return NoContent(c)
}
