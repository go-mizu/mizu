package insta

import "time"

// Profile represents an Instagram user profile.
type Profile struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	FullName       string    `json:"full_name"`
	Biography      string    `json:"biography"`
	ProfilePicURL  string    `json:"profile_pic_url"`
	ExternalURL    string    `json:"external_url"`
	IsPrivate      bool      `json:"is_private"`
	IsVerified     bool      `json:"is_verified"`
	IsBusiness     bool      `json:"is_business_account"`
	CategoryName   string    `json:"category_name"`
	FollowerCount  int64     `json:"follower_count"`
	FollowingCount int64     `json:"following_count"`
	PostCount      int64     `json:"post_count"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// Post represents an Instagram post (image, video, or carousel).
type Post struct {
	ID           string    `json:"id"`
	Shortcode    string    `json:"shortcode"`
	TypeName     string    `json:"type_name"` // GraphImage, GraphVideo, GraphSidecar
	Caption      string    `json:"caption"`
	DisplayURL   string    `json:"display_url"`
	VideoURL     string    `json:"video_url"`
	IsVideo      bool      `json:"is_video"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	ViewCount    int64     `json:"view_count"`
	TakenAt      time.Time `json:"taken_at"`
	LocationID   string    `json:"location_id"`
	LocationName string    `json:"location_name"`
	OwnerID      string    `json:"owner_id"`
	OwnerName    string    `json:"owner_username"`
	Children     []Post    `json:"children,omitempty"`
	FetchedAt    time.Time `json:"fetched_at"`
}

// Comment represents a comment on an Instagram post.
type Comment struct {
	ID         string    `json:"id"`
	Text       string    `json:"text"`
	AuthorID   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	LikeCount  int64     `json:"like_count"`
	CreatedAt  time.Time `json:"created_at"`
	PostID     string    `json:"post_id"`
}

// SearchResult represents a search result from Instagram's top search.
type SearchResult struct {
	Users     []SearchUser     `json:"users"`
	Hashtags  []SearchHashtag  `json:"hashtags"`
	Places    []SearchPlace    `json:"places"`
}

// SearchUser is a user result from search.
type SearchUser struct {
	ID         string `json:"pk"`
	Username   string `json:"username"`
	FullName   string `json:"full_name"`
	IsPrivate  bool   `json:"is_private"`
	IsVerified bool   `json:"is_verified"`
	PicURL     string `json:"profile_pic_url"`
	Followers  int64  `json:"follower_count"`
}

// SearchHashtag is a hashtag result from search.
type SearchHashtag struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	MediaCount int64  `json:"media_count"`
}

// SearchPlace is a place/location result from search.
type SearchPlace struct {
	LocationID int64   `json:"location_id"`
	Title      string  `json:"title"`
	Address    string  `json:"address"`
	City       string  `json:"city"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

// MediaItem represents a downloadable media file.
type MediaItem struct {
	URL       string `json:"url"`
	PostID    string `json:"post_id"`
	Shortcode string `json:"shortcode"`
	Type      string `json:"type"` // image, video
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Index     int    `json:"index"` // carousel index (0 for single)
}

// Progress reports scraping progress.
type Progress struct {
	Phase   string // "profile", "posts", "comments", "download", etc.
	Total   int64  // total items expected (0 if unknown)
	Current int64  // items fetched so far
	Done    bool
}

// ProgressCallback is called with progress updates.
type ProgressCallback func(Progress)

// ── JSON response types for API parsing ──

type graphQLResponse struct {
	Data   graphQLData `json:"data"`
	Status string      `json:"status"`
}

type graphQLData struct {
	User    *graphQLUser    `json:"user"`
	Hashtag *graphQLHashtag `json:"hashtag"`
}

type graphQLUser struct {
	EdgeOwnerToTimelineMedia *mediaConnection `json:"edge_owner_to_timeline_media"`
	EdgeMediaToComment       *mediaConnection `json:"edge_media_to_comment"`
}

type graphQLHashtag struct {
	EdgeHashtagToMedia *mediaConnection `json:"edge_hashtag_to_media"`
	Name               string           `json:"name"`
	MediaCount         int64            `json:"media_count"`
}

type mediaConnection struct {
	Count    int64    `json:"count"`
	PageInfo pageInfo `json:"page_info"`
	Edges    []edge   `json:"edges"`
}

type pageInfo struct {
	HasNextPage bool   `json:"has_next_page"`
	EndCursor   string `json:"end_cursor"`
}

type edge struct {
	Node mediaNode `json:"node"`
}

type mediaNode struct {
	ID                    string           `json:"id"`
	Shortcode             string           `json:"shortcode"`
	TypeName              string           `json:"__typename"`
	DisplayURL            string           `json:"display_url"`
	VideoURL              string           `json:"video_url"`
	IsVideo               bool             `json:"is_video"`
	Dimensions            dimensions       `json:"dimensions"`
	EdgeMediaToCaption    captionEdge      `json:"edge_media_to_caption"`
	EdgeLikedBy           countField       `json:"edge_liked_by"`
	EdgeMediaPreviewLike  countField       `json:"edge_media_preview_like"`
	EdgeMediaToComment    countField       `json:"edge_media_to_comment"`
	VideoViewCount        int64            `json:"video_view_count"`
	TakenAtTimestamp      int64            `json:"taken_at_timestamp"`
	Location              *locationNode    `json:"location"`
	Owner                 ownerNode        `json:"owner"`
	EdgeSidecarToChildren *mediaConnection `json:"edge_sidecar_to_children"`
	// Comment fields
	Text      string `json:"text"`
	CreatedAt int64  `json:"created_at"`
}

type dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type captionEdge struct {
	Edges []captionNode `json:"edges"`
}

type captionNode struct {
	Node struct {
		Text string `json:"text"`
	} `json:"node"`
}

type countField struct {
	Count int64 `json:"count"`
}

type locationNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ownerNode struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// profileAPIResponse is the response from web_profile_info endpoint.
type profileAPIResponse struct {
	Data struct {
		User profileUserData `json:"user"`
	} `json:"data"`
	Status string `json:"status"`
}

type profileUserData struct {
	ID                       string          `json:"id"`
	Username                 string          `json:"username"`
	FullName                 string          `json:"full_name"`
	Biography                string          `json:"biography"`
	ProfilePicURLHD          string          `json:"profile_pic_url_hd"`
	ProfilePicURL            string          `json:"profile_pic_url"`
	ExternalURL              string          `json:"external_url"`
	IsPrivate                bool            `json:"is_private"`
	IsVerified               bool            `json:"is_verified"`
	IsBusinessAccount        bool            `json:"is_business_account"`
	IsProfessionalAccount    bool            `json:"is_professional_account"`
	CategoryName             string          `json:"category_name"`
	EdgeFollowedBy           countField      `json:"edge_followed_by"`
	EdgeFollow               countField      `json:"edge_follow"`
	EdgeOwnerToTimelineMedia *mediaConnection `json:"edge_owner_to_timeline_media"`
}

// topSearchResponse is the response from the topsearch endpoint.
type topSearchResponse struct {
	Users    []searchUserWrapper    `json:"users"`
	Hashtags []searchHashtagWrapper `json:"hashtags"`
	Places   []searchPlaceWrapper   `json:"places"`
}

type searchUserWrapper struct {
	User searchUserData `json:"user"`
}

type searchUserData struct {
	PK            string `json:"pk"`
	Username      string `json:"username"`
	FullName      string `json:"full_name"`
	IsPrivate     bool   `json:"is_private"`
	IsVerified    bool   `json:"is_verified"`
	ProfilePicURL string `json:"profile_pic_url"`
	FollowerCount int64  `json:"follower_count"`
}

type searchHashtagWrapper struct {
	Hashtag searchHashtagData `json:"hashtag"`
}

type searchHashtagData struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	MediaCount int64  `json:"media_count"`
}

type searchPlaceWrapper struct {
	Place searchPlaceData `json:"place"`
}

type searchPlaceData struct {
	Location searchLocationData `json:"location"`
	Title    string             `json:"title"`
}

type searchLocationData struct {
	PK      int64   `json:"pk"`
	Address string  `json:"address"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

// postDetailResponse is the response from /?__a=1&__d=dis endpoint.
type postDetailResponse struct {
	GraphQL struct {
		ShortcodeMedia mediaNode `json:"shortcode_media"`
	} `json:"graphql"`
	Items  []postDetailItem `json:"items"`
	Status string           `json:"status"`
}

// postDetailItem is a single item from the post detail response.
type postDetailItem struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	MediaType int    `json:"media_type"` // 1=image, 2=video, 8=carousel
	Caption   *struct {
		Text string `json:"text"`
	} `json:"caption"`
	ImageVersions2 *struct {
		Candidates []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"candidates"`
	} `json:"image_versions2"`
	VideoVersions []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"video_versions"`
	LikeCount    int64 `json:"like_count"`
	CommentCount int64 `json:"comment_count"`
	ViewCount    int64 `json:"view_count"`
	TakenAt      int64 `json:"taken_at"`
	CarouselMedia []struct {
		ID             string `json:"id"`
		MediaType      int    `json:"media_type"`
		ImageVersions2 *struct {
			Candidates []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"candidates"`
		} `json:"image_versions2"`
		VideoVersions []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"video_versions"`
	} `json:"carousel_media"`
	User struct {
		PK       int64  `json:"pk"`
		Username string `json:"username"`
	} `json:"user"`
	Location *struct {
		PK   int64  `json:"pk"`
		Name string `json:"name"`
	} `json:"location"`
}

// locationFeedResponse for GraphQL location queries.
type locationFeedResponse struct {
	Data struct {
		Location *struct {
			EdgeLocationToMedia *mediaConnection `json:"edge_location_to_media"`
			Name                string           `json:"name"`
			ID                  string           `json:"id"`
		} `json:"location"`
	} `json:"data"`
	Status string `json:"status"`
}
