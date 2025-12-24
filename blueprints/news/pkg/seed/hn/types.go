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

// Endpoint returns the API endpoint for a feed type.
func (f FeedType) Endpoint() string {
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
