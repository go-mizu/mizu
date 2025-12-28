package api

import (
	"context"
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// contextKey is a type for context keys
type contextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey contextKey = "user"
)

// GetUser returns the authenticated user from context
func GetUser(ctx context.Context) *users.User {
	u, _ := ctx.Value(UserContextKey).(*users.User)
	return u
}

// GetUserID returns the authenticated user ID from context
func GetUserID(ctx context.Context) int64 {
	u := GetUser(ctx)
	if u == nil {
		return 0
	}
	return u.ID
}

// GetUserFromCtx returns the authenticated user from Mizu context
func GetUserFromCtx(c *mizu.Ctx) *users.User {
	return GetUser(c.Context())
}

// GetUserIDFromCtx returns the authenticated user ID from Mizu context
func GetUserIDFromCtx(c *mizu.Ctx) int64 {
	return GetUserID(c.Context())
}

// PaginationParams extracts pagination parameters from request
type PaginationParams struct {
	Page    int
	PerPage int
}

// GetPagination extracts pagination params from Mizu context
func GetPagination(c *mizu.Ctx) PaginationParams {
	p := PaginationParams{
		Page:    1,
		PerPage: 30,
	}

	if page := c.Query("page"); page != "" {
		if n, err := strconv.Atoi(page); err == nil && n > 0 {
			p.Page = n
		}
	}

	if perPage := c.Query("per_page"); perPage != "" {
		if n, err := strconv.Atoi(perPage); err == nil && n > 0 && n <= 100 {
			p.PerPage = n
		}
	}

	return p
}

// ParamInt extracts an integer path parameter
func ParamInt(c *mizu.Ctx, name string) (int, error) {
	return strconv.Atoi(c.Param(name))
}

// ParamInt64 extracts an int64 path parameter
func ParamInt64(c *mizu.Ctx, name string) (int64, error) {
	return strconv.ParseInt(c.Param(name), 10, 64)
}

// QueryInt extracts an integer query parameter with default
func QueryInt(c *mizu.Ctx, name string, defaultVal int) int {
	s := c.Query(name)
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

// QueryBool extracts a boolean query parameter
func QueryBool(c *mizu.Ctx, name string) bool {
	s := c.Query(name)
	return s == "true" || s == "1"
}
