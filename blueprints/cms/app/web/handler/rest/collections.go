// Package rest provides REST API handlers.
package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/blueprints/cms/feature/collections"
	"github.com/go-mizu/mizu"
)

// Collections handles collection REST endpoints.
type Collections struct {
	service collections.API
}

// NewCollections creates a new Collections handler.
func NewCollections(service collections.API) *Collections {
	return &Collections{service: service}
}

// Find handles GET /api/{collection}
func (h *Collections) Find(c *mizu.Ctx) error {
	collection := c.Param("collection")

	input := &collections.FindInput{
		Where:  parseWhere(c),
		Sort:   c.Query("sort"),
		Limit:  parseIntParam(c, "limit", 10),
		Page:   parseIntParam(c, "page", 1),
		Depth:  parseIntParam(c, "depth", 1),
		Locale: c.Query("locale"),
	}

	result, err := h.service.Find(c.Context(), collection, input)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

// FindByID handles GET /api/{collection}/{id}
func (h *Collections) FindByID(c *mizu.Ctx) error {
	collection := c.Param("collection")
	id := c.Param("id")

	depth := parseIntParam(c, "depth", 1)
	locale := c.Query("locale")

	doc, err := h.service.FindByID(c.Context(), collection, id, depth, locale)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, doc)
}

// Count handles GET /api/{collection}/count
func (h *Collections) Count(c *mizu.Ctx) error {
	collection := c.Param("collection")
	where := parseWhere(c)

	count, err := h.service.Count(c.Context(), collection, where)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]int{"totalDocs": count})
}

// Create handles POST /api/{collection}
func (h *Collections) Create(c *mizu.Ctx) error {
	collection := c.Param("collection")

	var data map[string]any
	if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	input := &collections.CreateInput{
		Data:  data,
		Depth: parseIntParam(c, "depth", 1),
		Draft: c.Query("draft") == "true",
	}

	doc, err := h.service.Create(c.Context(), collection, input)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusCreated, DocResponse{
		Doc:     doc,
		Message: "Successfully created.",
	})
}

// UpdateByID handles PATCH /api/{collection}/{id}
func (h *Collections) UpdateByID(c *mizu.Ctx) error {
	collection := c.Param("collection")
	id := c.Param("id")

	var data map[string]any
	if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	input := &collections.UpdateInput{
		Data:     data,
		Depth:    parseIntParam(c, "depth", 1),
		Draft:    c.Query("draft") == "true",
		Autosave: c.Query("autosave") == "true",
	}

	doc, err := h.service.UpdateByID(c.Context(), collection, id, input)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, DocResponse{
		Doc:     doc,
		Message: "Successfully updated.",
	})
}

// Update handles PATCH /api/{collection} (bulk update)
func (h *Collections) Update(c *mizu.Ctx) error {
	collection := c.Param("collection")
	where := parseWhere(c)

	var data map[string]any
	if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	input := &collections.UpdateInput{
		Data:  data,
		Depth: parseIntParam(c, "depth", 1),
		Draft: c.Query("draft") == "true",
	}

	docs, err := h.service.Update(c.Context(), collection, where, input)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"docs":    docs,
		"message": "Successfully updated.",
	})
}

// DeleteByID handles DELETE /api/{collection}/{id}
func (h *Collections) DeleteByID(c *mizu.Ctx) error {
	collection := c.Param("collection")
	id := c.Param("id")

	result, err := h.service.DeleteByID(c.Context(), collection, id)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

// Delete handles DELETE /api/{collection} (bulk delete)
func (h *Collections) Delete(c *mizu.Ctx) error {
	collection := c.Param("collection")
	where := parseWhere(c)

	results, err := h.service.Delete(c.Context(), collection, where)
	if err != nil {
		return errorResponse(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"docs":    results,
		"message": "Successfully deleted.",
	})
}

// WithCollection methods return handlers with collection name pre-bound

// FindWithCollection returns a Find handler for a specific collection.
func (h *Collections) FindWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		input := &collections.FindInput{
			Where:  parseWhere(c),
			Sort:   c.Query("sort"),
			Limit:  parseIntParam(c, "limit", 10),
			Page:   parseIntParam(c, "page", 1),
			Depth:  parseIntParam(c, "depth", 1),
			Locale: c.Query("locale"),
		}

		result, err := h.service.Find(c.Context(), collection, input)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

// FindByIDWithCollection returns a FindByID handler for a specific collection.
func (h *Collections) FindByIDWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		id := c.Param("id")
		depth := parseIntParam(c, "depth", 1)
		locale := c.Query("locale")

		doc, err := h.service.FindByID(c.Context(), collection, id, depth, locale)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, doc)
	}
}

// CountWithCollection returns a Count handler for a specific collection.
func (h *Collections) CountWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		where := parseWhere(c)

		count, err := h.service.Count(c.Context(), collection, where)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]int{"totalDocs": count})
	}
}

// CreateWithCollection returns a Create handler for a specific collection.
func (h *Collections) CreateWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var data map[string]any
		if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		input := &collections.CreateInput{
			Data:  data,
			Depth: parseIntParam(c, "depth", 1),
			Draft: c.Query("draft") == "true",
		}

		doc, err := h.service.Create(c.Context(), collection, input)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusCreated, DocResponse{
			Doc:     doc,
			Message: "Successfully created.",
		})
	}
}

// UpdateByIDWithCollection returns an UpdateByID handler for a specific collection.
func (h *Collections) UpdateByIDWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		id := c.Param("id")

		var data map[string]any
		if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		input := &collections.UpdateInput{
			Data:     data,
			Depth:    parseIntParam(c, "depth", 1),
			Draft:    c.Query("draft") == "true",
			Autosave: c.Query("autosave") == "true",
		}

		doc, err := h.service.UpdateByID(c.Context(), collection, id, input)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, DocResponse{
			Doc:     doc,
			Message: "Successfully updated.",
		})
	}
}

// UpdateWithCollection returns an Update handler for a specific collection.
func (h *Collections) UpdateWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		where := parseWhere(c)

		var data map[string]any
		if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		input := &collections.UpdateInput{
			Data:  data,
			Depth: parseIntParam(c, "depth", 1),
			Draft: c.Query("draft") == "true",
		}

		docs, err := h.service.Update(c.Context(), collection, where, input)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]any{
			"docs":    docs,
			"message": "Successfully updated.",
		})
	}
}

// DeleteByIDWithCollection returns a DeleteByID handler for a specific collection.
func (h *Collections) DeleteByIDWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		id := c.Param("id")

		result, err := h.service.DeleteByID(c.Context(), collection, id)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, result)
	}
}

// DeleteWithCollection returns a Delete handler for a specific collection.
func (h *Collections) DeleteWithCollection(collection string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		where := parseWhere(c)

		results, err := h.service.Delete(c.Context(), collection, where)
		if err != nil {
			return errorResponse(c, err)
		}

		return c.JSON(http.StatusOK, map[string]any{
			"docs":    results,
			"message": "Successfully deleted.",
		})
	}
}

// Response types

// DocResponse is a single document response.
type DocResponse struct {
	Doc     map[string]any `json:"doc"`
	Message string         `json:"message,omitempty"`
}

// ErrorResponse is an error response.
type ErrorResponse struct {
	Errors []Error `json:"errors"`
}

// Error represents a single error.
type Error struct {
	Message string         `json:"message"`
	Field   string         `json:"field,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

// Helper functions

func parseIntParam(c *mizu.Ctx, name string, defaultVal int) int {
	val := c.Query(name)
	if val == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return i
}

func parseWhere(c *mizu.Ctx) map[string]any {
	// Parse where[field][operator]=value format
	where := make(map[string]any)

	for key, values := range c.Request().URL.Query() {
		if !strings.HasPrefix(key, "where[") {
			continue
		}

		// Parse where[field][operator] or where[field]
		key = strings.TrimPrefix(key, "where[")
		parts := strings.Split(strings.TrimSuffix(key, "]"), "][")

		if len(parts) == 1 {
			// where[field]=value (equals)
			field := parts[0]
			where[field] = map[string]any{"equals": values[0]}
		} else if len(parts) == 2 {
			// where[field][operator]=value
			field := parts[0]
			operator := parts[1]

			fieldWhere, ok := where[field].(map[string]any)
			if !ok {
				fieldWhere = make(map[string]any)
				where[field] = fieldWhere
			}

			// Handle array values for in/not_in
			if operator == "in" || operator == "not_in" || operator == "all" {
				var arrValues []any
				for _, v := range strings.Split(values[0], ",") {
					arrValues = append(arrValues, strings.TrimSpace(v))
				}
				fieldWhere[operator] = arrValues
			} else {
				fieldWhere[operator] = parseValue(values[0])
			}
		}
	}

	// Handle or/and operators
	// where[or][0][field][operator]=value
	for key, values := range c.Request().URL.Query() {
		if strings.HasPrefix(key, "where[or]") || strings.HasPrefix(key, "where[and]") {
			// Complex nested queries - would need more sophisticated parsing
			// For now, basic support
			_ = values
		}
	}

	return where
}

func parseValue(s string) any {
	// Try to parse as JSON first
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		return v
	}

	// Try as number
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Try as boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}

	return s
}

func errorResponse(c *mizu.Ctx, err error) error {
	switch err {
	case collections.ErrNotFound:
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Errors: []Error{{Message: "Not Found"}},
		})
	case collections.ErrCollectionNotFound:
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Errors: []Error{{Message: "Collection not found"}},
		})
	default:
		if strings.Contains(err.Error(), "validation") {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: err.Error()}},
			})
		}
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}
}
