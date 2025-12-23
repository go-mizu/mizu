package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/news/feature/comments"
	"github.com/go-mizu/mizu/blueprints/news/feature/stories"
	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// User handles user endpoints.
type User struct {
	users    *users.Service
	stories  *stories.Service
	comments *comments.Service
}

// NewUser creates a new user handler.
func NewUser(users *users.Service, stories *stories.Service, comments *comments.Service) *User {
	return &User{
		users:    users,
		stories:  stories,
		comments: comments,
	}
}

// Get gets a user by username.
func (h *User) Get(c *mizu.Ctx) error {
	username := c.Param("username")

	user, err := h.users.GetByUsername(c.Request().Context(), username)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "user")
		}
		return InternalError(c, err)
	}

	// Get story count
	storiesList, err := h.stories.ListByAuthor(c.Request().Context(), user.ID, 1, 0, "")
	if err == nil {
		user.StoryCount = int64(len(storiesList))
	}

	// Get comment count
	commentsList, err := h.comments.ListByAuthor(c.Request().Context(), user.ID, 1, 0, "")
	if err == nil {
		user.CommentCount = int64(len(commentsList))
	}

	return Success(c, user)
}

// ListStories lists a user's stories.
func (h *User) ListStories(c *mizu.Ctx) error {
	username := c.Param("username")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 30
	}
	offset, _ := strconv.Atoi(c.Query("offset"))

	user, err := h.users.GetByUsername(c.Request().Context(), username)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "user")
		}
		return InternalError(c, err)
	}

	list, err := h.stories.ListByAuthor(c.Request().Context(), user.ID, limit, offset, "")
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, list)
}

// ListComments lists a user's comments.
func (h *User) ListComments(c *mizu.Ctx) error {
	username := c.Param("username")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit == 0 {
		limit = 30
	}
	offset, _ := strconv.Atoi(c.Query("offset"))

	user, err := h.users.GetByUsername(c.Request().Context(), username)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "user")
		}
		return InternalError(c, err)
	}

	list, err := h.comments.ListByAuthor(c.Request().Context(), user.ID, limit, offset, "")
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, list)
}
