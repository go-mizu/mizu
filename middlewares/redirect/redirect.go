// Package redirect provides URL redirection middleware for Mizu.
package redirect

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu"
)

// Rule defines a redirect rule.
type Rule struct {
	// From is the source path or pattern.
	From string

	// To is the target URL (supports $1, $2 for regex captures).
	To string

	// Code is the HTTP redirect status code.
	// Default: 301.
	Code int

	// Regex indicates if From is a regex pattern.
	Regex bool

	compiled *regexp.Regexp
}

// HTTPSRedirect redirects HTTP to HTTPS.
func HTTPSRedirect() mizu.Middleware {
	return HTTPSRedirectCode(http.StatusMovedPermanently)
}

// HTTPSRedirectCode redirects HTTP to HTTPS with specific code.
func HTTPSRedirectCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			if c.Request().TLS == nil &&
				c.Request().Header.Get("X-Forwarded-Proto") != "https" {
				host := c.Request().Host
				url := "https://" + host + c.Request().URL.RequestURI()
				return c.Redirect(code, url)
			}
			return next(c)
		}
	}
}

// WWWRedirect redirects to www subdomain.
func WWWRedirect() mizu.Middleware {
	return WWWRedirectCode(http.StatusMovedPermanently)
}

// WWWRedirectCode redirects to www subdomain with specific code.
func WWWRedirectCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			host := c.Request().Host
			if !strings.HasPrefix(host, "www.") {
				scheme := "http"
				if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
					scheme = "https"
				}
				url := scheme + "://www." + host + c.Request().URL.RequestURI()
				return c.Redirect(code, url)
			}
			return next(c)
		}
	}
}

// NonWWWRedirect redirects to non-www domain.
func NonWWWRedirect() mizu.Middleware {
	return NonWWWRedirectCode(http.StatusMovedPermanently)
}

// NonWWWRedirectCode redirects to non-www domain with specific code.
func NonWWWRedirectCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			host := c.Request().Host
			if strings.HasPrefix(host, "www.") {
				scheme := "http"
				if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
					scheme = "https"
				}
				url := scheme + "://" + strings.TrimPrefix(host, "www.") + c.Request().URL.RequestURI()
				return c.Redirect(code, url)
			}
			return next(c)
		}
	}
}

// New creates redirect middleware with rules.
func New(rules []Rule) mizu.Middleware {
	// Compile regex patterns
	for i := range rules {
		if rules[i].Code == 0 {
			rules[i].Code = http.StatusMovedPermanently
		}
		if rules[i].Regex {
			rules[i].compiled = regexp.MustCompile(rules[i].From)
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			for _, rule := range rules {
				if rule.Regex && rule.compiled != nil {
					if matches := rule.compiled.FindStringSubmatch(path); matches != nil {
						target := rule.To
						for i, match := range matches {
							target = strings.ReplaceAll(target, "$"+string(rune('0'+i)), match)
						}
						// Preserve query string
						if c.Request().URL.RawQuery != "" {
							target += "?" + c.Request().URL.RawQuery
						}
						return c.Redirect(rule.Code, target)
					}
				} else if path == rule.From {
					target := rule.To
					if c.Request().URL.RawQuery != "" {
						target += "?" + c.Request().URL.RawQuery
					}
					return c.Redirect(rule.Code, target)
				}
			}

			return next(c)
		}
	}
}

// TrailingSlashRedirect redirects to URL with trailing slash.
func TrailingSlashRedirect() mizu.Middleware {
	return TrailingSlashRedirectCode(http.StatusMovedPermanently)
}

// TrailingSlashRedirectCode redirects with specific code.
func TrailingSlashRedirectCode(code int) mizu.Middleware {
	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path
			if path != "/" && !strings.HasSuffix(path, "/") {
				target := path + "/"
				if c.Request().URL.RawQuery != "" {
					target += "?" + c.Request().URL.RawQuery
				}
				return c.Redirect(code, target)
			}
			return next(c)
		}
	}
}
