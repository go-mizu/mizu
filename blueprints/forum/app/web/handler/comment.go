package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/bookmarks"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
	"github.com/go-mizu/mizu/blueprints/forum/feature/votes"
)

// Comment handles comment endpoints.
type Comment struct {
	comments     comments.API
	threads      threads.API
	votes        votes.API
	bookmarks    bookmarks.API
	getAccountID func(*mizu.Ctx) string
}

// NewComment creates a new comment handler.
func NewComment(comments comments.API, threads threads.API, votes votes.API, bookmarks bookmarks.API, getAccountID func(*mizu.Ctx) string) *Comment {
	return &Comment{
		comments:     comments,
		threads:      threads,
		votes:        votes,
		bookmarks:    bookmarks,
		getAccountID: getAccountID,
	}
}

// List lists comments for a thread.
func (h *Comment) List(c *mizu.Ctx) error {
	threadID := c.Param("id")

	opts := comments.TreeOpts{
		Sort:       comments.CommentSort(c.Query("sort")),
		Limit:      200,
		MaxDepth:   10,
		CollapseAt: 5,
	}
	if opts.Sort == "" {
		opts.Sort = comments.CommentSortBest
	}

	tree, err := h.comments.GetTree(c.Request().Context(), threadID, opts)
	if err != nil {
		return InternalError(c)
	}

	// Enrich with viewer state
	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.comments.EnrichComments(c.Request().Context(), tree, viewerID)
		h.enrichCommentVotes(c, tree, viewerID)
	}

	return Success(c, tree)
}

// Create creates a comment.
func (h *Comment) Create(c *mizu.Ctx) error {
	threadID := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	// Check thread exists
	thread, err := h.threads.GetByID(c.Request().Context(), threadID)
	if err != nil {
		if err == threads.ErrNotFound {
			return NotFound(c, "Thread")
		}
		return InternalError(c)
	}

	if thread.IsLocked {
		return BadRequest(c, "Thread is locked")
	}

	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}
	in.ThreadID = threadID

	comment, err := h.comments.Create(c.Request().Context(), accountID, in)
	if err != nil {
		switch err {
		case comments.ErrThreadLocked:
			return BadRequest(c, "Thread is locked")
		case comments.ErrMaxDepth:
			return BadRequest(c, "Maximum comment depth reached")
		default:
			return BadRequest(c, err.Error())
		}
	}

	return Created(c, comment)
}

// Get gets a comment by ID.
func (h *Comment) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	comment, err := h.comments.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return InternalError(c)
	}

	// Enrich with viewer state
	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.comments.EnrichComment(c.Request().Context(), comment, viewerID)

		// Get vote
		vote, err := h.votes.GetVote(c.Request().Context(), viewerID, votes.TargetComment, comment.ID)
		if err == nil && vote != nil {
			comment.Vote = vote.Value
		}
	}

	return Success(c, comment)
}

// Update updates a comment.
func (h *Comment) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	comment, err := h.comments.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return InternalError(c)
	}

	if comment.AuthorID != accountID {
		return Forbidden(c)
	}

	var in struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.comments.Update(c.Request().Context(), id, in.Content)
	if err != nil {
		return InternalError(c)
	}

	return Success(c, updated)
}

// Delete deletes a comment.
func (h *Comment) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	comment, err := h.comments.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "Comment")
		}
		return InternalError(c)
	}

	if comment.AuthorID != accountID {
		return Forbidden(c)
	}

	if err := h.comments.Delete(c.Request().Context(), id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Deleted"})
}

// Bookmark bookmarks a comment.
func (h *Comment) Bookmark(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.bookmarks.Create(c.Request().Context(), accountID, bookmarks.TargetComment, id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Bookmarked"})
}

// Unbookmark removes a bookmark.
func (h *Comment) Unbookmark(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	if err := h.bookmarks.Delete(c.Request().Context(), accountID, bookmarks.TargetComment, id); err != nil {
		return InternalError(c)
	}

	return Success(c, map[string]any{"message": "Unbookmarked"})
}

func (h *Comment) enrichCommentVotes(c *mizu.Ctx, commentList []*comments.Comment, viewerID string) {
	var allComments []*comments.Comment
	var collectComments func([]*comments.Comment)
	collectComments = func(list []*comments.Comment) {
		for _, comment := range list {
			allComments = append(allComments, comment)
			if len(comment.Children) > 0 {
				collectComments(comment.Children)
			}
		}
	}
	collectComments(commentList)

	if len(allComments) == 0 {
		return
	}

	ids := make([]string, len(allComments))
	for i, comment := range allComments {
		ids[i] = comment.ID
	}

	voteMap, _ := h.votes.GetVotes(c.Request().Context(), viewerID, votes.TargetComment, ids)

	for _, comment := range allComments {
		if v, ok := voteMap[comment.ID]; ok {
			comment.Vote = v
		}
	}
}
