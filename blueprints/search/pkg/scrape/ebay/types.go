// Package ebay scrapes public eBay item and search pages into a local DuckDB database.
package ebay

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	EntityItem   = "item"
	EntitySearch = "search"
)

var itemIDPattern = regexp.MustCompile(`/itm/(?:[^/?#]*/)?([0-9]{9,15})`)

// Item represents a single eBay item page.
type Item struct {
	ItemID              string            `json:"item_id"`
	Title               string            `json:"title"`
	Subtitle            string            `json:"subtitle"`
	Description         string            `json:"description"`
	Price               float64           `json:"price"`
	Currency            string            `json:"currency"`
	OriginalPrice       float64           `json:"original_price"`
	Condition           string            `json:"condition"`
	Availability        string            `json:"availability"`
	SellerName          string            `json:"seller_name"`
	SellerURL           string            `json:"seller_url"`
	SellerFeedbackScore int64             `json:"seller_feedback_score"`
	SellerPositivePct   float64           `json:"seller_positive_pct"`
	ShippingText        string            `json:"shipping_text"`
	ReturnsText         string            `json:"returns_text"`
	Location            string            `json:"location"`
	ImageURLs           []string          `json:"image_urls"`
	CategoryPath        []string          `json:"category_path"`
	ItemSpecifics       map[string]string `json:"item_specifics"`
	RawJSONLD           string            `json:"raw_jsonld"`
	URL                 string            `json:"url"`
	FetchedAt           time.Time         `json:"fetched_at"`
}

// SearchResult represents one fetched eBay search result page.
type SearchResult struct {
	SearchID      string    `json:"search_id"`
	Query         string    `json:"query"`
	Page          int       `json:"page"`
	TotalResults  string    `json:"total_results"`
	ResultItemIDs []string  `json:"result_item_ids"`
	URL           string    `json:"url"`
	NextPageURL   string    `json:"next_page_url"`
	FetchedAt     time.Time `json:"fetched_at"`
}

// QueueItem is a row from the crawl queue.
type QueueItem struct {
	ID         int64  `json:"id"`
	URL        string `json:"url"`
	EntityType string `json:"entity_type"`
	Priority   int    `json:"priority"`
}

// JobRecord holds one state job row for display.
type JobRecord struct {
	JobID       string    `json:"job_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

// DBStats holds row counts per table.
type DBStats struct {
	Items       int64
	SearchPages int64
	DBSize      int64
}

// ExtractItemID extracts the eBay item ID from a URL or bare item ID.
func ExtractItemID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if isDigits(s) && len(s) >= 9 && len(s) <= 15 {
		return s
	}
	m := itemIDPattern.FindStringSubmatch(s)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// NormalizeItemURL accepts a bare eBay item ID or URL and returns a canonical URL.
func NormalizeItemURL(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if id := ExtractItemID(s); id != "" {
			return BaseURL + "/itm/" + id
		}
		return s
	}
	if id := ExtractItemID(s); id != "" {
		return BaseURL + "/itm/" + id
	}
	return s
}

// SearchURL builds the eBay search results URL for a query and page.
func SearchURL(query string, page int) string {
	if page <= 0 {
		page = 1
	}
	v := url.Values{}
	v.Set("_nkw", query)
	v.Set("_pgn", strconv.Itoa(page))
	return BaseURL + "/sch/i.html?" + v.Encode()
}

// ParseSearchURL extracts the query and page from an eBay search URL.
func ParseSearchURL(raw string) (query string, page int, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", 0, err
	}
	query = strings.TrimSpace(u.Query().Get("_nkw"))
	if query == "" {
		return "", 0, fmt.Errorf("missing _nkw query parameter")
	}
	page = 1
	if p := strings.TrimSpace(u.Query().Get("_pgn")); p != "" {
		if n, convErr := strconv.Atoi(p); convErr == nil && n > 0 {
			page = n
		}
	}
	return query, page, nil
}

func searchID(query string, page int) string {
	sum := md5.Sum([]byte(query + ":" + strconv.Itoa(page)))
	return fmt.Sprintf("%x", sum)
}

func absoluteURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	u, err := url.Parse(href)
	if err == nil && u.IsAbs() {
		return u.String()
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(ref).String()
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
