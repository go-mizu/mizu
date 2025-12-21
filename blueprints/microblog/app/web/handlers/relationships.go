package handlers

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/relationships"
)

// RelationshipHandlers contains relationship-related handlers.
type RelationshipHandlers struct {
	relationships relationships.API
	getAccountID  func(*mizu.Ctx) string
}

// NewRelationshipHandlers creates new relationship handlers.
func NewRelationshipHandlers(
	relationships relationships.API,
	getAccountID func(*mizu.Ctx) string,
) *RelationshipHandlers {
	return &RelationshipHandlers{
		relationships: relationships,
		getAccountID:  getAccountID,
	}
}

// Follow follows an account.
func (h *RelationshipHandlers) Follow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Follow(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("FOLLOW_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Unfollow unfollows an account.
func (h *RelationshipHandlers) Unfollow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Unfollow(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("UNFOLLOW_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Block blocks an account.
func (h *RelationshipHandlers) Block(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Block(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("BLOCK_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Unblock unblocks an account.
func (h *RelationshipHandlers) Unblock(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Unblock(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("UNBLOCK_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Mute mutes an account.
func (h *RelationshipHandlers) Mute(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Mute(c.Request().Context(), accountID, targetID, true, nil); err != nil {
		return c.JSON(400, ErrorResponse("MUTE_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// Unmute unmutes an account.
func (h *RelationshipHandlers) Unmute(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	if err := h.relationships.Unmute(c.Request().Context(), accountID, targetID); err != nil {
		return c.JSON(400, ErrorResponse("UNMUTE_FAILED", err.Error()))
	}

	rel, _ := h.relationships.Get(c.Request().Context(), accountID, targetID)
	return c.JSON(200, map[string]any{"data": rel})
}

// GetRelationships returns relationships with specified accounts.
func (h *RelationshipHandlers) GetRelationships(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	ids := c.Query("id[]")

	if ids == "" {
		return c.JSON(200, map[string]any{"data": []any{}})
	}

	// Simple implementation - just get one relationship
	rel, _ := h.relationships.Get(c.Request().Context(), accountID, ids)
	return c.JSON(200, map[string]any{"data": []any{rel}})
}
