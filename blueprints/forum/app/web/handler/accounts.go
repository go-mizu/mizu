package handler

import (
	"github.com/go-mizu/blueprints/forum/feature/accounts"
	"github.com/go-mizu/mizu"
)

// Account contains account-related handlers.
type Account struct {
	accounts     accounts.API
	getAccountID func(*mizu.Ctx) string
	optionalAuth func(*mizu.Ctx) string
}

// NewAccount creates new account handlers.
func NewAccount(
	accounts accounts.API,
	getAccountID func(*mizu.Ctx) string,
	optionalAuth func(*mizu.Ctx) string,
) *Account {
	return &Account{
		accounts:     accounts,
		getAccountID: getAccountID,
		optionalAuth: optionalAuth,
	}
}

// Get returns an account by ID or username.
func (h *Account) Get(c *mizu.Ctx) error {
	idOrUsername := c.Param("id")

	// Try by ID first
	account, err := h.accounts.GetByID(c.Request().Context(), idOrUsername)
	if err != nil {
		// Try by username
		account, err = h.accounts.GetByUsername(c.Request().Context(), idOrUsername)
		if err != nil {
			return c.JSON(404, ErrorResponse("Account not found"))
		}
	}

	return c.JSON(200, DataResponse(map[string]any{
		"account": account,
	}))
}

// Update updates an account.
func (h *Account) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := h.getAccountID(c)

	if id != accountID {
		return c.JSON(403, ErrorResponse("Cannot update other user's account"))
	}

	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("Invalid request body"))
	}

	account, err := h.accounts.Update(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"account": account,
	}))
}

// Search searches for accounts.
func (h *Account) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.JSON(400, ErrorResponse("Query parameter 'q' is required"))
	}

	limit := IntQuery(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	accounts, err := h.accounts.Search(c.Request().Context(), query, limit)
	if err != nil {
		return c.JSON(500, ErrorResponse(err.Error()))
	}

	return c.JSON(200, DataResponse(map[string]any{
		"accounts": accounts,
	}))
}
