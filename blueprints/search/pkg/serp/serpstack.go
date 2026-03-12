package serp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SerpStackProvider implements Provider for serpstack.com.
// API: GET http://api.serpstack.com/search?access_key=KEY&query=QUERY
// Free: 100 searches/month (HTTP only on free tier)
type SerpStackProvider struct{}

func (p *SerpStackProvider) Name() string { return "serpstack" }

func (p *SerpStackProvider) Search(apiKey, query string) (*SearchResult, error) {
	// Free tier is HTTP only
	u := fmt.Sprintf("http://api.serpstack.com/search?access_key=%s&query=%s",
		url.QueryEscape(apiKey), url.QueryEscape(query))
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("serpstack: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
		Success *bool `json:"success"`
		Error   *struct {
			Code int    `json:"code"`
			Type string `json:"type"`
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("serpstack: decode: %w", err)
	}
	if raw.Error != nil {
		return nil, fmt.Errorf("serpstack: %s (%s)", raw.Error.Info, raw.Error.Type)
	}

	result := &SearchResult{}
	for _, r := range raw.OrganicResults {
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.URL, "snippet": r.Snippet,
		})
	}
	return result, nil
}
