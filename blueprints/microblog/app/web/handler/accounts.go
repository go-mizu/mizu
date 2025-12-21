package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
	"github.com/go-mizu/blueprints/microblog/feature/relationships"
	"github.com/go-mizu/blueprints/microblog/feature/timelines"
)

// Account contains account-related handlers.
type Account struct {
	accounts      accounts.API
	relationships relationships.API
	timelines     timelines.API
	getAccountID  func(*mizu.Ctx) string
	optionalAuth  func(*mizu.Ctx) string
}

// NewAccount creates new account handlers.
func NewAccount(
	accounts accounts.API,
	relationships relationships.API,
	timelines timelines.API,
	getAccountID func(*mizu.Ctx) string,
	optionalAuth func(*mizu.Ctx) string,
) *Account {
	return &Account{
		accounts:      accounts,
		relationships: relationships,
		timelines:     timelines,
		getAccountID:  getAccountID,
		optionalAuth:  optionalAuth,
	}
}

// VerifyCredentials returns the current user's account.
func (h *Account) VerifyCredentials(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	account, err := h.accounts.GetByID(c.Request().Context(), accountID)
	if err != nil {
		return c.JSON(404, ErrorResponse("NOT_FOUND", "Account not found"))
	}
	return c.JSON(200, map[string]any{"data": account})
}

// UpdateCredentials updates the current user's account.
func (h *Account) UpdateCredentials(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(400, ErrorResponse("INVALID_REQUEST", "Invalid request body"))
	}

	account, err := h.accounts.Update(c.Request().Context(), accountID, &in)
	if err != nil {
		return c.JSON(400, ErrorResponse("UPDATE_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]any{"data": account})
}

// GetAccount returns a specific account.
func (h *Account) GetAccount(c *mizu.Ctx) error {
	id := c.Param("id")

	// Check if it's a username
	var account *accounts.Account
	var err error

	if len(id) < 26 { // ULIDs are 26 chars
		account, err = h.accounts.GetByUsername(c.Request().Context(), id)
	} else {
		account, err = h.accounts.GetByID(c.Request().Context(), id)
	}

	if err != nil {
		return c.JSON(404, ErrorResponse("NOT_FOUND", "Account not found"))
	}

	// Load follower/following counts
	account.FollowersCount, _ = h.relationships.CountFollowers(c.Request().Context(), account.ID)
	account.FollowingCount, _ = h.relationships.CountFollowing(c.Request().Context(), account.ID)

	return c.JSON(200, map[string]any{"data": account})
}

// GetAccountPosts returns a specific account's posts.
func (h *Account) GetAccountPosts(c *mizu.Ctx) error {
	accountID := c.Param("id")
	viewerID := h.optionalAuth(c)
	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")

	postList, err := h.timelines.Account(c.Request().Context(), accountID, viewerID, limit, maxID, false, false)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	return c.JSON(200, map[string]any{"data": postList})
}

// GetAccountFollowers returns a specific account's followers.
func (h *Account) GetAccountFollowers(c *mizu.Ctx) error {
	accountID := c.Param("id")
	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	ids, err := h.relationships.GetFollowers(c.Request().Context(), accountID, limit, offset)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	// Load accounts
	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := h.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}

// GetAccountFollowing returns accounts that a specific account follows.
func (h *Account) GetAccountFollowing(c *mizu.Ctx) error {
	accountID := c.Param("id")
	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	ids, err := h.relationships.GetFollowing(c.Request().Context(), accountID, limit, offset)
	if err != nil {
		return c.JSON(500, ErrorResponse("FETCH_FAILED", err.Error()))
	}

	var accts []*accounts.Account
	for _, id := range ids {
		if a, err := h.accounts.GetByID(c.Request().Context(), id); err == nil {
			accts = append(accts, a)
		}
	}

	return c.JSON(200, map[string]any{"data": accts})
}
