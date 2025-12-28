package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/users"
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
func (h *UserHandler) GetAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}
	WriteJSON(w, http.StatusOK, user)
}

// UpdateAuthenticatedUser handles PATCH /user
func (h *UserHandler) UpdateAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	var in users.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.users.Update(r.Context(), user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// ListUsers handles GET /users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	pagination := GetPaginationParams(r)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	if since := QueryParam(r, "since"); since != "" {
		if n, err := PathParamInt64(r, "since"); err == nil {
			opts.Since = n
		}
	}

	userList, err := h.users.List(r.Context(), opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, userList)
}

// GetUser handles GET /users/{username}
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	if username == "" {
		WriteBadRequest(w, "username is required")
		return
	}

	user, err := h.users.GetByLogin(r.Context(), username)
	if err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

// ListFollowers handles GET /users/{username}/followers
func (h *UserHandler) ListFollowers(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	pagination := GetPaginationParams(r)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	followers, err := h.users.ListFollowers(r.Context(), username, opts)
	if err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, followers)
}

// ListFollowing handles GET /users/{username}/following
func (h *UserHandler) ListFollowing(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	pagination := GetPaginationParams(r)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	following, err := h.users.ListFollowing(r.Context(), username, opts)
	if err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, following)
}

// CheckFollowing handles GET /users/{username}/following/{target_user}
func (h *UserHandler) CheckFollowing(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	target := PathParam(r, "target_user")

	isFollowing, err := h.users.IsFollowing(r.Context(), username, target)
	if err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isFollowing {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Follow")
	}
}

// ListAuthenticatedUserFollowers handles GET /user/followers
func (h *UserHandler) ListAuthenticatedUserFollowers(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	followers, err := h.users.ListFollowers(r.Context(), user.Login, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, followers)
}

// ListAuthenticatedUserFollowing handles GET /user/following
func (h *UserHandler) ListAuthenticatedUserFollowing(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &users.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	following, err := h.users.ListFollowing(r.Context(), user.Login, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, following)
}

// CheckAuthenticatedUserFollowing handles GET /user/following/{username}
func (h *UserHandler) CheckAuthenticatedUserFollowing(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	target := PathParam(r, "username")
	isFollowing, err := h.users.IsFollowing(r.Context(), user.Login, target)
	if err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isFollowing {
		WriteNoContent(w)
	} else {
		WriteNotFound(w, "Follow")
	}
}

// FollowUser handles PUT /user/following/{username}
func (h *UserHandler) FollowUser(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	target := PathParam(r, "username")
	if err := h.users.Follow(r.Context(), user.ID, target); err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// UnfollowUser handles DELETE /user/following/{username}
func (h *UserHandler) UnfollowUser(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	target := PathParam(r, "username")
	if err := h.users.Unfollow(r.Context(), user.ID, target); err != nil {
		if err == users.ErrNotFound {
			WriteNotFound(w, "User")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
