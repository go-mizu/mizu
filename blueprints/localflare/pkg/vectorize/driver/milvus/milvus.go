// Package milvus provides a Milvus driver for the vectorize package.
// Import this package to register the "milvus" driver.
package milvus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func init() {
	driver.Register("milvus", &Driver{})
}

// Driver implements vectorize.Driver for Milvus.
type Driver struct{}

// Open creates a new Milvus connection.
// DSN format: host:port (e.g., "localhost:19530")
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := client.NewGrpcClient(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{client: c}, nil
}

// DB implements vectorize.DB for Milvus.
type DB struct {
	client client.Client
}

// CreateIndex creates a new collection in Milvus.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	// Check if collection exists
	has, err := db.client.HasCollection(ctx, index.Name)
	if err != nil {
		return err
	}
	if has {
		return vectorize.ErrIndexExists
	}

	metricType := entity.L2
	switch index.Metric {
	case vectorize.Cosine:
		metricType = entity.COSINE
	case vectorize.DotProduct:
		metricType = entity.IP
	}

	// Create schema
	schema := &entity.Schema{
		CollectionName: index.Name,
		Description:    index.Description,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeVarChar,
				PrimaryKey: true,
				AutoID:     false,
				TypeParams: map[string]string{"max_length": "256"},
			},
			{
				Name:       "namespace",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "256"},
			},
			{
				Name:       "metadata",
				DataType:   entity.FieldTypeVarChar,
				TypeParams: map[string]string{"max_length": "65535"},
			},
			{
				Name:     "embedding",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", index.Dimensions),
				},
			},
		},
	}

	if err := db.client.CreateCollection(ctx, schema, 1); err != nil {
		return err
	}

	// Create vector index
	idx, err := entity.NewIndexIvfFlat(metricType, 128)
	if err != nil {
		return err
	}

	return db.client.CreateIndex(ctx, index.Name, "embedding", idx, false)
}

// GetIndex retrieves collection information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	has, err := db.client.HasCollection(ctx, name)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, vectorize.ErrIndexNotFound
	}

	col, err := db.client.DescribeCollection(ctx, name)
	if err != nil {
		return nil, err
	}

	idx := &vectorize.Index{
		Name:        name,
		Description: col.Schema.Description,
		Metric:      vectorize.Euclidean, // Default
	}

	// Get dimensions from schema
	for _, field := range col.Schema.Fields {
		if field.Name == "embedding" {
			if dimStr, ok := field.TypeParams["dim"]; ok {
				fmt.Sscanf(dimStr, "%d", &idx.Dimensions)
			}
		}
	}

	// Get stats
	stats, err := db.client.GetCollectionStatistics(ctx, name)
	if err == nil {
		for k, v := range stats {
			if k == "row_count" {
				fmt.Sscanf(v, "%d", &idx.VectorCount)
			}
		}
	}

	return idx, nil
}

// ListIndexes returns all collections.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	collections, err := db.client.ListCollections(ctx)
	if err != nil {
		return nil, err
	}

	indexes := make([]*vectorize.Index, 0, len(collections))
	for _, col := range collections {
		idx, err := db.GetIndex(ctx, col.Name)
		if err != nil {
			continue
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// DeleteIndex removes a collection.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	has, err := db.client.HasCollection(ctx, name)
	if err != nil {
		return err
	}
	if !has {
		return vectorize.ErrIndexNotFound
	}

	return db.client.DropCollection(ctx, name)
}

// Insert adds vectors to a collection.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	ids := make([]string, len(vectors))
	namespaces := make([]string, len(vectors))
	metadatas := make([]string, len(vectors))
	embeddings := make([][]float32, len(vectors))

	for i, v := range vectors {
		ids[i] = v.ID
		namespaces[i] = v.Namespace
		if v.Metadata != nil {
			metaJSON, _ := json.Marshal(v.Metadata)
			metadatas[i] = string(metaJSON)
		}
		embeddings[i] = v.Values
	}

	idCol := entity.NewColumnVarChar("id", ids)
	nsCol := entity.NewColumnVarChar("namespace", namespaces)
	metaCol := entity.NewColumnVarChar("metadata", metadatas)
	embCol := entity.NewColumnFloatVector("embedding", len(vectors[0].Values), embeddings)

	_, err := db.client.Insert(ctx, indexName, "", idCol, nsCol, metaCol, embCol)
	return err
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	ids := make([]string, len(vectors))
	namespaces := make([]string, len(vectors))
	metadatas := make([]string, len(vectors))
	embeddings := make([][]float32, len(vectors))

	for i, v := range vectors {
		ids[i] = v.ID
		namespaces[i] = v.Namespace
		if v.Metadata != nil {
			metaJSON, _ := json.Marshal(v.Metadata)
			metadatas[i] = string(metaJSON)
		}
		embeddings[i] = v.Values
	}

	idCol := entity.NewColumnVarChar("id", ids)
	nsCol := entity.NewColumnVarChar("namespace", namespaces)
	metaCol := entity.NewColumnVarChar("metadata", metadatas)
	embCol := entity.NewColumnFloatVector("embedding", len(vectors[0].Values), embeddings)

	_, err := db.client.Upsert(ctx, indexName, "", idCol, nsCol, metaCol, embCol)
	return err
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	// Load collection for search
	if err := db.client.LoadCollection(ctx, indexName, false); err != nil {
		if !strings.Contains(err.Error(), "already loaded") {
			return nil, err
		}
	}

	// Build search parameters
	sp, err := entity.NewIndexIvfFlatSearchParam(16)
	if err != nil {
		return nil, err
	}

	// Prepare query vector
	vectors := []entity.Vector{entity.FloatVector(vector)}

	// Execute search
	outputFields := []string{"id", "namespace", "metadata"}
	results, err := db.client.Search(
		ctx,
		indexName,
		nil, // partitions
		"",  // expression
		outputFields,
		vectors,
		"embedding",
		entity.L2,
		opts.TopK,
		sp,
	)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []*vectorize.Match{}, nil
	}

	matches := make([]*vectorize.Match, 0, results[0].ResultCount)
	for i := 0; i < results[0].ResultCount; i++ {
		match := &vectorize.Match{
			Score: results[0].Scores[i],
		}

		// Extract fields
		for _, field := range results[0].Fields {
			switch field.Name() {
			case "id":
				if col, ok := field.(*entity.ColumnVarChar); ok {
					val, _ := col.ValueByIdx(i)
					match.ID = val
				}
			case "metadata":
				if opts.ReturnMetadata {
					if col, ok := field.(*entity.ColumnVarChar); ok {
						val, _ := col.ValueByIdx(i)
						if val != "" {
							json.Unmarshal([]byte(val), &match.Metadata)
						}
					}
				}
			}
		}

		// Apply score threshold
		if opts.ScoreThreshold > 0 && match.Score < opts.ScoreThreshold {
			continue
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Load collection
	if err := db.client.LoadCollection(ctx, indexName, false); err != nil {
		if !strings.Contains(err.Error(), "already loaded") {
			return nil, err
		}
	}

	// Build expression
	expr := fmt.Sprintf("id in [\"%s\"]", strings.Join(ids, "\",\""))

	results, err := db.client.Query(
		ctx,
		indexName,
		nil, // partitions
		expr,
		[]string{"id", "namespace", "metadata", "embedding"},
	)
	if err != nil {
		return nil, err
	}

	vectors := make([]*vectorize.Vector, 0)
	if len(results) == 0 {
		return vectors, nil
	}

	// Find number of results
	numResults := 0
	for _, col := range results {
		numResults = col.Len()
		break
	}

	for i := 0; i < numResults; i++ {
		vec := &vectorize.Vector{}

		for _, col := range results {
			switch col.Name() {
			case "id":
				if c, ok := col.(*entity.ColumnVarChar); ok {
					vec.ID, _ = c.ValueByIdx(i)
				}
			case "namespace":
				if c, ok := col.(*entity.ColumnVarChar); ok {
					vec.Namespace, _ = c.ValueByIdx(i)
				}
			case "metadata":
				if c, ok := col.(*entity.ColumnVarChar); ok {
					val, _ := c.ValueByIdx(i)
					if val != "" {
						json.Unmarshal([]byte(val), &vec.Metadata)
					}
				}
			case "embedding":
				if c, ok := col.(*entity.ColumnFloatVector); ok {
					vec.Values = c.Data()[i]
				}
			}
		}

		vectors = append(vectors, vec)
	}

	return vectors, nil
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	expr := fmt.Sprintf("id in [\"%s\"]", strings.Join(ids, "\",\""))
	return db.client.Delete(ctx, indexName, "", expr)
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	_, err := db.client.ListCollections(ctx)
	return err
}

// Close releases resources.
func (db *DB) Close() error {
	return db.client.Close()
}
