package preferences

import (
	"context"

	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

// Service implements the preferences API.
type Service struct {
	store *duckdb.PreferencesStore
}

// NewService creates a new preferences service.
func NewService(store *duckdb.PreferencesStore) *Service {
	return &Service{store: store}
}

// Get retrieves a preference value.
func (s *Service) Get(ctx context.Context, userID, key string) (any, error) {
	pref, err := s.store.Get(ctx, userID, key)
	if err != nil {
		return nil, err
	}
	if pref == nil {
		return nil, nil
	}
	return pref.Value, nil
}

// Set sets a preference value.
func (s *Service) Set(ctx context.Context, userID, key string, value any) error {
	_, err := s.store.Set(ctx, userID, key, value)
	return err
}

// Delete removes a preference.
func (s *Service) Delete(ctx context.Context, userID, key string) error {
	return s.store.Delete(ctx, userID, key)
}
