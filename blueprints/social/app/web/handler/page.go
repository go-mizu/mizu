package handler

import (
	"html/template"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/notifications"
	"github.com/go-mizu/blueprints/social/feature/posts"
	"github.com/go-mizu/blueprints/social/feature/relationships"
	"github.com/go-mizu/blueprints/social/feature/timelines"
	"github.com/go-mizu/blueprints/social/feature/trending"
)

// Page handles HTML page endpoints.
type Page struct {
	tmpl          *template.Template
	accounts      accounts.API
	posts         posts.API
	timelines     timelines.API
	relationships relationships.API
	notifications notifications.API
	trending      trending.API
	optionalAuth  func(*mizu.Ctx) string
	dev           bool
}

// NewPage creates a new page handler.
func NewPage(tmpl *template.Template, accountsSvc accounts.API, postsSvc posts.API, timelinesSvc timelines.API, relsSvc relationships.API, notificationsSvc notifications.API, trendingSvc trending.API, optionalAuth func(*mizu.Ctx) string, dev bool) *Page {
	return &Page{
		tmpl:          tmpl,
		accounts:      accountsSvc,
		posts:         postsSvc,
		timelines:     timelinesSvc,
		relationships: relsSvc,
		notifications: notificationsSvc,
		trending:      trendingSvc,
		optionalAuth:  optionalAuth,
		dev:           dev,
	}
}

func (h *Page) render(c *mizu.Ctx, name string, data map[string]interface{}) error {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["Dev"] = h.dev

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(c.Writer(), name, data)
}

// Home handles GET /
func (h *Page) Home(c *mizu.Ctx) error {
	return h.render(c, "home.html", nil)
}

// Login handles GET /login
func (h *Page) Login(c *mizu.Ctx) error {
	return h.render(c, "login.html", nil)
}

// Register handles GET /register
func (h *Page) Register(c *mizu.Ctx) error {
	return h.render(c, "register.html", nil)
}

// Explore handles GET /explore
func (h *Page) Explore(c *mizu.Ctx) error {
	opts := trending.TrendingOpts{Limit: 10}

	tags, _ := h.trending.GetTrendingTags(c.Request().Context(), opts)
	trendingPosts, _ := h.trending.GetTrendingPosts(c.Request().Context(), opts)

	return h.render(c, "explore.html", map[string]interface{}{
		"TrendingTags":  tags,
		"TrendingPosts": trendingPosts,
	})
}

// Search handles GET /search
func (h *Page) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	return h.render(c, "search.html", map[string]interface{}{
		"Query": query,
	})
}

// Notifications handles GET /notifications
func (h *Page) Notifications(c *mizu.Ctx) error {
	return h.render(c, "notifications.html", nil)
}

// Bookmarks handles GET /bookmarks
func (h *Page) Bookmarks(c *mizu.Ctx) error {
	return h.render(c, "bookmarks.html", nil)
}

// Lists handles GET /lists
func (h *Page) Lists(c *mizu.Ctx) error {
	return h.render(c, "lists.html", nil)
}

// ListView handles GET /lists/:id
func (h *Page) ListView(c *mizu.Ctx) error {
	id := c.Param("id")
	return h.render(c, "list.html", map[string]interface{}{
		"ListID": id,
	})
}

// Settings handles GET /settings
func (h *Page) Settings(c *mizu.Ctx) error {
	page := c.Param("page")
	if page == "" {
		page = "profile"
	}
	return h.render(c, "settings.html", map[string]interface{}{
		"Page": page,
	})
}

// Profile handles GET /@:username
func (h *Page) Profile(c *mizu.Ctx) error {
	username := c.Param("username")

	account, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return h.render(c, "404.html", nil)
	}

	_ = h.accounts.PopulateStats(c.Request().Context(), account)

	return h.render(c, "profile.html", map[string]interface{}{
		"Account": account,
	})
}

// Post handles GET /@:username/post/:id
func (h *Page) Post(c *mizu.Ctx) error {
	id := c.Param("id")

	post, err := h.posts.GetByID(c.Request().Context(), id)
	if err != nil {
		return h.render(c, "404.html", nil)
	}

	_ = h.posts.PopulateAccount(c.Request().Context(), post)

	ctx, _ := h.posts.GetContext(c.Request().Context(), id)
	if ctx != nil {
		_ = h.posts.PopulateAccounts(c.Request().Context(), ctx.Ancestors)
		_ = h.posts.PopulateAccounts(c.Request().Context(), ctx.Descendants)
	}

	return h.render(c, "post.html", map[string]interface{}{
		"Post":    post,
		"Context": ctx,
	})
}

// FollowList handles GET /@:username/followers and /@:username/following
func (h *Page) FollowList(c *mizu.Ctx) error {
	username := c.Param("username")
	path := c.Request().URL.Path

	account, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return h.render(c, "404.html", nil)
	}

	listType := "followers"
	if strings.HasSuffix(path, "/following") {
		listType = "following"
	}

	return h.render(c, "follow_list.html", map[string]interface{}{
		"Account":  account,
		"ListType": listType,
	})
}

// Tag handles GET /tags/:tag
func (h *Page) Tag(c *mizu.Ctx) error {
	tag := c.Param("tag")
	return h.render(c, "tag.html", map[string]interface{}{
		"Tag": tag,
	})
}
