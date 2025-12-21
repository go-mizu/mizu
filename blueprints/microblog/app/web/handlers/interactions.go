package handlers

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/interactions"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
)

// InteractionHandlers contains interaction-related handlers.
type InteractionHandlers struct {
	interactions interactions.API
	posts        posts.API
	accounts     accounts.API
	getAccountID func(*mizu.Ctx) string
}

// NewInteractionHandlers creates new interaction handlers.
func NewInteractionHandlers(
	interactions interactions.API,
	posts posts.API,
	accounts accounts.API,
	getAccountID func(*mizu.Ctx) string,
) *InteractionHandlers {
	return &InteractionHandlers{
		interactions: interactions,
		posts:        posts,
		accounts:     accounts,
		getAccountID: getAccountID,
	}
}

// Like adds a like to a post.
func (h *InteractionHandlers) Like(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Like(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, ErrorResponse("LIKE_FAILED", err.Error()))
	}

	post, _ := h.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

// Unlike removes a like from a post.
func (h *InteractionHandlers) Unlike(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Unlike(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, ErrorResponse("UNLIKE_FAILED", err.Error()))
	}

	post, _ := h.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

// Repost adds a repost.
func (h *InteractionHandlers) Repost(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Repost(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, ErrorResponse("REPOST_FAILED", err.Error()))
	}

	post, _ := h.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

// Unrepost removes a repost.
func (h *InteractionHandlers) Unrepost(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Unrepost(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, ErrorResponse("UNREPOST_FAILED", err.Error()))
	}

	post, _ := h.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

// Bookmark adds a bookmark.
func (h *InteractionHandlers) Bookmark(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Bookmark(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, ErrorResponse("BOOKMARK_FAILED", err.Error()))
	}

	post, _ := h.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

// Unbookmark removes a bookmark.
func (h *InteractionHandlers) Unbookmark(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	postID := c.Param("id")

	if err := h.interactions.Unbookmark(c.Request().Context(), accountID, postID); err != nil {
		return c.JSON(400, ErrorResponse("UNBOOKMARK_FAILED", err.Error()))
	}

	post, _ := h.posts.GetByID(c.Request().Context(), postID, accountID)
	return c.JSON(200, map[string]any{"data": post})
}

// LikedBy returns accounts that liked a post.
func (h *InteractionHandlers) LikedBy(c *mizu.Ctx) error {
	postID := c.Param("id")
	limit := IntQuery(c, "limit", 40)

	ids, err := h.interactions.GetLikedBy(c.Request().Context(), postID, limit, 0)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := h.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}

// RepostedBy returns accounts that reposted a post.
func (h *InteractionHandlers) RepostedBy(c *mizu.Ctx) error {
	postID := c.Param("id")
	limit := IntQuery(c, "limit", 40)

	ids, err := h.interactions.GetRepostedBy(c.Request().Context(), postID, limit, 0)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := h.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}
