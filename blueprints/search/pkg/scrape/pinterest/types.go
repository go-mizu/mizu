// Package pinterest scrapes public Pinterest data into a local DuckDB database.
package pinterest

import (
	"net/url"
	"strings"
	"time"
)

const (
	BaseURL      = "https://www.pinterest.com"
	EntitySearch = "search"
	EntityBoard  = "board"
	EntityUser   = "user"
)

// Pin represents a single Pinterest pin.
type Pin struct {
	PinID        string    `json:"pin_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	AltText      string    `json:"alt_text"`
	ImageURL     string    `json:"image_url"`
	ImageWidth   int       `json:"image_width"`
	ImageHeight  int       `json:"image_height"`
	PinURL       string    `json:"pin_url"`
	SourceURL    string    `json:"source_url"`
	BoardID      string    `json:"board_id"`
	BoardName    string    `json:"board_name"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	SavedCount   int       `json:"saved_count"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// Board represents a Pinterest board (collection of pins).
type Board struct {
	BoardID       string    `json:"board_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	PinCount      int       `json:"pin_count"`
	FollowerCount int       `json:"follower_count"`
	CoverURL      string    `json:"cover_url"`
	Category      string    `json:"category"`
	IsSecret      bool      `json:"is_secret"`
	URL           string    `json:"url"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// User represents a Pinterest user profile.
type User struct {
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	FullName       string    `json:"full_name"`
	Bio            string    `json:"bio"`
	Website        string    `json:"website"`
	FollowerCount  int       `json:"follower_count"`
	FollowingCount int       `json:"following_count"`
	BoardCount     int       `json:"board_count"`
	PinCount       int       `json:"pin_count"`
	MonthlyViews   int64     `json:"monthly_views"`
	AvatarURL      string    `json:"avatar_url"`
	URL            string    `json:"url"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// QueueItem is a row from the state.duckdb queue table.
type QueueItem struct {
	ID         int64
	URL        string
	EntityType string
	Priority   int
}

// DBStats holds row counts per table.
type DBStats struct {
	Pins   int64
	Boards int64
	Users  int64
	DBSize int64
}

// ExtractUsername extracts the Pinterest username from a URL or returns the input unchanged.
func ExtractUsername(s string) string {
	if !strings.HasPrefix(s, "http") {
		// Bare username — strip any leading @
		return strings.TrimPrefix(s, "@")
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return s
}

// ExtractBoardSlug extracts (username, slug) from a board URL or "user/board" string.
func ExtractBoardSlug(s string) (username, slug string) {
	// Normalize to a path string
	path := s
	if strings.HasPrefix(s, "http") {
		u, err := url.Parse(s)
		if err != nil {
			return "", ""
		}
		path = u.Path
	}
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// NormalizeBoardURL accepts a bare "user/board" slug or full URL and returns a full Pinterest URL.
func NormalizeBoardURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	return BaseURL + "/" + strings.Trim(s, "/") + "/"
}

// NormalizeUserURL accepts a bare username or full URL and returns a full Pinterest URL.
func NormalizeUserURL(s string) string {
	if strings.HasPrefix(s, "http") {
		return s
	}
	username := strings.TrimPrefix(s, "@")
	return BaseURL + "/" + username + "/"
}
