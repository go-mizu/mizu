package handler

import (
	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
)

// User handles user endpoints.
type User struct {
	accounts  accounts.API
	questions questions.API
}

// NewUser creates a new user handler.
func NewUser(accounts accounts.API, questions questions.API) *User {
	return &User{accounts: accounts, questions: questions}
}

// Get returns a user by ID.
func (h *User) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	user, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "User")
	}
	return Success(c, user)
}
