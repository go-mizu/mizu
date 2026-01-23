package api

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/bi/store"
	"github.com/go-mizu/blueprints/bi/store/sqlite"
)

// Questions handles question API endpoints.
type Questions struct {
	store *sqlite.Store
}

// NewQuestions creates a new Questions handler.
func NewQuestions(store *sqlite.Store) *Questions {
	return &Questions{store: store}
}

// List returns all questions.
func (h *Questions) List(c *mizu.Ctx) error {
	questions, err := h.store.Questions().List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, questions)
}

// Create creates a new question.
func (h *Questions) Create(c *mizu.Ctx) error {
	var q store.Question
	if err := c.BindJSON(&q, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	if err := h.store.Questions().Create(c.Request().Context(), &q); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, q)
}

// Get returns a question by ID.
func (h *Questions) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	q, err := h.store.Questions().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if q == nil {
		return c.JSON(404, map[string]string{"error": "Question not found"})
	}
	return c.JSON(200, q)
}

// Update updates a question.
func (h *Questions) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	var q store.Question
	if err := c.BindJSON(&q, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request body"})
	}
	q.ID = id

	if err := h.store.Questions().Update(c.Request().Context(), &q); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, q)
}

// Delete deletes a question.
func (h *Questions) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.store.Questions().Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}

// Execute executes a question query.
func (h *Questions) Execute(c *mizu.Ctx) error {
	id := c.Param("id")
	q, err := h.store.Questions().GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	if q == nil {
		return c.JSON(404, map[string]string{"error": "Question not found"})
	}

	// Get the data source
	ds, err := h.store.DataSources().GetByID(c.Request().Context(), q.DataSourceID)
	if err != nil || ds == nil {
		return c.JSON(500, map[string]string{"error": "Data source not found"})
	}

	var result *store.QueryResult

	// Check if this is a native SQL query
	if q.QueryType == "native" {
		// Extract SQL from query object
		sqlQuery, ok := q.Query["sql"].(string)
		if !ok || sqlQuery == "" {
			return c.JSON(400, map[string]string{"error": "Native query missing SQL"})
		}
		// Extract params if present
		var params []any
		if p, ok := q.Query["params"].([]any); ok {
			params = p
		}
		result, err = executeNativeQuery(ds, sqlQuery, params)
	} else {
		// Execute structured query
		result, err = executeQuery(ds, q.Query)
	}

	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result)
}
