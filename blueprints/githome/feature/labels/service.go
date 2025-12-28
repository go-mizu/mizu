package labels

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/go-mizu/blueprints/githome/feature/issues"
	"github.com/go-mizu/blueprints/githome/feature/milestones"
	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// Service implements the labels API
type Service struct {
	store          Store
	repoStore      repos.Store
	issueStore     issues.Store
	milestoneStore milestones.Store
	baseURL        string
}

// NewService creates a new labels service
func NewService(store Store, repoStore repos.Store, issueStore issues.Store, milestoneStore milestones.Store, baseURL string) *Service {
	return &Service{
		store:          store,
		repoStore:      repoStore,
		issueStore:     issueStore,
		milestoneStore: milestoneStore,
		baseURL:        baseURL,
	}
}

// List returns labels for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Label, error) {
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

	labels, err := s.store.List(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		s.populateURLs(label, owner, repo)
	}
	return labels, nil
}

// Get retrieves a label by name
func (s *Service) Get(ctx context.Context, owner, repo, name string) (*Label, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	label, err := s.store.GetByName(ctx, r.ID, name)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(label, owner, repo)
	return label, nil
}

// GetByID retrieves a label by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Label, error) {
	label, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrNotFound
	}
	return label, nil
}

// Create creates a new label
func (s *Service) Create(ctx context.Context, owner, repo string, in *CreateIn) (*Label, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check if label exists
	existing, err := s.store.GetByName(ctx, r.ID, in.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrLabelExists
	}

	color := in.Color
	if color == "" {
		color = "ededed" // Default gray
	}

	label := &Label{
		Name:        in.Name,
		Description: in.Description,
		Color:       color,
		RepoID:      r.ID,
	}

	if err := s.store.Create(ctx, label); err != nil {
		return nil, err
	}

	s.populateURLs(label, owner, repo)
	return label, nil
}

// Update updates a label
func (s *Service) Update(ctx context.Context, owner, repo, name string, in *UpdateIn) (*Label, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	label, err := s.store.GetByName(ctx, r.ID, name)
	if err != nil {
		return nil, err
	}
	if label == nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, label.ID, in); err != nil {
		return nil, err
	}

	// Return with new name if changed
	newName := name
	if in.NewName != nil {
		newName = *in.NewName
	}
	return s.Get(ctx, owner, repo, newName)
}

// Delete removes a label
func (s *Service) Delete(ctx context.Context, owner, repo, name string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	label, err := s.store.GetByName(ctx, r.ID, name)
	if err != nil {
		return err
	}
	if label == nil {
		return ErrNotFound
	}

	return s.store.Delete(ctx, label.ID)
}

// ListForIssue returns labels for an issue
func (s *Service) ListForIssue(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Label, error) {
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
		opts = &ListOpts{PerPage: 100}
	}

	labels, err := s.store.ListForIssue(ctx, issue.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		s.populateURLs(label, owner, repo)
	}
	return labels, nil
}

// AddToIssue adds labels to an issue
func (s *Service) AddToIssue(ctx context.Context, owner, repo string, number int, labelNames []string) ([]*Label, error) {
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

	labels, err := s.store.GetByNames(ctx, r.ID, labelNames)
	if err != nil {
		return nil, err
	}

	for _, label := range labels {
		_ = s.store.AddToIssue(ctx, issue.ID, label.ID)
	}

	return s.ListForIssue(ctx, owner, repo, number, nil)
}

// SetForIssue replaces all labels on an issue
func (s *Service) SetForIssue(ctx context.Context, owner, repo string, number int, labelNames []string) ([]*Label, error) {
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

	labels, err := s.store.GetByNames(ctx, r.ID, labelNames)
	if err != nil {
		return nil, err
	}

	var labelIDs []int64
	for _, label := range labels {
		labelIDs = append(labelIDs, label.ID)
	}

	if err := s.store.SetForIssue(ctx, issue.ID, labelIDs); err != nil {
		return nil, err
	}

	return s.ListForIssue(ctx, owner, repo, number, nil)
}

// RemoveFromIssue removes a label from an issue
func (s *Service) RemoveFromIssue(ctx context.Context, owner, repo string, number int, name string) error {
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

	label, err := s.store.GetByName(ctx, r.ID, name)
	if err != nil {
		return err
	}
	if label == nil {
		return ErrNotFound
	}

	return s.store.RemoveFromIssue(ctx, issue.ID, label.ID)
}

// RemoveAllFromIssue removes all labels from an issue
func (s *Service) RemoveAllFromIssue(ctx context.Context, owner, repo string, number int) error {
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

	return s.store.RemoveAllFromIssue(ctx, issue.ID)
}

// ListForMilestone returns labels for issues in a milestone
func (s *Service) ListForMilestone(ctx context.Context, owner, repo string, number int, opts *ListOpts) ([]*Label, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Get milestone by number to verify it exists
	milestone, err := s.milestoneStore.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if milestone == nil {
		return nil, milestones.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 100}
	}

	// Get all issues for this milestone using milestone filter
	issueOpts := &issues.ListOpts{
		PerPage:   100,
		State:     "all",
		Milestone: fmt.Sprintf("%d", number),
	}
	issueList, err := s.issueStore.List(ctx, r.ID, issueOpts)
	if err != nil {
		return nil, err
	}

	// Aggregate labels from all issues
	labelMap := make(map[int64]*Label)
	for _, issue := range issueList {
		issueLabels, err := s.store.ListForIssue(ctx, issue.ID, opts)
		if err != nil {
			continue
		}
		for _, label := range issueLabels {
			labelMap[label.ID] = label
		}
	}

	// Convert to slice
	labels := make([]*Label, 0, len(labelMap))
	for _, label := range labelMap {
		s.populateURLs(label, owner, repo)
		labels = append(labels, label)
	}

	return labels, nil
}

// populateURLs fills in the URL fields for a label
func (s *Service) populateURLs(label *Label, owner, repo string) {
	label.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Label:%d", label.ID)))
	label.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/labels/%s", s.baseURL, owner, repo, label.Name)
}
