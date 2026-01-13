package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"

	"github.com/go-mizu/blueprints/localflare/store"
)

// D1 handles D1 database requests.
type D1 struct {
	store store.D1Store
}

// NewD1 creates a new D1 handler.
func NewD1(store store.D1Store) *D1 {
	return &D1{store: store}
}

// ListDatabases lists all D1 databases.
func (h *D1) ListDatabases(c *mizu.Ctx) error {
	databases, err := h.store.ListDatabases(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  databases,
	})
}

// GetDatabase retrieves a database by ID.
func (h *D1) GetDatabase(c *mizu.Ctx) error {
	id := c.Param("id")
	database, err := h.store.GetDatabase(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Database not found"})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  database,
	})
}

// CreateDatabaseInput is the input for creating a D1 database.
type CreateDatabaseInput struct {
	Name string `json:"name"`
}

// CreateDatabase creates a new D1 database.
func (h *D1) CreateDatabase(c *mizu.Ctx) error {
	var input CreateDatabaseInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]string{"error": "Name is required"})
	}

	database := &store.D1Database{
		ID:        ulid.Make().String(),
		Name:      input.Name,
		Version:   "1",
		NumTables: 0,
		FileSize:  0,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateDatabase(c.Request().Context(), database); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"result":  database,
	})
}

// DeleteDatabase deletes a D1 database.
func (h *D1) DeleteDatabase(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.DeleteDatabase(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// QueryInput is the input for executing a query.
type QueryInput struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params"`
}

// Query executes a query on a D1 database.
func (h *D1) Query(c *mizu.Ctx) error {
	id := c.Param("id")

	var input QueryInput
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	if input.SQL == "" {
		return c.JSON(400, map[string]string{"error": "SQL is required"})
	}

	results, err := h.store.Query(c.Request().Context(), id, input.SQL, input.Params)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"result": map[string]interface{}{
			"results": results,
			"meta": map[string]interface{}{
				"duration":     0,
				"changes":      0,
				"last_row_id":  0,
				"rows_read":    len(results),
				"rows_written": 0,
			},
		},
	})
}
