package sync

import (
	"net/http"

	"github.com/go-mizu/mizu"
)

// PushRequest is the payload for POST /_sync/push
type PushRequest struct {
	Mutations []Mutation `json:"mutations"`
}

// PushResponse is returned from POST /_sync/push
type PushResponse struct {
	Results []MutationResult `json:"results"`
	Cursor  uint64           `json:"cursor"`
}

// PullRequest is the payload for POST /_sync/pull
type PullRequest struct {
	Scope  string `json:"scope"`
	Cursor uint64 `json:"cursor"`
	Limit  int    `json:"limit,omitempty"`
}

// PullResponse is returned from POST /_sync/pull
type PullResponse struct {
	Changes []Change `json:"changes"`
	Cursor  uint64   `json:"cursor"`
	HasMore bool     `json:"has_more"`
}

// SnapshotRequest is the payload for POST /_sync/snapshot
type SnapshotRequest struct {
	Scope string `json:"scope"`
}

// SnapshotResponse is returned from POST /_sync/snapshot
type SnapshotResponse struct {
	Data   map[string]map[string]any `json:"data"`
	Cursor uint64                    `json:"cursor"`
}

// Mount registers sync routes on a Mizu app.
func (e *Engine) Mount(app *mizu.App) {
	app.Post("/_sync/push", e.pushHandler())
	app.Post("/_sync/pull", e.pullHandler())
	app.Post("/_sync/snapshot", e.snapshotHandler())
}

// MountAt registers sync routes at a custom prefix.
func (e *Engine) MountAt(app *mizu.App, prefix string) {
	app.Post(prefix+"/push", e.pushHandler())
	app.Post(prefix+"/pull", e.pullHandler())
	app.Post(prefix+"/snapshot", e.snapshotHandler())
}

// pushHandler handles POST /_sync/push
func (e *Engine) pushHandler() mizu.Handler {
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

		// Get final cursor
		cursor, _ := e.changelog.Cursor(c.Context())

		return c.JSON(http.StatusOK, PushResponse{
			Results: results,
			Cursor:  cursor,
		})
	}
}

// pullHandler handles POST /_sync/pull
func (e *Engine) pullHandler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req PullRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		changes, cursor, hasMore, err := e.Pull(c.Context(), req.Scope, req.Cursor, req.Limit)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, PullResponse{
			Changes: changes,
			Cursor:  cursor,
			HasMore: hasMore,
		})
	}
}

// snapshotHandler handles POST /_sync/snapshot
func (e *Engine) snapshotHandler() mizu.Handler {
	return func(c *mizu.Ctx) error {
		var req SnapshotRequest
		if err := c.BindJSON(&req, 1<<16); err != nil { // 64KB max
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		if req.Scope == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "scope is required",
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

// Handlers returns individual handlers for custom mounting.
type Handlers struct {
	Push     mizu.Handler
	Pull     mizu.Handler
	Snapshot mizu.Handler
}

// GetHandlers returns individual sync handlers.
func (e *Engine) GetHandlers() Handlers {
	return Handlers{
		Push:     e.pushHandler(),
		Pull:     e.pullHandler(),
		Snapshot: e.snapshotHandler(),
	}
}
