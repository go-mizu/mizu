// Package duckdb provides a DuckDB embedded driver for the vectorize package.
// Import this package to register the "duckdb" driver.
// Note: This implementation uses the DuckDB VSS extension with HNSW indexing
// for accelerated vector similarity search.
package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"

	_ "github.com/duckdb/duckdb-go/v2"
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

	// Initialize schema and VSS extension
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func initSchema(db *sql.DB) error {
	// Install and load VSS extension for vector similarity search
	// This enables HNSW indexing and native array distance functions
	_, err := db.Exec(`INSTALL vss; LOAD vss;`)
	if err != nil {
		// VSS extension might already be installed, try just loading
		_, err = db.Exec(`LOAD vss;`)
		if err != nil {
			return fmt.Errorf("failed to load VSS extension: %w", err)
		}
	}

	// Enable experimental HNSW persistence for disk-based databases
	// This allows HNSW indexes to be persisted and loaded from disk
	_, _ = db.Exec(`SET hnsw_enable_experimental_persistence = true;`)

	// Create index metadata table
	schema := `
		CREATE TABLE IF NOT EXISTS vector_indexes (
			name VARCHAR PRIMARY KEY,
			dimensions INTEGER NOT NULL,
			metric VARCHAR NOT NULL DEFAULT 'cosine',
			description VARCHAR,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err = db.Exec(schema)
	return err
}

// DB implements vectorize.DB for DuckDB.
type DB struct {
	db *sql.DB
}

// vectorTableName returns the per-index vector table name
func (db *DB) vectorTableName(indexName string) string {
	// Sanitize index name for use in table name
	safe := strings.ReplaceAll(indexName, "-", "_")
	safe = strings.ReplaceAll(safe, ".", "_")
	return "vectors_" + safe
}

// metricToHNSW maps vectorize metric to DuckDB HNSW metric name
func metricToHNSW(metric vectorize.DistanceMetric) string {
	switch metric {
	case vectorize.Euclidean:
		return "l2sq"
	case vectorize.DotProduct:
		return "ip"
	default: // Cosine
		return "cosine"
	}
}

// metricToDistanceFunc returns the SQL distance function for a metric
func metricToDistanceFunc(metric vectorize.DistanceMetric) string {
	switch metric {
	case vectorize.Euclidean:
		return "array_distance"
	case vectorize.DotProduct:
		return "array_negative_inner_product"
	default: // Cosine
		return "array_cosine_distance"
	}
}

// CreateIndex creates a new index with a dedicated vector table and HNSW index.
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

	tableName := db.vectorTableName(index.Name)
	metricName := metricToHNSW(index.Metric)

	// Create per-index vector table with native FLOAT[] array type
	// This allows HNSW indexing for fast similarity search
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR PRIMARY KEY,
			namespace VARCHAR DEFAULT '',
			embedding FLOAT[%d],
			metadata_json VARCHAR DEFAULT '{}'
		)
	`, tableName, index.Dimensions)

	_, err = db.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create vector table: %w", err)
	}

	// Create HNSW index for fast similarity search
	createIndexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_hnsw_idx
		ON %s USING HNSW (embedding)
		WITH (metric = '%s')
	`, tableName, tableName, metricName)

	_, err = db.db.ExecContext(ctx, createIndexSQL)
	if err != nil {
		return fmt.Errorf("failed to create HNSW index: %w", err)
	}

	// Insert metadata
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

	// Get count from per-index vector table
	tableName := db.vectorTableName(name)
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, tableName)
	db.db.QueryRowContext(ctx, countSQL).Scan(&idx.VectorCount)

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

// DeleteIndex removes an index and its vector table.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	// Drop the per-index vector table (includes HNSW index)
	tableName := db.vectorTableName(name)
	dropSQL := fmt.Sprintf(`DROP TABLE IF EXISTS %s`, tableName)
	_, err := db.db.ExecContext(ctx, dropSQL)
	if err != nil {
		return err
	}

	// Delete metadata
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

// float32ArrayToSQL converts float32 slice to DuckDB array literal
func float32ArrayToSQL(values []float32) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// Insert adds vectors to an index.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	// Get index info
	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tableName := db.vectorTableName(indexName)

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Use INSERT with array literal syntax for FLOAT[] type
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (id, namespace, embedding, metadata_json)
		VALUES (?, ?, ?::FLOAT[%d], ?)
	`, tableName, idx.Dimensions)

	stmt, err := tx.PrepareContext(ctx, insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}
		embeddingLiteral := float32ArrayToSQL(v.Values)
		metadataJSON, _ := json.Marshal(v.Metadata)
		if _, err := stmt.ExecContext(ctx, v.ID, v.Namespace, embeddingLiteral, string(metadataJSON)); err != nil {
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

	tableName := db.vectorTableName(indexName)

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// DuckDB supports INSERT OR REPLACE
	upsertSQL := fmt.Sprintf(`
		INSERT OR REPLACE INTO %s (id, namespace, embedding, metadata_json)
		VALUES (?, ?, ?::FLOAT[%d], ?)
	`, tableName, idx.Dimensions)

	stmt, err := tx.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}
		embeddingLiteral := float32ArrayToSQL(v.Values)
		metadataJSON, _ := json.Marshal(v.Metadata)
		if _, err := stmt.ExecContext(ctx, v.ID, v.Namespace, embeddingLiteral, string(metadataJSON)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Search finds similar vectors using HNSW-accelerated similarity search.
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

	tableName := db.vectorTableName(indexName)
	distanceFunc := metricToDistanceFunc(idx.Metric)
	queryVector := float32ArrayToSQL(vector)

	// Build HNSW-accelerated search query
	// The ORDER BY ... LIMIT pattern triggers HNSW index usage
	var query string
	var args []interface{}

	if opts.Namespace != "" {
		query = fmt.Sprintf(`
			SELECT id, %s(embedding, %s::FLOAT[%d]) as distance, metadata_json
			FROM %s
			WHERE namespace = ?
			ORDER BY distance
			LIMIT ?
		`, distanceFunc, queryVector, idx.Dimensions, tableName)
		args = []interface{}{opts.Namespace, opts.TopK}
	} else {
		query = fmt.Sprintf(`
			SELECT id, %s(embedding, %s::FLOAT[%d]) as distance, metadata_json
			FROM %s
			ORDER BY distance
			LIMIT ?
		`, distanceFunc, queryVector, idx.Dimensions, tableName)
		args = []interface{}{opts.TopK}
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]*vectorize.Match, 0, opts.TopK)
	for rows.Next() {
		var id string
		var distance float64
		var metadataJSON string
		if err := rows.Scan(&id, &distance, &metadataJSON); err != nil {
			return nil, err
		}

		// Convert distance to similarity score (higher is better)
		// For cosine distance: score = 1 - distance
		// For L2/IP: score = 1 / (1 + distance)
		var score float32
		if idx.Metric == vectorize.Cosine {
			score = float32(1.0 - distance)
		} else {
			score = float32(1.0 / (1.0 + distance))
		}

		if opts.ScoreThreshold > 0 && score < opts.ScoreThreshold {
			continue
		}

		match := &vectorize.Match{
			ID:    id,
			Score: score,
		}

		if opts.ReturnMetadata {
			var metadata map[string]any
			json.Unmarshal([]byte(metadataJSON), &metadata)

			// Apply metadata filter if specified
			if len(opts.Filter) > 0 && !matchesFilter(metadata, opts.Filter) {
				continue
			}
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

	tableName := db.vectorTableName(indexName)

	// Build query with placeholders
	query := fmt.Sprintf(`SELECT id, namespace, embedding, metadata_json FROM %s WHERE id IN (`, tableName)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
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
		var namespace sql.NullString
		var metadataJSON string
		var embeddingRaw interface{}

		if err := rows.Scan(&v.ID, &namespace, &embeddingRaw, &metadataJSON); err != nil {
			return nil, err
		}
		if namespace.Valid {
			v.Namespace = namespace.String
		}

		// Parse embedding array from DuckDB
		if embSlice, ok := embeddingRaw.([]interface{}); ok {
			v.Values = make([]float32, len(embSlice))
			for i, val := range embSlice {
				if f, ok := val.(float64); ok {
					v.Values[i] = float32(f)
				}
			}
		}

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

	tableName := db.vectorTableName(indexName)

	query := fmt.Sprintf(`DELETE FROM %s WHERE id IN (`, tableName)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
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
