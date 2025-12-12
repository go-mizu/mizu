// Package language provides content negotiation middleware for Mizu.
package language

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Options configures the language middleware.
type Options struct {
	// Supported is the list of supported languages.
	// Default: []string{"en"}.
	Supported []string

	// Default is the default language.
	// Default: "en".
	Default string

	// QueryParam is the query parameter to check.
	// Default: "lang".
	QueryParam string

	// CookieName is the cookie name to check.
	// Default: "lang".
	CookieName string

	// Header is the header to check.
	// Default: "Accept-Language".
	Header string

	// PathPrefix enables path prefix detection (e.g., /en/page).
	// Default: false.
	PathPrefix bool
}

// New creates language middleware with supported languages.
func New(supported ...string) mizu.Middleware {
	return WithOptions(Options{Supported: supported})
}

// WithOptions creates language middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	if len(opts.Supported) == 0 {
		opts.Supported = []string{"en"}
	}
	if opts.Default == "" {
		opts.Default = opts.Supported[0]
	}
	if opts.QueryParam == "" {
		opts.QueryParam = "lang"
	}
	if opts.CookieName == "" {
		opts.CookieName = "lang"
	}
	if opts.Header == "" {
		opts.Header = "Accept-Language"
	}

	// Create lookup map with lowercase keys for case-insensitive matching
	supportedMap := make(map[string]bool)
	for _, lang := range opts.Supported {
		supportedMap[strings.ToLower(lang)] = true
		// Also support language codes without region
		if idx := strings.Index(lang, "-"); idx > 0 {
			supportedMap[strings.ToLower(lang[:idx])] = true
		}
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			r := c.Request()
			var lang string

			// 1. Check query parameter
			if q := c.Query(opts.QueryParam); q != "" {
				if isSupported(q, supportedMap) {
					lang = normalize(q, opts.Supported)
				}
			}

			// 2. Check path prefix
			if lang == "" && opts.PathPrefix {
				path := r.URL.Path
				if len(path) >= 3 && path[0] == '/' {
					prefix := path[1:3]
					if len(path) == 3 || path[3] == '/' {
						if isSupported(prefix, supportedMap) {
							lang = normalize(prefix, opts.Supported)
							// Strip prefix from path
							if len(path) > 3 {
								r.URL.Path = path[3:]
							} else {
								r.URL.Path = "/"
							}
						}
					}
				}
			}

			// 3. Check cookie
			if lang == "" {
				if cookie, err := c.Cookie(opts.CookieName); err == nil {
					if isSupported(cookie.Value, supportedMap) {
						lang = normalize(cookie.Value, opts.Supported)
					}
				}
			}

			// 4. Check Accept-Language header
			if lang == "" {
				acceptLang := r.Header.Get(opts.Header)
				if acceptLang != "" {
					lang = parseAcceptLanguage(acceptLang, opts.Supported, supportedMap)
				}
			}

			// 5. Fall back to default
			if lang == "" {
				lang = opts.Default
			}

			// Store in context
			ctx := context.WithValue(c.Context(), contextKey{}, lang)
			req := r.WithContext(ctx)
			*c.Request() = *req

			return next(c)
		}
	}
}

// Get retrieves the detected language from context.
func Get(c *mizu.Ctx) string {
	if lang, ok := c.Context().Value(contextKey{}).(string); ok {
		return lang
	}
	return "en"
}

// FromContext is an alias for Get.
func FromContext(c *mizu.Ctx) string {
	return Get(c)
}

func isSupported(lang string, supported map[string]bool) bool {
	lang = strings.ToLower(lang)
	if supported[lang] {
		return true
	}
	// Check without region
	if idx := strings.Index(lang, "-"); idx > 0 {
		return supported[lang[:idx]]
	}
	return false
}

func normalize(lang string, supported []string) string {
	lang = strings.ToLower(lang)
	// First pass: look for exact match
	for _, s := range supported {
		if strings.EqualFold(s, lang) {
			return s
		}
	}
	// Second pass: match by base language (less specific)
	for _, s := range supported {
		if idx := strings.Index(s, "-"); idx > 0 {
			if strings.EqualFold(s[:idx], lang) {
				return s
			}
		}
		if idx := strings.Index(lang, "-"); idx > 0 {
			if strings.EqualFold(s, lang[:idx]) {
				return s
			}
		}
	}
	return lang
}

type langQuality struct {
	lang    string
	quality float64
}

func parseAcceptLanguage(header string, supported []string, supportedMap map[string]bool) string {
	parts := strings.Split(header, ",")
	langs := make([]langQuality, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		var lang string
		quality := 1.0

		if idx := strings.Index(part, ";"); idx > 0 {
			lang = strings.TrimSpace(part[:idx])
			qPart := strings.TrimSpace(part[idx+1:])
			if strings.HasPrefix(qPart, "q=") {
				if q, err := strconv.ParseFloat(qPart[2:], 64); err == nil {
					quality = q
				}
			}
		} else {
			lang = part
		}

		if isSupported(lang, supportedMap) {
			langs = append(langs, langQuality{lang: normalize(lang, supported), quality: quality})
		}
	}

	if len(langs) == 0 {
		return ""
	}

	// Sort by quality descending
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].quality > langs[j].quality
	})

	return langs[0].lang
}
