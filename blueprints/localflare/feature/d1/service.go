package d1

import (
	"context"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the D1 API.
type Service struct {
	store Store
}

// NewService creates a new D1 service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateDatabase creates a new database.
func (s *Service) CreateDatabase(ctx context.Context, in *CreateDatabaseIn) (*Database, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	db := &Database{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		Version:   "1",
		NumTables: 0,
		FileSize:  0,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateDatabase(ctx, db); err != nil {
		return nil, err
	}

	return db, nil
}

// GetDatabase retrieves a database by ID.
func (s *Service) GetDatabase(ctx context.Context, id string) (*Database, error) {
	db, err := s.store.GetDatabase(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return db, nil
}

// ListDatabases lists all databases.
func (s *Service) ListDatabases(ctx context.Context) ([]*Database, error) {
	return s.store.ListDatabases(ctx)
}

// DeleteDatabase deletes a database.
func (s *Service) DeleteDatabase(ctx context.Context, id string) error {
	return s.store.DeleteDatabase(ctx, id)
}

// Query executes a SQL query.
func (s *Service) Query(ctx context.Context, dbID string, in *QueryIn) (*QueryResult, error) {
	// Determine if this is a SELECT query
	sql := strings.TrimSpace(strings.ToUpper(in.SQL))
	isSelect := strings.HasPrefix(sql, "SELECT")

	if isSelect {
		results, err := s.store.Query(ctx, dbID, in.SQL, in.Params)
		if err != nil {
			return &QueryResult{Success: false}, err
		}
		return &QueryResult{
			Results: results,
			Success: true,
			Meta: &QueryMeta{
				ChangedDB: false,
				RowsRead:  int64(len(results)),
			},
		}, nil
	}

	// For INSERT, UPDATE, DELETE, etc.
	affected, err := s.store.Exec(ctx, dbID, in.SQL, in.Params)
	if err != nil {
		return &QueryResult{Success: false}, err
	}

	return &QueryResult{
		Results: nil,
		Success: true,
		Meta: &QueryMeta{
			ChangedDB:   true,
			Changes:     affected,
			RowsWritten: affected,
		},
	}, nil
}
