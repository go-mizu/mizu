// Package analytics_engine provides Analytics Engine management.
package analytics_engine

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("dataset not found")
	ErrNameRequired = errors.New("name is required")
)

// Dataset represents an Analytics Engine dataset.
type Dataset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// DataPoint represents an analytics data point.
type DataPoint struct {
	Dataset   string    `json:"dataset"`
	Timestamp time.Time `json:"timestamp"`
	Indexes   []string  `json:"indexes,omitempty"`
	Doubles   []float64 `json:"doubles,omitempty"`
	Blobs     [][]byte  `json:"blobs,omitempty"`
}

// CreateDatasetIn contains input for creating a dataset.
type CreateDatasetIn struct {
	Name string `json:"name"`
}

// WriteDataPointsIn contains input for writing data points.
type WriteDataPointsIn struct {
	DataPoints []*DataPoint `json:"datapoints"`
}

// QueryIn contains input for SQL queries.
type QueryIn struct {
	SQL string `json:"sql"`
}

// API defines the Analytics Engine service contract.
type API interface {
	CreateDataset(ctx context.Context, in *CreateDatasetIn) (*Dataset, error)
	GetDataset(ctx context.Context, name string) (*Dataset, error)
	ListDatasets(ctx context.Context) ([]*Dataset, error)
	DeleteDataset(ctx context.Context, name string) error
	WriteDataPoints(ctx context.Context, datasetName string, in *WriteDataPointsIn) error
	Query(ctx context.Context, in *QueryIn) ([]map[string]interface{}, error)
}

// Store defines the data access contract.
type Store interface {
	CreateDataset(ctx context.Context, dataset *Dataset) error
	GetDataset(ctx context.Context, name string) (*Dataset, error)
	ListDatasets(ctx context.Context) ([]*Dataset, error)
	DeleteDataset(ctx context.Context, name string) error
	WriteDataPoint(ctx context.Context, point *DataPoint) error
	WriteBatch(ctx context.Context, points []*DataPoint) error
	Query(ctx context.Context, sql string) ([]map[string]interface{}, error)
}
