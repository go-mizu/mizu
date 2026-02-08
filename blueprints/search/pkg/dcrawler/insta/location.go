package insta

import (
	"context"
	"fmt"
)

// GetLocationPosts fetches posts for a location by ID.
// Note: The GraphQL query_hash for location feeds has been retired by Instagram.
// This endpoint currently requires authentication.
func (c *Client) GetLocationPosts(ctx context.Context, locationID string, maxPosts int, cb ProgressCallback) ([]Post, error) {
	return nil, fmt.Errorf("location endpoint requires authentication (GraphQL query_hash retired by Instagram)")
}
