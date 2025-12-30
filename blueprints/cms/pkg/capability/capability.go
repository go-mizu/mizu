// Package capability provides WordPress-compatible role and capability management.
package capability

// Role represents a WordPress user role.
type Role struct {
	Name         string
	DisplayName  string
	Capabilities map[string]bool
}

// Default roles with their capabilities.
var (
	// Administrator has full access.
	Administrator = Role{
		Name:        "administrator",
		DisplayName: "Administrator",
		Capabilities: map[string]bool{
			// Posts
			"edit_posts":             true,
			"edit_others_posts":      true,
			"edit_published_posts":   true,
			"publish_posts":          true,
			"delete_posts":           true,
			"delete_others_posts":    true,
			"delete_published_posts": true,
			"delete_private_posts":   true,
			"edit_private_posts":     true,
			"read_private_posts":     true,
			// Pages
			"edit_pages":             true,
			"edit_others_pages":      true,
			"edit_published_pages":   true,
			"publish_pages":          true,
			"delete_pages":           true,
			"delete_others_pages":    true,
			"delete_published_pages": true,
			"delete_private_pages":   true,
			"edit_private_pages":     true,
			"read_private_pages":     true,
			// Users
			"list_users":    true,
			"create_users":  true,
			"edit_users":    true,
			"delete_users":  true,
			"promote_users": true,
			// Themes
			"switch_themes":  true,
			"edit_themes":    true,
			"delete_themes":  true,
			"install_themes": true,
			"update_themes":  true,
			"edit_theme_options": true,
			// Plugins
			"activate_plugins": true,
			"edit_plugins":     true,
			"install_plugins":  true,
			"update_plugins":   true,
			"delete_plugins":   true,
			// Files
			"edit_files":   true,
			"upload_files": true,
			// Comments
			"moderate_comments": true,
			// Options
			"manage_options": true,
			// Categories
			"manage_categories": true,
			// Links
			"manage_links": true,
			// Import/Export
			"import": true,
			"export": true,
			// Updates
			"update_core": true,
			// Other
			"unfiltered_html":   true,
			"unfiltered_upload": true,
			"read":              true,
		},
	}

	// Editor can manage content but not settings.
	Editor = Role{
		Name:        "editor",
		DisplayName: "Editor",
		Capabilities: map[string]bool{
			// Posts
			"edit_posts":             true,
			"edit_others_posts":      true,
			"edit_published_posts":   true,
			"publish_posts":          true,
			"delete_posts":           true,
			"delete_others_posts":    true,
			"delete_published_posts": true,
			"delete_private_posts":   true,
			"edit_private_posts":     true,
			"read_private_posts":     true,
			// Pages
			"edit_pages":             true,
			"edit_others_pages":      true,
			"edit_published_pages":   true,
			"publish_pages":          true,
			"delete_pages":           true,
			"delete_others_pages":    true,
			"delete_published_pages": true,
			"delete_private_pages":   true,
			"edit_private_pages":     true,
			"read_private_pages":     true,
			// Files
			"upload_files": true,
			// Comments
			"moderate_comments": true,
			// Categories
			"manage_categories": true,
			// Links
			"manage_links": true,
			// Other
			"unfiltered_html": true,
			"read":            true,
		},
	}

	// Author can manage their own content.
	Author = Role{
		Name:        "author",
		DisplayName: "Author",
		Capabilities: map[string]bool{
			"edit_posts":             true,
			"edit_published_posts":   true,
			"publish_posts":          true,
			"delete_posts":           true,
			"delete_published_posts": true,
			"upload_files":           true,
			"read":                   true,
		},
	}

	// Contributor can write but not publish.
	Contributor = Role{
		Name:        "contributor",
		DisplayName: "Contributor",
		Capabilities: map[string]bool{
			"edit_posts":   true,
			"delete_posts": true,
			"read":         true,
		},
	}

	// Subscriber can only read.
	Subscriber = Role{
		Name:        "subscriber",
		DisplayName: "Subscriber",
		Capabilities: map[string]bool{
			"read": true,
		},
	}
)

// AllRoles returns all default roles.
func AllRoles() []Role {
	return []Role{Administrator, Editor, Author, Contributor, Subscriber}
}

// GetRole returns a role by name.
func GetRole(name string) *Role {
	switch name {
	case "administrator":
		return &Administrator
	case "editor":
		return &Editor
	case "author":
		return &Author
	case "contributor":
		return &Contributor
	case "subscriber":
		return &Subscriber
	default:
		return nil
	}
}

// HasCapability checks if a role has a specific capability.
func HasCapability(role, capability string) bool {
	r := GetRole(role)
	if r == nil {
		return false
	}
	return r.Capabilities[capability]
}

// CanEditPost checks if a user with the given role can edit a post.
func CanEditPost(role string, isOwnPost bool) bool {
	if HasCapability(role, "edit_others_posts") {
		return true
	}
	if isOwnPost && HasCapability(role, "edit_posts") {
		return true
	}
	return false
}

// CanDeletePost checks if a user with the given role can delete a post.
func CanDeletePost(role string, isOwnPost bool) bool {
	if HasCapability(role, "delete_others_posts") {
		return true
	}
	if isOwnPost && HasCapability(role, "delete_posts") {
		return true
	}
	return false
}

// CanPublish checks if a user with the given role can publish posts.
func CanPublish(role string) bool {
	return HasCapability(role, "publish_posts")
}

// CanModerateComments checks if a user with the given role can moderate comments.
func CanModerateComments(role string) bool {
	return HasCapability(role, "moderate_comments")
}

// CanManageOptions checks if a user with the given role can manage options.
func CanManageOptions(role string) bool {
	return HasCapability(role, "manage_options")
}

// CanManageUsers checks if a user with the given role can manage users.
func CanManageUsers(role string) bool {
	return HasCapability(role, "list_users") && HasCapability(role, "edit_users")
}
