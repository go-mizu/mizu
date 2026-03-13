package facebook

import "time"

type Page struct {
	PageID         string    `json:"page_id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	About          string    `json:"about"`
	LikesCount     int64     `json:"likes_count"`
	FollowersCount int64     `json:"followers_count"`
	Verified       bool      `json:"verified"`
	Website        string    `json:"website"`
	Phone          string    `json:"phone"`
	Address        string    `json:"address"`
	URL            string    `json:"url"`
	FetchedAt      time.Time `json:"fetched_at"`
}

type Profile struct {
	ProfileID      string    `json:"profile_id"`
	Username       string    `json:"username"`
	Name           string    `json:"name"`
	Intro          string    `json:"intro"`
	Bio            string    `json:"bio"`
	FollowersCount int64     `json:"followers_count"`
	FriendsCount   int64     `json:"friends_count"`
	Verified       bool      `json:"verified"`
	Hometown       string    `json:"hometown"`
	CurrentCity    string    `json:"current_city"`
	Work           string    `json:"work"`
	Education      string    `json:"education"`
	URL            string    `json:"url"`
	FetchedAt      time.Time `json:"fetched_at"`
}

type Group struct {
	GroupID      string    `json:"group_id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Privacy      string    `json:"privacy"`
	MembersCount int64     `json:"members_count"`
	URL          string    `json:"url"`
	FetchedAt    time.Time `json:"fetched_at"`
}

type Post struct {
	PostID        string    `json:"post_id"`
	OwnerID       string    `json:"owner_id"`
	OwnerName     string    `json:"owner_name"`
	OwnerType     string    `json:"owner_type"`
	Text          string    `json:"text"`
	CreatedAtText string    `json:"created_at_text"`
	LikeCount     int64     `json:"like_count"`
	CommentCount  int64     `json:"comment_count"`
	ShareCount    int64     `json:"share_count"`
	Permalink     string    `json:"permalink"`
	MediaURLs     []string  `json:"media_urls"`
	ExternalLinks []string  `json:"external_links"`
	FetchedAt     time.Time `json:"fetched_at"`
}

type Comment struct {
	CommentID     string    `json:"comment_id"`
	PostID        string    `json:"post_id"`
	AuthorID      string    `json:"author_id"`
	AuthorName    string    `json:"author_name"`
	Text          string    `json:"text"`
	CreatedAtText string    `json:"created_at_text"`
	LikeCount     int64     `json:"like_count"`
	Permalink     string    `json:"permalink"`
	FetchedAt     time.Time `json:"fetched_at"`
}

type SearchResult struct {
	Query      string    `json:"query"`
	ResultURL  string    `json:"result_url"`
	EntityType string    `json:"entity_type"`
	Title      string    `json:"title"`
	Snippet    string    `json:"snippet"`
	FetchedAt  time.Time `json:"fetched_at"`
}

type QueueItem struct {
	ID         int64
	URL        string
	EntityType string
	Priority   int
}

type JobRecord struct {
	JobID       string
	Name        string
	Type        string
	Status      string
	StartedAt   time.Time
	CompletedAt time.Time
}
