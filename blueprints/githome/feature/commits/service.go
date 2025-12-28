package commits

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the commits API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new commits service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// List returns commits for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Commit, error) {
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

	// Would integrate with git to list commits
	// For now return empty list
	return []*Commit{}, nil
}

// Get retrieves a commit by ref
func (s *Service) Get(ctx context.Context, owner, repo, ref string) (*Commit, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would integrate with git to get commit
	// For now return placeholder
	commit := &Commit{
		SHA:    ref,
		NodeID: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Commit:%s", ref))),
		Commit: &CommitData{
			Message: "Commit message",
			Author: &CommitAuthor{
				Name:  "Author",
				Email: "author@example.com",
				Date:  time.Now(),
			},
			Committer: &CommitAuthor{
				Name:  "Committer",
				Email: "committer@example.com",
				Date:  time.Now(),
			},
		},
		Parents: []*CommitRef{},
	}

	s.populateURLs(commit, owner, repo)
	return commit, nil
}

// Compare compares two commits
func (s *Service) Compare(ctx context.Context, owner, repo, base, head string) (*Comparison, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would integrate with git to compare commits
	comparison := &Comparison{
		Status:       "ahead",
		AheadBy:      0,
		BehindBy:     0,
		TotalCommits: 0,
		Commits:      []*Commit{},
		Files:        []*CommitFile{},
	}

	comparison.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/compare/%s...%s", s.baseURL, owner, repo, base, head)
	comparison.HTMLURL = fmt.Sprintf("%s/%s/%s/compare/%s...%s", s.baseURL, owner, repo, base, head)
	comparison.PermalinkURL = comparison.HTMLURL
	comparison.DiffURL = fmt.Sprintf("%s/%s/%s/compare/%s...%s.diff", s.baseURL, owner, repo, base, head)
	comparison.PatchURL = fmt.Sprintf("%s/%s/%s/compare/%s...%s.patch", s.baseURL, owner, repo, base, head)

	return comparison, nil
}

// ListBranchesForHead returns branches containing the commit
func (s *Service) ListBranchesForHead(ctx context.Context, owner, repo, sha string) ([]*Branch, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would integrate with git to find branches
	return []*Branch{}, nil
}

// ListPullsForCommit returns PRs associated with a commit
func (s *Service) ListPullsForCommit(ctx context.Context, owner, repo, sha string, opts *ListOpts) ([]*PullRequest, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Would search PRs by commit SHA
	return []*PullRequest{}, nil
}

// GetCombinedStatus returns combined status for a ref
func (s *Service) GetCombinedStatus(ctx context.Context, owner, repo, ref string) (*CombinedStatus, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	combined, err := s.store.GetCombinedStatus(ctx, r.ID, ref)
	if err != nil {
		return nil, err
	}
	if combined == nil {
		combined = &CombinedStatus{
			State:      "pending",
			Statuses:   []*Status{},
			SHA:        ref,
			TotalCount: 0,
			Repository: &Repository{
				ID:       r.ID,
				NodeID:   base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Repository:%d", r.ID))),
				Name:     r.Name,
				FullName: r.FullName,
			},
		}
	}

	combined.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s/status", s.baseURL, owner, repo, ref)
	combined.CommitURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, ref)
	if combined.Repository != nil {
		combined.Repository.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s", s.baseURL, owner, repo)
		combined.Repository.HTMLURL = fmt.Sprintf("%s/%s/%s", s.baseURL, owner, repo)
	}

	return combined, nil
}

// ListStatuses returns statuses for a ref
func (s *Service) ListStatuses(ctx context.Context, owner, repo, ref string, opts *ListOpts) ([]*Status, error) {
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

	statuses, err := s.store.ListStatuses(ctx, r.ID, ref, opts)
	if err != nil {
		return nil, err
	}

	for _, status := range statuses {
		s.populateStatusURLs(status, owner, repo)
	}
	return statuses, nil
}

// CreateStatus creates a status for a SHA
func (s *Service) CreateStatus(ctx context.Context, owner, repo, sha string, creatorID int64, in *CreateStatusIn) (*Status, error) {
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

	context := in.Context
	if context == "" {
		context = "default"
	}

	now := time.Now()
	status := &Status{
		State:       in.State,
		TargetURL:   in.TargetURL,
		Description: in.Description,
		Context:     context,
		CreatedAt:   now,
		UpdatedAt:   now,
		Creator:     creator.ToSimple(),
	}

	if err := s.store.CreateStatus(ctx, r.ID, sha, status); err != nil {
		return nil, err
	}

	s.populateStatusURLs(status, owner, repo)
	return status, nil
}

// populateURLs fills in the URL fields for a commit
func (s *Service) populateURLs(c *Commit, owner, repo string) {
	c.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s", s.baseURL, owner, repo, c.SHA)
	c.HTMLURL = fmt.Sprintf("%s/%s/%s/commit/%s", s.baseURL, owner, repo, c.SHA)
	c.CommentsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/commits/%s/comments", s.baseURL, owner, repo, c.SHA)
	if c.Commit != nil {
		c.Commit.URL = c.URL
	}
}

// populateStatusURLs fills in the URL fields for a status
func (s *Service) populateStatusURLs(status *Status, owner, repo string) {
	status.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Status:%d", status.ID)))
	status.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/statuses/%d", s.baseURL, owner, repo, status.ID)
}
