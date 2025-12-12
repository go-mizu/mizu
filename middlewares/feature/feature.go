// Package feature provides feature flag middleware for Mizu.
package feature

import (
	"context"
	"sync"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Flag represents a feature flag.
type Flag struct {
	Name        string
	Enabled     bool
	Description string
	Metadata    map[string]any
}

// Flags is a collection of feature flags.
type Flags map[string]*Flag

// Provider provides feature flags.
type Provider interface {
	GetFlags(c *mizu.Ctx) (Flags, error)
}

// Options configures the feature middleware.
type Options struct {
	// Provider provides feature flags.
	// Default: static provider.
	Provider Provider

	// Flags is a static list of flags.
	// Used when Provider is nil.
	Flags Flags
}

// New creates feature middleware with static flags.
func New(flags Flags) mizu.Middleware {
	return WithOptions(Options{Flags: flags})
}

// WithProvider creates feature middleware with provider.
func WithProvider(provider Provider) mizu.Middleware {
	return WithOptions(Options{Provider: provider})
}

// WithOptions creates feature middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Provider == nil && opts.Flags != nil {
		opts.Provider = StaticProvider(opts.Flags)
	}
	if opts.Provider == nil {
		opts.Provider = StaticProvider(make(Flags))
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			flags, err := opts.Provider.GetFlags(c)
			if err != nil {
				// Continue with empty flags on error
				flags = make(Flags)
			}

			// Store flags in context
			ctx := context.WithValue(c.Context(), contextKey{}, flags)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// GetFlags retrieves all flags from context.
func GetFlags(c *mizu.Ctx) Flags {
	if flags, ok := c.Context().Value(contextKey{}).(Flags); ok {
		return flags
	}
	return make(Flags)
}

// Get retrieves a specific flag from context.
func Get(c *mizu.Ctx, name string) *Flag {
	flags := GetFlags(c)
	return flags[name]
}

// IsEnabled checks if a flag is enabled.
func IsEnabled(c *mizu.Ctx, name string) bool {
	flag := Get(c, name)
	return flag != nil && flag.Enabled
}

// IsDisabled checks if a flag is disabled.
func IsDisabled(c *mizu.Ctx, name string) bool {
	return !IsEnabled(c, name)
}

// Require creates middleware that requires a flag to be enabled.
func Require(name string, handler mizu.Handler) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if !IsEnabled(c, name) {
				if handler != nil {
					return handler(c)
				}
				return c.Text(404, "feature not available")
			}
			return next(c)
		}
	}
}

// RequireAll creates middleware that requires all flags to be enabled.
func RequireAll(names []string, handler mizu.Handler) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			for _, name := range names {
				if !IsEnabled(c, name) {
					if handler != nil {
						return handler(c)
					}
					return c.Text(404, "feature not available")
				}
			}
			return next(c)
		}
	}
}

// RequireAny creates middleware that requires any flag to be enabled.
func RequireAny(names []string, handler mizu.Handler) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			for _, name := range names {
				if IsEnabled(c, name) {
					return next(c)
				}
			}
			if handler != nil {
				return handler(c)
			}
			return c.Text(404, "feature not available")
		}
	}
}

// StaticProvider provides static flags.
type staticProvider struct {
	flags Flags
}

// StaticProvider creates a static flag provider.
func StaticProvider(flags Flags) Provider {
	return &staticProvider{flags: flags}
}

func (p *staticProvider) GetFlags(c *mizu.Ctx) (Flags, error) {
	// Return a copy to prevent modification
	result := make(Flags)
	for k, v := range p.flags {
		result[k] = v
	}
	return result, nil
}

// MemoryProvider provides mutable in-memory flags.
type MemoryProvider struct {
	mu    sync.RWMutex
	flags Flags
}

// NewMemoryProvider creates a new memory provider.
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		flags: make(Flags),
	}
}

func (p *MemoryProvider) GetFlags(c *mizu.Ctx) (Flags, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(Flags)
	for k, v := range p.flags {
		result[k] = v
	}
	return result, nil
}

// Set sets a flag.
func (p *MemoryProvider) Set(name string, enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.flags[name] == nil {
		p.flags[name] = &Flag{Name: name}
	}
	p.flags[name].Enabled = enabled
}

// SetFlag sets a complete flag.
func (p *MemoryProvider) SetFlag(flag *Flag) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.flags[flag.Name] = flag
}

// Delete removes a flag.
func (p *MemoryProvider) Delete(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.flags, name)
}

// Enable enables a flag.
func (p *MemoryProvider) Enable(name string) {
	p.Set(name, true)
}

// Disable disables a flag.
func (p *MemoryProvider) Disable(name string) {
	p.Set(name, false)
}

// Toggle toggles a flag.
func (p *MemoryProvider) Toggle(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.flags[name] == nil {
		p.flags[name] = &Flag{Name: name, Enabled: true}
	} else {
		p.flags[name].Enabled = !p.flags[name].Enabled
	}
}
