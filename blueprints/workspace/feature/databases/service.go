package databases

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrNotFound = errors.New("database not found")
)

// Service implements the databases API.
type Service struct {
	store Store
}

// NewService creates a new databases service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new database.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Database, error) {
	now := time.Now()

	// Ensure we have at least a title property
	props := in.Properties
	if len(props) == 0 {
		props = []Property{
			{ID: ulid.New(), Name: "Name", Type: PropTitle},
		}
	} else {
		// Add IDs to properties that don't have them
		for i := range props {
			if props[i].ID == "" {
				props[i].ID = ulid.New()
			}
		}
	}

	db := &Database{
		ID:          ulid.New(),
		WorkspaceID: in.WorkspaceID,
		PageID:      in.PageID,
		Title:       in.Title,
		Icon:        in.Icon,
		IsInline:    in.IsInline,
		Properties:  props,
		CreatedBy:   in.CreatedBy,
		CreatedAt:   now,
		UpdatedBy:   in.CreatedBy,
		UpdatedAt:   now,
	}

	if db.Title == "" {
		db.Title = "Untitled"
	}

	if err := s.store.Create(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

// GetByID retrieves a database by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Database, error) {
	db, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return db, nil
}

// Update updates a database.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Database, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a database.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByWorkspace lists databases in a workspace.
func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Database, error) {
	return s.store.ListByWorkspace(ctx, workspaceID)
}

// ListByPage lists databases in a page.
func (s *Service) ListByPage(ctx context.Context, pageID string) ([]*Database, error) {
	return s.store.ListByPage(ctx, pageID)
}

// AddProperty adds a property to a database.
func (s *Service) AddProperty(ctx context.Context, dbID string, prop Property) (*Database, error) {
	db, err := s.store.GetByID(ctx, dbID)
	if err != nil {
		return nil, ErrNotFound
	}

	if prop.ID == "" {
		prop.ID = ulid.New()
	}

	props := append(db.Properties, prop)
	if err := s.store.UpdateProperties(ctx, dbID, props); err != nil {
		return nil, err
	}

	return s.store.GetByID(ctx, dbID)
}

// UpdateProperty updates a property in a database.
func (s *Service) UpdateProperty(ctx context.Context, dbID, propID string, prop Property) error {
	db, err := s.store.GetByID(ctx, dbID)
	if err != nil {
		return ErrNotFound
	}

	props := make([]Property, 0, len(db.Properties))
	for _, p := range db.Properties {
		if p.ID == propID {
			prop.ID = propID
			props = append(props, prop)
		} else {
			props = append(props, p)
		}
	}

	return s.store.UpdateProperties(ctx, dbID, props)
}

// DeleteProperty deletes a property from a database.
func (s *Service) DeleteProperty(ctx context.Context, dbID, propID string) error {
	db, err := s.store.GetByID(ctx, dbID)
	if err != nil {
		return ErrNotFound
	}

	props := make([]Property, 0, len(db.Properties)-1)
	for _, p := range db.Properties {
		if p.ID != propID {
			props = append(props, p)
		}
	}

	return s.store.UpdateProperties(ctx, dbID, props)
}

// ReorderProperties reorders properties in a database.
func (s *Service) ReorderProperties(ctx context.Context, dbID string, propIDs []string) error {
	db, err := s.store.GetByID(ctx, dbID)
	if err != nil {
		return ErrNotFound
	}

	// Create a map for quick lookup
	propMap := make(map[string]Property)
	for _, p := range db.Properties {
		propMap[p.ID] = p
	}

	// Reorder
	props := make([]Property, 0, len(propIDs))
	for _, id := range propIDs {
		if p, ok := propMap[id]; ok {
			props = append(props, p)
			delete(propMap, id)
		}
	}

	// Add any remaining properties
	for _, p := range propMap {
		props = append(props, p)
	}

	return s.store.UpdateProperties(ctx, dbID, props)
}

// Duplicate creates a copy of a database.
func (s *Service) Duplicate(ctx context.Context, id string, targetPageID string, userID string) (*Database, error) {
	original, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}

	pageID := targetPageID
	if pageID == "" {
		pageID = original.PageID
	}

	// Create new properties with new IDs
	newProps := make([]Property, len(original.Properties))
	for i, p := range original.Properties {
		newProps[i] = Property{
			ID:     ulid.New(),
			Name:   p.Name,
			Type:   p.Type,
			Config: p.Config,
		}
	}

	return s.Create(ctx, &CreateIn{
		WorkspaceID: original.WorkspaceID,
		PageID:      pageID,
		Title:       original.Title + " (copy)",
		Icon:        original.Icon,
		IsInline:    original.IsInline,
		Properties:  newProps,
		CreatedBy:   userID,
	})
}
