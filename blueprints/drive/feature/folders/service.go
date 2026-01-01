package folders

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/ulid"
)

// Service implements the folders API.
type Service struct {
	store Store
}

// NewService creates a new folders service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new folder.
func (s *Service) Create(ctx context.Context, ownerID string, in *CreateIn) (*Folder, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrInvalidParent
	}

	// Get parent folder
	var parentPath string
	var depth int
	parentID := in.ParentID

	if parentID != "" {
		parent, err := s.store.GetByID(ctx, parentID)
		if err != nil {
			return nil, ErrInvalidParent
		}
		if parent.OwnerID != ownerID {
			return nil, ErrNotOwner
		}
		parentPath = parent.Path
		depth = parent.Depth + 1
	} else {
		// Use root folder
		root, err := s.EnsureRoot(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		parentID = root.ID
		parentPath = root.Path
		depth = 1
	}

	// Check for duplicate name
	if existing, _ := s.store.GetByOwnerAndParentAndName(ctx, ownerID, parentID, name); existing != nil {
		return nil, ErrNameTaken
	}

	now := time.Now()
	folder := &Folder{
		ID:        ulid.New(),
		OwnerID:   ownerID,
		ParentID:  parentID,
		Name:      name,
		Path:      path.Join(parentPath, name),
		Depth:     depth,
		Color:     in.Color,
		IsRoot:    false,
		Starred:   false,
		Trashed:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

// GetByID retrieves a folder by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Folder, error) {
	return s.store.GetByID(ctx, id)
}

// GetRoot retrieves the root folder for a user.
func (s *Service) GetRoot(ctx context.Context, ownerID string) (*Folder, error) {
	return s.store.GetRoot(ctx, ownerID)
}

// EnsureRoot creates root folder if it doesn't exist.
func (s *Service) EnsureRoot(ctx context.Context, ownerID string) (*Folder, error) {
	root, err := s.store.GetRoot(ctx, ownerID)
	if err == nil {
		return root, nil
	}

	now := time.Now()
	root = &Folder{
		ID:        ulid.New(),
		OwnerID:   ownerID,
		ParentID:  "",
		Name:      "My Drive",
		Path:      "/",
		Depth:     0,
		IsRoot:    true,
		Starred:   false,
		Trashed:   false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, root); err != nil {
		// Race condition - another goroutine created it
		return s.store.GetRoot(ctx, ownerID)
	}

	return root, nil
}

// List lists folders.
func (s *Service) List(ctx context.Context, ownerID string, in *ListIn) ([]*Folder, error) {
	return s.store.List(ctx, ownerID, in)
}

// Update updates a folder.
func (s *Service) Update(ctx context.Context, id, ownerID string, in *UpdateIn) (*Folder, error) {
	folder, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if folder.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	if folder.IsRoot {
		return nil, ErrCannotMove
	}

	// Check for name collision
	if in.Name != nil && *in.Name != folder.Name {
		name := strings.TrimSpace(*in.Name)
		if existing, _ := s.store.GetByOwnerAndParentAndName(ctx, ownerID, folder.ParentID, name); existing != nil {
			return nil, ErrNameTaken
		}

		// Update path for this folder and descendants
		oldPath := folder.Path
		newPath := path.Join(path.Dir(folder.Path), name)

		if err := s.store.Update(ctx, id, in); err != nil {
			return nil, err
		}
		if err := s.store.UpdatePath(ctx, id, newPath, folder.Depth); err != nil {
			return nil, err
		}

		// Update descendant paths
		descendants, _ := s.store.ListDescendants(ctx, id)
		for _, desc := range descendants {
			descNewPath := strings.Replace(desc.Path, oldPath, newPath, 1)
			s.store.UpdatePath(ctx, desc.ID, descNewPath, desc.Depth)
		}
	} else {
		if err := s.store.Update(ctx, id, in); err != nil {
			return nil, err
		}
	}

	return s.store.GetByID(ctx, id)
}

// Move moves a folder to a new parent.
func (s *Service) Move(ctx context.Context, id, ownerID, newParentID string) (*Folder, error) {
	folder, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if folder.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	if folder.IsRoot {
		return nil, ErrCannotMove
	}

	if newParentID == id {
		return nil, ErrCannotMove
	}

	// Get new parent
	var newParentPath string
	var newDepth int

	if newParentID == "" {
		root, err := s.EnsureRoot(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		newParentID = root.ID
		newParentPath = root.Path
		newDepth = 1
	} else {
		parent, err := s.store.GetByID(ctx, newParentID)
		if err != nil {
			return nil, ErrInvalidParent
		}
		if parent.OwnerID != ownerID {
			return nil, ErrNotOwner
		}

		// Can't move into descendant
		if strings.HasPrefix(parent.Path, folder.Path+"/") {
			return nil, ErrCannotMove
		}

		newParentPath = parent.Path
		newDepth = parent.Depth + 1
	}

	// Check name collision in new parent
	if existing, _ := s.store.GetByOwnerAndParentAndName(ctx, ownerID, newParentID, folder.Name); existing != nil {
		return nil, ErrNameTaken
	}

	oldPath := folder.Path
	newPath := path.Join(newParentPath, folder.Name)
	depthDiff := newDepth - folder.Depth

	// Update this folder
	if err := s.store.UpdateParent(ctx, id, newParentID, newPath, newDepth); err != nil {
		return nil, err
	}

	// Update descendant paths
	descendants, _ := s.store.ListDescendants(ctx, id)
	for _, desc := range descendants {
		descNewPath := strings.Replace(desc.Path, oldPath, newPath, 1)
		s.store.UpdatePath(ctx, desc.ID, descNewPath, desc.Depth+depthDiff)
	}

	return s.store.GetByID(ctx, id)
}

// Copy copies a folder and its contents.
func (s *Service) Copy(ctx context.Context, id, ownerID, destParentID string) (*Folder, error) {
	folder, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if folder.OwnerID != ownerID {
		return nil, ErrNotOwner
	}

	// Create copy in destination
	return s.Create(ctx, ownerID, &CreateIn{
		Name:     folder.Name + " (copy)",
		ParentID: destParentID,
		Color:    folder.Color,
	})
}

// Delete moves a folder to trash.
func (s *Service) Delete(ctx context.Context, id, ownerID string) error {
	folder, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if folder.OwnerID != ownerID {
		return ErrNotOwner
	}

	if folder.IsRoot {
		return ErrCannotMove
	}

	return s.store.UpdateTrashed(ctx, id, true)
}

// Star stars or unstars a folder.
func (s *Service) Star(ctx context.Context, id, ownerID string, starred bool) error {
	folder, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if folder.OwnerID != ownerID {
		return ErrNotOwner
	}

	return s.store.UpdateStarred(ctx, id, starred)
}

// GetTree returns the folder tree.
func (s *Service) GetTree(ctx context.Context, ownerID string, rootID string) (*TreeNode, error) {
	var root *Folder
	var err error

	if rootID == "" {
		root, err = s.store.GetRoot(ctx, ownerID)
	} else {
		root, err = s.store.GetByID(ctx, rootID)
	}
	if err != nil {
		return nil, err
	}

	return s.buildTree(ctx, ownerID, root)
}

func (s *Service) buildTree(ctx context.Context, ownerID string, folder *Folder) (*TreeNode, error) {
	node := &TreeNode{
		ID:   folder.ID,
		Name: folder.Name,
	}

	children, err := s.store.ListByParent(ctx, ownerID, folder.ID)
	if err != nil {
		return node, nil
	}

	for _, child := range children {
		if child.Trashed {
			continue
		}
		childNode, _ := s.buildTree(ctx, ownerID, child)
		node.Children = append(node.Children, childNode)
	}

	return node, nil
}
