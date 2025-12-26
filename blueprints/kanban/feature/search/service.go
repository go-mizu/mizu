package search

import (
	"context"

	"github.com/go-mizu/blueprints/kanban/feature/issues"
)

// Service implements the search API.
type Service struct {
	issues issues.API
}

// NewService creates a new search service.
func NewService(issues issues.API) *Service {
	return &Service{issues: issues}
}

func (s *Service) SearchIssues(ctx context.Context, projectID, query string, limit int) ([]*issues.Issue, error) {
	return s.issues.Search(ctx, projectID, query, limit)
}
