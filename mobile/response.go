package mobile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Standard error codes.
const (
	ErrInvalidRequest  = "invalid_request"
	ErrUnauthorized    = "unauthorized"
	ErrForbidden       = "forbidden"
	ErrNotFound        = "not_found"
	ErrConflict        = "conflict"
	ErrRateLimited     = "rate_limited"
	ErrValidation      = "validation_error"
	ErrInternal        = "internal_error"
	ErrServiceDown     = "service_unavailable"
	ErrUpgradeRequired = "upgrade_required"
	ErrMaintenance     = "maintenance"
)

// Error is a structured API error for mobile clients.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	TraceID string         `json:"trace_id,omitempty"`
	DocURL  string         `json:"doc_url,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

// NewError creates a new Error.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// WithDetails adds a detail to the error.
func (e *Error) WithDetails(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithTraceID adds a trace ID to the error.
func (e *Error) WithTraceID(id string) *Error {
	e.TraceID = id
	return e
}

// WithDocURL adds a documentation URL to the error.
func (e *Error) WithDocURL(url string) *Error {
	e.DocURL = url
	return e
}

// SendError sends a structured error response.
func SendError(c *mizu.Ctx, statusCode int, err *Error) error {
	// Try to get trace ID from request
	if err.TraceID == "" {
		if traceID := c.Request().Header.Get(HeaderRequestID); traceID != "" {
			err.TraceID = traceID
		}
	}
	return c.JSON(statusCode, err)
}

// Page represents a paginated response.
type Page[T any] struct {
	Data       []T    `json:"data"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"per_page,omitempty"`
	Total      int    `json:"total,omitempty"`
	TotalPages int    `json:"total_pages,omitempty"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

// PageRequest contains pagination parameters from request.
type PageRequest struct {
	Page    int    // ?page=1 (1-indexed)
	PerPage int    // ?per_page=20
	Cursor  string // ?cursor=xxx (for cursor pagination)
	After   string // ?after=xxx (alias for cursor)
	Before  string // ?before=xxx (reverse cursor)
}

// Offset returns the SQL offset for page-based pagination.
func (p PageRequest) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

// Limit returns the limit (same as PerPage).
func (p PageRequest) Limit() int {
	return p.PerPage
}

// IsCursor returns true if cursor-based pagination is requested.
func (p PageRequest) IsCursor() bool {
	return p.Cursor != "" || p.After != "" || p.Before != ""
}

// CursorValue returns the effective cursor value.
func (p PageRequest) CursorValue() string {
	if p.Cursor != "" {
		return p.Cursor
	}
	if p.After != "" {
		return p.After
	}
	return p.Before
}

// IsReverse returns true if reverse pagination (before) is requested.
func (p PageRequest) IsReverse() bool {
	return p.Before != ""
}

// ParsePageRequest extracts pagination from request.
func ParsePageRequest(c *mizu.Ctx) PageRequest {
	return ParsePageRequestWithDefaults(c, 1, 20)
}

// ParsePageRequestWithDefaults extracts pagination with custom defaults.
func ParsePageRequestWithDefaults(c *mizu.Ctx, defaultPage, defaultPerPage int) PageRequest {
	p := PageRequest{
		Page:    defaultPage,
		PerPage: defaultPerPage,
		Cursor:  c.Query("cursor"),
		After:   c.Query("after"),
		Before:  c.Query("before"),
	}

	if page := c.Query("page"); page != "" {
		if n, err := strconv.Atoi(page); err == nil && n > 0 {
			p.Page = n
		}
	}

	if perPage := c.Query("per_page"); perPage != "" {
		if n, err := strconv.Atoi(perPage); err == nil && n > 0 {
			p.PerPage = n
		}
	}

	// Also check "limit" as alternative to "per_page"
	if limit := c.Query("limit"); limit != "" {
		if n, err := strconv.Atoi(limit); err == nil && n > 0 {
			p.PerPage = n
		}
	}

	// Cap per page to reasonable maximum
	if p.PerPage > 100 {
		p.PerPage = 100
	}

	return p
}

// NewPage creates a page response for offset-based pagination.
func NewPage[T any](data []T, req PageRequest, total int) Page[T] {
	if data == nil {
		data = []T{}
	}

	totalPages := 0
	if req.PerPage > 0 {
		totalPages = (total + req.PerPage - 1) / req.PerPage
	}

	return Page[T]{
		Data:       data,
		Page:       req.Page,
		PerPage:    req.PerPage,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    req.Page < totalPages,
	}
}

// NewCursorPage creates a page response for cursor-based pagination.
func NewCursorPage[T any](data []T, nextCursor, prevCursor string, hasMore bool) Page[T] {
	if data == nil {
		data = []T{}
	}
	return Page[T]{
		Data:       data,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
	}
}

// ETag generates an ETag from data.
// Uses SHA-256 truncated to 16 chars for brevity.
func ETag(data any) string {
	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(b)
	return `"` + hex.EncodeToString(hash[:8]) + `"`
}

// WeakETag generates a weak ETag (W/"...").
func WeakETag(data any) string {
	etag := ETag(data)
	if etag == "" {
		return ""
	}
	return "W/" + etag
}

// CheckETag checks If-None-Match and returns true if matched.
// If matched, sends 304 Not Modified response.
func CheckETag(c *mizu.Ctx, etag string) bool {
	if etag == "" {
		return false
	}

	ifNoneMatch := c.Request().Header.Get("If-None-Match")
	if ifNoneMatch == "" {
		return false
	}

	// Check for match (handles multiple ETags and weak comparison)
	for _, e := range strings.Split(ifNoneMatch, ",") {
		e = strings.TrimSpace(e)
		// Remove weak indicator for comparison
		e = strings.TrimPrefix(e, "W/")
		checkEtag := strings.TrimPrefix(etag, "W/")
		if e == "*" || e == checkEtag {
			c.Writer().WriteHeader(http.StatusNotModified)
			return true
		}
	}

	return false
}

// Conditional sends 304 if ETag matches, otherwise sends data with ETag.
func Conditional(c *mizu.Ctx, data any) error {
	etag := ETag(data)
	if CheckETag(c, etag) {
		return nil
	}
	c.Header().Set("ETag", etag)
	return c.JSON(http.StatusOK, data)
}

// CacheControl configures response caching.
type CacheControl struct {
	MaxAge         time.Duration
	Private        bool
	Public         bool
	NoCache        bool
	NoStore        bool
	MustRevalidate bool
	Immutable      bool
	NoTransform    bool
}

// Cache presets.
var (
	CacheNone      = CacheControl{NoStore: true, NoCache: true}
	CachePrivate   = CacheControl{Private: true, NoCache: true}
	CacheShort     = CacheControl{Private: true, MaxAge: 5 * time.Minute}
	CacheMedium    = CacheControl{Private: true, MaxAge: 1 * time.Hour}
	CacheLong      = CacheControl{Private: true, MaxAge: 24 * time.Hour}
	CacheImmutable = CacheControl{Public: true, MaxAge: 365 * 24 * time.Hour, Immutable: true}
)

// Apply sets Cache-Control header on response.
func (cc CacheControl) Apply(c *mizu.Ctx) {
	c.Header().Set("Cache-Control", cc.String())
}

// String returns the Cache-Control header value.
func (cc CacheControl) String() string {
	var parts []string

	if cc.NoStore {
		parts = append(parts, "no-store")
	}
	if cc.NoCache {
		parts = append(parts, "no-cache")
	}
	if cc.Private {
		parts = append(parts, "private")
	}
	if cc.Public {
		parts = append(parts, "public")
	}
	if cc.MaxAge > 0 {
		parts = append(parts, "max-age="+strconv.Itoa(int(cc.MaxAge.Seconds())))
	}
	if cc.MustRevalidate {
		parts = append(parts, "must-revalidate")
	}
	if cc.Immutable {
		parts = append(parts, "immutable")
	}
	if cc.NoTransform {
		parts = append(parts, "no-transform")
	}

	if len(parts) == 0 {
		return "no-cache"
	}
	return strings.Join(parts, ", ")
}
