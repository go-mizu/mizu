// Package filter provides request filtering middleware for the Mizu web framework.
//
// The filter middleware enables conditional request processing based on various criteria
// including HTTP methods, URL paths, hostnames, user agents, and custom filter functions.
// It supports both allow-list and block-list approaches with flexible glob pattern matching.
//
// # Basic Usage
//
// Create a filter with default options (allows all requests):
//
//	app := mizu.New()
//	app.Use(filter.New())
//
// # Method Filtering
//
// Restrict requests to specific HTTP methods:
//
//	app.Use(filter.Methods("GET", "POST"))
//
// This will only allow GET and POST requests. All other methods will receive a 403 Forbidden response.
//
// # Path Filtering
//
// Allow only specific paths using glob patterns:
//
//	app.Use(filter.Paths("/api/*", "/public/*"))
//
// Block specific paths:
//
//	app.Use(filter.BlockPaths("/admin/*", "/internal/**"))
//
// Pattern syntax:
//   - * matches any characters except path separators (e.g., /api/* matches /api/users)
//   - ** matches any characters including path separators (e.g., /api/** matches /api/users/1/details)
//   - ? matches exactly one character
//
// # Host Filtering
//
// Restrict access to specific hostnames:
//
//	app.Use(filter.Hosts("example.com", "api.example.com"))
//
// Block specific hosts:
//
//	app.Use(filter.WithOptions(filter.Options{
//	    BlockedHosts: []string{"spam.com", "malware.net"},
//	}))
//
// Host matching is case-insensitive and automatically strips port numbers.
//
// # User Agent Filtering
//
// Block specific user agents (useful for blocking bots or scrapers):
//
//	app.Use(filter.BlockUserAgents("curl/**", "**bot**"))
//
// # Custom Filters
//
// Implement custom filtering logic:
//
//	app.Use(filter.Custom(func(c *mizu.Ctx) bool {
//	    apiKey := c.Request().Header.Get("X-API-Key")
//	    return apiKey != "" && isValidAPIKey(apiKey)
//	}))
//
// The function should return true to allow the request, false to block it.
//
// # Advanced Configuration
//
// Combine multiple filter criteria and customize the block response:
//
//	app.Use(filter.WithOptions(filter.Options{
//	    AllowedMethods:    []string{"GET", "POST"},
//	    AllowedPaths:      []string{"/api/*"},
//	    BlockedUserAgents: []string{"**bot**"},
//	    CustomFilter: func(c *mizu.Ctx) bool {
//	        return c.Request().Header.Get("X-API-Key") != ""
//	    },
//	    OnBlock: func(c *mizu.Ctx) error {
//	        return c.JSON(403, map[string]string{
//	            "error": "Access denied",
//	        })
//	    },
//	}))
//
// # Filter Evaluation Order
//
// Filters are evaluated in the following order:
//  1. HTTP Methods
//  2. Blocked Hosts (takes precedence over allowed hosts)
//  3. Allowed Hosts
//  4. Blocked Paths (takes precedence over allowed paths)
//  5. Allowed Paths
//  6. Blocked User Agents (takes precedence over allowed user agents)
//  7. Allowed User Agents
//  8. Custom Filter
//
// If any check fails, the request is immediately blocked and the OnBlock handler is invoked.
//
// # Performance Considerations
//
// The middleware is optimized for performance:
//   - HTTP methods and hosts are stored in maps for O(1) lookup
//   - Glob patterns are compiled to regular expressions once during initialization
//   - No runtime compilation or parsing overhead
//   - Short-circuit evaluation stops at the first failing check
//
// # Error Handling
//
// By default, blocked requests receive a 403 Forbidden response with "Forbidden" as the body.
// Customize this behavior using the OnBlock option:
//
//	app.Use(filter.WithOptions(filter.Options{
//	    AllowedMethods: []string{"GET"},
//	    OnBlock: func(c *mizu.Ctx) error {
//	        return c.JSON(http.StatusMethodNotAllowed, map[string]string{
//	            "error": "Method not allowed",
//	            "allowed": "GET",
//	        })
//	    },
//	}))
//
// # Examples
//
// API endpoint protection:
//
//	// Only allow authenticated requests to API routes
//	apiGroup := app.Group("/api")
//	apiGroup.Use(filter.Custom(func(c *mizu.Ctx) bool {
//	    token := c.Request().Header.Get("Authorization")
//	    return validateToken(token)
//	}))
//
// Admin panel protection:
//
//	// Block all access to admin routes from non-local hosts
//	app.Use(filter.WithOptions(filter.Options{
//	    BlockedPaths: []string{"/admin/**"},
//	    CustomFilter: func(c *mizu.Ctx) bool {
//	        if strings.HasPrefix(c.Request().URL.Path, "/admin") {
//	            return isLocalRequest(c)
//	        }
//	        return true
//	    },
//	}))
//
// Bot protection:
//
//	// Block known bots and crawlers
//	app.Use(filter.BlockUserAgents(
//	    "**bot**",
//	    "**crawler**",
//	    "**spider**",
//	    "curl/**",
//	    "wget/**",
//	))
package filter
