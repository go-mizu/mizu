package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// ContactHandler handles contact API endpoints.
type ContactHandler struct {
	store store.Store
}

// NewContactHandler creates a new contact handler.
func NewContactHandler(st store.Store) *ContactHandler {
	return &ContactHandler{store: st}
}

// List returns contacts, optionally filtered by a search query for autocomplete.
func (h *ContactHandler) List(c *mizu.Ctx) error {
	query := c.Query("q")

	contacts, err := h.store.ListContacts(c.Context(), query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list contacts"})
	}

	return c.JSON(http.StatusOK, contacts)
}

// Create creates a new contact.
func (h *ContactHandler) Create(c *mizu.Ctx) error {
	var req struct {
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Email == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email address is required"})
	}

	// Check if contact already exists
	existing, err := h.store.ListContacts(c.Context(), req.Email)
	if err == nil {
		for _, ct := range existing {
			if ct.Email == req.Email {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "contact with this email already exists"})
			}
		}
	}

	contact := &types.Contact{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Name:      req.Name,
		AvatarURL: req.AvatarURL,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.store.CreateContact(c.Context(), contact); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create contact"})
	}

	return c.JSON(http.StatusCreated, contact)
}

// Update updates an existing contact.
func (h *ContactHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "contact id is required"})
	}

	var updates map[string]any
	if err := c.BindJSON(&updates, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Only allow specific fields to be updated
	allowed := map[string]bool{
		"name":       true,
		"email":      true,
		"avatar_url": true,
	}
	filtered := make(map[string]any)
	for k, v := range updates {
		if allowed[k] {
			filtered[k] = v
		}
	}

	if len(filtered) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no valid fields to update"})
	}

	if err := h.store.UpdateContact(c.Context(), id, filtered); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "contact not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "contact updated"})
}

// Delete removes a contact by ID.
func (h *ContactHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "contact id is required"})
	}

	if err := h.store.DeleteContact(c.Context(), id); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "contact not found"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "contact deleted"})
}
