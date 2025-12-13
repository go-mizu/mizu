// Package filter provides request filtering middleware for Mizu.
package filter

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu"
)

// Options configures the filter middleware.
type Options struct {
	// AllowedMethods are HTTP methods to allow.
	// Empty allows all.
	AllowedMethods []string

	// AllowedPaths are path patterns to allow.
	// Supports glob patterns (* and **).
	AllowedPaths []string

	// BlockedPaths are path patterns to block.
	BlockedPaths []string

	// AllowedHosts are hostnames to allow.
	AllowedHosts []string

	// BlockedHosts are hostnames to block.
	BlockedHosts []string

	// AllowedUserAgents are user agent patterns to allow.
	AllowedUserAgents []string

	// BlockedUserAgents are user agent patterns to block.
	BlockedUserAgents []string

	// CustomFilter is a custom filter function.
	// Return true to allow, false to block.
	CustomFilter func(c *mizu.Ctx) bool

	// OnBlock is called when a request is blocked.
	OnBlock func(c *mizu.Ctx) error
}

// New creates filter middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates filter middleware with custom options.
//
//nolint:cyclop // Request filtering requires checking multiple criteria
func WithOptions(opts Options) mizu.Middleware {
	allowedMethods := make(map[string]bool)
	for _, m := range opts.AllowedMethods {
		allowedMethods[strings.ToUpper(m)] = true
	}

	allowedHosts := make(map[string]bool)
	for _, h := range opts.AllowedHosts {
		allowedHosts[strings.ToLower(h)] = true
	}

	blockedHosts := make(map[string]bool)
	for _, h := range opts.BlockedHosts {
		blockedHosts[strings.ToLower(h)] = true
	}

	allowedPathPatterns := compilePatterns(opts.AllowedPaths)
	blockedPathPatterns := compilePatterns(opts.BlockedPaths)
	allowedUAPatterns := compilePatterns(opts.AllowedUserAgents)
	blockedUAPatterns := compilePatterns(opts.BlockedUserAgents)

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Check method
			if len(allowedMethods) > 0 && !allowedMethods[r.Method] {
				return block(c, opts)
			}

			// Check host
			host := strings.ToLower(r.Host)
			if idx := strings.Index(host, ":"); idx != -1 {
				host = host[:idx]
			}

			if len(blockedHosts) > 0 && blockedHosts[host] {
				return block(c, opts)
			}

			if len(allowedHosts) > 0 && !allowedHosts[host] {
				return block(c, opts)
			}

			// Check path
			path := r.URL.Path
			if matchesAny(path, blockedPathPatterns) {
				return block(c, opts)
			}

			if len(allowedPathPatterns) > 0 && !matchesAny(path, allowedPathPatterns) {
				return block(c, opts)
			}

			// Check user agent
			ua := r.UserAgent()
			if matchesAny(ua, blockedUAPatterns) {
				return block(c, opts)
			}

			if len(allowedUAPatterns) > 0 && !matchesAny(ua, allowedUAPatterns) {
				return block(c, opts)
			}

			// Custom filter
			if opts.CustomFilter != nil && !opts.CustomFilter(c) {
				return block(c, opts)
			}

			return next(c)
		}
	}
}

func block(c *mizu.Ctx, opts Options) error {
	if opts.OnBlock != nil {
		return opts.OnBlock(c)
	}
	return c.Text(http.StatusForbidden, "Forbidden")
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		// Convert glob to regex
		regex := globToRegex(p)
		if re, err := regexp.Compile(regex); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

func globToRegex(glob string) string {
	var result strings.Builder
	result.WriteString("^")

	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch c {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				result.WriteString(".*")
				i++
			} else {
				result.WriteString("[^/]*")
			}
		case '?':
			result.WriteString(".")
		case '.', '+', '^', '$', '|', '(', ')', '[', ']', '{', '}', '\\':
			result.WriteString("\\")
			result.WriteByte(c)
		default:
			result.WriteByte(c)
		}
	}

	result.WriteString("$")
	return result.String()
}

func matchesAny(s string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}

// Methods creates middleware that only allows specific HTTP methods.
func Methods(methods ...string) mizu.Middleware {
	return WithOptions(Options{AllowedMethods: methods})
}

// Paths creates middleware that only allows specific paths.
func Paths(patterns ...string) mizu.Middleware {
	return WithOptions(Options{AllowedPaths: patterns})
}

// BlockPaths creates middleware that blocks specific paths.
func BlockPaths(patterns ...string) mizu.Middleware {
	return WithOptions(Options{BlockedPaths: patterns})
}

// Hosts creates middleware that only allows specific hosts.
func Hosts(hosts ...string) mizu.Middleware {
	return WithOptions(Options{AllowedHosts: hosts})
}

// BlockUserAgents creates middleware that blocks specific user agents.
func BlockUserAgents(patterns ...string) mizu.Middleware {
	return WithOptions(Options{BlockedUserAgents: patterns})
}

// Custom creates middleware with a custom filter function.
func Custom(filter func(c *mizu.Ctx) bool) mizu.Middleware {
	return WithOptions(Options{CustomFilter: filter})
}
