package x

import "time"

// Profile represents an X/Twitter user profile.
type Profile struct {
	ID              string    `json:"id"`
	Username        string    `json:"username"`
	Name            string    `json:"name"`
	Biography       string    `json:"biography"`
	Avatar          string    `json:"avatar"`
	Banner          string    `json:"banner"`
	Location        string    `json:"location"`
	Website         string    `json:"website"`
	URL             string    `json:"url"`               // t.co short URL
	Joined          time.Time `json:"joined"`
	Birthday        string    `json:"birthday"`
	FollowersCount  int       `json:"followers_count"`
	FollowingCount  int       `json:"following_count"`
	TweetsCount     int       `json:"tweets_count"`
	LikesCount      int       `json:"likes_count"`
	MediaCount      int       `json:"media_count"`
	ListedCount     int       `json:"listed_count"`
	IsPrivate       bool      `json:"is_private"`
	IsVerified      bool      `json:"is_verified"`
	IsBlueVerified  bool      `json:"is_blue_verified"`
	PinnedTweetIDs  []string  `json:"pinned_tweet_ids"`  // pinned tweet IDs
	ProfessionalType string   `json:"professional_type"` // e.g. "Business", "Creator"
	ProfessionalCategory string `json:"professional_category"`
	CanDM           bool      `json:"can_dm"`
	DefaultProfile  bool      `json:"default_profile"`
	DefaultAvatar   bool      `json:"default_avatar"`
	DescriptionURLs []string  `json:"description_urls"`  // expanded URLs in bio
	FetchedAt       time.Time `json:"fetched_at"`
}

// Tweet represents an X/Twitter tweet.
type Tweet struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Text           string    `json:"text"`
	HTML           string    `json:"html"`
	Username       string    `json:"username"`
	UserID         string    `json:"user_id"`
	Name           string    `json:"name"`
	PermanentURL   string    `json:"permanent_url"`
	IsRetweet      bool      `json:"is_retweet"`
	IsReply        bool      `json:"is_reply"`
	IsQuote        bool      `json:"is_quote"`
	IsPin          bool      `json:"is_pin"`
	ReplyToID      string    `json:"reply_to_id"`
	ReplyToUser    string    `json:"reply_to_user"`    // username of reply target
	QuotedID       string    `json:"quoted_id"`
	RetweetedID    string    `json:"retweeted_id"`
	Likes          int       `json:"likes"`
	Retweets       int       `json:"retweets"`
	Replies        int       `json:"replies"`
	Views          int       `json:"views"`
	Bookmarks      int       `json:"bookmarks"`
	Quotes         int       `json:"quotes"`
	Photos         []string  `json:"photos"`
	Videos         []string  `json:"videos"`
	GIFs           []string  `json:"gifs"`
	Hashtags       []string  `json:"hashtags"`
	Mentions       []string  `json:"mentions"`
	URLs           []string  `json:"urls"`
	Sensitive      bool      `json:"sensitive"`
	Language       string    `json:"language"`          // BCP47 lang code (e.g. "en")
	Source         string    `json:"source"`            // client app name
	Place          string    `json:"place"`             // geo place name
	IsEdited       bool      `json:"is_edited"`
	PostedAt       time.Time `json:"posted_at"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// FollowUser represents a user in a followers/following list.
type FollowUser struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	Biography      string `json:"biography"`
	FollowersCount int    `json:"followers_count"`
	FollowingCount int    `json:"following_count"`
	IsVerified     bool   `json:"is_verified"`
	IsPrivate      bool   `json:"is_private"`
}

// Progress reports scraping progress.
type Progress struct {
	Phase   string
	Total   int64
	Current int64
	Done    bool
}

// ProgressCallback is called with progress updates.
type ProgressCallback func(Progress)

// List represents an X/Twitter list.
type List struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Banner      string `json:"banner"`
	MemberCount int    `json:"member_count"`
	OwnerID     string `json:"owner_id"`
	OwnerName   string `json:"owner_name"`
}

// Space represents an X/Twitter audio space.
type Space struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
