// Package insta provides Instagram scraping and search functionality
// using Instagram's public web GraphQL API and REST endpoints.
package insta

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// WebAppID is the Instagram web application ID (constant, required header).
	WebAppID = "936619743392459"

	// GraphQL endpoint
	GraphQLURL = "https://www.instagram.com/graphql/query/"

	// REST endpoints
	WebProfileURL = "https://i.instagram.com/api/v1/users/web_profile_info/"
	TopSearchURL  = "https://www.instagram.com/web/search/topsearch/"

	// GraphQL query hashes (legacy but stable)
	HashUserFeed     = "e7e2f4da4b02303f74f0841279e52d76"
	HashComments     = "f0986789a5c5d17c2400faebf16efd0d"
	HashTagFeed      = "f92f56d47dc7a55b606908374b43a314"
	HashLocationFeed = "1b84447a4d8b6d6d0426fefb34514485"
	HashTaggedFeed   = "ff260833edf142911047af6024eb634a"

	// Default User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

	// Default page sizes
	PostsPerPage    = 12
	CommentsPerPage = 50
)

// Config holds configuration for Instagram operations.
type Config struct {
	DataDir   string
	Delay     time.Duration // delay between API requests
	MaxRetry  int
	Timeout   time.Duration
	UserAgent string
	Workers   int // concurrent download workers
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DataDir:   defaultDataDir(),
		Delay:     3 * time.Second,
		MaxRetry:  3,
		Timeout:   30 * time.Second,
		UserAgent: DefaultUserAgent,
		Workers:   8,
	}
}

func defaultDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "instagram")
}

// UserDir returns the directory for a specific user's data.
func (c Config) UserDir(username string) string {
	return filepath.Join(c.DataDir, username)
}

// UserMediaDir returns the media download directory for a user.
func (c Config) UserMediaDir(username string) string {
	return filepath.Join(c.DataDir, username, "media")
}

// UserDBPath returns the DuckDB path for a user's posts.
func (c Config) UserDBPath(username string) string {
	return filepath.Join(c.DataDir, username, "posts.duckdb")
}

// ProfilePath returns the JSON file path for a user's profile.
func (c Config) ProfilePath(username string) string {
	return filepath.Join(c.DataDir, username, "profile.json")
}

// HashtagDir returns the directory for a hashtag.
func (c Config) HashtagDir(tag string) string {
	return filepath.Join(c.DataDir, "hashtag", tag)
}

// HashtagDBPath returns the DuckDB path for a hashtag's posts.
func (c Config) HashtagDBPath(tag string) string {
	return filepath.Join(c.DataDir, "hashtag", tag, "posts.duckdb")
}

// LocationDir returns the directory for a location.
func (c Config) LocationDir(id string) string {
	return filepath.Join(c.DataDir, "location", id)
}

// LocationDBPath returns the DuckDB path for a location's posts.
func (c Config) LocationDBPath(id string) string {
	return filepath.Join(c.DataDir, "location", id, "posts.duckdb")
}
