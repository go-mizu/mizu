package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/table/feature/bases"
	"github.com/go-mizu/blueprints/table/feature/tables"
)

// Base handles base endpoints.
type Base struct {
	bases     *bases.Service
	tables    *tables.Service
	getUserID func(*mizu.Ctx) string
}

// NewBase creates a new base handler.
func NewBase(bases *bases.Service, tables *tables.Service, getUserID func(*mizu.Ctx) string) *Base {
	return &Base{bases: bases, tables: tables, getUserID: getUserID}
}

// Create creates a new base.
func (h *Base) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in bases.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	base, err := h.bases.Create(c.Context(), userID, in)
	if err != nil {
		return InternalError(c, "failed to create base")
	}

	return Created(c, map[string]any{"base": base})
}

// Get returns a base by ID with its tables.
func (h *Base) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	base, err := h.bases.GetByID(c.Context(), id)
	if err != nil {
		return NotFound(c, "base not found")
	}

	tableList, _ := h.tables.ListByBase(c.Context(), id)

	return OK(c, map[string]any{
		"base":   base,
		"tables": tableList,
	})
}

// Update updates a base.
func (h *Base) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in bases.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	base, err := h.bases.Update(c.Context(), id, in)
	if err != nil {
		if err == bases.ErrNotFound {
			return NotFound(c, "base not found")
		}
		return InternalError(c, "failed to update base")
	}

	return OK(c, map[string]any{"base": base})
}

// Delete deletes a base.
func (h *Base) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.bases.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete base")
	}

	return NoContent(c)
}

// ListTables returns all tables in a base.
func (h *Base) ListTables(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.tables.ListByBase(c.Context(), id)
	if err != nil {
		return InternalError(c, "failed to list tables")
	}

	return OK(c, map[string]any{"tables": list})
}
