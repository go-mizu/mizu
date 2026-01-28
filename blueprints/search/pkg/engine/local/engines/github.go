package engines

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GitHub implements GitHub repository search.
type GitHub struct {
	*BaseEngine
}

// NewGitHub creates a new GitHub engine.
func NewGitHub() *GitHub {
	g := &GitHub{
		BaseEngine: NewBaseEngine("github", "gh", []Category{CategoryIT}),
	}

	g.SetPaging(false).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:         "https://github.com",
			WikidataID:      "Q364",
			OfficialAPIDocs: "https://docs.github.com/en/rest",
			UseOfficialAPI:  true,
			Results:         "JSON",
		})

	return g
}

func (g *GitHub) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("sort", "stars")
	queryParams.Set("order", "desc")

	params.URL = "https://api.github.com/search/repositories?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/vnd.github.preview.text-match+json")
	params.Headers.Set("User-Agent", "SearXNG")

	return nil
}

func (g *GitHub) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp githubSearchResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	for _, item := range apiResp.Items {
		result := Result{
			URL:      item.HTMLURL,
			Title:    item.FullName,
			Content:  item.Description,
			Template: "packages",
		}

		// Add thumbnail from owner avatar
		if item.Owner.AvatarURL != "" {
			result.ThumbnailURL = item.Owner.AvatarURL
		}

		// Parse dates
		if item.UpdatedAt != "" {
			if t, err := time.Parse(time.RFC3339, item.UpdatedAt); err == nil {
				result.PublishedAt = t
			}
		}

		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

type githubSearchResponse struct {
	TotalCount int          `json:"total_count"`
	Items      []githubRepo `json:"items"`
}

type githubRepo struct {
	ID          int    `json:"id"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
	Fork        bool   `json:"fork"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Watchers    int    `json:"watchers_count"`
	Forks       int    `json:"forks_count"`
	OpenIssues  int    `json:"open_issues_count"`
	License     struct {
		Name   string `json:"name"`
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
	Topics    []string `json:"topics"`
	UpdatedAt string   `json:"updated_at"`
	CreatedAt string   `json:"created_at"`
	Homepage  string   `json:"homepage"`
	CloneURL  string   `json:"clone_url"`
	Owner     struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	} `json:"owner"`
}

// GitHubCode implements GitHub code search.
type GitHubCode struct {
	*BaseEngine
}

// NewGitHubCode creates a new GitHub code search engine.
// Note: GitHub Code Search API requires authentication.
func NewGitHubCode() *GitHubCode {
	g := &GitHubCode{
		BaseEngine: NewBaseEngine("github code", "ghc", []Category{CategoryIT}),
	}

	g.SetPaging(true).
		SetTimeout(5 * time.Second).
		SetDisabled(true). // Requires authentication
		SetAbout(EngineAbout{
			Website:         "https://github.com",
			WikidataID:      "Q364",
			OfficialAPIDocs: "https://docs.github.com/en/rest",
			UseOfficialAPI:  true,
			RequireAPIKey:   true,
			Results:         "JSON",
		})

	return g
}

func (g *GitHubCode) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)

	if params.PageNo > 1 {
		queryParams.Set("page", string(rune('0'+params.PageNo)))
	}

	params.URL = "https://api.github.com/search/code?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/vnd.github.preview.text-match+json")
	params.Headers.Set("User-Agent", "SearXNG")

	return nil
}

func (g *GitHubCode) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp githubCodeResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	for _, item := range apiResp.Items {
		result := Result{
			URL:      item.HTMLURL,
			Title:    item.Path,
			Content:  item.Repository.FullName,
			Template: "code",
		}

		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

type githubCodeResponse struct {
	TotalCount int              `json:"total_count"`
	Items      []githubCodeItem `json:"items"`
}

type githubCodeItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	SHA        string `json:"sha"`
	HTMLURL    string `json:"html_url"`
	Repository struct {
		ID       int    `json:"id"`
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
}
