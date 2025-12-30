package menus

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/slug"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

var (
	ErrMenuNotFound    = errors.New("menu not found")
	ErrItemNotFound    = errors.New("menu item not found")
	ErrMissingName     = errors.New("name is required")
	ErrMissingTitle    = errors.New("title is required")
)

// Service implements the menus API.
type Service struct {
	store Store
}

// NewService creates a new menus service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateMenu(ctx context.Context, in *CreateMenuIn) (*Menu, error) {
	if in.Name == "" {
		return nil, ErrMissingName
	}

	now := time.Now()
	menuSlug := in.Slug
	if menuSlug == "" {
		menuSlug = slug.Generate(in.Name)
	}

	menu := &Menu{
		ID:        ulid.New(),
		Name:      in.Name,
		Slug:      menuSlug,
		Location:  in.Location,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.CreateMenu(ctx, menu); err != nil {
		return nil, err
	}

	return menu, nil
}

func (s *Service) GetMenu(ctx context.Context, id string) (*Menu, error) {
	menu, err := s.store.GetMenu(ctx, id)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		return nil, ErrMenuNotFound
	}

	// Load items
	items, err := s.store.GetItemsByMenu(ctx, id)
	if err != nil {
		return nil, err
	}
	menu.Items = buildItemTree(items)

	return menu, nil
}

func (s *Service) GetMenuBySlug(ctx context.Context, menuSlug string) (*Menu, error) {
	menu, err := s.store.GetMenuBySlug(ctx, menuSlug)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		return nil, ErrMenuNotFound
	}

	// Load items
	items, err := s.store.GetItemsByMenu(ctx, menu.ID)
	if err != nil {
		return nil, err
	}
	menu.Items = buildItemTree(items)

	return menu, nil
}

func (s *Service) GetMenuByLocation(ctx context.Context, location string) (*Menu, error) {
	menu, err := s.store.GetMenuByLocation(ctx, location)
	if err != nil {
		return nil, err
	}
	if menu == nil {
		return nil, ErrMenuNotFound
	}

	// Load items
	items, err := s.store.GetItemsByMenu(ctx, menu.ID)
	if err != nil {
		return nil, err
	}
	menu.Items = buildItemTree(items)

	return menu, nil
}

func (s *Service) ListMenus(ctx context.Context) ([]*Menu, error) {
	return s.store.ListMenus(ctx)
}

func (s *Service) UpdateMenu(ctx context.Context, id string, in *UpdateMenuIn) (*Menu, error) {
	if err := s.store.UpdateMenu(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetMenu(ctx, id)
}

func (s *Service) DeleteMenu(ctx context.Context, id string) error {
	// Delete all items first
	if err := s.store.DeleteItemsByMenu(ctx, id); err != nil {
		return err
	}
	return s.store.DeleteMenu(ctx, id)
}

func (s *Service) CreateItem(ctx context.Context, menuID string, in *CreateItemIn) (*MenuItem, error) {
	if in.Title == "" {
		return nil, ErrMissingTitle
	}

	target := in.Target
	if target == "" {
		target = "_self"
	}

	item := &MenuItem{
		ID:        ulid.New(),
		MenuID:    menuID,
		ParentID:  in.ParentID,
		Title:     in.Title,
		URL:       in.URL,
		Target:    target,
		LinkType:  in.LinkType,
		LinkID:    in.LinkID,
		CSSClass:  in.CSSClass,
		SortOrder: in.SortOrder,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateItem(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *Service) UpdateItem(ctx context.Context, id string, in *UpdateItemIn) (*MenuItem, error) {
	if err := s.store.UpdateItem(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetItem(ctx, id)
}

func (s *Service) DeleteItem(ctx context.Context, id string) error {
	return s.store.DeleteItem(ctx, id)
}

func (s *Service) ReorderItems(ctx context.Context, menuID string, itemIDs []string) error {
	for i, itemID := range itemIDs {
		order := i
		if err := s.store.UpdateItem(ctx, itemID, &UpdateItemIn{SortOrder: &order}); err != nil {
			return err
		}
	}
	return nil
}

// buildItemTree builds a hierarchical tree from flat items.
func buildItemTree(items []*MenuItem) []*MenuItem {
	if len(items) == 0 {
		return nil
	}

	// Build a map of items by ID
	itemMap := make(map[string]*MenuItem)
	for _, item := range items {
		item.Children = nil // Reset children
		itemMap[item.ID] = item
	}

	// Build the tree
	var roots []*MenuItem
	for _, item := range items {
		if item.ParentID == "" {
			roots = append(roots, item)
		} else if parent, ok := itemMap[item.ParentID]; ok {
			parent.Children = append(parent.Children, item)
		}
	}

	return roots
}
