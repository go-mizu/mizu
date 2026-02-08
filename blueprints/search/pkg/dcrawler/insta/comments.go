package insta

import (
	"context"
	"fmt"
)

// GetComments fetches comments for a post by shortcode.
// Note: The GraphQL query_hash for comments has been retired by Instagram.
// This endpoint currently requires authentication.
func (c *Client) GetComments(ctx context.Context, shortcode string, maxComments int, cb ProgressCallback) ([]Comment, error) {
	return nil, fmt.Errorf("comments endpoint requires authentication (GraphQL query_hash retired by Instagram)")
}
