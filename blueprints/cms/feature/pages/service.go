package pages

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/slug"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrNotFound     = errors.New("page not found")
	ErrMissingTitle = errors.New("title is required")
)

// Service implements the pages API.
type Service struct {
	store Store
}

// NewService creates a new pages service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, authorID string, in *CreateIn) (*Page, error) {
	if in.Title == "" {
		return nil, ErrMissingTitle
	}

	now := time.Now()
	pageSlug := in.Slug
	if pageSlug == "" {
		pageSlug = slug.Generate(in.Title)
	}

	contentFormat := in.ContentFormat
	if contentFormat == "" {
		contentFormat = "markdown"
	}

	status := in.Status
	if status == "" {
		status = "draft"
	}

	visibility := in.Visibility
	if visibility == "" {
		visibility = "public"
	}

	page := &Page{
		ID:              ulid.New(),
		AuthorID:        authorID,
		ParentID:        in.ParentID,
		Title:           in.Title,
		Slug:            pageSlug,
		Content:         in.Content,
		ContentFormat:   contentFormat,
		FeaturedImageID: in.FeaturedImageID,
		Template:        in.Template,
		Status:          status,
		Visibility:      visibility,
		Meta:            in.Meta,
		SortOrder:       in.SortOrder,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.store.Create(ctx, page); err != nil {
		return nil, err
	}

	return page, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Page, error) {
	page, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, ErrNotFound
	}
	return page, nil
}

func (s *Service) GetBySlug(ctx context.Context, pageSlug string) (*Page, error) {
	page, err := s.store.GetBySlug(ctx, pageSlug)
	if err != nil {
		return nil, err
	}
	if page == nil {
		return nil, ErrNotFound
	}
	return page, nil
}

// GetByPath looks up a page by its full URL path (e.g., "parent/child/grandchild").
func (s *Service) GetByPath(ctx context.Context, path string) (*Page, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		return nil, ErrNotFound
	}

	var currentPage *Page
	parentID := ""

	for _, part := range parts {
		page, err := s.store.GetByParentAndSlug(ctx, parentID, part)
		if err != nil {
			return nil, err
		}
		if page == nil {
			return nil, ErrNotFound
		}
		currentPage = page
		parentID = page.ID
	}

	return currentPage, nil
}

func (s *Service) List(ctx context.Context, in *ListIn) ([]*Page, int, error) {
	if in.Limit <= 0 {
		in.Limit = 20
	}
	return s.store.List(ctx, in)
}

func (s *Service) GetTree(ctx context.Context) ([]*Page, error) {
	return s.store.GetTree(ctx)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Page, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
