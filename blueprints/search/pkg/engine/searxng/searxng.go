// Package searxng provides a SearXNG search engine implementation.
package searxng

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine"
)

// Engine implements the engine.Engine interface using SearXNG.
type Engine struct {
	client *Client
}

// New creates a new SearXNG engine.
func New(baseURL string) *Engine {
	return &Engine{
		client: NewClient(baseURL),
	}
}

// NewWithClient creates a new SearXNG engine with a custom client.
func NewWithClient(client *Client) *Engine {
	return &Engine{
		client: client,
	}
}

// Name returns the engine name.
func (e *Engine) Name() string {
	return "searxng"
}

// Categories returns the list of supported categories.
func (e *Engine) Categories() []engine.Category {
	return engine.AllCategories()
}

// Search performs a search using SearXNG.
func (e *Engine) Search(ctx context.Context, query string, opts engine.SearchOptions) (*engine.SearchResponse, error) {
	start := time.Now()

	// Build request
	req := SearchRequest{
		Query:      query,
		Categories: string(opts.Category),
		Language:   opts.Language,
		PageNo:     opts.Page,
		TimeRange:  opts.TimeRange,
		SafeSearch: opts.SafeSearch,
	}

	// Execute search
	resp, err := e.client.Search(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert to engine response
	results := make([]engine.Result, 0, len(resp.Results))
	for _, r := range resp.Results {
		result := convertResult(r)
		results = append(results, result)
	}

	// Convert infoboxes
	infoboxes := make([]engine.Infobox, 0, len(resp.Infoboxes))
	for _, ib := range resp.Infoboxes {
		infoboxes = append(infoboxes, convertInfobox(ib))
	}

	// Build corrected query from corrections
	var correctedQuery string
	if len(resp.Corrections) > 0 {
		correctedQuery = resp.Corrections[0]
	}

	// SearXNG often returns 0 for number_of_results when aggregating from multiple engines.
	// Fall back to results count when this happens, and estimate total based on pagination.
	totalResults := resp.NumberOfResults
	if totalResults == 0 && len(results) > 0 {
		// Estimate total: if we got a full page, assume there are more pages
		// Otherwise, this is likely the last/only page
		if len(results) >= opts.PerPage {
			// Estimate conservatively - at least 10 pages worth
			totalResults = int64(opts.PerPage * 10)
		} else {
			// We got a partial page, so this is likely the total
			totalResults = int64((opts.Page-1)*opts.PerPage + len(results))
		}
	}

	return &engine.SearchResponse{
		Query:          query,
		CorrectedQuery: correctedQuery,
		TotalResults:   totalResults,
		Results:        results,
		Suggestions:    resp.Suggestions,
		Infoboxes:      infoboxes,
		SearchTimeMs:   float64(time.Since(start).Milliseconds()),
		Page:           opts.Page,
		PerPage:        opts.PerPage,
	}, nil
}

// convertResult converts a raw SearXNG result to an engine.Result.
func convertResult(r RawResult) engine.Result {
	result := engine.Result{
		URL:       r.URL,
		Title:     r.Title,
		Content:   r.Content,
		Category:  engine.Category(r.Category),
		Engine:    r.Engine,
		Engines:   r.Engines,
		Score:     r.Score,
		ParsedURL: r.ParsedURL,

		// Image fields
		ThumbnailURL: firstNonEmpty(r.Thumbnail, r.ThumbnailSrc),
		ImageURL:     r.ImgSrc,
		ImgFormat:    r.ImgFormat,
		Source:       extractDomain(r.Source, r.URL),
		Resolution:   r.Resolution,

		// Video fields
		Duration:  firstNonEmpty(r.Duration, anyToString(r.Length)),
		EmbedURL:  firstNonEmpty(r.EmbedURL, r.IframeSrc),
		IFrameSrc: r.IframeSrc,

		// Music fields
		Artist: r.Artist,
		Album:  r.Album,
		Track:  r.Track,

		// File fields
		FileSize:   r.FileSize,
		MagnetLink: r.MagnetLink,
		Seed:       r.Seed,
		Leech:      r.Leech,

		// Science fields
		DOI:         r.DOI,
		ISSN:        anyToString(r.ISSN),
		ISBN:        anyToString(r.ISBN),
		Authors:     anyToStringSlice(r.Authors),
		Publisher:   r.Publisher,
		Journal:     r.Journal,
		Type:        r.Type,
		AccessRight: r.AccessRight,

		// Map fields
		Latitude:    r.Latitude,
		Longitude:   r.Longitude,
		BoundingBox: r.BoundingBox,
		Geojson:     r.Geojson,
		OSMType:     r.OSMType,
		OSMID:       r.OSMID,
	}

	// Convert address
	if r.Address != nil {
		result.Address = &engine.Address{
			Name:        r.Address.Name,
			Road:        r.Address.Road,
			Locality:    r.Address.Locality,
			PostalCode:  r.Address.PostCode,
			Country:     r.Address.Country,
			CountryCode: r.Address.CountryCode,
			HouseNumber: r.Address.HouseNumber,
		}
	}

	// Parse published date
	if r.PublishedDate != "" {
		if t, err := parseDate(r.PublishedDate); err == nil {
			result.PublishedAt = t
		}
	} else if r.PubDate != "" {
		if t, err := parseDate(r.PubDate); err == nil {
			result.PublishedAt = t
		}
	}

	return result
}

// convertInfobox converts a raw SearXNG infobox to an engine.Infobox.
func convertInfobox(ib RawInfobox) engine.Infobox {
	urls := make([]engine.InfoboxURL, len(ib.URLs))
	for i, u := range ib.URLs {
		urls[i] = engine.InfoboxURL{
			Title: u.Title,
			URL:   u.URL,
		}
	}

	attrs := make([]engine.InfoboxAttribute, len(ib.Attributes))
	for i, a := range ib.Attributes {
		attrs[i] = engine.InfoboxAttribute{
			Label: a.Label,
			Value: a.Value,
		}
	}

	return engine.Infobox{
		ID:         ib.ID,
		Infobox:    ib.Infobox,
		Content:    ib.Content,
		ImageURL:   ib.ImgSrc,
		URLs:       urls,
		Attributes: attrs,
		Engine:     ib.Engine,
	}
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// anyToString converts an any value to a string.
func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []any:
		if len(val) > 0 {
			return anyToString(val[0])
		}
		return ""
	case []string:
		if len(val) > 0 {
			return val[0]
		}
		return ""
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// anyToStringSlice converts an any value to a string slice.
func anyToStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			switch s := item.(type) {
			case string:
				result = append(result, s)
			case map[string]any:
				// Handle author objects with "name" field
				if name, ok := s["name"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return result
	default:
		return nil
	}
}

// extractDomain extracts the domain from a URL or returns the source.
func extractDomain(source, rawURL string) string {
	if source != "" {
		return source
	}
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
}

// parseDate attempts to parse a date string in various formats.
func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan 2, 2006",
		"January 2, 2006",
		"02 Jan 2006",
		"2 Jan 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil
}

// Healthz checks if SearXNG is healthy.
func (e *Engine) Healthz(ctx context.Context) error {
	return e.client.Healthz(ctx)
}
