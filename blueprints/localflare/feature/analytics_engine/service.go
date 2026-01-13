package analytics_engine

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Analytics Engine API.
type Service struct {
	store Store
}

// NewService creates a new Analytics Engine service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateDataset creates a new dataset.
func (s *Service) CreateDataset(ctx context.Context, in *CreateDatasetIn) (*Dataset, error) {
	if in.Name == "" {
		return nil, ErrNameRequired
	}

	dataset := &Dataset{
		ID:        ulid.Make().String(),
		Name:      in.Name,
		CreatedAt: time.Now(),
	}

	if err := s.store.CreateDataset(ctx, dataset); err != nil {
		return nil, err
	}

	return dataset, nil
}

// GetDataset retrieves a dataset by name.
func (s *Service) GetDataset(ctx context.Context, name string) (*Dataset, error) {
	dataset, err := s.store.GetDataset(ctx, name)
	if err != nil {
		return nil, ErrNotFound
	}
	return dataset, nil
}

// ListDatasets lists all datasets.
func (s *Service) ListDatasets(ctx context.Context) ([]*Dataset, error) {
	return s.store.ListDatasets(ctx)
}

// DeleteDataset deletes a dataset.
func (s *Service) DeleteDataset(ctx context.Context, name string) error {
	return s.store.DeleteDataset(ctx, name)
}

// WriteDataPoints writes data points to a dataset.
func (s *Service) WriteDataPoints(ctx context.Context, datasetName string, in *WriteDataPointsIn) error {
	if len(in.DataPoints) == 0 {
		return nil
	}

	// Set dataset name and timestamp for each point
	now := time.Now()
	for _, dp := range in.DataPoints {
		dp.Dataset = datasetName
		if dp.Timestamp.IsZero() {
			dp.Timestamp = now
		}
	}

	return s.store.WriteBatch(ctx, in.DataPoints)
}

// Query executes a SQL query.
func (s *Service) Query(ctx context.Context, in *QueryIn) ([]map[string]interface{}, error) {
	return s.store.Query(ctx, in.SQL)
}
