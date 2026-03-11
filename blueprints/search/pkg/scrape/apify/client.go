package apify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client performs HTTP requests to Algolia and Apify APIs.
type Client struct {
	cfg  Config
	http *http.Client
}

type SearchRequest struct {
	Page     int
	Category string
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) SearchStorePage(ctx context.Context, sr SearchRequest) (*StoreSearchResponse, error) {
	endpoint := strings.TrimRight(c.cfg.AlgoliaDSNBaseURL, "/") + "/1/indexes/" + c.cfg.AlgoliaIndexName + "/query"
	payload := map[string]any{
		"query":       "",
		"hitsPerPage": c.cfg.HitsPerPage,
		"page":        sr.Page,
		"attributesToRetrieve": []string{
			"objectID",
			"name",
			"username",
			"title",
			"description",
			"categories",
			"modifiedAt",
			"createdAt",
			"pictureUrl",
		},
	}
	if sr.Category != "" {
		payload["filters"] = "categories:" + sr.Category
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-algolia-api-key", c.cfg.AlgoliaAPIKey)
	req.Header.Set("x-algolia-application-id", c.cfg.AlgoliaAppID)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "search-apify-crawler/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("algolia search failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out StoreSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListCategories(ctx context.Context) ([]string, error) {
	endpoint := strings.TrimRight(c.cfg.AlgoliaDSNBaseURL, "/") + "/1/indexes/" + c.cfg.AlgoliaIndexName + "/query"
	payload := map[string]any{
		"query":       "",
		"hitsPerPage": 0,
		"page":        0,
		"facets":      []string{"categories"},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-algolia-api-key", c.cfg.AlgoliaAPIKey)
	req.Header.Set("x-algolia-application-id", c.cfg.AlgoliaAppID)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "search-apify-crawler/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("algolia facets failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out struct {
		Facets map[string]map[string]any `json:"facets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	catsMap := out.Facets["categories"]
	cats := make([]string, 0, len(catsMap))
	for k := range catsMap {
		if k == "" {
			continue
		}
		cats = append(cats, k)
	}
	return cats, nil
}

func (c *Client) FetchActorDetail(ctx context.Context, objectID string) (*ActorDetailResponse, int, []byte, error) {
	escaped := url.PathEscape(objectID)
	endpoint := strings.TrimRight(c.cfg.ActorAPIBaseURL, "/") + "/v2/acts/" + escaped
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "search-apify-crawler/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, body, fmt.Errorf("detail fetch failed status=%d", resp.StatusCode)
	}

	var out ActorDetailResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, body, err
	}
	return &out, resp.StatusCode, body, nil
}

func (c *Client) FetchActorVersions(ctx context.Context, objectID string, limit int, offset int) (*ActorVersionsResponse, int, []byte, error) {
	if limit <= 0 {
		limit = 1000
	}
	escaped := url.PathEscape(objectID)
	endpoint := strings.TrimRight(c.cfg.ActorAPIBaseURL, "/") + "/v2/acts/" + escaped + "/versions?limit=" + fmt.Sprint(limit) + "&offset=" + fmt.Sprint(offset) + "&desc=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "search-apify-crawler/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, body, fmt.Errorf("versions fetch failed status=%d", resp.StatusCode)
	}

	var out ActorVersionsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, body, err
	}
	return &out, resp.StatusCode, body, nil
}

func (c *Client) FetchActorBuild(ctx context.Context, buildID string) (*ActorBuildResponse, int, []byte, error) {
	escaped := url.PathEscape(buildID)
	endpoint := strings.TrimRight(c.cfg.ActorAPIBaseURL, "/") + "/v2/actor-builds/" + escaped
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "search-apify-crawler/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, body, fmt.Errorf("build fetch failed status=%d", resp.StatusCode)
	}

	var out ActorBuildResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp.StatusCode, body, err
	}
	return &out, resp.StatusCode, body, nil
}

func sleepBackoff(attempt int) {
	if attempt <= 0 {
		return
	}
	time.Sleep(time.Duration(attempt*attempt) * 250 * time.Millisecond)
}
