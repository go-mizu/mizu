package firecrawl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/serp"
)

func init() {
	serp.AddProvider("firecrawl", &Provider{})
}

// Provider implements serp.Provider for Firecrawl (firecrawl.dev).
// Auth: Bearer token (fc- prefix)
// Free: 500 credits
// Note: Firecrawl is primarily a scraper, but has a search endpoint.
type Provider struct{}

func (p *Provider) Name() string { return "firecrawl" }

func (p *Provider) Search(apiKey, query string) (*serp.SearchResult, error) {
	body, _ := json.Marshal(map[string]any{
		"query": query,
		"limit": 10,
	})
	req, _ := http.NewRequest("POST", "https://api.firecrawl.dev/v1/search", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("firecrawl: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		Success bool `json:"success"`
		Data    []struct {
			URL         string            `json:"url"`
			Title       string            `json:"title"`
			Description string            `json:"description"`
			Markdown    string            `json:"markdown"`
			Metadata    map[string]string `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("firecrawl: decode: %w", err)
	}

	result := &serp.SearchResult{}
	for _, r := range raw.Data {
		snippet := r.Description
		if snippet == "" && len(r.Markdown) > 200 {
			snippet = r.Markdown[:200]
		}
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.URL, "snippet": snippet,
		})
	}
	return result, nil
}
