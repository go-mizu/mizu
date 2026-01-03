package favorites

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/pages"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrAlreadyFavorite = errors.New("page is already a favorite")
)

// Service implements the favorites API.
type Service struct {
	store Store
	pages pages.API
}

// NewService creates a new favorites service.
func NewService(store Store, pages pages.API) *Service {
	return &Service{store: store, pages: pages}
}

// Add adds a page to favorites.
func (s *Service) Add(ctx context.Context, userID, pageID, workspaceID string) (*Favorite, error) {
	// Check if already favorite
	exists, _ := s.store.Exists(ctx, userID, pageID)
	if exists {
		return nil, ErrAlreadyFavorite
	}

	favorite := &Favorite{
		ID:          ulid.New(),
		UserID:      userID,
		PageID:      pageID,
		WorkspaceID: workspaceID,
		CreatedAt:   time.Now(),
	}

	if err := s.store.Create(ctx, favorite); err != nil {
		return nil, err
	}

	return s.enrichFavorite(ctx, favorite)
}

// Remove removes a page from favorites.
func (s *Service) Remove(ctx context.Context, userID, pageID string) error {
	return s.store.Delete(ctx, userID, pageID)
}

// List lists a user's favorites in a workspace.
func (s *Service) List(ctx context.Context, userID, workspaceID string) ([]*Favorite, error) {
	favorites, err := s.store.List(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}

	return s.enrichFavorites(ctx, favorites)
}

// IsFavorite checks if a page is a favorite.
func (s *Service) IsFavorite(ctx context.Context, userID, pageID string) (bool, error) {
	return s.store.Exists(ctx, userID, pageID)
}

// enrichFavorite adds page data.
func (s *Service) enrichFavorite(ctx context.Context, f *Favorite) (*Favorite, error) {
	page, _ := s.pages.GetByID(ctx, f.PageID)
	f.Page = page
	return f, nil
}

// enrichFavorites adds page data to multiple favorites.
func (s *Service) enrichFavorites(ctx context.Context, favorites []*Favorite) ([]*Favorite, error) {
	for _, f := range favorites {
		page, _ := s.pages.GetByID(ctx, f.PageID)
		f.Page = page
	}
	return favorites, nil
}
