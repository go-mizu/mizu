// Package graphql provides GraphQL query validation middleware for Mizu.
//
// The graphql middleware validates GraphQL queries to protect against malicious
// queries including depth attacks, complexity attacks, and unauthorized introspection.
// It provides configurable limits for query depth and complexity, introspection control,
// operation allowlisting, and field-level access control.
//
// # Features
//
//   - Query depth validation to prevent deeply nested queries
//   - Query complexity analysis to limit resource consumption
//   - Introspection control to hide schema details in production
//   - Operation allowlisting for fine-grained access control
//   - Field blocking to prevent access to sensitive data
//   - Custom error handling for better user experience
//
// # Quick Start
//
// Use default validation settings (max depth: 10, max complexity: 100):
//
//	app := mizu.New()
//	app.Use(graphql.New())
//	app.Post("/graphql", yourGraphQLHandler)
//
// # Configuration
//
// Create middleware with custom options:
//
//	app.Use(graphql.WithOptions(graphql.Options{
//	    MaxDepth:             5,
//	    MaxComplexity:        50,
//	    DisableIntrospection: true,
//	    AllowedOperations:    []string{"GetUser", "ListPosts"},
//	    BlockedFields:        []string{"password", "secret"},
//	}))
//
// # Production Setup
//
// Use production-ready configuration with introspection disabled:
//
//	app.Use(graphql.Production())
//
// This is equivalent to:
//
//	app.Use(graphql.WithOptions(graphql.Options{
//	    MaxDepth:             10,
//	    MaxComplexity:        100,
//	    DisableIntrospection: true,
//	}))
//
// # Security Features
//
// Disable introspection queries (__schema, __type):
//
//	app.Use(graphql.NoIntrospection())
//
// Block access to sensitive fields:
//
//	app.Use(graphql.BlockFields("password", "ssn", "apiKey"))
//
// Limit query depth to prevent nested query attacks:
//
//	app.Use(graphql.MaxDepth(5))
//
// Limit query complexity to prevent resource exhaustion:
//
//	app.Use(graphql.MaxComplexity(50))
//
// # Custom Error Handling
//
// Implement custom error responses:
//
//	app.Use(graphql.WithOptions(graphql.Options{
//	    MaxDepth: 10,
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        return c.JSON(http.StatusForbidden, map[string]any{
//	            "errors": []map[string]string{
//	                {"message": "Query validation failed", "detail": err.Error()},
//	            },
//	        })
//	    },
//	}))
//
// # How It Works
//
// The middleware validates GraphQL queries through the following process:
//
// 1. Request Filtering: Only processes POST requests with application/json content type
// 2. Query Parsing: Parses the request body into a Query structure
// 3. Validation Chain: Applies validations in order:
//   - Introspection check (if enabled)
//   - Operation allowlist check (if configured)
//   - Depth calculation and validation
//   - Complexity calculation and validation
//   - Blocked fields check (if configured)
//
// 4. Body Restoration: Restores the request body for downstream handlers
//
// # Depth Calculation
//
// Query depth is calculated by counting the maximum nesting level of braces:
//
//	{ users { posts { id } } }  // Depth: 3
//
// The middleware tracks the current depth and records the maximum depth reached.
//
// # Complexity Calculation
//
// Query complexity is estimated using:
//   - Count of field selections with braces
//   - Plus total count of opening braces
//
// This provides a simple heuristic for query cost estimation.
//
// # Introspection Detection
//
// Introspection queries are detected using regex pattern matching for:
//   - __schema - Schema introspection
//   - __type - Type introspection
//
// # Error Types
//
// The middleware returns the following validation errors:
//
//	ErrQueryTooDeep          - Query exceeds maximum depth
//	ErrQueryTooComplex       - Query exceeds maximum complexity
//	ErrIntrospectionDisabled - Introspection queries are disabled
//	ErrOperationNotAllowed   - Operation not in allowlist
//	ErrBlockedField          - Query contains blocked field
//
// # Security Considerations
//
// Query Depth Attacks: Deep nested queries can cause exponential computation.
// Set MaxDepth based on your schema's maximum legitimate nesting.
//
// Query Complexity Attacks: Complex queries can exhaust server resources.
// Use MaxComplexity to limit total query cost.
//
// Introspection Leaks: Introspection exposes your entire schema structure.
// Disable introspection in production with NoIntrospection() or Production().
//
// Sensitive Field Access: Prevent access to sensitive fields using BlockFields().
//
// # Best Practices
//
//   - Disable introspection in production environments
//   - Set appropriate depth limits based on your schema structure
//   - Use complexity limits to prevent resource exhaustion
//   - Block sensitive fields at the middleware level
//   - Implement custom error handlers for better user experience
//   - Combine with rate limiting middleware for additional protection
//
// # Related Middlewares
//
//   - ratelimit: Rate limiting to prevent abuse
//   - timeout: Request timeout to prevent long-running queries
//   - cors: Cross-origin resource sharing for web clients
package graphql
