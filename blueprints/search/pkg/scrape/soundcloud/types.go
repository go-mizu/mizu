package soundcloud

import "time"

type User struct {
	UserID             int64     `json:"user_id"`
	Username           string    `json:"username"`
	FullName           string    `json:"full_name"`
	Description        string    `json:"description"`
	AvatarURL          string    `json:"avatar_url"`
	City               string    `json:"city"`
	CountryCode        string    `json:"country_code"`
	FollowersCount     int       `json:"followers_count"`
	FollowingsCount    int       `json:"followings_count"`
	TrackCount         int       `json:"track_count"`
	PlaylistCount      int       `json:"playlist_count"`
	LikesCount         int       `json:"likes_count"`
	PlaylistLikesCount int       `json:"playlist_likes_count"`
	Verified           bool      `json:"verified"`
	URL                string    `json:"url"`
	CreatedAt          time.Time `json:"created_at"`
	FetchedAt          time.Time `json:"fetched_at"`
}

type Track struct {
	TrackID       int64     `json:"track_id"`
	UserID        int64     `json:"user_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Genre         string    `json:"genre"`
	TagList       string    `json:"tag_list"`
	ArtworkURL    string    `json:"artwork_url"`
	WaveformURL   string    `json:"waveform_url"`
	LabelName     string    `json:"label_name"`
	License       string    `json:"license"`
	DurationMS    int64     `json:"duration_ms"`
	PlaybackCount int64     `json:"playback_count"`
	LikesCount    int       `json:"likes_count"`
	CommentCount  int       `json:"comment_count"`
	DownloadCount int       `json:"download_count"`
	RepostsCount  int       `json:"reposts_count"`
	Downloadable  bool      `json:"downloadable"`
	Streamable    bool      `json:"streamable"`
	ReleaseDate   time.Time `json:"release_date"`
	CreatedAt     time.Time `json:"created_at"`
	URL           string    `json:"url"`
	FetchedAt     time.Time `json:"fetched_at"`
}

type Playlist struct {
	PlaylistID   int64     `json:"playlist_id"`
	UserID       int64     `json:"user_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ArtworkURL   string    `json:"artwork_url"`
	TrackCount   int       `json:"track_count"`
	DurationMS   int64     `json:"duration_ms"`
	LikesCount   int       `json:"likes_count"`
	RepostsCount int       `json:"reposts_count"`
	SetType      string    `json:"set_type"`
	IsAlbum      bool      `json:"is_album"`
	CreatedAt    time.Time `json:"created_at"`
	PublishedAt  time.Time `json:"published_at"`
	URL          string    `json:"url"`
	FetchedAt    time.Time `json:"fetched_at"`
}

type PlaylistTrack struct {
	PlaylistID int64  `json:"playlist_id"`
	TrackID    int64  `json:"track_id"`
	Position   int    `json:"position"`
	TrackURL   string `json:"track_url"`
}

type Comment struct {
	CommentID string    `json:"comment_id"`
	TrackID   int64     `json:"track_id"`
	UserName  string    `json:"user_name"`
	UserURL   string    `json:"user_url"`
	Body      string    `json:"body"`
	PostedAt  time.Time `json:"posted_at"`
	FetchedAt time.Time `json:"fetched_at"`
}

type SearchResult struct {
	SearchID  string    `json:"search_id"`
	Query     string    `json:"query"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	FetchedAt time.Time `json:"fetched_at"`
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
