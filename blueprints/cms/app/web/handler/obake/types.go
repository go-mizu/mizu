// Package obake provides Ghost CMS-compatible admin interface handlers.
package obake

import (
	"time"

	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
)

// NavItem represents a navigation item in the Ghost admin sidebar.
type NavItem struct {
	ID       string
	Title    string
	URL      string
	Icon     string // SVG icon name
	Active   bool
	Badge    int    // Notification count
	External bool   // Opens in new tab
	Children []NavItem
}

// Breadcrumb represents navigation breadcrumb.
type Breadcrumb struct {
	Label string
	URL   string
}

// Toast represents a notification toast message.
type Toast struct {
	Type    string // "success", "error", "warning", "info"
	Message string
}

// BaseData contains common data for all admin pages.
type BaseData struct {
	Title       string
	SiteTitle   string
	SiteURL     string
	User        *users.User
	Nav         []NavItem
	ActiveNav   string
	Breadcrumbs []Breadcrumb
	Toast       *Toast
}

// DashboardMetric represents a single metric on the dashboard.
type DashboardMetric struct {
	Label      string
	Value      string
	Trend      string  // "+12%", "-5%", etc.
	TrendUp    bool    // true = positive trend
	Sparkline  []int   // Data points for mini chart
}

// TopContent represents a top performing content item.
type TopContent struct {
	ID        string
	Title     string
	Views     int
	Members   int
	Revenue   string
	Thumbnail string
	URL       string
	Type      string // "post", "page"
}

// TopSource represents a traffic source.
type TopSource struct {
	Name    string
	URL     string
	Visits  int
	Percent float64
}

// DashboardData holds data for the Ghost admin dashboard.
type DashboardData struct {
	BaseData

	// Overview tab
	ActiveTab       string // "overview", "newsletters", "growth", "locations"
	DateRange       string // "today", "7d", "30d", "90d", "all"
	AudienceFilter  string // "all", "free", "paid"

	// Metrics
	UniqueVisitors  DashboardMetric
	TotalPageviews  DashboardMetric
	RealtimeVisitors int
	TopContent      []TopContent
	TopSources      []TopSource

	// Newsletters tab
	TotalSubscribers     DashboardMetric
	AverageOpenRate      DashboardMetric
	AverageClickRate     DashboardMetric
	TopNewsletters       []NewsletterStats

	// Growth tab
	TotalMembers   DashboardMetric
	FreeMembers    DashboardMetric
	PaidMembers    DashboardMetric
	MRR            DashboardMetric

	// Locations tab
	TopCountries   []CountryStats
}

// NewsletterStats represents newsletter performance data.
type NewsletterStats struct {
	ID          string
	Title       string
	SentDate    time.Time
	Recipients  int
	OpenRate    float64
	ClickRate   float64
}

// CountryStats represents traffic by country.
type CountryStats struct {
	Country string
	Code    string
	Visits  int
	Percent float64
}

// PostRow represents a post in the list view.
type PostRow struct {
	*posts.Post
	Author       *users.User
	FeaturedTag  *tags.Tag
	Tags         []*tags.Tag
	AccessLevel  string // "public", "members", "paid"
	Thumbnail    string
	Featured     bool
	Status       string // "published", "draft", "scheduled"
	PublishDate  time.Time
	CommentCount int
}

// PostsListData holds data for the posts list page.
type PostsListData struct {
	BaseData

	Posts        []*PostRow
	TotalPosts   int
	FilterStatus string // "all", "published", "draft", "scheduled", "featured"
	FilterAuthor string
	FilterTag    string
	FilterAccess string
	SortBy       string
	SortOrder    string
	SearchQuery  string
	ViewMode     string // "list", "grid"

	// For filters
	Authors      []*users.User
	AllTags      []*tags.Tag
}

// PostCard represents an editor card/block.
type PostCard struct {
	Type    string // "image", "markdown", "html", "gallery", etc.
	Content string
	Data    map[string]interface{}
}

// PostEditData holds data for the post editor.
type PostEditData struct {
	BaseData

	Post          *posts.Post
	IsNew         bool
	PostType      string // "post" or "page"

	// Sidebar settings
	Slug          string
	PublishDate   string
	PublishTime   string
	Authors       []*users.User
	SelectedAuthors []string
	AllAuthors    []*users.User
	Tags          []*tags.Tag
	SelectedTags  []string
	AllTags       []*tags.Tag
	FeaturedImage *media.Media
	Excerpt       string
	AccessLevel   string
	Featured      bool

	// SEO
	MetaTitle       string
	MetaDescription string
	CanonicalURL    string

	// Social
	TwitterTitle    string
	TwitterDescription string
	TwitterImage    string
	FacebookTitle   string
	FacebookDescription string
	FacebookImage   string

	// Code injection
	CodeHead        string
	CodeFoot        string

	// Template
	Template        string
	Templates       []SelectOption
}

// SelectOption represents a select dropdown option.
type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

// PageRow represents a page in the list view.
type PageRow struct {
	*pages.Page
	Author      *users.User
	Thumbnail   string
	Status      string
	PublishDate time.Time
}

// PagesListData holds data for pages list.
type PagesListData struct {
	BaseData

	Pages        []*PageRow
	TotalPages   int
	FilterStatus string
	SortBy       string
	SearchQuery  string
	ViewMode     string
}

// PageEditData holds data for page editor.
type PageEditData struct {
	BaseData

	Page           *pages.Page
	IsNew          bool
	ParentPages    []*pages.Page
	SelectedParent string
	Slug           string
	PublishDate    string
	PublishTime    string
	Authors        []*users.User
	SelectedAuthors []string
	FeaturedImage  *media.Media
	Excerpt        string
	AccessLevel    string
	MetaTitle      string
	MetaDescription string
	Template       string
	Templates      []SelectOption
}

// TagRow represents a tag in the list view.
type TagRow struct {
	*tags.Tag
	PostCount   int
	IsInternal  bool // Tags starting with #
	Featured    *media.Media
}

// TagsListData holds data for tags list.
type TagsListData struct {
	BaseData

	Tags         []*TagRow
	ShowInternal bool
	SearchQuery  string
}

// TagEditData holds data for tag editor.
type TagEditData struct {
	BaseData

	Tag             *tags.Tag
	IsNew           bool
	IsInternal      bool
	Slug            string
	Description     string
	FeaturedImage   *media.Media
	MetaTitle       string
	MetaDescription string
	CanonicalURL    string
	TwitterTitle    string
	TwitterDescription string
	TwitterImage    string
	FacebookTitle   string
	FacebookDescription string
	FacebookImage   string
	CodeHead        string
	CodeFoot        string
	PostCount       int
}

// MemberRow represents a member in the list view.
type MemberRow struct {
	ID           string
	Email        string
	Name         string
	Avatar       string
	Status       string // "free", "paid", "comped"
	Tier         string
	Subscribed   time.Time
	LastSeen     time.Time
	OpenRate     float64
	Labels       []string
}

// MembersListData holds data for members list.
type MembersListData struct {
	BaseData

	Members       []*MemberRow
	TotalMembers  int
	FreeCount     int
	PaidCount     int
	FilterStatus  string
	FilterLabel   string
	SortBy        string
	SearchQuery   string
	Labels        []string
}

// MemberActivity represents a member activity log entry.
type MemberActivity struct {
	Type      string // "signup", "login", "subscription", "email_open"
	Details   string
	Timestamp time.Time
}

// MemberDetailData holds data for member detail view.
type MemberDetailData struct {
	BaseData

	Member       *MemberRow
	Notes        string
	Subscriptions []MemberSubscription
	Activity     []MemberActivity
	EmailActivity []EmailActivity
}

// MemberSubscription represents a member's subscription.
type MemberSubscription struct {
	Tier      string
	Status    string
	StartDate time.Time
	Amount    string
	Interval  string
}

// EmailActivity represents email engagement.
type EmailActivity struct {
	Subject   string
	SentDate  time.Time
	Opened    bool
	Clicked   bool
}

// StaffRow represents a staff member in the list.
type StaffRow struct {
	*users.User
	Role         string
	RoleLabel    string
	PostCount    int
	LastSeen     time.Time
	Avatar       string
}

// StaffListData holds data for staff list.
type StaffListData struct {
	BaseData

	Staff       []*StaffRow
	InviteURL   string
}

// StaffEditData holds data for staff edit.
type StaffEditData struct {
	BaseData

	Staff          *users.User
	IsNew          bool
	IsSelf         bool
	IsOwner        bool

	// Profile
	Name           string
	Slug           string
	Email          string
	Location       string
	Website        string
	Bio            string
	ProfileImage   *media.Media
	CoverImage     *media.Media
	FacebookURL    string
	TwitterURL     string

	// Role
	Role           string
	Roles          []SelectOption

	// Password
	ShowPassword   bool
}

// SettingsGeneralData holds data for general settings.
type SettingsGeneralData struct {
	BaseData

	// Title & description
	SiteTitle       string
	SiteDescription string
	SiteTimezone    string
	Timezones       []SelectOption
	SiteLanguage    string
	Languages       []SelectOption

	// Meta data
	MetaTitle       string
	MetaDescription string

	// Social
	FacebookURL     string
	TwitterURL      string
	TwitterTitle    string
	TwitterDescription string
	TwitterImage    string

	// Privacy
	IsPrivate       bool
	PrivatePassword string
}

// SettingsDesignData holds data for design settings.
type SettingsDesignData struct {
	BaseData

	// Branding
	Icon            *media.Media
	Logo            *media.Media
	CoverImage      *media.Media
	AccentColor     string

	// Navigation
	PrimaryNav      []NavLink
	SecondaryNav    []NavLink

	// Announcement
	AnnouncementEnabled bool
	AnnouncementContent string
	AnnouncementBg      string
}

// NavLink represents a navigation link.
type NavLink struct {
	Label string
	URL   string
}

// SettingsMembershipData holds data for membership settings.
type SettingsMembershipData struct {
	BaseData

	// Access
	DefaultAccess    string // "public", "members", "paid"
	CommentAccess    string

	// Portal
	PortalEnabled    bool
	PortalButton     bool
	PortalSignup     bool
	PortalAccountLink bool

	// Tiers
	Tiers            []Tier
}

// Tier represents a membership tier.
type Tier struct {
	ID          string
	Name        string
	Description string
	MonthlyPrice string
	YearlyPrice  string
	Benefits    []string
	Active      bool
}

// SettingsEmailData holds data for email settings.
type SettingsEmailData struct {
	BaseData

	// Newsletter
	NewsletterEnabled bool
	DefaultRecipients string
	SenderName        string
	SenderEmail       string
	ReplyTo           string

	// Mailgun
	MailgunDomain     string
	MailgunAPIKey     string
	MailgunBaseURL    string
}

// SettingsAdvancedData holds data for advanced settings.
type SettingsAdvancedData struct {
	BaseData

	// Code injection
	CodeHead          string
	CodeFoot          string

	// Integrations
	Integrations      []Integration

	// Labs
	LabsFeatures      []LabsFeature

	// Data
	ExportURL         string
}

// Integration represents a custom integration.
type Integration struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Slug        string
	APIKey      string
	WebhooksURL string
}

// LabsFeature represents a labs/beta feature.
type LabsFeature struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	Flag        string
}

// LoginData holds data for the login page.
type LoginData struct {
	SiteTitle    string
	SiteURL      string
	SiteLogo     string
	Error        string
	Email        string
	RedirectTo   string
}

// SearchResultItem represents a search result.
type SearchResultItem struct {
	Type      string // "post", "page", "tag", "member", "staff"
	Title     string
	Subtitle  string
	URL       string
	Icon      string
	Thumbnail string
}

// SearchData holds data for search results.
type SearchData struct {
	Query   string
	Results []SearchResultItem
	Recent  []SearchResultItem
}

// MediaLibraryData holds data for media library.
type MediaLibraryData struct {
	BaseData

	Items       []*media.Media
	TotalItems  int
	FilterType  string
	SearchQuery string
	Page        int
	PerPage     int
}

// CommentRow represents a comment in moderation view.
type CommentRow struct {
	*comments.Comment
	Author      string
	AuthorEmail string
	AuthorAvatar string
	PostTitle   string
	PostURL     string
	Status      string
}

// CommentsListData holds data for comments list.
type CommentsListData struct {
	BaseData

	Comments     []*CommentRow
	TotalCount   int
	FilterStatus string
	SearchQuery  string
}

// CategoryRow represents a category in the list.
type CategoryRow struct {
	*categories.Category
	PostCount int
	Children  []*CategoryRow
	Depth     int
}

// CategoriesListData holds data for categories list.
type CategoriesListData struct {
	BaseData

	Categories  []*CategoryRow
	SearchQuery string
}
