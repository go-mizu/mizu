// Package site provides the public-facing frontend handlers.
package site

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

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

// markdown renderer for converting markdown to HTML
var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,            // GitHub Flavored Markdown
		extension.Typographer,    // Smart quotes, dashes, etc.
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(), // Auto-generate heading IDs
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(), // Convert \n to <br>
		html.WithUnsafe(),    // Allow raw HTML in markdown
	),
)

// renderMarkdown converts markdown content to HTML.
func renderMarkdown(content string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return content // Return original on error
	}
	return buf.String()
}

// Config holds the handler configuration.
type Config struct {
	BaseURL    string
	Posts      posts.API
	Pages      pages.API
	Categories categories.API
	Tags       tags.API
	Users      users.API
	Media      media.API
	Comments   comments.API
	Settings   settings.API
	Menus      menus.API
	GetUserID  func(*mizu.Ctx) string
	GetUser    func(*mizu.Ctx) *users.User
}

// TemplateLoader loads templates for a given theme slug.
type TemplateLoader func(slug string) (map[string]*template.Template, error)

// Handler handles frontend site requests.
type Handler struct {
	templateLoader TemplateLoader
	templateCache  map[string]map[string]*template.Template // theme slug -> templates
	cfg            Config
}

// New creates a new site handler.
func New(templateLoader TemplateLoader, cfg Config) *Handler {
	return &Handler{
		templateLoader: templateLoader,
		templateCache:  make(map[string]map[string]*template.Template),
		cfg:            cfg,
	}
}

// getTemplates returns templates for the active theme.
func (h *Handler) getTemplates(c *mizu.Ctx) (map[string]*template.Template, error) {
	// Get active theme from settings
	activeTheme := "default"
	if setting, err := h.cfg.Settings.Get(c.Context(), "active_theme"); err == nil && setting.Value != "" {
		activeTheme = setting.Value
	}

	// Check cache
	if templates, ok := h.templateCache[activeTheme]; ok {
		return templates, nil
	}

	// Load templates for this theme
	templates, err := h.templateLoader(activeTheme)
	if err != nil {
		return nil, err
	}

	// Cache the templates
	h.templateCache[activeTheme] = templates
	return templates, nil
}

// render renders a template with the given data.
func (h *Handler) render(c *mizu.Ctx, name string, data interface{}) error {
	return h.renderWithStatus(c, name, data, http.StatusOK)
}

// renderWithStatus renders a template with the given data and status code.
func (h *Handler) renderWithStatus(c *mizu.Ctx, name string, data interface{}, status int) error {
	templates, err := h.getTemplates(c)
	if err != nil {
		return c.Text(http.StatusInternalServerError, "Failed to load templates: "+err.Error())
	}

	tmpl, ok := templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().WriteHeader(status)
	// Execute the "layout" template explicitly (the base layout)
	return tmpl.ExecuteTemplate(c.Writer(), "layout", data)
}

// getSiteContext builds site context from settings.
func (h *Handler) getSiteContext(c *mizu.Ctx) SiteContext {
	ctx := c.Context()

	site := SiteContext{
		Name:     "CMS",
		URL:      h.cfg.BaseURL,
		Language: "en",
		Timezone: "UTC",
	}

	if s, err := h.cfg.Settings.Get(ctx, "site_title"); err == nil && s != nil {
		site.Name = s.Value
	}
	if s, err := h.cfg.Settings.Get(ctx, "site_tagline"); err == nil && s != nil {
		site.Tagline = s.Value
	}
	if s, err := h.cfg.Settings.Get(ctx, "site_description"); err == nil && s != nil {
		site.Description = s.Value
	}

	return site
}

// getThemeContext builds theme context from the active theme.
func (h *Handler) getThemeContext(c *mizu.Ctx) ThemeContext {
	ctx := c.Context()

	// Get active theme from settings
	activeTheme := "default"
	if setting, err := h.cfg.Settings.Get(ctx, "active_theme"); err == nil && setting.Value != "" {
		activeTheme = setting.Value
	}

	// Try to load theme configuration from theme.json
	themeJSON, err := assets.GetTheme(activeTheme)
	if err != nil {
		// Fallback to default theme
		themeJSON, _ = assets.GetTheme("default")
	}

	// If still nil, return hardcoded defaults
	if themeJSON == nil {
		return h.getDefaultThemeContext()
	}

	// Build config map from theme.json config
	config := make(map[string]interface{})
	for k, v := range themeJSON.Config {
		config[k] = v
	}

	return ThemeContext{
		Name:       themeJSON.Name,
		Slug:       themeJSON.Slug,
		Version:    themeJSON.Version,
		Config:     config,
		Colors:     themeJSON.Colors,
		DarkColors: themeJSON.DarkColors,
		Fonts:      themeJSON.Fonts,
		Features:   themeJSON.Features,
	}
}

// getDefaultThemeContext returns hardcoded default theme context as fallback.
func (h *Handler) getDefaultThemeContext() ThemeContext {
	return ThemeContext{
		Name:    "Default Theme",
		Slug:    "default",
		Version: "1.0.0",
		Config: map[string]interface{}{
			"posts_per_page":       10,
			"sidebar_position":     "right",
			"show_author_bio":      true,
			"show_related_posts":   true,
			"related_posts_count":  3,
			"show_reading_time":    true,
			"show_post_navigation": true,
			"enable_comments":      true,
			"enable_newsletter":    true,
		},
		Colors: map[string]string{
			"primary":        "#2563eb",
			"primary_hover":  "#1d4ed8",
			"secondary":      "#64748b",
			"accent":         "#f59e0b",
			"background":     "#ffffff",
			"surface":        "#f8fafc",
			"surface_alt":    "#f1f5f9",
			"text":           "#1e293b",
			"text_secondary": "#475569",
			"text_muted":     "#64748b",
			"heading":        "#0f172a",
			"link":           "#2563eb",
			"link_hover":     "#1d4ed8",
			"border":         "#e2e8f0",
			"border_light":   "#f1f5f9",
			"success":        "#22c55e",
			"warning":        "#f59e0b",
			"error":          "#ef4444",
			"info":           "#3b82f6",
		},
		DarkColors: map[string]string{
			"primary":        "#3b82f6",
			"primary_hover":  "#60a5fa",
			"secondary":      "#94a3b8",
			"accent":         "#fbbf24",
			"background":     "#0f172a",
			"surface":        "#1e293b",
			"surface_alt":    "#334155",
			"text":           "#f1f5f9",
			"text_secondary": "#cbd5e1",
			"text_muted":     "#94a3b8",
			"heading":        "#f8fafc",
			"link":           "#60a5fa",
			"link_hover":     "#93c5fd",
			"border":         "#334155",
			"border_light":   "#1e293b",
			"success":        "#4ade80",
			"warning":        "#fbbf24",
			"error":          "#f87171",
			"info":           "#60a5fa",
		},
		Fonts: map[string]string{
			"heading": "'Inter', system-ui, sans-serif",
			"body":    "'Inter', system-ui, sans-serif",
			"mono":    "'JetBrains Mono', monospace",
		},
		Features: map[string]bool{
			"dark_mode":        true,
			"sticky_header":    true,
			"back_to_top":      true,
			"reading_progress": true,
			"comments":         true,
			"social_share":     true,
			"related_posts":    true,
			"author_box":       true,
			"search":           true,
			"newsletter":       true,
		},
	}
}

// getMenus builds menu contexts.
func (h *Handler) getMenus(c *mizu.Ctx) map[string]*MenuContext {
	ctx := c.Context()
	menuMap := make(map[string]*MenuContext)

	// Try to get menus from database
	locations := []string{"primary", "footer", "social"}
	for _, loc := range locations {
		menu, err := h.cfg.Menus.GetMenuByLocation(ctx, loc)
		if err != nil || menu == nil {
			menuMap[loc] = &MenuContext{Items: []*MenuItem{}}
			continue
		}

		menuItems := make([]*MenuItem, 0, len(menu.Items))
		for _, item := range menu.Items {
			if item.ParentID == "" {
				menuItems = append(menuItems, h.buildMenuItem(item, menu.Items))
			}
		}
		menuMap[loc] = &MenuContext{Items: menuItems}
	}

	return menuMap
}

func (h *Handler) buildMenuItem(item *menus.MenuItem, allItems []*menus.MenuItem) *MenuItem {
	mi := &MenuItem{
		Title:    item.Title,
		URL:      item.URL,
		Target:   item.Target,
		CSSClass: item.CSSClass,
	}

	// Find children
	for _, child := range allItems {
		if child.ParentID == item.ID {
			mi.Children = append(mi.Children, h.buildMenuItem(child, allItems))
		}
	}

	return mi
}

// getCategories fetches all categories.
func (h *Handler) getCategories(c *mizu.Ctx) []*categories.Category {
	cats, _, _ := h.cfg.Categories.List(c.Context(), &categories.ListIn{Limit: 20})
	return cats
}

// getTags fetches all tags.
func (h *Handler) getTags(c *mizu.Ctx) []*tags.Tag {
	ts, _, _ := h.cfg.Tags.List(c.Context(), &tags.ListIn{Limit: 20})
	return ts
}

// getRecentPosts fetches recent posts.
func (h *Handler) getRecentPosts(c *mizu.Ctx, limit int) []*PostView {
	ps, _, _ := h.cfg.Posts.List(c.Context(), &posts.ListIn{
		Status:  "published",
		Limit:   limit,
		OrderBy: "published_at",
		Order:   "desc",
	})
	return h.postsToViews(c, ps)
}

// baseData creates common template data.
func (h *Handler) baseData(c *mizu.Ctx, isHome, isSingle, isPage, isArchive, isSearch bool) BaseData {
	return BaseData{
		Site:  h.getSiteContext(c),
		Theme: h.getThemeContext(c),
		Request: RequestContext{
			URL:       h.cfg.BaseURL + c.Request().URL.Path,
			Path:      c.Request().URL.Path,
			IsHome:    isHome,
			IsSingle:  isSingle,
			IsPage:    isPage,
			IsArchive: isArchive,
			IsSearch:  isSearch,
		},
		Menus:       h.getMenus(c),
		User:        h.cfg.GetUser(c),
		Categories:  h.getCategories(c),
		Tags:        h.getTags(c),
		RecentPosts: h.getRecentPosts(c, 5),
	}
}

// postToView converts a post to a view model.
func (h *Handler) postToView(c *mizu.Ctx, p *posts.Post) *PostView {
	if p == nil {
		return nil
	}

	ctx := c.Context()

	// Render markdown content to HTML
	content := p.Content
	if p.ContentFormat == "markdown" || p.ContentFormat == "" {
		content = renderMarkdown(p.Content)
	}

	view := &PostView{
		ID:            p.ID,
		Title:         p.Title,
		Slug:          p.Slug,
		Excerpt:       p.Excerpt,
		Content:       content,
		PublishedAt:   p.PublishedAt,
		UpdatedAt:     p.UpdatedAt,
		ReadingTime:   p.ReadingTime,
		WordCount:     p.WordCount,
		AllowComments: p.AllowComments,
		IsFeatured:    p.IsFeatured,
		IsSticky:      p.IsSticky,
	}

	// Get author
	if p.AuthorID != "" {
		author, err := h.cfg.Users.GetByID(ctx, p.AuthorID)
		if err == nil {
			view.Author = author
		}
	}

	// Get featured image
	if p.FeaturedImageID != "" {
		img, err := h.cfg.Media.GetByID(ctx, p.FeaturedImageID)
		if err == nil {
			view.FeaturedImage = img
		}
	}

	// Get categories
	catIDs, _ := h.cfg.Posts.GetCategoryIDs(ctx, p.ID)
	for _, catID := range catIDs {
		cat, err := h.cfg.Categories.GetByID(ctx, catID)
		if err == nil {
			view.Categories = append(view.Categories, cat)
		}
	}

	// Get tags
	tagIDs, _ := h.cfg.Posts.GetTagIDs(ctx, p.ID)
	for _, tagID := range tagIDs {
		tag, err := h.cfg.Tags.GetByID(ctx, tagID)
		if err == nil {
			view.Tags = append(view.Tags, tag)
		}
	}

	return view
}

// postsToViews converts multiple posts to view models.
func (h *Handler) postsToViews(c *mizu.Ctx, ps []*posts.Post) []*PostView {
	views := make([]*PostView, 0, len(ps))
	for _, p := range ps {
		views = append(views, h.postToView(c, p))
	}
	return views
}

// pageToView converts a page to a view model.
func (h *Handler) pageToView(c *mizu.Ctx, p *pages.Page) *PageView {
	if p == nil {
		return nil
	}

	ctx := c.Context()

	// Render markdown content to HTML
	content := p.Content
	if p.ContentFormat == "markdown" || p.ContentFormat == "" {
		content = renderMarkdown(p.Content)
	}

	view := &PageView{
		ID:        p.ID,
		Title:     p.Title,
		Slug:      p.Slug,
		Excerpt:   "",
		Content:   content,
		Template:  p.Template,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	// Get author
	if p.AuthorID != "" {
		author, err := h.cfg.Users.GetByID(ctx, p.AuthorID)
		if err == nil {
			view.Author = author
		}
	}

	// Get featured image
	if p.FeaturedImageID != "" {
		img, err := h.cfg.Media.GetByID(ctx, p.FeaturedImageID)
		if err == nil {
			view.FeaturedImage = img
		}
	}

	return view
}

// buildPagination creates pagination data.
func (h *Handler) buildPagination(currentPage, total, perPage int, baseURL string) Pagination {
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	pag := Pagination{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		Total:       total,
		PerPage:     perPage,
		BaseURL:     baseURL,
	}

	if currentPage > 1 {
		pag.PrevURL = fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)
	}
	if currentPage < totalPages {
		pag.NextURL = fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)
	}

	// Build page numbers
	pages := make([]int, 0)
	for i := 1; i <= totalPages; i++ {
		if i == 1 || i == totalPages || (i >= currentPage-2 && i <= currentPage+2) {
			pages = append(pages, i)
		} else if len(pages) > 0 && pages[len(pages)-1] != -1 {
			pages = append(pages, -1) // ellipsis
		}
	}
	pag.Pages = pages

	return pag
}

// Home handles the homepage.
func (h *Handler) Home(c *mizu.Ctx) error {
	ctx := c.Context()
	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	offset := (page - 1) * perPage

	// Get featured posts
	featured, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:     "published",
		IsFeatured: boolPtr(true),
		Limit:      4,
		OrderBy:    "published_at",
		Order:      "desc",
	})

	// Get regular posts (excluding featured)
	allPosts, total, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:  "published",
		Limit:   perPage,
		Offset:  offset,
		OrderBy: "published_at",
		Order:   "desc",
	})

	data := HomeData{
		BaseData:   h.baseData(c, true, false, false, false, false),
		Posts:      h.postsToViews(c, allPosts),
		Featured:   h.postsToViews(c, featured),
		Pagination: h.buildPagination(page, total, perPage, "/"),
	}

	return h.render(c, "index", data)
}

// Post handles single post pages.
func (h *Handler) Post(c *mizu.Ctx) error {
	ctx := c.Context()
	slug := c.Param("slug")

	post, err := h.cfg.Posts.GetBySlug(ctx, slug)
	if err != nil || post == nil || post.Status != "published" {
		return h.NotFound(c)
	}

	postView := h.postToView(c, post)

	// Get related posts (same category)
	var related []*PostView
	if len(postView.Categories) > 0 {
		relatedPosts, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
			Status:     "published",
			CategoryID: postView.Categories[0].ID,
			Limit:      4,
			OrderBy:    "published_at",
			Order:      "desc",
		})
		for _, rp := range relatedPosts {
			if rp.ID != post.ID {
				related = append(related, h.postToView(c, rp))
			}
		}
		if len(related) > 3 {
			related = related[:3]
		}
	}

	// Get comments
	var commentViews []*CommentView
	if post.AllowComments {
		cmts, _, _ := h.cfg.Comments.ListByPost(ctx, post.ID, &comments.ListIn{
			Status: "approved",
			Limit:  100,
		})
		commentViews = h.buildCommentTree(c, cmts)
	}

	// Get prev/next posts
	var prevPost, nextPost *PostView
	allPosts, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:  "published",
		Limit:   100,
		OrderBy: "published_at",
		Order:   "desc",
	})
	for i, p := range allPosts {
		if p.ID == post.ID {
			if i > 0 {
				nextPost = h.postToView(c, allPosts[i-1])
			}
			if i < len(allPosts)-1 {
				prevPost = h.postToView(c, allPosts[i+1])
			}
			break
		}
	}

	data := PostData{
		BaseData: h.baseData(c, false, true, false, false, false),
		Post:     postView,
		Author:   postView.Author,
		PrevPost: prevPost,
		NextPost: nextPost,
		Related:  related,
		Comments: commentViews,
	}

	return h.render(c, "post", data)
}

// buildCommentTree organizes comments into a tree structure.
func (h *Handler) buildCommentTree(c *mizu.Ctx, cmts []*comments.Comment) []*CommentView {
	viewMap := make(map[string]*CommentView)
	var roots []*CommentView

	// First pass: create all views
	for _, cmt := range cmts {
		view := &CommentView{
			ID:         cmt.ID,
			Content:    cmt.Content,
			AuthorName: cmt.AuthorName,
			ParentID:   cmt.ParentID,
			CreatedAt:  cmt.CreatedAt,
		}
		if cmt.AuthorID != "" {
			author, _ := h.cfg.Users.GetByID(c.Context(), cmt.AuthorID)
			view.Author = author
		}
		viewMap[cmt.ID] = view
	}

	// Second pass: build tree
	for _, view := range viewMap {
		if view.ParentID == "" {
			roots = append(roots, view)
		} else if parent, ok := viewMap[view.ParentID]; ok {
			parent.Replies = append(parent.Replies, view)
		}
	}

	return roots
}

// Page handles single page display.
func (h *Handler) Page(c *mizu.Ctx) error {
	ctx := c.Context()
	slug := c.Param("slug")

	page, err := h.cfg.Pages.GetBySlug(ctx, slug)
	if err != nil || page == nil || page.Status != "published" {
		return h.NotFound(c)
	}

	pageView := h.pageToView(c, page)

	// Get child pages
	var childPages []*pages.Page
	allPages, _, _ := h.cfg.Pages.List(ctx, &pages.ListIn{Status: "published", Limit: 100})
	for _, p := range allPages {
		if p.ParentID == page.ID {
			childPages = append(childPages, p)
		}
	}

	data := PageData{
		BaseData:   h.baseData(c, false, false, true, false, false),
		Page:       pageView,
		ChildPages: childPages,
	}

	return h.render(c, "page", data)
}

// Category handles category archive pages.
func (h *Handler) Category(c *mizu.Ctx) error {
	ctx := c.Context()
	slug := c.Param("slug")

	cat, err := h.cfg.Categories.GetBySlug(ctx, slug)
	if err != nil || cat == nil {
		return h.NotFound(c)
	}

	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	offset := (page - 1) * perPage

	ps, total, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:     "published",
		CategoryID: cat.ID,
		Limit:      perPage,
		Offset:     offset,
		OrderBy:    "published_at",
		Order:      "desc",
	})

	data := ArchiveData{
		BaseData:   h.baseData(c, false, false, false, true, false),
		Posts:      h.postsToViews(c, ps),
		Pagination: h.buildPagination(page, total, perPage, "/category/"+slug),
		Category:   cat,
	}

	return h.render(c, "category", data)
}

// Tag handles tag archive pages.
func (h *Handler) Tag(c *mizu.Ctx) error {
	ctx := c.Context()
	slug := c.Param("slug")

	tag, err := h.cfg.Tags.GetBySlug(ctx, slug)
	if err != nil || tag == nil {
		return h.NotFound(c)
	}

	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	offset := (page - 1) * perPage

	ps, total, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:  "published",
		TagID:   tag.ID,
		Limit:   perPage,
		Offset:  offset,
		OrderBy: "published_at",
		Order:   "desc",
	})

	data := ArchiveData{
		BaseData:   h.baseData(c, false, false, false, true, false),
		Posts:      h.postsToViews(c, ps),
		Pagination: h.buildPagination(page, total, perPage, "/tag/"+slug),
		Tag:        tag,
	}

	return h.render(c, "tag", data)
}

// Author handles author archive pages.
func (h *Handler) Author(c *mizu.Ctx) error {
	ctx := c.Context()
	slug := c.Param("slug")

	author, err := h.cfg.Users.GetBySlug(ctx, slug)
	if err != nil || author == nil {
		return h.NotFound(c)
	}

	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	offset := (page - 1) * perPage

	ps, total, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:   "published",
		AuthorID: author.ID,
		Limit:    perPage,
		Offset:   offset,
		OrderBy:  "published_at",
		Order:    "desc",
	})

	data := ArchiveData{
		BaseData:   h.baseData(c, false, false, false, true, false),
		Posts:      h.postsToViews(c, ps),
		Pagination: h.buildPagination(page, total, perPage, "/author/"+slug),
		AuthorData: author,
	}

	return h.render(c, "author", data)
}

// Archive handles the general archive page.
func (h *Handler) Archive(c *mizu.Ctx) error {
	ctx := c.Context()

	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	offset := (page - 1) * perPage

	ps, total, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:  "published",
		Limit:   perPage,
		Offset:  offset,
		OrderBy: "published_at",
		Order:   "desc",
	})

	data := ArchiveData{
		BaseData:   h.baseData(c, false, false, false, true, false),
		Posts:      h.postsToViews(c, ps),
		Pagination: h.buildPagination(page, total, perPage, "/archive"),
	}

	return h.render(c, "archive", data)
}

// Search handles search results.
func (h *Handler) Search(c *mizu.Ctx) error {
	ctx := c.Context()
	query := strings.TrimSpace(c.Query("q"))

	page := 1
	if p := c.Query("page"); p != "" {
		page, _ = strconv.Atoi(p)
		if page < 1 {
			page = 1
		}
	}

	perPage := 10
	offset := (page - 1) * perPage

	var postViews []*PostView
	var pageViews []*PageView
	var total int

	if query != "" {
		// Search posts
		ps, postTotal, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
			Status:  "published",
			Search:  query,
			Limit:   perPage,
			Offset:  offset,
			OrderBy: "published_at",
			Order:   "desc",
		})
		postViews = h.postsToViews(c, ps)
		total = postTotal

		// Search pages
		pgs, _, _ := h.cfg.Pages.List(ctx, &pages.ListIn{
			Status: "published",
			Search: query,
			Limit:  10,
		})
		for _, pg := range pgs {
			pageViews = append(pageViews, h.pageToView(c, pg))
		}
	}

	data := SearchData{
		BaseData:   h.baseData(c, false, false, false, false, true),
		Query:      query,
		Posts:      postViews,
		Pages:      pageViews,
		Total:      total + len(pageViews),
		Pagination: h.buildPagination(page, total, perPage, "/search"),
	}

	return h.render(c, "search", data)
}

// NotFound handles 404 pages.
func (h *Handler) NotFound(c *mizu.Ctx) error {
	data := ErrorData{
		BaseData:  h.baseData(c, false, false, false, false, false),
		ErrorCode: 404,
	}
	data.Request.Is404 = true

	return h.renderWithStatus(c, "error", data, http.StatusNotFound)
}

// ServerError handles 500 pages.
func (h *Handler) ServerError(c *mizu.Ctx) error {
	data := ErrorData{
		BaseData:     h.baseData(c, false, false, false, false, false),
		ErrorCode:    500,
		ErrorMessage: "Internal server error",
	}

	return h.renderWithStatus(c, "error", data, http.StatusInternalServerError)
}

// Feed handles RSS feed.
func (h *Handler) Feed(c *mizu.Ctx) error {
	ctx := c.Context()

	ps, _, _ := h.cfg.Posts.List(ctx, &posts.ListIn{
		Status:  "published",
		Limit:   20,
		OrderBy: "published_at",
		Order:   "desc",
	})

	site := h.getSiteContext(c)

	c.Writer().Header().Set("Content-Type", "application/rss+xml; charset=utf-8")

	// Build RSS feed
	rss := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
<title>%s</title>
<link>%s</link>
<description>%s</description>
<language>%s</language>
<atom:link href="%s/feed" rel="self" type="application/rss+xml"/>`,
		site.Name, site.URL, site.Description, site.Language, site.URL)

	for _, p := range ps {
		pubDate := ""
		if p.PublishedAt != nil {
			pubDate = p.PublishedAt.Format("Mon, 02 Jan 2006 15:04:05 -0700")
		}
		rss += fmt.Sprintf(`
<item>
<title>%s</title>
<link>%s/%s</link>
<guid>%s/%s</guid>
<pubDate>%s</pubDate>
<description><![CDATA[%s]]></description>
</item>`, p.Title, site.URL, p.Slug, site.URL, p.Slug, pubDate, p.Excerpt)
	}

	rss += `
</channel>
</rss>`

	return c.Text(http.StatusOK, rss)
}

func boolPtr(b bool) *bool {
	return &b
}
