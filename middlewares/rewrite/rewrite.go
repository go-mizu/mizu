// Package rewrite provides URL rewriting middleware for Mizu.
package rewrite

import (
	"regexp"
	"strings"

	"github.com/go-mizu/mizu"
)

// Rule represents a rewrite rule.
type Rule struct {
	// Match is the pattern to match.
	// Can be a simple prefix or a regex pattern.
	Match string

	// Rewrite is the replacement pattern.
	// Use $1, $2, etc. for regex capture groups.
	Rewrite string

	// Regex indicates if Match is a regex pattern.
	Regex bool

	// compiled regex (internal)
	re *regexp.Regexp
}

// Options configures the rewrite middleware.
type Options struct {
	// Rules is the list of rewrite rules.
	Rules []Rule
}

// New creates rewrite middleware with rules.
func New(rules ...Rule) mizu.Middleware {
	return WithOptions(Options{Rules: rules})
}

// WithOptions creates rewrite middleware with options.
func WithOptions(opts Options) mizu.Middleware {
	// Compile regex rules
	for i := range opts.Rules {
		if opts.Rules[i].Regex {
			opts.Rules[i].re = regexp.MustCompile(opts.Rules[i].Match)
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			path := c.Request().URL.Path

			for _, rule := range opts.Rules {
				var newPath string
				var matched bool

				if rule.Regex && rule.re != nil {
					if rule.re.MatchString(path) {
						newPath = rule.re.ReplaceAllString(path, rule.Rewrite)
						matched = true
					}
				} else {
					if strings.HasPrefix(path, rule.Match) {
						newPath = rule.Rewrite + strings.TrimPrefix(path, rule.Match)
						matched = true
					}
				}

				if matched {
					c.Request().URL.Path = newPath
					break
				}
			}

			return next(c)
		}
	}
}

// Prefix creates a simple prefix rewrite rule.
func Prefix(from, to string) Rule {
	return Rule{
		Match:   from,
		Rewrite: to,
	}
}

// Regex creates a regex-based rewrite rule.
func Regex(pattern, replacement string) Rule {
	return Rule{
		Match:   pattern,
		Rewrite: replacement,
		Regex:   true,
	}
}
