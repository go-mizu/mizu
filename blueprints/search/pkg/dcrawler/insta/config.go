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

	// Endpoints
	GraphQLURL    = "https://www.instagram.com/graphql/query"
	LoginURL      = "https://www.instagram.com/api/v1/web/accounts/login/ajax/"
	TwoFactorURL  = "https://www.instagram.com/accounts/login/ajax/two_factor/"
	WebProfileURL = "https://i.instagram.com/api/v1/users/web_profile_info/"
	TopSearchURL  = "https://www.instagram.com/web/search/topsearch/"

	// doc_id for POST-based GraphQL queries (from instaloader v4.15)
	DocIDPostDetail       = "8845758582119845" // post by shortcode
	DocIDProfilePostsAnon = "7950326061742207" // anonymous profile posts pagination
	DocIDProfilePostsAuth = "7898261790222653" // authenticated profile posts
	DocIDProfileReels     = "7845543455542541" // profile reels
	DocIDCommentReplies   = "51fdd02b67508306ad4484ff574a0b62"

	// query_hash for GET-based GraphQL queries (legacy, work with auth)
	HashSessionTest = "d6f4427fbe92d846298cf93df0b937d3"
	HashComments    = "97b41c52301f77ce508f55e66d17620e"
	HashPostLikes   = "1cb6ec562846122743b61e492c85999f"
	HashHashtagFeed = "9b498c08113f1e09617a1703c22b2f32"
	HashTaggedPosts = "e31a871f7301132ceaab56507a66bbb7"
	HashFollowers   = "37479f2b8209594dde7facb0d904896a"
	HashFollowing   = "58712303d941c6855d4e888c5f0cd22f"
	HashSavedPosts  = "f883d95537fbcd400f466f63d42bd8a1"

	// Default User-Agent
	DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"

	// Default page sizes
	PostsPerPage    = 12
	CommentsPerPage = 50
)

// Config holds configuration for Instagram operations.
type Config struct {
	DataDir     string
	SessionDir  string        // directory for session files
	Delay       time.Duration // delay between API requests
	MaxRetry    int
	Timeout     time.Duration
	UserAgent   string
	Workers     int    // concurrent download workers
	SessionFile string // path to session file (overrides SessionDir)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DataDir:    defaultDataDir(),
		SessionDir: defaultSessionDir(),
		Delay:      3 * time.Second,
		MaxRetry:   3,
		Timeout:    30 * time.Second,
		UserAgent:  DefaultUserAgent,
		Workers:    8,
	}
}

func defaultSessionDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "data", "instagram", ".sessions")
}

// SessionPath returns the session file path for a username.
func (c Config) SessionPath(username string) string {
	if c.SessionFile != "" {
		return c.SessionFile
	}
	return filepath.Join(c.SessionDir, username+".json")
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
