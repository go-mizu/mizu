package history

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/blocks"
	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrRevisionNotFound = errors.New("revision not found")
)

// Service implements the history API.
type Service struct {
	store  Store
	users  users.API
	pages  pages.API
	blocks blocks.API
}

// NewService creates a new history service.
func NewService(store Store, users users.API, pages pages.API, blocks blocks.API) *Service {
	return &Service{
		store:  store,
		users:  users,
		pages:  pages,
		blocks: blocks,
	}
}

// CreateRevision creates a new revision of a page.
func (s *Service) CreateRevision(ctx context.Context, pageID, authorID string) (*Revision, error) {
	// Get current page
	page, err := s.pages.GetByID(ctx, pageID)
	if err != nil {
		return nil, err
	}

	// Get current blocks
	pageBlocks, err := s.blocks.GetByPage(ctx, pageID)
	if err != nil {
		return nil, err
	}

	// Get next version number
	version, _ := s.store.GetLatestVersion(ctx, pageID)
	version++

	revision := &Revision{
		ID:         ulid.New(),
		PageID:     pageID,
		Version:    version,
		Title:      page.Title,
		Blocks:     pageBlocks,
		Properties: page.Properties,
		AuthorID:   authorID,
		CreatedAt:  time.Now(),
	}

	if err := s.store.CreateRevision(ctx, revision); err != nil {
		return nil, err
	}

	return s.enrichRevision(ctx, revision)
}

// GetRevision retrieves a revision by ID.
func (s *Service) GetRevision(ctx context.Context, id string) (*Revision, error) {
	rev, err := s.store.GetRevision(ctx, id)
	if err != nil {
		return nil, ErrRevisionNotFound
	}
	return s.enrichRevision(ctx, rev)
}

// ListRevisions lists revisions for a page.
func (s *Service) ListRevisions(ctx context.Context, pageID string, limit int) ([]*Revision, error) {
	if limit <= 0 {
		limit = 20
	}

	revisions, err := s.store.ListRevisions(ctx, pageID, limit)
	if err != nil {
		return nil, err
	}

	return s.enrichRevisions(ctx, revisions)
}

// RestoreRevision restores a page to a specific revision.
func (s *Service) RestoreRevision(ctx context.Context, pageID, revisionID, userID string) error {
	rev, err := s.store.GetRevision(ctx, revisionID)
	if err != nil {
		return ErrRevisionNotFound
	}

	// Create a new revision first (to preserve current state)
	s.CreateRevision(ctx, pageID, userID)

	// Update page title
	s.pages.Update(ctx, pageID, &pages.UpdateIn{
		Title:     &rev.Title,
		UpdatedBy: userID,
	})

	// TODO: Restore blocks - this would require deleting current blocks and recreating from revision

	// Record activity
	s.RecordActivity(ctx, "", pageID, "", userID, ActionRestore, map[string]interface{}{
		"revision_id": revisionID,
		"version":     rev.Version,
	})

	return nil
}

// CompareRevisions compares two revisions.
func (s *Service) CompareRevisions(ctx context.Context, revID1, revID2 string) (*Diff, error) {
	rev1, err := s.store.GetRevision(ctx, revID1)
	if err != nil {
		return nil, ErrRevisionNotFound
	}

	rev2, err := s.store.GetRevision(ctx, revID2)
	if err != nil {
		return nil, ErrRevisionNotFound
	}

	diff := &Diff{}

	// Create maps for comparison
	blocks1 := make(map[string]*blocks.Block)
	for _, b := range rev1.Blocks {
		blocks1[b.ID] = b
	}

	blocks2 := make(map[string]*blocks.Block)
	for _, b := range rev2.Blocks {
		blocks2[b.ID] = b
	}

	// Find removed (in rev1 but not rev2)
	for id, b := range blocks1 {
		if _, exists := blocks2[id]; !exists {
			diff.Removed = append(diff.Removed, b)
		}
	}

	// Find added (in rev2 but not rev1)
	for id, b := range blocks2 {
		if _, exists := blocks1[id]; !exists {
			diff.Added = append(diff.Added, b)
		}
	}

	return diff, nil
}

// RecordActivity records an activity.
func (s *Service) RecordActivity(ctx context.Context, workspaceID, pageID, blockID, actorID string, action ActionType, details interface{}) error {
	activity := &Activity{
		ID:          ulid.New(),
		WorkspaceID: workspaceID,
		PageID:      pageID,
		BlockID:     blockID,
		ActorID:     actorID,
		Action:      action,
		Details:     details,
		CreatedAt:   time.Now(),
	}

	return s.store.CreateActivity(ctx, activity)
}

// ListByWorkspace lists activities for a workspace.
func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string, opts ActivityOpts) ([]*Activity, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	activities, err := s.store.ListByWorkspace(ctx, workspaceID, opts)
	if err != nil {
		return nil, err
	}

	return s.enrichActivities(ctx, activities)
}

// ListByPage lists activities for a page.
func (s *Service) ListByPage(ctx context.Context, pageID string, opts ActivityOpts) ([]*Activity, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	activities, err := s.store.ListByPage(ctx, pageID, opts)
	if err != nil {
		return nil, err
	}

	return s.enrichActivities(ctx, activities)
}

// ListByUser lists activities for a user.
func (s *Service) ListByUser(ctx context.Context, userID string, opts ActivityOpts) ([]*Activity, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	activities, err := s.store.ListByUser(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	return s.enrichActivities(ctx, activities)
}

// enrichRevision adds user data to a revision.
func (s *Service) enrichRevision(ctx context.Context, r *Revision) (*Revision, error) {
	if r.AuthorID != "" {
		author, _ := s.users.GetByID(ctx, r.AuthorID)
		r.Author = author
	}
	return r, nil
}

// enrichRevisions adds user data to multiple revisions.
func (s *Service) enrichRevisions(ctx context.Context, revisions []*Revision) ([]*Revision, error) {
	if len(revisions) == 0 {
		return revisions, nil
	}

	authorIDs := make([]string, 0, len(revisions))
	for _, r := range revisions {
		authorIDs = append(authorIDs, r.AuthorID)
	}

	usersMap, _ := s.users.GetByIDs(ctx, authorIDs)

	for _, r := range revisions {
		r.Author = usersMap[r.AuthorID]
	}

	return revisions, nil
}

// enrichActivities adds user and page data to activities.
func (s *Service) enrichActivities(ctx context.Context, activities []*Activity) ([]*Activity, error) {
	if len(activities) == 0 {
		return activities, nil
	}

	// Collect IDs
	actorIDs := make([]string, 0, len(activities))
	for _, a := range activities {
		actorIDs = append(actorIDs, a.ActorID)
	}

	// Batch fetch users
	usersMap, _ := s.users.GetByIDs(ctx, actorIDs)

	// Attach data
	for _, a := range activities {
		a.Actor = usersMap[a.ActorID]
		if a.PageID != "" {
			if page, err := s.pages.GetByID(ctx, a.PageID); err == nil {
				a.Page = &pages.PageRef{
					ID:    page.ID,
					Title: page.Title,
					Icon:  page.Icon,
				}
			}
		}
	}

	return activities, nil
}
