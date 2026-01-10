package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/shares"
	"github.com/go-mizu/blueprints/table/feature/tables"
)

// Share handles share endpoints.
type Share struct {
	shares    *shares.Service
	bases     *bases.Service
	tables    *tables.Service
	getUserID func(*mizu.Ctx) string
}

// NewShare creates a new share handler.
func NewShare(shares *shares.Service, bases *bases.Service, tables *tables.Service, getUserID func(*mizu.Ctx) string) *Share {
	return &Share{shares: shares, bases: bases, tables: tables, getUserID: getUserID}
}

// ListByBase returns all shares for a base.
func (h *Share) ListByBase(c *mizu.Ctx) error {
	baseID := c.Param("baseId")

	list, err := h.shares.ListByBase(c.Context(), baseID)
	if err != nil {
		return InternalError(c, "failed to list shares")
	}

	return OK(c, map[string]any{"shares": list})
}

// CreateShareRequest is the request body for creating a share.
type CreateShareRequest struct {
	BaseID     string `json:"base_id"`
	Type       string `json:"type"`
	Permission string `json:"permission"`
	Email      string `json:"email,omitempty"`
}

// Create creates a new share.
func (h *Share) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var req CreateShareRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	share, err := h.shares.Create(c.Context(), userID, shares.CreateIn{
		BaseID:     req.BaseID,
		Type:       req.Type,
		Permission: req.Permission,
		Email:      req.Email,
	})
	if err != nil {
		return InternalError(c, "failed to create share")
	}

	return Created(c, map[string]any{"share": share})
}

// Delete deletes a share.
func (h *Share) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.shares.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete share")
	}

	return NoContent(c)
}

// GetByToken returns a share by its token.
func (h *Share) GetByToken(c *mizu.Ctx) error {
	token := c.Param("token")

	share, err := h.shares.GetByToken(c.Context(), token)
	if err != nil {
		if err == shares.ErrTokenExpired {
			return BadRequest(c, "share link has expired")
		}
		return NotFound(c, "share not found")
	}

	// Get the associated base and tables
	base, err := h.bases.GetByID(c.Context(), share.BaseID)
	if err != nil {
		return InternalError(c, "failed to get base")
	}

	tableList, _ := h.tables.ListByBase(c.Context(), share.BaseID)

	return OK(c, map[string]any{
		"share":  share,
		"base":   base,
		"tables": tableList,
	})
}
