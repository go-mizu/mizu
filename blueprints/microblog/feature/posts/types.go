// Package posts provides post management functionality.
package posts

import (
	"time"

	"github.com/go-mizu/blueprints/microblog/feature/accounts"
)

// Visibility determines who can see a post.
type Visibility string

const (
	VisibilityPublic    Visibility = "public"    // Everyone can see
	VisibilityUnlisted  Visibility = "unlisted"  // Anyone with link
	VisibilityFollowers Visibility = "followers" // Only followers
	VisibilityDirect    Visibility = "direct"    // Only mentioned users
)

// Post represents a microblog post.
type Post struct {
	ID             string     `json:"id"`
	AccountID      string     `json:"account_id"`
	Content        string     `json:"content"`
	ContentWarning string     `json:"content_warning,omitempty"`
	Visibility     Visibility `json:"visibility"`
	ReplyToID      string     `json:"reply_to_id,omitempty"`
	ThreadID       string     `json:"thread_id,omitempty"`
	QuoteOfID      string     `json:"quote_of_id,omitempty"`
	Language       string     `json:"language,omitempty"`
	Sensitive      bool       `json:"sensitive"`
	EditedAt       *time.Time `json:"edited_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`

	// Counts
	LikesCount   int `json:"likes_count"`
	RepostsCount int `json:"reposts_count"`
	RepliesCount int `json:"replies_count"`

	// Relationships
	Account   *accounts.Account `json:"account,omitempty"`
	Media     []*Media          `json:"media,omitempty"`
	Poll      *Poll             `json:"poll,omitempty"`
	ReplyTo   *Post             `json:"reply_to,omitempty"`
	QuoteOf   *Post             `json:"quote_of,omitempty"`
	Mentions  []string          `json:"mentions,omitempty"`
	Hashtags  []string          `json:"hashtags,omitempty"`

	// Current user state
	Liked      bool `json:"liked,omitempty"`
	Reposted   bool `json:"reposted,omitempty"`
	Bookmarked bool `json:"bookmarked,omitempty"`
}

// Media represents a media attachment.
type Media struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // image, video, audio
	URL        string `json:"url"`
	PreviewURL string `json:"preview_url,omitempty"`
	AltText    string `json:"alt_text,omitempty"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Position   int    `json:"position"`
}

// Poll represents a poll attached to a post.
type Poll struct {
	ID          string       `json:"id"`
	Options     []PollOption `json:"options"`
	Multiple    bool         `json:"multiple"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	VotersCount int          `json:"voters_count"`
	Voted       bool         `json:"voted,omitempty"`
	OwnVotes    []int        `json:"own_votes,omitempty"`
	Expired     bool         `json:"expired"`
}

// PollOption represents a poll option.
type PollOption struct {
	Title      string `json:"title"`
	VotesCount int    `json:"votes_count"`
}

// CreateIn contains input for creating a post.
type CreateIn struct {
	Content        string          `json:"content"`
	ContentWarning string          `json:"content_warning,omitempty"`
	Visibility     Visibility      `json:"visibility,omitempty"`
	ReplyToID      string          `json:"reply_to_id,omitempty"`
	QuoteOfID      string          `json:"quote_of_id,omitempty"`
	Language       string          `json:"language,omitempty"`
	Sensitive      bool            `json:"sensitive,omitempty"`
	MediaIDs       []string        `json:"media_ids,omitempty"`
	Poll           *CreatePollIn   `json:"poll,omitempty"`
}

// CreatePollIn contains input for creating a poll.
type CreatePollIn struct {
	Options   []string      `json:"options"`
	Multiple  bool          `json:"multiple,omitempty"`
	ExpiresIn time.Duration `json:"expires_in,omitempty"` // Duration until expiration
}

// UpdateIn contains input for updating a post.
type UpdateIn struct {
	Content        *string `json:"content,omitempty"`
	ContentWarning *string `json:"content_warning,omitempty"`
	Sensitive      *bool   `json:"sensitive,omitempty"`
}

// ThreadContext contains a post with its ancestors and descendants.
type ThreadContext struct {
	Ancestors   []*Post `json:"ancestors"`
	Post        *Post   `json:"post"`
	Descendants []*Post `json:"descendants"`
}

// PostList is a paginated list of posts.
type PostList struct {
	Posts  []*Post `json:"posts"`
	MaxID  string  `json:"max_id,omitempty"`
	MinID  string  `json:"min_id,omitempty"`
}
