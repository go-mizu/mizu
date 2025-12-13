// Package multitenancy provides multi-tenant middleware for Mizu applications.
//
// The multitenancy middleware extracts and provides tenant information for
// multi-tenant SaaS applications. It supports various resolution strategies
// including subdomain, header, path, and query parameters.
//
// # Quick Start
//
// Basic usage with subdomain resolution:
//
//	app := mizu.New()
//	app.Use(multitenancy.New(multitenancy.SubdomainResolver()))
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    tenant := multitenancy.Get(c)
//	    return c.JSON(200, map[string]string{
//	        "tenant_id": tenant.ID,
//	    })
//	})
//
// # Tenant Structure
//
// The Tenant struct contains tenant identification and metadata:
//
//	type Tenant struct {
//	    ID       string         // Unique tenant identifier
//	    Name     string         // Human-readable tenant name
//	    Domain   string         // Tenant domain
//	    Metadata map[string]any // Additional tenant data
//	}
//
// # Resolution Strategies
//
// The package provides multiple built-in resolvers:
//
// Subdomain Resolution:
//
//	// tenant1.example.com → tenant_id: "tenant1"
//	app.Use(multitenancy.New(multitenancy.SubdomainResolver()))
//
// Header Resolution:
//
//	// X-Tenant-ID: tenant1 → tenant_id: "tenant1"
//	app.Use(multitenancy.New(multitenancy.HeaderResolver("X-Tenant-ID")))
//
// Path Resolution:
//
//	// /tenant1/users → tenant_id: "tenant1", path: /users
//	app.Use(multitenancy.New(multitenancy.PathResolver()))
//
// Query Parameter Resolution:
//
//	// /users?tenant=tenant1 → tenant_id: "tenant1"
//	app.Use(multitenancy.New(multitenancy.QueryResolver("tenant")))
//
// # Advanced Usage
//
// Database Lookup:
//
//	resolver := multitenancy.LookupResolver(
//	    multitenancy.SubdomainResolver(),
//	    func(id string) (*multitenancy.Tenant, error) {
//	        var tenant Tenant
//	        err := db.QueryRow(
//	            "SELECT id, name, plan FROM tenants WHERE slug = ?",
//	            id,
//	        ).Scan(&tenant.ID, &tenant.Name, &tenant.Plan)
//
//	        if err != nil {
//	            return nil, multitenancy.ErrTenantNotFound
//	        }
//
//	        return &multitenancy.Tenant{
//	            ID:   tenant.ID,
//	            Name: tenant.Name,
//	            Metadata: map[string]any{
//	                "plan": tenant.Plan,
//	            },
//	        }, nil
//	    },
//	)
//
//	app.Use(multitenancy.New(resolver))
//
// Chain Multiple Resolvers:
//
//	// Try subdomain first, then header, then query param
//	resolver := multitenancy.ChainResolver(
//	    multitenancy.SubdomainResolver(),
//	    multitenancy.HeaderResolver("X-Tenant-ID"),
//	    multitenancy.QueryResolver("tenant"),
//	)
//
//	app.Use(multitenancy.New(resolver))
//
// Custom Error Handler:
//
//	app.Use(multitenancy.WithOptions(multitenancy.Options{
//	    Resolver: multitenancy.SubdomainResolver(),
//	    ErrorHandler: func(c *mizu.Ctx, err error) error {
//	        return c.JSON(400, map[string]string{
//	            "error":   "Invalid tenant",
//	            "message": err.Error(),
//	        })
//	    },
//	}))
//
// Optional Tenant:
//
//	app.Use(multitenancy.WithOptions(multitenancy.Options{
//	    Resolver: multitenancy.SubdomainResolver(),
//	    Required: false, // Allow requests without tenant
//	}))
//
//	app.Get("/", func(c *mizu.Ctx) error {
//	    tenant := multitenancy.Get(c)
//	    if tenant == nil {
//	        return c.JSON(200, "Welcome to the platform")
//	    }
//	    return c.JSON(200, "Welcome, "+tenant.Name)
//	})
//
// # Retrieving Tenant Information
//
// The package provides three functions to retrieve tenant from context:
//
// Get returns the tenant or nil if not found:
//
//	tenant := multitenancy.Get(c)
//	if tenant != nil {
//	    // Use tenant
//	}
//
// FromContext is an alias for Get:
//
//	tenant := multitenancy.FromContext(c)
//
// MustGet returns the tenant or panics if not found:
//
//	tenant := multitenancy.MustGet(c)
//	// Use tenant (guaranteed to be non-nil)
//
// # Custom Resolvers
//
// You can implement custom resolvers by implementing the Resolver function type:
//
//	func JWTTenantResolver() multitenancy.Resolver {
//	    return func(c *mizu.Ctx) (*multitenancy.Tenant, error) {
//	        claims := c.Get("jwt_claims").(jwt.MapClaims)
//	        tenantID, ok := claims["tenant_id"].(string)
//	        if !ok || tenantID == "" {
//	            return nil, multitenancy.ErrTenantNotFound
//	        }
//
//	        return &multitenancy.Tenant{
//	            ID:   tenantID,
//	            Name: tenantID,
//	        }, nil
//	    }
//	}
//
//	app.Use(jwtauth.New(jwtSecret))
//	app.Use(multitenancy.New(JWTTenantResolver()))
//
// # Error Handling
//
// The package defines two standard errors:
//
//   - ErrTenantNotFound: Returned when tenant cannot be resolved
//   - ErrTenantInvalid: Returned when tenant data is invalid
//
// The middleware supports both required and optional tenant resolution:
//
//   - Required mode (default): Returns error via ErrorHandler when tenant not found
//   - Optional mode: Continues request processing with nil tenant
//   - Custom error handler: Allows application-specific error responses
//
// # Implementation Details
//
// The middleware uses Go's context package to store tenant information using
// a private contextKey{} type. This ensures type safety and prevents key
// collisions with other middleware or application code.
//
// The PathResolver automatically rewrites the request path to remove the
// tenant prefix, allowing routes to be defined without tenant prefixes while
// maintaining tenant isolation.
//
// # Performance Considerations
//
//   - Context lookups are O(1) operations
//   - Subdomain parsing uses efficient string operations
//   - Header lookups use Go's optimized HTTP header map
//   - Consider caching with LookupResolver for database-backed tenants
//
// # Best Practices
//
//   - Use subdomain resolution for user-friendly URLs
//   - Implement database lookup for rich tenant data
//   - Use chain resolver for flexibility
//   - Always validate tenant access to resources
//   - Include tenant ID in all database queries
//   - Cache tenant lookups for performance
package multitenancy
