// Package x provides X/Twitter scraping via direct GraphQL API calls
// with cookie-based authentication, ported from Nitter's approach.
package x

import (
	"os"
	"path/filepath"
	"time"
)

// Config holds configuration for X/Twitter operations.
type Config struct {
	DataDir    string
	SessionDir string
	Delay      time.Duration // delay between API requests
	MaxRetry   int
	Timeout    time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DataDir:    defaultDataDir(),
		SessionDir: defaultSessionDir(),
		Delay:      500 * time.Millisecond,
		MaxRetry:   3,
		Timeout:    30 * time.Second,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "x")
}

func defaultSessionDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "x", ".sessions")
}

// SessionPath returns the session file path for a username.
func (c Config) SessionPath(username string) string {
	return filepath.Join(c.SessionDir, username+".json")
}

// UserDir returns the directory for a specific user's data.
func (c Config) UserDir(username string) string {
	return filepath.Join(c.DataDir, username)
}

// UserDBPath returns the DuckDB path for a user's tweets.
func (c Config) UserDBPath(username string) string {
	return filepath.Join(c.DataDir, username, "tweets.duckdb")
}

// UserMediaDir returns the media download directory for a user.
func (c Config) UserMediaDir(username string) string {
	return filepath.Join(c.DataDir, username, "media")
}

// ProfilePath returns the JSON file path for a user's profile.
func (c Config) ProfilePath(username string) string {
	return filepath.Join(c.DataDir, username, "profile.json")
}

// SearchDir returns the directory for search results.
func (c Config) SearchDir(query string) string {
	return filepath.Join(c.DataDir, "search", sanitizeDirName(query))
}

// SearchDBPath returns the DuckDB path for search results.
func (c Config) SearchDBPath(query string) string {
	return filepath.Join(c.DataDir, "search", sanitizeDirName(query), "tweets.duckdb")
}

// HashtagDir returns the directory for a hashtag.
func (c Config) HashtagDir(tag string) string {
	return filepath.Join(c.DataDir, "hashtag", tag)
}

// HashtagDBPath returns the DuckDB path for a hashtag's tweets.
func (c Config) HashtagDBPath(tag string) string {
	return filepath.Join(c.DataDir, "hashtag", tag, "tweets.duckdb")
}

// sanitizeDirName replaces characters unsafe for directory names.
func sanitizeDirName(s string) string {
	var b []byte
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	if len(b) == 0 {
		return "_"
	}
	return string(b)
}
