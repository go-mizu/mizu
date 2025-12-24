package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/lists"
)

// List handles list endpoints.
type List struct {
	lists        lists.API
	getAccountID func(*mizu.Ctx) string
}

// NewList creates a new list handler.
func NewList(listsSvc lists.API, getAccountID func(*mizu.Ctx) string) *List {
	return &List{
		lists:        listsSvc,
		getAccountID: getAccountID,
	}
}

// List handles GET /api/v1/lists
func (h *List) List(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	ls, err := h.lists.GetByAccount(c.Request().Context(), accountID)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, ls)
}

// Create handles POST /api/v1/lists
func (h *List) Create(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	var in lists.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	if in.Title == "" {
		return UnprocessableEntity(c, "title is required")
	}

	list, err := h.lists.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		return InternalError(c, err)
	}

	return Created(c, list)
}

// Get handles GET /api/v1/lists/:id
func (h *List) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	list, err := h.lists.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "list")
	}

	return Success(c, list)
}

// Update handles PUT /api/v1/lists/:id
func (h *List) Update(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	id := c.Param("id")

	var in lists.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	list, err := h.lists.Update(c.Request().Context(), accountID, id, &in)
	if err != nil {
		switch err {
		case lists.ErrNotFound:
			return NotFound(c, "list")
		case lists.ErrUnauthorized:
			return Forbidden(c)
		default:
			return InternalError(c, err)
		}
	}

	return Success(c, list)
}

// Delete handles DELETE /api/v1/lists/:id
func (h *List) Delete(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	id := c.Param("id")

	if err := h.lists.Delete(c.Request().Context(), accountID, id); err != nil {
		switch err {
		case lists.ErrNotFound:
			return NotFound(c, "list")
		case lists.ErrUnauthorized:
			return Forbidden(c)
		default:
			return InternalError(c, err)
		}
	}

	return NoContent(c)
}

// GetMembers handles GET /api/v1/lists/:id/accounts
func (h *List) GetMembers(c *mizu.Ctx) error {
	id := c.Param("id")
	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	members, err := h.lists.GetMembers(c.Request().Context(), id, limit, offset)
	if err != nil {
		return InternalError(c, err)
	}

	// Return accounts
	accounts := make([]interface{}, len(members))
	for i, m := range members {
		accounts[i] = m.Account
	}

	return Success(c, accounts)
}

// AddMember handles POST /api/v1/lists/:id/accounts
func (h *List) AddMember(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	listID := c.Param("id")

	var body struct {
		AccountIDs []string `json:"account_ids"`
	}
	if err := c.BindJSON(&body, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	for _, memberID := range body.AccountIDs {
		if err := h.lists.AddMember(c.Request().Context(), accountID, listID, memberID); err != nil {
			switch err {
			case lists.ErrNotFound:
				return NotFound(c, "list")
			case lists.ErrUnauthorized:
				return Forbidden(c)
			case lists.ErrAlreadyMember:
				// Skip
			default:
				return InternalError(c, err)
			}
		}
	}

	return NoContent(c)
}

// RemoveMember handles DELETE /api/v1/lists/:id/accounts
func (h *List) RemoveMember(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	if accountID == "" {
		return Unauthorized(c)
	}

	listID := c.Param("id")

	var body struct {
		AccountIDs []string `json:"account_ids"`
	}
	if err := c.BindJSON(&body, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	for _, memberID := range body.AccountIDs {
		if err := h.lists.RemoveMember(c.Request().Context(), accountID, listID, memberID); err != nil {
			switch err {
			case lists.ErrNotFound:
				return NotFound(c, "list")
			case lists.ErrUnauthorized:
				return Forbidden(c)
			case lists.ErrNotMember:
				// Skip
			default:
				return InternalError(c, err)
			}
		}
	}

	return NoContent(c)
}
