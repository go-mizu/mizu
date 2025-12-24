package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/interactions"
	"github.com/go-mizu/blueprints/social/feature/notifications"
	"github.com/go-mizu/blueprints/social/feature/posts"
)

// Interaction handles interaction endpoints.
type Interaction struct {
	interactions  interactions.API
	posts         posts.API
	notifications notifications.API
	getAccountID  func(*mizu.Ctx) string
}

// NewInteraction creates a new interaction handler.
func NewInteraction(interactionsSvc interactions.API, postsSvc posts.API, notificationsSvc notifications.API, getAccountID func(*mizu.Ctx) string) *Interaction {
	return &Interaction{
		interactions:  interactionsSvc,
		posts:         postsSvc,
		notifications: notificationsSvc,
		getAccountID:  getAccountID,
	}
}

// Like handles POST /api/v1/posts/:id/like
func (h *Interaction) Like(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	post, err := h.posts.GetByID(c.Request().Context(), postID)
	if err != nil {
		return NotFound(c, "post")
	}

	if err := h.interactions.Like(c.Request().Context(), accountID, postID); err != nil {
		if err == interactions.ErrAlreadyLiked {
			return Conflict(c, "already liked")
		}
		return InternalError(c, err)
	}

	// Notify post author
	if h.notifications != nil && post.AccountID != accountID {
		_ = h.notifications.NotifyLike(c.Request().Context(), accountID, post.AccountID, postID)
	}

	// Return updated post
	post, _ = h.posts.GetByID(c.Request().Context(), postID)
	_ = h.posts.PopulateAccount(c.Request().Context(), post)
	_ = h.posts.PopulateViewerState(c.Request().Context(), post, accountID)

	return Success(c, post)
}

// Unlike handles DELETE /api/v1/posts/:id/like
func (h *Interaction) Unlike(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Unlike(c.Request().Context(), accountID, postID); err != nil {
		if err == interactions.ErrNotLiked {
			return Conflict(c, "not liked")
		}
		return InternalError(c, err)
	}

	// Return updated post
	post, _ := h.posts.GetByID(c.Request().Context(), postID)
	_ = h.posts.PopulateAccount(c.Request().Context(), post)
	_ = h.posts.PopulateViewerState(c.Request().Context(), post, accountID)

	return Success(c, post)
}

// Repost handles POST /api/v1/posts/:id/repost
func (h *Interaction) Repost(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	post, err := h.posts.GetByID(c.Request().Context(), postID)
	if err != nil {
		return NotFound(c, "post")
	}

	if err := h.interactions.Repost(c.Request().Context(), accountID, postID); err != nil {
		if err == interactions.ErrAlreadyReposted {
			return Conflict(c, "already reposted")
		}
		return InternalError(c, err)
	}

	// Notify post author
	if h.notifications != nil && post.AccountID != accountID {
		_ = h.notifications.NotifyRepost(c.Request().Context(), accountID, post.AccountID, postID)
	}

	// Return updated post
	post, _ = h.posts.GetByID(c.Request().Context(), postID)
	_ = h.posts.PopulateAccount(c.Request().Context(), post)
	_ = h.posts.PopulateViewerState(c.Request().Context(), post, accountID)

	return Success(c, post)
}

// Unrepost handles DELETE /api/v1/posts/:id/repost
func (h *Interaction) Unrepost(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Unrepost(c.Request().Context(), accountID, postID); err != nil {
		if err == interactions.ErrNotReposted {
			return Conflict(c, "not reposted")
		}
		return InternalError(c, err)
	}

	// Return updated post
	post, _ := h.posts.GetByID(c.Request().Context(), postID)
	_ = h.posts.PopulateAccount(c.Request().Context(), post)
	_ = h.posts.PopulateViewerState(c.Request().Context(), post, accountID)

	return Success(c, post)
}

// Bookmark handles POST /api/v1/posts/:id/bookmark
func (h *Interaction) Bookmark(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Bookmark(c.Request().Context(), accountID, postID); err != nil {
		if err == interactions.ErrAlreadyBookmarked {
			return Conflict(c, "already bookmarked")
		}
		return InternalError(c, err)
	}

	// Return updated post
	post, _ := h.posts.GetByID(c.Request().Context(), postID)
	_ = h.posts.PopulateAccount(c.Request().Context(), post)
	_ = h.posts.PopulateViewerState(c.Request().Context(), post, accountID)

	return Success(c, post)
}

// Unbookmark handles DELETE /api/v1/posts/:id/bookmark
func (h *Interaction) Unbookmark(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Unbookmark(c.Request().Context(), accountID, postID); err != nil {
		if err == interactions.ErrNotBookmarked {
			return Conflict(c, "not bookmarked")
		}
		return InternalError(c, err)
	}

	// Return updated post
	post, _ := h.posts.GetByID(c.Request().Context(), postID)
	_ = h.posts.PopulateAccount(c.Request().Context(), post)
	_ = h.posts.PopulateViewerState(c.Request().Context(), post, accountID)

	return Success(c, post)
}

// LikedBy handles GET /api/v1/posts/:id/liked_by
func (h *Interaction) LikedBy(c *mizu.Ctx) error {
	postID := c.Param("id")
	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	accountIDs, err := h.interactions.GetLikedBy(c.Request().Context(), postID, limit, offset)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, map[string]interface{}{
		"account_ids": accountIDs,
	})
}

// RepostedBy handles GET /api/v1/posts/:id/reposted_by
func (h *Interaction) RepostedBy(c *mizu.Ctx) error {
	postID := c.Param("id")
	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	accountIDs, err := h.interactions.GetRepostedBy(c.Request().Context(), postID, limit, offset)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, map[string]interface{}{
		"account_ids": accountIDs,
	})
}
