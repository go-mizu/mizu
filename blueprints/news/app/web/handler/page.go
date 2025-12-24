package handler

import (
	"html/template"
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// Page handles HTML page rendering.
type Page struct {
	templates map[string]*template.Template
	users     *users.Service
	stories   *stories.Service
	comments  *comments.Service
	getUserID func(*mizu.Ctx) string
}

// NewPage creates a new page handler.
func NewPage(
	templates map[string]*template.Template,
	users *users.Service,
	stories *stories.Service,
	comments *comments.Service,
	getUserID func(*mizu.Ctx) string,
) *Page {
	return &Page{
		templates: templates,
		users:     users,
		stories:   stories,
		comments:  comments,
		getUserID: getUserID,
	}
}

// PageData holds common page data.
type PageData struct {
	Title       string
	User        *users.User
	Stories     []*stories.Story
	Story       *stories.Story
	Comments    []*comments.Comment
	Profile     *users.User
	Sort        string
	Page        int
	HasMore     bool
	Error       string
	Success     string
	Next        string
	StartOffset int
}

func (h *Page) render(c *mizu.Ctx, name string, data PageData) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(500, "Template not found: "+name)
	}
	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(c.Writer(), name, data)
}

func (h *Page) getCurrentUser(c *mizu.Ctx) *users.User {
	userID := h.getUserID(c)
	if userID == "" {
		return nil
	}
	user, _ := h.users.GetByID(c.Request().Context(), userID)
	return user
}

// Home renders the home page (hot stories).
func (h *Page) Home(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("p"))
	if page < 1 {
		page = 1
	}
	limit := 30
	offset := (page - 1) * limit

	userID := h.getUserID(c)
	storiesList, _ := h.stories.List(c.Request().Context(), stories.ListIn{
		Sort:   "hot",
		Limit:  limit + 1, // Fetch one extra to check for more
		Offset: offset,
	}, userID)

	hasMore := len(storiesList) > limit
	if hasMore {
		storiesList = storiesList[:limit]
	}

	return h.render(c, "home.html", PageData{
		Title:       "News",
		User:        h.getCurrentUser(c),
		Stories:     storiesList,
		Sort:        "hot",
		Page:        page,
		HasMore:     hasMore,
		StartOffset: offset,
	})
}

// Newest renders the newest stories page.
func (h *Page) Newest(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("p"))
	if page < 1 {
		page = 1
	}
	limit := 30
	offset := (page - 1) * limit

	userID := h.getUserID(c)
	storiesList, _ := h.stories.List(c.Request().Context(), stories.ListIn{
		Sort:   "new",
		Limit:  limit + 1,
		Offset: offset,
	}, userID)

	hasMore := len(storiesList) > limit
	if hasMore {
		storiesList = storiesList[:limit]
	}

	return h.render(c, "home.html", PageData{
		Title:       "Newest | News",
		User:        h.getCurrentUser(c),
		Stories:     storiesList,
		Sort:        "new",
		Page:        page,
		HasMore:     hasMore,
		StartOffset: offset,
	})
}

// Top renders the top stories page.
func (h *Page) Top(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("p"))
	if page < 1 {
		page = 1
	}
	limit := 30
	offset := (page - 1) * limit

	userID := h.getUserID(c)
	storiesList, _ := h.stories.List(c.Request().Context(), stories.ListIn{
		Sort:   "top",
		Limit:  limit + 1,
		Offset: offset,
	}, userID)

	hasMore := len(storiesList) > limit
	if hasMore {
		storiesList = storiesList[:limit]
	}

	return h.render(c, "home.html", PageData{
		Title:       "Top | News",
		User:        h.getCurrentUser(c),
		Stories:     storiesList,
		Sort:        "top",
		Page:        page,
		HasMore:     hasMore,
		StartOffset: offset,
	})
}

// Story renders a story page with comments.
func (h *Page) Story(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	story, err := h.stories.GetByID(c.Request().Context(), id, userID)
	if err != nil {
		return c.Text(404, "Story not found")
	}

	commentsList, _ := h.comments.ListByStory(c.Request().Context(), id, userID)

	return h.render(c, "story.html", PageData{
		Title:    story.Title + " | News",
		User:     h.getCurrentUser(c),
		Story:    story,
		Comments: commentsList,
	})
}

// User renders a user profile page.
func (h *Page) User(c *mizu.Ctx) error {
	username := c.Param("username")

	profile, err := h.users.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return c.Text(404, "User not found")
	}

	// Get user's stories
	userStories, _ := h.stories.ListByAuthor(c.Request().Context(), profile.ID, 30, 0, "")

	return h.render(c, "user.html", PageData{
		Title:   profile.Username + " | News",
		User:    h.getCurrentUser(c),
		Profile: profile,
		Stories: userStories,
	})
}


