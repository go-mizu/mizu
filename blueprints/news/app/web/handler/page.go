package handler

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/tags"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// Page handles HTML page rendering.
type Page struct {
	templates *template.Template
	users     *users.Service
	stories   *stories.Service
	comments  *comments.Service
	tags      *tags.Service
	getUserID func(*mizu.Ctx) string
}

// NewPage creates a new page handler.
func NewPage(
	templates *template.Template,
	users *users.Service,
	stories *stories.Service,
	comments *comments.Service,
	tags *tags.Service,
	getUserID func(*mizu.Ctx) string,
) *Page {
	return &Page{
		templates: templates,
		users:     users,
		stories:   stories,
		comments:  comments,
		tags:      tags,
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
	Tags        []*tags.Tag
	Tag         *tags.Tag
	Sort        string
	Page        int
	HasMore     bool
	Error       string
	Success     string
	Next        string
	StartOffset int
}

func (h *Page) render(c *mizu.Ctx, name string, data PageData) error {
	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(c.Writer(), name, data)
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

// Submit renders the submit form.
func (h *Page) Submit(c *mizu.Ctx) error {
	return h.render(c, "submit.html", PageData{
		Title: "Submit | News",
		User:  h.getCurrentUser(c),
	})
}

// SubmitPost handles the submit form.
func (h *Page) SubmitPost(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.Redirect(302, "/login")
	}

	form, err := c.Form()
	if err != nil {
		return h.render(c, "submit.html", PageData{
			Title: "Submit | News",
			User:  h.getCurrentUser(c),
			Error: "Failed to parse form",
		})
	}
	title := form.Get("title")
	url := form.Get("url")
	text := form.Get("text")

	story, err := h.stories.Create(c.Request().Context(), userID, stories.CreateIn{
		Title: title,
		URL:   url,
		Text:  text,
	})
	if err != nil {
		return h.render(c, "submit.html", PageData{
			Title: "Submit | News",
			User:  h.getCurrentUser(c),
			Error: err.Error(),
		})
	}

	return c.Redirect(302, "/story/"+story.ID)
}

// CommentPost handles comment submission.
func (h *Page) CommentPost(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return c.Redirect(302, "/login")
	}

	form, err := c.Form()
	if err != nil {
		return c.Redirect(302, "/")
	}
	storyID := form.Get("story_id")
	parentID := form.Get("parent_id")
	text := form.Get("text")

	_, err = h.comments.Create(c.Request().Context(), userID, comments.CreateIn{
		StoryID:  storyID,
		ParentID: parentID,
		Text:     text,
	})
	if err != nil {
		return c.Redirect(302, "/story/"+storyID+"?error="+err.Error())
	}

	return c.Redirect(302, "/story/"+storyID)
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

// Tag renders stories filtered by tag.
func (h *Page) Tag(c *mizu.Ctx) error {
	name := c.Param("name")
	page, _ := strconv.Atoi(c.Query("p"))
	if page < 1 {
		page = 1
	}
	limit := 30
	offset := (page - 1) * limit

	tag, err := h.tags.GetByName(c.Request().Context(), name)
	if err != nil {
		return c.Text(404, "Tag not found")
	}

	userID := h.getUserID(c)
	storiesList, _ := h.stories.List(c.Request().Context(), stories.ListIn{
		Tag:    name,
		Sort:   "hot",
		Limit:  limit + 1,
		Offset: offset,
	}, userID)

	hasMore := len(storiesList) > limit
	if hasMore {
		storiesList = storiesList[:limit]
	}

	return h.render(c, "tag.html", PageData{
		Title:       tag.Name + " | News",
		User:        h.getCurrentUser(c),
		Tag:         tag,
		Stories:     storiesList,
		Page:        page,
		HasMore:     hasMore,
		StartOffset: offset,
	})
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	next := c.Query("next")
	if next == "" {
		next = "/"
	}
	return h.render(c, "login.html", PageData{
		Title: "Login | News",
		User:  h.getCurrentUser(c),
		Next:  next,
	})
}

// LoginPost handles the login form.
func (h *Page) LoginPost(c *mizu.Ctx) error {
	form, err := c.Form()
	if err != nil {
		return h.render(c, "login.html", PageData{
			Title: "Login | News",
			Error: "Failed to parse form",
		})
	}
	username := form.Get("username")
	password := form.Get("password")
	next := form.Get("next")
	if next == "" {
		next = "/"
	}

	user, err := h.users.Login(c.Request().Context(), users.LoginIn{
		Username: username,
		Password: password,
	})
	if err != nil {
		return h.render(c, "login.html", PageData{
			Title: "Login | News",
			Error: "Invalid username or password",
			Next:  next,
		})
	}

	session, err := h.users.CreateSession(c.Request().Context(), user.ID)
	if err != nil {
		return h.render(c, "login.html", PageData{
			Title: "Login | News",
			Error: "Failed to create session",
			Next:  next,
		})
	}

	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(302, next)
}

// Register renders the register page.
func (h *Page) Register(c *mizu.Ctx) error {
	return h.render(c, "register.html", PageData{
		Title: "Register | News",
		User:  h.getCurrentUser(c),
	})
}

// RegisterPost handles the registration form.
func (h *Page) RegisterPost(c *mizu.Ctx) error {
	form, err := c.Form()
	if err != nil {
		return h.render(c, "register.html", PageData{
			Title: "Register | News",
			Error: "Failed to parse form",
		})
	}
	username := form.Get("username")
	email := form.Get("email")
	password := form.Get("password")

	user, err := h.users.Create(c.Request().Context(), users.CreateIn{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		return h.render(c, "register.html", PageData{
			Title: "Register | News",
			Error: err.Error(),
		})
	}

	session, err := h.users.CreateSession(c.Request().Context(), user.ID)
	if err != nil {
		return h.render(c, "login.html", PageData{
			Title:   "Login | News",
			Success: "Account created! Please log in.",
		})
	}

	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(302, "/")
}

// LogoutPost handles logout.
func (h *Page) LogoutPost(c *mizu.Ctx) error {
	cookie, err := c.Request().Cookie("session")
	if err == nil && cookie != nil {
		_ = h.users.DeleteSession(c.Request().Context(), cookie.Value)
	}

	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return c.Redirect(302, "/")
}
