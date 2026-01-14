// Package pgvector provides a PostgreSQL/pgvector driver for the vectorize package.
// Import this package to register the "pgvector" driver.
package pgvector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

func init() {
	driver.Register("pgvector", &Driver{})
}

// Driver implements vectorize.Driver for pgvector.
type Driver struct{}

// Open creates a new pgvector connection.
// DSN format: postgres://user:pass@host:port/database
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrInvalidDSN, err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{pool: pool}, nil
}

// DB implements vectorize.DB for pgvector.
type DB struct {
	pool *pgxpool.Pool
}

// CreateIndex creates a new vector index.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	// Check if index exists
	var exists bool
	err := db.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM vector_indexes WHERE name = $1)`, index.Name).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return vectorize.ErrIndexExists
	}

	// Insert index metadata
	_, err = db.pool.Exec(ctx, `
		INSERT INTO vector_indexes (name, dimensions, metric, description, vector_count, created_at)
		VALUES ($1, $2, $3, $4, 0, $5)
	`, index.Name, index.Dimensions, string(index.Metric), index.Description, time.Now())
	if err != nil {
		return err
	}

	return nil
}

// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	var idx vectorize.Index
	var metric string
	var desc *string
	err := db.pool.QueryRow(ctx, `
		SELECT name, dimensions, metric, description, vector_count, created_at
		FROM vector_indexes WHERE name = $1
	`, name).Scan(&idx.Name, &idx.Dimensions, &metric, &desc, &idx.VectorCount, &idx.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, vectorize.ErrIndexNotFound
		}
		return nil, err
	}
	idx.Metric = vectorize.DistanceMetric(metric)
	if desc != nil {
		idx.Description = *desc
	}
	return &idx, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT name, dimensions, metric, description, vector_count, created_at
		FROM vector_indexes ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*vectorize.Index
	for rows.Next() {
		var idx vectorize.Index
		var metric string
		var desc *string
		if err := rows.Scan(&idx.Name, &idx.Dimensions, &metric, &desc, &idx.VectorCount, &idx.CreatedAt); err != nil {
			return nil, err
		}
		idx.Metric = vectorize.DistanceMetric(metric)
		if desc != nil {
			idx.Description = *desc
		}
		indexes = append(indexes, &idx)
	}
	return indexes, rows.Err()
}

// DeleteIndex removes an index and all its vectors.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	result, err := db.pool.Exec(ctx, `DELETE FROM vector_indexes WHERE name = $1`, name)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return vectorize.ErrIndexNotFound
	}
	return nil
}

// Insert adds vectors to an index.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	// Get index dimensions
	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		metadataJSON, _ := json.Marshal(v.Metadata)

		_, err := tx.Exec(ctx, `
			INSERT INTO vectors (id, index_name, namespace, embedding, metadata)
			VALUES ($1, $2, $3, $4, $5)
		`, v.ID, indexName, v.Namespace, pgvector.NewVector(v.Values), metadataJSON)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				return vectorize.ErrVectorExists
			}
			return err
		}
	}

	// Update vector count
	_, err = tx.Exec(ctx, `
		UPDATE vector_indexes SET vector_count = (
			SELECT COUNT(*) FROM vectors WHERE index_name = $1
		) WHERE name = $1
	`, indexName)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	// Get index dimensions
	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		metadataJSON, _ := json.Marshal(v.Metadata)

		_, err := tx.Exec(ctx, `
			INSERT INTO vectors (id, index_name, namespace, embedding, metadata)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (index_name, id) DO UPDATE SET
				namespace = EXCLUDED.namespace,
				embedding = EXCLUDED.embedding,
				metadata = EXCLUDED.metadata
		`, v.ID, indexName, v.Namespace, pgvector.NewVector(v.Values), metadataJSON)
		if err != nil {
			return err
		}
	}

	// Update vector count
	_, err = tx.Exec(ctx, `
		UPDATE vector_indexes SET vector_count = (
			SELECT COUNT(*) FROM vectors WHERE index_name = $1
		) WHERE name = $1
	`, indexName)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	// Get index metric
	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return nil, err
	}

	// Build query based on metric
	var distanceExpr string
	switch idx.Metric {
	case vectorize.Cosine:
		distanceExpr = "1 - (embedding <=> $1)" // Cosine similarity
	case vectorize.Euclidean:
		distanceExpr = "1 / (1 + (embedding <-> $1))" // Convert distance to similarity
	case vectorize.DotProduct:
		distanceExpr = "(embedding <#> $1) * -1" // Inner product (negated for proper ordering)
	default:
		distanceExpr = "1 - (embedding <=> $1)"
	}

	query := fmt.Sprintf(`
		SELECT id, %s as score, embedding, metadata
		FROM vectors
		WHERE index_name = $2
	`, distanceExpr)
	args := []any{pgvector.NewVector(vector), indexName}
	argIdx := 3

	// Add namespace filter
	if opts.Namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIdx)
		args = append(args, opts.Namespace)
		argIdx++
	}

	// Add metadata filters
	for key, val := range opts.Filter {
		query += fmt.Sprintf(" AND metadata->>'%s' = $%d", key, argIdx)
		args = append(args, fmt.Sprintf("%v", val))
		argIdx++
	}

	// Order by similarity and limit
	query += fmt.Sprintf(" ORDER BY score DESC LIMIT $%d", argIdx)
	args = append(args, opts.TopK)

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*vectorize.Match
	for rows.Next() {
		var id string
		var score float32
		var embedding pgvector.Vector
		var metadataJSON []byte

		if err := rows.Scan(&id, &score, &embedding, &metadataJSON); err != nil {
			return nil, err
		}

		// Apply score threshold
		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    id,
			Score: score,
		}

		if opts.ReturnValues {
			match.Values = embedding.Slice()
		}

		if opts.ReturnMetadata && len(metadataJSON) > 0 {
			var metadata map[string]any
			json.Unmarshal(metadataJSON, &metadata)
			match.Metadata = metadata
		}

		matches = append(matches, match)
	}

	return matches, rows.Err()
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := db.pool.Query(ctx, `
		SELECT id, namespace, embedding, metadata
		FROM vectors
		WHERE index_name = $1 AND id = ANY($2)
	`, indexName, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vectors []*vectorize.Vector
	for rows.Next() {
		var v vectorize.Vector
		var namespace *string
		var embedding pgvector.Vector
		var metadataJSON []byte

		if err := rows.Scan(&v.ID, &namespace, &embedding, &metadataJSON); err != nil {
			return nil, err
		}

		if namespace != nil {
			v.Namespace = *namespace
		}
		v.Values = embedding.Slice()

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &v.Metadata)
		}

		vectors = append(vectors, &v)
	}

	return vectors, rows.Err()
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		DELETE FROM vectors WHERE index_name = $1 AND id = ANY($2)
	`, indexName, ids)
	if err != nil {
		return err
	}

	// Update vector count
	_, err = tx.Exec(ctx, `
		UPDATE vector_indexes SET vector_count = (
			SELECT COUNT(*) FROM vectors WHERE index_name = $1
		) WHERE name = $1
	`, indexName)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close releases resources.
func (db *DB) Close() error {
	db.pool.Close()
	return nil
}
