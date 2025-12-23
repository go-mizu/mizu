package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/notifications"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
	"github.com/go-mizu/mizu/blueprints/forum/pkg/text"
)

// Page handles HTML page endpoints.
type Page struct {
	templates     map[string]*template.Template
	accounts      accounts.API
	boards        boards.API
	threads       threads.API
	comments      comments.API
	votes         votes.API
	bookmarks     bookmarks.API
	notifications notifications.API
	getAccountID  func(*mizu.Ctx) string
}

// NewPage creates a new page handler.
func NewPage(
	templates map[string]*template.Template,
	accounts accounts.API,
	boards boards.API,
	threads threads.API,
	comments comments.API,
	votes votes.API,
	bookmarks bookmarks.API,
	notifications notifications.API,
	getAccountID func(*mizu.Ctx) string,
) *Page {
	return &Page{
		templates:     templates,
		accounts:      accounts,
		boards:        boards,
		threads:       threads,
		comments:      comments,
		votes:         votes,
		bookmarks:     bookmarks,
		notifications: notifications,
		getAccountID:  getAccountID,
	}
}

// PageData is the data passed to page templates.
type PageData struct {
	Title       string
	CurrentUser *accounts.Account
	UnreadCount int64
	Data        any
}

func (h *Page) render(c *mizu.Ctx, name string, data PageData) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")

	// Get current user
	accountID := h.getAccountID(c)
	if accountID != "" {
		data.CurrentUser, _ = h.accounts.GetByID(c.Request().Context(), accountID)
		data.UnreadCount, _ = h.notifications.GetUnreadCount(c.Request().Context(), accountID)
	}

	return tmpl.ExecuteTemplate(c.Writer(), name, data)
}

// Home renders the home page.
func (h *Page) Home(c *mizu.Ctx) error {
	opts := threads.ListOpts{
		Limit:     25,
		SortBy:    threads.SortBy(c.Query("sort")),
		TimeRange: threads.TimeRange(c.Query("t")),
	}
	if opts.SortBy == "" {
		opts.SortBy = threads.SortHot
	}

	threadList, err := h.threads.List(c.Request().Context(), opts)
	if err != nil {
		return h.renderError(c, "Error loading threads")
	}

	// Get popular boards
	popularBoards, _ := h.boards.ListPopular(c.Request().Context(), 5)

	viewerID := h.getAccountID(c)
	h.enrichThreads(c, threadList, viewerID)

	return h.render(c, "home.html", PageData{
		Title: "Home",
		Data: map[string]any{
			"Threads":       threadList,
			"PopularBoards": popularBoards,
			"Sort":          opts.SortBy,
			"TimeRange":     opts.TimeRange,
		},
	})
}

// All renders the all posts page.
func (h *Page) All(c *mizu.Ctx) error {
	opts := threads.ListOpts{
		Limit:  25,
		SortBy: threads.SortNew,
	}

	threadList, err := h.threads.List(c.Request().Context(), opts)
	if err != nil {
		return h.renderError(c, "Error loading threads")
	}

	viewerID := h.getAccountID(c)
	h.enrichThreads(c, threadList, viewerID)

	return h.render(c, "all.html", PageData{
		Title: "All Posts",
		Data: map[string]any{
			"Threads": threadList,
		},
	})
}

// Board renders a board page.
func (h *Page) Board(c *mizu.Ctx) error {
	name := c.Param("name")

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		return h.renderError(c, "Board not found")
	}

	opts := threads.ListOpts{
		Limit:     25,
		SortBy:    threads.SortBy(c.Query("sort")),
		TimeRange: threads.TimeRange(c.Query("t")),
	}
	if opts.SortBy == "" {
		opts.SortBy = threads.SortHot
	}

	threadList, err := h.threads.ListByBoard(c.Request().Context(), board.ID, opts)
	if err != nil {
		return h.renderError(c, "Error loading threads")
	}

	viewerID := h.getAccountID(c)
	_ = h.boards.EnrichBoard(c.Request().Context(), board, viewerID)
	h.enrichThreads(c, threadList, viewerID)

	return h.render(c, "board.html", PageData{
		Title: "b/" + board.Name,
		Data: map[string]any{
			"Board":     board,
			"Threads":   threadList,
			"Sort":      opts.SortBy,
			"TimeRange": opts.TimeRange,
		},
	})
}

// Thread renders a thread page.
func (h *Page) Thread(c *mizu.Ctx) error {
	id := c.Param("id")

	thread, err := h.threads.GetByID(c.Request().Context(), id)
	if err != nil {
		return h.renderError(c, "Thread not found")
	}

	// Increment views
	_ = h.threads.IncrementViews(c.Request().Context(), id)

	// Get comments
	commentTree, err := h.comments.GetTree(c.Request().Context(), id, comments.TreeOpts{
		Sort:       comments.CommentSort(c.Query("sort")),
		Limit:      200,
		MaxDepth:   10,
		CollapseAt: 5,
	})
	if err != nil {
		commentTree = []*comments.Comment{}
	}

	viewerID := h.getAccountID(c)
	h.enrichThread(c, thread, viewerID)
	_ = h.comments.EnrichComments(c.Request().Context(), commentTree, viewerID)
	h.enrichCommentVotes(c, commentTree, viewerID)

	return h.render(c, "thread.html", PageData{
		Title: thread.Title,
		Data: map[string]any{
			"Thread":   thread,
			"Comments": commentTree,
		},
	})
}

// Submit renders the submit page.
func (h *Page) Submit(c *mizu.Ctx) error {
	name := c.Param("name")

	board, err := h.boards.GetByName(c.Request().Context(), name)
	if err != nil {
		return h.renderError(c, "Board not found")
	}

	return h.render(c, "submit.html", PageData{
		Title: "Submit to b/" + board.Name,
		Data: map[string]any{
			"Board": board,
		},
	})
}

// User renders a user profile page.
func (h *Page) User(c *mizu.Ctx) error {
	username := c.Param("username")

	account, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		return h.renderError(c, "User not found")
	}

	tab := c.Query("tab")
	if tab == "" {
		tab = "posts"
	}

	var userThreads []*threads.Thread
	var userComments []*comments.Comment

	if tab == "posts" {
		userThreads, _ = h.threads.ListByAuthor(c.Request().Context(), account.ID, threads.ListOpts{Limit: 25})
	} else if tab == "comments" {
		userComments, _ = h.comments.ListByAuthor(c.Request().Context(), account.ID, comments.ListOpts{Limit: 25})
	}

	return h.render(c, "user.html", PageData{
		Title: "u/" + account.Username,
		Data: map[string]any{
			"User":     account,
			"Tab":      tab,
			"Threads":  userThreads,
			"Comments": userComments,
		},
	})
}

// Search renders the search page.
func (h *Page) Search(c *mizu.Ctx) error {
	query := c.Query("q")

	var searchBoards []*boards.Board
	var searchUsers []*accounts.Account

	if query != "" {
		searchBoards, _ = h.boards.Search(c.Request().Context(), query, 10)
		searchUsers, _ = h.accounts.Search(c.Request().Context(), query, 10)
	}

	return h.render(c, "search.html", PageData{
		Title: "Search",
		Data: map[string]any{
			"Query":  query,
			"Boards": searchBoards,
			"Users":  searchUsers,
		},
	})
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	// Redirect if already logged in
	if h.getAccountID(c) != "" {
		http.Redirect(c.Writer(), c.Request(), "/", http.StatusFound)
		return nil
	}

	return h.render(c, "login.html", PageData{
		Title: "Login",
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	// Redirect if already logged in
	if h.getAccountID(c) != "" {
		http.Redirect(c.Writer(), c.Request(), "/", http.StatusFound)
		return nil
	}

	return h.render(c, "register.html", PageData{
		Title: "Register",
	})
}

// Settings renders the settings page.
func (h *Page) Settings(c *mizu.Ctx) error {
	return h.render(c, "settings.html", PageData{
		Title: "Settings",
	})
}

// Bookmarks renders the bookmarks page.
func (h *Page) Bookmarks(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return h.renderError(c, "Please log in to view bookmarks")
	}

	threadBookmarks, _ := h.bookmarks.List(c.Request().Context(), accountID, bookmarks.TargetThread, bookmarks.ListOpts{Limit: 50})

	// Load threads
	var savedThreads []*threads.Thread
	for _, b := range threadBookmarks {
		if thread, err := h.threads.GetByID(c.Request().Context(), b.TargetID); err == nil {
			savedThreads = append(savedThreads, thread)
		}
	}

	return h.render(c, "bookmarks.html", PageData{
		Title: "Bookmarks",
		Data: map[string]any{
			"Threads": savedThreads,
		},
	})
}

// Notifications renders the notifications page.
func (h *Page) Notifications(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return h.renderError(c, "Please log in to view notifications")
	}

	notificationList, _ := h.notifications.List(c.Request().Context(), accountID, notifications.ListOpts{Limit: 50})

	return h.render(c, "notifications.html", PageData{
		Title: "Notifications",
		Data: map[string]any{
			"Notifications": notificationList,
		},
	})
}

func (h *Page) renderError(c *mizu.Ctx, message string) error {
	return h.render(c, "error.html", PageData{
		Title: "Error",
		Data: map[string]any{
			"Message": message,
		},
	})
}

func (h *Page) enrichThread(c *mizu.Ctx, thread *threads.Thread, viewerID string) {
	if viewerID == "" {
		return
	}

	_ = h.threads.EnrichThread(c.Request().Context(), thread, viewerID)

	vote, err := h.votes.GetVote(c.Request().Context(), viewerID, votes.TargetThread, thread.ID)
	if err == nil && vote != nil {
		thread.Vote = vote.Value
	}

	isBookmarked, _ := h.bookmarks.IsBookmarked(c.Request().Context(), viewerID, bookmarks.TargetThread, thread.ID)
	thread.IsBookmarked = isBookmarked
}

func (h *Page) enrichThreads(c *mizu.Ctx, threadList []*threads.Thread, viewerID string) {
	if viewerID == "" {
		return
	}

	ids := make([]string, len(threadList))
	for i, t := range threadList {
		ids[i] = t.ID
	}

	voteMap, _ := h.votes.GetVotes(c.Request().Context(), viewerID, votes.TargetThread, ids)
	bookmarkMap, _ := h.bookmarks.GetBookmarked(c.Request().Context(), viewerID, bookmarks.TargetThread, ids)

	for _, t := range threadList {
		_ = h.threads.EnrichThread(c.Request().Context(), t, viewerID)
		if v, ok := voteMap[t.ID]; ok {
			t.Vote = v
		}
		if bookmarkMap[t.ID] {
			t.IsBookmarked = true
		}
	}
}

func (h *Page) enrichCommentVotes(c *mizu.Ctx, commentList []*comments.Comment, viewerID string) {
	if viewerID == "" || len(commentList) == 0 {
		return
	}

	var allComments []*comments.Comment
	var collect func([]*comments.Comment)
	collect = func(list []*comments.Comment) {
		for _, comment := range list {
			allComments = append(allComments, comment)
			if len(comment.Children) > 0 {
				collect(comment.Children)
			}
		}
	}
	collect(commentList)

	ids := make([]string, len(allComments))
	for i, comment := range allComments {
		ids[i] = comment.ID
	}

	voteMap, _ := h.votes.GetVotes(c.Request().Context(), viewerID, votes.TargetComment, ids)

	for _, comment := range allComments {
		if v, ok := voteMap[comment.ID]; ok {
			comment.Vote = v
		}
	}
}

// Slug generates a URL slug for a thread title.
func Slug(title string) string {
	return text.Slugify(title)
}
