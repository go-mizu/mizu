package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/boards"
	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
)

// Thread handles thread endpoints.
type Thread struct {
	threads      threads.API
	boards       boards.API
	votes        votes.API
	bookmarks    bookmarks.API
	getAccountID func(*mizu.Ctx) string
}

// NewThread creates a new thread handler.
func NewThread(threads threads.API, boards boards.API, votes votes.API, bookmarks bookmarks.API, getAccountID func(*mizu.Ctx) string) *Thread {
	return &Thread{
		threads:      threads,
		boards:       boards,
		votes:        votes,
		bookmarks:    bookmarks,
		getAccountID: getAccountID,
	}
}

// List lists threads.
func (h *Thread) List(c *mizu.Ctx) error {
	opts := threads.ListOpts{
		Limit:     25,
		SortBy:    threads.SortBy(c.Query("sort")),
		TimeRange: threads.TimeRange(c.Query("t")),
	}
	if opts.SortBy == "" {
		opts.SortBy = threads.SortHot
	}

	threadList, err := h.threads.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c)
	}

	// Enrich with viewer state
	viewerID := h.getAccountID(c)
	h.enrichThreads(c, threadList, viewerID)

	return Success(c, threadList)
}

// Create creates a thread.
func (h *Thread) Create(c *mizu.Ctx) error {
	boardName := c.Param("name")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	board, err := h.boards.GetByName(c.Request().Context(), boardName)
	if err != nil {
		if err == boards.ErrNotFound {
			return NotFound(c, "Board")
		}
		return InternalError(c)
	}

	var in threads.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	in.BoardID = board.ID

	thread, err := h.threads.Create(c.Request().Context(), accountID, in)
	if err != nil {
		switch err {
		case threads.ErrBoardLocked:
			return BadRequest(c, "Board is archived")
		default:
			return BadRequest(c, err.Error())
		}
	}

	return Created(c, thread)
}

// Get gets a thread by ID.
func (h *Thread) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	thread, err := h.threads.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == threads.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return InternalError(c)
	}

	// Increment view count
	_ = h.threads.IncrementViews(c.Request().Context(), id)

	// Enrich with viewer state
	viewerID := h.getAccountID(c)
	h.enrichThread(c, thread, viewerID)

	return Success(c, thread)
}

// Update updates a thread.
func (h *Thread) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	thread, err := h.threads.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == threads.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return InternalError(c)
	}

	if thread.AuthorID != accountID {
		return Forbidden(c)
	}

	var in threads.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.threads.Update(c.Request().Context(), id, in)
	if err != nil {
		return InternalError(c)
	}

	return Success(c, updated)
}

// Delete deletes a thread.
func (h *Thread) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	thread, err := h.threads.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == threads.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return InternalError(c)
	}

	if thread.AuthorID != accountID {
		return Forbidden(c)
	}

	if err := h.threads.Delete(c.Request().Context(), id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Deleted"})
}

// Bookmark bookmarks a thread.
func (h *Thread) Bookmark(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.bookmarks.Create(c.Request().Context(), accountID, bookmarks.TargetThread, id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Bookmarked"})
}

// Unbookmark removes a bookmark.
func (h *Thread) Unbookmark(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.bookmarks.Delete(c.Request().Context(), accountID, bookmarks.TargetThread, id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Unbookmarked"})
}

func (h *Thread) enrichThread(c *mizu.Ctx, thread *threads.Thread, viewerID string) {
	if viewerID == "" {
		return
	}

	_ = h.threads.EnrichThread(c.Request().Context(), thread, viewerID)

	// Get vote
	vote, err := h.votes.GetVote(c.Request().Context(), viewerID, votes.TargetThread, thread.ID)
	if err == nil && vote != nil {
		thread.Vote = vote.Value
	}

	// Get bookmark status
	isBookmarked, _ := h.bookmarks.IsBookmarked(c.Request().Context(), viewerID, bookmarks.TargetThread, thread.ID)
	thread.IsBookmarked = isBookmarked
}

func (h *Thread) enrichThreads(c *mizu.Ctx, threadList []*threads.Thread, viewerID string) {
	if viewerID == "" {
		return
	}

	// Get all IDs
	ids := make([]string, len(threadList))
	for i, t := range threadList {
		ids[i] = t.ID
	}

	// Get votes
	voteMap, _ := h.votes.GetVotes(c.Request().Context(), viewerID, votes.TargetThread, ids)

	// Get bookmarks
	bookmarkMap, _ := h.bookmarks.GetBookmarked(c.Request().Context(), viewerID, bookmarks.TargetThread, ids)

	// Apply to threads
	for _, t := range threadList {
		_ = h.threads.EnrichThread(c.Request().Context(), t, viewerID)
		if v, ok := voteMap[t.ID]; ok {
			t.Vote = v
		}
		if bookmarkMap[t.ID] {
			t.IsBookmarked = true
		}
	}
}
