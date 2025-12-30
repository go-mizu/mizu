package rest

import (
	"context"
	"net/http"

	"github.com/go-mizu/mizu"
)

// GlobalsAPI defines the globals service interface.
type GlobalsAPI interface {
	Get(ctx context.Context, slug string) (map[string]any, error)
	Update(ctx context.Context, slug string, data map[string]any) (map[string]any, error)
}

// Globals handles global REST endpoints.
type Globals struct {
	service GlobalsAPI
}

// NewGlobals creates a new Globals handler.
func NewGlobals(service GlobalsAPI) *Globals {
	return &Globals{service: service}
}

// Get handles GET /api/globals/{slug}
func (h *Globals) Get(c *mizu.Ctx) error {
	slug := c.Param("slug")

	data, err := h.service.Get(c.Context(), slug)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}
	if data == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Errors: []Error{{Message: "Global not found"}},
		})
	}

	return c.JSON(http.StatusOK, data)
}

// Update handles POST /api/globals/{slug}
func (h *Globals) Update(c *mizu.Ctx) error {
	slug := c.Param("slug")

	var data map[string]any
	if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	result, err := h.service.Update(c.Context(), slug, data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}

	return c.JSON(http.StatusOK, DocResponse{
		Doc:     result,
		Message: "Global updated successfully.",
	})
}

// WithSlug methods return handlers with the slug pre-bound

// GetWithSlug returns a Get handler for a specific global.
func (h *Globals) GetWithSlug(slug string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		data, err := h.service.Get(c.Context(), slug)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Errors: []Error{{Message: err.Error()}},
			})
		}
		if data == nil {
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Errors: []Error{{Message: "Global not found"}},
			})
		}

		return c.JSON(http.StatusOK, data)
	}
}

// UpdateWithSlug returns an Update handler for a specific global.
func (h *Globals) UpdateWithSlug(slug string) mizu.Handler {
	return func(c *mizu.Ctx) error {
		var data map[string]any
		if err := c.BindJSON(&data, 10<<20); err != nil { // 10MB limit
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Errors: []Error{{Message: "Invalid JSON body"}},
			})
		}

		result, err := h.service.Update(c.Context(), slug, data)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Errors: []Error{{Message: err.Error()}},
			})
		}

		return c.JSON(http.StatusOK, DocResponse{
			Doc:     result,
			Message: "Global updated successfully.",
		})
	}
}
