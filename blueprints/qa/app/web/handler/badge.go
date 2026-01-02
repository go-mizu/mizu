package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/badges"
)

// Badge handles badge endpoints.
type Badge struct {
	badges badges.API
}

// NewBadge creates a new badge handler.
func NewBadge(badges badges.API) *Badge {
	return &Badge{badges: badges}
}

// List lists badges.
func (h *Badge) List(c *mizu.Ctx) error {
	list, err := h.badges.List(c.Request().Context(), 100)
	if err != nil {
		return InternalError(c)
	}
	return Success(c, list)
}
