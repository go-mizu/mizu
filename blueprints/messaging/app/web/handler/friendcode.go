package handler

import (
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/messaging/feature/friendcode"
)

// FriendCode handles friend code endpoints.
type FriendCode struct {
	svc       friendcode.API
	getUserID func(*mizu.Ctx) string
}

// NewFriendCode creates a new FriendCode handler.
func NewFriendCode(svc friendcode.API, getUserID func(*mizu.Ctx) string) *FriendCode {
	return &FriendCode{
		svc:       svc,
		getUserID: getUserID,
	}
}

// Generate creates or retrieves a friend code for the current user.
func (h *FriendCode) Generate(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	code, err := h.svc.Generate(c.Request().Context(), userID)
	if err != nil {
		return InternalError(c, "Failed to generate friend code")
	}

	return Success(c, code)
}

// Resolve validates a friend code and returns user info.
func (h *FriendCode) Resolve(c *mizu.Ctx) error {
	code := c.Param("code")
	if code == "" {
		return BadRequest(c, "Code is required")
	}

	user, err := h.svc.Resolve(c.Request().Context(), code)
	if err != nil {
		switch err {
		case friendcode.ErrNotFound:
			return NotFound(c, "Invalid friend code")
		case friendcode.ErrExpired:
			return BadRequest(c, "Friend code has expired")
		default:
			return InternalError(c, "Failed to resolve friend code")
		}
	}

	return Success(c, user)
}

// AddFriend adds a contact using a friend code.
func (h *FriendCode) AddFriend(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	code := c.Param("code")
	if code == "" {
		return BadRequest(c, "Code is required")
	}

	contact, err := h.svc.AddFriend(c.Request().Context(), userID, code)
	if err != nil {
		switch err {
		case friendcode.ErrNotFound:
			return NotFound(c, "Invalid friend code")
		case friendcode.ErrExpired:
			return BadRequest(c, "Friend code has expired")
		case friendcode.ErrSelfAdd:
			return BadRequest(c, "Cannot add yourself as a friend")
		case friendcode.ErrAlreadyAdded:
			return BadRequest(c, "User is already in your contacts")
		default:
			return InternalError(c, "Failed to add friend")
		}
	}

	return Created(c, contact)
}

// Revoke invalidates the current user's friend code.
func (h *FriendCode) Revoke(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "Authentication required")
	}

	if err := h.svc.Revoke(c.Request().Context(), userID); err != nil {
		return InternalError(c, "Failed to revoke friend code")
	}

	return Success(c, nil)
}
