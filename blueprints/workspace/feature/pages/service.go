package pages

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound       = errors.New("page not found")
	ErrInvalidParent  = errors.New("invalid parent")
)

// Service implements the pages API.
type Service struct {
	store Store
	users users.API
}

// NewService creates a new pages service.
func NewService(store Store, users users.API) *Service {
	return &Service{store: store, users: users}
}

// Create creates a new page.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Page, error) {
	now := time.Now()
	page := &Page{
		ID:          ulid.New(),
		WorkspaceID: in.WorkspaceID,
		ParentID:    in.ParentID,
		ParentType:  in.ParentType,
		Title:       in.Title,
		Icon:        in.Icon,
		Cover:       in.Cover,
		CoverY:      0.5,
		Properties:  in.Properties,
		IsTemplate:  in.IsTemplate,
		IsArchived:  false,
		CreatedBy:   in.CreatedBy,
		CreatedAt:   now,
		UpdatedBy:   in.CreatedBy,
		UpdatedAt:   now,
	}

	if page.ParentType == "" {
		page.ParentType = ParentWorkspace
	}

	if err := s.store.Create(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

// GetByID retrieves a page by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Page, error) {
	page, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Enrich with author
	if page.CreatedBy != "" {
		author, _ := s.users.GetByID(ctx, page.CreatedBy)
		page.Author = author
	}

	return page, nil
}

// Update updates a page.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Page, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a page.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByWorkspace lists root pages in a workspace.
func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string, opts ListOpts) ([]*Page, error) {
	pages, err := s.store.ListByWorkspace(ctx, workspaceID, opts)
	if err != nil {
		return nil, err
	}

	// Batch load children to avoid N+1 queries
	if len(pages) > 0 {
		parentIDs := make([]string, len(pages))
		for i, p := range pages {
			parentIDs[i] = p.ID
		}

		childrenMap, err := s.store.ListByParentIDs(ctx, parentIDs, ParentPage)
		if err == nil {
			for _, p := range pages {
				p.Children = childrenMap[p.ID]
			}
		}
	}

	return pages, nil
}

// ListByParent lists pages under a parent.
func (s *Service) ListByParent(ctx context.Context, parentID string, parentType ParentType) ([]*Page, error) {
	return s.store.ListByParent(ctx, parentID, parentType)
}

// GetBreadcrumb returns the breadcrumb path to a page.
func (s *Service) GetBreadcrumb(ctx context.Context, id string) ([]*PageRef, error) {
	var breadcrumb []*PageRef

	page, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	// Walk up the tree
	for page.ParentType == ParentPage && page.ParentID != "" {
		parent, err := s.store.GetByID(ctx, page.ParentID)
		if err != nil {
			break
		}
		breadcrumb = append([]*PageRef{{
			ID:    parent.ID,
			Title: parent.Title,
			Icon:  parent.Icon,
		}}, breadcrumb...)
		page = parent
	}

	return breadcrumb, nil
}

// Move moves a page to a new parent.
func (s *Service) Move(ctx context.Context, id, newParentID string, newParentType ParentType) error {
	return s.store.Move(ctx, id, newParentID, newParentType)
}

// Duplicate creates a copy of a page.
func (s *Service) Duplicate(ctx context.Context, id string, targetParentID string, userID string) (*Page, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	parentID := targetParentID
	if parentID == "" {
		parentID = original.ParentID
	}

	return s.Create(ctx, &CreateIn{
		WorkspaceID: original.WorkspaceID,
		ParentID:    parentID,
		ParentType:  original.ParentType,
		Title:       original.Title + " (copy)",
		Icon:        original.Icon,
		Cover:       original.Cover,
		Properties:  original.Properties,
		CreatedBy:   userID,
	})
}

// Archive archives a page.
func (s *Service) Archive(ctx context.Context, id string) error {
	return s.store.Archive(ctx, id)
}

// Restore restores an archived page.
func (s *Service) Restore(ctx context.Context, id string) error {
	return s.store.Restore(ctx, id)
}

// ListArchived lists archived pages in a workspace.
func (s *Service) ListArchived(ctx context.Context, workspaceID string) ([]*Page, error) {
	return s.store.ListArchived(ctx, workspaceID)
}

// Search searches for pages in a workspace.
func (s *Service) Search(ctx context.Context, workspaceID, query string, opts SearchOpts) ([]*Page, error) {
	return s.store.Search(ctx, workspaceID, query, opts)
}

// GetRecent returns recently accessed pages.
func (s *Service) GetRecent(ctx context.Context, userID, workspaceID string, limit int) ([]*Page, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.GetRecent(ctx, userID, workspaceID, limit)
}
