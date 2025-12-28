package reactions

import (
	"context"
	"fmt"

	"github.com/go-mizu/blueprints/githome/feature/comments"
	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// Service implements the reactions API
type Service struct {
	store        Store
	repoStore    repos.Store
	issueStore   issues.Store
	commentStore comments.Store
	baseURL      string
}

// NewService creates a new reactions service
func NewService(store Store, repoStore repos.Store, issueStore issues.Store, commentStore comments.Store, baseURL string) *Service {
	return &Service{
		store:        store,
		repoStore:    repoStore,
		issueStore:   issueStore,
		commentStore: commentStore,
		baseURL:      baseURL,
	}
}

// ListForIssue returns reactions for an issue
func (s *Service) ListForIssue(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Reaction, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.issueStore.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, issues.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.List(ctx, "issue", issue.ID, opts)
}

// CreateForIssue creates a reaction for an issue
func (s *Service) CreateForIssue(ctx context.Context, owner, repo string, number int, userID int64, content string) (*Reaction, error) {
	if !ValidContent(content) {
		return nil, ErrInvalidContent
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	issue, err := s.issueStore.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if issue == nil {
		return nil, issues.ErrNotFound
	}

	// Check if reaction already exists
	existing, err := s.store.GetByUserAndContent(ctx, "issue", issue.ID, userID, content)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil // Idempotent - return existing reaction
	}

	return s.store.Create(ctx, "issue", issue.ID, userID, content)
}

// DeleteForIssue deletes a reaction from an issue
func (s *Service) DeleteForIssue(ctx context.Context, owner, repo string, number int, reactionID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	issue, err := s.issueStore.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return err
	}
	if issue == nil {
		return issues.ErrNotFound
	}

	reaction, err := s.store.GetByID(ctx, reactionID)
	if err != nil {
		return err
	}
	if reaction == nil {
		return ErrNotFound
	}

	return s.store.Delete(ctx, reactionID)
}

// ListForIssueComment returns reactions for an issue comment
func (s *Service) ListForIssueComment(ctx context.Context, owner, repo string, commentID int64, opts *ListOpts) ([]*Reaction, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.commentStore.GetIssueCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.RepoID != r.ID {
		return nil, comments.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.List(ctx, "issue_comment", commentID, opts)
}

// CreateForIssueComment creates a reaction for an issue comment
func (s *Service) CreateForIssueComment(ctx context.Context, owner, repo string, commentID int64, userID int64, content string) (*Reaction, error) {
	if !ValidContent(content) {
		return nil, ErrInvalidContent
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.commentStore.GetIssueCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.RepoID != r.ID {
		return nil, comments.ErrNotFound
	}

	// Check if reaction already exists
	existing, err := s.store.GetByUserAndContent(ctx, "issue_comment", commentID, userID, content)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	return s.store.Create(ctx, "issue_comment", commentID, userID, content)
}

// DeleteForIssueComment deletes a reaction from an issue comment
func (s *Service) DeleteForIssueComment(ctx context.Context, owner, repo string, commentID int64, reactionID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	c, err := s.commentStore.GetIssueCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil || c.RepoID != r.ID {
		return comments.ErrNotFound
	}

	return s.store.Delete(ctx, reactionID)
}

// ListForPullReviewComment returns reactions for a PR review comment
func (s *Service) ListForPullReviewComment(ctx context.Context, owner, repo string, commentID int64, opts *ListOpts) ([]*Reaction, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.List(ctx, "pull_request_review_comment", commentID, opts)
}

// CreateForPullReviewComment creates a reaction for a PR review comment
func (s *Service) CreateForPullReviewComment(ctx context.Context, owner, repo string, commentID int64, userID int64, content string) (*Reaction, error) {
	if !ValidContent(content) {
		return nil, ErrInvalidContent
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check if reaction already exists
	existing, err := s.store.GetByUserAndContent(ctx, "pull_request_review_comment", commentID, userID, content)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	return s.store.Create(ctx, "pull_request_review_comment", commentID, userID, content)
}

// DeleteForPullReviewComment deletes a reaction from a PR review comment
func (s *Service) DeleteForPullReviewComment(ctx context.Context, owner, repo string, commentID int64, reactionID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	return s.store.Delete(ctx, reactionID)
}

// ListForCommitComment returns reactions for a commit comment
func (s *Service) ListForCommitComment(ctx context.Context, owner, repo string, commentID int64, opts *ListOpts) ([]*Reaction, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.commentStore.GetCommitCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.RepoID != r.ID {
		return nil, comments.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.List(ctx, "commit_comment", commentID, opts)
}

// CreateForCommitComment creates a reaction for a commit comment
func (s *Service) CreateForCommitComment(ctx context.Context, owner, repo string, commentID int64, userID int64, content string) (*Reaction, error) {
	if !ValidContent(content) {
		return nil, ErrInvalidContent
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.commentStore.GetCommitCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil || c.RepoID != r.ID {
		return nil, comments.ErrNotFound
	}

	// Check if reaction already exists
	existing, err := s.store.GetByUserAndContent(ctx, "commit_comment", commentID, userID, content)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	return s.store.Create(ctx, "commit_comment", commentID, userID, content)
}

// DeleteForCommitComment deletes a reaction from a commit comment
func (s *Service) DeleteForCommitComment(ctx context.Context, owner, repo string, commentID int64, reactionID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	c, err := s.commentStore.GetCommitCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil || c.RepoID != r.ID {
		return comments.ErrNotFound
	}

	return s.store.Delete(ctx, reactionID)
}

// ListForRelease returns reactions for a release
func (s *Service) ListForRelease(ctx context.Context, owner, repo string, releaseID int64, opts *ListOpts) ([]*Reaction, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.List(ctx, "release", releaseID, opts)
}

// CreateForRelease creates a reaction for a release
func (s *Service) CreateForRelease(ctx context.Context, owner, repo string, releaseID int64, userID int64, content string) (*Reaction, error) {
	if !ValidContent(content) {
		return nil, ErrInvalidContent
	}

	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check if reaction already exists
	existing, err := s.store.GetByUserAndContent(ctx, "release", releaseID, userID, content)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	return s.store.Create(ctx, "release", releaseID, userID, content)
}

// DeleteForRelease deletes a reaction from a release
func (s *Service) DeleteForRelease(ctx context.Context, owner, repo string, releaseID int64, reactionID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	return s.store.Delete(ctx, reactionID)
}

// GetRollup returns reaction rollup for a subject
func (s *Service) GetRollup(ctx context.Context, subjectType string, subjectID int64) (*Reactions, error) {
	rollup, err := s.store.GetRollup(ctx, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	if rollup == nil {
		rollup = &Reactions{}
	}
	rollup.URL = fmt.Sprintf("%s/api/v3/reactions", s.baseURL)
	return rollup, nil
}
