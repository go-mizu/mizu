package collections

import (
	"github.com/go-mizu/blueprints/cms/config"
)

// Users is the default users collection with authentication.
var Users = config.CollectionConfig{
	Slug: "users",
	Labels: config.Labels{
		Singular: "User",
		Plural:   "Users",
	},
	Auth: &config.AuthConfig{
		TokenExpiration:  7200,  // 2 hours
		MaxLoginAttempts: 5,
		LockTime:         600,   // 10 minutes
		Verify:           false, // Email verification
		ForgotPassword:   true,
	},
	Fields: []config.Field{
		{
			Type:     config.FieldTypeEmail,
			Name:     "email",
			Label:    "Email",
			Required: true,
			Unique:   true,
			Index:    true,
		},
		{
			Type:  config.FieldTypeText,
			Name:  "firstName",
			Label: "First Name",
		},
		{
			Type:  config.FieldTypeText,
			Name:  "lastName",
			Label: "Last Name",
		},
		{
			Type:    config.FieldTypeSelect,
			Name:    "roles",
			Label:   "Roles",
			HasMany: true,
			Options: []config.SelectOption{
				{Label: "Admin", Value: "admin"},
				{Label: "Editor", Value: "editor"},
				{Label: "User", Value: "user"},
			},
			DefaultValue: []string{"user"},
		},
	},
	Timestamps: true,
	Admin: &config.CollectionAdmin{
		UseAsTitle:     "email",
		DefaultColumns: []string{"email", "firstName", "lastName", "roles", "createdAt"},
		ListSearchableFields: []string{"email", "firstName", "lastName"},
	},
}

// AdminOnly allows only admin users.
func AdminOnly(ctx *config.AccessContext) (*config.AccessResult, error) {
	if ctx.User == nil {
		return &config.AccessResult{Allowed: false}, nil
	}
	roles, ok := ctx.User["roles"].([]string)
	if !ok {
		return &config.AccessResult{Allowed: false}, nil
	}
	for _, role := range roles {
		if role == "admin" {
			return &config.AccessResult{Allowed: true}, nil
		}
	}
	return &config.AccessResult{Allowed: false}, nil
}

// SelfOrAdmin allows users to access their own data or admins to access any.
func SelfOrAdmin(ctx *config.AccessContext) (*config.AccessResult, error) {
	if ctx.User == nil {
		return &config.AccessResult{Allowed: false}, nil
	}

	// Admin can access all
	roles, _ := ctx.User["roles"].([]string)
	for _, role := range roles {
		if role == "admin" {
			return &config.AccessResult{Allowed: true}, nil
		}
	}

	// User can only access their own data
	userID, _ := ctx.User["id"].(string)
	if userID != "" && ctx.ID == userID {
		return &config.AccessResult{Allowed: true}, nil
	}

	// Return a filter that limits to own data
	return &config.AccessResult{
		Allowed: true,
		Where: map[string]any{
			"id": map[string]any{
				"equals": userID,
			},
		},
	}, nil
}
