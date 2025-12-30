package rest

import (
	"context"
	"net/http"

	"github.com/go-mizu/mizu"
)

// PreferencesAPI defines the preferences service interface.
type PreferencesAPI interface {
	Get(ctx context.Context, userID, key string) (any, error)
	Set(ctx context.Context, userID, key string, value any) error
	Delete(ctx context.Context, userID, key string) error
}

// Preferences handles preference endpoints.
type Preferences struct {
	service PreferencesAPI
}

// NewPreferences creates a new Preferences handler.
func NewPreferences(service PreferencesAPI) *Preferences {
	return &Preferences{service: service}
}

// Get handles GET /api/payload-preferences/{key}
func (h *Preferences) Get(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := getUserID(c)

	if userID == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Errors: []Error{{Message: "Unauthorized"}},
		})
	}

	value, err := h.service.Get(c.Context(), userID, key)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"key":   key,
		"value": value,
	})
}

// Set handles POST /api/payload-preferences/{key}
func (h *Preferences) Set(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := getUserID(c)

	if userID == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Errors: []Error{{Message: "Unauthorized"}},
		})
	}

	var input struct {
		Value any `json:"value"`
	}
	if err := c.BindJSON(&input, 10<<20); err != nil { // 10MB limit
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Errors: []Error{{Message: "Invalid JSON body"}},
		})
	}

	if err := h.service.Set(c.Context(), userID, key, input.Value); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"key":     key,
		"value":   input.Value,
		"message": "Preference saved successfully.",
	})
}

// Delete handles DELETE /api/payload-preferences/{key}
func (h *Preferences) Delete(c *mizu.Ctx) error {
	key := c.Param("key")
	userID := getUserID(c)

	if userID == "" {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Errors: []Error{{Message: "Unauthorized"}},
		})
	}

	if err := h.service.Delete(c.Context(), userID, key); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Errors: []Error{{Message: err.Error()}},
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Preference deleted successfully.",
	})
}

func getUserID(c *mizu.Ctx) string {
	user := getUserFromContext(c)
	if user == nil {
		return ""
	}
	if id, ok := user["id"].(string); ok {
		return id
	}
	return ""
}
