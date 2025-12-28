package search

import (
	"context"
)

// Service implements the search API
type Service struct {
	store   Store
	baseURL string
}

// NewService creates a new search service
func NewService(store Store, baseURL string) *Service {
	return &Service{
		store:   store,
		baseURL: baseURL,
	}
}

// Code searches code
func (s *Service) Code(ctx context.Context, query string, opts *SearchCodeOpts) (*Result[CodeResult], error) {
	if opts == nil {
		opts = &SearchCodeOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchCode(ctx, query, opts)
}

// Commits searches commits
func (s *Service) Commits(ctx context.Context, query string, opts *SearchOpts) (*Result[CommitResult], error) {
	if opts == nil {
		opts = &SearchOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchCommits(ctx, query, opts)
}

// IssuesAndPullRequests searches issues and PRs
func (s *Service) IssuesAndPullRequests(ctx context.Context, query string, opts *SearchIssuesOpts) (*Result[Issue], error) {
	if opts == nil {
		opts = &SearchIssuesOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchIssues(ctx, query, opts)
}

// Labels searches labels
func (s *Service) Labels(ctx context.Context, repoID int64, query string, opts *SearchOpts) (*Result[Label], error) {
	if opts == nil {
		opts = &SearchOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchLabels(ctx, repoID, query, opts)
}

// Repositories searches repositories
func (s *Service) Repositories(ctx context.Context, query string, opts *SearchReposOpts) (*Result[Repository], error) {
	if opts == nil {
		opts = &SearchReposOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchRepositories(ctx, query, opts)
}

// Topics searches topics
func (s *Service) Topics(ctx context.Context, query string, opts *SearchOpts) (*Result[Topic], error) {
	if opts == nil {
		opts = &SearchOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchTopics(ctx, query, opts)
}

// Users searches users
func (s *Service) Users(ctx context.Context, query string, opts *SearchUsersOpts) (*Result[User], error) {
	if opts == nil {
		opts = &SearchUsersOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.SearchUsers(ctx, query, opts)
}
