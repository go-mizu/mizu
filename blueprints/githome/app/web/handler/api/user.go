package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// UserHandler handles user endpoints
type UserHandler struct {
	users users.API
}

// NewUserHandler creates a new user handler
func NewUserHandler(users users.API) *UserHandler {
	return &UserHandler{users: users}
}

// GetAuthenticatedUser handles GET /user
func (h *UserHandler) GetAuthenticatedUser(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}
	return c.JSON(http.StatusOK, user)
}

// UpdateAuthenticatedUser handles PATCH /user
func (h *UserHandler) UpdateAuthenticatedUser(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	var in users.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.users.Update(c.Context(), user.ID, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// ListUsers handles GET /users
func (h *UserHandler) ListUsers(c *mizu.Ctx) error {
	pagination := GetPagination(c)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	if since := c.Query("since"); since != "" {
		if n, err := strconv.ParseInt(since, 10, 64); err == nil {
			opts.Since = n
		}
	}

	userList, err := h.users.List(c.Context(), opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, userList)
}

// GetUser handles GET /users/{username}
func (h *UserHandler) GetUser(c *mizu.Ctx) error {
	username := c.Param("username")
	if username == "" {
		return BadRequest(c, "username is required")
	}

	user, err := h.users.GetByLogin(c.Context(), username)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, user)
}

// ListFollowers handles GET /users/{username}/followers
func (h *UserHandler) ListFollowers(c *mizu.Ctx) error {
	username := c.Param("username")
	pagination := GetPagination(c)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	followers, err := h.users.ListFollowers(c.Context(), username, opts)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, followers)
}

// ListFollowing handles GET /users/{username}/following
func (h *UserHandler) ListFollowing(c *mizu.Ctx) error {
	username := c.Param("username")
	pagination := GetPagination(c)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	following, err := h.users.ListFollowing(c.Context(), username, opts)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, following)
}

// CheckFollowing handles GET /users/{username}/following/{target_user}
func (h *UserHandler) CheckFollowing(c *mizu.Ctx) error {
	username := c.Param("username")
	target := c.Param("target_user")

	isFollowing, err := h.users.IsFollowing(c.Context(), username, target)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isFollowing {
		return NoContent(c)
	}
	return NotFound(c, "Follow")
}

// ListAuthenticatedUserFollowers handles GET /user/followers
func (h *UserHandler) ListAuthenticatedUserFollowers(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	followers, err := h.users.ListFollowers(c.Context(), user.Login, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, followers)
}

// ListAuthenticatedUserFollowing handles GET /user/following
func (h *UserHandler) ListAuthenticatedUserFollowing(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	following, err := h.users.ListFollowing(c.Context(), user.Login, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, following)
}

// CheckAuthenticatedUserFollowing handles GET /user/following/{username}
func (h *UserHandler) CheckAuthenticatedUserFollowing(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	target := c.Param("username")
	isFollowing, err := h.users.IsFollowing(c.Context(), user.Login, target)
	if err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if isFollowing {
		return NoContent(c)
	}
	return NotFound(c, "Follow")
}

// FollowUser handles PUT /user/following/{username}
func (h *UserHandler) FollowUser(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	target := c.Param("username")
	if err := h.users.Follow(c.Context(), user.ID, target); err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// UnfollowUser handles DELETE /user/following/{username}
func (h *UserHandler) UnfollowUser(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	target := c.Param("username")
	if err := h.users.Unfollow(c.Context(), user.ID, target); err != nil {
		if err == users.ErrNotFound {
			return NotFound(c, "User")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}
