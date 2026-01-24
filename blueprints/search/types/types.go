// Package types contains shared data types for the search blueprint.
package types

import (
	"time"
)

// ========== Document Types ==========

// Document represents an indexed document/page.
type Document struct {
	ID          string         `json:"id"`
	URL         string         `json:"url"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Domain      string         `json:"domain"`
	Language    string         `json:"language"`
	ContentType string         `json:"content_type"`
	Favicon     string         `json:"favicon,omitempty"`
	WordCount   int            `json:"word_count"`
	CrawledAt   time.Time      `json:"crawled_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Sitelink represents a sub-page link for major sites.
type Sitelink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID         string     `json:"id"`
	URL        string     `json:"url"`
	Title      string     `json:"title"`
	Snippet    string     `json:"snippet"`
	Domain     string     `json:"domain"`
	Favicon    string     `json:"favicon,omitempty"`
	Score      float64    `json:"score"`
	Highlights []string   `json:"highlights,omitempty"`
	Sitelinks  []Sitelink `json:"sitelinks,omitempty"`
	CrawledAt  time.Time  `json:"crawled_at"`
	// Engine that provided this result
	Engine  string   `json:"engine,omitempty"`
	Engines []string `json:"engines,omitempty"`
}

// ImageResult represents an image search result.
type ImageResult struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Title        string `json:"title"`
	SourceURL    string `json:"source_url"`
	SourceDomain string `json:"source_domain"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size"`
	Format       string `json:"format"`
	Engine       string `json:"engine,omitempty"`
}

// VideoResult represents a video search result.
type VideoResult struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Duration     int       `json:"duration_seconds"`
	Channel      string    `json:"channel"`
	Views        int64     `json:"views"`
	PublishedAt  time.Time `json:"published_at"`
	EmbedURL     string    `json:"embed_url,omitempty"`
	Engine       string    `json:"engine,omitempty"`
}

// NewsResult represents a news search result.
type NewsResult struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Snippet     string    `json:"snippet"`
	Source      string    `json:"source"`
	ImageURL    string    `json:"image_url,omitempty"`
	PublishedAt time.Time `json:"published_at"`
	Engine      string    `json:"engine,omitempty"`
}

// MusicResult represents a music search result.
type MusicResult struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Artist       string `json:"artist,omitempty"`
	Album        string `json:"album,omitempty"`
	Track        string `json:"track,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	EmbedURL     string `json:"embed_url,omitempty"`
	Engine       string `json:"engine,omitempty"`
}

// FileResult represents a file/torrent search result.
type FileResult struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Content    string `json:"content,omitempty"`
	FileSize   string `json:"filesize,omitempty"`
	MagnetLink string `json:"magnetlink,omitempty"`
	Seed       int    `json:"seed,omitempty"`
	Leech      int    `json:"leech,omitempty"`
	Engine     string `json:"engine,omitempty"`
}

// ITResult represents an IT/developer search result.
type ITResult struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Type    string `json:"type,omitempty"` // repository, package, question, etc.
	Engine  string `json:"engine,omitempty"`
}

// ScienceResult represents a science/academic search result.
type ScienceResult struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	Authors     []string  `json:"authors,omitempty"`
	DOI         string    `json:"doi,omitempty"`
	Journal     string    `json:"journal,omitempty"`
	Publisher   string    `json:"publisher,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	AccessRight string    `json:"access_right,omitempty"`
	Engine      string    `json:"engine,omitempty"`
}

// SocialResult represents a social media search result.
type SocialResult struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Content      string    `json:"content,omitempty"`
	Author       string    `json:"author,omitempty"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	PublishedAt  time.Time `json:"published_at,omitempty"`
	Engine       string    `json:"engine,omitempty"`
}

// SearchOptions contains search configuration.
type SearchOptions struct {
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
	TimeRange   string `json:"time_range,omitempty"`   // day, week, month, year
	Region      string `json:"region,omitempty"`       // us, uk, de, etc.
	Language    string `json:"language,omitempty"`
	SafeSearch  string `json:"safe_search,omitempty"`  // off, moderate, strict
	Verbatim    bool   `json:"verbatim,omitempty"`
	Site        string `json:"site,omitempty"`         // site: operator
	FileType    string `json:"file_type,omitempty"`    // filetype: operator
	ExcludeSite string `json:"exclude_site,omitempty"` // -site: operator
	Lens        string `json:"lens,omitempty"`         // custom lens ID
	Refetch     bool   `json:"refetch,omitempty"`      // force bypass cache and refetch
	Version     int    `json:"version,omitempty"`      // specific cache version (0 = latest)
}

// SearchResponse represents complete search results.
type SearchResponse struct {
	Query           string          `json:"query"`
	CorrectedQuery  string          `json:"corrected_query,omitempty"`
	TotalResults    int64           `json:"total_results"`
	Results         []SearchResult  `json:"results"`
	Suggestions     []string        `json:"suggestions,omitempty"`
	InstantAnswer   *InstantAnswer  `json:"instant_answer,omitempty"`
	KnowledgePanel  *KnowledgePanel `json:"knowledge_panel,omitempty"`
	RelatedSearches []string        `json:"related_searches,omitempty"`
	SearchTimeMs    float64         `json:"search_time_ms"`
	Page            int             `json:"page"`
	PerPage         int             `json:"per_page"`
}

// ========== Suggestion Types ==========

// Suggestion represents an autocomplete suggestion.
type Suggestion struct {
	Text      string `json:"text"`
	Type      string `json:"type"` // query, history, trending
	Frequency int    `json:"frequency,omitempty"`
}

// ========== Instant Answer Types ==========

// InstantAnswer represents calculator, unit conversion, etc.
type InstantAnswer struct {
	Type   string `json:"type"` // calculator, currency, weather, definition, time, unit
	Query  string `json:"query"`
	Result string `json:"result"`
	Data   any    `json:"data,omitempty"`
}

// CalculatorResult contains calculator answer data.
type CalculatorResult struct {
	Expression string  `json:"expression"`
	Result     float64 `json:"result"`
	Formatted  string  `json:"formatted"`
}

// UnitConversionResult contains unit conversion data.
type UnitConversionResult struct {
	FromValue float64 `json:"from_value"`
	FromUnit  string  `json:"from_unit"`
	ToValue   float64 `json:"to_value"`
	ToUnit    string  `json:"to_unit"`
	Category  string  `json:"category"`
}

// CurrencyResult contains currency conversion data.
type CurrencyResult struct {
	FromAmount   float64 `json:"from_amount"`
	FromCurrency string  `json:"from_currency"`
	ToAmount     float64 `json:"to_amount"`
	ToCurrency   string  `json:"to_currency"`
	Rate         float64 `json:"rate"`
	UpdatedAt    string  `json:"updated_at"`
}

// WeatherResult contains weather data.
type WeatherResult struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Unit        string  `json:"unit"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	WindUnit    string  `json:"wind_unit"`
	Icon        string  `json:"icon"`
}

// DefinitionResult contains dictionary definition.
type DefinitionResult struct {
	Word         string   `json:"word"`
	Phonetic     string   `json:"phonetic,omitempty"`
	PartOfSpeech string   `json:"part_of_speech"`
	Definitions  []string `json:"definitions"`
	Synonyms     []string `json:"synonyms,omitempty"`
	Antonyms     []string `json:"antonyms,omitempty"`
	Examples     []string `json:"examples,omitempty"`
}

// TimeResult contains world time data.
type TimeResult struct {
	Location string `json:"location"`
	Time     string `json:"time"`
	Date     string `json:"date"`
	Timezone string `json:"timezone"`
	Offset   string `json:"offset"`
}

// ========== Knowledge Panel Types ==========

// KnowledgePanel represents entity information.
type KnowledgePanel struct {
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	Description string `json:"description"`
	Image       string `json:"image,omitempty"`
	Facts       []Fact `json:"facts,omitempty"`
	Links       []Link `json:"links,omitempty"`
	Source      string `json:"source,omitempty"`
}

// Fact represents a key-value fact in knowledge panel.
type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Link represents an external link.
type Link struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Icon  string `json:"icon,omitempty"`
}

// Entity represents a knowledge graph entity.
type Entity struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"` // person, place, organization, thing
	Description string         `json:"description"`
	Image       string         `json:"image,omitempty"`
	Facts       map[string]any `json:"facts,omitempty"`
	Links       []Link         `json:"links,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// ========== User Preference Types ==========

// UserPreference represents personalization settings.
type UserPreference struct {
	ID        string    `json:"id"`
	Domain    string    `json:"domain"`
	Action    string    `json:"action"` // upvote, downvote, block
	CreatedAt time.Time `json:"created_at"`
}

// SearchLens represents a custom search filter.
type SearchLens struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Domains     []string  `json:"domains,omitempty"`  // include domains
	Exclude     []string  `json:"exclude,omitempty"`  // exclude domains
	Keywords    []string  `json:"keywords,omitempty"` // filter keywords
	IsPublic    bool      `json:"is_public"`
	IsBuiltIn   bool      `json:"is_built_in"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchHistory represents a user's search history.
type SearchHistory struct {
	ID         string    `json:"id"`
	Query      string    `json:"query"`
	Results    int       `json:"results"`
	ClickedURL string    `json:"clicked_url,omitempty"`
	SearchedAt time.Time `json:"searched_at"`
}

// ========== Settings Types ==========

// SearchSettings represents user search settings.
type SearchSettings struct {
	SafeSearch     string `json:"safe_search"`      // off, moderate, strict
	ResultsPerPage int    `json:"results_per_page"` // 10, 20, 30, 50
	Region         string `json:"region"`
	Language       string `json:"language"`
	Theme          string `json:"theme"` // light, dark, system
	OpenInNewTab   bool   `json:"open_in_new_tab"`
	ShowThumbnails bool   `json:"show_thumbnails"`
}

// ========== Index Statistics ==========

// IndexStats contains search index statistics.
type IndexStats struct {
	TotalDocuments int64          `json:"total_documents"`
	TotalSize      int64          `json:"total_size_bytes"`
	LastUpdated    time.Time      `json:"last_updated"`
	Languages      map[string]int `json:"languages"`
	ContentTypes   map[string]int `json:"content_types"`
	TopDomains     []DomainStat   `json:"top_domains"`
}

// DomainStat represents statistics for a domain.
type DomainStat struct {
	Domain    string `json:"domain"`
	Documents int    `json:"documents"`
}
