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

	// If not logged in, show landing page
	if userID == "" {
		return h.Landing(c)
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

// Landing renders the public landing page for unauthenticated users.
func (h *Page) Landing(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	// Get featured public servers
	publicServers, _ := h.servers.ListPublic(ctx, 6, 0)

	return h.render(c, "home.html", PageData{
		Title: "Welcome",
		Data: map[string]any{
			"publicServers": publicServers,
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

// Explore renders the explore public servers page (publicly accessible).
func (h *Page) Explore(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)

	var user any
	var srvs []*servers.Server

	// If logged in, get user data and their servers
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
		srvs, _ = h.servers.ListByUser(ctx, userID, 100, 0)
	}

	// Get public servers
	publicServers, _ := h.servers.ListPublic(ctx, 50, 0)

	return h.render(c, "explore.html", PageData{
		Title: "Explore Communities",
		User:  user,
		Data: map[string]any{
			"servers":       srvs,
			"publicServers": publicServers,
			"isExplorePage": true,
			"isLoggedIn":    userID != "",
		},
		Dev: h.dev,
	})
}

// ServerViewNoChannel handles /channels/{server_id} and redirects to the default channel.
// Public servers can be viewed without authentication (read-only).
func (h *Page) ServerViewNoChannel(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	serverID := c.Param("server_id")

	// Get server first to check if it's public
	srv, err := h.servers.GetByID(ctx, serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}

	// If not logged in and server is not public, redirect to login
	if userID == "" && !srv.IsPublic {
		return c.Redirect(302, "/login")
	}

	// Get channels for this server and find the default one
	chs, _ := h.channels.ListByServer(ctx, serverID)
	if len(chs) > 0 {
		// Redirect to the first channel
		return c.Redirect(302, "/channels/"+serverID+"/"+chs[0].ID)
	}

	// No channels - render empty server view directly
	isLoggedIn := userID != ""
	var user any
	var srvs []*servers.Server
	if isLoggedIn {
		user, _ = h.accounts.GetByID(ctx, userID)
		srvs, _ = h.servers.ListByUser(ctx, userID, 100, 0)
	}

	return h.render(c, "app.html", PageData{
		Title: srv.Name + " - Chat",
		User:  user,
		Data: map[string]any{
			"servers":       srvs,
			"currentServer": srv,
			"channels":      chs,
			"isLoggedIn":    isLoggedIn,
		},
		Dev: h.dev,
	})
}

// ServerView renders a server view with full SSR.
// Public servers can be viewed without authentication (read-only).
func (h *Page) ServerView(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	serverID := c.Param("server_id")
	channelID := c.Param("channel_id")

	// Get server first to check if it's public
	srv, err := h.servers.GetByID(ctx, serverID)
	if err != nil {
		return NotFound(c, "Server not found")
	}

	// If not logged in and server is not public, redirect to login
	if userID == "" && !srv.IsPublic {
		return c.Redirect(302, "/login")
	}

	isLoggedIn := userID != ""

	// Get user info if logged in
	var user any
	var srvs []*servers.Server
	if isLoggedIn {
		user, _ = h.accounts.GetByID(ctx, userID)
		// Get user's servers for sidebar
		srvs, _ = h.servers.ListByUser(ctx, userID, 100, 0)
	}

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
		// Reverse to show oldest first (messages come DESC from DB)
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
		// Populate author info for each message
		for _, msg := range msgs {
			if author, err := h.accounts.GetByID(ctx, msg.AuthorID); err == nil {
				msg.Author = author
			}
		}
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
			"isLoggedIn":     isLoggedIn,
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
