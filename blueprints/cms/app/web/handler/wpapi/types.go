// Package wpapi provides WordPress REST API compatible handlers.
package wpapi

import (
	"hash/fnv"
	"time"
)

// WPRendered represents a WordPress rendered field.
type WPRendered struct {
	Rendered string `json:"rendered"`
	Raw      string `json:"raw,omitempty"`
}

// WPContent represents a WordPress content field with protection flag.
type WPContent struct {
	Rendered  string `json:"rendered"`
	Raw       string `json:"raw,omitempty"`
	Protected bool   `json:"protected"`
}

// WPLink represents a HAL-style link.
type WPLink struct {
	Href       string `json:"href"`
	Embeddable bool   `json:"embeddable,omitempty"`
	Taxonomy   string `json:"taxonomy,omitempty"`
	Count      int    `json:"count,omitempty"`
	Templated  bool   `json:"templated,omitempty"`
}

// WPError represents a WordPress API error response.
type WPError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    WPErrorData `json:"data"`
}

// WPErrorData contains error metadata.
type WPErrorData struct {
	Status int                    `json:"status"`
	Params map[string]string      `json:"params,omitempty"`
	Details map[string]WPErrorDetail `json:"details,omitempty"`
}

// WPErrorDetail contains validation error details.
type WPErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WPPost represents a WordPress post.
type WPPost struct {
	ID            int64              `json:"id"`
	Date          string             `json:"date"`
	DateGMT       string             `json:"date_gmt"`
	GUID          WPRendered         `json:"guid"`
	Modified      string             `json:"modified"`
	ModifiedGMT   string             `json:"modified_gmt"`
	Slug          string             `json:"slug"`
	Status        string             `json:"status"`
	Type          string             `json:"type"`
	Link          string             `json:"link"`
	Title         WPRendered         `json:"title"`
	Content       WPContent          `json:"content"`
	Excerpt       WPContent          `json:"excerpt"`
	Author        int64              `json:"author"`
	FeaturedMedia int64              `json:"featured_media"`
	CommentStatus string             `json:"comment_status"`
	PingStatus    string             `json:"ping_status"`
	Sticky        bool               `json:"sticky"`
	Template      string             `json:"template"`
	Format        string             `json:"format"`
	Meta          []any              `json:"meta"`
	Categories    []int64            `json:"categories"`
	Tags          []int64            `json:"tags"`
	Links         map[string][]WPLink `json:"_links,omitempty"`
}

// WPPage represents a WordPress page.
type WPPage struct {
	ID            int64              `json:"id"`
	Date          string             `json:"date"`
	DateGMT       string             `json:"date_gmt"`
	GUID          WPRendered         `json:"guid"`
	Modified      string             `json:"modified"`
	ModifiedGMT   string             `json:"modified_gmt"`
	Slug          string             `json:"slug"`
	Status        string             `json:"status"`
	Type          string             `json:"type"`
	Link          string             `json:"link"`
	Title         WPRendered         `json:"title"`
	Content       WPContent          `json:"content"`
	Excerpt       WPContent          `json:"excerpt"`
	Author        int64              `json:"author"`
	FeaturedMedia int64              `json:"featured_media"`
	Parent        int64              `json:"parent"`
	MenuOrder     int                `json:"menu_order"`
	CommentStatus string             `json:"comment_status"`
	PingStatus    string             `json:"ping_status"`
	Template      string             `json:"template"`
	Meta          []any              `json:"meta"`
	Links         map[string][]WPLink `json:"_links,omitempty"`
}

// WPUser represents a WordPress user.
type WPUser struct {
	ID                int64              `json:"id"`
	Username          string             `json:"username,omitempty"`
	Name              string             `json:"name"`
	FirstName         string             `json:"first_name,omitempty"`
	LastName          string             `json:"last_name,omitempty"`
	Email             string             `json:"email,omitempty"`
	URL               string             `json:"url"`
	Description       string             `json:"description"`
	Link              string             `json:"link"`
	Locale            string             `json:"locale,omitempty"`
	Nickname          string             `json:"nickname,omitempty"`
	Slug              string             `json:"slug"`
	RegisteredDate    string             `json:"registered_date,omitempty"`
	Roles             []string           `json:"roles,omitempty"`
	Capabilities      map[string]bool    `json:"capabilities,omitempty"`
	ExtraCapabilities map[string]bool    `json:"extra_capabilities,omitempty"`
	AvatarURLs        map[string]string  `json:"avatar_urls"`
	Meta              []any              `json:"meta"`
	Links             map[string][]WPLink `json:"_links,omitempty"`
}

// WPCategory represents a WordPress category.
type WPCategory struct {
	ID          int64              `json:"id"`
	Count       int                `json:"count"`
	Description string             `json:"description"`
	Link        string             `json:"link"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Taxonomy    string             `json:"taxonomy"`
	Parent      int64              `json:"parent"`
	Meta        []any              `json:"meta"`
	Links       map[string][]WPLink `json:"_links,omitempty"`
}

// WPTag represents a WordPress tag.
type WPTag struct {
	ID          int64              `json:"id"`
	Count       int                `json:"count"`
	Description string             `json:"description"`
	Link        string             `json:"link"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Taxonomy    string             `json:"taxonomy"`
	Meta        []any              `json:"meta"`
	Links       map[string][]WPLink `json:"_links,omitempty"`
}

// WPMedia represents a WordPress media item.
type WPMedia struct {
	ID            int64              `json:"id"`
	Date          string             `json:"date"`
	DateGMT       string             `json:"date_gmt"`
	GUID          WPRendered         `json:"guid"`
	Modified      string             `json:"modified"`
	ModifiedGMT   string             `json:"modified_gmt"`
	Slug          string             `json:"slug"`
	Status        string             `json:"status"`
	Type          string             `json:"type"`
	Link          string             `json:"link"`
	Title         WPRendered         `json:"title"`
	Author        int64              `json:"author"`
	CommentStatus string             `json:"comment_status"`
	PingStatus    string             `json:"ping_status"`
	Template      string             `json:"template"`
	Meta          []any              `json:"meta"`
	Description   WPContent          `json:"description"`
	Caption       WPContent          `json:"caption"`
	AltText       string             `json:"alt_text"`
	MediaType     string             `json:"media_type"`
	MimeType      string             `json:"mime_type"`
	MediaDetails  WPMediaDetails     `json:"media_details"`
	Post          int64              `json:"post"`
	SourceURL     string             `json:"source_url"`
	Links         map[string][]WPLink `json:"_links,omitempty"`
}

// WPMediaDetails contains media file details.
type WPMediaDetails struct {
	Width    int                    `json:"width,omitempty"`
	Height   int                    `json:"height,omitempty"`
	File     string                 `json:"file,omitempty"`
	FileSize int64                  `json:"filesize,omitempty"`
	Sizes    map[string]WPImageSize `json:"sizes,omitempty"`
}

// WPImageSize represents an image size variant.
type WPImageSize struct {
	File      string `json:"file"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	MimeType  string `json:"mime_type"`
	SourceURL string `json:"source_url"`
}

// WPComment represents a WordPress comment.
type WPComment struct {
	ID               int64              `json:"id"`
	Post             int64              `json:"post"`
	Parent           int64              `json:"parent"`
	Author           int64              `json:"author"`
	AuthorName       string             `json:"author_name"`
	AuthorEmail      string             `json:"author_email,omitempty"`
	AuthorURL        string             `json:"author_url"`
	AuthorIP         string             `json:"author_ip,omitempty"`
	AuthorUserAgent  string             `json:"author_user_agent,omitempty"`
	Date             string             `json:"date"`
	DateGMT          string             `json:"date_gmt"`
	Content          WPContent          `json:"content"`
	Link             string             `json:"link"`
	Status           string             `json:"status"`
	Type             string             `json:"type"`
	AuthorAvatarURLs map[string]string  `json:"author_avatar_urls"`
	Meta             []any              `json:"meta"`
	Links            map[string][]WPLink `json:"_links,omitempty"`
}

// WPSettings represents WordPress site settings.
type WPSettings struct {
	Title                string `json:"title"`
	Description          string `json:"description"`
	URL                  string `json:"url"`
	Email                string `json:"email"`
	Timezone             string `json:"timezone"`
	DateFormat           string `json:"date_format"`
	TimeFormat           string `json:"time_format"`
	StartOfWeek          int    `json:"start_of_week"`
	Language             string `json:"language"`
	UseSmilies           bool   `json:"use_smilies"`
	DefaultCategory      int64  `json:"default_category"`
	DefaultPostFormat    string `json:"default_post_format"`
	PostsPerPage         int    `json:"posts_per_page"`
	ShowOnFront          string `json:"show_on_front"`
	PageOnFront          int64  `json:"page_on_front"`
	PageForPosts         int64  `json:"page_for_posts"`
	DefaultPingStatus    string `json:"default_ping_status"`
	DefaultCommentStatus string `json:"default_comment_status"`
	SiteLogo             int64  `json:"site_logo"`
	SiteIcon             int64  `json:"site_icon"`
}

// WPDiscovery represents the API discovery response.
type WPDiscovery struct {
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	URL             string                    `json:"url"`
	Home            string                    `json:"home"`
	GMTOffset       int                       `json:"gmt_offset"`
	TimezoneString  string                    `json:"timezone_string"`
	Namespaces      []string                  `json:"namespaces"`
	Authentication  map[string]any            `json:"authentication"`
	Routes          map[string]WPRoute        `json:"routes"`
}

// WPRoute represents an API route definition.
type WPRoute struct {
	Namespace string         `json:"namespace"`
	Methods   []string       `json:"methods"`
	Endpoints []WPEndpoint   `json:"endpoints"`
	Links     WPRouteLinks   `json:"_links,omitempty"`
}

// WPEndpoint represents an endpoint definition.
type WPEndpoint struct {
	Methods []string               `json:"methods"`
	Args    map[string]WPArg       `json:"args"`
}

// WPArg represents an endpoint argument definition.
type WPArg struct {
	Description string   `json:"description,omitempty"`
	Type        any      `json:"type,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

// WPRouteLinks contains route-level links.
type WPRouteLinks struct {
	Self []WPLink `json:"self,omitempty"`
}

// Create request types

// WPCreatePostRequest represents a request to create a post.
type WPCreatePostRequest struct {
	Date          string  `json:"date,omitempty"`
	DateGMT       string  `json:"date_gmt,omitempty"`
	Slug          string  `json:"slug,omitempty"`
	Status        string  `json:"status,omitempty"`
	Password      string  `json:"password,omitempty"`
	Title         any     `json:"title,omitempty"`      // Can be string or {raw: string}
	Content       any     `json:"content,omitempty"`    // Can be string or {raw: string}
	Excerpt       any     `json:"excerpt,omitempty"`    // Can be string or {raw: string}
	Author        int64   `json:"author,omitempty"`
	FeaturedMedia int64   `json:"featured_media,omitempty"`
	CommentStatus string  `json:"comment_status,omitempty"`
	PingStatus    string  `json:"ping_status,omitempty"`
	Format        string  `json:"format,omitempty"`
	Meta          any     `json:"meta,omitempty"`
	Sticky        *bool   `json:"sticky,omitempty"`
	Template      string  `json:"template,omitempty"`
	Categories    []int64 `json:"categories,omitempty"`
	Tags          []int64 `json:"tags,omitempty"`
}

// WPCreatePageRequest represents a request to create a page.
type WPCreatePageRequest struct {
	Date          string `json:"date,omitempty"`
	DateGMT       string `json:"date_gmt,omitempty"`
	Slug          string `json:"slug,omitempty"`
	Status        string `json:"status,omitempty"`
	Password      string `json:"password,omitempty"`
	Parent        int64  `json:"parent,omitempty"`
	Title         any    `json:"title,omitempty"`
	Content       any    `json:"content,omitempty"`
	Excerpt       any    `json:"excerpt,omitempty"`
	Author        int64  `json:"author,omitempty"`
	FeaturedMedia int64  `json:"featured_media,omitempty"`
	CommentStatus string `json:"comment_status,omitempty"`
	PingStatus    string `json:"ping_status,omitempty"`
	MenuOrder     int    `json:"menu_order,omitempty"`
	Meta          any    `json:"meta,omitempty"`
	Template      string `json:"template,omitempty"`
}

// WPCreateUserRequest represents a request to create a user.
type WPCreateUserRequest struct {
	Username    string   `json:"username"`
	Name        string   `json:"name,omitempty"`
	FirstName   string   `json:"first_name,omitempty"`
	LastName    string   `json:"last_name,omitempty"`
	Email       string   `json:"email"`
	URL         string   `json:"url,omitempty"`
	Description string   `json:"description,omitempty"`
	Locale      string   `json:"locale,omitempty"`
	Nickname    string   `json:"nickname,omitempty"`
	Slug        string   `json:"slug,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Password    string   `json:"password"`
	Meta        any      `json:"meta,omitempty"`
}

// WPCreateCategoryRequest represents a request to create a category.
type WPCreateCategoryRequest struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Parent      int64  `json:"parent,omitempty"`
	Meta        any    `json:"meta,omitempty"`
}

// WPCreateTagRequest represents a request to create a tag.
type WPCreateTagRequest struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug,omitempty"`
	Meta        any    `json:"meta,omitempty"`
}

// WPCreateCommentRequest represents a request to create a comment.
type WPCreateCommentRequest struct {
	Author      int64  `json:"author,omitempty"`
	AuthorEmail string `json:"author_email,omitempty"`
	AuthorIP    string `json:"author_ip,omitempty"`
	AuthorName  string `json:"author_name,omitempty"`
	AuthorURL   string `json:"author_url,omitempty"`
	Content     any    `json:"content"`
	Date        string `json:"date,omitempty"`
	DateGMT     string `json:"date_gmt,omitempty"`
	Parent      int64  `json:"parent,omitempty"`
	Post        int64  `json:"post"`
	Status      string `json:"status,omitempty"`
	Meta        any    `json:"meta,omitempty"`
}

// WPUpdateMediaRequest represents a request to update media metadata.
type WPUpdateMediaRequest struct {
	Date          string `json:"date,omitempty"`
	DateGMT       string `json:"date_gmt,omitempty"`
	Slug          string `json:"slug,omitempty"`
	Status        string `json:"status,omitempty"`
	Title         any    `json:"title,omitempty"`
	Author        int64  `json:"author,omitempty"`
	CommentStatus string `json:"comment_status,omitempty"`
	PingStatus    string `json:"ping_status,omitempty"`
	Meta          any    `json:"meta,omitempty"`
	AltText       string `json:"alt_text,omitempty"`
	Caption       any    `json:"caption,omitempty"`
	Description   any    `json:"description,omitempty"`
	Post          int64  `json:"post,omitempty"`
}

// Helper functions

// ULIDToNumericID converts a ULID string to a stable numeric ID.
func ULIDToNumericID(ulid string) int64 {
	h := fnv.New64a()
	h.Write([]byte(ulid))
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
}

// FormatWPDateTime formats a time for WordPress API.
func FormatWPDateTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}

// FormatWPDateTimeGMT formats a time in UTC for WordPress API.
func FormatWPDateTimeGMT(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05")
}

// ExtractRawContent extracts raw content from a WordPress content field.
// WordPress accepts both string and {raw: string} formats.
func ExtractRawContent(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if raw, ok := val["raw"].(string); ok {
			return raw
		}
		if rendered, ok := val["rendered"].(string); ok {
			return rendered
		}
	}
	return ""
}
