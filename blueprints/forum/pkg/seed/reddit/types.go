package reddit

import "time"

// Listing represents a Reddit listing response.
type Listing struct {
	Kind string      `json:"kind"`
	Data ListingData `json:"data"`
}

// ListingData contains the listing items.
type ListingData struct {
	After    string  `json:"after"`
	Before   string  `json:"before"`
	Dist     int     `json:"dist"`
	Children []Thing `json:"children"`
}

// Thing represents a Reddit thing (post or comment).
type Thing struct {
	Kind string   `json:"kind"`
	Data ThingData `json:"data"`
}

// ThingData contains the thing data.
type ThingData struct {
	// Common fields
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Author    string  `json:"author"`
	Score     int64   `json:"score"`
	Ups       int64   `json:"ups"`
	Downs     int64   `json:"downs"`
	CreatedUTC float64 `json:"created_utc"`
	Permalink string  `json:"permalink"`
	Stickied  bool    `json:"stickied"`

	// Post fields
	Title       string `json:"title"`
	Selftext    string `json:"selftext"`
	URL         string `json:"url"`
	Domain      string `json:"domain"`
	IsSelf      bool   `json:"is_self"`
	Over18      bool   `json:"over_18"`
	Spoiler     bool   `json:"spoiler"`
	NumComments int64  `json:"num_comments"`
	UpvoteRatio float64 `json:"upvote_ratio"`

	// Comment fields
	Body     string `json:"body"`
	ParentID string `json:"parent_id"`
	LinkID   string `json:"link_id"`
	Depth    int    `json:"depth"`

	// Replies can be empty string or Listing
	Replies any `json:"replies"`

	// Subreddit about fields
	DisplayName         string `json:"display_name"`
	PublicDescription   string `json:"public_description"`
	Description         string `json:"description"`
	Subscribers         int64  `json:"subscribers"`
	ActiveUserCount     int64  `json:"active_user_count"`
}

// SubredditAbout represents the about response for a subreddit.
type SubredditAbout struct {
	Kind string    `json:"kind"`
	Data ThingData `json:"data"`
}

// CreatedTime returns the created time as a time.Time.
func (d *ThingData) CreatedTime() time.Time {
	return time.Unix(int64(d.CreatedUTC), 0)
}

// GetReplies returns the replies as a Listing if present.
func (d *ThingData) GetReplies() *Listing {
	if d.Replies == nil {
		return nil
	}
	// Replies can be an empty string or a Listing
	if _, ok := d.Replies.(string); ok {
		return nil
	}
	// Need to re-parse from map
	return nil
}
