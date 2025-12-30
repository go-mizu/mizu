// Package options provides site settings management.
package options

import (
	"context"
)

// Default option names
const (
	OptionSiteURL        = "siteurl"
	OptionHome           = "home"
	OptionBlogName       = "blogname"
	OptionBlogDescription = "blogdescription"
	OptionAdminEmail     = "admin_email"
	OptionUsersCanRegister = "users_can_register"
	OptionDefaultRole    = "default_role"
	OptionTimezone       = "timezone_string"
	OptionDateFormat     = "date_format"
	OptionTimeFormat     = "time_format"
	OptionStartOfWeek    = "start_of_week"
	OptionLanguage       = "WPLANG"

	OptionPostsPerPage      = "posts_per_page"
	OptionShowOnFront       = "show_on_front"
	OptionPageOnFront       = "page_on_front"
	OptionPageForPosts      = "page_for_posts"
	OptionBlogPublic        = "blog_public"

	OptionDefaultCategory   = "default_category"
	OptionDefaultPostFormat = "default_post_format"

	OptionDefaultCommentStatus = "default_comment_status"
	OptionDefaultPingStatus    = "default_ping_status"
	OptionCommentModeration    = "comment_moderation"
	OptionCommentRegistration  = "comment_registration"
	OptionCloseCommentsForOld  = "close_comments_for_old_posts"
	OptionThreadComments       = "thread_comments"
	OptionThreadCommentsDepth  = "thread_comments_depth"
	OptionCommentsPerPage      = "comments_per_page"
	OptionDefaultCommentsPage  = "default_comments_page"
	OptionCommentOrder         = "comment_order"

	OptionPermalinks       = "permalink_structure"
	OptionCategoryBase     = "category_base"
	OptionTagBase          = "tag_base"

	OptionThumbnailSizeW   = "thumbnail_size_w"
	OptionThumbnailSizeH   = "thumbnail_size_h"
	OptionMediumSizeW      = "medium_size_w"
	OptionMediumSizeH      = "medium_size_h"
	OptionLargeSizeW       = "large_size_w"
	OptionLargeSizeH       = "large_size_h"
	OptionUploadPath       = "upload_path"
	OptionUploadsUseYearMonth = "uploads_use_yearmonth_folders"

	OptionActiveTheme      = "template"
	OptionStylesheet       = "stylesheet"
	OptionActivePlugins    = "active_plugins"

	OptionStickyPosts      = "sticky_posts"
)

// Settings represents WordPress site settings (exposed via REST API).
type Settings struct {
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
	DefaultCategory      string `json:"default_category"`
	DefaultPostFormat    string `json:"default_post_format"`
	PostsPerPage         int    `json:"posts_per_page"`
	ShowOnFront          string `json:"show_on_front"`
	PageOnFront          string `json:"page_on_front"`
	PageForPosts         string `json:"page_for_posts"`
	DefaultPingStatus    string `json:"default_ping_status"`
	DefaultCommentStatus string `json:"default_comment_status"`
	SiteIcon             string `json:"site_icon"`
	SiteLogo             string `json:"site_logo"`
}

// UpdateSettingsIn contains input for updating settings.
type UpdateSettingsIn struct {
	Title                *string `json:"title,omitempty"`
	Description          *string `json:"description,omitempty"`
	URL                  *string `json:"url,omitempty"`
	Email                *string `json:"email,omitempty"`
	Timezone             *string `json:"timezone,omitempty"`
	DateFormat           *string `json:"date_format,omitempty"`
	TimeFormat           *string `json:"time_format,omitempty"`
	StartOfWeek          *int    `json:"start_of_week,omitempty"`
	Language             *string `json:"language,omitempty"`
	DefaultCategory      *string `json:"default_category,omitempty"`
	DefaultPostFormat    *string `json:"default_post_format,omitempty"`
	PostsPerPage         *int    `json:"posts_per_page,omitempty"`
	ShowOnFront          *string `json:"show_on_front,omitempty"`
	PageOnFront          *string `json:"page_on_front,omitempty"`
	PageForPosts         *string `json:"page_for_posts,omitempty"`
	DefaultPingStatus    *string `json:"default_ping_status,omitempty"`
	DefaultCommentStatus *string `json:"default_comment_status,omitempty"`
}

// API defines the options service interface.
type API interface {
	// Option management
	Get(ctx context.Context, name string) (string, error)
	GetWithDefault(ctx context.Context, name, defaultValue string) (string, error)
	Set(ctx context.Context, name, value string) error
	Delete(ctx context.Context, name string) error

	// Batch operations
	GetMultiple(ctx context.Context, names []string) (map[string]string, error)
	SetMultiple(ctx context.Context, options map[string]string) error

	// Settings (REST API format)
	GetSettings(ctx context.Context) (*Settings, error)
	UpdateSettings(ctx context.Context, in UpdateSettingsIn) (*Settings, error)

	// Autoload
	GetAutoloaded(ctx context.Context) (map[string]string, error)

	// Initialize default options
	InitDefaults(ctx context.Context, siteURL, siteTitle, adminEmail string) error
}
