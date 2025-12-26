// Package search provides search functionality.
package search

import (
	"context"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
)

// API defines the search service contract.
type API interface {
	SearchIssues(ctx context.Context, projectID, query string, limit int) ([]*issues.Issue, error)
}
