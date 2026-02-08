package insta

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetPostLikes fetches the list of users who liked a post.
// Requires authentication. Uses GraphQL query_hash pagination.
func (c *Client) GetPostLikes(ctx context.Context, shortcode string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("post likes endpoint requires authentication")
	}

	if maxUsers <= 0 {
		maxUsers = 1000
	}

	var allUsers []FollowUser
	cursor := ""

	for {
		if err := c.doSleep(ctx); err != nil {
			return allUsers, err
		}

		vars := map[string]any{
			"shortcode": shortcode,
			"first":     PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, HashPostLikes, vars)
		if err != nil {
			return allUsers, fmt.Errorf("fetch post likes: %w", err)
		}

		var resp likesResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allUsers, fmt.Errorf("parse likes response: %w", err)
		}

		if resp.Data.ShortcodeMedia == nil || resp.Data.ShortcodeMedia.EdgeLikedBy == nil {
			break
		}

		conn := resp.Data.ShortcodeMedia.EdgeLikedBy
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
			cb(Progress{Phase: "likes", Total: conn.Count, Current: int64(len(allUsers))})
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
		cb(Progress{Phase: "likes", Total: int64(len(allUsers)), Current: int64(len(allUsers)), Done: true})
	}

	return allUsers, nil
}

// GetCommentLikes fetches the list of users who liked a comment.
// Requires authentication. Uses GraphQL query_hash pagination.
func (c *Client) GetCommentLikes(ctx context.Context, commentID string, maxUsers int, cb ProgressCallback) ([]FollowUser, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("comment likes endpoint requires authentication")
	}

	if maxUsers <= 0 {
		maxUsers = 1000
	}

	var allUsers []FollowUser
	cursor := ""

	for {
		if err := c.doSleep(ctx); err != nil {
			return allUsers, err
		}

		vars := map[string]any{
			"comment_id": commentID,
			"first":      PostsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQLWithAutoReduce(ctx, HashCommentLikes, vars)
		if err != nil {
			return allUsers, fmt.Errorf("fetch comment likes: %w", err)
		}

		var resp commentLikesResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allUsers, fmt.Errorf("parse comment likes response: %w", err)
		}

		if resp.Data.Comment == nil || resp.Data.Comment.EdgeLikedBy == nil {
			break
		}

		conn := resp.Data.Comment.EdgeLikedBy
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
			cb(Progress{Phase: "comment_likes", Total: conn.Count, Current: int64(len(allUsers))})
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
		cb(Progress{Phase: "comment_likes", Total: int64(len(allUsers)), Current: int64(len(allUsers)), Done: true})
	}

	return allUsers, nil
}
