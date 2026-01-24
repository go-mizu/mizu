package engine

import (
	"time"
)

// Category represents a search category.
type Category string

// Search categories supported by SearXNG.
const (
	CategoryGeneral Category = "general"
	CategoryImages  Category = "images"
	CategoryVideos  Category = "videos"
	CategoryNews    Category = "news"
	CategoryMusic   Category = "music"
	CategoryFiles   Category = "files"
	CategoryIT      Category = "it"
	CategoryScience Category = "science"
	CategorySocial  Category = "social media"
	CategoryMaps    Category = "map"
)

// AllCategories returns all supported categories.
func AllCategories() []Category {
	return []Category{
		CategoryGeneral,
		CategoryImages,
		CategoryVideos,
		CategoryNews,
		CategoryMusic,
		CategoryFiles,
		CategoryIT,
		CategoryScience,
		CategorySocial,
		CategoryMaps,
	}
}

// SearchOptions contains search configuration.
type SearchOptions struct {
	Category   Category `json:"category"`
	Page       int      `json:"page"`
	PerPage    int      `json:"per_page"`
	TimeRange  string   `json:"time_range,omitempty"`  // day, week, month, year
	Language   string   `json:"language,omitempty"`
	Region     string   `json:"region,omitempty"`
	SafeSearch int      `json:"safe_search,omitempty"` // 0=off, 1=moderate, 2=strict
}

// SearchResponse represents the complete search response.
type SearchResponse struct {
	Query           string   `json:"query"`
	CorrectedQuery  string   `json:"corrected_query,omitempty"`
	TotalResults    int64    `json:"total_results"`
	Results         []Result `json:"results"`
	Suggestions     []string `json:"suggestions,omitempty"`
	Infoboxes       []Infobox `json:"infoboxes,omitempty"`
	SearchTimeMs    float64  `json:"search_time_ms"`
	Page            int      `json:"page"`
	PerPage         int      `json:"per_page"`
}

// Result represents a generic search result.
// It contains all possible fields from different categories.
type Result struct {
	// Common fields
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	Category    Category  `json:"category"`
	Engine      string    `json:"engine"`
	Engines     []string  `json:"engines,omitempty"`
	Score       float64   `json:"score"`
	ParsedURL   []string  `json:"parsed_url,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`

	// Image-specific fields
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	ImageURL     string `json:"img_src,omitempty"`
	ImgFormat    string `json:"img_format,omitempty"`
	Source       string `json:"source,omitempty"`
	Resolution   string `json:"resolution,omitempty"`

	// Video-specific fields
	Duration    string `json:"duration,omitempty"`
	EmbedURL    string `json:"embed_url,omitempty"`
	IFrameSrc   string `json:"iframe_src,omitempty"`
	Length      string `json:"length,omitempty"`

	// Music-specific fields
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	Track       string `json:"track,omitempty"`

	// File-specific fields
	FileSize    string `json:"filesize,omitempty"`
	MagnetLink  string `json:"magnetlink,omitempty"`
	Seed        int    `json:"seed,omitempty"`
	Leech       int    `json:"leech,omitempty"`

	// Science/IT-specific fields
	DOI         string   `json:"doi,omitempty"`
	ISSN        string   `json:"issn,omitempty"`
	ISBN        string   `json:"isbn,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	Journal     string   `json:"journal,omitempty"`
	Type        string   `json:"type,omitempty"`
	AccessRight string   `json:"access_right,omitempty"`

	// Map-specific fields
	Latitude    float64  `json:"latitude,omitempty"`
	Longitude   float64  `json:"longitude,omitempty"`
	BoundingBox []string `json:"boundingbox,omitempty"`
	Geojson     any      `json:"geojson,omitempty"`
	Address     *Address `json:"address,omitempty"`
	OSMType     string   `json:"osm_type,omitempty"`
	OSMID       string   `json:"osm_id,omitempty"`
}

// Address represents a geographic address.
type Address struct {
	Name        string `json:"name,omitempty"`
	Road        string `json:"road,omitempty"`
	Locality    string `json:"locality,omitempty"`
	PostalCode  string `json:"postcode,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	HouseNumber string `json:"house_number,omitempty"`
}

// Infobox represents a knowledge panel/infobox from SearXNG.
type Infobox struct {
	ID          string      `json:"id,omitempty"`
	Infobox     string      `json:"infobox"`
	Content     string      `json:"content,omitempty"`
	ImageURL    string      `json:"img_src,omitempty"`
	URLs        []InfoboxURL `json:"urls,omitempty"`
	Attributes  []InfoboxAttribute `json:"attributes,omitempty"`
	Engine      string      `json:"engine"`
}

// InfoboxURL represents a link in an infobox.
type InfoboxURL struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// InfoboxAttribute represents a key-value attribute in an infobox.
type InfoboxAttribute struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ImageResult is a convenience type for image search results.
type ImageResult struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	ThumbnailURL string `json:"thumbnail_url"`
	ImageURL     string `json:"img_src"`
	Source       string `json:"source"`
	SourceDomain string `json:"source_domain"`
	Resolution   string `json:"resolution"`
	Format       string `json:"format"`
	Engine       string `json:"engine"`
	Score        float64 `json:"score"`
}

// VideoResult is a convenience type for video search results.
type VideoResult struct {
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     string    `json:"duration"`
	EmbedURL     string    `json:"embed_url"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
	Engine       string    `json:"engine"`
	Score        float64   `json:"score"`
}

// NewsResult is a convenience type for news search results.
type NewsResult struct {
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	Source       string    `json:"source"`
	ImageURL     string    `json:"image_url,omitempty"`
	PublishedAt  time.Time `json:"published_at"`
	Engine       string    `json:"engine"`
	Score        float64   `json:"score"`
}

// MusicResult is a convenience type for music search results.
type MusicResult struct {
	URL          string `json:"url"`
	Title        string `json:"title"`
	Artist       string `json:"artist,omitempty"`
	Album        string `json:"album,omitempty"`
	Track        string `json:"track,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	EmbedURL     string `json:"embed_url,omitempty"`
	Engine       string `json:"engine"`
	Score        float64 `json:"score"`
}

// FileResult is a convenience type for file search results.
type FileResult struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Content    string `json:"content,omitempty"`
	FileSize   string `json:"filesize,omitempty"`
	MagnetLink string `json:"magnetlink,omitempty"`
	Seed       int    `json:"seed,omitempty"`
	Leech      int    `json:"leech,omitempty"`
	Engine     string `json:"engine"`
	Score      float64 `json:"score"`
}

// ITResult is a convenience type for IT/developer search results.
type ITResult struct {
	URL       string   `json:"url"`
	Title     string   `json:"title"`
	Content   string   `json:"content,omitempty"`
	Type      string   `json:"type,omitempty"` // repository, package, question, etc.
	Engine    string   `json:"engine"`
	Score     float64  `json:"score"`
}

// ScienceResult is a convenience type for science/academic search results.
type ScienceResult struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Content     string   `json:"content,omitempty"`
	Authors     []string `json:"authors,omitempty"`
	DOI         string   `json:"doi,omitempty"`
	Journal     string   `json:"journal,omitempty"`
	Publisher   string   `json:"publisher,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	AccessRight string   `json:"access_right,omitempty"`
	Engine      string   `json:"engine"`
	Score       float64  `json:"score"`
}

// SocialResult is a convenience type for social media search results.
type SocialResult struct {
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Content      string    `json:"content,omitempty"`
	Author       string    `json:"author,omitempty"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
	Engine       string    `json:"engine"`
	Score        float64   `json:"score"`
}

// MapResult is a convenience type for map search results.
type MapResult struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Address     *Address `json:"address,omitempty"`
	Latitude    float64  `json:"latitude,omitempty"`
	Longitude   float64  `json:"longitude,omitempty"`
	BoundingBox []string `json:"boundingbox,omitempty"`
	OSMType     string   `json:"osm_type,omitempty"`
	OSMID       string   `json:"osm_id,omitempty"`
	Engine      string   `json:"engine"`
	Score       float64  `json:"score"`
}

// ToImageResult converts a generic Result to ImageResult.
func (r *Result) ToImageResult() ImageResult {
	return ImageResult{
		URL:          r.URL,
		Title:        r.Title,
		ThumbnailURL: r.ThumbnailURL,
		ImageURL:     r.ImageURL,
		Source:       r.Source,
		Resolution:   r.Resolution,
		Format:       r.ImgFormat,
		Engine:       r.Engine,
		Score:        r.Score,
	}
}

// ToVideoResult converts a generic Result to VideoResult.
func (r *Result) ToVideoResult() VideoResult {
	return VideoResult{
		URL:          r.URL,
		Title:        r.Title,
		Content:      r.Content,
		ThumbnailURL: r.ThumbnailURL,
		Duration:     r.Duration,
		EmbedURL:     r.EmbedURL,
		PublishedAt:  r.PublishedAt,
		Engine:       r.Engine,
		Score:        r.Score,
	}
}

// ToNewsResult converts a generic Result to NewsResult.
func (r *Result) ToNewsResult() NewsResult {
	return NewsResult{
		URL:         r.URL,
		Title:       r.Title,
		Content:     r.Content,
		Source:      r.Source,
		ImageURL:    r.ThumbnailURL,
		PublishedAt: r.PublishedAt,
		Engine:      r.Engine,
		Score:       r.Score,
	}
}

// ToMusicResult converts a generic Result to MusicResult.
func (r *Result) ToMusicResult() MusicResult {
	return MusicResult{
		URL:          r.URL,
		Title:        r.Title,
		Artist:       r.Artist,
		Album:        r.Album,
		Track:        r.Track,
		ThumbnailURL: r.ThumbnailURL,
		EmbedURL:     r.EmbedURL,
		Engine:       r.Engine,
		Score:        r.Score,
	}
}

// ToFileResult converts a generic Result to FileResult.
func (r *Result) ToFileResult() FileResult {
	return FileResult{
		URL:        r.URL,
		Title:      r.Title,
		Content:    r.Content,
		FileSize:   r.FileSize,
		MagnetLink: r.MagnetLink,
		Seed:       r.Seed,
		Leech:      r.Leech,
		Engine:     r.Engine,
		Score:      r.Score,
	}
}

// ToITResult converts a generic Result to ITResult.
func (r *Result) ToITResult() ITResult {
	return ITResult{
		URL:     r.URL,
		Title:   r.Title,
		Content: r.Content,
		Type:    r.Type,
		Engine:  r.Engine,
		Score:   r.Score,
	}
}

// ToScienceResult converts a generic Result to ScienceResult.
func (r *Result) ToScienceResult() ScienceResult {
	return ScienceResult{
		URL:         r.URL,
		Title:       r.Title,
		Content:     r.Content,
		Authors:     r.Authors,
		DOI:         r.DOI,
		Journal:     r.Journal,
		Publisher:   r.Publisher,
		PublishedAt: r.PublishedAt,
		AccessRight: r.AccessRight,
		Engine:      r.Engine,
		Score:       r.Score,
	}
}

// ToSocialResult converts a generic Result to SocialResult.
func (r *Result) ToSocialResult() SocialResult {
	return SocialResult{
		URL:          r.URL,
		Title:        r.Title,
		Content:      r.Content,
		ThumbnailURL: r.ThumbnailURL,
		PublishedAt:  r.PublishedAt,
		Engine:       r.Engine,
		Score:        r.Score,
	}
}

// ToMapResult converts a generic Result to MapResult.
func (r *Result) ToMapResult() MapResult {
	return MapResult{
		URL:         r.URL,
		Title:       r.Title,
		Address:     r.Address,
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
		BoundingBox: r.BoundingBox,
		OSMType:     r.OSMType,
		OSMID:       r.OSMID,
		Engine:      r.Engine,
		Score:       r.Score,
	}
}
