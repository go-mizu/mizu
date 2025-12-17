// Package http provides HTTP transport for the sync engine.
//
// This package handles HTTP request/response serialization, error code mapping,
// and request validation. It separates transport concerns from the core sync logic.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/sync"
)

// Error codes for HTTP responses.
const (
	CodeNotFound     = "not_found"
	CodeInvalid      = "invalid_mutation"
	CodeCursorTooOld = "cursor_too_old"
	CodeConflict     = "conflict"
	CodeInternal     = "internal_error"
)

// Transport wraps a sync.Engine with HTTP handlers.
type Transport struct {
	engine       *sync.Engine
	scopeFunc    func(context.Context, string) (string, error)
	maxPullLimit int
	maxPushBatch int
}

// Options configures the HTTP transport.
type Options struct {
	// Engine is the sync engine to wrap (required).
	Engine *sync.Engine

	// ScopeFunc derives the authoritative scope from request context and claimed scope.
	// If nil, the claimed scope from the request is used directly.
	// Use this to enforce authorization (e.g., derive scope from JWT claims).
	ScopeFunc func(ctx context.Context, claimed string) (string, error)

	// MaxPullLimit caps the limit parameter for Pull requests.
	// Defaults to 1000 if zero.
	MaxPullLimit int

	// MaxPushBatch caps the number of mutations in a Push request.
	// Defaults to 100 if zero.
	MaxPushBatch int
}

// New creates a new HTTP transport.
func New(opts Options) *Transport {
	maxPull := opts.MaxPullLimit
	if maxPull <= 0 {
		maxPull = 1000
	}
	maxPush := opts.MaxPushBatch
	if maxPush <= 0 {
		maxPush = 100
	}
	return &Transport{
		engine:       opts.Engine,
		scopeFunc:    opts.ScopeFunc,
		maxPullLimit: maxPull,
		maxPushBatch: maxPush,
	}
}

// Mount registers sync routes on a Mizu app at /_sync/*.
func (t *Transport) Mount(app *mizu.App) {
	t.MountAt(app, "/_sync")
}

// MountAt registers sync routes at a custom prefix.
func (t *Transport) MountAt(app *mizu.App, prefix string) {
	app.Post(prefix+"/push", t.handlePush())
	app.Post(prefix+"/pull", t.handlePull())
	app.Post(prefix+"/snapshot", t.handleSnapshot())
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// PushRequest is the wire format for push requests.
type PushRequest struct {
	Mutations []sync.Mutation `json:"mutations"`
}

// PushResponse is the wire format for push responses.
type PushResponse struct {
	Results []sync.Result `json:"results"`
}

// PullRequest is the wire format for pull requests.
type PullRequest struct {
	Scope  string `json:"scope,omitempty"`
	Cursor uint64 `json:"cursor"`
	Limit  int    `json:"limit,omitempty"`
}

// PullResponse is the wire format for pull responses.
type PullResponse struct {
	Changes    []sync.Change `json:"changes"`
	HasMore    bool          `json:"has_more"`
	NextCursor uint64        `json:"next_cursor,omitempty"`
}

// SnapshotRequest is the wire format for snapshot requests.
type SnapshotRequest struct {
	Scope string `json:"scope,omitempty"`
}

// SnapshotResponse is the wire format for snapshot responses.
type SnapshotResponse struct {
	Data   json.RawMessage `json:"data"`
	Cursor uint64          `json:"cursor"`
}

// ErrorResponse is the wire format for error responses.
type ErrorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

// -----------------------------------------------------------------------------
// Handlers
// -----------------------------------------------------------------------------

func (t *Transport) handlePush() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req PushRequest
		if err := c.BindJSON(&req, 1<<20); err != nil { // 1MB max
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:  CodeInvalid,
				Error: "invalid request body",
			})
		}

		if len(req.Mutations) == 0 {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:  CodeInvalid,
				Error: "no mutations provided",
			})
		}

		if len(req.Mutations) > t.maxPushBatch {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:  CodeInvalid,
				Error: "too many mutations in batch",
			})
		}

		// Apply ScopeFunc if configured
		if t.scopeFunc != nil {
			for i := range req.Mutations {
				scope := req.Mutations[i].Scope
				if scope == "" {
					scope = sync.DefaultScope
				}
				authScope, err := t.scopeFunc(c.Context(), scope)
				if err != nil {
					return c.JSON(http.StatusForbidden, ErrorResponse{
						Code:  CodeInternal,
						Error: err.Error(),
					})
				}
				req.Mutations[i].Scope = authScope
			}
		}

		results, err := t.engine.Push(c.Context(), req.Mutations)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:  CodeInternal,
				Error: err.Error(),
			})
		}

		return c.JSON(http.StatusOK, PushResponse{Results: results})
	}
}

func (t *Transport) handlePull() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req PullRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:  CodeInvalid,
				Error: "invalid request body",
			})
		}

		// Apply ScopeFunc if configured
		scope := req.Scope
		if t.scopeFunc != nil {
			if scope == "" {
				scope = sync.DefaultScope
			}
			authScope, err := t.scopeFunc(c.Context(), scope)
			if err != nil {
				return c.JSON(http.StatusForbidden, ErrorResponse{
					Code:  CodeInternal,
					Error: err.Error(),
				})
			}
			scope = authScope
		}

		// Cap limit
		limit := req.Limit
		if limit <= 0 {
			limit = 100
		}
		if limit > t.maxPullLimit {
			limit = t.maxPullLimit
		}

		changes, hasMore, err := t.engine.Pull(c.Context(), scope, req.Cursor, limit)
		if err != nil {
			code := http.StatusInternalServerError
			errCode := CodeInternal
			if errors.Is(err, sync.ErrCursorTooOld) {
				code = http.StatusGone
				errCode = CodeCursorTooOld
			}
			return c.JSON(code, ErrorResponse{
				Code:  errCode,
				Error: err.Error(),
			})
		}

		// Compute next cursor
		var nextCursor uint64
		if len(changes) > 0 {
			nextCursor = changes[len(changes)-1].Cursor
		}

		return c.JSON(http.StatusOK, PullResponse{
			Changes:    changes,
			HasMore:    hasMore,
			NextCursor: nextCursor,
		})
	}
}

func (t *Transport) handleSnapshot() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req SnapshotRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:  CodeInvalid,
				Error: "invalid request body",
			})
		}

		// Apply ScopeFunc if configured
		scope := req.Scope
		if t.scopeFunc != nil {
			if scope == "" {
				scope = sync.DefaultScope
			}
			authScope, err := t.scopeFunc(c.Context(), scope)
			if err != nil {
				return c.JSON(http.StatusForbidden, ErrorResponse{
					Code:  CodeInternal,
					Error: err.Error(),
				})
			}
			scope = authScope
		}

		data, cursor, err := t.engine.Snapshot(c.Context(), scope)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Code:  CodeInternal,
				Error: err.Error(),
			})
		}

		return c.JSON(http.StatusOK, SnapshotResponse{
			Data:   data,
			Cursor: cursor,
		})
	}
}

// MapError maps sync errors to HTTP status codes and error codes.
// This is useful for custom handlers that need error mapping.
func MapError(err error) (int, string) {
	switch {
	case errors.Is(err, sync.ErrNotFound):
		return http.StatusNotFound, CodeNotFound
	case errors.Is(err, sync.ErrInvalidMutation):
		return http.StatusBadRequest, CodeInvalid
	case errors.Is(err, sync.ErrConflict):
		return http.StatusConflict, CodeConflict
	case errors.Is(err, sync.ErrCursorTooOld):
		return http.StatusGone, CodeCursorTooOld
	default:
		return http.StatusInternalServerError, CodeInternal
	}
}
