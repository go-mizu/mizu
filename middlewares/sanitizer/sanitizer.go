// Package sanitizer provides request sanitization middleware for Mizu.
package sanitizer

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-mizu/mizu"
)

// Options configures the sanitizer middleware.
type Options struct {
	// HTMLEscape escapes HTML characters.
	// Default: true.
	HTMLEscape bool

	// TrimSpaces trims leading/trailing spaces.
	// Default: true.
	TrimSpaces bool

	// StripTags removes HTML tags.
	// Default: false.
	StripTags bool

	// StripNonPrintable removes non-printable characters.
	// Default: true.
	StripNonPrintable bool

	// MaxLength truncates values to max length.
	// Default: 0 (unlimited).
	MaxLength int

	// Fields specifies fields to sanitize.
	// Default: all query and form fields.
	Fields []string

	// Exclude specifies fields to exclude from sanitization.
	Exclude []string
}

// New creates sanitizer middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{
		HTMLEscape:        true,
		TrimSpaces:        true,
		StripNonPrintable: true,
	})
}

// WithOptions creates sanitizer middleware with custom options.
//
//nolint:cyclop // Input sanitization requires multiple field and policy checks
func WithOptions(opts Options) mizu.Middleware {
	excludeMap := make(map[string]bool)
	for _, field := range opts.Exclude {
		excludeMap[field] = true
	}

	fieldMap := make(map[string]bool)
	for _, field := range opts.Fields {
		fieldMap[field] = true
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()

			// Sanitize query parameters
			query := r.URL.Query()
			for key, values := range query {
				if shouldSanitize(key, fieldMap, excludeMap, opts) {
					for i, v := range values {
						query[key][i] = sanitizeValue(v, opts)
					}
				}
			}
			r.URL.RawQuery = query.Encode()

			// Parse form if POST/PUT/PATCH
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				if r.Form == nil {
					_ = r.ParseForm()
				}
				for key, values := range r.Form {
					if shouldSanitize(key, fieldMap, excludeMap, opts) {
						for i, v := range values {
							r.Form[key][i] = sanitizeValue(v, opts)
						}
					}
				}
				for key, values := range r.PostForm {
					if shouldSanitize(key, fieldMap, excludeMap, opts) {
						for i, v := range values {
							r.PostForm[key][i] = sanitizeValue(v, opts)
						}
					}
				}
			}

			return next(c)
		}
	}
}

func shouldSanitize(field string, fields, exclude map[string]bool, opts Options) bool {
	if exclude[field] {
		return false
	}
	if len(opts.Fields) > 0 {
		return fields[field]
	}
	return true
}

func sanitizeValue(value string, opts Options) string {
	if opts.TrimSpaces {
		value = strings.TrimSpace(value)
	}

	if opts.StripNonPrintable {
		value = stripNonPrintable(value)
	}

	if opts.StripTags {
		value = stripTags(value)
	}

	if opts.HTMLEscape {
		value = html.EscapeString(value)
	}

	if opts.MaxLength > 0 && len(value) > opts.MaxLength {
		value = value[:opts.MaxLength]
	}

	return value
}

func stripNonPrintable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			return r
		}
		return -1
	}, s)
}

var (
	// Script/style tags with their contents
	scriptRegex = regexp.MustCompile(`(?i)<(script|style)[^>]*>[\s\S]*?</(script|style)>`)
	// All other HTML tags
	tagRegex = regexp.MustCompile(`<[^>]*>`)
)

func stripTags(s string) string {
	// First remove script/style tags with their contents
	s = scriptRegex.ReplaceAllString(s, "")
	// Then remove remaining tags
	return tagRegex.ReplaceAllString(s, "")
}

// XSS creates XSS prevention middleware.
func XSS() mizu.Middleware {
	return WithOptions(Options{
		HTMLEscape:        true,
		TrimSpaces:        true,
		StripNonPrintable: true,
	})
}

// StripHTML creates middleware that strips HTML tags.
func StripHTML() mizu.Middleware {
	return WithOptions(Options{
		StripTags:  true,
		TrimSpaces: true,
	})
}

// Trim creates middleware that trims whitespace.
func Trim() mizu.Middleware {
	return WithOptions(Options{
		TrimSpaces: true,
	})
}

// Sanitize sanitizes a single string value.
func Sanitize(value string, opts Options) string {
	return sanitizeValue(value, opts)
}

// SanitizeHTML sanitizes HTML content.
func SanitizeHTML(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}

// StripTagsString removes HTML tags from a string.
func StripTagsString(value string) string {
	return stripTags(value)
}

// TrimString trims whitespace from a string.
func TrimString(value string) string {
	return strings.TrimSpace(value)
}

// Clean applies all sanitization operations.
func Clean(value string) string {
	return sanitizeValue(value, Options{
		HTMLEscape:        true,
		TrimSpaces:        true,
		StripTags:         true,
		StripNonPrintable: true,
	})
}
