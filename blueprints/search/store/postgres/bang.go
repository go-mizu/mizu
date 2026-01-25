package postgres

import (
	"context"
	"database/sql"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// BangStore handles bang shortcuts storage (stub implementation).
type BangStore struct {
	db *sql.DB
}

func (s *BangStore) CreateBang(ctx context.Context, bang *store.Bang) error {
	return nil
}

func (s *BangStore) GetBang(ctx context.Context, trigger string) (*store.Bang, error) {
	return nil, nil
}

func (s *BangStore) ListBangs(ctx context.Context) ([]*store.Bang, error) {
	return nil, nil
}

func (s *BangStore) ListUserBangs(ctx context.Context, userID string) ([]*store.Bang, error) {
	return nil, nil
}

func (s *BangStore) DeleteBang(ctx context.Context, id int64) error {
	return nil
}

func (s *BangStore) SeedBuiltinBangs(ctx context.Context) error {
	return nil
}
