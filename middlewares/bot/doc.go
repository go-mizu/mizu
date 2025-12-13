/*
Package bot provides bot detection middleware for Mizu.

# Overview

The bot middleware detects and optionally blocks bot traffic based on
User-Agent analysis. It identifies various types of bots including search
engine crawlers, social media bots, SEO tools, and HTTP clients.

# Usage

Basic usage to detect bots:

	app := mizu.New()
	app.Use(bot.New())

	app.Get("/", func(c *mizu.Ctx) error {
	    if bot.IsBot(c) {
	        return c.Text(200, "Hello, bot!")
	    }
	    return c.Text(200, "Hello, human!")
	})

# Configuration

Options:

  - BlockBots: Block detected bots (default: false)
  - AllowedBots: List of allowed bot names (allowlist mode)
  - BlockedBots: List of blocked bot names (blocklist mode)
  - CustomPatterns: Additional bot detection patterns
  - ErrorHandler: Custom handler for blocked requests

# Bot Categories

The middleware categorizes bots:

  - search: Googlebot, Bingbot, Yandex, Baidu, DuckDuckBot
  - social: Facebook, Twitter, LinkedIn, Pinterest
  - seo: Semrush, Ahrefs, Majestic
  - tool: curl, wget, Python requests, Go HTTP client
  - crawler: Generic crawlers, spiders, scrapers

# Example

Block all bots except search engines:

	app.Use(bot.AllowSearchEngines())

Get bot info:

	info := bot.Get(c)
	fmt.Printf("Bot: %s, Category: %s\n", info.BotName, info.Category)

# See Also

  - Package ipfilter for IP-based access control
  - Package ratelimit for rate limiting
*/
package bot
