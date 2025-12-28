// Package github provides a client for fetching data from GitHub
// to seed a local GitHome instance for testing or migration.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	userAgent      = "GitHome-Seeder/1.0"
)

// Client is a GitHub API client
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new GitHub client
// If token is empty, requests will be unauthenticated (with lower rate limits)
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		token:      token,
	}
}

// doRequest performs an HTTP request with proper headers
func (c *Client) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

// get performs a GET request and decodes the JSON response
func (c *Client) get(ctx context.Context, path string, v interface{}) error {
	resp, err := c.doRequest(ctx, "GET", path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api error: %d - %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

// getAll fetches all pages of a paginated response
func (c *Client) getAll(ctx context.Context, path string, perPage int) ([]json.RawMessage, error) {
	var all []json.RawMessage
	page := 1

	for {
		pagePath := fmt.Sprintf("%s?per_page=%d&page=%d", path, perPage, page)

		resp, err := c.doRequest(ctx, "GET", pagePath)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("github api error: %d - %s", resp.StatusCode, string(body))
		}

		var items []json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(items) == 0 {
			break
		}

		all = append(all, items...)

		// Check if there are more pages
		linkHeader := resp.Header.Get("Link")
		if !hasNextPage(linkHeader) {
			break
		}

		page++
	}

	return all, nil
}

// hasNextPage checks if the Link header contains a next page
func hasNextPage(link string) bool {
	return link != "" && (contains(link, `rel="next"`) || contains(link, `rel='next'`))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GitHub API response types

// Repository represents a GitHub repository
type Repository struct {
	ID              int64     `json:"id"`
	NodeID          string    `json:"node_id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Owner           User      `json:"owner"`
	Description     string    `json:"description"`
	Private         bool      `json:"private"`
	Fork            bool      `json:"fork"`
	Language        string    `json:"language"`
	StargazersCount int       `json:"stargazers_count"`
	ForksCount      int       `json:"forks_count"`
	WatchersCount   int       `json:"watchers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	DefaultBranch   string    `json:"default_branch"`
	License         *License  `json:"license"`
	Topics          []string  `json:"topics"`
	Archived        bool      `json:"archived"`
	Disabled        bool      `json:"disabled"`
	HasIssues       bool      `json:"has_issues"`
	HasWiki         bool      `json:"has_wiki"`
	HasProjects     bool      `json:"has_projects"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PushedAt        time.Time `json:"pushed_at"`
}

// License represents a repository license
type License struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	SPDXID string `json:"spdx_id"`
}

// User represents a GitHub user
type User struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
}

// Issue represents a GitHub issue
type Issue struct {
	ID               int64       `json:"id"`
	NodeID           string      `json:"node_id"`
	Number           int         `json:"number"`
	Title            string      `json:"title"`
	Body             string      `json:"body"`
	State            string      `json:"state"`
	Locked           bool        `json:"locked"`
	ActiveLockReason string      `json:"active_lock_reason"`
	User             User        `json:"user"`
	Labels           []Label     `json:"labels"`
	Assignees        []User      `json:"assignees"`
	Milestone        *Milestone  `json:"milestone"`
	Comments         int         `json:"comments"`
	PullRequest      *PRLink     `json:"pull_request,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	ClosedAt         *time.Time  `json:"closed_at"`
	ClosedBy         *User       `json:"closed_by"`
	StateReason      string      `json:"state_reason"`
}

// PRLink indicates if an issue is actually a pull request
type PRLink struct {
	URL string `json:"url"`
}

// Label represents a GitHub label
type Label struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

// Milestone represents a GitHub milestone
type Milestone struct {
	ID           int64      `json:"id"`
	NodeID       string     `json:"node_id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
	Creator      User       `json:"creator"`
	DueOn        *time.Time `json:"due_on"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ClosedAt     *time.Time `json:"closed_at"`
}

// Comment represents a GitHub comment
type Comment struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID                int64      `json:"id"`
	NodeID            string     `json:"node_id"`
	Number            int        `json:"number"`
	Title             string     `json:"title"`
	Body              string     `json:"body"`
	State             string     `json:"state"`
	Locked            bool       `json:"locked"`
	User              User       `json:"user"`
	Labels            []Label    `json:"labels"`
	Milestone         *Milestone `json:"milestone"`
	Assignees         []User     `json:"assignees"`
	Head              Branch     `json:"head"`
	Base              Branch     `json:"base"`
	Draft             bool       `json:"draft"`
	Merged            bool       `json:"merged"`
	Mergeable         *bool      `json:"mergeable"`
	MergeableState    string     `json:"mergeable_state"`
	MergedAt          *time.Time `json:"merged_at"`
	MergedBy          *User      `json:"merged_by"`
	Additions         int        `json:"additions"`
	Deletions         int        `json:"deletions"`
	ChangedFiles      int        `json:"changed_files"`
	Comments          int        `json:"comments"`
	ReviewComments    int        `json:"review_comments"`
	Commits           int        `json:"commits"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ClosedAt          *time.Time `json:"closed_at"`
}

// Branch represents a git branch in a PR
type Branch struct {
	Label string     `json:"label"`
	Ref   string     `json:"ref"`
	SHA   string     `json:"sha"`
	User  User       `json:"user"`
	Repo  *Repository `json:"repo"`
}

// FetchRepository fetches repository metadata
func (c *Client) FetchRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)

	var r Repository
	if err := c.get(ctx, path, &r); err != nil {
		return nil, err
	}

	return &r, nil
}

// FetchIssues fetches all issues from a repository
func (c *Client) FetchIssues(ctx context.Context, owner, repo string) ([]*Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues?state=all", owner, repo)

	rawItems, err := c.getAll(ctx, path, 100)
	if err != nil {
		return nil, err
	}

	issues := make([]*Issue, 0, len(rawItems))
	for _, raw := range rawItems {
		var issue Issue
		if err := json.Unmarshal(raw, &issue); err != nil {
			continue
		}
		// Skip PRs (GitHub API returns PRs in issues endpoint)
		if issue.PullRequest != nil {
			continue
		}
		issues = append(issues, &issue)
	}

	return issues, nil
}

// FetchPullRequests fetches all pull requests from a repository
func (c *Client) FetchPullRequests(ctx context.Context, owner, repo string) ([]*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=all", owner, repo)

	rawItems, err := c.getAll(ctx, path, 100)
	if err != nil {
		return nil, err
	}

	prs := make([]*PullRequest, 0, len(rawItems))
	for _, raw := range rawItems {
		var pr PullRequest
		if err := json.Unmarshal(raw, &pr); err != nil {
			continue
		}
		prs = append(prs, &pr)
	}

	return prs, nil
}

// FetchComments fetches comments for an issue or PR
func (c *Client) FetchComments(ctx context.Context, owner, repo string, number int) ([]*Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)

	rawItems, err := c.getAll(ctx, path, 100)
	if err != nil {
		return nil, err
	}

	comments := make([]*Comment, 0, len(rawItems))
	for _, raw := range rawItems {
		var comment Comment
		if err := json.Unmarshal(raw, &comment); err != nil {
			continue
		}
		comments = append(comments, &comment)
	}

	return comments, nil
}

// FetchLabels fetches all labels from a repository
func (c *Client) FetchLabels(ctx context.Context, owner, repo string) ([]*Label, error) {
	path := fmt.Sprintf("/repos/%s/%s/labels", owner, repo)

	rawItems, err := c.getAll(ctx, path, 100)
	if err != nil {
		return nil, err
	}

	labels := make([]*Label, 0, len(rawItems))
	for _, raw := range rawItems {
		var label Label
		if err := json.Unmarshal(raw, &label); err != nil {
			continue
		}
		labels = append(labels, &label)
	}

	return labels, nil
}

// FetchMilestones fetches all milestones from a repository
func (c *Client) FetchMilestones(ctx context.Context, owner, repo string) ([]*Milestone, error) {
	path := fmt.Sprintf("/repos/%s/%s/milestones?state=all", owner, repo)

	rawItems, err := c.getAll(ctx, path, 100)
	if err != nil {
		return nil, err
	}

	milestones := make([]*Milestone, 0, len(rawItems))
	for _, raw := range rawItems {
		var m Milestone
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		milestones = append(milestones, &m)
	}

	return milestones, nil
}

// FetchContributors fetches contributors for a repository
func (c *Client) FetchContributors(ctx context.Context, owner, repo string) ([]*User, error) {
	path := fmt.Sprintf("/repos/%s/%s/contributors", owner, repo)

	rawItems, err := c.getAll(ctx, path, 100)
	if err != nil {
		return nil, err
	}

	users := make([]*User, 0, len(rawItems))
	for _, raw := range rawItems {
		var u User
		if err := json.Unmarshal(raw, &u); err != nil {
			continue
		}
		users = append(users, &u)
	}

	return users, nil
}

// RateLimit returns the current rate limit status
type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
	Used      int       `json:"used"`
}

type rateLimitResponse struct {
	Rate RateLimit `json:"rate"`
}

// GetRateLimit returns the current rate limit status
func (c *Client) GetRateLimit(ctx context.Context) (*RateLimit, error) {
	var resp rateLimitResponse
	if err := c.get(ctx, "/rate_limit", &resp); err != nil {
		return nil, err
	}
	return &resp.Rate, nil
}

// Unused but satisfies the import
var _ = strconv.Itoa
