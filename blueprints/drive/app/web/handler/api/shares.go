package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/shares"
)

// Shares handles share endpoints.
type Shares struct {
	shares    shares.API
	getUserID func(*mizu.Ctx) string
}

// NewShares creates a new Shares handler.
func NewShares(shares shares.API, getUserID func(*mizu.Ctx) string) *Shares {
	return &Shares{
		shares:    shares,
		getUserID: getUserID,
	}
}

// List lists shares created by the user.
func (h *Shares) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	sharesList, err := h.shares.ListByOwner(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sharesList)
}

// SharedWithMe lists shares shared with the user.
func (h *Shares) SharedWithMe(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	sharesList, err := h.shares.ListSharedWithMe(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, sharesList)
}

// Create creates a share with another user.
func (h *Shares) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in struct {
		ResourceID   string `json:"resource_id"`
		ResourceType string `json:"resource_type"`
		SharedWithID string `json:"shared_with_id"`
		Permission   string `json:"permission"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	share, err := h.shares.Create(c.Context(), userID, in.ResourceID, in.ResourceType, in.SharedWithID, in.Permission)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, share)
}

// CreateLink creates a link share.
func (h *Shares) CreateLink(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in struct {
		ResourceID   string               `json:"resource_id"`
		ResourceType string               `json:"resource_type"`
		Options      shares.CreateLinkIn  `json:"options"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	share, err := h.shares.CreateLink(c.Context(), userID, in.ResourceID, in.ResourceType, &in.Options)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, share)
}

// Get gets a share by ID.
func (h *Shares) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	share, err := h.shares.GetByID(c.Context(), id)
	if err != nil {
		if err == shares.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "share not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, share)
}

// GetByToken gets a share by link token (public endpoint).
func (h *Shares) GetByToken(c *mizu.Ctx) error {
	token := c.Param("token")

	share, err := h.shares.GetByToken(c.Context(), token)
	if err != nil {
		if err == shares.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "share not found"})
		}
		if err == shares.ErrExpired {
			return c.JSON(http.StatusGone, map[string]string{"error": "share has expired"})
		}
		if err == shares.ErrDownloadLimit {
			return c.JSON(http.StatusGone, map[string]string{"error": "download limit reached"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, share)
}

// Update updates a share.
func (h *Shares) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in shares.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	share, err := h.shares.Update(c.Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, share)
}

// Delete deletes a share.
func (h *Shares) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.shares.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
