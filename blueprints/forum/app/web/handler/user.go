package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/forum/feature/comments"
	"github.com/go-mizu/mizu/blueprints/forum/feature/threads"
)

// User handles user endpoints.
type User struct {
	accounts accounts.API
	threads  threads.API
	comments comments.API
}

// NewUser creates a new user handler.
func NewUser(accounts accounts.API, threads threads.API, comments comments.API) *User {
	return &User{
		accounts: accounts,
		threads:  threads,
		comments: comments,
	}
}

// Get gets a user by username.
func (h *User) Get(c *mizu.Ctx) error {
	username := c.Param("username")

	account, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		if err == accounts.ErrNotFound {
			return NotFound(c, "User")
		}
		return InternalError(c)
	}

	// Don't expose email
	account.Email = ""

	return Success(c, account)
}

// ListThreads lists a user's threads.
func (h *User) ListThreads(c *mizu.Ctx) error {
	username := c.Param("username")

	account, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		if err == accounts.ErrNotFound {
			return NotFound(c, "User")
		}
		return InternalError(c)
	}

	opts := threads.ListOpts{
		Limit:  25,
		SortBy: threads.SortNew,
	}

	threadList, err := h.threads.ListByAuthor(c.Request().Context(), account.ID, opts)
	if err != nil {
		return InternalError(c)
	}

	return Success(c, threadList)
}

// ListComments lists a user's comments.
func (h *User) ListComments(c *mizu.Ctx) error {
	username := c.Param("username")

	account, err := h.accounts.GetByUsername(c.Request().Context(), username)
	if err != nil {
		if err == accounts.ErrNotFound {
			return NotFound(c, "User")
		}
		return InternalError(c)
	}

	opts := comments.ListOpts{
		Limit:  25,
		SortBy: comments.CommentSortNew,
	}

	commentList, err := h.comments.ListByAuthor(c.Request().Context(), account.ID, opts)
	if err != nil {
		return InternalError(c)
	}

	return Success(c, commentList)
}
