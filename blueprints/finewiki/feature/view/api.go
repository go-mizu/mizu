// feature/view/api.go
package view

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// InfoboxItem represents a single key-value pair in an infobox.
type InfoboxItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Infobox represents a parsed infobox from Wikipedia.
type Infobox struct {
	Name  string        `json:"name,omitempty"`
	Items []InfoboxItem `json:"items,omitempty"`
}

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

	// Parsed fields (computed, not serialized by default)
	Infoboxes       []Infobox `json:"infoboxes_parsed,omitempty"`
	DateModifiedRel string    `json:"date_modified_relative,omitempty"` // "3 days ago"
	DateModifiedFmt string    `json:"date_modified_formatted,omitempty"` // "Dec 18, 2024"
}

// ParseInfoboxes decodes InfoboxesJSON into the Infoboxes slice.
func (p *Page) ParseInfoboxes() error {
	if p.InfoboxesJSON == "" || p.InfoboxesJSON == "[]" {
		p.Infoboxes = nil
		return nil
	}
	return json.Unmarshal([]byte(p.InfoboxesJSON), &p.Infoboxes)
}

// FormatDates populates DateModifiedRel and DateModifiedFmt from DateModified.
func (p *Page) FormatDates() {
	if p.DateModified == "" {
		return
	}
	t, err := time.Parse(time.RFC3339, p.DateModified)
	if err != nil {
		// Try alternative formats
		t, err = time.Parse("2006-01-02T15:04:05Z", p.DateModified)
		if err != nil {
			t, err = time.Parse("2006-01-02", p.DateModified)
			if err != nil {
				return
			}
		}
	}
	p.DateModifiedRel = FormatRelativeTime(t)
	p.DateModifiedFmt = FormatDate(t)
}

// FormatRelativeTime returns a human-readable relative time string.
func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		return "in the future"
	}

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// FormatDate returns a formatted date string like "Dec 18, 2024".
func FormatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

type Store interface {
	GetByID(ctx context.Context, id string) (*Page, error)
	GetByTitle(ctx context.Context, wikiname, title string) (*Page, error)
}

type API interface {
	ByID(ctx context.Context, id string) (*Page, error)
	ByTitle(ctx context.Context, wikiname, title string) (*Page, error)
}
