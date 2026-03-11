package insta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GetReels fetches reels for a user.
// Requires authentication. Uses doc_id POST-based GraphQL.
func (c *Client) GetReels(ctx context.Context, userID string, maxReels int, cb ProgressCallback) ([]Reel, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("reels endpoint requires authentication")
	}

	if maxReels <= 0 {
		maxReels = 1000
	}

	var allReels []Reel
	seen := make(map[string]bool)
	var cursor *string

	for {
		if maxReels > 0 && len(allReels) >= maxReels {
			break
		}
		if cursor != nil {
			if err := c.doSleep(ctx); err != nil {
				break
			}
		}

		vars := map[string]any{
			"data": map[string]any{
				"page_size":          PostsPerPage,
				"include_feed_video": true,
				"target_user_id":     userID,
			},
			"after":  cursor,
			"before": nil,
			"first":  PostsPerPage,
			"last":   nil,
			"__relay_internal__pv__PolarisFeedShareMenurelayprovider": false,
		}

		data, err := c.docIDQuery(ctx, DocIDProfileReels, vars)
		if err != nil {
			if len(allReels) > 0 {
				break
			}
			return nil, fmt.Errorf("fetch reels: %w", err)
		}

		reels, nextCursor, hasMore := parseReelsResponse(data)
		for _, r := range reels {
			if !seen[r.ID] {
				allReels = append(allReels, r)
				seen[r.ID] = true
			}
		}

		if cb != nil {
			cb(Progress{Phase: "reels", Total: int64(maxReels), Current: int64(len(allReels))})
		}

		if !hasMore || nextCursor == "" || len(reels) == 0 {
			break
		}
		c := nextCursor
		cursor = &c
	}

	if maxReels > 0 && len(allReels) > maxReels {
		allReels = allReels[:maxReels]
	}

	if cb != nil {
		cb(Progress{Phase: "reels", Total: int64(len(allReels)), Current: int64(len(allReels)), Done: true})
	}

	return allReels, nil
}

func parseReelsResponse(data []byte) (reels []Reel, cursor string, hasMore bool) {
	// Try XDT reels format
	var resp reelsResponse
	if err := json.Unmarshal(data, &resp); err != nil || resp.Data.Conn == nil {
		// Fallback: try the user timeline format (reels may come as posts)
		var fallback struct {
			Data struct {
				Conn *xdtConnection `json:"xdt_api__v1__feed__user_timeline_graphql_connection"`
			} `json:"data"`
		}
		if json.Unmarshal(data, &fallback) != nil || fallback.Data.Conn == nil {
			return nil, "", false
		}
		conn := fallback.Data.Conn
		for _, e := range conn.Edges {
			reels = append(reels, xdtNodeToReel(e.Node))
		}
		return reels, conn.PageInfo.EndCursor, conn.PageInfo.HasNextPage
	}

	conn := resp.Data.Conn
	for _, e := range conn.Edges {
		reels = append(reels, xdtNodeToReel(e.Node))
	}
	return reels, conn.PageInfo.EndCursor, conn.PageInfo.HasNextPage
}

func xdtNodeToReel(n xdtNode) Reel {
	reel := Reel{
		ID:           n.ID,
		Shortcode:    n.Code,
		LikeCount:    n.LikeCount,
		CommentCount: n.CommentCount,
		ViewCount:    n.ViewCount,
		TakenAt:      time.Unix(n.TakenAt, 0),
		OwnerID:      n.User.PK,
		OwnerName:    n.User.Username,
		Width:        n.OriginalWidth,
		Height:       n.OriginalHeight,
	}

	if n.Caption != nil {
		reel.Caption = n.Caption.Text
	}

	if len(n.VideoVersions) > 0 {
		reel.VideoURL = n.VideoVersions[0].URL
		if reel.Width == 0 {
			reel.Width = n.VideoVersions[0].Width
			reel.Height = n.VideoVersions[0].Height
		}
	}
	if n.ImageVersions2 != nil && len(n.ImageVersions2.Candidates) > 0 {
		reel.DisplayURL = n.ImageVersions2.Candidates[0].URL
	}

	return reel
}
