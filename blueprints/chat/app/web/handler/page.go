package handler

import (
	"html/template"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
	"github.com/go-mizu/blueprints/chat/feature/servers"
)

// Page handles HTML page rendering.
type Page struct {
	tmpl      *template.Template
	accounts  accounts.API
	servers   servers.API
	getUserID func(*mizu.Ctx) string
	dev       bool
}

// NewPage creates a new Page handler.
func NewPage(
	tmpl *template.Template,
	accounts accounts.API,
	servers servers.API,
	getUserID func(*mizu.Ctx) string,
	dev bool,
) *Page {
	return &Page{
		tmpl:      tmpl,
		accounts:  accounts,
		servers:   servers,
		getUserID: getUserID,
		dev:       dev,
	}
}

// PageData is the base data for all pages.
type PageData struct {
	Title       string
	User        any
	Data        any
	Dev         bool
}

// Home renders the home page.
func (h *Page) Home(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	// If not logged in, show landing page
	if userID == "" {
		return h.render(c, "landing.html", PageData{
			Title: "Chat - Realtime Messaging",
			Dev:   h.dev,
		})
	}

	// Get user
	user, _ := h.accounts.GetByID(c.Request().Context(), userID)

	// Get user's servers
	srvs, _ := h.servers.ListByUser(c.Request().Context(), userID, 100, 0)

	return h.render(c, "app.html", PageData{
		Title: "Chat",
		User:  user,
		Data: map[string]any{
			"servers": srvs,
		},
		Dev: h.dev,
	})
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	return h.render(c, "login.html", PageData{
		Title: "Login - Chat",
		Dev:   h.dev,
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	return h.render(c, "register.html", PageData{
		Title: "Register - Chat",
		Dev:   h.dev,
	})
}

// Explore renders the server discovery page.
func (h *Page) Explore(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	var user any
	if userID != "" {
		user, _ = h.accounts.GetByID(c.Request().Context(), userID)
	}

	srvs, _ := h.servers.ListPublic(c.Request().Context(), 50, 0)

	return h.render(c, "explore.html", PageData{
		Title: "Explore Servers - Chat",
		User:  user,
		Data: map[string]any{
			"servers": srvs,
		},
		Dev: h.dev,
	})
}

// ServerView renders a server view.
func (h *Page) ServerView(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.Redirect(302, "/login")
	}

	serverID := c.Param("server_id")
	channelID := c.Param("channel_id")

	user, _ := h.accounts.GetByID(c.Request().Context(), userID)
	srv, err := h.servers.GetByID(c.Request().Context(), serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}

	// Get user's servers for sidebar
	srvs, _ := h.servers.ListByUser(c.Request().Context(), userID, 100, 0)

	return h.render(c, "app.html", PageData{
		Title: srv.Name + " - Chat",
		User:  user,
		Data: map[string]any{
			"servers":       srvs,
			"currentServer": srv,
			"channelID":     channelID,
		},
		Dev: h.dev,
	})
}

// Settings renders the settings page.
func (h *Page) Settings(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.Redirect(302, "/login")
	}

	user, _ := h.accounts.GetByID(c.Request().Context(), userID)

	return h.render(c, "settings.html", PageData{
		Title: "Settings - Chat",
		User:  user,
		Dev:   h.dev,
	})
}

func (h *Page) render(c *mizu.Ctx, name string, data PageData) error {
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(c.Writer(), name, data)
}
