package engines

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Reddit implements Reddit search.
type Reddit struct {
	*BaseEngine
}

// NewReddit creates a new Reddit engine.
func NewReddit() *Reddit {
	r := &Reddit{
		BaseEngine: NewBaseEngine("reddit", "re", []Category{CategorySocial}),
	}

	r.SetPaging(false).
		SetTimeout(5 * time.Second).
		SetAbout(EngineAbout{
			Website:    "https://www.reddit.com",
			WikidataID: "Q1136",
			Results:    "JSON",
		})

	return r
}

func (r *Reddit) Request(ctx context.Context, query string, params *RequestParams) error {
	queryParams := url.Values{}
	queryParams.Set("q", query)
	queryParams.Set("limit", "25")

	params.URL = "https://www.reddit.com/search.json?" + queryParams.Encode()
	params.Headers.Set("Accept", "application/json")
	params.Headers.Set("User-Agent", "Mozilla/5.0 (compatible; SearXNG)")

	return nil
}

func (r *Reddit) Response(ctx context.Context, resp *http.Response, params *RequestParams) (*EngineResults, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := NewEngineResults()

	var apiResp redditResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	for _, child := range apiResp.Data.Children {
		post := child.Data

		result := Result{
			URL:     "https://www.reddit.com" + post.Permalink,
			Title:   post.Title,
			Content: truncateString(post.Selftext, 500),
		}

		// Parse timestamp
		if post.CreatedUTC > 0 {
			result.PublishedAt = time.Unix(int64(post.CreatedUTC), 0)
		}

		// Check if it has a valid thumbnail
		if isValidURL(post.Thumbnail) {
			result.ThumbnailURL = post.Thumbnail
			result.Template = "images"
			if isValidURL(post.URL) {
				result.ImageURL = post.URL
			}
		}

		result.ParsedURL, _ = url.Parse(result.URL)
		results.Add(result)
	}

	return results, nil
}

type redditResponse struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	Title      string  `json:"title"`
	Selftext   string  `json:"selftext"`
	Permalink  string  `json:"permalink"`
	URL        string  `json:"url"`
	Thumbnail  string  `json:"thumbnail"`
	CreatedUTC float64 `json:"created_utc"`
	Subreddit  string  `json:"subreddit"`
	Author     string  `json:"author"`
	Score      int     `json:"score"`
	NumComments int    `json:"num_comments"`
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func isValidURL(s string) bool {
	if s == "" || s == "self" || s == "default" || s == "nsfw" || s == "spoiler" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}
