// Package bot provides bot detection middleware for Mizu.
package bot

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-mizu/mizu"
)

type contextKey struct{}

// Info contains bot detection results.
type Info struct {
	IsBot    bool
	BotName  string
	Category string
}

// Options configures the bot middleware.
type Options struct {
	// BlockBots blocks detected bots.
	// Default: false.
	BlockBots bool

	// AllowedBots is a list of allowed bot names.
	AllowedBots []string

	// BlockedBots is a list of blocked bot names.
	BlockedBots []string

	// CustomPatterns adds custom bot detection patterns.
	CustomPatterns []string

	// ErrorHandler handles blocked bot requests.
	ErrorHandler func(c *mizu.Ctx, info *Info) error
}

// Common bot patterns - ordered by specificity (more specific patterns first)
var defaultPatterns = []string{
	// Search engines (specific bot names)
	`googlebot`, `bingbot`, `yandexbot`, `baiduspider`, `duckduckbot`,
	// Social media
	`facebookexternalhit`, `twitterbot`, `linkedinbot`, `pinterestbot`,
	// SEO (specific names, not generic)
	`semrushbot`, `semrush`, `ahrefsbot`, `ahrefs`, `mj12bot`, `majestic`,
	// Tools (specific patterns)
	`curl/`, `wget/`, `python-requests`, `go-http-client`, `java/`,
	// Generic patterns (last, to avoid false positives with specific patterns)
	`crawler`, `spider`, `scraper`,
}

// Bot categories
var botCategories = map[string]string{
	"googlebot":           "search",
	"bingbot":             "search",
	"yandexbot":           "search",
	"baiduspider":         "search",
	"duckduckbot":         "search",
	"facebookexternalhit": "social",
	"twitterbot":          "social",
	"linkedinbot":         "social",
	"pinterestbot":        "social",
	"curl/":               "tool",
	"wget/":               "tool",
	"python-requests":     "tool",
	"go-http-client":      "tool",
	"java/":               "tool",
	"semrushbot":          "seo",
	"semrush":             "seo",
	"ahrefsbot":           "seo",
	"ahrefs":              "seo",
	"mj12bot":             "seo",
	"majestic":            "seo",
	"crawler":             "crawler",
	"spider":              "crawler",
	"scraper":             "crawler",
}

// New creates bot detection middleware with default options.
func New() mizu.Middleware {
	return WithOptions(Options{})
}

// WithOptions creates bot detection middleware with custom options.
func WithOptions(opts Options) mizu.Middleware {
	// Compile patterns
	patterns := append(defaultPatterns, opts.CustomPatterns...)
	regex := regexp.MustCompile(`(?i)(` + strings.Join(patterns, "|") + `)`)

	allowedMap := make(map[string]bool)
	for _, bot := range opts.AllowedBots {
		allowedMap[strings.ToLower(bot)] = true
	}

	blockedMap := make(map[string]bool)
	for _, bot := range opts.BlockedBots {
		blockedMap[strings.ToLower(bot)] = true
	}

	// Helper to check if bot matches any entry (supports partial matching)
	matchesEntry := func(botName string, entryMap map[string]bool) bool {
		if entryMap[botName] {
			return true
		}
		// Also check if bot name contains any entry as prefix
		for entry := range entryMap {
			if strings.HasPrefix(botName, entry) || strings.HasPrefix(entry, botName) {
				return true
			}
		}
		return false
	}

	return func(next mizu.Handler) mizu.Handler {
		return func(c *mizu.Ctx) error {
			userAgent := strings.ToLower(c.Request().UserAgent())

			info := &Info{}

			// Detect bot
			if matches := regex.FindStringSubmatch(userAgent); len(matches) > 0 {
				info.IsBot = true
				info.BotName = matches[1]
				info.Category = botCategories[info.BotName]
				if info.Category == "" {
					info.Category = "other"
				}
			}

			// Store info in context
			ctx := context.WithValue(c.Context(), contextKey{}, info)
			req := c.Request().WithContext(ctx)
			*c.Request() = *req

			// Check if should block
			if opts.BlockBots && info.IsBot {
				botName := strings.ToLower(info.BotName)
				shouldBlock := false

				if len(opts.AllowedBots) > 0 {
					// Allowlist mode: block all bots except those in AllowedBots
					if !matchesEntry(botName, allowedMap) {
						shouldBlock = true
					}
				} else if len(opts.BlockedBots) > 0 {
					// Blocklist mode: only block bots in BlockedBots
					if matchesEntry(botName, blockedMap) {
						shouldBlock = true
					}
				} else {
					// Block all bots
					shouldBlock = true
				}

				if shouldBlock {
					if opts.ErrorHandler != nil {
						return opts.ErrorHandler(c, info)
					}
					return c.Text(http.StatusForbidden, "bot access denied")
				}
			}

			return next(c)
		}
	}
}

// Get retrieves bot info from context.
func Get(c *mizu.Ctx) *Info {
	if info, ok := c.Context().Value(contextKey{}).(*Info); ok {
		return info
	}
	return &Info{}
}

// IsBot returns whether the request is from a bot.
func IsBot(c *mizu.Ctx) bool {
	return Get(c).IsBot
}

// BotName returns the detected bot name.
func BotName(c *mizu.Ctx) string {
	return Get(c).BotName
}

// Category returns the bot category.
func Category(c *mizu.Ctx) string {
	return Get(c).Category
}

// Block creates middleware that blocks all bots.
func Block() mizu.Middleware {
	return WithOptions(Options{BlockBots: true})
}

// Allow creates middleware that allows only specified bots.
func Allow(bots ...string) mizu.Middleware {
	return WithOptions(Options{
		BlockBots:   true,
		AllowedBots: bots,
	})
}

// Deny creates middleware that denies specified bots.
func Deny(bots ...string) mizu.Middleware {
	return WithOptions(Options{
		BlockBots:   true,
		BlockedBots: bots,
	})
}

// AllowSearchEngines creates middleware that allows search engine bots.
func AllowSearchEngines() mizu.Middleware {
	return Allow("googlebot", "bingbot", "yandexbot", "baiduspider", "duckduckbot")
}

// AllowSocialBots creates middleware that allows social media bots.
func AllowSocialBots() mizu.Middleware {
	return Allow("facebookexternalhit", "twitterbot", "linkedinbot", "pinterestbot")
}
