package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Crawler fetches GitHub data from the web interface JSON endpoints.
// This is used as a fallback when the API is rate limited or token is invalid.
type Crawler struct {
	httpClient *http.Client
	baseURL    string
}

// NewCrawler creates a new GitHub web crawler.
func NewCrawler(baseURL string) *Crawler {
	if baseURL == "" {
		baseURL = "https://github.com"
	}
	// Ensure we're using the web URL, not API URL
	baseURL = strings.Replace(baseURL, "api.github.com", "github.com", 1)
	baseURL = strings.TrimSuffix(baseURL, "/api/v3") // GitHub Enterprise

	return &Crawler{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}
}

// crawlIssueListItem represents an issue in the list JSON response.
type crawlIssueListItem struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ClosedAt  string `json:"closedAt"`
	Comments  int    `json:"commentsCount"`
}

// crawlIssueDetail represents a detailed issue response.
type crawlIssueDetail struct {
	ID          int64  `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	BodyHTML    string `json:"bodyHTML"`
	State       string `json:"state"`
	StateReason string `json:"stateReason"`
	Locked      bool   `json:"locked"`
	Author      struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	Labels []struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	} `json:"labels"`
	Assignees []struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"assignees"`
	Milestone *struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"milestone"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ClosedAt  string `json:"closedAt"`
	Comments  int    `json:"commentsCount"`
}

// crawlPRListItem represents a PR in the list JSON response.
type crawlPRListItem struct {
	ID     int64  `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Draft  bool   `json:"isDraft"`
	Author struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	ClosedAt    string `json:"closedAt"`
	MergedAt    string `json:"mergedAt"`
}

// crawlComment represents a comment from the web interface.
type crawlComment struct {
	ID     int64  `json:"id"`
	Body   string `json:"body"`
	Author struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatarUrl"`
	} `json:"author"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// crawlLabel represents a label from the labels page.
type crawlLabel struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// FetchIssues fetches issues from GitHub's web interface.
func (c *Crawler) FetchIssues(ctx context.Context, owner, repo string, page int, state string) ([]*ghIssue, error) {
	// GitHub web uses different state values
	webState := "open"
	if state == "all" {
		webState = "all"
	} else if state == "closed" {
		webState = "closed"
	}

	u := fmt.Sprintf("%s/%s/%s/issues", c.baseURL, owner, repo)
	query := url.Values{}
	query.Set("q", fmt.Sprintf("is:issue state:%s", webState))
	query.Set("page", strconv.Itoa(page))

	body, err := c.doRequest(ctx, u+"?"+query.Encode())
	if err != nil {
		return nil, err
	}

	// Try to parse as JSON array first
	var items []crawlIssueListItem
	if err := json.Unmarshal(body, &items); err != nil {
		// Try wrapped response
		var wrapped struct {
			Issues []crawlIssueListItem `json:"issues"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return nil, fmt.Errorf("parse issues JSON: %w (body: %s)", err, truncate(string(body), 200))
		}
		items = wrapped.Issues
	}

	// Convert to ghIssue
	issues := make([]*ghIssue, 0, len(items))
	for _, item := range items {
		issue := &ghIssue{
			ID:        item.ID,
			Number:    item.Number,
			Title:     item.Title,
			State:     item.State,
			Comments:  item.Comments,
			User:      &ghUser{Login: item.Author.Login, AvatarURL: item.Author.AvatarURL},
			Labels:    make([]*ghLabel, 0, len(item.Labels)),
			CreatedAt: parseTime(item.CreatedAt),
			UpdatedAt: parseTime(item.UpdatedAt),
		}
		if item.ClosedAt != "" {
			closedAt := parseTime(item.ClosedAt)
			issue.ClosedAt = &closedAt
		}
		for _, l := range item.Labels {
			issue.Labels = append(issue.Labels, &ghLabel{Name: l.Name, Color: l.Color})
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// FetchIssueDetail fetches detailed issue data including body.
func (c *Crawler) FetchIssueDetail(ctx context.Context, owner, repo string, number int) (*ghIssue, error) {
	u := fmt.Sprintf("%s/%s/%s/issues/%d", c.baseURL, owner, repo, number)

	body, err := c.doRequest(ctx, u)
	if err != nil {
		return nil, err
	}

	var detail crawlIssueDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("parse issue detail JSON: %w", err)
	}

	issue := &ghIssue{
		ID:          detail.ID,
		Number:      detail.Number,
		Title:       detail.Title,
		Body:        detail.Body,
		State:       detail.State,
		StateReason: detail.StateReason,
		Locked:      detail.Locked,
		Comments:    detail.Comments,
		User:        &ghUser{Login: detail.Author.Login, AvatarURL: detail.Author.AvatarURL},
		Labels:      make([]*ghLabel, 0, len(detail.Labels)),
		Assignees:   make([]*ghUser, 0, len(detail.Assignees)),
		CreatedAt:   parseTime(detail.CreatedAt),
		UpdatedAt:   parseTime(detail.UpdatedAt),
	}

	if detail.ClosedAt != "" {
		closedAt := parseTime(detail.ClosedAt)
		issue.ClosedAt = &closedAt
	}

	for _, l := range detail.Labels {
		issue.Labels = append(issue.Labels, &ghLabel{Name: l.Name, Color: l.Color, Description: l.Description})
	}

	for _, a := range detail.Assignees {
		issue.Assignees = append(issue.Assignees, &ghUser{Login: a.Login, AvatarURL: a.AvatarURL})
	}

	if detail.Milestone != nil {
		issue.Milestone = &ghMilestone{Number: detail.Milestone.Number, Title: detail.Milestone.Title}
	}

	return issue, nil
}

// FetchPullRequests fetches PRs from GitHub's web interface.
func (c *Crawler) FetchPullRequests(ctx context.Context, owner, repo string, page int, state string) ([]*ghPullRequest, error) {
	webState := "open"
	if state == "all" {
		webState = "all"
	} else if state == "closed" || state == "merged" {
		webState = "closed"
	}

	u := fmt.Sprintf("%s/%s/%s/pulls", c.baseURL, owner, repo)
	query := url.Values{}
	query.Set("q", fmt.Sprintf("is:pr state:%s", webState))
	query.Set("page", strconv.Itoa(page))

	body, err := c.doRequest(ctx, u+"?"+query.Encode())
	if err != nil {
		return nil, err
	}

	var items []crawlPRListItem
	if err := json.Unmarshal(body, &items); err != nil {
		var wrapped struct {
			PullRequests []crawlPRListItem `json:"pullRequests"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return nil, fmt.Errorf("parse PRs JSON: %w", err)
		}
		items = wrapped.PullRequests
	}

	prs := make([]*ghPullRequest, 0, len(items))
	for _, item := range items {
		pr := &ghPullRequest{
			ID:        item.ID,
			Number:    item.Number,
			Title:     item.Title,
			State:     item.State,
			Draft:     item.Draft,
			User:      &ghUser{Login: item.Author.Login, AvatarURL: item.Author.AvatarURL},
			Labels:    make([]*ghLabel, 0, len(item.Labels)),
			Head:      &ghBranch{Ref: item.HeadRefName},
			Base:      &ghBranch{Ref: item.BaseRefName},
			CreatedAt: parseTime(item.CreatedAt),
			UpdatedAt: parseTime(item.UpdatedAt),
		}

		if item.ClosedAt != "" {
			closedAt := parseTime(item.ClosedAt)
			pr.ClosedAt = &closedAt
		}
		if item.MergedAt != "" {
			mergedAt := parseTime(item.MergedAt)
			pr.MergedAt = &mergedAt
			pr.Merged = true
		}

		for _, l := range item.Labels {
			pr.Labels = append(pr.Labels, &ghLabel{Name: l.Name, Color: l.Color})
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// FetchComments fetches comments for an issue/PR.
func (c *Crawler) FetchComments(ctx context.Context, owner, repo string, number int) ([]*ghComment, error) {
	u := fmt.Sprintf("%s/%s/%s/issues/%d/comments", c.baseURL, owner, repo, number)

	body, err := c.doRequest(ctx, u)
	if err != nil {
		return nil, err
	}

	var items []crawlComment
	if err := json.Unmarshal(body, &items); err != nil {
		var wrapped struct {
			Comments []crawlComment `json:"comments"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return nil, fmt.Errorf("parse comments JSON: %w", err)
		}
		items = wrapped.Comments
	}

	comments := make([]*ghComment, 0, len(items))
	for _, item := range items {
		comments = append(comments, &ghComment{
			ID:        item.ID,
			Body:      item.Body,
			User:      &ghUser{Login: item.Author.Login, AvatarURL: item.Author.AvatarURL},
			CreatedAt: parseTime(item.CreatedAt),
			UpdatedAt: parseTime(item.UpdatedAt),
		})
	}

	return comments, nil
}

// FetchLabels fetches labels from GitHub's labels page.
func (c *Crawler) FetchLabels(ctx context.Context, owner, repo string) ([]*ghLabel, error) {
	u := fmt.Sprintf("%s/%s/%s/labels", c.baseURL, owner, repo)

	body, err := c.doRequest(ctx, u)
	if err != nil {
		return nil, err
	}

	var items []crawlLabel
	if err := json.Unmarshal(body, &items); err != nil {
		var wrapped struct {
			Labels []crawlLabel `json:"labels"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return nil, fmt.Errorf("parse labels JSON: %w", err)
		}
		items = wrapped.Labels
	}

	labels := make([]*ghLabel, 0, len(items))
	for _, item := range items {
		labels = append(labels, &ghLabel{
			Name:        item.Name,
			Color:       item.Color,
			Description: item.Description,
		})
	}

	return labels, nil
}

// FetchRepository fetches basic repository info from GitHub's web interface.
func (c *Crawler) FetchRepository(ctx context.Context, owner, repo string) (*ghRepository, error) {
	u := fmt.Sprintf("%s/%s/%s", c.baseURL, owner, repo)

	body, err := c.doRequest(ctx, u)
	if err != nil {
		return nil, err
	}

	var data struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		FullName    string `json:"fullName"`
		Description string `json:"description"`
		Private     bool   `json:"isPrivate"`
		Fork        bool   `json:"isFork"`
		Owner       struct {
			Login     string `json:"login"`
			AvatarURL string `json:"avatarUrl"`
			Type      string `json:"type"`
		} `json:"owner"`
		DefaultBranch   string `json:"defaultBranch"`
		StargazersCount int    `json:"stargazersCount"`
		ForksCount      int    `json:"forksCount"`
		WatchersCount   int    `json:"watchersCount"`
		OpenIssuesCount int    `json:"openIssuesCount"`
		CreatedAt       string `json:"createdAt"`
		UpdatedAt       string `json:"updatedAt"`
		PushedAt        string `json:"pushedAt"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse repository JSON: %w", err)
	}

	ghRepo := &ghRepository{
		ID:              data.ID,
		Name:            data.Name,
		FullName:        data.FullName,
		Description:     data.Description,
		Private:         data.Private,
		Fork:            data.Fork,
		DefaultBranch:   data.DefaultBranch,
		StargazersCount: data.StargazersCount,
		ForksCount:      data.ForksCount,
		WatchersCount:   data.WatchersCount,
		OpenIssuesCount: data.OpenIssuesCount,
		CreatedAt:       parseTime(data.CreatedAt),
		UpdatedAt:       parseTime(data.UpdatedAt),
		PushedAt:        parseTime(data.PushedAt),
	}

	if data.Owner.Login != "" {
		ghRepo.Owner = &ghUser{
			Login:     data.Owner.Login,
			AvatarURL: data.Owner.AvatarURL,
			Type:      data.Owner.Type,
		}
	}

	return ghRepo, nil
}

// doRequest performs an HTTP request expecting JSON response.
func (c *Crawler) doRequest(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Request JSON response
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GitHome-Seeder/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found: %s", urlStr)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}

// parseTime parses various time formats from GitHub.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Time{}
}

// truncate truncates a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
