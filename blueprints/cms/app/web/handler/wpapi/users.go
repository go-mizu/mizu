package wpapi

import (
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/feature/users"
)

// ListUsers handles GET /wp/v2/users
func (h *Handler) ListUsers(c *mizu.Ctx) error {
	params := ParseListParams(c)

	in := &users.ListIn{
		Search: params.Search,
		Limit:  params.PerPage,
		Offset: params.Offset,
	}

	if in.Offset == 0 && params.Page > 1 {
		in.Offset = (params.Page - 1) * params.PerPage
	}

	// Role filter
	if roles := c.Query("roles"); roles != "" {
		// Take first role
		parts := strings.Split(roles, ",")
		if len(parts) > 0 {
			in.Role = parts[0]
		}
	}

	// Slug filter
	if slug := c.Query("slug"); slug != "" {
		user, err := h.users.GetBySlug(c.Context(), slug)
		if err != nil {
			return OKList(c, []WPUser{}, 0, params.Page, params.PerPage)
		}
		wpUser := h.userToWP(user, params.Context)
		return OKList(c, []WPUser{wpUser}, 1, params.Page, params.PerPage)
	}

	list, total, err := h.users.List(c.Context(), in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_read", "Could not read users")
	}

	wpUsers := make([]WPUser, len(list))
	for i, user := range list {
		wpUsers[i] = h.userToWP(user, params.Context)
	}

	return OKList(c, wpUsers, total, params.Page, params.PerPage)
}

// CreateUser handles POST /wp/v2/users
func (h *Handler) CreateUser(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	var req WPCreateUserRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	if req.Username == "" {
		return ErrorInvalidParam(c, "username", "Username is required")
	}
	if req.Email == "" {
		return ErrorInvalidParam(c, "email", "Email is required")
	}
	if req.Password == "" {
		return ErrorInvalidParam(c, "password", "Password is required")
	}

	in := &users.RegisterIn{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	}
	if in.Name == "" {
		in.Name = req.Username
	}

	user, _, err := h.users.Register(c.Context(), in)
	if err != nil {
		if err == users.ErrUserExists {
			return ErrorBadRequest(c, "rest_user_exists", "User already exists")
		}
		if err == users.ErrInvalidEmail {
			return ErrorInvalidParam(c, "email", "Invalid email address")
		}
		return ErrorInternal(c, "rest_cannot_create", "Could not create user")
	}

	// Update with additional fields if provided
	updateIn := &users.UpdateIn{}
	needsUpdate := false

	if req.Description != "" {
		updateIn.Bio = &req.Description
		needsUpdate = true
	}
	if req.URL != "" {
		updateIn.AvatarURL = &req.URL
		needsUpdate = true
	}
	if len(req.Roles) > 0 {
		updateIn.Role = &req.Roles[0]
		needsUpdate = true
	}

	if needsUpdate {
		user, _ = h.users.Update(c.Context(), user.ID, updateIn)
	}

	return Created(c, h.userToWP(user, ContextEdit))
}

// GetUser handles GET /wp/v2/users/{id}
func (h *Handler) GetUser(c *mizu.Ctx) error {
	id := ParseID(c)
	context := c.Query("context")
	if context == "" {
		context = ContextView
	}

	user, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		if err == users.ErrNotFound {
			return ErrorNotFound(c, "rest_user_invalid_id", "Invalid user ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read user")
	}

	return OK(c, h.userToWP(user, context))
}

// GetCurrentUser handles GET /wp/v2/users/me
func (h *Handler) GetCurrentUser(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	context := c.Query("context")
	if context == "" {
		context = ContextEdit
	}

	user := h.getUser(c)
	if user == nil {
		return ErrorUnauthorized(c)
	}

	return OK(c, h.userToWP(user, context))
}

// UpdateUser handles POST/PUT/PATCH /wp/v2/users/{id}
func (h *Handler) UpdateUser(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	var req WPCreateUserRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &users.UpdateIn{}

	if req.Name != "" {
		in.Name = &req.Name
	}

	if req.Description != "" {
		in.Bio = &req.Description
	}

	if req.URL != "" {
		in.AvatarURL = &req.URL
	}

	if len(req.Roles) > 0 {
		in.Role = &req.Roles[0]
	}

	user, err := h.users.Update(c.Context(), id, in)
	if err != nil {
		if err == users.ErrNotFound {
			return ErrorNotFound(c, "rest_user_invalid_id", "Invalid user ID.")
		}
		return ErrorInternal(c, "rest_cannot_update", "Could not update user")
	}

	return OK(c, h.userToWP(user, ContextEdit))
}

// UpdateCurrentUser handles POST/PUT/PATCH /wp/v2/users/me
func (h *Handler) UpdateCurrentUser(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	user := h.getUser(c)
	if user == nil {
		return ErrorUnauthorized(c)
	}

	var req WPCreateUserRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return ErrorBadRequest(c, "rest_invalid_json", "Invalid JSON body")
	}

	in := &users.UpdateIn{}

	if req.Name != "" {
		in.Name = &req.Name
	}

	if req.Description != "" {
		in.Bio = &req.Description
	}

	if req.URL != "" {
		in.AvatarURL = &req.URL
	}

	updatedUser, err := h.users.Update(c.Context(), user.ID, in)
	if err != nil {
		return ErrorInternal(c, "rest_cannot_update", "Could not update user")
	}

	return OK(c, h.userToWP(updatedUser, ContextEdit))
}

// DeleteUser handles DELETE /wp/v2/users/{id}
func (h *Handler) DeleteUser(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	id := ParseID(c)

	// WordPress requires force=true and reassign parameter
	force := c.Query("force") == "true"
	if !force {
		return ErrorBadRequest(c, "rest_trash_not_supported", "Users do not support trashing. Set force=true to delete.")
	}

	user, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		if err == users.ErrNotFound {
			return ErrorNotFound(c, "rest_user_invalid_id", "Invalid user ID.")
		}
		return ErrorInternal(c, "rest_cannot_read", "Could not read user")
	}

	if err := h.users.Delete(c.Context(), id); err != nil {
		return ErrorInternal(c, "rest_cannot_delete", "Could not delete user")
	}

	wpUser := h.userToWP(user, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpUser,
	})
}

// DeleteCurrentUser handles DELETE /wp/v2/users/me
func (h *Handler) DeleteCurrentUser(c *mizu.Ctx) error {
	if err := h.RequireAuth(c); err != nil {
		return err
	}

	user := h.getUser(c)
	if user == nil {
		return ErrorUnauthorized(c)
	}

	force := c.Query("force") == "true"
	if !force {
		return ErrorBadRequest(c, "rest_trash_not_supported", "Users do not support trashing. Set force=true to delete.")
	}

	if err := h.users.Delete(c.Context(), user.ID); err != nil {
		return ErrorInternal(c, "rest_cannot_delete", "Could not delete user")
	}

	wpUser := h.userToWP(user, ContextEdit)
	return OK(c, map[string]any{
		"deleted":  true,
		"previous": wpUser,
	})
}

// userToWP converts an internal user to WordPress format.
func (h *Handler) userToWP(u *users.User, context string) WPUser {
	numericID := ULIDToNumericID(u.ID)

	// Parse name into first/last
	firstName := ""
	lastName := ""
	nameParts := strings.SplitN(u.Name, " ", 2)
	if len(nameParts) > 0 {
		firstName = nameParts[0]
	}
	if len(nameParts) > 1 {
		lastName = nameParts[1]
	}

	wp := WPUser{
		ID:          numericID,
		Name:        u.Name,
		URL:         "",
		Description: u.Bio,
		Link:        h.UserURL(u.Slug),
		Slug:        u.Slug,
		AvatarURLs:  AvatarURLs(u.Email),
		Meta:        []any{},
	}

	// Edit context includes more fields
	if context == ContextEdit {
		wp.Username = u.Email // Use email as username
		wp.Email = u.Email
		wp.FirstName = firstName
		wp.LastName = lastName
		wp.Nickname = u.Name
		wp.Locale = "en_US"
		wp.RegisteredDate = FormatWPDateTime(u.CreatedAt)
		wp.Roles = []string{u.Role}
		wp.Capabilities = roleCapabilities(u.Role)
		wp.ExtraCapabilities = map[string]bool{u.Role: true}
	}

	// Avatar URL
	if u.AvatarURL != "" {
		wp.AvatarURLs["96"] = u.AvatarURL
		wp.AvatarURLs["48"] = u.AvatarURL
		wp.AvatarURLs["24"] = u.AvatarURL
	}

	wp.Links = map[string][]WPLink{
		"self":       {h.SelfLink("/users/" + strconv.FormatInt(numericID, 10))},
		"collection": {h.CollectionLink("/users")},
	}

	return wp
}

// roleCapabilities returns capabilities for a role.
func roleCapabilities(role string) map[string]bool {
	caps := map[string]bool{
		"read": true,
	}

	switch role {
	case "admin", "administrator":
		caps["manage_options"] = true
		caps["edit_posts"] = true
		caps["edit_others_posts"] = true
		caps["edit_pages"] = true
		caps["edit_others_pages"] = true
		caps["publish_posts"] = true
		caps["publish_pages"] = true
		caps["delete_posts"] = true
		caps["delete_others_posts"] = true
		caps["delete_pages"] = true
		caps["delete_others_pages"] = true
		caps["upload_files"] = true
		caps["edit_users"] = true
		caps["create_users"] = true
		caps["delete_users"] = true
		caps["moderate_comments"] = true
	case "editor":
		caps["edit_posts"] = true
		caps["edit_others_posts"] = true
		caps["edit_pages"] = true
		caps["edit_others_pages"] = true
		caps["publish_posts"] = true
		caps["publish_pages"] = true
		caps["delete_posts"] = true
		caps["delete_others_posts"] = true
		caps["delete_pages"] = true
		caps["delete_others_pages"] = true
		caps["upload_files"] = true
		caps["moderate_comments"] = true
	case "author":
		caps["edit_posts"] = true
		caps["publish_posts"] = true
		caps["delete_posts"] = true
		caps["upload_files"] = true
	case "contributor":
		caps["edit_posts"] = true
		caps["delete_posts"] = true
	}

	return caps
}
