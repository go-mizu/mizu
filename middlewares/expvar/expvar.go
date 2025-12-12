// Package expvar provides expvar endpoint middleware for Mizu.
package expvar

import (
	"expvar"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the expvar middleware.
type Options struct {
	// Path is the URL path for expvar endpoint.
	// Default: "/debug/vars".
	Path string
}

// New creates expvar middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates expvar middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if opts.Path == "" {
		opts.Path = "/debug/vars"
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().URL.Path == opts.Path {
				expvar.Handler().ServeHTTP(c.Writer(), c.Request())
				return nil
			}
			return next(c)
		}
	}
}

// NewInt creates and publishes a new expvar.Int.
func NewInt(name string) *expvar.Int {
	return expvar.NewInt(name)
}

// NewFloat creates and publishes a new expvar.Float.
func NewFloat(name string) *expvar.Float {
	return expvar.NewFloat(name)
}

// NewString creates and publishes a new expvar.String.
func NewString(name string) *expvar.String {
	return expvar.NewString(name)
}

// NewMap creates and publishes a new expvar.Map.
func NewMap(name string) *expvar.Map {
	return expvar.NewMap(name)
}

// Publish publishes a named expvar.Var.
func Publish(name string, v expvar.Var) {
	expvar.Publish(name, v)
}

// Get retrieves a published Var by name.
func Get(name string) expvar.Var {
	return expvar.Get(name)
}

// KeyValue is a type alias for expvar.KeyValue.
type KeyValue = expvar.KeyValue

// Do calls f for each exported variable.
func Do(f func(KeyValue)) {
	expvar.Do(f)
}

// Handler returns the expvar HTTP handler.
func Handler() http.Handler {
	return expvar.Handler()
}

// JSON returns all expvars as JSON string.
func JSON() string {
	var b strings.Builder
	b.WriteString("{")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			b.WriteString(",")
		}
		first = false
		b.WriteString(`"`)
		b.WriteString(kv.Key)
		b.WriteString(`":`)
		b.WriteString(kv.Value.String())
	})
	b.WriteString("}")
	return b.String()
}
