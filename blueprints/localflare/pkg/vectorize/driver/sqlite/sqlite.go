// Package sqlite provides a SQLite driver with sqlite-vec extension for the vectorize package.
// Import this package to register the "sqlite" driver.
// This driver uses the sqlite-vec extension for vector similarity search.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	// Enable sqlite-vec extension for all connections
	sqlite_vec.Auto()
	driver.Register("sqlite", &Driver{})
}

// Driver implements vectorize.Driver for SQLite with sqlite-vec.
type Driver struct{}

// Open creates a new SQLite connection.
// DSN format: /path/to/db.sqlite or :memory: for in-memory
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	// Verify sqlite-vec is available
	var vecVersion string
	if err := db.QueryRow("SELECT vec_version()").Scan(&vecVersion); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite-vec extension not available: %w", err)
	}

	// Initialize metadata schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return &DB{db: db}, nil
}

func initSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS vector_indexes (
			name TEXT PRIMARY KEY,
			dimensions INTEGER NOT NULL,
			metric TEXT NOT NULL DEFAULT 'cosine',
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(schema)
	return err
}

// DB implements vectorize.DB for SQLite.
type DB struct {
	db *sql.DB
}

// vectorTableName returns the vec0 virtual table name for an index.
func vectorTableName(indexName string) string {
	safe := strings.ReplaceAll(indexName, "-", "_")
	safe = strings.ReplaceAll(safe, ".", "_")
	return "vec_" + safe
}

// metadataTableName returns the metadata table name for an index.
func metadataTableName(indexName string) string {
	safe := strings.ReplaceAll(indexName, "-", "_")
	safe = strings.ReplaceAll(safe, ".", "_")
	return "meta_" + safe
}

// CreateIndex creates a new vec0 virtual table.
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

	vecTable := vectorTableName(index.Name)
	metaTable := metadataTableName(index.Name)

	// Determine distance metric
	// sqlite-vec uses L2 by default, but we can specify cosine
	distanceType := "float"
	if index.Metric == vectorize.Cosine {
		// sqlite-vec doesn't have native cosine, we'll use L2 on normalized vectors
		// or handle in the query
		distanceType = "float"
	}

	// Create vec0 virtual table for vectors
	createVecSQL := fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS %s USING vec0(
			embedding %s[%d]
		)
	`, vecTable, distanceType, index.Dimensions)

	_, err = db.db.ExecContext(ctx, createVecSQL)
	if err != nil {
		return fmt.Errorf("failed to create vec0 table: %w", err)
	}

	// Create metadata table (vec0 doesn't support extra columns)
	createMetaSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			rowid INTEGER PRIMARY KEY,
			id TEXT UNIQUE NOT NULL,
			namespace TEXT DEFAULT '',
			metadata_json TEXT DEFAULT '{}'
		)
	`, metaTable)

	_, err = db.db.ExecContext(ctx, createMetaSQL)
	if err != nil {
		return fmt.Errorf("failed to create metadata table: %w", err)
	}

	// Create index on id for fast lookups
	createIdxSQL := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_id_idx ON %s(id)`, metaTable, metaTable)
	db.db.ExecContext(ctx, createIdxSQL)

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

	// Get count from vec0 table
	vecTable := vectorTableName(name)
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, vecTable)
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

// DeleteIndex removes an index and its tables.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	vecTable := vectorTableName(name)
	metaTable := metadataTableName(name)

	// Drop tables
	db.db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, vecTable))
	db.db.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s`, metaTable))

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

// serializeVector converts float32 slice to JSON string for sqlite-vec.
func serializeVector(values []float32) string {
	// sqlite-vec accepts JSON array format
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

	idx, err := db.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	vecTable := vectorTableName(indexName)
	metaTable := metadataTableName(indexName)

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statements
	insertMetaSQL := fmt.Sprintf(`INSERT INTO %s (id, namespace, metadata_json) VALUES (?, ?, ?)`, metaTable)
	metaStmt, err := tx.PrepareContext(ctx, insertMetaSQL)
	if err != nil {
		return err
	}
	defer metaStmt.Close()

	insertVecSQL := fmt.Sprintf(`INSERT INTO %s (rowid, embedding) VALUES (?, ?)`, vecTable)
	vecStmt, err := tx.PrepareContext(ctx, insertVecSQL)
	if err != nil {
		return err
	}
	defer vecStmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		metadataJSON, _ := json.Marshal(v.Metadata)

		// Insert metadata first to get rowid
		result, err := metaStmt.ExecContext(ctx, v.ID, v.Namespace, string(metadataJSON))
		if err != nil {
			return err
		}

		rowid, err := result.LastInsertId()
		if err != nil {
			return err
		}

		// Insert vector with same rowid
		embedding := serializeVector(v.Values)
		if _, err := vecStmt.ExecContext(ctx, rowid, embedding); err != nil {
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

	vecTable := vectorTableName(indexName)
	metaTable := metadataTableName(indexName)

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return vectorize.ErrDimensionMismatch
		}

		metadataJSON, _ := json.Marshal(v.Metadata)
		embedding := serializeVector(v.Values)

		// Check if exists
		var existingRowid int64
		err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT rowid FROM %s WHERE id = ?`, metaTable), v.ID).Scan(&existingRowid)

		if err == sql.ErrNoRows {
			// Insert new
			insertMetaSQL := fmt.Sprintf(`INSERT INTO %s (id, namespace, metadata_json) VALUES (?, ?, ?)`, metaTable)
			result, err := tx.ExecContext(ctx, insertMetaSQL, v.ID, v.Namespace, string(metadataJSON))
			if err != nil {
				return err
			}
			rowid, _ := result.LastInsertId()

			insertVecSQL := fmt.Sprintf(`INSERT INTO %s (rowid, embedding) VALUES (?, ?)`, vecTable)
			if _, err := tx.ExecContext(ctx, insertVecSQL, rowid, embedding); err != nil {
				return err
			}
		} else if err == nil {
			// Update existing
			updateMetaSQL := fmt.Sprintf(`UPDATE %s SET namespace = ?, metadata_json = ? WHERE rowid = ?`, metaTable)
			if _, err := tx.ExecContext(ctx, updateMetaSQL, v.Namespace, string(metadataJSON), existingRowid); err != nil {
				return err
			}

			// vec0 doesn't support UPDATE, need to DELETE and INSERT
			deleteVecSQL := fmt.Sprintf(`DELETE FROM %s WHERE rowid = ?`, vecTable)
			if _, err := tx.ExecContext(ctx, deleteVecSQL, existingRowid); err != nil {
				return err
			}

			insertVecSQL := fmt.Sprintf(`INSERT INTO %s (rowid, embedding) VALUES (?, ?)`, vecTable)
			if _, err := tx.ExecContext(ctx, insertVecSQL, existingRowid, embedding); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return tx.Commit()
}

// Search finds similar vectors using vec0's KNN search.
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

	vecTable := vectorTableName(indexName)
	metaTable := metadataTableName(indexName)
	queryVector := serializeVector(vector)

	// Build KNN search query using vec0's MATCH syntax
	var query string
	var args []interface{}

	if opts.Namespace != "" {
		query = fmt.Sprintf(`
			SELECT v.rowid, v.distance, m.id, m.metadata_json
			FROM %s v
			JOIN %s m ON v.rowid = m.rowid
			WHERE v.embedding MATCH ?
			  AND k = ?
			  AND m.namespace = ?
			ORDER BY v.distance
		`, vecTable, metaTable)
		args = []interface{}{queryVector, opts.TopK, opts.Namespace}
	} else {
		query = fmt.Sprintf(`
			SELECT v.rowid, v.distance, m.id, m.metadata_json
			FROM %s v
			JOIN %s m ON v.rowid = m.rowid
			WHERE v.embedding MATCH ?
			  AND k = ?
			ORDER BY v.distance
		`, vecTable, metaTable)
		args = []interface{}{queryVector, opts.TopK}
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make([]*vectorize.Match, 0, opts.TopK)
	for rows.Next() {
		var rowid int64
		var distance float64
		var id, metadataJSON string

		if err := rows.Scan(&rowid, &distance, &id, &metadataJSON); err != nil {
			return nil, err
		}

		// Convert L2 distance to similarity score (higher is better)
		// score = 1 / (1 + distance)
		score := float32(1.0 / (1.0 + distance))

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

			// Apply metadata filter
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

	metaTable := metadataTableName(indexName)

	// Build query with placeholders
	query := fmt.Sprintf(`SELECT rowid, id, namespace, metadata_json FROM %s WHERE id IN (`, metaTable)
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
		var rowid int64
		var v vectorize.Vector
		var namespace sql.NullString
		var metadataJSON string

		if err := rows.Scan(&rowid, &v.ID, &namespace, &metadataJSON); err != nil {
			return nil, err
		}
		if namespace.Valid {
			v.Namespace = namespace.String
		}
		json.Unmarshal([]byte(metadataJSON), &v.Metadata)

		// Note: vec0 doesn't support retrieving vector values easily
		// Would need to query the vec0 table by rowid but it's not straightforward
		vectors = append(vectors, &v)
	}

	return vectors, rows.Err()
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	vecTable := vectorTableName(indexName)
	metaTable := metadataTableName(indexName)

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, id := range ids {
		// Get rowid first
		var rowid int64
		err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT rowid FROM %s WHERE id = ?`, metaTable), id).Scan(&rowid)
		if err != nil {
			continue // Skip if not found
		}

		// Delete from vec0 table
		tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE rowid = ?`, vecTable), rowid)

		// Delete from metadata table
		tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE rowid = ?`, metaTable), rowid)
	}

	return tx.Commit()
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
