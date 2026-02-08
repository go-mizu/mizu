package insta

import (
	"context"
	"fmt"
)

// GetHashtagPosts fetches posts for a hashtag.
// Note: The GraphQL query_hash for hashtag feeds has been retired by Instagram.
// This endpoint currently requires authentication.
func (c *Client) GetHashtagPosts(ctx context.Context, tag string, maxPosts int, cb ProgressCallback) ([]Post, error) {
	return nil, fmt.Errorf("hashtag endpoint requires authentication (GraphQL query_hash retired by Instagram)")
}
