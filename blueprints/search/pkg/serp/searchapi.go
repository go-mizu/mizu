package serp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SearchAPIProvider implements Provider for searchapi.io.
// API: GET https://www.searchapi.io/api/v1/search?engine=google&api_key=KEY&q=QUERY
// Free: 100 searches (one-time)
type SearchAPIProvider struct{}

func (p *SearchAPIProvider) Name() string { return "searchapi" }

func (p *SearchAPIProvider) Search(apiKey, query string) (*SearchResult, error) {
	u := fmt.Sprintf("https://www.searchapi.io/api/v1/search?engine=google&api_key=%s&q=%s",
		url.QueryEscape(apiKey), url.QueryEscape(query))
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("searchapi: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("searchapi: decode: %w", err)
	}
	if raw.Error != "" {
		return nil, fmt.Errorf("searchapi: %s", raw.Error)
	}

	result := &SearchResult{}
	for _, r := range raw.OrganicResults {
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.Link, "snippet": r.Snippet,
		})
	}
	return result, nil
}
