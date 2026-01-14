// Package duckdb provides a DuckDB embedded driver for the vectorize package.
// Import this package to register the "duckdb" driver.
// Note: DuckDB VSS extension provides vector similarity search.
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	_ "github.com/marcboeker/go-duckdb"
)

func init() {
	driver.Register("duckdb", &Driver{})
}

// Driver implements vectorize.Driver for DuckDB.
type Driver struct{}

// Open creates a new DuckDB connection.
// DSN format: /path/to/db.duckdb or :memory: for in-memory
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func initSchema(db *sql.DB) error {
	// DuckDB doesn't support ON DELETE CASCADE with foreign keys
	// We'll handle cascade deletes manually in DeleteIndex
	schema := `
		CREATE TABLE IF NOT EXISTS vector_indexes (
			name VARCHAR PRIMARY KEY,
			dimensions INTEGER NOT NULL,
			metric VARCHAR NOT NULL DEFAULT 'cosine',
			description VARCHAR,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS vectors (
			id VARCHAR NOT NULL,
			index_name VARCHAR NOT NULL,
			namespace VARCHAR DEFAULT '',
			values_json VARCHAR NOT NULL,
			metadata_json VARCHAR DEFAULT '{}',
			PRIMARY KEY (index_name, id)
		);

		CREATE INDEX IF NOT EXISTS idx_vectors_namespace ON vectors(index_name, namespace);
	`
	_, err := db.Exec(schema)
	return err
}

// DB implements vectorize.DB for DuckDB.
type DB struct {
	db *sql.DB
}

// CreateIndex creates a new index.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	// Check if exists
	var exists bool
	err := db.db.QueryRowContext(ctx, `SELECT COUNT(*) > 0 FROM vector_indexes WHERE name = ?`, index.Name).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return vectorize.ErrIndexExists
	}

	_, err = db.db.ExecContext(ctx, `
		INSERT INTO vector_indexes (name, dimensions, metric, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, index.Name, index.Dimensions, string(index.Metric), index.Description, time.Now())
	return err
}

// GetIndex retrieves index information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	var idx vectorize.Index
	var metric string
	var desc sql.NullString
	err := db.db.QueryRowContext(ctx, `
		SELECT name, dimensions, metric, description, created_at
		FROM vector_indexes WHERE name = ?
	`, name).Scan(&idx.Name, &idx.Dimensions, &metric, &desc, &idx.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, vectorize.ErrIndexNotFound
		}
		return nil, err
	}
	idx.Metric = vectorize.DistanceMetric(metric)
	if desc.Valid {
		idx.Description = desc.String
	}

	// Get count
	db.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vectors WHERE index_name = ?`, name).Scan(&idx.VectorCount)

	return &idx, nil
}

// ListIndexes returns all indexes.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	rows, err := db.db.QueryContext(ctx, `
		SELECT name, dimensions, metric, description, created_at
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
		var desc sql.NullString
		if err := rows.Scan(&idx.Name, &idx.Dimensions, &metric, &desc, &idx.CreatedAt); err != nil {
			return nil, err
		}
		idx.Metric = vectorize.DistanceMetric(metric)
		if desc.Valid {
			idx.Description = desc.String
		}
		indexes = append(indexes, &idx)
	}
	return indexes, rows.Err()
}

// DeleteIndex removes an index and all its vectors.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	// First delete all vectors (manual cascade since DuckDB doesn't support ON DELETE CASCADE)
	_, err := db.db.ExecContext(ctx, `DELETE FROM vectors WHERE index_name = ?`, name)
	if err != nil {
		return err
	}

	result, err := db.db.ExecContext(ctx, `DELETE FROM vector_indexes WHERE name = ?`, name)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
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

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO vectors (id, index_name, namespace, values_json, metadata_json)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}
		valuesJSON, _ := json.Marshal(v.Values)
		metadataJSON, _ := json.Marshal(v.Metadata)
		if _, err := stmt.ExecContext(ctx, v.ID, indexName, v.Namespace, string(valuesJSON), string(metadataJSON)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// DuckDB supports INSERT OR REPLACE
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO vectors (id, index_name, namespace, values_json, metadata_json)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}
		valuesJSON, _ := json.Marshal(v.Values)
		metadataJSON, _ := json.Marshal(v.Metadata)
		if _, err := stmt.ExecContext(ctx, v.ID, indexName, v.Namespace, string(valuesJSON), string(metadataJSON)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return nil, err
	}

	if len(vector) != idx.Dimensions {
		return nil, vectorize.ErrDimensionMismatch
	}

	// Build query
	query := `SELECT id, values_json, metadata_json FROM vectors WHERE index_name = ?`
	args := []interface{}{indexName}

	if opts.Namespace != "" {
		query += ` AND namespace = ?`
		args = append(args, opts.Namespace)
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scored struct {
		id       string
		score    float32
		values   []float32
		metadata map[string]any
	}

	var candidates []scored
	for rows.Next() {
		var id, valuesJSON, metadataJSON string
		if err := rows.Scan(&id, &valuesJSON, &metadataJSON); err != nil {
			return nil, err
		}

		var values []float32
		json.Unmarshal([]byte(valuesJSON), &values)

		var metadata map[string]any
		json.Unmarshal([]byte(metadataJSON), &metadata)

		// Apply metadata filter
		if len(opts.Filter) > 0 && !matchesFilter(metadata, opts.Filter) {
			continue
		}

		score := vectorize.ComputeScore(vector, values, idx.Metric)
		candidates = append(candidates, scored{id: id, score: score, values: values, metadata: metadata})
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Take top K
	if len(candidates) > opts.TopK {
		candidates = candidates[:opts.TopK]
	}

	matches := make([]*vectorize.Match, 0, len(candidates))
	for _, c := range candidates {
		if opts.ScoreThreshold > 0 && c.score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    c.id,
			Score: c.score,
		}

		if opts.ReturnValues {
			match.Values = c.values
		}
		if opts.ReturnMetadata {
			match.Metadata = c.metadata
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

	// Build query with placeholders
	query := `SELECT id, namespace, values_json, metadata_json FROM vectors WHERE index_name = ? AND id IN (`
	args := []interface{}{indexName}
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vectors []*vectorize.Vector
	for rows.Next() {
		var v vectorize.Vector
		var valuesJSON, metadataJSON string
		var namespace sql.NullString
		if err := rows.Scan(&v.ID, &namespace, &valuesJSON, &metadataJSON); err != nil {
			return nil, err
		}
		if namespace.Valid {
			v.Namespace = namespace.String
		}
		json.Unmarshal([]byte(valuesJSON), &v.Values)
		json.Unmarshal([]byte(metadataJSON), &v.Metadata)
		vectors = append(vectors, &v)
	}
	return vectors, rows.Err()
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `DELETE FROM vectors WHERE index_name = ? AND id IN (`
	args := []interface{}{indexName}
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	_, err := db.db.ExecContext(ctx, query, args...)
	return err
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

// Close releases resources.
func (db *DB) Close() error {
	return db.db.Close()
}

func matchesFilter(metadata map[string]any, filter map[string]any) bool {
	for k, expected := range filter {
		actual, ok := metadata[k]
		if !ok {
			return false
		}
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}
