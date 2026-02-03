package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	_ "modernc.org/sqlite"
)

// MemoryStore provides SQLite-backed storage for the memory index.
// It manages files, chunks, FTS5 full-text search, and embedding vectors.
type MemoryStore struct {
	db *sql.DB
}

// NewMemoryStore opens (or creates) a SQLite database at dbPath and returns
// a ready-to-use MemoryStore. Call EnsureSchema before first use.
func NewMemoryStore(dbPath string) (*MemoryStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}

	// Enable WAL and foreign keys for consistency with the rest of the codebase.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %s: %w", pragma, err)
		}
	}

	return &MemoryStore{db: db}, nil
}

// EnsureSchema creates all required tables and indexes if they do not exist.
// The schema matches OpenClaw's memory storage layout.
func (s *MemoryStore) EnsureSchema() error {
	queries := []string{
		// files: tracks indexed files and their content hashes for incremental re-indexing.
		`CREATE TABLE IF NOT EXISTS files (
			path    TEXT PRIMARY KEY,
			source  TEXT NOT NULL DEFAULT '',
			hash    TEXT NOT NULL DEFAULT '',
			mtime   INTEGER NOT NULL DEFAULT 0,
			size    INTEGER NOT NULL DEFAULT 0
		)`,

		// chunks: stores document fragments with optional embeddings.
		`CREATE TABLE IF NOT EXISTS chunks (
			id         TEXT PRIMARY KEY,
			path       TEXT NOT NULL,
			source     TEXT NOT NULL DEFAULT '',
			start_line INTEGER NOT NULL DEFAULT 0,
			end_line   INTEGER NOT NULL DEFAULT 0,
			hash       TEXT NOT NULL DEFAULT '',
			model      TEXT NOT NULL DEFAULT '',
			text       TEXT NOT NULL DEFAULT '',
			embedding  TEXT NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		`CREATE INDEX IF NOT EXISTS idx_chunks_path ON chunks(path)`,
		`CREATE INDEX IF NOT EXISTS idx_chunks_hash ON chunks(hash)`,

		// chunks_fts: FTS5 virtual table for keyword search over chunk text.
		`CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
			id,
			path,
			text,
			content=chunks,
			content_rowid=rowid
		)`,

		// Triggers to keep FTS index in sync with the chunks table.
		`CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
			INSERT INTO chunks_fts(rowid, id, path, text)
			VALUES (new.rowid, new.id, new.path, new.text);
		END`,

		`CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
			INSERT INTO chunks_fts(chunks_fts, rowid, id, path, text)
			VALUES ('delete', old.rowid, old.id, old.path, old.text);
		END`,

		`CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
			INSERT INTO chunks_fts(chunks_fts, rowid, id, path, text)
			VALUES ('delete', old.rowid, old.id, old.path, old.text);
			INSERT INTO chunks_fts(rowid, id, path, text)
			VALUES (new.rowid, new.id, new.path, new.text);
		END`,

		// embedding_cache: caches embedding vectors by content hash to avoid
		// redundant API calls for unchanged content.
		`CREATE TABLE IF NOT EXISTS embedding_cache (
			hash       TEXT NOT NULL,
			model      TEXT NOT NULL,
			embedding  TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (hash, model)
		)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("schema exec: %w", err)
		}
	}

	return nil
}

// UpsertFile inserts or updates a file tracking record.
func (s *MemoryStore) UpsertFile(path, source, hash string, mtime, size int64) error {
	_, err := s.db.Exec(`
		INSERT INTO files (path, source, hash, mtime, size)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			source = excluded.source,
			hash   = excluded.hash,
			mtime  = excluded.mtime,
			size   = excluded.size
	`, path, source, hash, mtime, size)
	if err != nil {
		return fmt.Errorf("upsert file %s: %w", path, err)
	}
	return nil
}

// UpsertChunk inserts or updates a chunk record. The embedding is stored
// as a JSON array of floats.
func (s *MemoryStore) UpsertChunk(id, path, source string, startLine, endLine int, hash, model, text string, embedding []float64) error {
	embJSON, err := encodeEmbedding(embedding)
	if err != nil {
		return fmt.Errorf("encode embedding: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO chunks (id, path, source, start_line, end_line, hash, model, text, embedding, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			path       = excluded.path,
			source     = excluded.source,
			start_line = excluded.start_line,
			end_line   = excluded.end_line,
			hash       = excluded.hash,
			model      = excluded.model,
			text       = excluded.text,
			embedding  = excluded.embedding,
			updated_at = excluded.updated_at
	`, id, path, source, startLine, endLine, hash, model, text, embJSON, time.Now().UTC().Format(time.DateTime))
	if err != nil {
		return fmt.Errorf("upsert chunk %s: %w", id, err)
	}
	return nil
}

// DeleteChunksByPath removes all chunks associated with the given file path.
func (s *MemoryStore) DeleteChunksByPath(path string) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE path = ?`, path)
	if err != nil {
		return fmt.Errorf("delete chunks for %s: %w", path, err)
	}
	return nil
}

// GetFileHash returns the stored content hash for a file, or empty string
// if the file has not been indexed.
func (s *MemoryStore) GetFileHash(path string) (string, error) {
	var hash string
	err := s.db.QueryRow(`SELECT hash FROM files WHERE path = ?`, path).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get file hash %s: %w", path, err)
	}
	return hash, nil
}

// DeleteFile removes a file record from the files table.
func (s *MemoryStore) DeleteFile(path string) error {
	_, err := s.db.Exec(`DELETE FROM files WHERE path = ?`, path)
	if err != nil {
		return fmt.Errorf("delete file %s: %w", path, err)
	}
	return nil
}

// SearchFTS performs a full-text search using FTS5 BM25 ranking.
// Results are ordered by relevance (best match first).
func (s *MemoryStore) SearchFTS(query string, limit int) ([]KeywordResult, error) {
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	ftsQuery := BuildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT
			c.path,
			c.start_line,
			c.end_line,
			c.text,
			c.source,
			bm25(chunks_fts) AS rank
		FROM chunks_fts f
		JOIN chunks c ON c.id = f.id
		WHERE chunks_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("FTS search: %w", err)
	}
	defer rows.Close()

	var results []KeywordResult
	for rows.Next() {
		var r KeywordResult
		if err := rows.Scan(&r.Path, &r.StartLine, &r.EndLine, &r.Snippet, &r.Source, &r.Rank); err != nil {
			return nil, fmt.Errorf("scan FTS result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate FTS results: %w", err)
	}

	return results, nil
}

// SearchFTSWithSource performs a source-filtered full-text search using FTS5.
func (s *MemoryStore) SearchFTSWithSource(query, source string, limit int) ([]KeywordResult, error) {
	if source == "" {
		return s.SearchFTS(query, limit)
	}
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	ftsQuery := BuildFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	rows, err := s.db.Query(`
		SELECT
			c.path,
			c.start_line,
			c.end_line,
			c.text,
			c.source,
			bm25(chunks_fts) AS rank
		FROM chunks_fts f
		JOIN chunks c ON c.id = f.id
		WHERE chunks_fts MATCH ? AND c.source = ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, source, limit)
	if err != nil {
		return nil, fmt.Errorf("FTS search with source: %w", err)
	}
	defer rows.Close()

	var results []KeywordResult
	for rows.Next() {
		var r KeywordResult
		if err := rows.Scan(&r.Path, &r.StartLine, &r.EndLine, &r.Snippet, &r.Source, &r.Rank); err != nil {
			return nil, fmt.Errorf("scan FTS result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate FTS results: %w", err)
	}

	return results, nil
}

// SearchVectorWithSource performs a source-filtered vector similarity search.
func (s *MemoryStore) SearchVectorWithSource(queryVec []float64, source string, limit int) ([]VectorResult, error) {
	if source == "" {
		return s.SearchVector(queryVec, limit)
	}
	if len(queryVec) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT id, path, source, start_line, end_line, text, embedding
		FROM chunks
		WHERE embedding != '' AND source = ?
	`, source)
	if err != nil {
		return nil, fmt.Errorf("vector search query: %w", err)
	}
	defer rows.Close()

	type scored struct {
		result VectorResult
		score  float64
	}

	var candidates []scored
	for rows.Next() {
		var (
			id, path, src, text, embJSON string
			startLine, endLine           int
		)
		if err := rows.Scan(&id, &path, &src, &startLine, &endLine, &text, &embJSON); err != nil {
			return nil, fmt.Errorf("scan vector row: %w", err)
		}

		emb, err := decodeEmbedding(embJSON)
		if err != nil || len(emb) == 0 {
			continue
		}

		sim := cosineSimilarity(queryVec, emb)
		if sim > 0 {
			candidates = append(candidates, scored{
				result: VectorResult{
					Path:      path,
					StartLine: startLine,
					EndLine:   endLine,
					Score:     sim,
					Snippet:   text,
					Source:    src,
				},
				score: sim,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vector rows: %w", err)
	}

	// Sort by descending similarity.
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	results := make([]VectorResult, len(candidates))
	for i, c := range candidates {
		results[i] = c.result
	}
	return results, nil
}

// SearchVector performs a brute-force cosine similarity search across all
// stored embeddings. This is suitable for small-to-medium corpora (< 100k
// chunks). For larger datasets, consider a dedicated vector store.
func (s *MemoryStore) SearchVector(queryVec []float64, limit int) ([]VectorResult, error) {
	if len(queryVec) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT id, path, source, start_line, end_line, text, embedding
		FROM chunks
		WHERE embedding != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("vector search query: %w", err)
	}
	defer rows.Close()

	type scored struct {
		result VectorResult
		score  float64
	}

	var candidates []scored
	for rows.Next() {
		var (
			id, path, source, text, embJSON string
			startLine, endLine              int
		)
		if err := rows.Scan(&id, &path, &source, &startLine, &endLine, &text, &embJSON); err != nil {
			return nil, fmt.Errorf("scan vector row: %w", err)
		}

		emb, err := decodeEmbedding(embJSON)
		if err != nil || len(emb) == 0 {
			continue
		}

		sim := cosineSimilarity(queryVec, emb)
		if sim > 0 {
			candidates = append(candidates, scored{
				result: VectorResult{
					Path:      path,
					StartLine: startLine,
					EndLine:   endLine,
					Score:     sim,
					Snippet:   text,
					Source:    source,
				},
				score: sim,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate vector rows: %w", err)
	}

	// Sort by descending similarity.
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	results := make([]VectorResult, len(candidates))
	for i, c := range candidates {
		results[i] = c.result
	}
	return results, nil
}

// GetCachedEmbedding retrieves a cached embedding for the given content hash
// and model. Returns nil if not cached.
func (s *MemoryStore) GetCachedEmbedding(hash, model string) ([]float64, error) {
	var embJSON string
	err := s.db.QueryRow(`
		SELECT embedding FROM embedding_cache WHERE hash = ? AND model = ?
	`, hash, model).Scan(&embJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get cached embedding: %w", err)
	}
	return decodeEmbedding(embJSON)
}

// SetCachedEmbedding stores an embedding in the cache.
func (s *MemoryStore) SetCachedEmbedding(hash, model string, embedding []float64) error {
	embJSON, err := encodeEmbedding(embedding)
	if err != nil {
		return fmt.Errorf("encode cached embedding: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO embedding_cache (hash, model, embedding)
		VALUES (?, ?, ?)
		ON CONFLICT(hash, model) DO UPDATE SET
			embedding  = excluded.embedding,
			created_at = datetime('now')
	`, hash, model, embJSON)
	if err != nil {
		return fmt.Errorf("set cached embedding: %w", err)
	}
	return nil
}

// Stats returns aggregate counts of indexed files and chunks.
func (s *MemoryStore) Stats() (fileCount int, chunkCount int, err error) {
	row := s.db.QueryRow("SELECT COUNT(*) FROM files")
	if err := row.Scan(&fileCount); err != nil {
		return 0, 0, err
	}
	row = s.db.QueryRow("SELECT COUNT(*) FROM chunks")
	if err := row.Scan(&chunkCount); err != nil {
		return 0, 0, err
	}
	return fileCount, chunkCount, nil
}

// Close releases database resources.
func (s *MemoryStore) Close() error {
	return s.db.Close()
}

// encodeEmbedding serializes an embedding vector to a JSON array string.
func encodeEmbedding(v []float64) (string, error) {
	if len(v) == 0 {
		return "", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeEmbedding deserializes a JSON array string to an embedding vector.
func decodeEmbedding(s string) ([]float64, error) {
	if s == "" {
		return nil, nil
	}
	var v []float64
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	return dot / denom
}
