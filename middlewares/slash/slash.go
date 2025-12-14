// Package slash provides trailing slash handling middleware for Mizu.
package slash

import (
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
)

func Add() mizu.Middleware { return AddCode(http.StatusMovedPermanently) }

func AddCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			req := c.Request()

			// Avoid redirecting non-idempotent requests.
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				return next(c)
			}

			p := req.URL.Path
			if p == "" {
				p = "/"
			}
			if p != "/" && !strings.HasSuffix(p, "/") {
				target := p + "/"
				if q := req.URL.RawQuery; q != "" {
					target += "?" + q
				}
				return c.Redirect(code, target)
			}
			return next(c)
		}
	}
}

func Remove() mizu.Middleware { return RemoveCode(http.StatusMovedPermanently) }

func RemoveCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			req := c.Request()

			// Avoid redirecting non-idempotent requests.
			if req.Method != http.MethodGet && req.Method != http.MethodHead {
				return next(c)
			}

			p := req.URL.Path
			if p == "" {
				p = "/"
			}
			if p != "/" && strings.HasSuffix(p, "/") {
				target := strings.TrimSuffix(p, "/")
				if q := req.URL.RawQuery; q != "" {
					target += "?" + q
				}
				return c.Redirect(code, target)
			}
			return next(c)
		}
	}
}
