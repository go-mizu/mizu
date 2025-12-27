package handler

import (
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/assets"
	"github.com/go-mizu/blueprints/messaging/feature/accounts"
	"github.com/go-mizu/blueprints/messaging/feature/chats"
	"github.com/go-mizu/blueprints/messaging/feature/messages"
)

// Page handles page rendering.
type Page struct {
	allTemplates map[string]map[string]*template.Template // theme -> page -> template
	accounts     accounts.API
	chats        chats.API
	messages     messages.API
	getUserID    func(*mizu.Ctx) string
	dev          bool
	assetHashes  *assets.AssetHashes
}

// NewPage creates a new Page handler.
func NewPage(templates map[string]*template.Template, accounts accounts.API, chats chats.API, msgs messages.API, getUserID func(*mizu.Ctx) string, dev bool) *Page {
	// Wrap single theme templates in allTemplates format for backwards compatibility
	allTemplates := map[string]map[string]*template.Template{
		"default": templates,
	}
	return &Page{
		allTemplates: allTemplates,
		accounts:     accounts,
		chats:        chats,
		messages:     msgs,
		getUserID:    getUserID,
		dev:          dev,
		assetHashes:  assets.ComputeAssetHashes(),
	}
}

// NewPageWithThemes creates a new Page handler with multiple theme support.
func NewPageWithThemes(allTemplates map[string]map[string]*template.Template, accounts accounts.API, chats chats.API, msgs messages.API, getUserID func(*mizu.Ctx) string, dev bool) *Page {
	return &Page{
		allTemplates: allTemplates,
		accounts:     accounts,
		chats:        chats,
		messages:     msgs,
		getUserID:    getUserID,
		dev:          dev,
		assetHashes:  assets.ComputeAssetHashes(),
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

// getTheme reads the theme preference from the cookie.
func (h *Page) getTheme(c *mizu.Ctx) string {
	cookie, err := c.Request().Cookie("theme")
	if err != nil || cookie.Value == "" {
		return "default"
	}
	// Validate theme exists
	if _, ok := h.allTemplates[cookie.Value]; ok {
		return cookie.Value
	}
	return "default"
}

func (h *Page) render(c *mizu.Ctx, name string, data any) error {
	theme := h.getTheme(c)
	templates, ok := h.allTemplates[theme]
	if !ok {
		templates = h.allTemplates["default"]
	}

	tmpl, ok := templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	// Get CSS hash for current theme (fall back to default if not found)
	cssHash := h.assetHashes.CSS[theme]
	if cssHash == "" {
		cssHash = h.assetHashes.CSS["default"]
	}

	// Wrap data to include asset hashes for cache busting
	viewData := map[string]any{
		"Version":    h.assetHashes.AppJS,
		"CSSVersion": cssHash,
	}
	if data != nil {
		if m, ok := data.(map[string]any); ok {
			for k, v := range m {
				viewData[k] = v
			}
		}
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), viewData)
}
