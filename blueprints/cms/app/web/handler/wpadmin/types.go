// Package wpadmin provides WordPress-compatible admin interface handlers.
package wpadmin

import (
	"time"

	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/menus"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// MenuItem represents a navigation menu item in the admin sidebar.
type MenuItem struct {
	ID        string
	Title     string
	URL       string
	Icon      string // Dashicon class name
	Active    bool
	Open      bool // Whether submenu is expanded
	Children  []MenuItem
	Badge     int    // Notification count (e.g., pending comments)
	Cap       string // Required capability
	Separator bool   // Whether this is a separator line
}

// Breadcrumb represents a navigation breadcrumb.
type Breadcrumb struct {
	Label string
	URL   string
}

// AdminNotice represents a dismissible notice message.
type AdminNotice struct {
	Type       string // "success", "error", "warning", "info"
	Message    string
	Dismissble bool
}

// Pagination holds pagination state for list views.
type Pagination struct {
	CurrentPage int
	TotalPages  int
	TotalItems  int
	PerPage     int
	BaseURL     string
}

// BulkAction represents an available bulk action.
type BulkAction struct {
	Value string
	Label string
}

// TableColumn represents a column in a list table.
type TableColumn struct {
	ID       string
	Label    string
	Sortable bool
	Primary  bool
	Class    string
}

// StatusTab represents a status filter tab.
type StatusTab struct {
	Status string
	Label  string
	Count  int
	Active bool
	URL    string
}

// RoleTab represents a role filter tab for users list.
type RoleTab struct {
	Role   string
	Label  string
	Count  int
	Active bool
	URL    string
}

// SelectOption represents an option in a select dropdown.
type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

// PostStatus represents a post status option.
type PostStatus struct {
	Value string
	Label string
}

// Visibility represents post visibility option.
type Visibility struct {
	Value string
	Label string
}

// PostFormat represents a post format option.
type PostFormat struct {
	Value string
	Label string
}

// AtAGlance holds "At a Glance" widget data.
type AtAGlance struct {
	PostCount     int
	PageCount     int
	CommentCount  int
	CategoryCount int
	TagCount      int
	UserCount     int
	MediaCount    int
	Theme         string
	GoVersion     string
}

// ActivityItem represents an item in the activity widget.
type ActivityItem struct {
	Type      string // "post", "comment", "page"
	Title     string
	Author    string
	AuthorURL string
	Date      time.Time
	Status    string
	EditURL   string
	ViewURL   string
}

// SiteHealth represents site health status.
type SiteHealth struct {
	Status   string // "good", "recommended", "critical"
	Tests    int
	Passed   int
	Issues   int
	Critical int
}

// QuickDraftData holds data for the Quick Draft widget.
type QuickDraftData struct {
	Title   string
	Content string
	Enabled bool
}

// RecentDraft represents a recent draft in the Quick Draft widget.
type RecentDraft struct {
	ID       string
	Title    string
	Date     time.Time
	EditURL  string
}

// DashboardData holds data for the dashboard page.
type DashboardData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice

	SiteTitle       string
	SiteURL         string
	AtAGlance       AtAGlance
	Activity        []ActivityItem
	QuickDraft      QuickDraftData
	RecentDrafts    []RecentDraft
	SiteHealth      SiteHealth
	RecentComments  []*CommentRow
	WelcomePanel    bool
}

// PostRow represents a post in the list table.
type PostRow struct {
	*posts.Post
	Author        *users.User
	Categories    []*categories.Category
	Tags          []*tags.Tag
	CommentCount  int
	ThumbnailURL  string
	RowActions    []RowAction
}

// RowAction represents an action link in a table row.
type RowAction struct {
	Label  string
	URL    string
	Class  string
	Attr   string // Additional attributes
}

// PostsListData holds data for the posts list page.
type PostsListData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Posts       []*PostRow
	Columns     []TableColumn
	Pagination  Pagination
	BulkActions []BulkAction
	StatusTabs  []StatusTab
	ActiveTab   string
	SearchQuery string
	OrderBy     string
	Order       string
	PostType    string
	DateFilter  string
	CatFilter   string
}

// PostEditData holds data for the post edit page.
type PostEditData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Post              *posts.Post
	IsNew             bool
	PostType          string
	SelectedCategories []string
	AllCategories     []*CategoryOption
	SelectedTags      []string
	AllTags           []*tags.Tag
	FeaturedMedia     *media.Media
	Statuses          []PostStatus
	Visibilities      []Visibility
	CurrentStatus     string
	CurrentVisibility string
	PublishDate       string
	PublishTime       string
	Excerpt           string
	Slug              string
	Template          string
	Templates         []SelectOption
}

// CategoryOption represents a category in the category checklist.
type CategoryOption struct {
	*categories.Category
	Depth    int
	Selected bool
	Children []*CategoryOption
}

// PageRow represents a page in the list table.
type PageRow struct {
	*pages.Page
	Author       *users.User
	CommentCount int
	Depth        int
	RowActions   []RowAction
}

// PagesListData holds data for the pages list page.
type PagesListData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Pages       []*PageRow
	Columns     []TableColumn
	Pagination  Pagination
	BulkActions []BulkAction
	StatusTabs  []StatusTab
	ActiveTab   string
	SearchQuery string
	OrderBy     string
	Order       string
}

// PageEditData holds data for the page edit page.
type PageEditData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Page              *pages.Page
	IsNew             bool
	ParentPages       []*PageOption
	SelectedParent    string
	Statuses          []PostStatus
	Visibilities      []Visibility
	CurrentStatus     string
	CurrentVisibility string
	PublishDate       string
	PublishTime       string
	MenuOrder         int
	Template          string
	Templates         []SelectOption
}

// PageOption represents a page in the parent dropdown.
type PageOption struct {
	*pages.Page
	Depth int
}

// MediaItem represents a media item in the library.
type MediaItem struct {
	*media.Media
	Uploader     *users.User
	ThumbnailURL string
	FileSize     string
	Dimensions   string
	Duration     string // For audio/video
	AttachedTo   []*AttachedPost
	RowActions   []RowAction
}

// AttachedPost represents a post that a media item is attached to.
type AttachedPost struct {
	ID      string
	Title   string
	Type    string
	EditURL string
}

// MediaLibraryData holds data for the media library page.
type MediaLibraryData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Items       []*MediaItem
	ViewMode    string // "grid" or "list"
	FilterType  string // "all", "image", "video", "audio", "document"
	FilterDate  string
	Pagination  Pagination
	SearchQuery string
	Columns     []TableColumn
	BulkActions []BulkAction
}

// MediaEditData holds data for the media edit page.
type MediaEditData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Media         *media.Media
	Uploader      *users.User
	ThumbnailURL  string
	FileURL       string
	FileSize      string
	Dimensions    string
	Duration      string
	MimeType      string
	AttachedTo    []*AttachedPost
	AltText       string
	Caption       string
	Description   string
}

// CommentRow represents a comment in the list table.
type CommentRow struct {
	*comments.Comment
	AuthorName    string
	AuthorEmail   string
	AuthorURL     string
	AuthorIP      string
	AuthorAvatar  string
	Post          *posts.Post
	PostTitle     string
	PostEditURL   string
	PostViewURL   string
	InReplyTo     *comments.Comment
	RowActions    []RowAction
	Pending       bool
}

// CommentsListData holds data for the comments list page.
type CommentsListData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Comments    []*CommentRow
	Columns     []TableColumn
	Pagination  Pagination
	BulkActions []BulkAction
	StatusTabs  []StatusTab
	ActiveTab   string
	SearchQuery string
	PostFilter  string
	TypeFilter  string
}

// CommentEditData holds data for the comment edit page.
type CommentEditData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Comment       *comments.Comment
	AuthorName    string
	AuthorEmail   string
	AuthorURL     string
	Post          *posts.Post
	PostTitle     string
	Statuses      []SelectOption
	CurrentStatus string
}

// UserRow represents a user in the list table.
type UserRow struct {
	*users.User
	PostCount   int
	AvatarURL   string
	Role        string
	RoleDisplay string
	LastLogin   time.Time
	RowActions  []RowAction
}

// UsersListData holds data for the users list page.
type UsersListData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Users       []*UserRow
	Columns     []TableColumn
	Pagination  Pagination
	BulkActions []BulkAction
	RoleTabs    []RoleTab
	ActiveRole  string
	SearchQuery string
	OrderBy     string
	Order       string
}

// UserEditData holds data for the user edit page.
type UserEditData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	EditUser      *users.User
	IsNew         bool
	IsSelf        bool
	Roles         []SelectOption
	CurrentRole   string
	AvatarURL     string
	ShowPassword  bool
}

// ProfileData holds data for the profile page.
type ProfileData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	AvatarURL       string
	ColorSchemes    []ColorScheme
	CurrentScheme   string
	AdminBarFront   bool
}

// ColorScheme represents an admin color scheme option.
type ColorScheme struct {
	ID     string
	Name   string
	Colors []string
}

// TaxonomyRow represents a term (category/tag) in the list table.
type TaxonomyRow struct {
	ID          string
	Name        string
	Slug        string
	Description string
	Parent      string
	ParentName  string
	Count       int
	RowActions  []RowAction
}

// TaxonomyListData holds data for categories/tags list page.
type TaxonomyListData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Taxonomy    string // "category" or "post_tag"
	TaxLabel    string // "Categories" or "Tags"
	Terms       []*TaxonomyRow
	Columns     []TableColumn
	Pagination  Pagination
	BulkActions []BulkAction
	SearchQuery string

	// For add form
	ParentTerms []*TaxonomyRow
	ShowParent  bool // Categories have parents, tags don't
}

// MenuItemView represents a menu item in the menu builder.
type MenuItemView struct {
	*menus.MenuItem
	TypeLabel   string // "Page", "Post", "Custom Link", etc.
	OriginalObj interface{}
	Children    []*MenuItemView
	Depth       int
}

// MenuLocation represents a theme menu location.
type MenuLocation struct {
	Name        string
	Description string
	AssignedID  string
}

// AvailableMenuItems holds available items to add to menus.
type AvailableMenuItems struct {
	Pages       []*pages.Page
	Posts       []*posts.Post
	Categories  []*categories.Category
	Tags        []*tags.Tag
	CustomLinks bool
}

// MenusData holds data for the menus page.
type MenusData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Menus          []*menus.Menu
	ActiveMenu     *menus.Menu
	MenuItems      []*MenuItemView
	Locations      []MenuLocation
	AvailableItems AvailableMenuItems
	IsNew          bool
}

// SettingsData holds data for settings pages.
type SettingsData struct {
	Title       string
	BodyClass   string
	User        *users.User
	Menu        []MenuItem
	Breadcrumbs []Breadcrumb
	Notices     []AdminNotice
	SiteTitle   string
	SiteURL     string

	Section    string // "general", "writing", "reading", etc.
	Settings   map[string]string
	Options    map[string][]SelectOption
	Timezones  []SelectOption
	DateFormats []SelectOption
	TimeFormats []SelectOption
}

// LoginData holds data for the login page.
type LoginData struct {
	Title        string
	SiteTitle    string
	SiteURL      string
	Error        string
	Message      string
	RedirectTo   string
	RememberMe   bool
	Interim      bool
	CustomizeLogin bool
}

// Screen options for list tables
type ScreenOptions struct {
	PerPage       int
	Columns       []string
	ViewMode      string
}
