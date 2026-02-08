package insta

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetLocationPosts fetches posts for a location by ID.
// Requires authentication (GraphQL query_hash approach).
func (c *Client) GetLocationPosts(ctx context.Context, locationID string, maxPosts int, cb ProgressCallback) ([]Post, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("location endpoint requires authentication (use --session or login first)")
	}

	if maxPosts <= 0 {
		maxPosts = 1000
	}

	var allPosts []Post
	cursor := ""

	for {
		if err := c.delay(ctx); err != nil {
			return allPosts, err
		}

		vars := map[string]any{
			"id":    locationID,
			"first": PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, HashLocationFeed, vars)
		if err != nil {
			return allPosts, fmt.Errorf("fetch location %q: %w", locationID, err)
		}

		var resp locationFeedResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allPosts, fmt.Errorf("parse location response: %w", err)
		}

		if resp.Data.Location == nil || resp.Data.Location.EdgeLocationToMedia == nil {
			break
		}

		conn := resp.Data.Location.EdgeLocationToMedia
		for _, e := range conn.Edges {
			allPosts = append(allPosts, nodeToPost(e.Node))
		}

		if cb != nil {
			cb(Progress{Phase: "location", Total: conn.Count, Current: int64(len(allPosts))})
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
		cb(Progress{Phase: "location", Total: int64(len(allPosts)), Current: int64(len(allPosts)), Done: true})
	}

	return allPosts, nil
}
