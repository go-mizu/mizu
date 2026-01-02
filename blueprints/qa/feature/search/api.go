package search

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/qa/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/qa/feature/questions"
	"github.com/go-mizu/mizu/blueprints/qa/feature/tags"
)

// Result contains search results.
type Result struct {
	Questions []*questions.Question
	Tags      []*tags.Tag
	Users     []*accounts.Account
}

// API defines the search service interface.
type API interface {
	Search(ctx context.Context, query string, limit int) (*Result, error)
}

// Service implements the search API.
type Service struct {
	questions questions.API
	tags      tags.API
	accounts  accounts.API
}

// NewService creates a new search service.
func NewService(questions questions.API, tags tags.API, accounts accounts.API) *Service {
	return &Service{questions: questions, tags: tags, accounts: accounts}
}

// Search searches across questions, tags, and users.
func (s *Service) Search(ctx context.Context, query string, limit int) (*Result, error) {
	if limit <= 0 {
		limit = 20
	}
	qs, _ := s.questions.Search(ctx, query, limit)
	tagsList, _ := s.tags.List(ctx, tags.ListOpts{Limit: 10, Query: query})
	users, _ := s.accounts.Search(ctx, query, 10)
	return &Result{
		Questions: qs,
		Tags:      tagsList,
		Users:     users,
	}, nil
}
