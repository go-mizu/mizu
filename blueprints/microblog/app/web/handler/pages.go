package handler

import (
	"html/template"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/notifications"
	"github.com/go-mizu/blueprints/microblog/feature/posts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/timelines"
	"github.com/go-mizu/blueprints/microblog/feature/trending"
)

// Page contains web page handlers.
type Page struct {
	templates     *template.Template
	accounts      accounts.API
	posts         posts.API
	timelines     timelines.API
	relationships relationships.API
	notifications notifications.API
	trending      trending.API
	optionalAuth  func(*mizu.Ctx) string
	dev           bool
}

// NewPage creates new page handlers.
func NewPage(
	templates *template.Template,
	accounts accounts.API,
	posts posts.API,
	timelines timelines.API,
	relationships relationships.API,
	notifications notifications.API,
	trending trending.API,
	optionalAuth func(*mizu.Ctx) string,
	dev bool,
) *Page {
	return &Page{
		templates:     templates,
		accounts:      accounts,
		posts:         posts,
		timelines:     timelines,
		relationships: relationships,
		notifications: notifications,
		trending:      trending,
		optionalAuth:  optionalAuth,
		dev:           dev,
	}
}

func (h *Page) render(c *mizu.Ctx, name string, data map[string]any) error {
	if data == nil {
		data = make(map[string]any)
	}
	data["Dev"] = h.dev

	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(c.Writer(), name, data)
}

// Home renders the home page.
func (h *Page) Home(c *mizu.Ctx) error {
	viewerID := h.optionalAuth(c)
	var account *accounts.Account
	var postList []*posts.Post

	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
		postList, _ = h.timelines.Home(c.Request().Context(), viewerID, 20, "", "")
	} else {
		postList, _ = h.timelines.Local(c.Request().Context(), "", 20, "", "")
	}

	trendingTags, _ := h.trending.Tags(c.Request().Context(), 5)

	return h.render(c, "home", map[string]any{
		"Title":        "Home",
		"ActivePage":   "home",
		"Account":      account,
		"Posts":        postList,
		"TrendingTags": trendingTags,
	})
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	return h.render(c, "login", map[string]any{
		"Title": "Login",
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	return h.render(c, "register", map[string]any{
		"Title": "Register",
	})
}

// Profile renders a user's profile page.
func (h *Page) Profile(c *mizu.Ctx) error {
	username := c.Param("username")
	viewerID := h.optionalAuth(c)

	profile, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return c.Text(404, "User not found")
	}

	profile.FollowersCount, _ = h.relationships.CountFollowers(c.Request().Context(), profile.ID)
	profile.FollowingCount, _ = h.relationships.CountFollowing(c.Request().Context(), profile.ID)

	postList, _ := h.timelines.Account(c.Request().Context(), profile.ID, viewerID, 20, "", false, false)

	var account *accounts.Account
	isOwner := false
	isFollowing := false
	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
		isOwner = viewerID == profile.ID
		if !isOwner {
			rel, _ := h.relationships.Get(c.Request().Context(), viewerID, profile.ID)
			if rel != nil {
				isFollowing = rel.Following
			}
		}
	}

	return h.render(c, "profile", map[string]any{
		"Title":       "@" + profile.Username,
		"ActivePage":  "profile",
		"Account":     account,
		"Profile":     profile,
		"Posts":       postList,
		"IsOwner":     isOwner,
		"IsFollowing": isFollowing,
		"Tab":         "posts",
	})
}

// Post renders a single post/thread page.
func (h *Page) Post(c *mizu.Ctx) error {
	postID := c.Param("id")
	viewerID := h.optionalAuth(c)

	thread, err := h.posts.GetThread(c.Request().Context(), postID, viewerID)
	if err != nil {
		return c.Text(404, "Post not found")
	}

	var account *accounts.Account
	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
	}

	trendingTags, _ := h.trending.Tags(c.Request().Context(), 5)

	return h.render(c, "post", map[string]any{
		"Title":        "Post",
		"Account":      account,
		"Thread":       thread,
		"TrendingTags": trendingTags,
	})
}

// Tag renders a hashtag timeline page.
func (h *Page) Tag(c *mizu.Ctx) error {
	tag := c.Param("tag")
	viewerID := h.optionalAuth(c)

	postList, _ := h.timelines.Hashtag(c.Request().Context(), tag, viewerID, 20, "", "")

	var account *accounts.Account
	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
	}

	trendingTags, _ := h.trending.Tags(c.Request().Context(), 5)

	return h.render(c, "tag", map[string]any{
		"Title":        "#" + tag,
		"Account":      account,
		"Tag":          tag,
		"Posts":        postList,
		"TrendingTags": trendingTags,
	})
}

// Explore renders the explore page.
func (h *Page) Explore(c *mizu.Ctx) error {
	viewerID := h.optionalAuth(c)

	trendingTags, _ := h.trending.Tags(c.Request().Context(), 10)

	postIDs, _ := h.trending.Posts(c.Request().Context(), 20)
	var postList []*posts.Post
	for _, id := range postIDs {
		if p, err := h.posts.GetByID(c.Request().Context(), id, viewerID); err == nil {
			postList = append(postList, p)
		}
	}

	var account *accounts.Account
	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
	}

	return h.render(c, "explore", map[string]any{
		"Title":        "Explore",
		"ActivePage":   "explore",
		"Account":      account,
		"TrendingTags": trendingTags,
		"Posts":        postList,
	})
}

// Notifications renders the notifications page.
func (h *Page) Notifications(c *mizu.Ctx) error {
	accountID := h.optionalAuth(c)
	if accountID == "" {
		return c.Redirect(302, "/login")
	}

	account, _ := h.accounts.GetByID(c.Request().Context(), accountID)
	notifs, _ := h.notifications.List(c.Request().Context(), accountID, nil, 30, "", "", nil)
	trendingTags, _ := h.trending.Tags(c.Request().Context(), 5)

	return h.render(c, "notifications", map[string]any{
		"Title":         "Notifications",
		"ActivePage":    "notifications",
		"Account":       account,
		"Notifications": notifs,
		"TrendingTags":  trendingTags,
	})
}

// Bookmarks renders the bookmarks page.
func (h *Page) Bookmarks(c *mizu.Ctx) error {
	accountID := h.optionalAuth(c)
	if accountID == "" {
		return c.Redirect(302, "/login")
	}

	account, _ := h.accounts.GetByID(c.Request().Context(), accountID)
	postList, _ := h.timelines.Bookmarks(c.Request().Context(), accountID, 20, "")
	trendingTags, _ := h.trending.Tags(c.Request().Context(), 5)

	return h.render(c, "bookmarks", map[string]any{
		"Title":        "Bookmarks",
		"ActivePage":   "bookmarks",
		"Account":      account,
		"Posts":        postList,
		"TrendingTags": trendingTags,
	})
}

// Search renders the search page.
func (h *Page) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	viewerID := h.optionalAuth(c)

	var account *accounts.Account
	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
	}

	trendingTags, _ := h.trending.Tags(c.Request().Context(), 5)

	data := map[string]any{
		"Title":        "Search",
		"Account":      account,
		"Query":        query,
		"Tab":          "top",
		"TrendingTags": trendingTags,
	}

	// If there's a query, perform search
	// (simplified - real implementation would use search service)

	return h.render(c, "search", data)
}

// Settings renders the settings page.
func (h *Page) Settings(c *mizu.Ctx) error {
	accountID := h.optionalAuth(c)
	if accountID == "" {
		return c.Redirect(302, "/login")
	}

	account, _ := h.accounts.GetByID(c.Request().Context(), accountID)

	return h.render(c, "settings", map[string]any{
		"Title":        "Settings",
		"ActivePage":   "settings",
		"Account":      account,
		"SettingsPage": "profile",
	})
}

// FollowList renders the followers/following list page.
func (h *Page) FollowList(c *mizu.Ctx) error {
	username := c.Param("username")
	listType := c.Param("type") // "followers" or "following"
	viewerID := h.optionalAuth(c)

	profile, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return c.Text(404, "User not found")
	}

	var account *accounts.Account
	if viewerID != "" {
		account, _ = h.accounts.GetByID(c.Request().Context(), viewerID)
	}

	var users []*accounts.Account
	if listType == "followers" {
		ids, _ := h.relationships.GetFollowers(c.Request().Context(), profile.ID, 40, 0)
		for _, id := range ids {
			if a, err := h.accounts.GetByID(c.Request().Context(), id); err == nil {
				users = append(users, a)
			}
		}
	} else {
		ids, _ := h.relationships.GetFollowing(c.Request().Context(), profile.ID, 40, 0)
		for _, id := range ids {
			if a, err := h.accounts.GetByID(c.Request().Context(), id); err == nil {
				users = append(users, a)
			}
		}
	}

	return h.render(c, "follow_list", map[string]any{
		"Title":    listType + " - @" + profile.Username,
		"Account":  account,
		"Profile":  profile,
		"Users":    users,
		"ListType": listType,
	})
}
