package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/localflare/feature/d1"
)

// D1 handles D1 database requests.
type D1 struct {
	svc d1.API
}

// NewD1 creates a new D1 handler.
func NewD1(svc d1.API) *D1 {
	return &D1{svc: svc}
}

// ListDatabases lists all D1 databases.
func (h *D1) ListDatabases(c *mizu.Ctx) error {
	databases, err := h.svc.ListDatabases(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  databases,
	})
}

// CreateDatabase creates a new D1 database.
func (h *D1) CreateDatabase(c *mizu.Ctx) error {
	var input d1.CreateDatabaseIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	db, err := h.svc.CreateDatabase(c.Request().Context(), &input)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  db,
	})
}

// GetDatabase retrieves a database.
func (h *D1) GetDatabase(c *mizu.Ctx) error {
	id := c.Param("id")
	db, err := h.svc.GetDatabase(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Database not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  db,
	})
}

// DeleteDatabase deletes a database.
func (h *D1) DeleteDatabase(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.DeleteDatabase(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// Query executes a SQL query.
func (h *D1) Query(c *mizu.Ctx) error {
	dbID := c.Param("id")

	var input d1.QueryIn
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	result, err := h.svc.Query(c.Request().Context(), dbID, &input)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}
