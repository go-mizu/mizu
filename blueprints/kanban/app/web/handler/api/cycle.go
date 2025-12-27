package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/kanban/feature/cycles"
)

// Cycle handles cycle endpoints.
type Cycle struct {
	cycles cycles.API
}

// NewCycle creates a new cycle handler.
func NewCycle(cycles cycles.API) *Cycle {
	return &Cycle{cycles: cycles}
}

// List returns all cycles for a team.
func (h *Cycle) List(c *mizu.Ctx) error {
	teamID := c.Param("teamID")

	list, err := h.cycles.ListByTeam(c.Context(), teamID)
	if err != nil {
		return InternalError(c, "failed to list cycles")
	}

	return OK(c, list)
}

// Create creates a new cycle.
func (h *Cycle) Create(c *mizu.Ctx) error {
	teamID := c.Param("teamID")

	var in cycles.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	cycle, err := h.cycles.Create(c.Context(), teamID, &in)
	if err != nil {
		return InternalError(c, "failed to create cycle")
	}

	return Created(c, cycle)
}

// Get returns a cycle by ID.
func (h *Cycle) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	cycle, err := h.cycles.GetByID(c.Context(), id)
	if err != nil {
		if err == cycles.ErrNotFound {
			return NotFound(c, "cycle not found")
		}
		return InternalError(c, "failed to get cycle")
	}

	return OK(c, cycle)
}

// GetActive returns the active cycle for a team.
func (h *Cycle) GetActive(c *mizu.Ctx) error {
	teamID := c.Param("teamID")

	cycle, err := h.cycles.GetActive(c.Context(), teamID)
	if err != nil {
		if err == cycles.ErrNotFound {
			return NotFound(c, "no active cycle")
		}
		return InternalError(c, "failed to get active cycle")
	}

	return OK(c, cycle)
}

// Update updates a cycle.
func (h *Cycle) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in cycles.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	cycle, err := h.cycles.Update(c.Context(), id, &in)
	if err != nil {
		if err == cycles.ErrNotFound {
			return NotFound(c, "cycle not found")
		}
		return InternalError(c, "failed to update cycle")
	}

	return OK(c, cycle)
}

// UpdateStatus updates a cycle's status.
func (h *Cycle) UpdateStatus(c *mizu.Ctx) error {
	id := c.Param("id")

	var in struct {
		Status string `json:"status"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if err := h.cycles.UpdateStatus(c.Context(), id, in.Status); err != nil {
		if err == cycles.ErrInvalidStatus {
			return BadRequest(c, "invalid status")
		}
		return InternalError(c, "failed to update status")
	}

	return OK(c, map[string]string{"message": "status updated"})
}

// Delete deletes a cycle.
func (h *Cycle) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.cycles.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete cycle")
	}

	return OK(c, map[string]string{"message": "cycle deleted"})
}
