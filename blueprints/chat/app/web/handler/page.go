package handler

import (
	"html/template"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/chat/feature/accounts"
	"github.com/go-mizu/blueprints/chat/feature/channels"
	"github.com/go-mizu/blueprints/chat/feature/members"
	"github.com/go-mizu/blueprints/chat/feature/messages"
	"github.com/go-mizu/blueprints/chat/feature/servers"
)

// Page handles HTML page rendering.
type Page struct {
	templates map[string]*template.Template
	accounts  accounts.API
	servers   servers.API
	channels  channels.API
	messages  messages.API
	members   members.API
	getUserID func(*mizu.Ctx) string
	dev       bool
}

// NewPage creates a new Page handler.
func NewPage(
	templates map[string]*template.Template,
	accounts accounts.API,
	servers servers.API,
	channels channels.API,
	messages messages.API,
	members members.API,
	getUserID func(*mizu.Ctx) string,
	dev bool,
) *Page {
	return &Page{
		templates: templates,
		accounts:  accounts,
		servers:   servers,
		channels:  channels,
		messages:  messages,
		members:   members,
		getUserID: getUserID,
		dev:       dev,
	}
}

// PageData is the base data for all pages.
type PageData struct {
	Title string
	User  any
	Data  map[string]any
	Dev   bool
}

// Home renders the home page.
func (h *Page) Home(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	// If not logged in, redirect to login
	if userID == "" {
		return c.Redirect(302, "/login")
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
		Data:  map[string]any{},
		Dev:   h.dev,
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	return h.render(c, "register.html", PageData{
		Title: "Register - Chat",
		Data:  map[string]any{},
		Dev:   h.dev,
	})
}

// Explore redirects to home page (explore page was removed).
func (h *Page) Explore(c *mizu.Ctx) error {
	return c.Redirect(302, "/")
}

// ServerView renders a server view with full SSR.
func (h *Page) ServerView(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.Redirect(302, "/login")
	}

	ctx := c.Request().Context()
	serverID := c.Param("server_id")
	channelID := c.Param("channel_id")

	user, _ := h.accounts.GetByID(ctx, userID)
	srv, err := h.servers.GetByID(ctx, serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}

	// Get user's servers for sidebar
	srvs, _ := h.servers.ListByUser(ctx, userID, 100, 0)

	// Get channels for this server
	chs, _ := h.channels.ListByServer(ctx, serverID)

	// Get current channel
	var currentChannel any
	if channelID != "" {
		currentChannel, _ = h.channels.GetByID(ctx, channelID)
	}

	// Get messages for current channel (server-side rendered)
	var msgs []*messages.Message
	if channelID != "" {
		msgs, _ = h.messages.List(ctx, channelID, messages.ListOpts{Limit: 50})
	}

	// Get members for this server
	mems, _ := h.members.List(ctx, serverID, 100, 0)

	// Enrich members with user info and status
	type EnrichedMember struct {
		UserID      string
		Nickname    string
		DisplayName string
		AvatarURL   string
		Status      string
	}
	enrichedMembers := make([]EnrichedMember, 0, len(mems))
	for _, m := range mems {
		u, _ := h.accounts.GetByID(ctx, m.UserID)
		if u != nil {
			enrichedMembers = append(enrichedMembers, EnrichedMember{
				UserID:      m.UserID,
				Nickname:    m.Nickname,
				DisplayName: u.DisplayName,
				AvatarURL:   u.AvatarURL,
				Status:      string(u.Status),
			})
		}
	}

	return h.render(c, "app.html", PageData{
		Title: srv.Name + " - Chat",
		User:  user,
		Data: map[string]any{
			"servers":        srvs,
			"currentServer":  srv,
			"channels":       chs,
			"currentChannel": currentChannel,
			"channelID":      channelID,
			"messages":       msgs,
			"members":        enrichedMembers,
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
		Data:  map[string]any{},
		Dev:   h.dev,
	})
}

func (h *Page) render(c *mizu.Ctx, name string, data PageData) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return InternalError(c, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(c.Writer(), name, data)
}
