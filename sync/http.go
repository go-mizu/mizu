package sync

import (
	"errors"
	"net/http"

	"github.com/go-mizu/mizu"
)

// PushRequest is the payload for POST /_sync/push
type PushRequest struct {
	Mutations []Mutation `json:"mutations"`
}

// PushResponse is returned from POST /_sync/push
type PushResponse struct {
	Results []Result `json:"results"`
}

// PullRequest is the payload for POST /_sync/pull
type PullRequest struct {
	Scope  string `json:"scope,omitempty"`
	Cursor uint64 `json:"cursor"`
	Limit  int    `json:"limit,omitempty"`
}

// PullResponse is returned from POST /_sync/pull
type PullResponse struct {
	Changes []Change `json:"changes"`
	HasMore bool     `json:"has_more"`
}

// SnapshotRequest is the payload for POST /_sync/snapshot
type SnapshotRequest struct {
	Scope string `json:"scope,omitempty"`
}

// SnapshotResponse is returned from POST /_sync/snapshot
type SnapshotResponse struct {
	Data   map[string]map[string][]byte `json:"data"`
	Cursor uint64                       `json:"cursor"`
}

// Mount registers sync routes on a Mizu app at /_sync/*.
func (e *Engine) Mount(app *mizu.App) {
	e.MountAt(app, "/_sync")
}

// MountAt registers sync routes at a custom prefix.
func (e *Engine) MountAt(app *mizu.App, prefix string) {
	app.Post(prefix+"/push", e.handlePush())
	app.Post(prefix+"/pull", e.handlePull())
	app.Post(prefix+"/snapshot", e.handleSnapshot())
}

func (e *Engine) handlePush() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req PushRequest
		if err := c.BindJSON(&req, 1<<20); err != nil { // 1MB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		if len(req.Mutations) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "no mutations provided",
			})
		}

		results, err := e.Push(c.Context(), req.Mutations)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, PushResponse{Results: results})
	}
}

func (e *Engine) handlePull() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req PullRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		changes, hasMore, err := e.Pull(c.Context(), req.Scope, req.Cursor, req.Limit)
		if err != nil {
			code := http.StatusInternalServerError
			errCode := CodeInternal
			if errors.Is(err, ErrCursorTooOld) {
				code = http.StatusGone
				errCode = CodeCursorTooOld
			}
			return c.JSON(code, map[string]string{
				"code":  errCode,
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, PullResponse{
			Changes: changes,
			HasMore: hasMore,
		})
	}
}

func (e *Engine) handleSnapshot() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req SnapshotRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		data, cursor, err := e.Snapshot(c.Context(), req.Scope)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, SnapshotResponse{
			Data:   data,
			Cursor: cursor,
		})
	}
}

// Handlers provides individual handlers for custom mounting.
type Handlers struct {
	Push     mizu.Handler
	Pull     mizu.Handler
	Snapshot mizu.Handler
}

// Handlers returns individual sync handlers.
func (e *Engine) Handlers() Handlers {
	return Handlers{
		Push:     e.handlePush(),
		Pull:     e.handlePull(),
		Snapshot: e.handleSnapshot(),
	}
}
