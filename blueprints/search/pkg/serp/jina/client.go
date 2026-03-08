package jina

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
	serp.AddProvider("jina", &Provider{})
}

// Provider implements serp.Provider for Jina AI Search (s.jina.ai).
// Auth: Bearer token (jina_ prefix)
// Free: 1M tokens (no credit card)
type Provider struct{}

func (p *Provider) Name() string { return "jina" }

func (p *Provider) Search(apiKey, query string) (*serp.SearchResult, error) {
	body, _ := json.Marshal(map[string]any{"q": query, "num": 10})
	req, _ := http.NewRequest("POST", "https://s.jina.ai/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jina: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		Code int `json:"code"`
		Data []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Content     string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("jina: decode: %w", err)
	}

	result := &serp.SearchResult{}
	for _, r := range raw.Data {
		snippet := r.Description
		if snippet == "" && len(r.Content) > 200 {
			snippet = r.Content[:200]
		} else if snippet == "" {
			snippet = r.Content
		}
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.URL, "snippet": snippet,
		})
	}
	return result, nil
}
