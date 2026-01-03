package templates

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("template not found")
)

// Service implements the templates API.
type Service struct {
	store Store
	pages pages.API
}

// NewService creates a new templates service.
func NewService(store Store, pages pages.API) *Service {
	return &Service{store: store, pages: pages}
}

// ListSystemTemplates lists system templates.
func (s *Service) ListSystemTemplates(ctx context.Context, category string) ([]*Template, error) {
	return s.store.ListSystem(ctx, category)
}

// GetSystemTemplate gets a system template.
func (s *Service) GetSystemTemplate(ctx context.Context, id string) (*Template, error) {
	template, err := s.store.GetByID(ctx, id)
	if err != nil || !template.IsSystem {
		return nil, ErrNotFound
	}
	return template, nil
}

// CreateTemplate creates a template from a page.
func (s *Service) CreateTemplate(ctx context.Context, pageID string, name, category, workspaceID, userID string) (*Template, error) {
	// Verify page exists
	page, err := s.pages.GetByID(ctx, pageID)
	if err != nil {
		return nil, err
	}

	template := &Template{
		ID:          ulid.New(),
		Name:        name,
		Category:    category,
		PageID:      page.ID,
		IsSystem:    false,
		WorkspaceID: workspaceID,
		CreatedAt:   time.Now(),
	}

	if template.Name == "" {
		template.Name = page.Title
	}

	if err := s.store.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// ListWorkspaceTemplates lists templates for a workspace.
func (s *Service) ListWorkspaceTemplates(ctx context.Context, workspaceID, category string) ([]*Template, error) {
	return s.store.ListByWorkspace(ctx, workspaceID, category)
}

// DeleteTemplate deletes a template.
func (s *Service) DeleteTemplate(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// UseTemplate creates a new page from a template.
func (s *Service) UseTemplate(ctx context.Context, templateID, targetParentID, userID string) (*pages.Page, error) {
	template, err := s.store.GetByID(ctx, templateID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Get the template page
	templatePage, err := s.pages.GetByID(ctx, template.PageID)
	if err != nil {
		return nil, err
	}

	// Duplicate the template page
	return s.pages.Duplicate(ctx, templatePage.ID, targetParentID, userID)
}
