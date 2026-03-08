package serp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SerplyProvider implements Provider for serply.io.
// API: GET https://api.serply.io/v1/search/q=QUERY
// Auth: X-Api-Key header
// Free: monthly credits
type SerplyProvider struct{}

func (p *SerplyProvider) Name() string { return "serply" }

func (p *SerplyProvider) Search(apiKey, query string) (*SearchResult, error) {
	u := fmt.Sprintf("https://api.serply.io/v1/search/q=%s", url.QueryEscape(query))
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("serply: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		Results []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Description string `json:"description"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("serply: decode: %w", err)
	}

	result := &SearchResult{}
	for _, r := range raw.Results {
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.Link, "snippet": r.Description,
		})
	}
	return result, nil
}
