package insta

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetSavedPosts fetches the logged-in user's saved posts.
// Only works for the authenticated user's own saved posts.
// Requires authentication. Uses GraphQL query_hash pagination.
func (c *Client) GetSavedPosts(ctx context.Context, maxPosts int, cb ProgressCallback) ([]Post, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("saved posts endpoint requires authentication")
	}

	if c.userID == "" {
		return nil, fmt.Errorf("user ID not available (login required)")
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
			"id":    c.userID,
			"first": PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, HashSavedPosts, vars)
		if err != nil {
			return allPosts, fmt.Errorf("fetch saved posts: %w", err)
		}

		var resp struct {
			Data struct {
				User *struct {
					EdgeSavedMedia *mediaConnection `json:"edge_saved_media"`
				} `json:"user"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return allPosts, fmt.Errorf("parse saved posts response: %w", err)
		}

		if resp.Data.User == nil || resp.Data.User.EdgeSavedMedia == nil {
			break
		}

		conn := resp.Data.User.EdgeSavedMedia
		for _, e := range conn.Edges {
			allPosts = append(allPosts, nodeToPost(e.Node))
		}

		if cb != nil {
			cb(Progress{Phase: "saved", Total: conn.Count, Current: int64(len(allPosts))})
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
		cb(Progress{Phase: "saved", Total: int64(len(allPosts)), Current: int64(len(allPosts)), Done: true})
	}

	return allPosts, nil
}
