package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetComments fetches comments for a post by shortcode.
// Requires authentication (uses GraphQL query_hash).
func (c *Client) GetComments(ctx context.Context, shortcode string, maxComments int, cb ProgressCallback) ([]Comment, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("comments endpoint requires authentication (use --session or login first)")
	}

	if maxComments <= 0 {
		maxComments = 1000
	}

	var allComments []Comment
	cursor := ""

	for {
		if err := c.delay(ctx); err != nil {
			return allComments, err
		}

		vars := map[string]any{
			"shortcode": shortcode,
			"first":     CommentsPerPage,
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		data, err := c.graphQL(ctx, HashComments, vars)
		if err != nil {
			return allComments, fmt.Errorf("fetch comments: %w", err)
		}

		var resp graphQLResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return allComments, fmt.Errorf("parse comments: %w", err)
		}

		var conn *mediaConnection
		if resp.Data.User != nil && resp.Data.User.EdgeMediaToComment != nil {
			conn = resp.Data.User.EdgeMediaToComment
		}
		if conn == nil {
			// Try alternate response structure
			var altResp struct {
				Data struct {
					ShortcodeMedia *struct {
						EdgeMediaToParentComment *mediaConnection `json:"edge_media_to_parent_comment"`
						EdgeMediaToComment       *mediaConnection `json:"edge_media_to_comment"`
					} `json:"shortcode_media"`
				} `json:"data"`
			}
			if json.Unmarshal(data, &altResp) == nil && altResp.Data.ShortcodeMedia != nil {
				if altResp.Data.ShortcodeMedia.EdgeMediaToParentComment != nil {
					conn = altResp.Data.ShortcodeMedia.EdgeMediaToParentComment
				} else {
					conn = altResp.Data.ShortcodeMedia.EdgeMediaToComment
				}
			}
		}

		if conn == nil {
			break
		}

		for _, e := range conn.Edges {
			comment := Comment{
				ID:         e.Node.ID,
				Text:       e.Node.Text,
				AuthorID:   e.Node.Owner.ID,
				AuthorName: e.Node.Owner.Username,
				CreatedAt:  time.Unix(e.Node.CreatedAt, 0),
				PostID:     shortcode,
			}
			if e.Node.EdgeLikedBy.Count > 0 {
				comment.LikeCount = e.Node.EdgeLikedBy.Count
			}
			allComments = append(allComments, comment)
		}

		if cb != nil {
			cb(Progress{Phase: "comments", Total: conn.Count, Current: int64(len(allComments))})
		}

		if len(allComments) >= maxComments || !conn.PageInfo.HasNextPage || conn.PageInfo.EndCursor == "" {
			break
		}
		cursor = conn.PageInfo.EndCursor
	}

	if maxComments > 0 && len(allComments) > maxComments {
		allComments = allComments[:maxComments]
	}

	if cb != nil {
		cb(Progress{Phase: "comments", Total: int64(len(allComments)), Current: int64(len(allComments)), Done: true})
	}

	return allComments, nil
}
