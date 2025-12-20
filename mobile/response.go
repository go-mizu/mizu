package mobile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
)

// Page represents a paginated response.
type Page[T any] struct {
	Items      []T    `json:"items"`
	Total      int    `json:"total,omitempty"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// NewPage creates a paginated response.
func NewPage[T any](items []T, page, perPage, total int) Page[T] {
	if items == nil {
		items = []T{}
	}
	hasMore := page*perPage < total
	return Page[T]{
		Items:   items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
		HasMore: hasMore,
	}
}

// NewCursorPage creates a cursor-based paginated response.
func NewCursorPage[T any](items []T, perPage int, nextCursor string) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{
		Items:      items,
		PerPage:    perPage,
		HasMore:    nextCursor != "",
		NextCursor: nextCursor,
	}
}

// Paginate parses pagination params from query string.
// Returns page number (1-indexed) and per page count.
func Paginate(c *mizu.Ctx) (page, perPage int) {
	page, _ = strconv.Atoi(c.Query("page"))
	perPage, _ = strconv.Atoi(c.Query("per_page"))

	// Also check common alternatives
	if perPage == 0 {
		perPage, _ = strconv.Atoi(c.Query("limit"))
	}
	if page == 0 {
		if offset, _ := strconv.Atoi(c.Query("offset")); offset > 0 && perPage > 0 {
			page = (offset / perPage) + 1
		}
	}

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return
}

// Cursor parses cursor from query string.
func Cursor(c *mizu.Ctx) string {
	if cursor := c.Query("cursor"); cursor != "" {
		return cursor
	}
	return c.Query("after")
}

// Error codes for mobile API responses.
const (
	ErrCodeInvalidRequest  = "invalid_request"
	ErrCodeUnauthorized    = "unauthorized"
	ErrCodeForbidden       = "forbidden"
	ErrCodeNotFound        = "not_found"
	ErrCodeConflict        = "conflict"
	ErrCodeRateLimited     = "rate_limited"
	ErrCodeServerError     = "server_error"
	ErrCodeMaintenance     = "maintenance"
	ErrCodeUpgradeRequired = "upgrade_required"
)

// Error represents a structured API error.
type Error struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	TraceID string            `json:"trace_id,omitempty"`
}

// Error implements the error interface.
func (e Error) Error() string { return e.Message }

// NewError creates an API error.
func NewError(code, message string) Error {
	return Error{Code: code, Message: message}
}

// WithDetails adds details to an error.
func (e Error) WithDetails(details map[string]string) Error {
	e.Details = details
	return e
}

// WithTraceID adds trace ID to an error.
func (e Error) WithTraceID(traceID string) Error {
	e.TraceID = traceID
	return e
}

// SendError sends an error response with the given status code.
func SendError(c *mizu.Ctx, status int, err Error) error {
	return c.JSON(status, map[string]Error{"error": err})
}

// Common error responses
var (
	ErrUnauthorized = Error{Code: ErrCodeUnauthorized, Message: "Authentication required"}
	ErrForbidden    = Error{Code: ErrCodeForbidden, Message: "Access denied"}
	ErrNotFound     = Error{Code: ErrCodeNotFound, Message: "Resource not found"}
	ErrServerError  = Error{Code: ErrCodeServerError, Message: "Internal server error"}
)

// ETag generates an ETag from data using SHA-256.
func ETag(data any) string {
	h := sha256.New()
	_ = json.NewEncoder(h).Encode(data)
	return fmt.Sprintf(`"%x"`, h.Sum(nil)[:8])
}

// WeakETag generates a weak ETag from data.
func WeakETag(data any) string {
	return "W/" + ETag(data)
}

// Conditional checks If-None-Match and returns 304 if matched.
// Returns true if 304 was sent (caller should return early).
func Conditional(c *mizu.Ctx, etag string) bool {
	c.Header().Set("ETag", etag)

	if match := c.Request().Header.Get("If-None-Match"); match != "" {
		// Handle comma-separated ETags
		for _, m := range strings.Split(match, ",") {
			m = strings.TrimSpace(m)
			if m == etag || m == "*" {
				c.Writer().WriteHeader(http.StatusNotModified)
				return true
			}
			// Handle weak ETag comparison
			if strings.HasPrefix(m, "W/") && strings.TrimPrefix(m, "W/") == strings.TrimPrefix(etag, "W/") {
				c.Writer().WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}
	return false
}

// CacheControl configures Cache-Control header for mobile responses.
type CacheControl struct {
	MaxAge         time.Duration
	Private        bool
	NoStore        bool
	NoCache        bool
	MustRevalidate bool
	Immutable      bool
}

// Set applies cache control header to response.
func (cc CacheControl) Set(c *mizu.Ctx) {
	c.Header().Set("Cache-Control", cc.String())
}

// String returns the Cache-Control header value.
func (cc CacheControl) String() string {
	var parts []string

	if cc.NoStore {
		return "no-store"
	}

	if cc.NoCache {
		parts = append(parts, "no-cache")
	}

	if cc.Private {
		parts = append(parts, "private")
	} else if !cc.NoCache {
		parts = append(parts, "public")
	}

	if cc.MaxAge > 0 {
		parts = append(parts, fmt.Sprintf("max-age=%d", int(cc.MaxAge.Seconds())))
	}

	if cc.MustRevalidate {
		parts = append(parts, "must-revalidate")
	}

	if cc.Immutable {
		parts = append(parts, "immutable")
	}

	if len(parts) == 0 {
		return "no-cache"
	}

	return strings.Join(parts, ", ")
}

// Common cache control configurations
var (
	// CachePrivate is suitable for user-specific data (5 minutes).
	CachePrivate = CacheControl{
		MaxAge:         5 * time.Minute,
		Private:        true,
		MustRevalidate: true,
	}

	// CacheShort is for frequently changing data (1 minute).
	CacheShort = CacheControl{
		MaxAge:         1 * time.Minute,
		MustRevalidate: true,
	}

	// CacheLong is for static content (1 day).
	CacheLong = CacheControl{
		MaxAge: 24 * time.Hour,
	}

	// CacheImmutable is for versioned/hashed assets (1 year).
	CacheImmutable = CacheControl{
		MaxAge:    365 * 24 * time.Hour,
		Immutable: true,
	}

	// NoCache disables caching.
	NoCache = CacheControl{NoCache: true}

	// NoStore prevents any caching (sensitive data).
	NoStore = CacheControl{NoStore: true}
)
