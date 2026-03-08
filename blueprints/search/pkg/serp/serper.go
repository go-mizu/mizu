package serp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SerperProvider implements Provider for serper.dev.
// API: POST https://google.serper.dev/search
// Auth: X-API-KEY header
// Free: 2,500 queries (one-time)
type SerperProvider struct{}

func (p *SerperProvider) Name() string { return "serper" }

func (p *SerperProvider) Search(apiKey, query string) (*SearchResult, error) {
	body, _ := json.Marshal(map[string]string{"q": query})
	req, _ := http.NewRequest("POST", "https://google.serper.dev/search", bytes.NewReader(body))
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("serper: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
		Message string `json:"message"` // error message
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("serper: decode: %w", err)
	}
	if raw.Message != "" {
		return nil, fmt.Errorf("serper: %s", raw.Message)
	}

	result := &SearchResult{}
	for _, r := range raw.Organic {
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.Link, "snippet": r.Snippet,
		})
	}
	return result, nil
}
