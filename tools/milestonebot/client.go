package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Client is the smallest GitHub REST client that does this job. It is not a
// general purpose binding and should not grow into one: when a command needs
// an endpoint, it adds the one call it needs.
type Client struct {
	Repo  string // owner/name
	Token string
	Base  string // API root, https://api.github.com unless overridden by a test
	HTTP  *http.Client

	// DryRun makes every write print what it would have sent and return
	// success. Reads still happen, so a dry run reports real drift.
	DryRun bool
}

func NewClient(repo, token string) *Client {
	return &Client{
		Repo:  repo,
		Token: token,
		Base:  "https://api.github.com",
		HTTP:  &http.Client{Timeout: 30 * time.Second},
	}
}

// do sends one request. A nil body means no request body, and a nil out means
// the response body is discarded. A write while DryRun is set is reported and
// skipped, which is why every caller passes the method through rather than
// having separate get and post paths that could disagree about it.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	if c.DryRun && method != http.MethodGet {
		fmt.Printf("dry run: %s %s\n", method, path)
		return nil
	}

	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		rd = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rd)
	if err != nil {
		return fmt.Errorf("build %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "milestonebot")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}

var nextLink = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// listAll walks a paginated collection to the end. GitHub caps a page at 100
// and the label and file lists here are well under a few hundred, so this
// stops when a page comes back short rather than following Link headers
// through an unbounded number of requests.
func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	const perPage = 100
	var all []T
	for page := 1; ; page++ {
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		var batch []T
		p := fmt.Sprintf("%s%sper_page=%d&page=%d", path, sep, perPage, page)
		if err := c.do(ctx, http.MethodGet, p, nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			return all, nil
		}
		if page == 20 {
			return all, fmt.Errorf("%s: more than %d pages, refusing to keep paging", path, page)
		}
	}
}

// Label is a repository label as the API returns it and as labels.yml
// describes it. Color has no leading hash on either side of the wire.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

func (c *Client) Labels(ctx context.Context) ([]Label, error) {
	return listAll[Label](ctx, c, "/repos/"+c.Repo+"/labels")
}

func (c *Client) CreateLabel(ctx context.Context, l Label) error {
	return c.do(ctx, http.MethodPost, "/repos/"+c.Repo+"/labels", l, nil)
}

func (c *Client) UpdateLabel(ctx context.Context, l Label) error {
	return c.do(ctx, http.MethodPatch, "/repos/"+c.Repo+"/labels/"+url.PathEscape(l.Name), l, nil)
}

// PullRequest carries only the fields the commands read. Everything else the
// API returns is ignored, which keeps this struct honest about what the tool
// depends on.
type PullRequest struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	HTMLURL   string `json:"html_url"`
	Merged    bool   `json:"merged"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []Label `json:"labels"`
}

func (c *Client) PullRequest(ctx context.Context, number int) (*PullRequest, error) {
	var pr PullRequest
	p := fmt.Sprintf("/repos/%s/pulls/%d", c.Repo, number)
	if err := c.do(ctx, http.MethodGet, p, nil, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// ChangedFiles returns the paths a pull request touches, which is what the
// area labels are derived from.
func (c *Client) ChangedFiles(ctx context.Context, number int) ([]string, error) {
	type file struct {
		Filename string `json:"filename"`
	}
	p := fmt.Sprintf("/repos/%s/pulls/%d/files", c.Repo, number)
	files, err := listAll[file](ctx, c, p)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = f.Filename
	}
	return paths, nil
}

// Issue is the tracking issue whose body holds the checklist.
type Issue struct {
	Number  int     `json:"number"`
	Title   string  `json:"title"`
	Body    string  `json:"body"`
	HTMLURL string  `json:"html_url"`
	Labels  []Label `json:"labels"`
}

// IssuesByLabel finds open issues carrying every one of the given labels.
func (c *Client) IssuesByLabel(ctx context.Context, labels ...string) ([]Issue, error) {
	q := url.Values{}
	q.Set("state", "open")
	q.Set("labels", strings.Join(labels, ","))
	return listAll[Issue](ctx, c, "/repos/"+c.Repo+"/issues?"+q.Encode())
}

func (c *Client) UpdateIssueBody(ctx context.Context, number int, body string) error {
	p := fmt.Sprintf("/repos/%s/issues/%d", c.Repo, number)
	return c.do(ctx, http.MethodPatch, p, map[string]string{"body": body}, nil)
}

func (c *Client) Comment(ctx context.Context, issue int, body string) error {
	p := fmt.Sprintf("/repos/%s/issues/%d/comments", c.Repo, issue)
	return c.do(ctx, http.MethodPost, p, map[string]string{"body": body}, nil)
}

// AddLabels adds labels without removing any. The tool never removes a label a
// human put on, because a maintainer who adds `breaking` to a pull request
// titled `feat:` is correcting the tool, not disagreeing with it by accident.
func (c *Client) AddLabels(ctx context.Context, issue int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	p := fmt.Sprintf("/repos/%s/issues/%d/labels", c.Repo, issue)
	return c.do(ctx, http.MethodPost, p, map[string][]string{"labels": labels}, nil)
}
