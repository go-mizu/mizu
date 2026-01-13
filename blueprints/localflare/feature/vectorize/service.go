package vectorize

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Vectorize API.
type Service struct {
	store Store
}

// NewService creates a new Vectorize service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateIndex creates a new vector index.
func (s *Service) CreateIndex(ctx context.Context, in *CreateIndexIn) (*Index, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	metric := in.Metric
	if metric == "" {
		metric = "cosine"
	}

	index := &Index{
		ID:          ulid.Make().String(),
		Name:        in.Name,
		Description: in.Description,
		Dimensions:  in.Dimensions,
		Metric:      metric,
		CreatedAt:   time.Now(),
		VectorCount: 0,
	}

	if err := s.store.CreateIndex(ctx, index); err != nil {
		return nil, err
	}

	return index, nil
}

// GetIndex retrieves an index by name.
func (s *Service) GetIndex(ctx context.Context, name string) (*Index, error) {
	index, err := s.store.GetIndex(ctx, name)
	if err != nil {
		return nil, ErrNotFound
	}
	return index, nil
}

// ListIndexes lists all indexes.
func (s *Service) ListIndexes(ctx context.Context) ([]*Index, error) {
	return s.store.ListIndexes(ctx)
}

// DeleteIndex deletes an index.
func (s *Service) DeleteIndex(ctx context.Context, name string) error {
	return s.store.DeleteIndex(ctx, name)
}

// InsertVectors inserts vectors into an index.
func (s *Service) InsertVectors(ctx context.Context, indexName string, vectors []*Vector) error {
	return s.store.Insert(ctx, indexName, vectors)
}

// UpsertVectors upserts vectors into an index.
func (s *Service) UpsertVectors(ctx context.Context, indexName string, vectors []*Vector) error {
	return s.store.Upsert(ctx, indexName, vectors)
}

// Query queries vectors.
func (s *Service) Query(ctx context.Context, indexName string, in *QueryIn) ([]*Match, error) {
	topK := in.TopK
	if topK <= 0 {
		topK = 10
	}
	if topK > 100 {
		topK = 100
	}

	returnMetadata := in.ReturnMetadata
	if returnMetadata == "" {
		returnMetadata = "none"
	}

	opts := &QueryOpts{
		TopK:           topK,
		Namespace:      in.Namespace,
		ReturnValues:   in.ReturnValues,
		ReturnMetadata: returnMetadata,
		Filter:         in.Filter,
	}

	return s.store.Query(ctx, indexName, in.Vector, opts)
}

// GetByIDs retrieves vectors by IDs.
func (s *Service) GetByIDs(ctx context.Context, indexName string, ids []string) ([]*Vector, error) {
	return s.store.GetByIDs(ctx, indexName, ids)
}

// DeleteByIDs deletes vectors by IDs.
func (s *Service) DeleteByIDs(ctx context.Context, indexName string, ids []string) error {
	return s.store.DeleteByIDs(ctx, indexName, ids)
}
