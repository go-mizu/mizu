package api

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/bot/feature/session"
)

// SessionHandler handles session API requests.
type SessionHandler struct {
	svc *session.Service
}

// NewSessionHandler creates a session handler.
func NewSessionHandler(svc *session.Service) *SessionHandler {
	return &SessionHandler{svc: svc}
}

func (h *SessionHandler) List(c *mizu.Ctx) error {
	sessions, err := h.svc.List(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, sessions)
}

func (h *SessionHandler) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	s, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, s)
}

func (h *SessionHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"deleted": id})
}

func (h *SessionHandler) Reset(c *mizu.Ctx) error {
	id := c.Param("id")
	if err := h.svc.Reset(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"reset": id, "message": "Session reset. A new session will be created on next message."})
}
