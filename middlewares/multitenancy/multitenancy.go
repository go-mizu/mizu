// Package multitenancy provides multi-tenant middleware for Mizu.
package multitenancy

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Tenant represents a tenant.
type Tenant struct {
	ID       string
	Name     string
	Domain   string
	Metadata map[string]any
}

// Resolver resolves tenant from request.
type Resolver func(c *mizu.Ctx) (*Tenant, error)

// Options configures the multitenancy middleware.
type Options struct {
	// Resolver resolves tenant from request.
	// Default: uses subdomain resolution.
	Resolver Resolver

	// ErrorHandler handles resolution errors.
	ErrorHandler func(c *mizu.Ctx, err error) error

	// Required requires tenant to be resolved.
	// Default: true.
	Required bool
}

// New creates multitenancy middleware with resolver.
func New(resolver Resolver) mizu.Middleware {
	return WithOptions(Options{Resolver: resolver})
}

// WithOptions creates multitenancy middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Resolver == nil {
		opts.Resolver = SubdomainResolver()
	}
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = defaultErrorHandler
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			tenant, err := opts.Resolver(c)
			if err != nil {
				if opts.Required {
					return opts.ErrorHandler(c, err)
				}
				// Continue without tenant
				return next(c)
			}

			if tenant == nil && opts.Required {
				return opts.ErrorHandler(c, ErrTenantNotFound)
			}

			if tenant != nil {
				// Store tenant in context
				ctx := context.WithValue(c.Context(), contextKey{}, tenant)
				req := c.Request().WithContext(ctx)
				*c.Request() = *req
			}

			return next(c)
		}
	}
}

// Get retrieves tenant from context.
func Get(c *mizu.Ctx) *Tenant {
	if tenant, ok := c.Context().Value(contextKey{}).(*Tenant); ok {
		return tenant
	}
	return nil
}

// FromContext is an alias for Get.
func FromContext(c *mizu.Ctx) *Tenant {
	return Get(c)
}

// MustGet retrieves tenant from context or panics.
func MustGet(c *mizu.Ctx) *Tenant {
	tenant := Get(c)
	if tenant == nil {
		panic("multitenancy: tenant not found in context")
	}
	return tenant
}

func defaultErrorHandler(c *mizu.Ctx, err error) error {
	return c.Text(http.StatusBadRequest, err.Error())
}

// Common resolvers

// SubdomainResolver resolves tenant from subdomain.
func SubdomainResolver() Resolver {
	return func(c *mizu.Ctx) (*Tenant, error) {
		host := c.Request().Host

		// Remove port if present
		if idx := strings.LastIndex(host, ":"); idx > 0 {
			host = host[:idx]
		}

		parts := strings.Split(host, ".")
		if len(parts) < 2 {
			return nil, ErrTenantNotFound
		}

		subdomain := parts[0]
		if subdomain == "" || subdomain == "www" {
			return nil, ErrTenantNotFound
		}

		return &Tenant{
			ID:     subdomain,
			Name:   subdomain,
			Domain: host,
		}, nil
	}
}

// HeaderResolver resolves tenant from header.
func HeaderResolver(header string) Resolver {
	return func(c *mizu.Ctx) (*Tenant, error) {
		tenantID := c.Request().Header.Get(header)
		if tenantID == "" {
			return nil, ErrTenantNotFound
		}

		return &Tenant{
			ID:   tenantID,
			Name: tenantID,
		}, nil
	}
}

// PathResolver resolves tenant from URL path prefix.
func PathResolver() Resolver {
	return func(c *mizu.Ctx) (*Tenant, error) {
		path := c.Request().URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")

		if len(parts) < 1 || parts[0] == "" {
			return nil, ErrTenantNotFound
		}

		tenantID := parts[0]

		// Rewrite path without tenant prefix
		newPath := "/" + strings.Join(parts[1:], "/")
		if newPath == "/" && len(parts) > 1 {
			newPath = "/"
		}
		c.Request().URL.Path = newPath

		return &Tenant{
			ID:   tenantID,
			Name: tenantID,
		}, nil
	}
}

// QueryResolver resolves tenant from query parameter.
func QueryResolver(param string) Resolver {
	return func(c *mizu.Ctx) (*Tenant, error) {
		tenantID := c.Query(param)
		if tenantID == "" {
			return nil, ErrTenantNotFound
		}

		return &Tenant{
			ID:   tenantID,
			Name: tenantID,
		}, nil
	}
}

// LookupResolver wraps a resolver with a lookup function.
func LookupResolver(resolver Resolver, lookup func(id string) (*Tenant, error)) Resolver {
	return func(c *mizu.Ctx) (*Tenant, error) {
		tenant, err := resolver(c)
		if err != nil {
			return nil, err
		}

		return lookup(tenant.ID)
	}
}

// ChainResolver tries multiple resolvers in order.
func ChainResolver(resolvers ...Resolver) Resolver {
	return func(c *mizu.Ctx) (*Tenant, error) {
		for _, resolver := range resolvers {
			tenant, err := resolver(c)
			if err == nil && tenant != nil {
				return tenant, nil
			}
		}
		return nil, ErrTenantNotFound
	}
}

// Errors
var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrTenantInvalid  = errors.New("tenant invalid")
)
