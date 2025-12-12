// Package graphql provides GraphQL query validation middleware for Mizu.
package graphql

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the GraphQL middleware.
type Options struct {
	// MaxDepth is the maximum query depth.
	// Default: 10.
	MaxDepth int

	// MaxComplexity is the maximum query complexity.
	// Default: 100.
	MaxComplexity int

	// DisableIntrospection disables introspection queries.
	// Default: false.
	DisableIntrospection bool

	// AllowedOperations are allowed operation names.
	// Empty allows all.
	AllowedOperations []string

	// BlockedFields are fields to block.
	BlockedFields []string

	// ErrorHandler handles validation errors.
	ErrorHandler func(c *mizu.Ctx, err error) error
}

// Query represents a GraphQL query.
type Query struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

// New creates GraphQL validation middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates GraphQL validation middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 10
	}
	if opts.MaxComplexity == 0 {
		opts.MaxComplexity = 100
	}

	allowedOps := make(map[string]bool)
	for _, op := range opts.AllowedOperations {
		allowedOps[op] = true
	}

	blockedFields := make(map[string]bool)
	for _, field := range opts.BlockedFields {
		blockedFields[field] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Only process POST requests to typical GraphQL endpoints
			if c.Request().Method != http.MethodPost {
				return next(c)
			}

			contentType := c.Request().Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				return next(c)
			}

			// Read and parse query
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return next(c)
			}

			var query Query
			if err := json.Unmarshal(body, &query); err != nil {
				return next(c)
			}

			// Validate query
			if err := validate(query, opts, allowedOps, blockedFields); err != nil {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c, err)
				}
				return c.JSON(http.StatusBadRequest, map[string]any{
					"errors": []map[string]string{
						{"message": err.Error()},
					},
				})
			}

			// Restore body
			c.Request().Body = io.NopCloser(strings.NewReader(string(body)))

			return next(c)
		}
	}
}

// Validation errors
type graphqlError string

func (e graphqlError) Error() string { return string(e) }

const (
	ErrQueryTooDeep           graphqlError = "query exceeds maximum depth"
	ErrQueryTooComplex        graphqlError = "query exceeds maximum complexity"
	ErrIntrospectionDisabled  graphqlError = "introspection queries are disabled"
	ErrOperationNotAllowed    graphqlError = "operation not allowed"
	ErrBlockedField           graphqlError = "query contains blocked field"
)

func validate(query Query, opts Options, allowedOps, blockedFields map[string]bool) error {
	q := query.Query

	// Check introspection
	if opts.DisableIntrospection && isIntrospection(q) {
		return ErrIntrospectionDisabled
	}

	// Check allowed operations
	if len(allowedOps) > 0 && query.OperationName != "" {
		if !allowedOps[query.OperationName] {
			return ErrOperationNotAllowed
		}
	}

	// Check depth
	depth := calculateDepth(q)
	if depth > opts.MaxDepth {
		return ErrQueryTooDeep
	}

	// Check complexity
	complexity := calculateComplexity(q)
	if complexity > opts.MaxComplexity {
		return ErrQueryTooComplex
	}

	// Check blocked fields
	for field := range blockedFields {
		if strings.Contains(q, field) {
			return ErrBlockedField
		}
	}

	return nil
}

func isIntrospection(query string) bool {
	introspectionPattern := regexp.MustCompile(`__schema|__type`)
	return introspectionPattern.MatchString(query)
}

func calculateDepth(query string) int {
	maxDepth := 0
	currentDepth := 0

	for _, c := range query {
		switch c {
		case '{':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case '}':
			currentDepth--
		}
	}

	return maxDepth
}

func calculateComplexity(query string) int {
	// Simple complexity calculation based on field count
	fieldPattern := regexp.MustCompile(`\w+\s*(?:\([^)]*\))?\s*{`)
	matches := fieldPattern.FindAllString(query, -1)
	return len(matches) + strings.Count(query, "{")
}

// MaxDepth creates middleware with a specific max depth.
func MaxDepth(depth int) mizu.Middleware {
	return WithOptions(Options{MaxDepth: depth})
}

// MaxComplexity creates middleware with a specific max complexity.
func MaxComplexity(complexity int) mizu.Middleware {
	return WithOptions(Options{MaxComplexity: complexity})
}

// NoIntrospection creates middleware that disables introspection.
func NoIntrospection() mizu.Middleware {
	return WithOptions(Options{DisableIntrospection: true})
}

// Production creates middleware with production-ready settings.
func Production() mizu.Middleware {
	return WithOptions(Options{
		MaxDepth:             10,
		MaxComplexity:        100,
		DisableIntrospection: true,
	})
}

// BlockFields creates middleware that blocks specific fields.
func BlockFields(fields ...string) mizu.Middleware {
	return WithOptions(Options{BlockedFields: fields})
}
