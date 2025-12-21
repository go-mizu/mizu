// feature/view/api.go
package view

import "context"

// Page represents a single wiki page record.
// This is a projection of dataset fields, kept as plain strings/ints for speed.
// DateModified remains a string to avoid parse cost and timezone ambiguity.
type Page struct {
	ID           string `json:"id,omitempty"`           // dataset id, for example "enwiki/32552979"
	WikiName     string `json:"wikiname,omitempty"`     // for example "enwiki"
	PageID       int64  `json:"page_id,omitempty"`      // original MediaWiki page id
	Title        string `json:"title,omitempty"`
	URL          string `json:"url,omitempty"`
	DateModified string `json:"date_modified,omitempty"` // dataset-provided timestamp string
	InLanguage   string `json:"in_language,omitempty"`   // for example "en"`

	// Text is the primary MVP body used for rendering.
	Text string `json:"text,omitempty"`

	// Optional fields, safe to omit for MVP but useful for later upgrades.
	WikidataID    string `json:"wikidata_id,omitempty"`
	BytesHTML     int64  `json:"bytes_html,omitempty"`
	HasMath       bool   `json:"has_math,omitempty"`
	WikiText      string `json:"wikitext,omitempty"`
	Version       string `json:"version,omitempty"`
	InfoboxesJSON string `json:"infoboxes,omitempty"` // raw JSON string
}

type Store interface {
	GetByID(ctx context.Context, id string) (*Page, error)
	GetByTitle(ctx context.Context, wikiname, title string) (*Page, error)
}

type API interface {
	ByID(ctx context.Context, id string) (*Page, error)
	ByTitle(ctx context.Context, wikiname, title string) (*Page, error)
}
