// Package config provides configuration types for the CMS.
package config

import (
	"net/http"
	"time"
)

// Config is the main CMS configuration.
type Config struct {
	ServerURL    string
	Secret       string
	Collections  []CollectionConfig
	Globals      []GlobalConfig
	DB           *DBConfig
	Upload       *UploadConfig
	Email        *EmailConfig
	Localization *LocalizationConfig
	Admin        *AdminConfig
	CORS         *CORSConfig
	RateLimit    *RateLimitConfig
}

// DBConfig holds database configuration.
type DBConfig struct {
	Path string
}

// UploadConfig holds upload configuration.
type UploadConfig struct {
	StaticDir  string
	StaticURL  string
	MaxSize    int64
	MimeTypes  []string
	ImageSizes []ImageSize
	FocalPoint bool
}

// ImageSize defines an image size variant.
type ImageSize struct {
	Name   string
	Width  int
	Height int
	Crop   bool
}

// EmailConfig holds email configuration.
type EmailConfig struct {
	From     string
	Host     string
	Port     int
	Username string
	Password string
}

// LocalizationConfig holds localization configuration.
type LocalizationConfig struct {
	Locales        []Locale
	DefaultLocale  string
	FallbackLocale string
}

// Locale represents a single locale.
type Locale struct {
	Code  string
	Label string
	RTL   bool
}

// AdminConfig holds admin panel configuration.
type AdminConfig struct {
	User       string // Collection slug for admin users
	Meta       *MetaConfig
	Components *AdminComponents
}

// MetaConfig holds HTML meta configuration.
type MetaConfig struct {
	Title       string
	Description string
	OGImage     string
}

// AdminComponents holds custom admin components.
type AdminComponents struct {
	Logo      string
	Dashboard string
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	Origins []string
	Headers []string
	Methods []string
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Window   time.Duration
	Max      int
	TrustProxy bool
}

// CollectionConfig defines a collection.
type CollectionConfig struct {
	Slug       string
	Labels     Labels
	Fields     []Field
	Admin      *CollectionAdmin
	Auth       *AuthConfig
	Upload     *CollectionUploadConfig
	Versions   *VersionsConfig
	Timestamps bool
	Access     *AccessConfig
	Hooks      *CollectionHooks
	DBName     string
	Indexes    []IndexConfig
	Endpoints  []EndpointConfig
}

// Labels provides singular and plural labels.
type Labels struct {
	Singular string
	Plural   string
}

// CollectionAdmin holds collection-specific admin config.
type CollectionAdmin struct {
	Group           string
	Hidden          bool
	UseAsTitle      string
	DefaultColumns  []string
	ListSearchableFields []string
	Pagination      *PaginationConfig
	Description     string
}

// PaginationConfig holds pagination settings.
type PaginationConfig struct {
	DefaultLimit int
	Limits       []int
}

// AuthConfig enables authentication on a collection.
type AuthConfig struct {
	TokenExpiration  int // seconds
	MaxLoginAttempts int
	LockTime         int // seconds
	UseAPIKey        bool
	Verify           bool
	ForgotPassword   bool
	Cookies          *CookieConfig
}

// CookieConfig holds cookie settings.
type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	Domain   string
}

// CollectionUploadConfig enables file uploads.
type CollectionUploadConfig struct {
	StaticDir  string
	StaticURL  string
	MimeTypes  []string
	MaxSize    int64
	ImageSizes []ImageSize
	FocalPoint bool
	BulkUpload bool
}

// VersionsConfig enables versioning.
type VersionsConfig struct {
	Drafts    bool
	MaxPerDoc int
}

// AccessConfig defines access control.
type AccessConfig struct {
	Create AccessFn
	Read   AccessFn
	Update AccessFn
	Delete AccessFn
	Admin  AccessFn
}

// AccessContext provides context for access control functions.
type AccessContext struct {
	Req        *http.Request
	User       map[string]any
	ID         string
	Data       map[string]any
	Collection string
}

// AccessResult can be a boolean or a query filter.
type AccessResult struct {
	Allowed bool
	Where   map[string]any
}

// AccessFn is an access control function.
type AccessFn func(ctx *AccessContext) (*AccessResult, error)

// CollectionHooks defines collection lifecycle hooks.
type CollectionHooks struct {
	BeforeOperation []BeforeOperationHook
	BeforeValidate  []BeforeChangeHook
	BeforeChange    []BeforeChangeHook
	AfterChange     []AfterChangeHook
	BeforeRead      []BeforeReadHook
	AfterRead       []AfterReadHook
	BeforeDelete    []BeforeDeleteHook
	AfterDelete     []AfterDeleteHook
	AfterOperation  []AfterOperationHook
	AfterError      []AfterErrorHook

	// Auth hooks
	BeforeLogin         []BeforeLoginHook
	AfterLogin          []AfterLoginHook
	AfterLogout         []AfterLogoutHook
	AfterMe             []AfterMeHook
	AfterRefresh        []AfterRefreshHook
	AfterForgotPassword []AfterForgotPasswordHook
}

// Hook function types
type (
	BeforeOperationHook   func(ctx *HookContext) error
	BeforeChangeHook      func(ctx *HookContext) error
	AfterChangeHook       func(ctx *HookContext) error
	BeforeReadHook        func(ctx *HookContext) error
	AfterReadHook         func(ctx *HookContext) error
	BeforeDeleteHook      func(ctx *HookContext) error
	AfterDeleteHook       func(ctx *HookContext) error
	AfterOperationHook    func(ctx *HookContext) error
	AfterErrorHook        func(ctx *HookContext, err error) error
	BeforeLoginHook       func(ctx *HookContext) error
	AfterLoginHook        func(ctx *HookContext) error
	AfterLogoutHook       func(ctx *HookContext) error
	AfterMeHook           func(ctx *HookContext) error
	AfterRefreshHook      func(ctx *HookContext) error
	AfterForgotPasswordHook func(ctx *HookContext) error
)

// HookContext provides context for hooks.
type HookContext struct {
	Req        *http.Request
	Collection string
	Operation  string
	ID         string
	Data       map[string]any
	OriginalDoc map[string]any
	User       map[string]any
	FindArgs   *FindArgs
}

// FindArgs holds find operation arguments.
type FindArgs struct {
	Where      map[string]any
	Sort       string
	Limit      int
	Page       int
	Depth      int
	Locale     string
	FallbackLocale string
}

// IndexConfig defines a database index.
type IndexConfig struct {
	Fields []string
	Unique bool
}

// EndpointConfig defines a custom endpoint.
type EndpointConfig struct {
	Path    string
	Method  string
	Handler http.HandlerFunc
}

// GlobalConfig defines a global.
type GlobalConfig struct {
	Slug      string
	Label     string
	Fields    []Field
	Admin     *GlobalAdmin
	Access    *AccessConfig
	Hooks     *GlobalHooks
	Versions  *VersionsConfig
	Endpoints []EndpointConfig
}

// GlobalAdmin holds global-specific admin config.
type GlobalAdmin struct {
	Group       string
	Hidden      bool
	Description string
}

// GlobalHooks defines global lifecycle hooks.
type GlobalHooks struct {
	BeforeChange []BeforeChangeHook
	AfterChange  []AfterChangeHook
	BeforeRead   []BeforeReadHook
	AfterRead    []AfterReadHook
}
