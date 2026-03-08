package serp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ZenserpProvider implements Provider for zenserp.com.
// API: GET https://app.zenserp.com/api/v2/search?apikey=KEY&q=QUERY
// Free: 50 searches/month
type ZenserpProvider struct{}

func (p *ZenserpProvider) Name() string { return "zenserp" }

func (p *ZenserpProvider) Search(apiKey, query string) (*SearchResult, error) {
	u := fmt.Sprintf("https://app.zenserp.com/api/v2/search?apikey=%s&q=%s",
		url.QueryEscape(apiKey), url.QueryEscape(query))
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("zenserp: HTTP %d: %s", resp.StatusCode, data)
	}

	var raw struct {
		Organic []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"organic"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("zenserp: decode: %w", err)
	}

	result := &SearchResult{}
	for _, r := range raw.Organic {
		result.OrganicResults = append(result.OrganicResults, map[string]any{
			"title": r.Title, "link": r.URL, "snippet": r.Description,
		})
	}
	return result, nil
}
