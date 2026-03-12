package insta

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetFollowers fetches the follower list for a user.
// Requires authentication. Uses GraphQL query_hash pagination.
func (c *Client) GetFollowers(ctx context.Context, userID string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	return c.getFollowList(ctx, userID, HashFollowers, "edge_followed_by", maxUsers, "followers", cb)
}

// GetFollowing fetches the following list for a user.
// Requires authentication. Uses GraphQL query_hash pagination.
func (c *Client) GetFollowing(ctx context.Context, userID string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	return c.getFollowList(ctx, userID, HashFollowing, "edge_follow", maxUsers, "following", cb)
}

func (c *Client) getFollowList(ctx context.Context, userID, queryHash, edgeName string, maxUsers int, phase string, cb ProgressCallback) ([]FollowUser, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("%s endpoint requires authentication", phase)
	}

	if maxUsers <= 0 {
		maxUsers = 10000
	}

	var allUsers []FollowUser
	cursor := ""

	for {
		if err := c.doSleep(ctx); err != nil {
			return allUsers, err
		}

		vars := map[string]any{
			"id":    userID,
			"first": PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, queryHash, vars)
		if err != nil {
			return allUsers, fmt.Errorf("fetch %s: %w", phase, err)
		}

		var resp followersResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allUsers, fmt.Errorf("parse %s response: %w", phase, err)
		}

		if resp.Data.User == nil {
			break
		}

		var conn *followConnection
		switch edgeName {
		case "edge_followed_by":
			conn = resp.Data.User.EdgeFollowedBy
		case "edge_follow":
			conn = resp.Data.User.EdgeFollow
		}

		if conn == nil {
			break
		}

		for _, e := range conn.Edges {
			allUsers = append(allUsers, FollowUser{
				ID:         e.Node.ID,
				Username:   e.Node.Username,
				FullName:   e.Node.FullName,
				IsPrivate:  e.Node.IsPrivate,
				IsVerified: e.Node.IsVerified,
				PicURL:     e.Node.PicURL,
			})
		}

		if cb != nil {
			cb(Progress{Phase: phase, Total: conn.Count, Current: int64(len(allUsers))})
		}

		if len(allUsers) >= maxUsers || !conn.PageInfo.HasNextPage || conn.PageInfo.EndCursor == "" {
			break
		}
		cursor = conn.PageInfo.EndCursor
	}

	if maxUsers > 0 && len(allUsers) > maxUsers {
		allUsers = allUsers[:maxUsers]
	}

	if cb != nil {
		cb(Progress{Phase: phase, Total: int64(len(allUsers)), Current: int64(len(allUsers)), Done: true})
	}

	return allUsers, nil
}
