package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/relationships"
)

// Relationship contains relationship-related handlers.
type Relationship struct {
	relationships relationships.API
	getAccountID  func(*mizu.Ctx) string
}

// NewRelationship creates new relationship handlers.
func NewRelationship(
	relationships relationships.API,
	getAccountID func(*mizu.Ctx) string,
) *Relationship {
	return &Relationship{
		relationships: relationships,
		getAccountID:  getAccountID,
	}
}

// Follow follows an account.
func (h *Relationship) Follow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Follow(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("FOLLOW_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Unfollow unfollows an account.
func (h *Relationship) Unfollow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Unfollow(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("UNFOLLOW_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Block blocks an account.
func (h *Relationship) Block(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Block(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("BLOCK_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Unblock unblocks an account.
func (h *Relationship) Unblock(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Unblock(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("UNBLOCK_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Mute mutes an account.
func (h *Relationship) Mute(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Mute(c.Request().Context(), accountID, targetID, true, nil); err != nil {
		return c.JSON(400, ErrorResponse("MUTE_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Unmute unmutes an account.
func (h *Relationship) Unmute(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Unmute(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("UNMUTE_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// GetRelationships returns relationships with specified accounts.
func (h *Relationship) GetRelationships(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	ids := c.Query("id[]")

	if ids == "" {
		return c.JSON(200, map[string]any{"data": []any{}})
	}

	// Simple implementation - just get one relationship
	rel, _ := h.relationships.Get(c.Request().Context(), accountID, ids)
	return c.JSON(200, map[string]any{"data": []any{rel}})
}
