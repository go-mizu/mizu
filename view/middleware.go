package view

import (
	"context"

	"github.com/go-mizu/mizu"
)

// engineKey is the context key for storing the engine.
type engineKey struct{}

// Middleware returns a Mizu middleware that adds the view engine to the context.
func Middleware(e *Engine) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			// Store engine in request context
			ctx := context.WithValue(c.Context(), engineKey{}, e)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req
			return next(c)
		}
	}
}

// Render renders a page template using the engine from the context.
func Render(c *mizu.Ctx, name string, data any, opts ...RenderOption) error {
	e := getEngine(c.Context())
	if e == nil {
		return ErrTemplateNotFound
	}

	// Apply render options
	cfg := &renderConfig{
		status: 200,
		layout: e.opts.DefaultLayout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Set content type and write header with status code
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer().WriteHeader(cfg.status)

	// Render
	return e.Render(c.Writer(), name, data, opts...)
}

// RenderComponent renders a component template directly.
func RenderComponent(c *mizu.Ctx, name string, data any) error {
	e := getEngine(c.Context())
	if e == nil {
		return ErrComponentNotFound
	}

	// Set content type
	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")

	return e.RenderComponent(c.Writer(), name, data)
}

// GetEngine returns the view engine from the context.
func GetEngine(c *mizu.Ctx) *Engine {
	return getEngine(c.Context())
}
