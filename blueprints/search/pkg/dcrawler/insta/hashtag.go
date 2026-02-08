package insta

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetHashtagPosts fetches posts for a hashtag.
// Requires authentication (uses GraphQL query_hash).
func (c *Client) GetHashtagPosts(ctx context.Context, tag string, maxPosts int, cb ProgressCallback) ([]Post, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("hashtag endpoint requires authentication (use --session or login first)")
	}

	if maxPosts <= 0 {
		maxPosts = 1000
	}

	var allPosts []Post
	cursor := ""

	for {
		if err := c.doSleep(ctx); err != nil {
			return allPosts, err
		}

		vars := map[string]any{
			"tag_name": tag,
			"first":    PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, HashHashtagFeed, vars)
		if err != nil {
			return allPosts, fmt.Errorf("fetch hashtag %q: %w", tag, err)
		}

		var resp graphQLResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allPosts, fmt.Errorf("parse hashtag response: %w", err)
		}

		if resp.Data.Hashtag == nil || resp.Data.Hashtag.EdgeHashtagToMedia == nil {
			break
		}

		conn := resp.Data.Hashtag.EdgeHashtagToMedia
		for _, e := range conn.Edges {
			allPosts = append(allPosts, nodeToPost(e.Node))
		}

		if cb != nil {
			cb(Progress{Phase: "hashtag", Total: conn.Count, Current: int64(len(allPosts))})
		}

		if len(allPosts) >= maxPosts || !conn.PageInfo.HasNextPage || conn.PageInfo.EndCursor == "" {
			break
		}
		cursor = conn.PageInfo.EndCursor
	}

	if maxPosts > 0 && len(allPosts) > maxPosts {
		allPosts = allPosts[:maxPosts]
	}

	if cb != nil {
		cb(Progress{Phase: "hashtag", Total: int64(len(allPosts)), Current: int64(len(allPosts)), Done: true})
	}

	return allPosts, nil
}
