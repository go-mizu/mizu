package wpadmin

import (
	"context"
	"crypto/md5"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/assets"
	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/menus"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/settings"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// Config holds handler configuration.
type Config struct {
	BaseURL    string
	Users      users.API
	Posts      posts.API
	Pages      pages.API
	Categories categories.API
	Tags       tags.API
	Media      media.API
	Comments   comments.API
	Settings   settings.API
	Menus      menus.API
	GetUserID  func(*mizu.Ctx) string
	GetUser    func(*mizu.Ctx) *users.User
}

// Handler handles WordPress admin pages.
type Handler struct {
	templates  map[string]*template.Template
	baseURL    string
	users      users.API
	posts      posts.API
	pages      pages.API
	categories categories.API
	tags       tags.API
	media      media.API
	comments   comments.API
	settings   settings.API
	menus      menus.API
	getUserID  func(*mizu.Ctx) string
	getUser    func(*mizu.Ctx) *users.User
}

// New creates a new Handler.
func New(templates map[string]*template.Template, cfg Config) *Handler {
	return &Handler{
		templates:  templates,
		baseURL:    cfg.BaseURL,
		users:      cfg.Users,
		posts:      cfg.Posts,
		pages:      cfg.Pages,
		categories: cfg.Categories,
		tags:       cfg.Tags,
		media:      cfg.Media,
		comments:   cfg.Comments,
		settings:   cfg.Settings,
		menus:      cfg.Menus,
		getUserID:  cfg.GetUserID,
		getUser:    cfg.GetUser,
	}
}

// render renders a template with the given data.
func render[T any](h *Handler, c *mizu.Ctx, name string, data T) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}

// requireAuth redirects to login if not authenticated.
func (h *Handler) requireAuth(c *mizu.Ctx) *users.User {
	user := h.getUser(c)
	if user == nil {
		redirectTo := c.Request().URL.String()
		http.Redirect(c.Writer(), c.Request(), "/wp-login.php?redirect_to="+redirectTo, http.StatusFound)
		return nil
	}
	return user
}

// buildMenu builds the admin navigation menu.
func (h *Handler) buildMenu(c *mizu.Ctx, activeItem string) []MenuItem {
	ctx := c.Request().Context()

	// Get pending comment count for badge
	pendingComments := 0
	if commentList, _, err := h.comments.List(ctx, &comments.ListIn{Status: "pending"}); err == nil {
		pendingComments = len(commentList)
	}

	return []MenuItem{
		{
			ID:     "dashboard",
			Title:  "Dashboard",
			URL:    "/wp-admin/",
			Icon:   "dashicons-dashboard",
			Active: activeItem == "dashboard",
		},
		{
			ID:     "posts",
			Title:  "Posts",
			URL:    "/wp-admin/edit.php",
			Icon:   "dashicons-admin-post",
			Active: strings.HasPrefix(activeItem, "post"),
			Open:   strings.HasPrefix(activeItem, "post"),
			Children: []MenuItem{
				{ID: "posts-all", Title: "All Posts", URL: "/wp-admin/edit.php", Active: activeItem == "posts"},
				{ID: "posts-new", Title: "Add New", URL: "/wp-admin/post-new.php", Active: activeItem == "post-new"},
				{ID: "categories", Title: "Categories", URL: "/wp-admin/edit-tags.php?taxonomy=category", Active: activeItem == "categories"},
				{ID: "tags", Title: "Tags", URL: "/wp-admin/edit-tags.php?taxonomy=post_tag", Active: activeItem == "tags"},
			},
		},
		{
			ID:     "media",
			Title:  "Media",
			URL:    "/wp-admin/upload.php",
			Icon:   "dashicons-admin-media",
			Active: strings.HasPrefix(activeItem, "media"),
			Open:   strings.HasPrefix(activeItem, "media"),
			Children: []MenuItem{
				{ID: "media-lib", Title: "Library", URL: "/wp-admin/upload.php", Active: activeItem == "media"},
				{ID: "media-new", Title: "Add New", URL: "/wp-admin/media-new.php", Active: activeItem == "media-new"},
			},
		},
		{
			ID:     "pages",
			Title:  "Pages",
			URL:    "/wp-admin/edit.php?post_type=page",
			Icon:   "dashicons-admin-page",
			Active: strings.HasPrefix(activeItem, "page"),
			Open:   strings.HasPrefix(activeItem, "page"),
			Children: []MenuItem{
				{ID: "pages-all", Title: "All Pages", URL: "/wp-admin/edit.php?post_type=page", Active: activeItem == "pages"},
				{ID: "pages-new", Title: "Add New", URL: "/wp-admin/post-new.php?post_type=page", Active: activeItem == "page-new"},
			},
		},
		{
			ID:     "comments",
			Title:  "Comments",
			URL:    "/wp-admin/edit-comments.php",
			Icon:   "dashicons-admin-comments",
			Active: activeItem == "comments",
			Badge:  pendingComments,
		},
		{
			ID:     "appearance",
			Title:  "Appearance",
			URL:    "/wp-admin/themes.php",
			Icon:   "dashicons-admin-appearance",
			Active: strings.HasPrefix(activeItem, "appearance"),
			Open:   strings.HasPrefix(activeItem, "appearance"),
			Children: []MenuItem{
				{ID: "themes", Title: "Themes", URL: "/wp-admin/themes.php", Active: activeItem == "themes"},
				{ID: "menus", Title: "Menus", URL: "/wp-admin/nav-menus.php", Active: activeItem == "menus"},
			},
		},
		{
			ID:     "users",
			Title:  "Users",
			URL:    "/wp-admin/users.php",
			Icon:   "dashicons-admin-users",
			Active: strings.HasPrefix(activeItem, "user"),
			Open:   strings.HasPrefix(activeItem, "user"),
			Children: []MenuItem{
				{ID: "users-all", Title: "All Users", URL: "/wp-admin/users.php", Active: activeItem == "users"},
				{ID: "users-new", Title: "Add New", URL: "/wp-admin/user-new.php", Active: activeItem == "user-new"},
				{ID: "profile", Title: "Profile", URL: "/wp-admin/profile.php", Active: activeItem == "profile"},
			},
		},
		{
			ID:     "settings",
			Title:  "Settings",
			URL:    "/wp-admin/options-general.php",
			Icon:   "dashicons-admin-settings",
			Active: strings.HasPrefix(activeItem, "settings"),
			Open:   strings.HasPrefix(activeItem, "settings"),
			Children: []MenuItem{
				{ID: "settings-general", Title: "General", URL: "/wp-admin/options-general.php", Active: activeItem == "settings-general"},
				{ID: "settings-writing", Title: "Writing", URL: "/wp-admin/options-writing.php", Active: activeItem == "settings-writing"},
				{ID: "settings-reading", Title: "Reading", URL: "/wp-admin/options-reading.php", Active: activeItem == "settings-reading"},
				{ID: "settings-discussion", Title: "Discussion", URL: "/wp-admin/options-discussion.php", Active: activeItem == "settings-discussion"},
				{ID: "settings-media", Title: "Media", URL: "/wp-admin/options-media.php", Active: activeItem == "settings-media"},
				{ID: "settings-permalinks", Title: "Permalinks", URL: "/wp-admin/options-permalink.php", Active: activeItem == "settings-permalinks"},
			},
		},
	}
}

// gravatarURL generates a Gravatar URL for an email.
func gravatarURL(email string, size int) string {
	hash := md5.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=mm", hash, size)
}

// formatFileSize formats a file size in bytes to human-readable format.
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// parseIntDefault parses an int from string with a default value.
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// getSiteTitle returns the site title from settings.
func (h *Handler) getSiteTitle(c *mizu.Ctx) string {
	ctx := c.Request().Context()
	if setting, err := h.settings.Get(ctx, "site_title"); err == nil && setting.Value != "" {
		return setting.Value
	}
	return "WordPress"
}

// Login renders the login page (GET).
func (h *Handler) Login(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	// Check if already logged in
	if user := h.getUser(c); user != nil {
		redirectTo := c.Query("redirect_to")
		if redirectTo == "" {
			redirectTo = "/wp-admin/"
		}
		http.Redirect(c.Writer(), c.Request(), redirectTo, http.StatusFound)
		return nil
	}

	// Get site settings
	siteTitle := "WordPress"
	if setting, err := h.settings.Get(ctx, "site_title"); err == nil && setting.Value != "" {
		siteTitle = setting.Value
	}

	redirectTo := c.Query("redirect_to")
	if redirectTo == "" {
		redirectTo = "/wp-admin/"
	}

	errorMsg := ""
	if c.Query("error") == "1" {
		errorMsg = "Invalid username or password."
	}
	if c.Query("loggedout") == "true" {
		errorMsg = ""
	}

	message := ""
	if c.Query("loggedout") == "true" {
		message = "You are now logged out."
	}

	return render(h, c, "login", LoginData{
		Title:      "Log In",
		SiteTitle:  siteTitle,
		SiteURL:    h.baseURL,
		RedirectTo: redirectTo,
		Error:      errorMsg,
		Message:    message,
	})
}

// LoginPost handles the login form submission (POST).
func (h *Handler) LoginPost(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		return h.loginError(c, "Invalid form data.")
	}

	email := c.Request().FormValue("log")
	password := c.Request().FormValue("pwd")
	redirectTo := c.Request().FormValue("redirect_to")
	rememberMe := c.Request().FormValue("rememberme") == "forever"

	if redirectTo == "" {
		redirectTo = "/wp-admin/"
	}

	// Attempt login
	user, session, err := h.users.Login(ctx, &users.LoginIn{
		Email:    email,
		Password: password,
	})
	if err != nil || user == nil {
		return h.loginError(c, "Invalid username or password.")
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if rememberMe {
		cookie.Expires = session.ExpiresAt
		cookie.MaxAge = 14 * 24 * 60 * 60 // 14 days
	}
	http.SetCookie(c.Writer(), cookie)

	// Redirect to destination
	http.Redirect(c.Writer(), c.Request(), redirectTo, http.StatusFound)
	return nil
}

// loginError renders the login page with an error message.
func (h *Handler) loginError(c *mizu.Ctx, errorMsg string) error {
	ctx := c.Request().Context()

	siteTitle := "WordPress"
	if setting, err := h.settings.Get(ctx, "site_title"); err == nil && setting.Value != "" {
		siteTitle = setting.Value
	}

	redirectTo := c.Request().FormValue("redirect_to")
	if redirectTo == "" {
		redirectTo = "/wp-admin/"
	}

	return render(h, c, "login", LoginData{
		Title:      "Log In",
		SiteTitle:  siteTitle,
		SiteURL:    h.baseURL,
		RedirectTo: redirectTo,
		Error:      errorMsg,
	})
}

// Logout handles user logout.
func (h *Handler) Logout(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	// Get session from cookie and invalidate it
	if cookie, err := c.Cookie("session"); err == nil && cookie.Value != "" {
		_ = h.users.Logout(ctx, cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to login page with logged out message
	http.Redirect(c.Writer(), c.Request(), "/wp-login.php?loggedout=true", http.StatusFound)
	return nil
}

// Dashboard renders the main dashboard page.
func (h *Handler) Dashboard(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Get site settings
	siteTitle := "WordPress"
	if setting, err := h.settings.Get(ctx, "site_title"); err == nil && setting.Value != "" {
		siteTitle = setting.Value
	}

	// Get counts for At a Glance
	postList, _, _ := h.posts.List(ctx, &posts.ListIn{Status: "published"})
	pageList, _, _ := h.pages.List(ctx, &pages.ListIn{Status: "published"})
	commentList, _, _ := h.comments.List(ctx, &comments.ListIn{})
	categoryList, _, _ := h.categories.List(ctx, &categories.ListIn{})
	tagList, _, _ := h.tags.List(ctx, &tags.ListIn{})
	userList, _, _ := h.users.List(ctx, &users.ListIn{})
	mediaList, _, _ := h.media.List(ctx, &media.ListIn{})

	atAGlance := AtAGlance{
		PostCount:     len(postList),
		PageCount:     len(pageList),
		CommentCount:  len(commentList),
		CategoryCount: len(categoryList),
		TagCount:      len(tagList),
		UserCount:     len(userList),
		MediaCount:    len(mediaList),
		Theme:         "Default Theme",
		GoVersion:     "Go 1.22",
	}

	// Get recent activity
	var activity []ActivityItem
	recentPosts, _, _ := h.posts.List(ctx, &posts.ListIn{Limit: 5, OrderBy: "created_at", Order: "desc"})
	for _, p := range recentPosts {
		var authorName string
		if author, err := h.users.GetByID(ctx, p.AuthorID); err == nil {
			authorName = author.Name
		}
		activity = append(activity, ActivityItem{
			Type:    "post",
			Title:   p.Title,
			Author:  authorName,
			Date:    p.CreatedAt,
			Status:  p.Status,
			EditURL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", p.ID),
			ViewURL: fmt.Sprintf("/%s/", p.Slug),
		})
	}

	// Get recent comments
	var recentComments []*CommentRow
	recentCommentsList, _, _ := h.comments.List(ctx, &comments.ListIn{Limit: 5})
	for _, cm := range recentCommentsList {
		var postTitle string
		if post, err := h.posts.GetByID(ctx, cm.PostID); err == nil {
			postTitle = post.Title
		}
		recentComments = append(recentComments, &CommentRow{
			Comment:     cm,
			AuthorName:  cm.AuthorName,
			AuthorEmail: cm.AuthorEmail,
			AuthorAvatar: gravatarURL(cm.AuthorEmail, 32),
			PostTitle:   postTitle,
			PostEditURL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", cm.PostID),
		})
	}

	// Get recent drafts
	var recentDrafts []RecentDraft
	draftPosts, _, _ := h.posts.List(ctx, &posts.ListIn{Status: "draft", AuthorID: user.ID, Limit: 3})
	for _, p := range draftPosts {
		recentDrafts = append(recentDrafts, RecentDraft{
			ID:      p.ID,
			Title:   p.Title,
			Date:    p.UpdatedAt,
			EditURL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", p.ID),
		})
	}

	return render(h, c, "dashboard", DashboardData{
		Title:       "Dashboard",
		User:        user,
		Menu:        h.buildMenu(c, "dashboard"),
		Breadcrumbs: []Breadcrumb{{Label: "Dashboard", URL: "/wp-admin/"}},
		SiteTitle:   siteTitle,
		SiteURL:     h.baseURL,
		AtAGlance:   atAGlance,
		Activity:    activity,
		QuickDraft:  QuickDraftData{Enabled: true},
		RecentDrafts:   recentDrafts,
		RecentComments: recentComments,
		WelcomePanel:   true,
		SiteHealth: SiteHealth{
			Status: "good",
			Tests:  10,
			Passed: 10,
		},
	})
}

// PostsList renders the posts list page.
func (h *Handler) PostsList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Parse query params
	status := c.Query("post_status")
	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)
	orderBy := c.Query("orderby")
	if orderBy == "" {
		orderBy = "date"
	}
	order := c.Query("order")
	if order == "" {
		order = "desc"
	}

	// Map orderby values
	dbOrderBy := "created_at"
	switch orderBy {
	case "title":
		dbOrderBy = "title"
	case "author":
		dbOrderBy = "author_id"
	case "date":
		dbOrderBy = "created_at"
	}

	// Get posts
	params := &posts.ListIn{
		Status:  status,
		Search:  search,
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
		OrderBy: dbOrderBy,
		Order:   order,
	}

	postList, _, _ := h.posts.List(ctx, params)

	// Get total count for each status
	allPosts, _, _ := h.posts.List(ctx, &posts.ListIn{})
	publishedCount := 0
	draftCount := 0
	trashCount := 0
	for _, p := range allPosts {
		switch p.Status {
		case "published":
			publishedCount++
		case "draft":
			draftCount++
		case "trash":
			trashCount++
		}
	}

	// Build post rows with related data
	var postRows []*PostRow
	for _, p := range postList {
		author, _ := h.users.GetByID(ctx, p.AuthorID)

		// Get categories for this post
		var cats []*categories.Category
		if catIDs, err := h.posts.GetCategoryIDs(ctx, p.ID); err == nil && len(catIDs) > 0 {
			for _, catID := range catIDs {
				if cat, err := h.categories.GetByID(ctx, catID); err == nil {
					cats = append(cats, cat)
				}
			}
		}

		// Get tags for this post
		var tagsList []*tags.Tag
		if tagIDs, err := h.posts.GetTagIDs(ctx, p.ID); err == nil && len(tagIDs) > 0 {
			tagsList, _ = h.tags.GetByIDs(ctx, tagIDs)
		}

		commentCount := 0
		if cms, _, err := h.comments.List(ctx, &comments.ListIn{PostID: p.ID}); err == nil {
			commentCount = len(cms)
		}

		postRows = append(postRows, &PostRow{
			Post:         p,
			Author:       author,
			Categories:   cats,
			Tags:         tagsList,
			CommentCount: commentCount,
			RowActions: []RowAction{
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", p.ID)},
				{Label: "Quick Edit", URL: "#", Class: "editinline"},
				{Label: "Trash", URL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=trash", p.ID), Class: "submitdelete"},
				{Label: "View", URL: fmt.Sprintf("/%s/", p.Slug)},
			},
		})
	}

	// Build status tabs
	statusTabs := []StatusTab{
		{Status: "", Label: "All", Count: len(allPosts), Active: status == "", URL: "/wp-admin/edit.php"},
		{Status: "published", Label: "Published", Count: publishedCount, Active: status == "published", URL: "/wp-admin/edit.php?post_status=published"},
		{Status: "draft", Label: "Draft", Count: draftCount, Active: status == "draft", URL: "/wp-admin/edit.php?post_status=draft"},
		{Status: "trash", Label: "Trash", Count: trashCount, Active: status == "trash", URL: "/wp-admin/edit.php?post_status=trash"},
	}

	// Build columns
	columns := []TableColumn{
		{ID: "title", Label: "Title", Sortable: true, Primary: true},
		{ID: "author", Label: "Author", Sortable: true},
		{ID: "categories", Label: "Categories"},
		{ID: "tags", Label: "Tags"},
		{ID: "comments", Label: "Comments", Class: "num"},
		{ID: "date", Label: "Date", Sortable: true},
	}

	// Build bulk actions
	bulkActions := []BulkAction{
		{Value: "edit", Label: "Edit"},
		{Value: "trash", Label: "Move to Trash"},
	}

	totalItems := len(allPosts)
	if status != "" {
		totalItems = len(postList)
	}

	return render(h, c, "posts", PostsListData{
		Title:       "Posts",
		User:        user,
		Menu:        h.buildMenu(c, "posts"),
		Breadcrumbs: []Breadcrumb{{Label: "Posts", URL: "/wp-admin/edit.php"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Posts:       postRows,
		Columns:     columns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (totalItems + perPage - 1) / perPage,
			TotalItems:  totalItems,
			PerPage:     perPage,
			BaseURL:     "/wp-admin/edit.php",
		},
		BulkActions: bulkActions,
		StatusTabs:  statusTabs,
		ActiveTab:   status,
		SearchQuery: search,
		OrderBy:     orderBy,
		Order:       order,
		PostType:    "post",
	})
}

// PostNew renders the new post page.
func (h *Handler) PostNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Get categories
	allCategories, _, _ := h.categories.List(ctx, &categories.ListIn{})
	categoryOptions := buildCategoryTree(allCategories, nil)

	// Get tags
	allTags, _, _ := h.tags.List(ctx, &tags.ListIn{})

	// Build statuses
	statuses := []PostStatus{
		{Value: "draft", Label: "Draft"},
		{Value: "pending", Label: "Pending Review"},
		{Value: "published", Label: "Published"},
	}

	visibilities := []Visibility{
		{Value: "public", Label: "Public"},
		{Value: "private", Label: "Private"},
		{Value: "password", Label: "Password Protected"},
	}

	return render(h, c, "post-edit", PostEditData{
		Title:             "Add New Post",
		User:              user,
		Menu:              h.buildMenu(c, "post-new"),
		Breadcrumbs:       []Breadcrumb{{Label: "Posts", URL: "/wp-admin/edit.php"}, {Label: "Add New"}},
		SiteTitle:         h.getSiteTitle(c),
		SiteURL:           h.baseURL,
		IsNew:             true,
		PostType:          "post",
		AllCategories:     categoryOptions,
		AllTags:           allTags,
		Statuses:          statuses,
		Visibilities:      visibilities,
		CurrentStatus:     "draft",
		CurrentVisibility: "public",
	})
}

// PostEdit renders the post edit page.
func (h *Handler) PostEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	postID := c.Query("post")
	if postID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	post, err := h.posts.GetByID(ctx, postID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	// Get categories
	allCategories, _, _ := h.categories.List(ctx, &categories.ListIn{})
	selectedCats, _ := h.posts.GetCategoryIDs(ctx, postID)
	categoryOptions := buildCategoryTree(allCategories, selectedCats)

	// Get tags
	allTags, _, _ := h.tags.List(ctx, &tags.ListIn{})
	selectedTagIDs, _ := h.posts.GetTagIDs(ctx, postID)

	// Get featured media
	var featuredMedia *media.Media
	if post.FeaturedImageID != "" {
		featuredMedia, _ = h.media.GetByID(ctx, post.FeaturedImageID)
	}

	// Build statuses
	statuses := []PostStatus{
		{Value: "draft", Label: "Draft"},
		{Value: "pending", Label: "Pending Review"},
		{Value: "published", Label: "Published"},
	}

	visibilities := []Visibility{
		{Value: "public", Label: "Public"},
		{Value: "private", Label: "Private"},
		{Value: "password", Label: "Password Protected"},
	}

	visibility := "public"
	if post.Visibility != "" {
		visibility = post.Visibility
	}

	return render(h, c, "post-edit", PostEditData{
		Title:              "Edit Post",
		User:               user,
		Menu:               h.buildMenu(c, "posts"),
		Breadcrumbs:        []Breadcrumb{{Label: "Posts", URL: "/wp-admin/edit.php"}, {Label: "Edit"}},
		SiteTitle:          h.getSiteTitle(c),
		SiteURL:            h.baseURL,
		Post:               post,
		IsNew:              false,
		PostType:           "post",
		SelectedCategories: selectedCats,
		AllCategories:      categoryOptions,
		SelectedTags:       selectedTagIDs,
		AllTags:            allTags,
		FeaturedMedia:      featuredMedia,
		Statuses:           statuses,
		Visibilities:       visibilities,
		CurrentStatus:      post.Status,
		CurrentVisibility:  visibility,
		Excerpt:            post.Excerpt,
		Slug:               post.Slug,
	})
}

// buildCategoryTree builds a hierarchical category list.
func buildCategoryTree(cats []*categories.Category, selected []string) []*CategoryOption {
	// Build parent map
	childrenMap := make(map[string][]*categories.Category)
	var rootCats []*categories.Category

	for _, cat := range cats {
		if cat.ParentID == "" {
			rootCats = append(rootCats, cat)
		} else {
			childrenMap[cat.ParentID] = append(childrenMap[cat.ParentID], cat)
		}
	}

	var buildTree func([]*categories.Category, int) []*CategoryOption
	buildTree = func(cats []*categories.Category, depth int) []*CategoryOption {
		var result []*CategoryOption
		for _, cat := range cats {
			isSelected := false
			for _, s := range selected {
				if s == cat.ID {
					isSelected = true
					break
				}
			}
			opt := &CategoryOption{
				Category: cat,
				Depth:    depth,
				Selected: isSelected,
			}
			if children, ok := childrenMap[cat.ID]; ok {
				opt.Children = buildTree(children, depth+1)
			}
			result = append(result, opt)
		}
		return result
	}

	return buildTree(rootCats, 0)
}

// PagesList renders the pages list page.
func (h *Handler) PagesList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Parse query params
	status := c.Query("post_status")
	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := parseIntDefault(c.Query("per_page"), 20)

	// Get pages
	params := &pages.ListIn{
		Status: status,
		Search: search,
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	pageList, _, _ := h.pages.List(ctx, params)

	// Get counts
	allPages, _, _ := h.pages.List(ctx, &pages.ListIn{})
	publishedCount := 0
	draftCount := 0
	for _, p := range allPages {
		switch p.Status {
		case "published":
			publishedCount++
		case "draft":
			draftCount++
		}
	}

	// Build page rows
	var pageRows []*PageRow
	for _, p := range pageList {
		author, _ := h.users.GetByID(ctx, p.AuthorID)
		pageRows = append(pageRows, &PageRow{
			Page:   p,
			Author: author,
			RowActions: []RowAction{
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit&post_type=page", p.ID)},
				{Label: "Quick Edit", URL: "#", Class: "editinline"},
				{Label: "Trash", URL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=trash", p.ID), Class: "submitdelete"},
				{Label: "View", URL: fmt.Sprintf("/%s/", p.Slug)},
			},
		})
	}

	// Build status tabs
	statusTabs := []StatusTab{
		{Status: "", Label: "All", Count: len(allPages), Active: status == "", URL: "/wp-admin/edit.php?post_type=page"},
		{Status: "published", Label: "Published", Count: publishedCount, Active: status == "published", URL: "/wp-admin/edit.php?post_type=page&post_status=published"},
		{Status: "draft", Label: "Draft", Count: draftCount, Active: status == "draft", URL: "/wp-admin/edit.php?post_type=page&post_status=draft"},
	}

	columns := []TableColumn{
		{ID: "title", Label: "Title", Sortable: true, Primary: true},
		{ID: "author", Label: "Author", Sortable: true},
		{ID: "date", Label: "Date", Sortable: true},
	}

	bulkActions := []BulkAction{
		{Value: "edit", Label: "Edit"},
		{Value: "trash", Label: "Move to Trash"},
	}

	return render(h, c, "pages-list", PagesListData{
		Title:       "Pages",
		User:        user,
		Menu:        h.buildMenu(c, "pages"),
		Breadcrumbs: []Breadcrumb{{Label: "Pages", URL: "/wp-admin/edit.php?post_type=page"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Pages:       pageRows,
		Columns:     columns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (len(allPages) + perPage - 1) / perPage,
			TotalItems:  len(allPages),
			PerPage:     perPage,
			BaseURL:     "/wp-admin/edit.php?post_type=page",
		},
		BulkActions: bulkActions,
		StatusTabs:  statusTabs,
		ActiveTab:   status,
		SearchQuery: search,
	})
}

// PageNew renders the new page page.
func (h *Handler) PageNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Get parent pages
	allPages, _, _ := h.pages.List(ctx, &pages.ListIn{})
	var parentPages []*PageOption
	for _, p := range allPages {
		parentPages = append(parentPages, &PageOption{Page: p, Depth: 0})
	}

	statuses := []PostStatus{
		{Value: "draft", Label: "Draft"},
		{Value: "pending", Label: "Pending Review"},
		{Value: "published", Label: "Published"},
	}

	visibilities := []Visibility{
		{Value: "public", Label: "Public"},
		{Value: "private", Label: "Private"},
		{Value: "password", Label: "Password Protected"},
	}

	return render(h, c, "page-edit", PageEditData{
		Title:             "Add New Page",
		User:              user,
		Menu:              h.buildMenu(c, "page-new"),
		Breadcrumbs:       []Breadcrumb{{Label: "Pages", URL: "/wp-admin/edit.php?post_type=page"}, {Label: "Add New"}},
		SiteTitle:         h.getSiteTitle(c),
		SiteURL:           h.baseURL,
		IsNew:             true,
		ParentPages:       parentPages,
		Statuses:          statuses,
		Visibilities:      visibilities,
		CurrentStatus:     "draft",
		CurrentVisibility: "public",
	})
}

// PageEdit renders the page edit page.
func (h *Handler) PageEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	pageID := c.Query("post")
	if pageID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?post_type=page", http.StatusFound)
		return nil
	}

	pg, err := h.pages.GetByID(ctx, pageID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?post_type=page", http.StatusFound)
		return nil
	}

	// Get parent pages
	allPages, _, _ := h.pages.List(ctx, &pages.ListIn{})
	var parentPages []*PageOption
	for _, p := range allPages {
		if p.ID != pageID { // Exclude current page
			parentPages = append(parentPages, &PageOption{Page: p, Depth: 0})
		}
	}

	statuses := []PostStatus{
		{Value: "draft", Label: "Draft"},
		{Value: "pending", Label: "Pending Review"},
		{Value: "published", Label: "Published"},
	}

	visibilities := []Visibility{
		{Value: "public", Label: "Public"},
		{Value: "private", Label: "Private"},
		{Value: "password", Label: "Password Protected"},
	}

	visibility := "public"
	if pg.Visibility != "" {
		visibility = pg.Visibility
	}

	return render(h, c, "page-edit", PageEditData{
		Title:             "Edit Page",
		User:              user,
		Menu:              h.buildMenu(c, "pages"),
		Breadcrumbs:       []Breadcrumb{{Label: "Pages", URL: "/wp-admin/edit.php?post_type=page"}, {Label: "Edit"}},
		SiteTitle:         h.getSiteTitle(c),
		SiteURL:           h.baseURL,
		Page:              pg,
		IsNew:             false,
		ParentPages:       parentPages,
		SelectedParent:    pg.ParentID,
		Statuses:          statuses,
		Visibilities:      visibilities,
		CurrentStatus:     pg.Status,
		CurrentVisibility: visibility,
		MenuOrder:         pg.SortOrder,
	})
}

// MediaLibrary renders the media library page.
func (h *Handler) MediaLibrary(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Parse query params
	mode := c.Query("mode")
	if mode == "" {
		mode = "grid"
	}
	mediaType := c.Query("attachment-filter")
	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := 40

	// Get media
	params := &media.ListIn{
		MimeType: mediaType,
		Search:   search,
		Limit:    perPage,
		Offset:   (page - 1) * perPage,
	}

	mediaList, _, _ := h.media.List(ctx, params)

	// Build media items
	var items []*MediaItem
	for _, m := range mediaList {
		uploader, _ := h.users.GetByID(ctx, m.UploaderID)

		thumbnailURL := m.URL
		if m.MimeType != "" && !strings.HasPrefix(m.MimeType, "image/") {
			thumbnailURL = "/wp-admin/images/media/default.png" // Placeholder
		}

		items = append(items, &MediaItem{
			Media:        m,
			Uploader:     uploader,
			ThumbnailURL: thumbnailURL,
			FileSize:     formatFileSize(m.FileSize),
			Dimensions:   fmt.Sprintf("%dx%d", m.Width, m.Height),
			RowActions: []RowAction{
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", m.ID)},
				{Label: "Delete Permanently", URL: fmt.Sprintf("/wp-admin/post.php?post=%s&action=delete", m.ID), Class: "submitdelete"},
				{Label: "View", URL: m.URL},
			},
		})
	}

	// Get total count
	allMedia, _, _ := h.media.List(ctx, &media.ListIn{})

	columns := []TableColumn{
		{ID: "file", Label: "File", Primary: true},
		{ID: "author", Label: "Author"},
		{ID: "uploaded", Label: "Uploaded to"},
		{ID: "date", Label: "Date", Sortable: true},
	}

	return render(h, c, "media", MediaLibraryData{
		Title:       "Media Library",
		User:        user,
		Menu:        h.buildMenu(c, "media"),
		Breadcrumbs: []Breadcrumb{{Label: "Media", URL: "/wp-admin/upload.php"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Items:       items,
		ViewMode:    mode,
		FilterType:  mediaType,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (len(allMedia) + perPage - 1) / perPage,
			TotalItems:  len(allMedia),
			PerPage:     perPage,
			BaseURL:     "/wp-admin/upload.php",
		},
		SearchQuery: search,
		Columns:     columns,
	})
}

// MediaNew renders the upload media page.
func (h *Handler) MediaNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	return render(h, c, "media", MediaLibraryData{
		Title:       "Upload New Media",
		User:        user,
		Menu:        h.buildMenu(c, "media-new"),
		Breadcrumbs: []Breadcrumb{{Label: "Media", URL: "/wp-admin/upload.php"}, {Label: "Add New"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		ViewMode:    "upload",
	})
}

// MediaEdit renders the media edit page.
func (h *Handler) MediaEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	mediaID := c.Query("post")
	if mediaID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/upload.php", http.StatusFound)
		return nil
	}

	m, err := h.media.GetByID(ctx, mediaID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/upload.php", http.StatusFound)
		return nil
	}

	uploader, _ := h.users.GetByID(ctx, m.UploaderID)

	thumbnailURL := m.URL
	if !strings.HasPrefix(m.MimeType, "image/") {
		thumbnailURL = "/wp-admin/images/media/default.png"
	}

	return render(h, c, "media-edit", MediaEditData{
		Title:        "Edit Media",
		User:         user,
		Menu:         h.buildMenu(c, "media"),
		Breadcrumbs:  []Breadcrumb{{Label: "Media", URL: "/wp-admin/upload.php"}, {Label: "Edit"}},
		SiteTitle:    h.getSiteTitle(c),
		SiteURL:      h.baseURL,
		Media:        m,
		Uploader:     uploader,
		ThumbnailURL: thumbnailURL,
		FileURL:      m.URL,
		FileSize:     formatFileSize(m.FileSize),
		Dimensions:   fmt.Sprintf("%dx%d", m.Width, m.Height),
		MimeType:     m.MimeType,
		AltText:      m.AltText,
		Caption:      m.Caption,
		Description:  m.Description,
	})
}

// CommentsList renders the comments list page.
func (h *Handler) CommentsList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Parse query params
	status := c.Query("comment_status")
	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := 20

	// Get comments
	params := &comments.ListIn{
		Status: status,
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
	// Note: comments.ListIn doesn't have Search field, filtering done by status only
	_ = search // search param not used currently

	commentList, _, _ := h.comments.List(ctx, params)

	// Get counts
	allComments, _, _ := h.comments.List(ctx, &comments.ListIn{})
	pendingCount := 0
	approvedCount := 0
	spamCount := 0
	trashCount := 0
	for _, cm := range allComments {
		switch cm.Status {
		case "pending":
			pendingCount++
		case "approved":
			approvedCount++
		case "spam":
			spamCount++
		case "trash":
			trashCount++
		}
	}

	// Build comment rows
	var commentRows []*CommentRow
	for _, cm := range commentList {
		var postTitle string
		post, _ := h.posts.GetByID(ctx, cm.PostID)
		if post != nil {
			postTitle = post.Title
		}

		commentRows = append(commentRows, &CommentRow{
			Comment:      cm,
			AuthorName:   cm.AuthorName,
			AuthorEmail:  cm.AuthorEmail,
			AuthorURL:    cm.AuthorURL,
			AuthorAvatar: gravatarURL(cm.AuthorEmail, 32),
			Post:         post,
			PostTitle:    postTitle,
			PostEditURL:  fmt.Sprintf("/wp-admin/post.php?post=%s&action=edit", cm.PostID),
			Pending:      cm.Status == "pending",
			RowActions: []RowAction{
				{Label: "Approve", URL: fmt.Sprintf("/wp-admin/comment.php?c=%s&action=approve", cm.ID)},
				{Label: "Reply", URL: "#", Class: "vim-r"},
				{Label: "Quick Edit", URL: "#", Class: "vim-q"},
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/comment.php?c=%s&action=edit", cm.ID)},
				{Label: "Spam", URL: fmt.Sprintf("/wp-admin/comment.php?c=%s&action=spam", cm.ID), Class: "vim-s"},
				{Label: "Trash", URL: fmt.Sprintf("/wp-admin/comment.php?c=%s&action=trash", cm.ID), Class: "submitdelete"},
			},
		})
	}

	// Build status tabs
	statusTabs := []StatusTab{
		{Status: "", Label: "All", Count: len(allComments), Active: status == "", URL: "/wp-admin/edit-comments.php"},
		{Status: "pending", Label: "Pending", Count: pendingCount, Active: status == "pending", URL: "/wp-admin/edit-comments.php?comment_status=pending"},
		{Status: "approved", Label: "Approved", Count: approvedCount, Active: status == "approved", URL: "/wp-admin/edit-comments.php?comment_status=approved"},
		{Status: "spam", Label: "Spam", Count: spamCount, Active: status == "spam", URL: "/wp-admin/edit-comments.php?comment_status=spam"},
		{Status: "trash", Label: "Trash", Count: trashCount, Active: status == "trash", URL: "/wp-admin/edit-comments.php?comment_status=trash"},
	}

	columns := []TableColumn{
		{ID: "author", Label: "Author", Primary: true},
		{ID: "comment", Label: "Comment"},
		{ID: "response", Label: "In Response To"},
		{ID: "date", Label: "Submitted On", Sortable: true},
	}

	bulkActions := []BulkAction{
		{Value: "approve", Label: "Approve"},
		{Value: "unapprove", Label: "Unapprove"},
		{Value: "spam", Label: "Mark as Spam"},
		{Value: "trash", Label: "Move to Trash"},
	}

	return render(h, c, "comments", CommentsListData{
		Title:       "Comments",
		User:        user,
		Menu:        h.buildMenu(c, "comments"),
		Breadcrumbs: []Breadcrumb{{Label: "Comments", URL: "/wp-admin/edit-comments.php"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Comments:    commentRows,
		Columns:     columns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (len(allComments) + perPage - 1) / perPage,
			TotalItems:  len(allComments),
			PerPage:     perPage,
			BaseURL:     "/wp-admin/edit-comments.php",
		},
		BulkActions: bulkActions,
		StatusTabs:  statusTabs,
		ActiveTab:   status,
		SearchQuery: search,
	})
}

// CommentEdit renders the comment edit page.
func (h *Handler) CommentEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	commentID := c.Query("c")
	if commentID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
		return nil
	}

	cm, err := h.comments.GetByID(ctx, commentID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
		return nil
	}

	post, _ := h.posts.GetByID(ctx, cm.PostID)

	statuses := []SelectOption{
		{Value: "pending", Label: "Pending", Selected: cm.Status == "pending"},
		{Value: "approved", Label: "Approved", Selected: cm.Status == "approved"},
		{Value: "spam", Label: "Spam", Selected: cm.Status == "spam"},
		{Value: "trash", Label: "Trash", Selected: cm.Status == "trash"},
	}

	return render(h, c, "comment-edit", CommentEditData{
		Title:         "Edit Comment",
		User:          user,
		Menu:          h.buildMenu(c, "comments"),
		Breadcrumbs:   []Breadcrumb{{Label: "Comments", URL: "/wp-admin/edit-comments.php"}, {Label: "Edit"}},
		SiteTitle:     h.getSiteTitle(c),
		SiteURL:       h.baseURL,
		Comment:       cm,
		AuthorName:    cm.AuthorName,
		AuthorEmail:   cm.AuthorEmail,
		AuthorURL:     cm.AuthorURL,
		Post:          post,
		PostTitle:     post.Title,
		Statuses:      statuses,
		CurrentStatus: cm.Status,
	})
}

// UsersList renders the users list page.
func (h *Handler) UsersList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Parse query params
	role := c.Query("role")
	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := 20

	// Get users
	params := &users.ListIn{
		Role:   role,
		Search: search,
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	userList, _, _ := h.users.List(ctx, params)

	// Get counts by role
	allUsers, _, _ := h.users.List(ctx, &users.ListIn{})
	adminCount := 0
	editorCount := 0
	authorCount := 0
	subscriberCount := 0
	for _, u := range allUsers {
		switch u.Role {
		case "administrator":
			adminCount++
		case "editor":
			editorCount++
		case "author":
			authorCount++
		case "subscriber":
			subscriberCount++
		}
	}

	// Build user rows
	var userRows []*UserRow
	for _, u := range userList {
		// Count posts by this user
		userPosts, _, _ := h.posts.List(ctx, &posts.ListIn{AuthorID: u.ID})

		roleDisplay := strings.Title(u.Role)

		userRows = append(userRows, &UserRow{
			User:        u,
			PostCount:   len(userPosts),
			AvatarURL:   gravatarURL(u.Email, 32),
			Role:        u.Role,
			RoleDisplay: roleDisplay,
			RowActions: []RowAction{
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/user-edit.php?user_id=%s", u.ID)},
				{Label: "Delete", URL: fmt.Sprintf("/wp-admin/users.php?action=delete&user=%s", u.ID), Class: "submitdelete"},
				{Label: "View", URL: fmt.Sprintf("/author/%s/", u.Slug)},
			},
		})
	}

	// Build role tabs
	roleTabs := []RoleTab{
		{Role: "", Label: "All", Count: len(allUsers), Active: role == "", URL: "/wp-admin/users.php"},
		{Role: "administrator", Label: "Administrator", Count: adminCount, Active: role == "administrator", URL: "/wp-admin/users.php?role=administrator"},
		{Role: "editor", Label: "Editor", Count: editorCount, Active: role == "editor", URL: "/wp-admin/users.php?role=editor"},
		{Role: "author", Label: "Author", Count: authorCount, Active: role == "author", URL: "/wp-admin/users.php?role=author"},
		{Role: "subscriber", Label: "Subscriber", Count: subscriberCount, Active: role == "subscriber", URL: "/wp-admin/users.php?role=subscriber"},
	}

	columns := []TableColumn{
		{ID: "username", Label: "Username", Sortable: true, Primary: true},
		{ID: "name", Label: "Name", Sortable: true},
		{ID: "email", Label: "Email", Sortable: true},
		{ID: "role", Label: "Role"},
		{ID: "posts", Label: "Posts", Class: "num"},
	}

	bulkActions := []BulkAction{
		{Value: "delete", Label: "Delete"},
	}

	return render(h, c, "users", UsersListData{
		Title:       "Users",
		User:        user,
		Menu:        h.buildMenu(c, "users"),
		Breadcrumbs: []Breadcrumb{{Label: "Users", URL: "/wp-admin/users.php"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Users:       userRows,
		Columns:     columns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (len(allUsers) + perPage - 1) / perPage,
			TotalItems:  len(allUsers),
			PerPage:     perPage,
			BaseURL:     "/wp-admin/users.php",
		},
		BulkActions: bulkActions,
		RoleTabs:    roleTabs,
		ActiveRole:  role,
		SearchQuery: search,
	})
}

// UserNew renders the add new user page.
func (h *Handler) UserNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	roles := []SelectOption{
		{Value: "subscriber", Label: "Subscriber"},
		{Value: "contributor", Label: "Contributor"},
		{Value: "author", Label: "Author"},
		{Value: "editor", Label: "Editor"},
		{Value: "administrator", Label: "Administrator"},
	}

	return render(h, c, "user-edit", UserEditData{
		Title:        "Add New User",
		User:         user,
		Menu:         h.buildMenu(c, "user-new"),
		Breadcrumbs:  []Breadcrumb{{Label: "Users", URL: "/wp-admin/users.php"}, {Label: "Add New"}},
		SiteTitle:    h.getSiteTitle(c),
		SiteURL:      h.baseURL,
		IsNew:        true,
		Roles:        roles,
		CurrentRole:  "subscriber",
		ShowPassword: true,
	})
}

// UserEdit renders the user edit page.
func (h *Handler) UserEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	userID := c.Query("user_id")
	if userID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/users.php", http.StatusFound)
		return nil
	}

	editUser, err := h.users.GetByID(ctx, userID)
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/users.php", http.StatusFound)
		return nil
	}

	roles := []SelectOption{
		{Value: "subscriber", Label: "Subscriber", Selected: editUser.Role == "subscriber"},
		{Value: "contributor", Label: "Contributor", Selected: editUser.Role == "contributor"},
		{Value: "author", Label: "Author", Selected: editUser.Role == "author"},
		{Value: "editor", Label: "Editor", Selected: editUser.Role == "editor"},
		{Value: "administrator", Label: "Administrator", Selected: editUser.Role == "administrator"},
	}

	return render(h, c, "user-edit", UserEditData{
		Title:       "Edit User",
		User:        user,
		Menu:        h.buildMenu(c, "users"),
		Breadcrumbs: []Breadcrumb{{Label: "Users", URL: "/wp-admin/users.php"}, {Label: "Edit User"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		EditUser:    editUser,
		IsNew:       false,
		IsSelf:      user.ID == editUser.ID,
		Roles:       roles,
		CurrentRole: editUser.Role,
		AvatarURL:   gravatarURL(editUser.Email, 96),
	})
}

// Profile renders the user profile page.
func (h *Handler) Profile(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	colorSchemes := []ColorScheme{
		{ID: "fresh", Name: "Default", Colors: []string{"#1d2327", "#2c3338", "#2271b1", "#72aee6"}},
		{ID: "light", Name: "Light", Colors: []string{"#e5e5e5", "#999", "#d64e07", "#04a4cc"}},
		{ID: "modern", Name: "Modern", Colors: []string{"#1e1e1e", "#3858e9", "#33f078", "#3858e9"}},
		{ID: "blue", Name: "Blue", Colors: []string{"#096484", "#4796b3", "#52accc", "#74b6ce"}},
		{ID: "coffee", Name: "Coffee", Colors: []string{"#46403c", "#59524c", "#c7a589", "#9ea476"}},
		{ID: "ectoplasm", Name: "Ectoplasm", Colors: []string{"#413256", "#523f6d", "#a3b745", "#d46f15"}},
		{ID: "midnight", Name: "Midnight", Colors: []string{"#25282b", "#363b3f", "#69a8bb", "#e14d43"}},
		{ID: "ocean", Name: "Ocean", Colors: []string{"#627c83", "#738e96", "#9ebaa0", "#aa9d88"}},
		{ID: "sunrise", Name: "Sunrise", Colors: []string{"#b43c38", "#cf4944", "#dd823b", "#ccaf0b"}},
	}

	return render(h, c, "profile", ProfileData{
		Title:         "Profile",
		User:          user,
		Menu:          h.buildMenu(c, "profile"),
		Breadcrumbs:   []Breadcrumb{{Label: "Users", URL: "/wp-admin/users.php"}, {Label: "Profile"}},
		SiteTitle:     h.getSiteTitle(c),
		SiteURL:       h.baseURL,
		AvatarURL:     gravatarURL(user.Email, 96),
		ColorSchemes:  colorSchemes,
		CurrentScheme: "fresh",
		AdminBarFront: true,
	})
}

// CategoriesList renders the categories list page.
func (h *Handler) CategoriesList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := 20

	// Get categories
	catList, _, _ := h.categories.List(ctx, &categories.ListIn{Search: search})

	// Build category rows
	var catRows []*TaxonomyRow
	parentMap := make(map[string]string)
	for _, cat := range catList {
		if cat.ParentID != "" {
			if parent, _ := h.categories.GetByID(ctx, cat.ParentID); parent != nil {
				parentMap[cat.ID] = parent.Name
			}
		}
	}

	for _, cat := range catList {
		count := 0
		if postsList, _, err := h.posts.List(ctx, &posts.ListIn{CategoryID: cat.ID}); err == nil {
			count = len(postsList)
		}

		catRows = append(catRows, &TaxonomyRow{
			ID:          cat.ID,
			Name:        cat.Name,
			Slug:        cat.Slug,
			Description: cat.Description,
			Parent:      cat.ParentID,
			ParentName:  parentMap[cat.ID],
			Count:       count,
			RowActions: []RowAction{
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/edit-tags.php?action=edit&taxonomy=category&tag_ID=%s", cat.ID)},
				{Label: "Quick Edit", URL: "#", Class: "editinline"},
				{Label: "Delete", URL: fmt.Sprintf("/wp-admin/edit-tags.php?action=delete&taxonomy=category&tag_ID=%s", cat.ID), Class: "submitdelete"},
				{Label: "View", URL: fmt.Sprintf("/category/%s/", cat.Slug)},
			},
		})
	}

	// Build parent terms for add form
	var parentTerms []*TaxonomyRow
	for _, cat := range catList {
		parentTerms = append(parentTerms, &TaxonomyRow{
			ID:   cat.ID,
			Name: cat.Name,
		})
	}

	columns := []TableColumn{
		{ID: "name", Label: "Name", Sortable: true, Primary: true},
		{ID: "description", Label: "Description"},
		{ID: "slug", Label: "Slug"},
		{ID: "posts", Label: "Count", Class: "num", Sortable: true},
	}

	bulkActions := []BulkAction{
		{Value: "delete", Label: "Delete"},
	}

	return render(h, c, "categories", TaxonomyListData{
		Title:       "Categories",
		User:        user,
		Menu:        h.buildMenu(c, "categories"),
		Breadcrumbs: []Breadcrumb{{Label: "Posts", URL: "/wp-admin/edit.php"}, {Label: "Categories"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Taxonomy:    "category",
		TaxLabel:    "Categories",
		Terms:       catRows,
		Columns:     columns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (len(catList) + perPage - 1) / perPage,
			TotalItems:  len(catList),
			PerPage:     perPage,
			BaseURL:     "/wp-admin/edit-tags.php?taxonomy=category",
		},
		BulkActions: bulkActions,
		SearchQuery: search,
		ParentTerms: parentTerms,
		ShowParent:  true,
	})
}

// TagsList renders the tags list page.
func (h *Handler) TagsList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	search := c.Query("s")
	page := parseIntDefault(c.Query("paged"), 1)
	perPage := 20

	// Get tags
	tagList, _, _ := h.tags.List(ctx, &tags.ListIn{Search: search})

	// Build tag rows
	var tagRows []*TaxonomyRow
	for _, tag := range tagList {
		count := 0
		if postsList, _, err := h.posts.List(ctx, &posts.ListIn{TagID: tag.ID}); err == nil {
			count = len(postsList)
		}

		tagRows = append(tagRows, &TaxonomyRow{
			ID:          tag.ID,
			Name:        tag.Name,
			Slug:        tag.Slug,
			Description: tag.Description,
			Count:       count,
			RowActions: []RowAction{
				{Label: "Edit", URL: fmt.Sprintf("/wp-admin/edit-tags.php?action=edit&taxonomy=post_tag&tag_ID=%s", tag.ID)},
				{Label: "Quick Edit", URL: "#", Class: "editinline"},
				{Label: "Delete", URL: fmt.Sprintf("/wp-admin/edit-tags.php?action=delete&taxonomy=post_tag&tag_ID=%s", tag.ID), Class: "submitdelete"},
				{Label: "View", URL: fmt.Sprintf("/tag/%s/", tag.Slug)},
			},
		})
	}

	columns := []TableColumn{
		{ID: "name", Label: "Name", Sortable: true, Primary: true},
		{ID: "description", Label: "Description"},
		{ID: "slug", Label: "Slug"},
		{ID: "posts", Label: "Count", Class: "num", Sortable: true},
	}

	bulkActions := []BulkAction{
		{Value: "delete", Label: "Delete"},
	}

	return render(h, c, "tags", TaxonomyListData{
		Title:       "Tags",
		User:        user,
		Menu:        h.buildMenu(c, "tags"),
		Breadcrumbs: []Breadcrumb{{Label: "Posts", URL: "/wp-admin/edit.php"}, {Label: "Tags"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Taxonomy:    "post_tag",
		TaxLabel:    "Tags",
		Terms:       tagRows,
		Columns:     columns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  (len(tagList) + perPage - 1) / perPage,
			TotalItems:  len(tagList),
			PerPage:     perPage,
			BaseURL:     "/wp-admin/edit-tags.php?taxonomy=post_tag",
		},
		BulkActions: bulkActions,
		SearchQuery: search,
		ShowParent:  false,
	})
}

// TaxonomyList routes to the correct taxonomy page.
func (h *Handler) TaxonomyList(c *mizu.Ctx) error {
	taxonomy := c.Query("taxonomy")
	if taxonomy == "post_tag" {
		return h.TagsList(c)
	}
	return h.CategoriesList(c)
}

// MenusPage renders the menus management page.
func (h *Handler) MenusPage(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Get all menus
	menuList, _ := h.menus.ListMenus(ctx)

	// Get active menu
	menuID := c.Query("menu")
	var activeMenu *menus.Menu
	if menuID != "" {
		activeMenu, _ = h.menus.GetMenu(ctx, menuID)
	} else if len(menuList) > 0 {
		activeMenu = menuList[0]
	}

	// Get menu items for active menu
	var menuItems []*MenuItemView
	if activeMenu != nil {
		menuItems = buildMenuItemTree(activeMenu.Items)
	}

	// Get available items
	pageList, _, _ := h.pages.List(ctx, &pages.ListIn{})
	postList, _, _ := h.posts.List(ctx, &posts.ListIn{})
	catList, _, _ := h.categories.List(ctx, &categories.ListIn{})
	tagList, _, _ := h.tags.List(ctx, &tags.ListIn{})

	availableItems := AvailableMenuItems{
		Pages:       pageList,
		Posts:       postList,
		Categories:  catList,
		Tags:        tagList,
		CustomLinks: true,
	}

	// Menu locations
	locations := []MenuLocation{
		{Name: "primary", Description: "Primary Menu"},
		{Name: "footer", Description: "Footer Menu"},
		{Name: "social", Description: "Social Links Menu"},
	}

	return render(h, c, "menus", MenusData{
		Title:          "Menus",
		User:           user,
		Menu:           h.buildMenu(c, "menus"),
		Breadcrumbs:    []Breadcrumb{{Label: "Appearance"}, {Label: "Menus"}},
		SiteTitle:      h.getSiteTitle(c),
		SiteURL:        h.baseURL,
		Menus:          menuList,
		ActiveMenu:     activeMenu,
		MenuItems:      menuItems,
		Locations:      locations,
		AvailableItems: availableItems,
		IsNew:          c.Query("action") == "edit" && menuID == "",
	})
}

// buildMenuItemTree builds a hierarchical menu item list.
func buildMenuItemTree(items []*menus.MenuItem) []*MenuItemView {
	// Build parent map
	childrenMap := make(map[string][]*menus.MenuItem)
	var rootItems []*menus.MenuItem

	for _, item := range items {
		if item.ParentID == "" {
			rootItems = append(rootItems, item)
		} else {
			childrenMap[item.ParentID] = append(childrenMap[item.ParentID], item)
		}
	}

	var buildTree func([]*menus.MenuItem, int) []*MenuItemView
	buildTree = func(items []*menus.MenuItem, depth int) []*MenuItemView {
		var result []*MenuItemView
		for _, item := range items {
			typeLabel := "Custom Link"
			switch item.LinkType {
			case "post":
				typeLabel = "Post"
			case "page":
				typeLabel = "Page"
			case "category":
				typeLabel = "Category"
			case "tag":
				typeLabel = "Tag"
			}

			view := &MenuItemView{
				MenuItem:  item,
				TypeLabel: typeLabel,
				Depth:     depth,
			}
			if children, ok := childrenMap[item.ID]; ok {
				view.Children = buildTree(children, depth+1)
			}
			result = append(result, view)
		}
		return result
	}

	return buildTree(rootItems, 0)
}

// SettingsGeneral renders the general settings page.
func (h *Handler) SettingsGeneral(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Get settings
	settingsMap := make(map[string]string)
	keys := []string{"site_title", "tagline", "site_url", "admin_email", "membership", "default_role", "site_language", "timezone", "date_format", "time_format", "week_starts"}
	for _, key := range keys {
		if val, err := h.settings.Get(ctx, key); err == nil {
			settingsMap[key] = val.Value
		}
	}

	// Build options
	options := make(map[string][]SelectOption)

	options["default_role"] = []SelectOption{
		{Value: "subscriber", Label: "Subscriber"},
		{Value: "contributor", Label: "Contributor"},
		{Value: "author", Label: "Author"},
		{Value: "editor", Label: "Editor"},
		{Value: "administrator", Label: "Administrator"},
	}

	options["week_starts"] = []SelectOption{
		{Value: "0", Label: "Sunday"},
		{Value: "1", Label: "Monday"},
		{Value: "2", Label: "Tuesday"},
		{Value: "3", Label: "Wednesday"},
		{Value: "4", Label: "Thursday"},
		{Value: "5", Label: "Friday"},
		{Value: "6", Label: "Saturday"},
	}

	dateFormats := []SelectOption{
		{Value: "F j, Y", Label: time.Now().Format("January 2, 2006")},
		{Value: "Y-m-d", Label: time.Now().Format("2006-01-02")},
		{Value: "m/d/Y", Label: time.Now().Format("01/02/2006")},
		{Value: "d/m/Y", Label: time.Now().Format("02/01/2006")},
	}

	timeFormats := []SelectOption{
		{Value: "g:i a", Label: time.Now().Format("3:04 pm")},
		{Value: "g:i A", Label: time.Now().Format("3:04 PM")},
		{Value: "H:i", Label: time.Now().Format("15:04")},
	}

	return render(h, c, "settings-general", SettingsData{
		Title:       "General Settings",
		User:        user,
		Menu:        h.buildMenu(c, "settings-general"),
		Breadcrumbs: []Breadcrumb{{Label: "Settings", URL: "/wp-admin/options-general.php"}, {Label: "General"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Section:     "general",
		Settings:    settingsMap,
		Options:     options,
		DateFormats: dateFormats,
		TimeFormats: timeFormats,
	})
}

// SettingsWriting renders the writing settings page.
func (h *Handler) SettingsWriting(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	settingsMap := make(map[string]string)
	keys := []string{"default_category", "default_post_format", "ping_sites"}
	for _, key := range keys {
		if val, err := h.settings.Get(ctx, key); err == nil {
			settingsMap[key] = val.Value
		}
	}

	// Get categories for default category option
	catList, _, _ := h.categories.List(ctx, &categories.ListIn{})
	options := make(map[string][]SelectOption)
	var catOptions []SelectOption
	for _, cat := range catList {
		catOptions = append(catOptions, SelectOption{Value: cat.ID, Label: cat.Name})
	}
	options["default_category"] = catOptions

	options["default_post_format"] = []SelectOption{
		{Value: "standard", Label: "Standard"},
		{Value: "aside", Label: "Aside"},
		{Value: "gallery", Label: "Gallery"},
		{Value: "link", Label: "Link"},
		{Value: "image", Label: "Image"},
		{Value: "quote", Label: "Quote"},
		{Value: "status", Label: "Status"},
		{Value: "video", Label: "Video"},
		{Value: "audio", Label: "Audio"},
		{Value: "chat", Label: "Chat"},
	}

	return render(h, c, "settings-writing", SettingsData{
		Title:       "Writing Settings",
		User:        user,
		Menu:        h.buildMenu(c, "settings-writing"),
		Breadcrumbs: []Breadcrumb{{Label: "Settings"}, {Label: "Writing"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Section:     "writing",
		Settings:    settingsMap,
		Options:     options,
	})
}

// SettingsReading renders the reading settings page.
func (h *Handler) SettingsReading(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	settingsMap := make(map[string]string)
	keys := []string{"show_on_front", "page_on_front", "page_for_posts", "posts_per_page", "posts_per_rss", "rss_use_excerpt", "blog_public"}
	for _, key := range keys {
		if val, err := h.settings.Get(ctx, key); err == nil {
			settingsMap[key] = val.Value
		}
	}

	// Get pages for front page options
	pageList, _, _ := h.pages.List(ctx, &pages.ListIn{})
	options := make(map[string][]SelectOption)
	var pageOptions []SelectOption
	pageOptions = append(pageOptions, SelectOption{Value: "", Label: "— Select —"})
	for _, pg := range pageList {
		pageOptions = append(pageOptions, SelectOption{Value: pg.ID, Label: pg.Title})
	}
	options["page_on_front"] = pageOptions
	options["page_for_posts"] = pageOptions

	return render(h, c, "settings-reading", SettingsData{
		Title:       "Reading Settings",
		User:        user,
		Menu:        h.buildMenu(c, "settings-reading"),
		Breadcrumbs: []Breadcrumb{{Label: "Settings"}, {Label: "Reading"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Section:     "reading",
		Settings:    settingsMap,
		Options:     options,
	})
}

// SettingsDiscussion renders the discussion settings page.
func (h *Handler) SettingsDiscussion(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	settingsMap := make(map[string]string)
	keys := []string{
		"default_pingback_flag", "default_ping_status", "default_comment_status",
		"require_name_email", "comment_registration", "close_comments_for_old_posts",
		"close_comments_days_old", "thread_comments", "thread_comments_depth",
		"page_comments", "comments_per_page", "default_comments_page", "comment_order",
		"comments_notify", "moderation_notify", "comment_moderation", "comment_previously_approved",
		"comment_max_links", "moderation_keys", "disallowed_keys",
		"show_avatars", "avatar_rating", "avatar_default",
	}
	for _, key := range keys {
		if val, err := h.settings.Get(ctx, key); err == nil {
			settingsMap[key] = val.Value
		}
	}

	options := make(map[string][]SelectOption)

	options["thread_comments_depth"] = []SelectOption{
		{Value: "2", Label: "2 levels"},
		{Value: "3", Label: "3 levels"},
		{Value: "4", Label: "4 levels"},
		{Value: "5", Label: "5 levels"},
		{Value: "6", Label: "6 levels"},
		{Value: "7", Label: "7 levels"},
		{Value: "8", Label: "8 levels"},
		{Value: "9", Label: "9 levels"},
		{Value: "10", Label: "10 levels"},
	}

	options["default_comments_page"] = []SelectOption{
		{Value: "newest", Label: "last"},
		{Value: "oldest", Label: "first"},
	}

	options["comment_order"] = []SelectOption{
		{Value: "asc", Label: "older"},
		{Value: "desc", Label: "newer"},
	}

	options["avatar_rating"] = []SelectOption{
		{Value: "G", Label: "G — Suitable for all audiences"},
		{Value: "PG", Label: "PG — Possibly offensive, usually for audiences 13 and above"},
		{Value: "R", Label: "R — Intended for adult audiences above 17"},
		{Value: "X", Label: "X — Even more mature than above"},
	}

	options["avatar_default"] = []SelectOption{
		{Value: "mystery", Label: "Mystery Person"},
		{Value: "blank", Label: "Blank"},
		{Value: "gravatar_default", Label: "Gravatar Logo"},
		{Value: "identicon", Label: "Identicon (Generated)"},
		{Value: "wavatar", Label: "Wavatar (Generated)"},
		{Value: "monsterid", Label: "MonsterID (Generated)"},
		{Value: "retro", Label: "Retro (Generated)"},
		{Value: "robohash", Label: "RoboHash (Generated)"},
	}

	return render(h, c, "settings-discussion", SettingsData{
		Title:       "Discussion Settings",
		User:        user,
		Menu:        h.buildMenu(c, "settings-discussion"),
		Breadcrumbs: []Breadcrumb{{Label: "Settings"}, {Label: "Discussion"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Section:     "discussion",
		Settings:    settingsMap,
		Options:     options,
	})
}

// SettingsMedia renders the media settings page.
func (h *Handler) SettingsMedia(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	settingsMap := make(map[string]string)
	keys := []string{
		"thumbnail_size_w", "thumbnail_size_h", "thumbnail_crop",
		"medium_size_w", "medium_size_h",
		"large_size_w", "large_size_h",
		"uploads_use_yearmonth_folders",
	}
	for _, key := range keys {
		if val, err := h.settings.Get(ctx, key); err == nil {
			settingsMap[key] = val.Value
		}
	}

	return render(h, c, "settings-media", SettingsData{
		Title:       "Media Settings",
		User:        user,
		Menu:        h.buildMenu(c, "settings-media"),
		Breadcrumbs: []Breadcrumb{{Label: "Settings"}, {Label: "Media"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Section:     "media",
		Settings:    settingsMap,
	})
}

// SettingsPermalinks renders the permalinks settings page.
func (h *Handler) SettingsPermalinks(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	settingsMap := make(map[string]string)
	keys := []string{"permalink_structure", "category_base", "tag_base"}
	for _, key := range keys {
		if val, err := h.settings.Get(ctx, key); err == nil {
			settingsMap[key] = val.Value
		}
	}

	return render(h, c, "settings-permalinks", SettingsData{
		Title:       "Permalink Settings",
		User:        user,
		Menu:        h.buildMenu(c, "settings-permalinks"),
		Breadcrumbs: []Breadcrumb{{Label: "Settings"}, {Label: "Permalinks"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Section:     "permalinks",
		Settings:    settingsMap,
	})
}

// Themes renders the themes page.
func (h *Handler) Themes(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Get active theme from settings
	activeTheme := "default"
	if setting, err := h.settings.Get(ctx, "active_theme"); err == nil && setting.Value != "" {
		activeTheme = setting.Value
	}

	// Get available themes from assets
	themeList, err := assets.ListThemes()
	if err != nil {
		themeList = []*assets.ThemeJSON{}
	}

	// Convert to ThemeInfo for template
	themes := make([]*ThemeInfo, 0, len(themeList))
	for _, t := range themeList {
		themes = append(themes, &ThemeInfo{
			Name:        t.Name,
			Slug:        t.Slug,
			Version:     t.Version,
			Description: t.Description,
			Author:      t.Author.Name,
			AuthorURL:   t.Author.URL,
			Screenshot:  "/themes/" + t.Slug + "/assets/images/screenshot.png",
			Active:      t.Slug == activeTheme,
		})
	}

	return render(h, c, "themes", ThemesData{
		Title:       "Themes",
		User:        user,
		Menu:        h.buildMenu(c, "themes"),
		Breadcrumbs: []Breadcrumb{{Label: "Appearance"}, {Label: "Themes"}},
		SiteTitle:   h.getSiteTitle(c),
		SiteURL:     h.baseURL,
		Themes:      themes,
		ActiveTheme: activeTheme,
	})
}

// ThemeActivate activates a theme.
func (h *Handler) ThemeActivate(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	themeSlug := c.Request().FormValue("theme")

	if themeSlug == "" {
		return h.redirectWithError(c, "/wp-admin/themes.php", "No theme specified")
	}

	// Verify theme exists
	if _, err := assets.GetTheme(themeSlug); err != nil {
		return h.redirectWithError(c, "/wp-admin/themes.php", "Theme not found")
	}

	// Save active theme setting
	_, err := h.settings.Set(ctx, &settings.SetIn{
		Key:       "active_theme",
		Value:     themeSlug,
		ValueType: "string",
		GroupName: "appearance",
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/themes.php", "Failed to activate theme")
	}

	return h.redirectWithSuccess(c, "/wp-admin/themes.php", "Theme activated successfully")
}

// ============================================================
// POST Handlers - Form Submissions
// ============================================================

// PostSave handles creating or updating a post.
func (h *Handler) PostSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit.php", "Invalid form data")
	}

	postID := c.Request().FormValue("post_ID")
	action := c.Request().FormValue("action")
	title := c.Request().FormValue("post_title")
	content := c.Request().FormValue("content")
	excerpt := c.Request().FormValue("excerpt")
	slug := c.Request().FormValue("post_name")
	status := c.Request().FormValue("post_status")
	visibility := c.Request().FormValue("visibility")
	categoryIDs := c.Request().Form["post_category[]"]

	// Filter out "0" from category IDs
	var validCatIDs []string
	for _, id := range categoryIDs {
		if id != "0" && id != "" {
			validCatIDs = append(validCatIDs, id)
		}
	}

	// Parse tags from textarea
	tagsInput := c.Request().FormValue("tax_input[post_tag]")
	var tagNames []string
	if tagsInput != "" {
		for _, name := range strings.Split(tagsInput, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				tagNames = append(tagNames, name)
			}
		}
	}

	if action == "create" || postID == "" {
		// Create new post
		post, err := h.posts.Create(ctx, user.ID, &posts.CreateIn{
			Title:       title,
			Slug:        slug,
			Content:     content,
			Excerpt:     excerpt,
			Status:      status,
			Visibility:  visibility,
			CategoryIDs: validCatIDs,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/post-new.php", "Failed to create post: "+err.Error())
		}

		// Handle tags - create if not exists
		if len(tagNames) > 0 {
			tagIDs, err := h.ensureTagsByName(ctx, tagNames)
			if err == nil && len(tagIDs) > 0 {
				_ = h.posts.SetTags(ctx, post.ID, tagIDs)
			}
		}

		http.Redirect(c.Writer(), c.Request(), "/wp-admin/post.php?post="+post.ID+"&action=edit&message=1", http.StatusFound)
		return nil
	}

	// Update existing post
	_, err := h.posts.Update(ctx, postID, &posts.UpdateIn{
		Title:      &title,
		Slug:       &slug,
		Content:    &content,
		Excerpt:    &excerpt,
		Status:     &status,
		Visibility: &visibility,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/post.php?post="+postID+"&action=edit", "Failed to update post: "+err.Error())
	}

	// Update categories
	if err := h.posts.SetCategories(ctx, postID, validCatIDs); err != nil {
		// Log but don't fail
	}

	// Handle tags
	if len(tagNames) > 0 {
		tagIDs, err := h.ensureTagsByName(ctx, tagNames)
		if err == nil {
			_ = h.posts.SetTags(ctx, postID, tagIDs)
		}
	} else {
		_ = h.posts.SetTags(ctx, postID, []string{})
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/post.php?post="+postID+"&action=edit&message=1", http.StatusFound)
	return nil
}

// ensureTagsByName creates tags if they don't exist and returns their IDs.
func (h *Handler) ensureTagsByName(ctx context.Context, names []string) ([]string, error) {
	var tagIDs []string
	for _, name := range names {
		// Try to find existing tag
		tag, err := h.tags.GetBySlug(ctx, slugify(name))
		if err != nil {
			// Create new tag
			tag, err = h.tags.Create(ctx, &tags.CreateIn{
				Name: name,
			})
			if err != nil {
				continue
			}
		}
		tagIDs = append(tagIDs, tag.ID)
	}
	return tagIDs, nil
}

// slugify converts a string to a URL-friendly slug.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

// PostTrash moves a post to trash.
func (h *Handler) PostTrash(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	postID := c.Query("post")
	if postID == "" {
		postID = c.Request().FormValue("post")
	}

	if postID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	// Update status to trash
	status := "trash"
	_, err := h.posts.Update(ctx, postID, &posts.UpdateIn{
		Status: &status,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/edit.php", "Failed to trash post")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?trashed=1", http.StatusFound)
	return nil
}

// PostRestore restores a post from trash.
func (h *Handler) PostRestore(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	postID := c.Query("post")

	if postID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	// Update status to draft
	status := "draft"
	_, err := h.posts.Update(ctx, postID, &posts.UpdateIn{
		Status: &status,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/edit.php?post_status=trash", "Failed to restore post")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?untrashed=1", http.StatusFound)
	return nil
}

// PostDelete permanently deletes a post.
func (h *Handler) PostDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	postID := c.Query("post")

	if postID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	if err := h.posts.Delete(ctx, postID); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit.php?post_status=trash", "Failed to delete post")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?deleted=1", http.StatusFound)
	return nil
}

// BulkPostAction handles bulk actions on posts.
func (h *Handler) BulkPostAction(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	action := c.Request().FormValue("action")
	if action == "-1" {
		action = c.Request().FormValue("action2")
	}

	postIDs := c.Request().Form["post[]"]
	if len(postIDs) == 0 {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
		return nil
	}

	switch action {
	case "trash":
		status := "trash"
		for _, id := range postIDs {
			h.posts.Update(ctx, id, &posts.UpdateIn{Status: &status})
		}
		http.Redirect(c.Writer(), c.Request(), fmt.Sprintf("/wp-admin/edit.php?trashed=%d", len(postIDs)), http.StatusFound)
	case "untrash":
		status := "draft"
		for _, id := range postIDs {
			h.posts.Update(ctx, id, &posts.UpdateIn{Status: &status})
		}
		http.Redirect(c.Writer(), c.Request(), fmt.Sprintf("/wp-admin/edit.php?untrashed=%d", len(postIDs)), http.StatusFound)
	case "delete":
		for _, id := range postIDs {
			h.posts.Delete(ctx, id)
		}
		http.Redirect(c.Writer(), c.Request(), fmt.Sprintf("/wp-admin/edit.php?deleted=%d", len(postIDs)), http.StatusFound)
	default:
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php", http.StatusFound)
	}

	return nil
}

// PageSave handles creating or updating a page.
func (h *Handler) PageSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit.php?post_type=page", "Invalid form data")
	}

	pageID := c.Request().FormValue("post_ID")
	action := c.Request().FormValue("action")
	title := c.Request().FormValue("post_title")
	content := c.Request().FormValue("content")
	slug := c.Request().FormValue("post_name")
	status := c.Request().FormValue("post_status")
	visibility := c.Request().FormValue("visibility")
	parentID := c.Request().FormValue("parent_id")
	menuOrder := parseIntDefault(c.Request().FormValue("menu_order"), 0)
	template := c.Request().FormValue("page_template")

	if action == "create" || pageID == "" {
		// Create new page
		page, err := h.pages.Create(ctx, user.ID, &pages.CreateIn{
			Title:      title,
			Slug:       slug,
			Content:    content,
			ParentID:   parentID,
			Status:     status,
			Visibility: visibility,
			Template:   template,
			SortOrder:  menuOrder,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/post-new.php?post_type=page", "Failed to create page: "+err.Error())
		}

		http.Redirect(c.Writer(), c.Request(), "/wp-admin/post.php?post="+page.ID+"&action=edit&post_type=page&message=1", http.StatusFound)
		return nil
	}

	// Update existing page
	_, err := h.pages.Update(ctx, pageID, &pages.UpdateIn{
		Title:      &title,
		Slug:       &slug,
		Content:    &content,
		ParentID:   &parentID,
		Status:     &status,
		Visibility: &visibility,
		Template:   &template,
		SortOrder:  &menuOrder,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/post.php?post="+pageID+"&action=edit&post_type=page", "Failed to update page: "+err.Error())
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/post.php?post="+pageID+"&action=edit&post_type=page&message=1", http.StatusFound)
	return nil
}

// PageTrash moves a page to trash.
func (h *Handler) PageTrash(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	pageID := c.Query("post")

	if pageID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?post_type=page", http.StatusFound)
		return nil
	}

	status := "trash"
	_, err := h.pages.Update(ctx, pageID, &pages.UpdateIn{
		Status: &status,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/edit.php?post_type=page", "Failed to trash page")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit.php?post_type=page&trashed=1", http.StatusFound)
	return nil
}

// CategorySave handles creating or updating a category.
func (h *Handler) CategorySave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=category", "Invalid form data")
	}

	catID := c.Request().FormValue("tag_ID")
	name := c.Request().FormValue("tag-name")
	slug := c.Request().FormValue("slug")
	parentID := c.Request().FormValue("parent")
	description := c.Request().FormValue("description")

	if parentID == "-1" {
		parentID = ""
	}

	if catID == "" {
		// Create new category
		_, err := h.categories.Create(ctx, &categories.CreateIn{
			Name:        name,
			Slug:        slug,
			ParentID:    parentID,
			Description: description,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=category", "Failed to create category: "+err.Error())
		}
	} else {
		// Update existing category
		_, err := h.categories.Update(ctx, catID, &categories.UpdateIn{
			Name:        &name,
			Slug:        &slug,
			ParentID:    &parentID,
			Description: &description,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=category", "Failed to update category: "+err.Error())
		}
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-tags.php?taxonomy=category&message=1", http.StatusFound)
	return nil
}

// CategoryDelete deletes a category.
func (h *Handler) CategoryDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	catID := c.Query("tag_ID")

	if catID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-tags.php?taxonomy=category", http.StatusFound)
		return nil
	}

	if err := h.categories.Delete(ctx, catID); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=category", "Failed to delete category")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-tags.php?taxonomy=category&deleted=1", http.StatusFound)
	return nil
}

// TagSave handles creating or updating a tag.
func (h *Handler) TagSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=post_tag", "Invalid form data")
	}

	tagID := c.Request().FormValue("tag_ID")
	name := c.Request().FormValue("tag-name")
	slug := c.Request().FormValue("slug")
	description := c.Request().FormValue("description")

	if tagID == "" {
		// Create new tag
		_, err := h.tags.Create(ctx, &tags.CreateIn{
			Name:        name,
			Slug:        slug,
			Description: description,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=post_tag", "Failed to create tag: "+err.Error())
		}
	} else {
		// Update existing tag
		_, err := h.tags.Update(ctx, tagID, &tags.UpdateIn{
			Name:        &name,
			Slug:        &slug,
			Description: &description,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=post_tag", "Failed to update tag: "+err.Error())
		}
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-tags.php?taxonomy=post_tag&message=1", http.StatusFound)
	return nil
}

// TagDelete deletes a tag.
func (h *Handler) TagDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	tagID := c.Query("tag_ID")

	if tagID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-tags.php?taxonomy=post_tag", http.StatusFound)
		return nil
	}

	if err := h.tags.Delete(ctx, tagID); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-tags.php?taxonomy=post_tag", "Failed to delete tag")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-tags.php?taxonomy=post_tag&deleted=1", http.StatusFound)
	return nil
}

// CommentAction handles comment moderation actions (approve, spam, trash).
func (h *Handler) CommentAction(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	action := c.Query("action")
	commentID := c.Query("c")

	if commentID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
		return nil
	}

	var err error
	switch action {
	case "approve", "approvecomment":
		_, err = h.comments.Approve(ctx, commentID)
	case "unapprove", "unapprovecomment":
		status := "pending"
		_, err = h.comments.Update(ctx, commentID, &comments.UpdateIn{Status: &status})
	case "spam", "spamcomment":
		_, err = h.comments.MarkAsSpam(ctx, commentID)
	case "trash", "trashcomment":
		status := "trash"
		_, err = h.comments.Update(ctx, commentID, &comments.UpdateIn{Status: &status})
	case "untrash":
		status := "pending"
		_, err = h.comments.Update(ctx, commentID, &comments.UpdateIn{Status: &status})
	case "delete":
		err = h.comments.Delete(ctx, commentID)
	}

	if err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-comments.php", "Failed to perform action")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
	return nil
}

// CommentSave handles updating a comment.
func (h *Handler) CommentSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-comments.php", "Invalid form data")
	}

	commentID := c.Request().FormValue("comment_ID")
	content := c.Request().FormValue("content")
	status := c.Request().FormValue("comment_status")
	authorName := c.Request().FormValue("newcomment_author")
	authorEmail := c.Request().FormValue("newcomment_author_email")
	authorURL := c.Request().FormValue("newcomment_author_url")

	if commentID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
		return nil
	}

	// Get existing comment to update
	comment, err := h.comments.GetByID(ctx, commentID)
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/edit-comments.php", "Comment not found")
	}

	// Update with new values
	_, err = h.comments.Update(ctx, commentID, &comments.UpdateIn{
		Content: &content,
		Status:  &status,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/comment.php?c="+commentID+"&action=edit", "Failed to update comment")
	}

	// Note: author fields would need to be added to UpdateIn if not already there
	_ = comment
	_ = authorName
	_ = authorEmail
	_ = authorURL

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php?updated=1", http.StatusFound)
	return nil
}

// BulkCommentAction handles bulk comment moderation.
func (h *Handler) BulkCommentAction(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
		return nil
	}

	action := c.Request().FormValue("action")
	if action == "-1" {
		action = c.Request().FormValue("action2")
	}

	commentIDs := c.Request().Form["delete_comments[]"]
	if len(commentIDs) == 0 {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/edit-comments.php", http.StatusFound)
		return nil
	}

	for _, id := range commentIDs {
		switch action {
		case "approve":
			h.comments.Approve(ctx, id)
		case "unapprove":
			status := "pending"
			h.comments.Update(ctx, id, &comments.UpdateIn{Status: &status})
		case "spam":
			h.comments.MarkAsSpam(ctx, id)
		case "trash":
			status := "trash"
			h.comments.Update(ctx, id, &comments.UpdateIn{Status: &status})
		case "delete":
			h.comments.Delete(ctx, id)
		}
	}

	http.Redirect(c.Writer(), c.Request(), fmt.Sprintf("/wp-admin/edit-comments.php?%s=%d", action, len(commentIDs)), http.StatusFound)
	return nil
}

// UserSave handles creating or updating a user.
func (h *Handler) UserSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/users.php", "Invalid form data")
	}

	userID := c.Request().FormValue("user_id")
	email := c.Request().FormValue("email")
	name := c.Request().FormValue("display_name")
	firstName := c.Request().FormValue("first_name")
	lastName := c.Request().FormValue("last_name")
	role := c.Request().FormValue("role")
	bio := c.Request().FormValue("description")
	password := c.Request().FormValue("pass1")

	// Combine first and last name if display name not set
	if name == "" && (firstName != "" || lastName != "") {
		name = strings.TrimSpace(firstName + " " + lastName)
	}

	if userID == "" {
		// Create new user
		if email == "" || password == "" {
			return h.redirectWithError(c, "/wp-admin/user-new.php", "Email and password are required")
		}

		_, _, err := h.users.Register(ctx, &users.RegisterIn{
			Email:    email,
			Password: password,
			Name:     name,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/user-new.php", "Failed to create user: "+err.Error())
		}

		// Update role after creation if specified
		// Note: This would require updating the user after registration

		http.Redirect(c.Writer(), c.Request(), "/wp-admin/users.php?created=1", http.StatusFound)
		return nil
	}

	// Update existing user
	_, err := h.users.Update(ctx, userID, &users.UpdateIn{
		Name: &name,
		Bio:  &bio,
		Role: &role,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/user-edit.php?user_id="+userID, "Failed to update user: "+err.Error())
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/user-edit.php?user_id="+userID+"&updated=1", http.StatusFound)
	return nil
}

// ProfileSave handles updating the current user's profile.
func (h *Handler) ProfileSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/profile.php", "Invalid form data")
	}

	name := c.Request().FormValue("display_name")
	firstName := c.Request().FormValue("first_name")
	lastName := c.Request().FormValue("last_name")
	bio := c.Request().FormValue("description")
	email := c.Request().FormValue("email")

	if name == "" && (firstName != "" || lastName != "") {
		name = strings.TrimSpace(firstName + " " + lastName)
	}

	_, err := h.users.Update(ctx, user.ID, &users.UpdateIn{
		Name: &name,
		Bio:  &bio,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/profile.php", "Failed to update profile: "+err.Error())
	}

	_ = email // Email update would need separate handling

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/profile.php?updated=1", http.StatusFound)
	return nil
}

// SettingsSave handles saving settings for all settings pages.
func (h *Handler) SettingsSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/options-general.php", "Invalid form data")
	}

	// Determine which settings page based on option_page field
	optionPage := c.Request().FormValue("option_page")
	redirectURL := "/wp-admin/options-general.php"

	switch optionPage {
	case "general":
		redirectURL = "/wp-admin/options-general.php"
	case "writing":
		redirectURL = "/wp-admin/options-writing.php"
	case "reading":
		redirectURL = "/wp-admin/options-reading.php"
	case "discussion":
		redirectURL = "/wp-admin/options-discussion.php"
	case "media":
		redirectURL = "/wp-admin/options-media.php"
	case "permalink":
		redirectURL = "/wp-admin/options-permalink.php"
	}

	// Save all form values as settings
	for key, values := range c.Request().Form {
		if key == "option_page" || key == "action" || key == "_wpnonce" {
			continue
		}

		value := ""
		if len(values) > 0 {
			value = values[0]
		}

		_, err := h.settings.Set(ctx, &settings.SetIn{
			Key:       key,
			Value:     value,
			GroupName: optionPage,
		})
		if err != nil {
			// Log but continue
		}
	}

	http.Redirect(c.Writer(), c.Request(), redirectURL+"?settings-updated=true", http.StatusFound)
	return nil
}

// MenuSave handles creating or updating a menu.
func (h *Handler) MenuSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/nav-menus.php", "Invalid form data")
	}

	menuID := c.Request().FormValue("menu")
	menuName := c.Request().FormValue("menu-name")
	action := c.Request().FormValue("action")

	if action == "delete" && menuID != "" {
		if err := h.menus.DeleteMenu(ctx, menuID); err != nil {
			return h.redirectWithError(c, "/wp-admin/nav-menus.php?menu="+menuID, "Failed to delete menu")
		}
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/nav-menus.php?deleted=1", http.StatusFound)
		return nil
	}

	if menuID == "" || action == "create" {
		// Create new menu
		menu, err := h.menus.CreateMenu(ctx, &menus.CreateMenuIn{
			Name: menuName,
		})
		if err != nil {
			return h.redirectWithError(c, "/wp-admin/nav-menus.php", "Failed to create menu: "+err.Error())
		}
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/nav-menus.php?menu="+menu.ID+"&created=1", http.StatusFound)
		return nil
	}

	// Update existing menu
	_, err := h.menus.UpdateMenu(ctx, menuID, &menus.UpdateMenuIn{
		Name: &menuName,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/nav-menus.php?menu="+menuID, "Failed to update menu: "+err.Error())
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/nav-menus.php?menu="+menuID+"&updated=1", http.StatusFound)
	return nil
}

// MenuItemSave handles creating or updating a menu item.
func (h *Handler) MenuItemSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/nav-menus.php", "Invalid form data")
	}

	menuID := c.Request().FormValue("menu")
	title := c.Request().FormValue("menu-item-title")
	url := c.Request().FormValue("menu-item-url")
	linkType := c.Request().FormValue("menu-item-type")
	linkID := c.Request().FormValue("menu-item-object-id")
	parentID := c.Request().FormValue("menu-item-parent-id")

	_, err := h.menus.CreateItem(ctx, menuID, &menus.CreateItemIn{
		Title:    title,
		URL:      url,
		LinkType: linkType,
		LinkID:   linkID,
		ParentID: parentID,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/nav-menus.php?menu="+menuID, "Failed to add menu item: "+err.Error())
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/nav-menus.php?menu="+menuID+"&item-added=1", http.StatusFound)
	return nil
}

// MenuItemDelete handles deleting a menu item.
func (h *Handler) MenuItemDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	menuID := c.Query("menu")
	itemID := c.Query("item")

	if itemID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/nav-menus.php", http.StatusFound)
		return nil
	}

	if err := h.menus.DeleteItem(ctx, itemID); err != nil {
		return h.redirectWithError(c, "/wp-admin/nav-menus.php?menu="+menuID, "Failed to delete menu item")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/nav-menus.php?menu="+menuID+"&item-deleted=1", http.StatusFound)
	return nil
}

// MediaUpload handles file uploads.
func (h *Handler) MediaUpload(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	// Parse multipart form (32 MB max)
	if err := c.Request().ParseMultipartForm(32 << 20); err != nil {
		return h.redirectWithError(c, "/wp-admin/upload.php", "Failed to parse upload: "+err.Error())
	}

	file, header, err := c.Request().FormFile("async-upload")
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/upload.php", "No file uploaded")
	}
	defer file.Close()

	// Detect MIME type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Upload via media service
	mediaItem, err := h.media.Upload(ctx, user.ID, &media.UploadIn{
		File:     file,
		Filename: header.Filename,
		MimeType: mimeType,
		FileSize: header.Size,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/upload.php", "Failed to upload: "+err.Error())
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/upload.php?item="+mediaItem.ID+"&uploaded=1", http.StatusFound)
	return nil
}

// MediaSave handles updating media metadata.
func (h *Handler) MediaSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		return h.redirectWithError(c, "/wp-admin/upload.php", "Invalid form data")
	}

	mediaID := c.Request().FormValue("post_ID")
	title := c.Request().FormValue("post_title")
	altText := c.Request().FormValue("_wp_attachment_image_alt")
	caption := c.Request().FormValue("post_excerpt")
	description := c.Request().FormValue("post_content")

	if mediaID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/upload.php", http.StatusFound)
		return nil
	}

	_, err := h.media.Update(ctx, mediaID, &media.UpdateIn{
		Title:       &title,
		AltText:     &altText,
		Caption:     &caption,
		Description: &description,
	})
	if err != nil {
		return h.redirectWithError(c, "/wp-admin/post.php?post="+mediaID+"&action=edit", "Failed to update media: "+err.Error())
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/post.php?post="+mediaID+"&action=edit&message=1", http.StatusFound)
	return nil
}

// MediaDelete handles deleting media.
func (h *Handler) MediaDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()
	mediaID := c.Query("post")

	if mediaID == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/upload.php", http.StatusFound)
		return nil
	}

	if err := h.media.Delete(ctx, mediaID); err != nil {
		return h.redirectWithError(c, "/wp-admin/upload.php", "Failed to delete media")
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/upload.php?deleted=1", http.StatusFound)
	return nil
}

// QuickDraftSave handles the Quick Draft form on the dashboard.
func (h *Handler) QuickDraftSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return nil
	}

	ctx := c.Request().Context()

	if err := c.Request().ParseForm(); err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/", http.StatusFound)
		return nil
	}

	title := c.Request().FormValue("post_title")
	content := c.Request().FormValue("content")

	if title == "" && content == "" {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/", http.StatusFound)
		return nil
	}

	if title == "" {
		title = "Quick Draft"
	}

	post, err := h.posts.Create(ctx, user.ID, &posts.CreateIn{
		Title:   title,
		Content: content,
		Status:  "draft",
	})
	if err != nil {
		http.Redirect(c.Writer(), c.Request(), "/wp-admin/?draft-error=1", http.StatusFound)
		return nil
	}

	http.Redirect(c.Writer(), c.Request(), "/wp-admin/post.php?post="+post.ID+"&action=edit&message=6", http.StatusFound)
	return nil
}

// redirectWithError redirects with an error message.
func (h *Handler) redirectWithError(c *mizu.Ctx, url, message string) error {
	// In a real implementation, we'd store the message in a session flash
	// For now, just redirect
	http.Redirect(c.Writer(), c.Request(), url, http.StatusFound)
	return nil
}

// redirectWithSuccess redirects with a success message.
func (h *Handler) redirectWithSuccess(c *mizu.Ctx, url, message string) error {
	// In a real implementation, we'd store the message in a session flash
	// For now, just redirect
	http.Redirect(c.Writer(), c.Request(), url, http.StatusFound)
	return nil
}
