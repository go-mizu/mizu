package spotify

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	BaseURL        = "https://open.spotify.com"
	EntityTrack    = "track"
	EntityAlbum    = "album"
	EntityArtist   = "artist"
	EntityPlaylist = "playlist"
)

type ParsedRef struct {
	EntityType string
	ID         string
	URL        string
	URI        string
}

type PageMetadata struct {
	Title       string
	Description string
	ImageURL    string
	Canonical   string
	OEmbedURL   string
}

type PageData struct {
	Ref    ParsedRef
	Meta   PageMetadata
	Item   map[string]any
	RawURL string
}

type ExternalLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Track struct {
	TrackID           string    `json:"track_id"`
	Name              string    `json:"name"`
	DurationMS        int64     `json:"duration_ms"`
	TrackNumber       int       `json:"track_number"`
	DiscNumber        int       `json:"disc_number"`
	Playable          bool      `json:"playable"`
	PreviewURL        string    `json:"preview_url"`
	Playcount         int64     `json:"playcount"`
	AlbumID           string    `json:"album_id"`
	AlbumName         string    `json:"album_name"`
	CoverURL          string    `json:"cover_url"`
	ReleaseDate       string    `json:"release_date"`
	URL               string    `json:"url"`
	SpotifyURI        string    `json:"spotify_uri"`
	SourceTitle       string    `json:"source_title"`
	SourceDescription string    `json:"source_description"`
	FetchedAt         time.Time `json:"fetched_at"`
}

type Album struct {
	AlbumID           string    `json:"album_id"`
	Name              string    `json:"name"`
	AlbumType         string    `json:"album_type"`
	ReleaseDate       string    `json:"release_date"`
	TotalTracks       int       `json:"total_tracks"`
	CoverURL          string    `json:"cover_url"`
	CopyrightText     string    `json:"copyright_text"`
	URL               string    `json:"url"`
	SpotifyURI        string    `json:"spotify_uri"`
	SourceTitle       string    `json:"source_title"`
	SourceDescription string    `json:"source_description"`
	FetchedAt         time.Time `json:"fetched_at"`
}

type Artist struct {
	ArtistID          string         `json:"artist_id"`
	Name              string         `json:"name"`
	Biography         string         `json:"biography"`
	Followers         int64          `json:"followers"`
	MonthlyListeners  int64          `json:"monthly_listeners"`
	AvatarURL         string         `json:"avatar_url"`
	ExternalLinks     []ExternalLink `json:"external_links"`
	URL               string         `json:"url"`
	SpotifyURI        string         `json:"spotify_uri"`
	SourceTitle       string         `json:"source_title"`
	SourceDescription string         `json:"source_description"`
	FetchedAt         time.Time      `json:"fetched_at"`
}

type Playlist struct {
	PlaylistID        string    `json:"playlist_id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Followers         int64     `json:"followers"`
	OwnerName         string    `json:"owner_name"`
	OwnerUsername     string    `json:"owner_username"`
	OwnerURI          string    `json:"owner_uri"`
	ImageURL          string    `json:"image_url"`
	TotalItems        int       `json:"total_items"`
	NextOffset        int       `json:"next_offset"`
	URL               string    `json:"url"`
	SpotifyURI        string    `json:"spotify_uri"`
	SourceTitle       string    `json:"source_title"`
	SourceDescription string    `json:"source_description"`
	FetchedAt         time.Time `json:"fetched_at"`
}

type TrackArtist struct {
	TrackID    string
	ArtistID   string
	ArtistName string
	Ord        int
}

type AlbumArtist struct {
	AlbumID    string
	ArtistID   string
	ArtistName string
	Ord        int
}

type AlbumTrack struct {
	AlbumID string
	TrackID string
	Ord     int
}

type PlaylistTrack struct {
	PlaylistID string
	TrackID    string
	Ord        int
	AddedBy    string
}

type ArtistRelated struct {
	ArtistID        string
	RelatedArtistID string
	RelatedName     string
	Ord             int
}

type QueueItem struct {
	ID         int64
	URL        string
	EntityType string
	Priority   int
}

type DBStats struct {
	Tracks    int64
	Albums    int64
	Artists   int64
	Playlists int64
	DBSize    int64
}

func ParseRef(raw, expected string) (ParsedRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ParsedRef{}, fmt.Errorf("empty input")
	}

	entity, id := parseSpotifyURI(raw)
	if entity == "" || id == "" {
		entity, id = parseSpotifyURL(raw)
	}
	if entity == "" || id == "" {
		if expected == "" {
			return ParsedRef{}, fmt.Errorf("cannot determine spotify entity type from %q", raw)
		}
		entity = expected
		id = strings.TrimSpace(raw)
	}
	if !isSupportedEntity(entity) {
		return ParsedRef{}, fmt.Errorf("unsupported spotify entity type %q", entity)
	}
	if expected != "" && entity != expected {
		return ParsedRef{}, fmt.Errorf("expected %s, got %s", expected, entity)
	}

	return ParsedRef{
		EntityType: entity,
		ID:         id,
		URL:        NormalizeEntityURL(entity, id),
		URI:        SpotifyURI(entity, id),
	}, nil
}

func NormalizeEntityURL(entity, id string) string {
	return BaseURL + "/" + entity + "/" + id
}

func SpotifyURI(entity, id string) string {
	return "spotify:" + entity + ":" + id
}

func parseSpotifyURI(raw string) (entity, id string) {
	if !strings.HasPrefix(raw, "spotify:") {
		return "", ""
	}
	parts := strings.Split(raw, ":")
	if len(parts) < 3 {
		return "", ""
	}
	entity = parts[1]
	id = parts[2]
	if isSupportedEntity(entity) && id != "" {
		return entity, id
	}
	return "", ""
}

func parseSpotifyURL(raw string) (entity, id string) {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	if !strings.Contains(u.Host, "spotify.com") {
		return "", ""
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 {
		return "", ""
	}
	if strings.HasPrefix(parts[0], "intl-") && len(parts) >= 3 {
		parts = parts[1:]
	}
	if len(parts) < 2 {
		return "", ""
	}
	entity, id = parts[0], parts[1]
	if isSupportedEntity(entity) && id != "" {
		return entity, id
	}
	return "", ""
}

func isSupportedEntity(entity string) bool {
	switch entity {
	case EntityTrack, EntityAlbum, EntityArtist, EntityPlaylist:
		return true
	default:
		return false
	}
}
