package comments

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the comments API
type Service struct {
	store      Store
	repoStore  repos.Store
	issueStore issues.Store
	userStore  users.Store
	baseURL    string
}

// NewService creates a new comments service
func NewService(store Store, repoStore repos.Store, issueStore issues.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:      store,
		repoStore:  repoStore,
		issueStore: issueStore,
		userStore:  userStore,
		baseURL:    baseURL,
	}
}

// ListForRepo returns issue comments for a repository
func (s *Service) ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*IssueComment, error) {
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
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	comments, err := s.store.ListIssueCommentsForRepo(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, c := range comments {
		s.populateIssueCommentURLs(c, owner, repo)
	}
	return comments, nil
}

// ListForIssue returns comments for an issue
func (s *Service) ListForIssue(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*IssueComment, error) {
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

	comments, err := s.store.ListIssueCommentsForIssue(ctx, issue.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, c := range comments {
		s.populateIssueCommentURLs(c, owner, repo)
	}
	return comments, nil
}

// ListForPR returns comments for a pull request using PR ID directly
func (s *Service) ListForPR(ctx context.Context, owner, repo string, prID int64, opts *ListOpts) ([]*IssueComment, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	// PR comments are stored with issue_id = prID
	comments, err := s.store.ListIssueCommentsForIssue(ctx, prID, opts)
	if err != nil {
		return nil, err
	}

	for _, c := range comments {
		s.populateIssueCommentURLs(c, owner, repo)
	}
	return comments, nil
}

// ListUniqueCommentersForIssue returns unique users who commented on an issue
func (s *Service) ListUniqueCommentersForIssue(ctx context.Context, owner, repo string, number int) ([]*users.SimpleUser, error) {
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

	return s.store.ListUniqueCommenters(ctx, issue.ID)
}

// GetIssueComment retrieves an issue comment by ID
func (s *Service) GetIssueComment(ctx context.Context, owner, repo string, commentID int64) (*IssueComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.store.GetIssueCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	if c.RepoID != r.ID {
		return nil, ErrNotFound
	}

	s.populateIssueCommentURLs(c, owner, repo)
	return c, nil
}

// CreateIssueComment creates a comment on an issue
func (s *Service) CreateIssueComment(ctx context.Context, owner, repo string, number int, creatorID int64, body string) (*IssueComment, error) {
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

	creator, err := s.userStore.GetByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, users.ErrNotFound
	}

	now := time.Now()
	c := &IssueComment{
		Body:              body,
		User:              creator.ToSimple(),
		CreatedAt:         now,
		UpdatedAt:         now,
		AuthorAssociation: s.getAuthorAssociation(ctx, r, creatorID),
		IssueID:           issue.ID,
		RepoID:            r.ID,
		CreatorID:         creatorID,
	}

	if err := s.store.CreateIssueComment(ctx, c); err != nil {
		return nil, err
	}

	// Increment issue comment count
	if err := s.issueStore.IncrementComments(ctx, issue.ID, 1); err != nil {
		return nil, err
	}

	s.populateIssueCommentURLs(c, owner, repo)
	return c, nil
}

// UpdateIssueComment updates an issue comment
func (s *Service) UpdateIssueComment(ctx context.Context, owner, repo string, commentID int64, body string) (*IssueComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.store.GetIssueCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	if c.RepoID != r.ID {
		return nil, ErrNotFound
	}

	if err := s.store.UpdateIssueComment(ctx, commentID, body); err != nil {
		return nil, err
	}

	return s.GetIssueComment(ctx, owner, repo, commentID)
}

// DeleteIssueComment deletes an issue comment
func (s *Service) DeleteIssueComment(ctx context.Context, owner, repo string, commentID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	c, err := s.store.GetIssueCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrNotFound
	}
	if c.RepoID != r.ID {
		return ErrNotFound
	}

	if err := s.store.DeleteIssueComment(ctx, commentID); err != nil {
		return err
	}

	// Decrement issue comment count
	return s.issueStore.IncrementComments(ctx, c.IssueID, -1)
}

// ListCommitCommentsForRepo returns commit comments for a repository
func (s *Service) ListCommitCommentsForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*CommitComment, error) {
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
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	comments, err := s.store.ListCommitCommentsForRepo(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, c := range comments {
		s.populateCommitCommentURLs(c, owner, repo)
	}
	return comments, nil
}

// ListForCommit returns comments for a commit
func (s *Service) ListForCommit(ctx context.Context, owner, repo, sha string, opts *ListOpts) ([]*CommitComment, error) {
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
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	comments, err := s.store.ListCommitCommentsForCommit(ctx, r.ID, sha, opts)
	if err != nil {
		return nil, err
	}

	for _, c := range comments {
		s.populateCommitCommentURLs(c, owner, repo)
	}
	return comments, nil
}

// GetCommitComment retrieves a commit comment by ID
func (s *Service) GetCommitComment(ctx context.Context, owner, repo string, commentID int64) (*CommitComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.store.GetCommitCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	if c.RepoID != r.ID {
		return nil, ErrNotFound
	}

	s.populateCommitCommentURLs(c, owner, repo)
	return c, nil
}

// CreateCommitComment creates a comment on a commit
func (s *Service) CreateCommitComment(ctx context.Context, owner, repo, sha string, creatorID int64, in *CreateCommitCommentIn) (*CommitComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	creator, err := s.userStore.GetByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, users.ErrNotFound
	}

	now := time.Now()
	c := &CommitComment{
		Body:      in.Body,
		User:      creator.ToSimple(),
		Path:      in.Path,
		Position:  in.Position,
		Line:      in.Line,
		CommitID:  sha,
		CreatedAt: now,
		UpdatedAt: now,
		RepoID:    r.ID,
		CreatorID: creatorID,
	}

	if err := s.store.CreateCommitComment(ctx, c); err != nil {
		return nil, err
	}

	s.populateCommitCommentURLs(c, owner, repo)
	return c, nil
}

// UpdateCommitComment updates a commit comment
func (s *Service) UpdateCommitComment(ctx context.Context, owner, repo string, commentID int64, body string) (*CommitComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	c, err := s.store.GetCommitCommentByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	if c.RepoID != r.ID {
		return nil, ErrNotFound
	}

	if err := s.store.UpdateCommitComment(ctx, commentID, body); err != nil {
		return nil, err
	}

	return s.GetCommitComment(ctx, owner, repo, commentID)
}

// DeleteCommitComment deletes a commit comment
func (s *Service) DeleteCommitComment(ctx context.Context, owner, repo string, commentID int64) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	c, err := s.store.GetCommitCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if c == nil {
		return ErrNotFound
	}
	if c.RepoID != r.ID {
		return ErrNotFound
	}

	return s.store.DeleteCommitComment(ctx, commentID)
}

// getAuthorAssociation determines the relationship between user and repo
func (s *Service) getAuthorAssociation(ctx context.Context, r *repos.Repository, userID int64) string {
	if r.OwnerID == userID {
		return "OWNER"
	}
	return "NONE"
}

// populateIssueCommentURLs fills in the URL fields for an issue comment
func (s *Service) populateIssueCommentURLs(c *IssueComment, owner, repo string) {
	c.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("IssueComment:%d", c.ID)))
	c.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/comments/%d", s.baseURL, owner, repo, c.ID)
	c.HTMLURL = fmt.Sprintf("%s/%s/%s/issues/%d#issuecomment-%d", s.baseURL, owner, repo, c.IssueID, c.ID)
	c.IssueURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d", s.baseURL, owner, repo, c.IssueID)
}

// populateCommitCommentURLs fills in the URL fields for a commit comment
func (s *Service) populateCommitCommentURLs(c *CommitComment, owner, repo string) {
	c.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("CommitComment:%d", c.ID)))
	c.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/comments/%d", s.baseURL, owner, repo, c.ID)
	c.HTMLURL = fmt.Sprintf("%s/%s/%s/commit/%s#commitcomment-%d", s.baseURL, owner, repo, c.CommitID, c.ID)
}
