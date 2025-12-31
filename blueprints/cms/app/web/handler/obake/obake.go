// Package obake provides Ghost CMS-compatible admin interface handlers.
package obake

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/assets"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/settings"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// Config holds the handler configuration.
type Config struct {
	BaseURL   string
	Users     users.API
	Posts     posts.API
	Pages     pages.API
	Tags      tags.API
	Media     media.API
	Settings  settings.API
	GetUserID func(*mizu.Ctx) string
	GetUser   func(*mizu.Ctx) *users.User
}

// Handler handles Ghost admin interface requests.
type Handler struct {
	templates map[string]*template.Template
	cfg       Config
}

// New creates a new Ghost admin handler.
func New(templates map[string]*template.Template, cfg Config) *Handler {
	return &Handler{
		templates: templates,
		cfg:       cfg,
	}
}

// render renders a template with the given data.
func (h *Handler) render(c *mizu.Ctx, name string, data interface{}) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Writer(), data)
}

// requireAuth checks if user is authenticated.
func (h *Handler) requireAuth(c *mizu.Ctx) *users.User {
	user := h.cfg.GetUser(c)
	if user == nil {
		return nil
	}
	return user
}

// getSiteTitle returns the site title from settings.
func (h *Handler) getSiteTitle(c *mizu.Ctx) string {
	setting, err := h.cfg.Settings.Get(c.Context(), "site_title")
	if err != nil || setting == nil || setting.Value == "" {
		return "CMS"
	}
	return setting.Value
}

// getSiteURL returns the site URL.
func (h *Handler) getSiteURL() string {
	return h.cfg.BaseURL
}

// buildNav builds the navigation menu.
func (h *Handler) buildNav(activeID string) []NavItem {
	return []NavItem{
		{ID: "dashboard", Title: "Dashboard", URL: "/obake/", Icon: "dashboard", Active: activeID == "dashboard"},
		{ID: "site", Title: "View site", URL: "/", Icon: "external", External: true},
		{ID: "posts", Title: "Posts", URL: "/obake/posts/", Icon: "posts", Active: activeID == "posts"},
		{ID: "pages", Title: "Pages", URL: "/obake/pages/", Icon: "pages", Active: activeID == "pages"},
		{ID: "tags", Title: "Tags", URL: "/obake/tags/", Icon: "tags", Active: activeID == "tags"},
		{ID: "members", Title: "Members", URL: "/obake/members/", Icon: "members", Active: activeID == "members"},
		{ID: "staff", Title: "Staff", URL: "/obake/settings/staff/", Icon: "staff", Active: activeID == "staff"},
		{ID: "settings", Title: "Settings", URL: "/obake/settings/", Icon: "settings", Active: activeID == "settings"},
	}
}

// baseData creates common template data.
func (h *Handler) baseData(c *mizu.Ctx, title, activeNav string, user *users.User) BaseData {
	return BaseData{
		Title:     title,
		SiteTitle: h.getSiteTitle(c),
		SiteURL:   h.getSiteURL(),
		User:      user,
		Nav:       h.buildNav(activeNav),
		ActiveNav: activeNav,
	}
}

// Login renders the Ghost login page.
func (h *Handler) Login(c *mizu.Ctx) error {
	// If already logged in, redirect to dashboard
	if user := h.requireAuth(c); user != nil {
		return c.Redirect(http.StatusFound, "/obake/")
	}

	var siteLogo string
	if setting, err := h.cfg.Settings.Get(c.Context(), "site_logo"); err == nil && setting != nil {
		siteLogo = setting.Value
	}

	data := LoginData{
		SiteTitle:  h.getSiteTitle(c),
		SiteURL:    h.getSiteURL(),
		SiteLogo:   siteLogo,
		RedirectTo: c.Query("redirect"),
	}

	return h.render(c, "login", data)
}

// LoginPost handles login form submission.
func (h *Handler) LoginPost(c *mizu.Ctx) error {
	email := c.Request().FormValue("email")
	password := c.Request().FormValue("password")
	redirectTo := c.Request().FormValue("redirect")

	user, session, err := h.cfg.Users.Login(c.Context(), &users.LoginIn{
		Email:    email,
		Password: password,
	})
	if err != nil || user == nil {
		var siteLogo string
		if setting, err := h.cfg.Settings.Get(c.Context(), "site_logo"); err == nil && setting != nil {
			siteLogo = setting.Value
		}
		data := LoginData{
			SiteTitle:  h.getSiteTitle(c),
			SiteURL:    h.getSiteURL(),
			SiteLogo:   siteLogo,
			Error:      "Incorrect email or password",
			Email:      email,
			RedirectTo: redirectTo,
		}
		return h.render(c, "login", data)
	}

	// Set session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   strings.HasPrefix(h.cfg.BaseURL, "https"),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30, // 30 days
	})

	if redirectTo != "" {
		return c.Redirect(http.StatusFound, redirectTo)
	}
	return c.Redirect(http.StatusFound, "/obake/")
}

// Logout handles user logout.
func (h *Handler) Logout(c *mizu.Ctx) error {
	// Clear session cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	return c.Redirect(http.StatusFound, "/obake/signin/")
}

// Dashboard renders the Ghost dashboard.
func (h *Handler) Dashboard(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	activeTab := c.Query("tab")
	if activeTab == "" {
		activeTab = "overview"
	}

	dateRange := c.Query("range")
	if dateRange == "" {
		dateRange = "30d"
	}

	// Get counts for metrics
	postsList, postCount, _ := h.cfg.Posts.List(ctx, &posts.ListIn{Limit: 1000})
	pagesList, pageCount, _ := h.cfg.Pages.List(ctx, &pages.ListIn{Limit: 1000})
	_, memberCount, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 1000})

	if postsList == nil {
		postCount = 0
	}
	if pagesList == nil {
		pageCount = 0
	}

	// Get recent posts for top content
	recentPosts, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Limit:   5,
		OrderBy: "created_at",
		Order:   "desc",
	})

	topContent := make([]TopContent, 0, len(recentPosts))
	for _, p := range recentPosts {
		topContent = append(topContent, TopContent{
			ID:    p.ID,
			Title: p.Title,
			Type:  "post",
			URL:   "/obake/editor/post/" + p.ID + "/",
		})
	}

	data := DashboardData{
		BaseData:  h.baseData(c, "Dashboard", "dashboard", user),
		ActiveTab: activeTab,
		DateRange: dateRange,

		UniqueVisitors: DashboardMetric{
			Label: "Unique visitors",
			Value: "0",
			Trend: "+0%",
		},
		TotalPageviews: DashboardMetric{
			Label: "Total pageviews",
			Value: strconv.Itoa(postCount + pageCount),
			Trend: "+0%",
		},
		RealtimeVisitors: 0,
		TopContent:       topContent,
		TopSources:       []TopSource{},

		TotalSubscribers: DashboardMetric{
			Label: "Total subscribers",
			Value: strconv.Itoa(memberCount),
			Trend: "+0%",
		},
		TotalMembers: DashboardMetric{
			Label: "Total members",
			Value: strconv.Itoa(memberCount),
			Trend: "+0%",
		},
		FreeMembers: DashboardMetric{
			Label: "Free members",
			Value: strconv.Itoa(memberCount),
		},
		PaidMembers: DashboardMetric{
			Label: "Paid members",
			Value: "0",
		},
		MRR: DashboardMetric{
			Label: "MRR",
			Value: "$0",
		},
	}

	return h.render(c, "dashboard", data)
}

// PostsList renders the posts list page.
func (h *Handler) PostsList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	filterStatus := c.Query("status")
	filterAuthor := c.Query("author")
	filterTag := c.Query("tag")
	searchQuery := c.Query("q")
	viewMode := c.Query("view")
	if viewMode == "" {
		viewMode = "list"
	}

	// Build list params
	params := &posts.ListIn{
		Limit:   50,
		OrderBy: "created_at",
		Order:   "desc",
	}

	if filterStatus == "published" {
		params.Status = "publish"
	} else if filterStatus == "draft" {
		params.Status = "draft"
	} else if filterStatus == "scheduled" {
		params.Status = "future"
	}

	if filterAuthor != "" {
		params.AuthorID = filterAuthor
	}

	if filterTag != "" {
		params.TagID = filterTag
	}

	if searchQuery != "" {
		params.Search = searchQuery
	}

	// Get posts
	postsList, _, err := h.cfg.Posts.List(ctx, params)
	if err != nil {
		postsList = []*posts.Post{}
	}

	// Build post rows
	postRows := make([]*PostRow, 0, len(postsList))
	for _, p := range postsList {
		author, _ := h.cfg.Users.GetByID(ctx, p.AuthorID)

		status := "draft"
		if p.Status == "publish" {
			status = "published"
		} else if p.Status == "future" {
			status = "scheduled"
		}

		postRows = append(postRows, &PostRow{
			Post:        p,
			Author:      author,
			Status:      status,
			AccessLevel: "public",
			PublishDate: p.CreatedAt,
		})
	}

	// Get authors for filter
	allAuthors, _, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 100})

	// Get tags for filter
	allTags, _, _ := h.cfg.Tags.List(ctx, &tags.ListIn{Limit: 100})

	data := PostsListData{
		BaseData:     h.baseData(c, "Posts", "posts", user),
		Posts:        postRows,
		TotalPosts:   len(postRows),
		FilterStatus: filterStatus,
		FilterAuthor: filterAuthor,
		FilterTag:    filterTag,
		SearchQuery:  searchQuery,
		ViewMode:     viewMode,
		Authors:      allAuthors,
		AllTags:      allTags,
	}

	return h.render(c, "posts", data)
}

// PostNew renders the new post page.
func (h *Handler) PostNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()

	allAuthors, _, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 100})
	allTags, _, _ := h.cfg.Tags.List(ctx, &tags.ListIn{Limit: 100})

	data := PostEditData{
		BaseData:        h.baseData(c, "New post", "posts", user),
		IsNew:           true,
		PostType:        "post",
		AllAuthors:      allAuthors,
		SelectedAuthors: []string{user.ID},
		AllTags:         allTags,
		AccessLevel:     "public",
		Templates: []SelectOption{
			{Value: "default", Label: "Default", Selected: true},
		},
	}

	return h.render(c, "post-edit", data)
}

// PostEdit renders the post editor.
func (h *Handler) PostEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	postID := c.Param("id")

	post, err := h.cfg.Posts.GetByID(ctx, postID)
	if err != nil {
		return c.Text(http.StatusNotFound, "Post not found")
	}

	allAuthors, _, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 100})
	allTags, _, _ := h.cfg.Tags.List(ctx, &tags.ListIn{Limit: 100})

	// Get post tags
	tagIDs, _ := h.cfg.Posts.GetTagIDs(ctx, postID)

	// Get featured image if exists
	var featuredImage *media.Media
	if post.FeaturedImageID != "" {
		featuredImage, _ = h.cfg.Media.GetByID(ctx, post.FeaturedImageID)
	}

	publishDate := post.CreatedAt.Format("2006-01-02")
	publishTime := post.CreatedAt.Format("15:04")

	data := PostEditData{
		BaseData:        h.baseData(c, post.Title, "posts", user),
		Post:            post,
		IsNew:           false,
		PostType:        "post",
		Slug:            post.Slug,
		PublishDate:     publishDate,
		PublishTime:     publishTime,
		AllAuthors:      allAuthors,
		SelectedAuthors: []string{post.AuthorID},
		AllTags:         allTags,
		SelectedTags:    tagIDs,
		FeaturedImage:   featuredImage,
		Excerpt:         post.Excerpt,
		AccessLevel:     "public",
		Templates: []SelectOption{
			{Value: "default", Label: "Default", Selected: true},
		},
	}

	return h.render(c, "post-edit", data)
}

// PostSave handles post save.
func (h *Handler) PostSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()
	postID := c.Request().FormValue("id")
	title := c.Request().FormValue("title")
	content := c.Request().FormValue("content")
	slug := c.Request().FormValue("slug")
	excerpt := c.Request().FormValue("excerpt")
	status := c.Request().FormValue("status")
	featuredImageID := c.Request().FormValue("featured_image_id")

	if postID == "" {
		// Create new post
		newPost, err := h.cfg.Posts.Create(ctx, user.ID, &posts.CreateIn{
			Title:           title,
			Content:         content,
			Slug:            slug,
			Excerpt:         excerpt,
			Status:          status,
			FeaturedImageID: featuredImageID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.Redirect(http.StatusFound, "/obake/editor/post/"+newPost.ID+"/")
	}

	// Update existing post
	_, err := h.cfg.Posts.Update(ctx, postID, &posts.UpdateIn{
		Title:           &title,
		Content:         &content,
		Slug:            &slug,
		Excerpt:         &excerpt,
		Status:          &status,
		FeaturedImageID: &featuredImageID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/editor/post/"+postID+"/")
}

// PostDelete handles post deletion.
func (h *Handler) PostDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	postID := c.Param("id")
	if err := h.cfg.Posts.Delete(c.Context(), postID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/posts/")
}

// PagesList renders the pages list.
func (h *Handler) PagesList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	filterStatus := c.Query("status")
	searchQuery := c.Query("q")

	params := &pages.ListIn{
		Limit: 50,
	}

	if filterStatus == "published" {
		params.Status = "publish"
	} else if filterStatus == "draft" {
		params.Status = "draft"
	}

	if searchQuery != "" {
		params.Search = searchQuery
	}

	pagesList, _, err := h.cfg.Pages.List(ctx, params)
	if err != nil {
		pagesList = []*pages.Page{}
	}

	pageRows := make([]*PageRow, 0, len(pagesList))
	for _, p := range pagesList {
		author, _ := h.cfg.Users.GetByID(ctx, p.AuthorID)

		status := "draft"
		if p.Status == "publish" {
			status = "published"
		}

		pageRows = append(pageRows, &PageRow{
			Page:        p,
			Author:      author,
			Status:      status,
			PublishDate: p.CreatedAt,
		})
	}

	data := PagesListData{
		BaseData:     h.baseData(c, "Pages", "pages", user),
		Pages:        pageRows,
		TotalPages:   len(pageRows),
		FilterStatus: filterStatus,
		SearchQuery:  searchQuery,
		ViewMode:     "list",
	}

	return h.render(c, "pages-list", data)
}

// PageNew renders the new page editor.
func (h *Handler) PageNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	allAuthors, _, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 100})

	data := PageEditData{
		BaseData:        h.baseData(c, "New page", "pages", user),
		IsNew:           true,
		SelectedAuthors: []string{user.ID},
		Authors:         allAuthors,
		AccessLevel:     "public",
		Templates: []SelectOption{
			{Value: "default", Label: "Default", Selected: true},
		},
	}

	return h.render(c, "page-edit", data)
}

// PageEdit renders the page editor.
func (h *Handler) PageEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	pageID := c.Param("id")

	page, err := h.cfg.Pages.GetByID(ctx, pageID)
	if err != nil {
		return c.Text(http.StatusNotFound, "Page not found")
	}

	allAuthors, _, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 100})
	parentPages, _ := h.cfg.Pages.GetTree(ctx)

	var featuredImage *media.Media
	if page.FeaturedImageID != "" {
		featuredImage, _ = h.cfg.Media.GetByID(ctx, page.FeaturedImageID)
	}

	data := PageEditData{
		BaseData:        h.baseData(c, page.Title, "pages", user),
		Page:            page,
		IsNew:           false,
		ParentPages:     parentPages,
		SelectedParent:  page.ParentID,
		Slug:            page.Slug,
		PublishDate:     page.CreatedAt.Format("2006-01-02"),
		PublishTime:     page.CreatedAt.Format("15:04"),
		Authors:         allAuthors,
		SelectedAuthors: []string{page.AuthorID},
		FeaturedImage:   featuredImage,
		AccessLevel:     "public",
		Templates: []SelectOption{
			{Value: "default", Label: "Default", Selected: true},
		},
	}

	return h.render(c, "page-edit", data)
}

// PageSave handles page save.
func (h *Handler) PageSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()
	pageID := c.Request().FormValue("id")
	title := c.Request().FormValue("title")
	content := c.Request().FormValue("content")
	slug := c.Request().FormValue("slug")
	status := c.Request().FormValue("status")
	parentID := c.Request().FormValue("parent_id")
	featuredImageID := c.Request().FormValue("featured_image_id")

	if pageID == "" {
		newPage, err := h.cfg.Pages.Create(ctx, user.ID, &pages.CreateIn{
			Title:           title,
			Content:         content,
			Slug:            slug,
			Status:          status,
			ParentID:        parentID,
			FeaturedImageID: featuredImageID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.Redirect(http.StatusFound, "/obake/editor/page/"+newPage.ID+"/")
	}

	_, err := h.cfg.Pages.Update(ctx, pageID, &pages.UpdateIn{
		Title:           &title,
		Content:         &content,
		Slug:            &slug,
		Status:          &status,
		ParentID:        &parentID,
		FeaturedImageID: &featuredImageID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/editor/page/"+pageID+"/")
}

// PageDelete handles page deletion.
func (h *Handler) PageDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	pageID := c.Param("id")
	if err := h.cfg.Pages.Delete(c.Context(), pageID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/pages/")
}

// TagsList renders the tags list.
func (h *Handler) TagsList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	searchQuery := c.Query("q")
	showInternal := c.Query("internal") == "1"

	params := &tags.ListIn{Limit: 100}
	if searchQuery != "" {
		params.Search = searchQuery
	}

	tagsList, _, err := h.cfg.Tags.List(ctx, params)
	if err != nil {
		tagsList = []*tags.Tag{}
	}

	tagRows := make([]*TagRow, 0, len(tagsList))
	for _, t := range tagsList {
		isInternal := strings.HasPrefix(t.Name, "#")
		if !showInternal && isInternal {
			continue
		}
		tagRows = append(tagRows, &TagRow{
			Tag:        t,
			PostCount:  t.PostCount,
			IsInternal: isInternal,
		})
	}

	data := TagsListData{
		BaseData:     h.baseData(c, "Tags", "tags", user),
		Tags:         tagRows,
		ShowInternal: showInternal,
		SearchQuery:  searchQuery,
	}

	return h.render(c, "tags", data)
}

// TagNew renders the new tag page.
func (h *Handler) TagNew(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	data := TagEditData{
		BaseData: h.baseData(c, "New tag", "tags", user),
		IsNew:    true,
	}

	return h.render(c, "tag-edit", data)
}

// TagEdit renders the tag editor.
func (h *Handler) TagEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	tagSlug := c.Param("slug")

	tag, err := h.cfg.Tags.GetBySlug(ctx, tagSlug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Tag not found")
	}

	var featuredImage *media.Media
	if tag.FeaturedImageID != "" {
		featuredImage, _ = h.cfg.Media.GetByID(ctx, tag.FeaturedImageID)
	}

	data := TagEditData{
		BaseData:      h.baseData(c, tag.Name, "tags", user),
		Tag:           tag,
		IsNew:         false,
		IsInternal:    strings.HasPrefix(tag.Name, "#"),
		Slug:          tag.Slug,
		Description:   tag.Description,
		FeaturedImage: featuredImage,
	}

	return h.render(c, "tag-edit", data)
}

// TagSave handles tag save.
func (h *Handler) TagSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()
	tagID := c.Request().FormValue("id")
	name := c.Request().FormValue("name")
	slug := c.Request().FormValue("slug")
	description := c.Request().FormValue("description")
	featuredImageID := c.Request().FormValue("featured_image_id")

	if tagID == "" {
		newTag, err := h.cfg.Tags.Create(ctx, &tags.CreateIn{
			Name:            name,
			Slug:            slug,
			Description:     description,
			FeaturedImageID: featuredImageID,
		})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.Redirect(http.StatusFound, "/obake/tags/"+newTag.Slug+"/")
	}

	_, err := h.cfg.Tags.Update(ctx, tagID, &tags.UpdateIn{
		Name:            &name,
		Slug:            &slug,
		Description:     &description,
		FeaturedImageID: &featuredImageID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/tags/"+slug+"/")
}

// TagDelete handles tag deletion.
func (h *Handler) TagDelete(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	tagSlug := c.Param("slug")
	tag, err := h.cfg.Tags.GetBySlug(c.Context(), tagSlug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tag not found"})
	}

	if err := h.cfg.Tags.Delete(c.Context(), tag.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/tags/")
}

// MembersList renders the members list (stub).
func (h *Handler) MembersList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	data := MembersListData{
		BaseData:     h.baseData(c, "Members", "members", user),
		Members:      []*MemberRow{},
		TotalMembers: 0,
	}

	return h.render(c, "members", data)
}

// MemberDetail renders member detail (stub).
func (h *Handler) MemberDetail(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	data := MemberDetailData{
		BaseData: h.baseData(c, "Member", "members", user),
	}

	return h.render(c, "member-detail", data)
}

// StaffList renders the staff list.
func (h *Handler) StaffList(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()

	usersList, _, err := h.cfg.Users.List(ctx, &users.ListIn{Limit: 100})
	if err != nil {
		usersList = []*users.User{}
	}

	staffRows := make([]*StaffRow, 0, len(usersList))
	for _, u := range usersList {
		role := u.Role
		roleLabel := strings.Title(role)
		if role == "" {
			role = "contributor"
			roleLabel = "Contributor"
		}

		staffRows = append(staffRows, &StaffRow{
			User:      u,
			Role:      role,
			RoleLabel: roleLabel,
			LastSeen:  time.Now(),
		})
	}

	data := StaffListData{
		BaseData: h.baseData(c, "Staff", "staff", user),
		Staff:    staffRows,
	}

	return h.render(c, "staff", data)
}

// StaffEdit renders the staff editor.
func (h *Handler) StaffEdit(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	staffSlug := c.Param("slug")

	staff, err := h.cfg.Users.GetBySlug(ctx, staffSlug)
	if err != nil {
		return c.Text(http.StatusNotFound, "Staff member not found")
	}

	isSelf := staff.ID == user.ID

	data := StaffEditData{
		BaseData: h.baseData(c, staff.DisplayName(), "staff", user),
		Staff:    staff,
		IsNew:    false,
		IsSelf:   isSelf,
		Name:     staff.DisplayName(),
		Slug:     staff.Username(),
		Email:    staff.Email,
		Bio:      staff.Bio,
		Role:     staff.Role,
		Roles: []SelectOption{
			{Value: "contributor", Label: "Contributor", Selected: staff.Role == "contributor"},
			{Value: "author", Label: "Author", Selected: staff.Role == "author"},
			{Value: "editor", Label: "Editor", Selected: staff.Role == "editor"},
			{Value: "administrator", Label: "Administrator", Selected: staff.Role == "administrator"},
		},
	}

	return h.render(c, "staff-edit", data)
}

// StaffSave handles staff save.
func (h *Handler) StaffSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()
	staffID := c.Request().FormValue("id")
	displayName := c.Request().FormValue("display_name")
	bio := c.Request().FormValue("bio")
	role := c.Request().FormValue("role")

	_, err := h.cfg.Users.Update(ctx, staffID, &users.UpdateIn{
		Name: &displayName,
		Bio:  &bio,
		Role: &role,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.Redirect(http.StatusFound, "/obake/settings/staff/")
}

// StaffInvite handles staff invite.
func (h *Handler) StaffInvite(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	// Stub - in a real implementation this would send an invitation email
	return c.Redirect(http.StatusFound, "/obake/settings/staff/")
}

// Settings renders the settings page.
func (h *Handler) Settings(c *mizu.Ctx) error {
	return c.Redirect(http.StatusFound, "/obake/settings/general/")
}

// SettingsGeneral renders general settings.
func (h *Handler) SettingsGeneral(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()

	var siteTitle, siteDescription, siteTimezone, siteLanguage string

	if setting, err := h.cfg.Settings.Get(ctx, "site_title"); err == nil && setting != nil {
		siteTitle = setting.Value
	}
	if setting, err := h.cfg.Settings.Get(ctx, "site_description"); err == nil && setting != nil {
		siteDescription = setting.Value
	}
	if setting, err := h.cfg.Settings.Get(ctx, "site_timezone"); err == nil && setting != nil {
		siteTimezone = setting.Value
	}
	if setting, err := h.cfg.Settings.Get(ctx, "site_language"); err == nil && setting != nil {
		siteLanguage = setting.Value
	}

	data := SettingsGeneralData{
		BaseData:        h.baseData(c, "General settings", "settings", user),
		SiteTitle:       siteTitle,
		SiteDescription: siteDescription,
		SiteTimezone:    siteTimezone,
		SiteLanguage:    siteLanguage,
		Timezones: []SelectOption{
			{Value: "UTC", Label: "UTC", Selected: siteTimezone == "UTC" || siteTimezone == ""},
			{Value: "America/New_York", Label: "Eastern Time", Selected: siteTimezone == "America/New_York"},
			{Value: "America/Los_Angeles", Label: "Pacific Time", Selected: siteTimezone == "America/Los_Angeles"},
			{Value: "Europe/London", Label: "London", Selected: siteTimezone == "Europe/London"},
			{Value: "Asia/Tokyo", Label: "Tokyo", Selected: siteTimezone == "Asia/Tokyo"},
		},
		Languages: []SelectOption{
			{Value: "en", Label: "English", Selected: siteLanguage == "en" || siteLanguage == ""},
			{Value: "es", Label: "Spanish", Selected: siteLanguage == "es"},
			{Value: "fr", Label: "French", Selected: siteLanguage == "fr"},
			{Value: "de", Label: "German", Selected: siteLanguage == "de"},
			{Value: "ja", Label: "Japanese", Selected: siteLanguage == "ja"},
		},
	}

	return h.render(c, "settings-general", data)
}

// SettingsDesign renders design settings.
func (h *Handler) SettingsDesign(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	accentColor := "#30CF43"
	if setting, err := h.cfg.Settings.Get(ctx, "accent_color"); err == nil && setting != nil && setting.Value != "" {
		accentColor = setting.Value
	}

	// Get active theme from settings
	activeTheme := "default"
	if setting, err := h.cfg.Settings.Get(ctx, "active_theme"); err == nil && setting != nil && setting.Value != "" {
		activeTheme = setting.Value
	}

	// Get available themes
	themeList, err := assets.ListThemes()
	if err != nil {
		themeList = []*assets.ThemeJSON{}
	}

	// Convert to ThemeOption for template
	themes := make([]*ThemeOption, 0, len(themeList))
	for _, t := range themeList {
		themes = append(themes, &ThemeOption{
			Name:        t.Name,
			Slug:        t.Slug,
			Version:     t.Version,
			Description: t.Description,
			Screenshot:  "/themes/" + t.Slug + "/assets/images/screenshot.png",
			Active:      t.Slug == activeTheme,
		})
	}

	data := SettingsDesignData{
		BaseData:     h.baseData(c, "Design settings", "settings", user),
		Themes:       themes,
		ActiveTheme:  activeTheme,
		AccentColor:  accentColor,
		PrimaryNav:   []NavLink{},
		SecondaryNav: []NavLink{},
	}

	return h.render(c, "settings-design", data)
}

// SettingsMembership renders membership settings.
func (h *Handler) SettingsMembership(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	data := SettingsMembershipData{
		BaseData:      h.baseData(c, "Membership settings", "settings", user),
		DefaultAccess: "public",
		PortalEnabled: true,
		Tiers:         []Tier{},
	}

	return h.render(c, "settings-membership", data)
}

// SettingsEmail renders email settings.
func (h *Handler) SettingsEmail(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	data := SettingsEmailData{
		BaseData:          h.baseData(c, "Email settings", "settings", user),
		NewsletterEnabled: true,
	}

	return h.render(c, "settings-email", data)
}

// SettingsAdvanced renders advanced settings.
func (h *Handler) SettingsAdvanced(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	var codeHead, codeFoot string

	if setting, err := h.cfg.Settings.Get(ctx, "code_injection_head"); err == nil && setting != nil {
		codeHead = setting.Value
	}
	if setting, err := h.cfg.Settings.Get(ctx, "code_injection_foot"); err == nil && setting != nil {
		codeFoot = setting.Value
	}

	data := SettingsAdvancedData{
		BaseData:     h.baseData(c, "Advanced settings", "settings", user),
		CodeHead:     codeHead,
		CodeFoot:     codeFoot,
		ExportURL:    "/obake/settings/export/",
		Integrations: []Integration{},
		LabsFeatures: []LabsFeature{},
	}

	return h.render(c, "settings-advanced", data)
}

// SettingsSave handles settings save.
func (h *Handler) SettingsSave(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()
	section := c.Request().FormValue("section")

	switch section {
	case "general":
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "site_title", Value: c.Request().FormValue("site_title")})
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "site_description", Value: c.Request().FormValue("site_description")})
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "site_timezone", Value: c.Request().FormValue("site_timezone")})
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "site_language", Value: c.Request().FormValue("site_language")})
		return c.Redirect(http.StatusFound, "/obake/settings/general/")

	case "design":
		// Save theme setting if provided
		if theme := c.Request().FormValue("active_theme"); theme != "" {
			h.cfg.Settings.Set(ctx, &settings.SetIn{
				Key:       "active_theme",
				Value:     theme,
				ValueType: "string",
				GroupName: "appearance",
			})
		}
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "accent_color", Value: c.Request().FormValue("accent_color")})
		return c.Redirect(http.StatusFound, "/obake/settings/design/")

	case "advanced":
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "code_injection_head", Value: c.Request().FormValue("code_head")})
		h.cfg.Settings.Set(ctx, &settings.SetIn{Key: "code_injection_foot", Value: c.Request().FormValue("code_foot")})
		return c.Redirect(http.StatusFound, "/obake/settings/advanced/")
	}

	return c.Redirect(http.StatusFound, "/obake/settings/")
}

// Search handles global search.
func (h *Handler) Search(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()
	query := c.Query("q")

	if query == "" {
		return c.JSON(200, SearchData{Results: []SearchResultItem{}})
	}

	results := []SearchResultItem{}

	// Search posts
	postsList, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Search: query,
		Limit:  5,
	})
	for _, p := range postsList {
		results = append(results, SearchResultItem{
			Type:  "post",
			Title: p.Title,
			URL:   "/obake/editor/post/" + p.ID + "/",
			Icon:  "posts",
		})
	}

	// Search pages
	pagesList, _, _ := h.cfg.Pages.List(ctx, &pages.ListIn{
		Search: query,
		Limit:  5,
	})
	for _, p := range pagesList {
		results = append(results, SearchResultItem{
			Type:  "page",
			Title: p.Title,
			URL:   "/obake/editor/page/" + p.ID + "/",
			Icon:  "pages",
		})
	}

	// Search tags
	tagsList, _, _ := h.cfg.Tags.List(ctx, &tags.ListIn{
		Search: query,
		Limit:  5,
	})
	for _, t := range tagsList {
		results = append(results, SearchResultItem{
			Type:  "tag",
			Title: t.Name,
			URL:   "/obake/tags/" + t.Slug + "/",
			Icon:  "tags",
		})
	}

	return c.JSON(200, SearchData{
		Query:   query,
		Results: results,
	})
}

// MediaLibrary renders the media library.
func (h *Handler) MediaLibrary(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.Redirect(http.StatusFound, "/obake/signin/")
	}

	ctx := c.Context()
	filterType := c.Query("type")

	params := &media.ListIn{Limit: 50}
	if filterType != "" {
		params.MimeType = filterType
	}

	mediaList, _, err := h.cfg.Media.List(ctx, params)
	if err != nil {
		mediaList = []*media.Media{}
	}

	data := MediaLibraryData{
		BaseData:   h.baseData(c, "Media", "posts", user),
		Items:      mediaList,
		TotalItems: len(mediaList),
		FilterType: filterType,
	}

	return h.render(c, "media", data)
}

// MediaUpload handles media upload.
func (h *Handler) MediaUpload(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no file uploaded"})
	}
	defer file.Close()

	uploaded, err := h.cfg.Media.Upload(ctx, user.ID, &media.UploadIn{
		File:     file,
		Filename: header.Filename,
		MimeType: header.Header.Get("Content-Type"),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, uploaded)
}

// Export handles content export.
func (h *Handler) Export(c *mizu.Ctx) error {
	user := h.requireAuth(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	ctx := c.Context()

	// Get all content
	allPosts, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{Limit: 10000})
	allPages, _, _ := h.cfg.Pages.List(ctx, &pages.ListIn{Limit: 10000})
	allTags, _, _ := h.cfg.Tags.List(ctx, &tags.ListIn{Limit: 10000})
	allUsers, _, _ := h.cfg.Users.List(ctx, &users.ListIn{Limit: 10000})

	export := map[string]interface{}{
		"meta": map[string]interface{}{
			"exported_on": time.Now().Unix(),
			"version":     "5.0.0",
		},
		"data": map[string]interface{}{
			"posts": allPosts,
			"pages": allPages,
			"tags":  allTags,
			"users": allUsers,
		},
	}

	c.Writer().Header().Set("Content-Type", "application/json")
	c.Writer().Header().Set("Content-Disposition", "attachment; filename=ghost-export.json")

	return json.NewEncoder(c.Writer()).Encode(export)
}
