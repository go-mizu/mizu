package hn

import "time"

// ItemType represents the type of HN item.
type ItemType string

const (
	ItemTypeStory   ItemType = "story"
	ItemTypeComment ItemType = "comment"
	ItemTypeJob     ItemType = "job"
	ItemTypePoll    ItemType = "poll"
	ItemTypePollOpt ItemType = "pollopt"
)

// Item represents a Hacker News item.
type Item struct {
	ID          int      `json:"id"`
	Type        ItemType `json:"type"`
	By          string   `json:"by"`
	Time        int64    `json:"time"`
	Text        string   `json:"text"`        // HTML content
	Title       string   `json:"title"`       // Story/job title
	URL         string   `json:"url"`         // Story URL
	Score       int      `json:"score"`
	Kids        []int    `json:"kids"`        // Child comment IDs
	Parent      int      `json:"parent"`      // Parent story/comment ID
	Poll        int      `json:"poll"`        // Associated poll
	Parts       []int    `json:"parts"`       // Poll options
	Descendants int      `json:"descendants"` // Total comment count
	Deleted     bool     `json:"deleted"`
	Dead        bool     `json:"dead"`
}

// CreatedTime returns the creation time as time.Time.
func (i *Item) CreatedTime() time.Time {
	return time.Unix(i.Time, 0)
}

// IsDeleted returns true if the item is deleted or dead.
func (i *Item) IsDeleted() bool {
	return i.Deleted || i.Dead
}

// IsStory returns true if the item is a story (including jobs).
func (i *Item) IsStory() bool {
	return i.Type == ItemTypeStory || i.Type == ItemTypeJob
}

// IsComment returns true if the item is a comment.
func (i *Item) IsComment() bool {
	return i.Type == ItemTypeComment
}

// User represents a Hacker News user profile.
type User struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	Karma     int    `json:"karma"`
	About     string `json:"about"`
	Submitted []int  `json:"submitted"`
}

// FeedType represents different HN feeds.
type FeedType string

const (
	FeedTop  FeedType = "top"
	FeedNew  FeedType = "new"
	FeedBest FeedType = "best"
	FeedAsk  FeedType = "ask"
	FeedShow FeedType = "show"
	FeedJobs FeedType = "jobs"
)

// FeedEndpoint returns the API endpoint for a feed type.
func (f FeedType) FeedEndpoint() string {
	switch f {
	case FeedTop:
		return "topstories"
	case FeedNew:
		return "newstories"
	case FeedBest:
		return "beststories"
	case FeedAsk:
		return "askstories"
	case FeedShow:
		return "showstories"
	case FeedJobs:
		return "jobstories"
	default:
		return "topstories"
	}
}

// HNFetchOpts contains HN-specific fetch options.
type HNFetchOpts struct {
	Feed         FeedType // top, new, best, ask, show, jobs
	Limit        int      // Max stories to fetch
	FromItemID   int      // Start from specific item ID (for resumable)
	SkipExisting bool     // Skip items that already exist in seed_mappings
	Force        bool     // Force re-fetch even if exists
}
