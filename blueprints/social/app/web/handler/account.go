package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/accounts"
	"github.com/go-mizu/blueprints/social/feature/relationships"
	"github.com/go-mizu/blueprints/social/feature/timelines"
)

// Account handles account endpoints.
type Account struct {
	accounts      accounts.API
	relationships relationships.API
	timelines     timelines.API
	getAccountID  func(*mizu.Ctx) string
	optionalAuth  func(*mizu.Ctx) string
}

// NewAccount creates a new account handler.
func NewAccount(accountsSvc accounts.API, relsSvc relationships.API, timelinesSvc timelines.API, getAccountID func(*mizu.Ctx) string, optionalAuth func(*mizu.Ctx) string) *Account {
	return &Account{
		accounts:      accountsSvc,
		relationships: relsSvc,
		timelines:     timelinesSvc,
		getAccountID:  getAccountID,
		optionalAuth:  optionalAuth,
	}
}

// VerifyCredentials handles GET /api/v1/accounts/verify_credentials
func (h *Account) VerifyCredentials(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	account, err := h.accounts.GetByID(c.Request().Context(), accountID)
	if err != nil {
		return NotFound(c, "account")
	}

	_ = h.accounts.PopulateStats(c.Request().Context(), account)

	return Success(c, account)
}

// UpdateCredentials handles PATCH /api/v1/accounts/update_credentials
func (h *Account) UpdateCredentials(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	var in accounts.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	account, err := h.accounts.Update(c.Request().Context(), accountID, &in)
	if err != nil {
		return InternalError(c, err)
	}

	_ = h.accounts.PopulateStats(c.Request().Context(), account)

	return Success(c, account)
}

// GetAccount handles GET /api/v1/accounts/:id
func (h *Account) GetAccount(c *mizu.Ctx) error {
	id := c.Param("id")

	account, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		// Try by username
		account, err = h.accounts.GetByUsername(c.Request().Context(), id)
		if err != nil {
			return NotFound(c, "account")
		}
	}

	_ = h.accounts.PopulateStats(c.Request().Context(), account)

	viewerID := h.getAccountID(c)
	if viewerID != "" {
		_ = h.accounts.PopulateRelationship(c.Request().Context(), account, viewerID)
	}

	return Success(c, account)
}

// GetAccountPosts handles GET /api/v1/accounts/:id/posts
func (h *Account) GetAccountPosts(c *mizu.Ctx) error {
	id := c.Param("id")

	account, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		account, err = h.accounts.GetByUsername(c.Request().Context(), id)
		if err != nil {
			return NotFound(c, "account")
		}
	}

	limit := IntQuery(c, "limit", 20)
	maxID := c.Query("max_id")
	minID := c.Query("min_id")
	excludeReplies := BoolQuery(c, "exclude_replies", false)
	onlyMedia := BoolQuery(c, "only_media", false)

	opts := timelines.TimelineOpts{
		Limit:     limit,
		MaxID:     maxID,
		MinID:     minID,
		OnlyMedia: onlyMedia,
	}

	posts, err := h.timelines.User(c.Request().Context(), account.ID, opts, !excludeReplies)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, posts)
}

// GetAccountFollowers handles GET /api/v1/accounts/:id/followers
func (h *Account) GetAccountFollowers(c *mizu.Ctx) error {
	id := c.Param("id")

	account, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		account, err = h.accounts.GetByUsername(c.Request().Context(), id)
		if err != nil {
			return NotFound(c, "account")
		}
	}

	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	follows, err := h.relationships.GetFollowers(c.Request().Context(), relationships.FollowersOpts{
		AccountID: account.ID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return InternalError(c, err)
	}

	// Get account info for followers
	accountIDs := make([]string, len(follows))
	for i, f := range follows {
		accountIDs[i] = f.FollowerID
	}

	accounts, err := h.accounts.GetByIDs(c.Request().Context(), accountIDs)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, accounts)
}

// GetAccountFollowing handles GET /api/v1/accounts/:id/following
func (h *Account) GetAccountFollowing(c *mizu.Ctx) error {
	id := c.Param("id")

	account, err := h.accounts.GetByID(c.Request().Context(), id)
	if err != nil {
		account, err = h.accounts.GetByUsername(c.Request().Context(), id)
		if err != nil {
			return NotFound(c, "account")
		}
	}

	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	follows, err := h.relationships.GetFollowing(c.Request().Context(), relationships.FollowersOpts{
		AccountID: account.ID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return InternalError(c, err)
	}

	// Get account info for following
	accountIDs := make([]string, len(follows))
	for i, f := range follows {
		accountIDs[i] = f.FollowingID
	}

	accounts, err := h.accounts.GetByIDs(c.Request().Context(), accountIDs)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, accounts)
}

// Search handles GET /api/v1/accounts/search
func (h *Account) Search(c *mizu.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return BadRequest(c, "query is required")
	}

	limit := IntQuery(c, "limit", 20)

	accounts, err := h.accounts.Search(c.Request().Context(), query, limit)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, accounts)
}
