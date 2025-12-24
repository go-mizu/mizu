package stories

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/news/feature/users"
)

// Errors
var (
	ErrNotFound     = errors.New("story not found")
	ErrInvalidTitle = errors.New("title is required")
	ErrInvalidURL   = errors.New("invalid URL")
	ErrDuplicateURL = errors.New("URL already submitted")
)

// Validation constants
const (
	TitleMinLen = 3
	TitleMaxLen = 150
	TextMaxLen  = 40000
)

// Story represents a submitted story/link.
type Story struct {
	ID           string    `json:"id"`
	AuthorID     string    `json:"author_id"`
	Title        string    `json:"title"`
	URL          string    `json:"url,omitempty"`
	Domain       string    `json:"domain,omitempty"`
	Text         string    `json:"text,omitempty"`
	TextHTML     string    `json:"text_html,omitempty"`
	Score        int64     `json:"score"`
	CommentCount int64     `json:"comment_count"`
	HotScore     float64   `json:"-"`
	IsRemoved    bool      `json:"-"`
	CreatedAt    time.Time `json:"created_at"`

	// Joined fields
	Author   *users.User `json:"author,omitempty"`
	Tags     []string    `json:"tags,omitempty"`
	UserVote int         `json:"user_vote,omitempty"`
}

// IsLink returns true if this is a link submission.
func (s *Story) IsLink() bool {
	return s.URL != ""
}

// IsText returns true if this is a text (Ask/Show) post.
func (s *Story) IsText() bool {
	return s.URL == "" && s.Text != ""
}

// CreateIn contains input for creating a story.
type CreateIn struct {
	Title string   `json:"title"`
	URL   string   `json:"url,omitempty"`
	Text  string   `json:"text,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

// Validate validates the create input.
func (in *CreateIn) Validate() error {
	in.Title = strings.TrimSpace(in.Title)
	in.URL = strings.TrimSpace(in.URL)
	in.Text = strings.TrimSpace(in.Text)

	if len(in.Title) < TitleMinLen || len(in.Title) > TitleMaxLen {
		return ErrInvalidTitle
	}

	if in.URL != "" {
		u, err := url.Parse(in.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return ErrInvalidURL
		}
	}

	if len(in.Text) > TextMaxLen {
		return errors.New("text too long")
	}

	return nil
}

// ExtractDomain extracts domain from URL.
func ExtractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Host
	// Remove www. prefix
	host = strings.TrimPrefix(host, "www.")
	return host
}

// ListIn contains options for listing stories.
type ListIn struct {
	Sort   string // "hot", "new", "top"
	Tag    string // Filter by tag
	Domain string // Filter by domain
	Limit  int
	Offset int
}

// API defines the stories service interface.
type API interface {
	GetByID(ctx context.Context, id string, viewerID string) (*Story, error)

	// Lists
	List(ctx context.Context, in ListIn, viewerID string) ([]*Story, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int, viewerID string) ([]*Story, error)
}

// Store defines the data storage interface for stories.
type Store interface {
	GetByID(ctx context.Context, id string) (*Story, error)
	GetByURL(ctx context.Context, url string) (*Story, error)

	// Lists
	List(ctx context.Context, in ListIn) ([]*Story, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]*Story, error)
	ListByTag(ctx context.Context, tagID string, limit, offset int) ([]*Story, error)

	// Tags
	GetTagsForStory(ctx context.Context, storyID string) ([]string, error)
	GetTagsForStories(ctx context.Context, storyIDs []string) (map[string][]string, error)
}
