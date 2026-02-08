package insta

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetTaggedPosts fetches posts where a user is tagged.
// Requires authentication. Uses GraphQL query_hash pagination.
func (c *Client) GetTaggedPosts(ctx context.Context, userID string, maxPosts int, cb ProgressCallback) ([]Post, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("tagged posts endpoint requires authentication")
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
			"id":    userID,
			"first": PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, HashTaggedPosts, vars)
		if err != nil {
			return allPosts, fmt.Errorf("fetch tagged posts: %w", err)
		}

		var resp struct {
			Data struct {
				User *struct {
					EdgeUserToPhotosOfYou *mediaConnection `json:"edge_user_to_photos_of_you"`
				} `json:"user"`
			} `json:"data"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return allPosts, fmt.Errorf("parse tagged posts response: %w", err)
		}

		if resp.Data.User == nil || resp.Data.User.EdgeUserToPhotosOfYou == nil {
			break
		}

		conn := resp.Data.User.EdgeUserToPhotosOfYou
		for _, e := range conn.Edges {
			allPosts = append(allPosts, nodeToPost(e.Node))
		}

		if cb != nil {
			cb(Progress{Phase: "tagged", Total: conn.Count, Current: int64(len(allPosts))})
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
		cb(Progress{Phase: "tagged", Total: int64(len(allPosts)), Current: int64(len(allPosts)), Done: true})
	}

	return allPosts, nil
}
