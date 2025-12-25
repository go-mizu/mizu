package handler

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
)

// Page handles page rendering.
type Page struct {
	templates map[string]*template.Template
	accounts  accounts.API
	chats     chats.API
	messages  messages.API
	getUserID func(*mizu.Ctx) string
	dev       bool
}

// NewPage creates a new Page handler.
func NewPage(templates map[string]*template.Template, accounts accounts.API, chats chats.API, msgs messages.API, getUserID func(*mizu.Ctx) string, dev bool) *Page {
	return &Page{
		templates: templates,
		accounts:  accounts,
		chats:     chats,
		messages:  msgs,
		getUserID: getUserID,
		dev:       dev,
	}
}

// Home renders the home page.
func (h *Page) Home(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}
	return h.render(c, "home", nil)
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}
	return h.render(c, "login", nil)
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}
	return h.render(c, "register", nil)
}

// App renders the main application page.
func (h *Page) App(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	user, _ := h.accounts.GetByID(ctx, userID)
	chatList, _ := h.chats.List(ctx, userID, chats.ListOpts{Limit: 50})

	return h.render(c, "app", map[string]any{
		"User":  user,
		"Chats": chatList,
	})
}

// ChatView renders a specific chat.
func (h *Page) ChatView(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	chatID := c.Param("id")

	user, _ := h.accounts.GetByID(ctx, userID)
	chatList, _ := h.chats.List(ctx, userID, chats.ListOpts{Limit: 50})
	chat, err := h.chats.GetByIDForUser(ctx, chatID, userID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/app", http.StatusFound)
		return nil
	}

	msgs, _ := h.messages.List(ctx, chatID, messages.ListOpts{Limit: 50})

	return h.render(c, "app", map[string]any{
		"User":        user,
		"Chats":       chatList,
		"CurrentChat": chat,
		"Messages":    msgs,
	})
}

// Settings renders the settings page.
func (h *Page) Settings(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/login", http.StatusFound)
		return nil
	}

	ctx := c.Request().Context()
	user, _ := h.accounts.GetByID(ctx, userID)

	return h.render(c, "settings", map[string]any{
		"User": user,
	})
}

func (h *Page) render(c *mizu.Ctx, name string, data any) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}
