package wpapi

import (
	"github.com/go-mizu/mizu"
)

// Discovery handles GET /wp-json
func (h *Handler) Discovery(c *mizu.Ctx) error {
	// Get site title from settings
	siteTitle := "CMS"
	siteDescription := "A WordPress-compatible CMS"

	if h.settings != nil {
		if s, err := h.settings.Get(c.Context(), "site_title"); err == nil && s != nil {
			siteTitle = s.Value
		}
		if s, err := h.settings.Get(c.Context(), "site_description"); err == nil && s != nil {
			siteDescription = s.Value
		}
	}

	discovery := WPDiscovery{
		Name:           siteTitle,
		Description:    siteDescription,
		URL:            h.baseURL + "/wp-json",
		Home:           h.baseURL,
		GMTOffset:      0,
		TimezoneString: "UTC",
		Namespaces:     []string{"wp/v2"},
		Authentication: map[string]any{
			"cookie": map[string]any{
				"nonce_endpoint": h.baseURL + "/wp-json/wp/v2/users/me",
			},
		},
		Routes: h.buildRoutes(),
	}

	return OK(c, discovery)
}

// NamespaceDiscovery handles GET /wp-json/wp/v2
func (h *Handler) NamespaceDiscovery(c *mizu.Ctx) error {
	routes := h.buildRoutes()

	// Filter to just wp/v2 routes
	wpV2Routes := make(map[string]WPRoute)
	for path, route := range routes {
		if route.Namespace == "wp/v2" {
			wpV2Routes[path] = route
		}
	}

	return OK(c, map[string]any{
		"namespace": "wp/v2",
		"routes":    wpV2Routes,
	})
}

// buildRoutes builds the route definitions for API discovery.
func (h *Handler) buildRoutes() map[string]WPRoute {
	routes := make(map[string]WPRoute)

	// Posts
	routes["/wp/v2/posts"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{
				Methods: []string{"GET"},
				Args:    h.postListArgs(),
			},
			{
				Methods: []string{"POST"},
				Args:    h.postCreateArgs(),
			},
		},
	}

	routes["/wp/v2/posts/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{
				Methods: []string{"GET"},
				Args:    h.contextArg(),
			},
			{
				Methods: []string{"POST", "PUT", "PATCH"},
				Args:    h.postCreateArgs(),
			},
			{
				Methods: []string{"DELETE"},
				Args:    h.forceArg(),
			},
		},
	}

	// Pages
	routes["/wp/v2/pages"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.pageListArgs()},
			{Methods: []string{"POST"}, Args: h.pageCreateArgs()},
		},
	}

	routes["/wp/v2/pages/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.pageCreateArgs()},
			{Methods: []string{"DELETE"}, Args: h.forceArg()},
		},
	}

	// Users
	routes["/wp/v2/users"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.userListArgs()},
			{Methods: []string{"POST"}, Args: h.userCreateArgs()},
		},
	}

	routes["/wp/v2/users/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.userCreateArgs()},
			{Methods: []string{"DELETE"}, Args: h.userDeleteArgs()},
		},
	}

	routes["/wp/v2/users/me"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.userCreateArgs()},
			{Methods: []string{"DELETE"}, Args: h.userDeleteArgs()},
		},
	}

	// Categories
	routes["/wp/v2/categories"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.taxonomyListArgs()},
			{Methods: []string{"POST"}, Args: h.categoryCreateArgs()},
		},
	}

	routes["/wp/v2/categories/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.categoryCreateArgs()},
			{Methods: []string{"DELETE"}, Args: h.forceArg()},
		},
	}

	// Tags
	routes["/wp/v2/tags"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.taxonomyListArgs()},
			{Methods: []string{"POST"}, Args: h.tagCreateArgs()},
		},
	}

	routes["/wp/v2/tags/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.tagCreateArgs()},
			{Methods: []string{"DELETE"}, Args: h.forceArg()},
		},
	}

	// Media
	routes["/wp/v2/media"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.mediaListArgs()},
			{Methods: []string{"POST"}, Args: h.mediaCreateArgs()},
		},
	}

	routes["/wp/v2/media/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.mediaUpdateArgs()},
			{Methods: []string{"DELETE"}, Args: h.forceArg()},
		},
	}

	// Comments
	routes["/wp/v2/comments"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.commentListArgs()},
			{Methods: []string{"POST"}, Args: h.commentCreateArgs()},
		},
	}

	routes["/wp/v2/comments/(?P<id>[\\d]+)"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: h.contextArg()},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.commentCreateArgs()},
			{Methods: []string{"DELETE"}, Args: h.forceArg()},
		},
	}

	// Settings
	routes["/wp/v2/settings"] = WPRoute{
		Namespace: "wp/v2",
		Methods:   []string{"GET", "POST", "PUT", "PATCH"},
		Endpoints: []WPEndpoint{
			{Methods: []string{"GET"}, Args: map[string]WPArg{}},
			{Methods: []string{"POST", "PUT", "PATCH"}, Args: h.settingsArgs()},
		},
	}

	return routes
}

// Argument builders

func (h *Handler) contextArg() map[string]WPArg {
	return map[string]WPArg{
		"context": {
			Description: "Scope under which the request is made; determines fields present in response.",
			Type:        "string",
			Enum:        []string{"view", "embed", "edit"},
			Default:     "view",
		},
	}
}

func (h *Handler) forceArg() map[string]WPArg {
	return map[string]WPArg{
		"force": {
			Description: "Whether to bypass trash and force deletion.",
			Type:        "boolean",
			Default:     false,
		},
	}
}

func (h *Handler) paginationArgs() map[string]WPArg {
	return map[string]WPArg{
		"context": {
			Description: "Scope under which the request is made.",
			Type:        "string",
			Enum:        []string{"view", "embed", "edit"},
			Default:     "view",
		},
		"page": {
			Description: "Current page of the collection.",
			Type:        "integer",
			Default:     1,
		},
		"per_page": {
			Description: "Maximum number of items to be returned in result set.",
			Type:        "integer",
			Default:     10,
		},
		"search": {
			Description: "Limit results to those matching a string.",
			Type:        "string",
		},
		"exclude": {
			Description: "Ensure result set excludes specific IDs.",
			Type:        []string{"array", "integer"},
		},
		"include": {
			Description: "Limit result set to specific IDs.",
			Type:        []string{"array", "integer"},
		},
		"offset": {
			Description: "Offset the result set by a specific number of items.",
			Type:        "integer",
		},
		"order": {
			Description: "Order sort attribute ascending or descending.",
			Type:        "string",
			Enum:        []string{"asc", "desc"},
			Default:     "desc",
		},
		"orderby": {
			Description: "Sort collection by attribute.",
			Type:        "string",
			Default:     "date",
		},
	}
}

func (h *Handler) postListArgs() map[string]WPArg {
	args := h.paginationArgs()
	args["status"] = WPArg{
		Description: "Limit result set to posts assigned one or more statuses.",
		Type:        "string",
		Default:     "publish",
	}
	args["author"] = WPArg{
		Description: "Limit result set to posts assigned to specific authors.",
		Type:        []string{"array", "integer"},
	}
	args["categories"] = WPArg{
		Description: "Limit result set to all items that have the specified term assigned in the categories taxonomy.",
		Type:        []string{"array", "integer"},
	}
	args["tags"] = WPArg{
		Description: "Limit result set to all items that have the specified term assigned in the tags taxonomy.",
		Type:        []string{"array", "integer"},
	}
	args["sticky"] = WPArg{
		Description: "Limit result set to items that are sticky.",
		Type:        "boolean",
	}
	return args
}

func (h *Handler) postCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"title": {
			Description: "The title for the post.",
			Type:        "object",
		},
		"content": {
			Description: "The content for the post.",
			Type:        "object",
		},
		"excerpt": {
			Description: "The excerpt for the post.",
			Type:        "object",
		},
		"status": {
			Description: "A named status for the post.",
			Type:        "string",
			Enum:        []string{"publish", "future", "draft", "pending", "private"},
		},
		"author": {
			Description: "The ID for the author of the post.",
			Type:        "integer",
		},
		"featured_media": {
			Description: "The ID of the featured media for the post.",
			Type:        "integer",
		},
		"categories": {
			Description: "The terms assigned to the post in the category taxonomy.",
			Type:        "array",
		},
		"tags": {
			Description: "The terms assigned to the post in the post_tag taxonomy.",
			Type:        "array",
		},
		"sticky": {
			Description: "Whether or not the post should be treated as sticky.",
			Type:        "boolean",
		},
	}
}

func (h *Handler) pageListArgs() map[string]WPArg {
	args := h.paginationArgs()
	args["status"] = WPArg{
		Description: "Limit result set to pages assigned one or more statuses.",
		Type:        "string",
		Default:     "publish",
	}
	args["parent"] = WPArg{
		Description: "Limit result set to items with particular parent IDs.",
		Type:        []string{"array", "integer"},
	}
	args["author"] = WPArg{
		Description: "Limit result set to pages assigned to specific authors.",
		Type:        []string{"array", "integer"},
	}
	return args
}

func (h *Handler) pageCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"title": {
			Description: "The title for the page.",
			Type:        "object",
		},
		"content": {
			Description: "The content for the page.",
			Type:        "object",
		},
		"status": {
			Description: "A named status for the page.",
			Type:        "string",
			Enum:        []string{"publish", "future", "draft", "pending", "private"},
		},
		"parent": {
			Description: "The ID for the parent of the page.",
			Type:        "integer",
		},
		"menu_order": {
			Description: "The order of the page in relation to other pages.",
			Type:        "integer",
		},
		"template": {
			Description: "The theme file to use to display the page.",
			Type:        "string",
		},
	}
}

func (h *Handler) userListArgs() map[string]WPArg {
	args := h.paginationArgs()
	args["roles"] = WPArg{
		Description: "Limit result set to users matching at least one specific role provided.",
		Type:        "array",
	}
	args["slug"] = WPArg{
		Description: "Limit result set to users with one or more specific slugs.",
		Type:        "array",
	}
	return args
}

func (h *Handler) userCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"username": {
			Description: "Login name for the user.",
			Type:        "string",
			Required:    true,
		},
		"name": {
			Description: "Display name for the user.",
			Type:        "string",
		},
		"email": {
			Description: "The email address for the user.",
			Type:        "string",
			Required:    true,
		},
		"password": {
			Description: "Password for the user.",
			Type:        "string",
			Required:    true,
		},
		"roles": {
			Description: "Roles assigned to the user.",
			Type:        "array",
		},
		"description": {
			Description: "Description of the user.",
			Type:        "string",
		},
	}
}

func (h *Handler) userDeleteArgs() map[string]WPArg {
	return map[string]WPArg{
		"force": {
			Description: "Required to be true, as users do not support trashing.",
			Type:        "boolean",
			Required:    true,
		},
		"reassign": {
			Description: "Reassign the deleted user's posts and links to this user ID.",
			Type:        "integer",
			Required:    true,
		},
	}
}

func (h *Handler) taxonomyListArgs() map[string]WPArg {
	args := h.paginationArgs()
	args["hide_empty"] = WPArg{
		Description: "Whether to hide terms not assigned to any posts.",
		Type:        "boolean",
		Default:     false,
	}
	args["parent"] = WPArg{
		Description: "Limit result set to terms assigned to a specific parent.",
		Type:        "integer",
	}
	args["post"] = WPArg{
		Description: "Limit result set to terms assigned to a specific post.",
		Type:        "integer",
	}
	args["slug"] = WPArg{
		Description: "Limit result set to terms with one or more specific slugs.",
		Type:        "array",
	}
	return args
}

func (h *Handler) categoryCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"name": {
			Description: "HTML title for the term.",
			Type:        "string",
			Required:    true,
		},
		"slug": {
			Description: "An alphanumeric identifier for the term unique to its type.",
			Type:        "string",
		},
		"description": {
			Description: "HTML description of the term.",
			Type:        "string",
		},
		"parent": {
			Description: "The parent term ID.",
			Type:        "integer",
		},
	}
}

func (h *Handler) tagCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"name": {
			Description: "HTML title for the term.",
			Type:        "string",
			Required:    true,
		},
		"slug": {
			Description: "An alphanumeric identifier for the term unique to its type.",
			Type:        "string",
		},
		"description": {
			Description: "HTML description of the term.",
			Type:        "string",
		},
	}
}

func (h *Handler) mediaListArgs() map[string]WPArg {
	args := h.paginationArgs()
	args["author"] = WPArg{
		Description: "Limit result set to posts assigned to specific authors.",
		Type:        []string{"array", "integer"},
	}
	args["media_type"] = WPArg{
		Description: "Limit result set to attachments of a particular media type.",
		Type:        "string",
		Enum:        []string{"image", "video", "audio", "application"},
	}
	args["mime_type"] = WPArg{
		Description: "Limit result set to attachments of a particular MIME type.",
		Type:        "string",
	}
	return args
}

func (h *Handler) mediaCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"title": {
			Description: "The title for the media item.",
			Type:        "object",
		},
		"alt_text": {
			Description: "Alternative text to display when the media item is not displayed.",
			Type:        "string",
		},
		"caption": {
			Description: "The caption for the media item.",
			Type:        "object",
		},
		"description": {
			Description: "The description for the media item.",
			Type:        "object",
		},
	}
}

func (h *Handler) mediaUpdateArgs() map[string]WPArg {
	return map[string]WPArg{
		"title": {
			Description: "The title for the media item.",
			Type:        "object",
		},
		"alt_text": {
			Description: "Alternative text to display when the media item is not displayed.",
			Type:        "string",
		},
		"caption": {
			Description: "The caption for the media item.",
			Type:        "object",
		},
		"description": {
			Description: "The description for the media item.",
			Type:        "object",
		},
	}
}

func (h *Handler) commentListArgs() map[string]WPArg {
	args := h.paginationArgs()
	args["status"] = WPArg{
		Description: "Limit result set to comments assigned a specific status.",
		Type:        "string",
		Default:     "approve",
	}
	args["post"] = WPArg{
		Description: "Limit result set to comments assigned to a specific post ID.",
		Type:        "integer",
	}
	args["parent"] = WPArg{
		Description: "Limit result set to comments of specific parent IDs.",
		Type:        []string{"array", "integer"},
	}
	args["author"] = WPArg{
		Description: "Limit result set to comments assigned to specific user IDs.",
		Type:        []string{"array", "integer"},
	}
	return args
}

func (h *Handler) commentCreateArgs() map[string]WPArg {
	return map[string]WPArg{
		"post": {
			Description: "The ID of the associated post object.",
			Type:        "integer",
			Required:    true,
		},
		"content": {
			Description: "The content for the comment.",
			Type:        "object",
			Required:    true,
		},
		"parent": {
			Description: "The ID for the parent of the comment.",
			Type:        "integer",
		},
		"author_name": {
			Description: "Display name for the comment author.",
			Type:        "string",
		},
		"author_email": {
			Description: "Email address for the comment author.",
			Type:        "string",
		},
		"author_url": {
			Description: "URL for the comment author.",
			Type:        "string",
		},
	}
}

func (h *Handler) settingsArgs() map[string]WPArg {
	return map[string]WPArg{
		"title": {
			Description: "Site title.",
			Type:        "string",
		},
		"description": {
			Description: "Site tagline.",
			Type:        "string",
		},
		"timezone": {
			Description: "A city in the same timezone as you.",
			Type:        "string",
		},
		"date_format": {
			Description: "A date format for all date strings.",
			Type:        "string",
		},
		"time_format": {
			Description: "A time format for all time strings.",
			Type:        "string",
		},
		"posts_per_page": {
			Description: "Blog pages show at most.",
			Type:        "integer",
		},
	}
}
