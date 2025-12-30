package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/comments"
)

// Comments handles comment endpoints.
type Comments struct {
	comments  comments.API
	getUserID func(*mizu.Ctx) string
}

// NewComments creates a new comments handler.
func NewComments(comments comments.API, getUserID func(*mizu.Ctx) string) *Comments {
	return &Comments{comments: comments, getUserID: getUserID}
}

// List lists comments.
func (h *Comments) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 50
	}

	in := &comments.ListIn{
		PostID:   c.Query("post_id"),
		ParentID: c.Query("parent_id"),
		AuthorID: c.Query("author_id"),
		Status:   c.Query("status"),
		Limit:    perPage,
		Offset:   (page - 1) * perPage,
	}

	list, total, err := h.comments.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list comments")
	}

	return List(c, list, total, page, perPage)
}

// ListByPost lists comments for a specific post.
func (h *Comments) ListByPost(c *mizu.Ctx) error {
	postID := c.Param("postID")
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 50
	}

	in := &comments.ListIn{
		Status: c.Query("status"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	list, total, err := h.comments.ListByPost(c.Context(), postID, in)
	if err != nil {
		return InternalError(c, "failed to list comments")
	}

	return List(c, list, total, page, perPage)
}

// Create creates a new comment.
func (h *Comments) Create(c *mizu.Ctx) error {
	postID := c.Param("postID")

	var in comments.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	in.PostID = postID
	in.AuthorID = h.getUserID(c)
	in.IPAddress = c.Request().RemoteAddr
	in.UserAgent = c.Request().UserAgent()

	comment, err := h.comments.Create(c.Context(), &in)
	if err != nil {
		if err == comments.ErrMissingContent {
			return BadRequest(c, err.Error())
		}
		return InternalError(c, "failed to create comment")
	}

	return Created(c, comment)
}

// Get retrieves a comment by ID.
func (h *Comments) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	comment, err := h.comments.GetByID(c.Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "comment not found")
		}
		return InternalError(c, "failed to get comment")
	}

	return OK(c, comment)
}

// Update updates a comment.
func (h *Comments) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in comments.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	comment, err := h.comments.Update(c.Context(), id, &in)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "comment not found")
		}
		return InternalError(c, "failed to update comment")
	}

	return OK(c, comment)
}

// Delete deletes a comment.
func (h *Comments) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.comments.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete comment")
	}

	return OK(c, map[string]string{"message": "comment deleted"})
}

// Approve approves a comment.
func (h *Comments) Approve(c *mizu.Ctx) error {
	id := c.Param("id")

	comment, err := h.comments.Approve(c.Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "comment not found")
		}
		return InternalError(c, "failed to approve comment")
	}

	return OK(c, comment)
}

// MarkAsSpam marks a comment as spam.
func (h *Comments) MarkAsSpam(c *mizu.Ctx) error {
	id := c.Param("id")

	comment, err := h.comments.MarkAsSpam(c.Context(), id)
	if err != nil {
		if err == comments.ErrNotFound {
			return NotFound(c, "comment not found")
		}
		return InternalError(c, "failed to mark comment as spam")
	}

	return OK(c, comment)
}
