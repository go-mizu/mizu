package handler

import (
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/social/feature/notifications"
	"github.com/go-mizu/blueprints/social/feature/relationships"
)

// Relationship handles relationship endpoints.
type Relationship struct {
	relationships relationships.API
	notifications notifications.API
	getAccountID  func(*mizu.Ctx) string
}

// NewRelationship creates a new relationship handler.
func NewRelationship(relsSvc relationships.API, notificationsSvc notifications.API, getAccountID func(*mizu.Ctx) string) *Relationship {
	return &Relationship{
		relationships: relsSvc,
		notifications: notificationsSvc,
		getAccountID:  getAccountID,
	}
}

// Follow handles POST /api/v1/accounts/:id/follow
func (h *Relationship) Follow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	rel, err := h.relationships.Follow(c.Request().Context(), accountID, targetID)
	if err != nil {
		switch err {
		case relationships.ErrCannotFollowSelf:
			return BadRequest(c, "cannot follow yourself")
		case relationships.ErrAlreadyFollowing:
			return Conflict(c, "already following")
		case relationships.ErrBlocked:
			return Forbidden(c)
		default:
			return InternalError(c, err)
		}
	}

	// Notify
	if h.notifications != nil {
		if rel.Requested {
			_ = h.notifications.NotifyFollowRequest(c.Request().Context(), accountID, targetID)
		} else {
			_ = h.notifications.NotifyFollow(c.Request().Context(), accountID, targetID)
		}
	}

	return Success(c, rel)
}

// Unfollow handles POST /api/v1/accounts/:id/unfollow
func (h *Relationship) Unfollow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	rel, err := h.relationships.Unfollow(c.Request().Context(), accountID, targetID)
	if err != nil {
		switch err {
		case relationships.ErrCannotFollowSelf:
			return BadRequest(c, "cannot unfollow yourself")
		case relationships.ErrNotFollowing:
			return Conflict(c, "not following")
		default:
			return InternalError(c, err)
		}
	}

	return Success(c, rel)
}

// Block handles POST /api/v1/accounts/:id/block
func (h *Relationship) Block(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	rel, err := h.relationships.Block(c.Request().Context(), accountID, targetID)
	if err != nil {
		switch err {
		case relationships.ErrCannotBlockSelf:
			return BadRequest(c, "cannot block yourself")
		case relationships.ErrAlreadyBlocked:
			return Conflict(c, "already blocked")
		default:
			return InternalError(c, err)
		}
	}

	return Success(c, rel)
}

// Unblock handles POST /api/v1/accounts/:id/unblock
func (h *Relationship) Unblock(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	rel, err := h.relationships.Unblock(c.Request().Context(), accountID, targetID)
	if err != nil {
		switch err {
		case relationships.ErrCannotBlockSelf:
			return BadRequest(c, "cannot unblock yourself")
		case relationships.ErrNotBlocked:
			return Conflict(c, "not blocked")
		default:
			return InternalError(c, err)
		}
	}

	return Success(c, rel)
}

// Mute handles POST /api/v1/accounts/:id/mute
func (h *Relationship) Mute(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	var in relationships.MuteIn
	_ = c.BindJSON(&in, 1<<20)

	rel, err := h.relationships.Mute(c.Request().Context(), accountID, targetID, &in)
	if err != nil {
		switch err {
		case relationships.ErrCannotMuteSelf:
			return BadRequest(c, "cannot mute yourself")
		case relationships.ErrAlreadyMuted:
			return Conflict(c, "already muted")
		default:
			return InternalError(c, err)
		}
	}

	return Success(c, rel)
}

// Unmute handles POST /api/v1/accounts/:id/unmute
func (h *Relationship) Unmute(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	targetID := c.Param("id")

	rel, err := h.relationships.Unmute(c.Request().Context(), accountID, targetID)
	if err != nil {
		switch err {
		case relationships.ErrCannotMuteSelf:
			return BadRequest(c, "cannot unmute yourself")
		case relationships.ErrNotMuted:
			return Conflict(c, "not muted")
		default:
			return InternalError(c, err)
		}
	}

	return Success(c, rel)
}

// GetRelationships handles GET /api/v1/accounts/relationships
func (h *Relationship) GetRelationships(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)

	ids := c.Query("id[]")
	if ids == "" {
		return BadRequest(c, "id[] is required")
	}

	targetIDs := strings.Split(ids, ",")

	rels, err := h.relationships.GetRelationships(c.Request().Context(), accountID, targetIDs)
	if err != nil {
		return InternalError(c, err)
	}

	return Success(c, rels)
}

// GetPendingFollowers handles GET /api/v1/follow_requests
func (h *Relationship) GetPendingFollowers(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	limit := IntQuery(c, "limit", 40)
	offset := IntQuery(c, "offset", 0)

	follows, err := h.relationships.GetPendingFollowers(c.Request().Context(), accountID, limit, offset)
	if err != nil {
		return InternalError(c, err)
	}

	// Return follower IDs
	followerIDs := make([]string, len(follows))
	for i, f := range follows {
		followerIDs[i] = f.FollowerID
	}

	return Success(c, followerIDs)
}

// AcceptFollow handles POST /api/v1/follow_requests/:id/authorize
func (h *Relationship) AcceptFollow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	followerID := c.Param("id")

	if err := h.relationships.AcceptFollow(c.Request().Context(), accountID, followerID); err != nil {
		if err == relationships.ErrFollowRequestNotFound {
			return NotFound(c, "follow request")
		}
		return InternalError(c, err)
	}

	// Notify the follower
	if h.notifications != nil {
		_ = h.notifications.NotifyFollow(c.Request().Context(), followerID, accountID)
	}

	return NoContent(c)
}

// RejectFollow handles POST /api/v1/follow_requests/:id/reject
func (h *Relationship) RejectFollow(c *mizu.Ctx) error {
	accountID := h.getAccountID(c)
	followerID := c.Param("id")

	if err := h.relationships.RejectFollow(c.Request().Context(), accountID, followerID); err != nil {
		if err == relationships.ErrFollowRequestNotFound {
			return NotFound(c, "follow request")
		}
		return InternalError(c, err)
	}

	return NoContent(c)
}
