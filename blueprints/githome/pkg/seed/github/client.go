package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client wraps GitHub API interactions.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a new GitHub API client.
func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		token:      token,
	}
}

// ValidateToken checks if the token is valid by making a simple API call.
// Returns true if valid, false if invalid. Returns an error only for network issues.
func (c *Client) ValidateToken(ctx context.Context) (bool, error) {
	if c.token == "" {
		return false, nil // No token to validate
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/user", nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil // Token is invalid
	}
	if resp.StatusCode == http.StatusOK {
		return true, nil // Token is valid
	}

	// Other status codes might indicate rate limiting or other issues
	return true, nil // Assume valid for other cases
}

// ClearToken removes the token from the client (for fallback to unauthenticated).
func (c *Client) ClearToken() {
	c.token = ""
}

// ListOptions contains pagination options.
type ListOptions struct {
	Page    int
	PerPage int
	State   string // open, closed, all
}

// RateLimitInfo contains rate limit information from GitHub API.
type RateLimitInfo struct {
	Remaining int
	Reset     time.Time
}

// ghRepository represents a GitHub repository from the API.
type ghRepository struct {
	ID              int64     `json:"id"`
	NodeID          string    `json:"node_id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Owner           *ghUser   `json:"owner"`
	Private         bool      `json:"private"`
	Description     string    `json:"description"`
	Fork            bool      `json:"fork"`
	DefaultBranch   string    `json:"default_branch"`
	HasIssues       bool      `json:"has_issues"`
	HasProjects     bool      `json:"has_projects"`
	HasWiki         bool      `json:"has_wiki"`
	HasDownloads    bool      `json:"has_downloads"`
	OpenIssuesCount int       `json:"open_issues_count"`
	ForksCount      int       `json:"forks_count"`
	StargazersCount int       `json:"stargazers_count"`
	WatchersCount   int       `json:"watchers_count"`
	Size            int       `json:"size"`
	Language        string    `json:"language"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PushedAt        time.Time `json:"pushed_at"`
}

// ghUser represents a GitHub user from the API.
type ghUser struct {
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"` // User or Organization
	SiteAdmin bool   `json:"site_admin"`
}

// ghIssue represents a GitHub issue from the API.
type ghIssue struct {
	ID              int64         `json:"id"`
	NodeID          string        `json:"node_id"`
	Number          int           `json:"number"`
	Title           string        `json:"title"`
	Body            string        `json:"body"`
	State           string        `json:"state"`
	StateReason     string        `json:"state_reason"`
	User            *ghUser       `json:"user"`
	Labels          []*ghLabel    `json:"labels"`
	Milestone       *ghMilestone  `json:"milestone"`
	Assignees       []*ghUser     `json:"assignees"`
	Locked          bool          `json:"locked"`
	ActiveLockReason string       `json:"active_lock_reason"`
	Comments        int           `json:"comments"`
	PullRequest     *ghIssuePR    `json:"pull_request"`
	ClosedAt        *time.Time    `json:"closed_at"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// ghIssuePR indicates if an issue is actually a PR.
type ghIssuePR struct {
	URL string `json:"url"`
}

// ghPullRequest represents a GitHub pull request from the API.
type ghPullRequest struct {
	ID                  int64        `json:"id"`
	NodeID              string       `json:"node_id"`
	Number              int          `json:"number"`
	Title               string       `json:"title"`
	Body                string       `json:"body"`
	State               string       `json:"state"`
	User                *ghUser      `json:"user"`
	Labels              []*ghLabel   `json:"labels"`
	Milestone           *ghMilestone `json:"milestone"`
	Assignees           []*ghUser    `json:"assignees"`
	Locked              bool         `json:"locked"`
	ActiveLockReason    string       `json:"active_lock_reason"`
	Head                *ghBranch    `json:"head"`
	Base                *ghBranch    `json:"base"`
	Draft               bool         `json:"draft"`
	Merged              bool         `json:"merged"`
	Mergeable           *bool        `json:"mergeable"`
	MergeableState      string       `json:"mergeable_state"`
	MergedAt            *time.Time   `json:"merged_at"`
	MergedBy            *ghUser      `json:"merged_by"`
	MergeCommitSHA      string       `json:"merge_commit_sha"`
	Comments            int          `json:"comments"`
	ReviewComments      int          `json:"review_comments"`
	Commits             int          `json:"commits"`
	Additions           int          `json:"additions"`
	Deletions           int          `json:"deletions"`
	ChangedFiles        int          `json:"changed_files"`
	ClosedAt            *time.Time   `json:"closed_at"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

// ghBranch represents a branch reference in a PR.
type ghBranch struct {
	Label string  `json:"label"`
	Ref   string  `json:"ref"`
	SHA   string  `json:"sha"`
	User  *ghUser `json:"user"`
}

// ghLabel represents a GitHub label from the API.
type ghLabel struct {
	ID          int64  `json:"id"`
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Default     bool   `json:"default"`
}

// ghMilestone represents a GitHub milestone from the API.
type ghMilestone struct {
	ID           int64      `json:"id"`
	NodeID       string     `json:"node_id"`
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        string     `json:"state"`
	Creator      *ghUser    `json:"creator"`
	OpenIssues   int        `json:"open_issues"`
	ClosedIssues int        `json:"closed_issues"`
	ClosedAt     *time.Time `json:"closed_at"`
	DueOn        *time.Time `json:"due_on"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ghComment represents a GitHub issue comment from the API.
type ghComment struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	Body      string    `json:"body"`
	User      *ghUser   `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ghReviewComment represents a GitHub PR review comment from the API.
type ghReviewComment struct {
	ID                  int64      `json:"id"`
	NodeID              string     `json:"node_id"`
	PullRequestReviewID int64      `json:"pull_request_review_id"`
	DiffHunk            string     `json:"diff_hunk"`
	Path                string     `json:"path"`
	Position            *int       `json:"position"`
	OriginalPosition    *int       `json:"original_position"`
	CommitID            string     `json:"commit_id"`
	OriginalCommitID    string     `json:"original_commit_id"`
	InReplyToID         int64      `json:"in_reply_to_id"`
	User                *ghUser    `json:"user"`
	Body                string     `json:"body"`
	Line                *int       `json:"line"`
	OriginalLine        *int       `json:"original_line"`
	StartLine           *int       `json:"start_line"`
	OriginalStartLine   *int       `json:"original_start_line"`
	Side                string     `json:"side"`
	StartSide           string     `json:"start_side"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// do performs an HTTP request and returns the response body.
func (c *Client) do(ctx context.Context, method, path string, query url.Values) ([]byte, *RateLimitInfo, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, nil, fmt.Errorf("parse url: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "GitHome-Seeder/1.0")
	if c.token != "" && len(c.token) > 0 {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Parse rate limit info
	rateInfo := &RateLimitInfo{}
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		rateInfo.Remaining, _ = strconv.Atoi(remaining)
	}
	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			rateInfo.Reset = time.Unix(ts, 0)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, rateInfo, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, rateInfo, fmt.Errorf("not found: %s", path)
	}
	if resp.StatusCode == http.StatusForbidden && rateInfo.Remaining == 0 {
		return nil, rateInfo, fmt.Errorf("rate limit exceeded, resets at %s", rateInfo.Reset.Format(time.RFC3339))
	}
	if resp.StatusCode >= 400 {
		return nil, rateInfo, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, rateInfo, nil
}

// GetRepository fetches repository metadata.
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*ghRepository, *RateLimitInfo, error) {
	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s", owner, repo), nil)
	if err != nil {
		return nil, rateInfo, err
	}

	var result ghRepository
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal repository: %w", err)
	}
	return &result, rateInfo, nil
}

// GetUser fetches user metadata.
func (c *Client) GetUser(ctx context.Context, login string) (*ghUser, *RateLimitInfo, error) {
	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/users/%s", login), nil)
	if err != nil {
		return nil, rateInfo, err
	}

	var result ghUser
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal user: %w", err)
	}
	return &result, rateInfo, nil
}

// ListIssues fetches issues for a repository.
func (c *Client) ListIssues(ctx context.Context, owner, repo string, opts *ListOptions) ([]*ghIssue, *RateLimitInfo, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.State != "" {
			query.Set("state", opts.State)
		}
	}
	query.Set("sort", "created")
	query.Set("direction", "asc")

	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/issues", owner, repo), query)
	if err != nil {
		return nil, rateInfo, err
	}

	var result []*ghIssue
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal issues: %w", err)
	}
	return result, rateInfo, nil
}

// ListPullRequests fetches pull requests for a repository.
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts *ListOptions) ([]*ghPullRequest, *RateLimitInfo, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.State != "" {
			query.Set("state", opts.State)
		}
	}
	query.Set("sort", "created")
	query.Set("direction", "asc")

	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls", owner, repo), query)
	if err != nil {
		return nil, rateInfo, err
	}

	var result []*ghPullRequest
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal pull requests: %w", err)
	}
	return result, rateInfo, nil
}

// ListIssueComments fetches comments for an issue.
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ghComment, *RateLimitInfo, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
	}

	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number), query)
	if err != nil {
		return nil, rateInfo, err
	}

	var result []*ghComment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal comments: %w", err)
	}
	return result, rateInfo, nil
}

// ListPRComments fetches review comments for a pull request.
func (c *Client) ListPRComments(ctx context.Context, owner, repo string, number int, opts *ListOptions) ([]*ghReviewComment, *RateLimitInfo, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
	}

	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number), query)
	if err != nil {
		return nil, rateInfo, err
	}

	var result []*ghReviewComment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal review comments: %w", err)
	}
	return result, rateInfo, nil
}

// ListLabels fetches labels for a repository.
func (c *Client) ListLabels(ctx context.Context, owner, repo string, opts *ListOptions) ([]*ghLabel, *RateLimitInfo, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
	}

	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/labels", owner, repo), query)
	if err != nil {
		return nil, rateInfo, err
	}

	var result []*ghLabel
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal labels: %w", err)
	}
	return result, rateInfo, nil
}

// ListMilestones fetches milestones for a repository.
func (c *Client) ListMilestones(ctx context.Context, owner, repo string, opts *ListOptions) ([]*ghMilestone, *RateLimitInfo, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.State != "" {
			query.Set("state", opts.State)
		}
	}

	body, rateInfo, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/milestones", owner, repo), query)
	if err != nil {
		return nil, rateInfo, err
	}

	var result []*ghMilestone
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rateInfo, fmt.Errorf("unmarshal milestones: %w", err)
	}
	return result, rateInfo, nil
}
