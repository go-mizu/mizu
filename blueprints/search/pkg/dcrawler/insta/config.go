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

	// IPhoneAppID is the iPad/iPhone app ID used for iPhone API requests.
	IPhoneAppID = "124024574287414"

	// Endpoints
	GraphQLURL    = "https://www.instagram.com/graphql/query"
	LoginURL      = "https://www.instagram.com/api/v1/web/accounts/login/ajax/"
	TwoFactorURL  = "https://www.instagram.com/accounts/login/ajax/two_factor/"
	WebProfileURL = "https://i.instagram.com/api/v1/users/web_profile_info/"
	TopSearchURL  = "https://www.instagram.com/web/search/topsearch/"
	IPhoneAPIBase = "https://i.instagram.com/"

	// doc_id for POST-based GraphQL queries (from instaloader v4.15)
	DocIDPostDetail       = "8845758582119845" // post by shortcode
	DocIDProfilePostsAnon = "7950326061742207" // anonymous profile posts pagination
	DocIDProfilePostsAuth = "7898261790222653" // authenticated profile posts
	DocIDProfileReels     = "7845543455542541" // profile reels
	DocIDCommentReplies   = "51fdd02b67508306ad4484ff574a0b62"

	// query_hash for GET-based GraphQL queries (legacy, work with auth)
	HashSessionTest    = "d6f4427fbe92d846298cf93df0b937d3"
	HashComments       = "97b41c52301f77ce508f55e66d17620e"
	HashPostLikes      = "1cb6ec562846122743b61e492c85999f"
	HashCommentLikes   = "5f0b1f6281e72053cbc07909c8d154ae"
	HashHashtagFeed    = "9b498c08113f1e09617a1703c22b2f32"
	HashTaggedPosts    = "e31a871f7301132ceaab56507a66bbb7"
	HashFollowers      = "37479f2b8209594dde7facb0d904896a"
	HashFollowing      = "58712303d941c6855d4e888c5f0cd22f"
	HashSavedPosts     = "f883d95537fbcd400f466f63d42bd8a1"
	HashLocationFeed   = "1b84447a4d8b6d6d0426fefb34514485"
	HashStories        = "303a4ae99711322310f25250d988f3b7"
	HashHighlightItems = "45246d3fe16ccc6577e0bd297a5db1ab"
	HashHighlights     = "7c16654f22c819fb63d1183034a5162f"
	HashSimilar        = "ad99dd9d3646cc3c0dda65debcd266a7"

	// Default User-Agent (web browser)
	DefaultUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36"

	// IPhoneUserAgent mimics the Instagram iPad app.
	IPhoneUserAgent = "Instagram 361.0.0.35.82 (iPad13,8; iOS 18_0; en_US; en-US; scale=2.00; 2048x2732; 674117118) AppleWebKit/420+"

	// Default page sizes
	PostsPerPage    = 12
	CommentsPerPage = 50

	// Rate limiting constants (from instaloader)
	RateWindowGQL       = 660  // 11 minutes: per-query-type window for GraphQL
	RateLimitGQL        = 200  // max GraphQL requests per query type per window
	RateWindowOther     = 660  // 11 minutes: per-query-type window for non-GraphQL
	RateLimitOther      = 75   // max non-GraphQL requests per window
	RateWindowGQLAccum  = 600  // 10 minutes: accumulated GQL window
	RateLimitGQLAccum   = 275  // max total GraphQL requests per accumulated window
	RateWindowIPhone    = 1800 // 30 minutes: iPhone API window
	RateLimitIPhone     = 199  // max iPhone API requests per window
	RateBufferRegular   = 6    // seconds: buffer added to wait time for regular queries
	RateBufferIPhone    = 18   // seconds: buffer added to wait time for iPhone queries
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
