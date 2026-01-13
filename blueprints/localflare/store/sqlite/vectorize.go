package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/go-mizu/blueprints/localflare/store"
)

// VectorizeStoreImpl implements store.VectorizeStore using SQLite.
// For production, consider using DuckDB or a specialized vector database.
type VectorizeStoreImpl struct {
	db *sql.DB
}

func (s *VectorizeStoreImpl) CreateIndex(ctx context.Context, index *store.VectorIndex) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO vector_indexes (id, name, description, dimensions, metric, created_at, vector_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		index.ID, index.Name, index.Description, index.Dimensions, index.Metric, index.CreatedAt, 0)
	return err
}

func (s *VectorizeStoreImpl) GetIndex(ctx context.Context, name string) (*store.VectorIndex, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, dimensions, metric, created_at, vector_count
		FROM vector_indexes WHERE name = ?`, name)
	var idx store.VectorIndex
	var desc sql.NullString
	if err := row.Scan(&idx.ID, &idx.Name, &desc, &idx.Dimensions, &idx.Metric, &idx.CreatedAt, &idx.VectorCount); err != nil {
		return nil, err
	}
	idx.Description = desc.String
	return &idx, nil
}

func (s *VectorizeStoreImpl) ListIndexes(ctx context.Context) ([]*store.VectorIndex, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, dimensions, metric, created_at, vector_count
		FROM vector_indexes ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*store.VectorIndex
	for rows.Next() {
		var idx store.VectorIndex
		var desc sql.NullString
		if err := rows.Scan(&idx.ID, &idx.Name, &desc, &idx.Dimensions, &idx.Metric, &idx.CreatedAt, &idx.VectorCount); err != nil {
			return nil, err
		}
		idx.Description = desc.String
		indexes = append(indexes, &idx)
	}
	return indexes, rows.Err()
}

func (s *VectorizeStoreImpl) DeleteIndex(ctx context.Context, name string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get index ID first
	var indexID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM vector_indexes WHERE name = ?`, name).Scan(&indexID)
	if err != nil {
		return err
	}

	// Delete vectors
	_, err = tx.ExecContext(ctx, `DELETE FROM vectors WHERE index_id = ?`, indexID)
	if err != nil {
		return err
	}

	// Delete index
	_, err = tx.ExecContext(ctx, `DELETE FROM vector_indexes WHERE name = ?`, name)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *VectorizeStoreImpl) Insert(ctx context.Context, indexName string, vectors []*store.Vector) error {
	// Get index ID and dimensions
	idx, err := s.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO vectors (id, index_id, namespace, values_json, metadata)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return fmt.Errorf("vector dimension mismatch: expected %d, got %d", idx.Dimensions, len(v.Values))
		}
		valuesJSON, _ := json.Marshal(v.Values)
		metadataJSON, _ := json.Marshal(v.Metadata)
		if _, err := stmt.ExecContext(ctx, v.ID, idx.ID, v.Namespace, valuesJSON, metadataJSON); err != nil {
			return err
		}
	}

	// Update vector count
	_, err = tx.ExecContext(ctx,
		`UPDATE vector_indexes SET vector_count = vector_count + ? WHERE id = ?`,
		len(vectors), idx.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *VectorizeStoreImpl) Upsert(ctx context.Context, indexName string, vectors []*store.Vector) error {
	idx, err := s.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO vectors (id, index_id, namespace, values_json, metadata)
		VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, v := range vectors {
		if len(v.Values) != idx.Dimensions {
			return fmt.Errorf("vector dimension mismatch: expected %d, got %d", idx.Dimensions, len(v.Values))
		}
		valuesJSON, _ := json.Marshal(v.Values)
		metadataJSON, _ := json.Marshal(v.Metadata)
		if _, err := stmt.ExecContext(ctx, v.ID, idx.ID, v.Namespace, valuesJSON, metadataJSON); err != nil {
			return err
		}
	}

	// Recalculate vector count
	var count int64
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM vectors WHERE index_id = ?`, idx.ID).Scan(&count)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE vector_indexes SET vector_count = ? WHERE id = ?`, count, idx.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *VectorizeStoreImpl) Query(ctx context.Context, indexName string, vector []float32, opts *store.VectorQueryOptions) ([]*store.VectorMatch, error) {
	idx, err := s.GetIndex(ctx, indexName)
	if err != nil {
		return nil, err
	}

	if len(vector) != idx.Dimensions {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d", idx.Dimensions, len(vector))
	}

	topK := 10
	if opts != nil && opts.TopK > 0 {
		topK = opts.TopK
		if topK > 100 {
			topK = 100
		}
	}

	// Build query
	query := `SELECT id, values_json, metadata FROM vectors WHERE index_id = ?`
	args := []interface{}{idx.ID}

	if opts != nil && opts.Namespace != "" {
		query += ` AND namespace = ?`
		args = append(args, opts.Namespace)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type scoredVector struct {
		id       string
		score    float32
		values   []float32
		metadata map[string]interface{}
	}

	var candidates []scoredVector
	for rows.Next() {
		var id, valuesJSON, metadataJSON string
		if err := rows.Scan(&id, &valuesJSON, &metadataJSON); err != nil {
			return nil, err
		}

		var values []float32
		json.Unmarshal([]byte(valuesJSON), &values)

		var metadata map[string]interface{}
		json.Unmarshal([]byte(metadataJSON), &metadata)

		// Apply metadata filter if specified
		if opts != nil && opts.Filter != nil && !matchesFilter(metadata, opts.Filter) {
			continue
		}

		// Calculate similarity score
		var score float32
		switch idx.Metric {
		case "cosine":
			score = cosineSimilarity(vector, values)
		case "euclidean":
			score = 1.0 / (1.0 + euclideanDistance(vector, values))
		case "dot-product":
			score = dotProduct(vector, values)
		default:
			score = cosineSimilarity(vector, values)
		}

		candidates = append(candidates, scoredVector{
			id:       id,
			score:    score,
			values:   values,
			metadata: metadata,
		})
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Take top K
	if len(candidates) > topK {
		candidates = candidates[:topK]
	}

	// Build results
	results := make([]*store.VectorMatch, len(candidates))
	for i, c := range candidates {
		match := &store.VectorMatch{
			ID:    c.id,
			Score: c.score,
		}
		if opts != nil && opts.ReturnValues {
			match.Values = c.values
		}
		if opts != nil && (opts.ReturnMetadata == "indexed" || opts.ReturnMetadata == "all") {
			match.Metadata = c.metadata
		}
		results[i] = match
	}

	return results, nil
}

func (s *VectorizeStoreImpl) GetByIDs(ctx context.Context, indexName string, ids []string) ([]*store.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	idx, err := s.GetIndex(ctx, indexName)
	if err != nil {
		return nil, err
	}

	query := `SELECT id, namespace, values_json, metadata FROM vectors WHERE index_id = ? AND id IN (`
	args := []interface{}{idx.ID}
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vectors []*store.Vector
	for rows.Next() {
		var v store.Vector
		var valuesJSON, metadataJSON string
		var namespace sql.NullString
		if err := rows.Scan(&v.ID, &namespace, &valuesJSON, &metadataJSON); err != nil {
			return nil, err
		}
		v.Namespace = namespace.String
		json.Unmarshal([]byte(valuesJSON), &v.Values)
		json.Unmarshal([]byte(metadataJSON), &v.Metadata)
		vectors = append(vectors, &v)
	}
	return vectors, rows.Err()
}

func (s *VectorizeStoreImpl) DeleteByIDs(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	idx, err := s.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM vectors WHERE index_id = ? AND id IN (`
	args := []interface{}{idx.ID}
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args = append(args, id)
	}
	query += ")"

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	// Update vector count
	var count int64
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM vectors WHERE index_id = ?`, idx.ID).Scan(&count)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE vector_indexes SET vector_count = ? WHERE id = ?`, count, idx.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *VectorizeStoreImpl) DeleteByNamespace(ctx context.Context, indexName, namespace string) error {
	idx, err := s.GetIndex(ctx, indexName)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM vectors WHERE index_id = ? AND namespace = ?`, idx.ID, namespace)
	if err != nil {
		return err
	}

	// Update vector count
	var count int64
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM vectors WHERE index_id = ?`, idx.ID).Scan(&count)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE vector_indexes SET vector_count = ? WHERE id = ?`, count, idx.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Vector similarity functions

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dotProd, normA, normB float32
	for i := range a {
		dotProd += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProd / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func euclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return float32(math.Sqrt(float64(sum)))
}

func dotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}

func matchesFilter(metadata map[string]interface{}, filter map[string]interface{}) bool {
	for key, expected := range filter {
		actual, exists := metadata[key]
		if !exists {
			return false
		}
		// Simple equality check
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}

// Schema for Vectorize
const vectorizeSchema = `
	-- Vector Indexes
	CREATE TABLE IF NOT EXISTS vector_indexes (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		dimensions INTEGER NOT NULL,
		metric TEXT DEFAULT 'cosine',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		vector_count INTEGER DEFAULT 0
	);

	-- Vectors
	CREATE TABLE IF NOT EXISTS vectors (
		id TEXT NOT NULL,
		index_id TEXT NOT NULL,
		namespace TEXT,
		values_json TEXT NOT NULL,
		metadata TEXT DEFAULT '{}',
		PRIMARY KEY (index_id, id),
		FOREIGN KEY (index_id) REFERENCES vector_indexes(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_vectors_namespace ON vectors(index_id, namespace);
`
