package xrequestedwith

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

type Options struct {
	Value        string
	SkipMethods  []string
	SkipPaths    []string
	ErrorHandler func(c *mizu.Ctx) error
}

func New() mizu.Middleware { return WithOptions(Options{}) }

func WithOptions(opts Options) mizu.Middleware {
	if opts.Value == "" {
		opts.Value = "XMLHttpRequest"
	}
	if opts.SkipMethods == nil {
		opts.SkipMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	}

	skipMethods := make(map[string]bool, len(opts.SkipMethods))
	for _, m := range opts.SkipMethods {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		skipMethods[strings.ToUpper(m)] = true
	}

	skipPaths := make(map[string]bool, len(opts.SkipPaths))
	for _, p := range opts.SkipPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		skipPaths[p] = true
	}

	want := opts.Value

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			req := c.Request()

			if skipMethods[strings.ToUpper(req.Method)] {
				return next(c)
			}

			path := ""
			if req.URL != nil {
				path = req.URL.Path
			}
			if skipPaths[path] {
				return next(c)
			}

			got := req.Header.Get("X-Requested-With")
			if !strings.EqualFold(got, want) {
				if opts.ErrorHandler != nil {
					return opts.ErrorHandler(c)
				}
				return c.Text(http.StatusBadRequest, "X-Requested-With header required")
			}

			return next(c)
		}
	}
}

func Require(value string) mizu.Middleware { return WithOptions(Options{Value: value}) }

func AJAXOnly() mizu.Middleware {
	return WithOptions(Options{Value: "XMLHttpRequest", SkipMethods: []string{}})
}

func IsAJAX(c *mizu.Ctx) bool {
	return strings.EqualFold(c.Request().Header.Get("X-Requested-With"), "XMLHttpRequest")
}
