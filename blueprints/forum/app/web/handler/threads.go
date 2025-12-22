package handler

import (
	"github.com/go-mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu"
)

// Thread contains thread-related handlers.
type Thread struct {
	threads      threads.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewThread creates new thread handlers.
func NewThread(
	threads threads.API,
	getAccountID func(*mizu.Ctx) string,
	optionalAuth func(*mizu.Ctx) string,
) *Thread {
	return &Thread{
		threads:      threads,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// ListByForum lists threads in a forum.
func (h *Thread) ListByForum(c *mizu.Ctx) error {
	forumID := c.Param("id")
	viewerID := h.optionalAuth(c)

	sort := StringQuery(c, "sort", "hot")
	limit := IntQuery(c, "limit", 25)
	if limit > 100 {
		limit = 100
	}
	after := c.Query("after")

	sortOption := threads.SortOption(sort)
	list, err := h.threads.ListByForum(c.Request().Context(), forumID, viewerID, sortOption, limit, after)
	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"threads": list.Threads,
		"max_id":  list.MaxID,
		"min_id":  list.MinID,
	}))
}

// Get returns a specific thread.
func (h *Thread) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	viewerID := h.optionalAuth(c)

	thread, err := h.threads.GetByID(c.Request().Context(), id, viewerID)
	if err != nil {
		return c.JSON(404, ErrorResponse("Thread not found"))
	}

	// Increment view count asynchronously
	go h.threads.IncrementViews(c.Request().Context(), id)

	return c.JSON(200, DataResponse(map[string]any{
		"thread": thread,
	}))
}

// Create creates a new thread.
func (h *Thread) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)

	var in threads.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	thread, err := h.threads.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(201, DataResponse(map[string]any{
		"thread": thread,
	}))
}

// Update updates a thread.
func (h *Thread) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	var in threads.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	thread, err := h.threads.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"thread": thread,
	}))
}

// Delete deletes a thread.
func (h *Thread) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.threads.Delete(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Thread deleted successfully",
	}))
}

// Lock locks a thread (moderator only).
func (h *Thread) Lock(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.threads.Lock(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Thread locked",
	}))
}

// Unlock unlocks a thread (moderator only).
func (h *Thread) Unlock(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.threads.Unlock(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Thread unlocked",
	}))
}

// Sticky stickies a thread (moderator only).
func (h *Thread) Sticky(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.threads.Sticky(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Thread stickied",
	}))
}

// Unsticky unstickies a thread (moderator only).
func (h *Thread) Unsticky(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if err := h.threads.Unsticky(c.Request().Context(), id, accountID); err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"message": "Thread unstickied",
	}))
}

// ListByAccount lists threads created by an account.
func (h *Thread) ListByAccount(c *mizu.Ctx) error {
	accountID := c.Param("id")
	viewerID := h.optionalAuth(c)

	limit := IntQuery(c, "limit", 25)
	if limit > 100 {
		limit = 100
	}
	after := c.Query("after")

	list, err := h.threads.ListByAccount(c.Request().Context(), accountID, viewerID, limit, after)
	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"threads": list.Threads,
		"max_id":  list.MaxID,
		"min_id":  list.MinID,
	}))
}
