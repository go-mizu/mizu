package pulls

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the pulls API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new pulls service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// List returns PRs for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30, State: "open"}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.State == "" {
		opts.State = "open"
	}

	prs, err := s.store.List(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, pr := range prs {
		s.populateURLs(pr, owner, repo)
	}
	return prs, nil
}

// Get retrieves a PR by number
func (s *Service) Get(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(pr, owner, repo)
	return pr, nil
}

// Create creates a new PR
func (s *Service) Create(ctx context.Context, owner, repo string, creatorID int64, in *CreateIn) (*PullRequest, error) {
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

	number, err := s.store.NextNumber(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	pr := &PullRequest{
		Number:              number,
		State:               "open",
		Title:               in.Title,
		Body:                in.Body,
		User:                creator.ToSimple(),
		Labels:              []*Label{},
		Assignees:           []*users.SimpleUser{},
		RequestedReviewers:  []*users.SimpleUser{},
		RequestedTeams:      []*TeamSimple{},
		Draft:               in.Draft,
		MaintainerCanModify: in.MaintainerCanModify,
		Head: &PRBranch{
			Ref:   in.Head,
			Label: fmt.Sprintf("%s:%s", owner, in.Head),
			User:  creator.ToSimple(),
		},
		Base: &PRBranch{
			Ref:   in.Base,
			Label: fmt.Sprintf("%s:%s", owner, in.Base),
		},
		CreatedAt:         now,
		UpdatedAt:         now,
		AuthorAssociation: s.getAuthorAssociation(ctx, r, creatorID),
		RepoID:            r.ID,
		CreatorID:         creatorID,
	}

	if err := s.store.Create(ctx, pr); err != nil {
		return nil, err
	}

	s.populateURLs(pr, owner, repo)
	return pr, nil
}

// Update updates a PR
func (s *Service) Update(ctx context.Context, owner, repo string, number int, in *UpdateIn) (*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, pr.ID, in); err != nil {
		return nil, err
	}

	return s.Get(ctx, owner, repo, number)
}

// ListCommits returns commits in a PR
func (s *Service) ListCommits(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Commit, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	// Would integrate with git to list commits between base and head
	return []*Commit{}, nil
}

// ListFiles returns files in a PR
func (s *Service) ListFiles(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*PRFile, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	// Would integrate with git to list changed files
	return []*PRFile{}, nil
}

// IsMerged checks if a PR is merged
func (s *Service) IsMerged(ctx context.Context, owner, repo string, number int) (bool, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	if r == nil {
		return false, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return false, err
	}
	if pr == nil {
		return false, ErrNotFound
	}

	return pr.Merged, nil
}

// Merge merges a PR
func (s *Service) Merge(ctx context.Context, owner, repo string, number int, in *MergeIn) (*MergeResult, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	if pr.Merged {
		return nil, ErrAlreadyMerged
	}

	if pr.State != "open" {
		return nil, ErrNotMergeable
	}

	// Check SHA if provided
	if in.SHA != "" && pr.Head != nil && pr.Head.SHA != in.SHA {
		return nil, ErrNotMergeable
	}

	// Perform merge - would integrate with git
	mergeCommitSHA := fmt.Sprintf("merge_%d", pr.ID) // Placeholder
	now := time.Now()

	if err := s.store.SetMerged(ctx, pr.ID, now, mergeCommitSHA, 0); err != nil {
		return nil, err
	}

	return &MergeResult{
		SHA:     mergeCommitSHA,
		Merged:  true,
		Message: "Pull Request successfully merged",
	}, nil
}

// UpdateBranch updates a PR branch
func (s *Service) UpdateBranch(ctx context.Context, owner, repo string, number int) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return err
	}
	if pr == nil {
		return ErrNotFound
	}

	// Would integrate with git to update branch
	return nil
}

// ListReviews returns reviews for a PR
func (s *Service) ListReviews(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Review, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListReviews(ctx, pr.ID, opts)
}

// GetReview retrieves a review by ID
func (s *Service) GetReview(ctx context.Context, owner, repo string, number int, reviewID int64) (*Review, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	_, err = s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}

	return s.store.GetReviewByID(ctx, reviewID)
}

// CreateReview creates a review
func (s *Service) CreateReview(ctx context.Context, owner, repo string, number int, userID int64, in *CreateReviewIn) (*Review, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	state := "PENDING"
	if in.Event != "" {
		state = in.Event
	}

	review := &Review{
		User:              user.ToSimple(),
		Body:              in.Body,
		State:             state,
		CommitID:          in.CommitID,
		SubmittedAt:       time.Now(),
		AuthorAssociation: s.getAuthorAssociation(ctx, r, userID),
	}

	if err := s.store.CreateReview(ctx, review); err != nil {
		return nil, err
	}

	s.populateReviewURLs(review, owner, repo, number)
	return review, nil
}

// UpdateReview updates a review
func (s *Service) UpdateReview(ctx context.Context, owner, repo string, number int, reviewID int64, body string) (*Review, error) {
	if err := s.store.UpdateReview(ctx, reviewID, body); err != nil {
		return nil, err
	}
	return s.GetReview(ctx, owner, repo, number, reviewID)
}

// SubmitReview submits a pending review
func (s *Service) SubmitReview(ctx context.Context, owner, repo string, number int, reviewID int64, in *SubmitReviewIn) (*Review, error) {
	review, err := s.store.GetReviewByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if review == nil {
		return nil, ErrNotFound
	}

	if err := s.store.SetReviewState(ctx, reviewID, in.Event); err != nil {
		return nil, err
	}

	if in.Body != "" {
		if err := s.store.UpdateReview(ctx, reviewID, in.Body); err != nil {
			return nil, err
		}
	}

	return s.GetReview(ctx, owner, repo, number, reviewID)
}

// DismissReview dismisses a review
func (s *Service) DismissReview(ctx context.Context, owner, repo string, number int, reviewID int64, message string) (*Review, error) {
	if err := s.store.SetReviewState(ctx, reviewID, "DISMISSED"); err != nil {
		return nil, err
	}
	return s.GetReview(ctx, owner, repo, number, reviewID)
}

// ListReviewComments returns review comments for a PR
func (s *Service) ListReviewComments(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*ReviewComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}

	return s.store.ListReviewComments(ctx, pr.ID, opts)
}

// CreateReviewComment creates a review comment
func (s *Service) CreateReviewComment(ctx context.Context, owner, repo string, number int, userID int64, in *CreateReviewCommentIn) (*ReviewComment, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	now := time.Now()
	comment := &ReviewComment{
		Body:              in.Body,
		Path:              in.Path,
		Position:          in.Position,
		CommitID:          in.CommitID,
		OriginalCommitID:  in.CommitID,
		User:              user.ToSimple(),
		CreatedAt:         now,
		UpdatedAt:         now,
		Side:              in.Side,
		Line:              in.Line,
		StartLine:         in.StartLine,
		StartSide:         in.StartSide,
		InReplyToID:       in.InReplyTo,
		AuthorAssociation: s.getAuthorAssociation(ctx, r, userID),
	}

	if err := s.store.CreateReviewComment(ctx, comment); err != nil {
		return nil, err
	}

	s.populateReviewCommentURLs(comment, owner, repo, number)
	return comment, nil
}

// RequestReviewers adds reviewers to a PR
func (s *Service) RequestReviewers(ctx context.Context, owner, repo string, number int, reviewers, teamReviewers []string) (*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	for _, login := range reviewers {
		user, err := s.userStore.GetByLogin(ctx, login)
		if err != nil || user == nil {
			continue
		}
		_ = s.store.AddRequestedReviewer(ctx, pr.ID, user.ID)
	}

	return s.Get(ctx, owner, repo, number)
}

// RemoveReviewers removes reviewers from a PR
func (s *Service) RemoveReviewers(ctx context.Context, owner, repo string, number int, reviewers, teamReviewers []string) (*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	pr, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, ErrNotFound
	}

	for _, login := range reviewers {
		user, err := s.userStore.GetByLogin(ctx, login)
		if err != nil || user == nil {
			continue
		}
		_ = s.store.RemoveRequestedReviewer(ctx, pr.ID, user.ID)
	}

	return s.Get(ctx, owner, repo, number)
}

// getAuthorAssociation determines the relationship between user and repo
func (s *Service) getAuthorAssociation(ctx context.Context, r *repos.Repository, userID int64) string {
	if r.OwnerID == userID {
		return "OWNER"
	}
	return "NONE"
}

// populateURLs fills in the URL fields for a pull request
func (s *Service) populateURLs(pr *PullRequest, owner, repo string) {
	pr.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("PullRequest:%d", pr.ID)))
	pr.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/%d", s.baseURL, owner, repo, pr.Number)
	pr.HTMLURL = fmt.Sprintf("%s/%s/%s/pull/%d", s.baseURL, owner, repo, pr.Number)
	pr.DiffURL = fmt.Sprintf("%s/%s/%s/pull/%d.diff", s.baseURL, owner, repo, pr.Number)
	pr.PatchURL = fmt.Sprintf("%s/%s/%s/pull/%d.patch", s.baseURL, owner, repo, pr.Number)
	pr.IssueURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d", s.baseURL, owner, repo, pr.Number)
	pr.CommitsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/%d/commits", s.baseURL, owner, repo, pr.Number)
	pr.ReviewCommentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/%d/comments", s.baseURL, owner, repo, pr.Number)
	pr.ReviewCommentURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/comments{/number}", s.baseURL, owner, repo)
	pr.CommentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/issues/%d/comments", s.baseURL, owner, repo, pr.Number)
	pr.StatusesURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/statuses/{sha}", s.baseURL, owner, repo)
}

// populateReviewURLs fills in the URL fields for a review
func (s *Service) populateReviewURLs(r *Review, owner, repo string, number int) {
	r.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Review:%d", r.ID)))
	r.HTMLURL = fmt.Sprintf("%s/%s/%s/pull/%d#pullrequestreview-%d", s.baseURL, owner, repo, number, r.ID)
	r.PullRequestURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/%d", s.baseURL, owner, repo, number)
}

// populateReviewCommentURLs fills in the URL fields for a review comment
func (s *Service) populateReviewCommentURLs(c *ReviewComment, owner, repo string, number int) {
	c.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("ReviewComment:%d", c.ID)))
	c.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/comments/%d", s.baseURL, owner, repo, c.ID)
	c.HTMLURL = fmt.Sprintf("%s/%s/%s/pull/%d#discussion_r%d", s.baseURL, owner, repo, number, c.ID)
	c.PullRequestURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls/%d", s.baseURL, owner, repo, number)
}
