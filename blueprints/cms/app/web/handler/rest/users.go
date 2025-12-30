package rest

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/feature/accounts"
)

// Users handles REST API requests for users.
type Users struct {
	accounts  accounts.API
	getUserID func(*mizu.Ctx) string
}

// NewUsers creates a new users handler.
func NewUsers(a accounts.API, getUserID func(*mizu.Ctx) string) *Users {
	return &Users{accounts: a, getUserID: getUserID}
}

// List lists users.
func (h *Users) List(c *mizu.Ctx) error {
	page, perPage := ParsePagination(c)
	orderBy, order := ParseOrder(c, "registered_date", "desc")

	opts := accounts.ListOpts{
		Page:    page,
		PerPage: perPage,
		OrderBy: orderBy,
		Order:   order,
		Search:  c.Query("search"),
	}

	users, total, err := h.accounts.List(c.Request().Context(), opts)
	if err != nil {
		return InternalError(c, "Error retrieving users")
	}

	totalPages := (total + perPage - 1) / perPage
	return SuccessWithPagination(c, h.formatUsers(users), total, totalPages)
}

// Get retrieves a single user.
func (h *Users) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	user, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == accounts.ErrNotFound {
			return NotFound(c, "Invalid user ID.")
		}
		return InternalError(c, "Error retrieving user")
	}

	return Success(c, h.formatUser(user))
}

// Me retrieves the current user.
func (h *Users) Me(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "You are not currently logged in.")
	}

	user, err := h.accounts.GetByID(c.Request().Context(), userID)
	if err != nil {
		return InternalError(c, "Error retrieving user")
	}

	return Success(c, h.formatUser(user))
}

// Create creates a new user.
func (h *Users) Create(c *mizu.Ctx) error {
	var in accounts.CreateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	user, err := h.accounts.Create(c.Request().Context(), in)
	if err != nil {
		if err == accounts.ErrLoginTaken {
			return Conflict(c, "Username already exists.")
		}
		if err == accounts.ErrEmailTaken {
			return Conflict(c, "Email already exists.")
		}
		return InternalError(c, "Error creating user")
	}

	return Created(c, h.formatUser(user))
}

// Update updates a user.
func (h *Users) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 0); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	user, err := h.accounts.Update(c.Request().Context(), id, in)
	if err != nil {
		if err == accounts.ErrNotFound {
			return NotFound(c, "Invalid user ID.")
		}
		return InternalError(c, "Error updating user")
	}

	return Success(c, h.formatUser(user))
}

// Delete deletes a user.
func (h *Users) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	reassign := c.Query("reassign")

	user, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		if err == accounts.ErrNotFound {
			return NotFound(c, "Invalid user ID.")
		}
		return InternalError(c, "Error retrieving user")
	}

	if err := h.accounts.Delete(c.Request().Context(), id, reassign); err != nil {
		return InternalError(c, "Error deleting user")
	}

	return Deleted(c, h.formatUser(user))
}

func (h *Users) formatUsers(users []*accounts.User) []map[string]interface{} {
	result := make([]map[string]interface{}, len(users))
	for i, u := range users {
		result[i] = h.formatUser(u)
	}
	return result
}

func (h *Users) formatUser(u *accounts.User) map[string]interface{} {
	return map[string]interface{}{
		"id":              u.ID,
		"username":        u.Username,
		"name":            u.DisplayName,
		"first_name":      "",
		"last_name":       "",
		"email":           u.Email,
		"url":             u.URL,
		"description":     u.Description,
		"link":            "",
		"locale":          "en_US",
		"nickname":        u.Username,
		"slug":            u.Nicename,
		"roles":           u.Roles,
		"registered_date": u.Registered.Format("2006-01-02T15:04:05"),
		"capabilities":    u.Capabilities,
		"extra_capabilities": map[string]bool{},
		"avatar_urls":     u.AvatarURLs,
	}
}
