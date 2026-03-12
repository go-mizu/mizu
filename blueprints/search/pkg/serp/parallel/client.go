package parallel

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
	serp.AddProvider("parallel", &Provider{})
}

// Provider implements serp.Provider for Parallel.ai Search API.
// Auth: x-api-key header
// Free tier available
type Provider struct{}

func (p *Provider) Name() string { return "parallel" }

func (p *Provider) Search(apiKey, query string) (*serp.SearchResult, error) {
	body, _ := json.Marshal(map[string]any{
		"objective":   query,
		"max_results": 10,
	})
	req, _ := http.NewRequest("POST", "https://api.parallel.ai/v1beta/search", bytes.NewReader(body))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("parallel-beta", "search-extract-2025-10-10")

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("parallel: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		SearchID string `json:"search_id"`
		Results  []struct {
			URL         string   `json:"url"`
			Title       string   `json:"title"`
			PublishDate string   `json:"publish_date"`
			Excerpts    []string `json:"excerpts"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parallel: decode: %w", err)
	}

	result := &serp.SearchResult{}
	for _, r := range raw.Results {
		snippet := ""
		if len(r.Excerpts) > 0 {
			snippet = r.Excerpts[0]
		}
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.URL, "snippet": snippet,
		})
	}
	return result, nil
}
