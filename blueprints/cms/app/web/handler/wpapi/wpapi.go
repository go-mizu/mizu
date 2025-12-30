package wpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/settings"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// Handler handles all WordPress API endpoints.
type Handler struct {
	baseURL    string
	users      users.API
	posts      posts.API
	pages      pages.API
	categories categories.API
	tags       tags.API
	media      media.API
	comments   comments.API
	settings   settings.API
	getUserID  func(*mizu.Ctx) string
	getUser    func(*mizu.Ctx) *users.User
}

// Config configures the WordPress API handler.
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
	GetUserID  func(*mizu.Ctx) string
	GetUser    func(*mizu.Ctx) *users.User
}

// New creates a new WordPress API handler.
func New(cfg Config) *Handler {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}
	return &Handler{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		users:      cfg.Users,
		posts:      cfg.Posts,
		pages:      cfg.Pages,
		categories: cfg.Categories,
		tags:       cfg.Tags,
		media:      cfg.Media,
		comments:   cfg.Comments,
		settings:   cfg.Settings,
		getUserID:  cfg.GetUserID,
		getUser:    cfg.GetUser,
	}
}

// Context types for WordPress API requests.
const (
	ContextView  = "view"
	ContextEdit  = "edit"
	ContextEmbed = "embed"
)

// Common query parameters

// ListParams contains common list query parameters.
type ListParams struct {
	Context   string
	Page      int
	PerPage   int
	Search    string
	Exclude   []string
	Include   []string
	Offset    int
	Order     string
	OrderBy   string
}

// ParseListParams parses common list query parameters.
func ParseListParams(c *mizu.Ctx) ListParams {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}

	offset, _ := strconv.Atoi(c.Query("offset"))

	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	order := c.Query("order")
	if order == "" {
		order = "desc"
	}

	orderBy := c.Query("orderby")
	if orderBy == "" {
		orderBy = "date"
	}

	var exclude, include []string
	if e := c.Query("exclude"); e != "" {
		exclude = strings.Split(e, ",")
	}
	if i := c.Query("include"); i != "" {
		include = strings.Split(i, ",")
	}

	return ListParams{
		Context:   context,
		Page:      page,
		PerPage:   perPage,
		Search:    c.Query("search"),
		Exclude:   exclude,
		Include:   include,
		Offset:    offset,
		Order:     order,
		OrderBy:   orderBy,
	}
}

// Response helpers

// SetPaginationHeaders sets WordPress pagination headers.
func SetPaginationHeaders(c *mizu.Ctx, total, totalPages int) {
	c.Header().Set("X-WP-Total", strconv.Itoa(total))
	c.Header().Set("X-WP-TotalPages", strconv.Itoa(totalPages))
}

// OK returns a successful response.
func OK(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusOK, data)
}

// OKList returns a successful list response with pagination headers.
func OKList(c *mizu.Ctx, data any, total, page, perPage int) error {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	SetPaginationHeaders(c, total, totalPages)
	return c.JSON(http.StatusOK, data)
}

// Created returns a 201 created response.
func Created(c *mizu.Ctx, data any) error {
	return c.JSON(http.StatusCreated, data)
}

// Error responses

// ErrorNotFound returns a 404 not found error.
func ErrorNotFound(c *mizu.Ctx, code, message string) error {
	return c.JSON(http.StatusNotFound, WPError{
		Code:    code,
		Message: message,
		Data:    WPErrorData{Status: 404},
	})
}

// ErrorBadRequest returns a 400 bad request error.
func ErrorBadRequest(c *mizu.Ctx, code, message string) error {
	return c.JSON(http.StatusBadRequest, WPError{
		Code:    code,
		Message: message,
		Data:    WPErrorData{Status: 400},
	})
}

// ErrorUnauthorized returns a 401 unauthorized error.
func ErrorUnauthorized(c *mizu.Ctx) error {
	return c.JSON(http.StatusUnauthorized, WPError{
		Code:    "rest_not_logged_in",
		Message: "You are not currently logged in.",
		Data:    WPErrorData{Status: 401},
	})
}

// ErrorForbidden returns a 403 forbidden error.
func ErrorForbidden(c *mizu.Ctx, message string) error {
	return c.JSON(http.StatusForbidden, WPError{
		Code:    "rest_forbidden",
		Message: message,
		Data:    WPErrorData{Status: 403},
	})
}

// ErrorInternal returns a 500 internal server error.
func ErrorInternal(c *mizu.Ctx, code, message string) error {
	return c.JSON(http.StatusInternalServerError, WPError{
		Code:    code,
		Message: message,
		Data:    WPErrorData{Status: 500},
	})
}

// ErrorInvalidParam returns a 400 error for invalid parameters.
func ErrorInvalidParam(c *mizu.Ctx, param, message string) error {
	return c.JSON(http.StatusBadRequest, WPError{
		Code:    "rest_invalid_param",
		Message: "Invalid parameter(s): " + param,
		Data: WPErrorData{
			Status: 400,
			Params: map[string]string{param: message},
		},
	})
}

// URL helpers

// PostURL returns the URL for a post.
func (h *Handler) PostURL(slug string) string {
	return h.baseURL + "/" + slug + "/"
}

// PageURL returns the URL for a page.
func (h *Handler) PageURL(slug string) string {
	return h.baseURL + "/" + slug + "/"
}

// UserURL returns the URL for a user.
func (h *Handler) UserURL(slug string) string {
	return h.baseURL + "/author/" + slug + "/"
}

// CategoryURL returns the URL for a category.
func (h *Handler) CategoryURL(slug string) string {
	return h.baseURL + "/category/" + slug + "/"
}

// TagURL returns the URL for a tag.
func (h *Handler) TagURL(slug string) string {
	return h.baseURL + "/tag/" + slug + "/"
}

// MediaURL returns the URL for a media item.
func (h *Handler) MediaURL(url string) string {
	if strings.HasPrefix(url, "http") {
		return url
	}
	return h.baseURL + url
}

// CommentURL returns the URL for a comment.
func (h *Handler) CommentURL(postSlug string, commentID int64) string {
	return h.baseURL + "/" + postSlug + "/#comment-" + strconv.FormatInt(commentID, 10)
}

// APIURL returns the API URL for an endpoint.
func (h *Handler) APIURL(path string) string {
	return h.baseURL + "/wp-json/wp/v2" + path
}

// Link helpers

// SelfLink creates a self link.
func (h *Handler) SelfLink(path string) WPLink {
	return WPLink{Href: h.APIURL(path)}
}

// CollectionLink creates a collection link.
func (h *Handler) CollectionLink(path string) WPLink {
	return WPLink{Href: h.APIURL(path)}
}

// AboutLink creates an about link.
func (h *Handler) AboutLink(path string) WPLink {
	return WPLink{Href: h.APIURL(path)}
}

// EmbeddableLink creates an embeddable link.
func (h *Handler) EmbeddableLink(path string) WPLink {
	return WPLink{Href: h.APIURL(path), Embeddable: true}
}

// TaxonomyLink creates a taxonomy link.
func (h *Handler) TaxonomyLink(path, taxonomy string) WPLink {
	return WPLink{Href: h.APIURL(path), Embeddable: true, Taxonomy: taxonomy}
}

// Authentication helpers

// RequireAuth returns an error if user is not authenticated.
func (h *Handler) RequireAuth(c *mizu.Ctx) error {
	if h.getUserID(c) == "" {
		return ErrorUnauthorized(c)
	}
	return nil
}

// IsAuthenticated checks if the user is authenticated.
func (h *Handler) IsAuthenticated(c *mizu.Ctx) bool {
	return h.getUserID(c) != ""
}

// ID conversion helpers

// ParseID parses an ID from the URL parameter.
// It returns the string ID (ULID or numeric string).
func ParseID(c *mizu.Ctx) string {
	return c.Param("id")
}

// Gravatar URL helper
func GravatarURL(email string, size int) string {
	// For simplicity, return a default avatar URL
	// In production, implement actual Gravatar hash
	return "https://secure.gravatar.com/avatar/?s=" + strconv.Itoa(size) + "&d=mm&r=g"
}

// AvatarURLs generates avatar URLs at different sizes.
func AvatarURLs(email string) map[string]string {
	return map[string]string{
		"24": GravatarURL(email, 24),
		"48": GravatarURL(email, 48),
		"96": GravatarURL(email, 96),
	}
}

// Status mapping helpers

// MapPostStatus maps internal status to WordPress status.
func MapPostStatus(status, visibility string) string {
	if visibility == "private" {
		return "private"
	}
	switch status {
	case "published":
		return "publish"
	case "draft":
		return "draft"
	case "pending":
		return "pending"
	case "scheduled":
		return "future"
	default:
		return status
	}
}

// MapWPPostStatus maps WordPress status to internal status.
func MapWPPostStatus(wpStatus string) (status, visibility string) {
	switch wpStatus {
	case "publish":
		return "published", "public"
	case "draft":
		return "draft", "public"
	case "pending":
		return "pending", "public"
	case "future":
		return "scheduled", "public"
	case "private":
		return "published", "private"
	default:
		return wpStatus, "public"
	}
}

// MapCommentStatus maps internal comment status to WordPress status.
func MapCommentStatus(status string) string {
	switch status {
	case "approved":
		return "approved"
	case "pending":
		return "hold"
	case "spam":
		return "spam"
	case "trash":
		return "trash"
	default:
		return status
	}
}

// MapWPCommentStatus maps WordPress comment status to internal status.
func MapWPCommentStatus(wpStatus string) string {
	switch wpStatus {
	case "approved", "1":
		return "approved"
	case "hold", "0":
		return "pending"
	case "spam":
		return "spam"
	case "trash":
		return "trash"
	default:
		return wpStatus
	}
}
