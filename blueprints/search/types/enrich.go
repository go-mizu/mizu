// Package types contains shared data types for the search blueprint.
package types

import "time"

// EnrichmentType represents the type of enrichment result.
type EnrichmentType int

const (
	EnrichTypeResult EnrichmentType = 0
)

// EnrichmentRequest represents an enrichment API request.
type EnrichmentRequest struct {
	Query string `json:"q"`
	Limit int    `json:"limit,omitempty"`
}

// EnrichmentResponse represents an enrichment API response.
type EnrichmentResponse struct {
	Meta EnrichmentMeta     `json:"meta"`
	Data []EnrichmentResult `json:"data"`
}

// EnrichmentMeta contains request metadata.
type EnrichmentMeta struct {
	ID   string `json:"id"`
	Node string `json:"node"`
	Ms   int64  `json:"ms"`
}

// EnrichmentResult represents a single enrichment result.
type EnrichmentResult struct {
	Type      EnrichmentType `json:"t"`
	Rank      int            `json:"rank"`
	URL       string         `json:"url"`
	Title     string         `json:"title"`
	Snippet   string         `json:"snippet,omitempty"`
	Published *time.Time     `json:"published,omitempty"`
}

// SmallWebEntry represents an entry in the small web index.
type SmallWebEntry struct {
	ID          int64     `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Snippet     string    `json:"snippet,omitempty"`
	SourceType  string    `json:"source_type"` // blog, forum, discussion
	Domain      string    `json:"domain"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	IndexedAt   time.Time `json:"indexed_at"`
}
