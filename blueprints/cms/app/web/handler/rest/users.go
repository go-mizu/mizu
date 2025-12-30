package rest

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/users"
)

// Users handles user endpoints.
type Users struct {
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewUsers creates a new users handler.
func NewUsers(users users.API, getUserID func(*mizu.Ctx) string) *Users {
	return &Users{users: users, getUserID: getUserID}
}

// List lists users.
func (h *Users) List(c *mizu.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	in := &users.ListIn{
		Role:   c.Query("role"),
		Status: c.Query("status"),
		Search: c.Query("search"),
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	list, total, err := h.users.List(c.Context(), in)
	if err != nil {
		return InternalError(c, "failed to list users")
	}

	return List(c, list, total, page, perPage)
}

// Get retrieves a user by ID.
func (h *Users) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	user, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "user not found")
		}
		return InternalError(c, "failed to get user")
	}

	return OK(c, user)
}

// Update updates a user.
func (h *Users) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in users.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	user, err := h.users.Update(c.Context(), id, &in)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "user not found")
		}
		return InternalError(c, "failed to update user")
	}

	return OK(c, user)
}

// Delete deletes a user.
func (h *Users) Delete(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.users.Delete(c.Context(), id); err != nil {
		return InternalError(c, "failed to delete user")
	}

	return OK(c, map[string]string{"message": "user deleted"})
}
