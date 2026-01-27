package fineweb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// FTSConfig contains configuration for FTS index creation.
type FTSConfig struct {
	// Stemmer algorithm: porter, arabic, german, etc. or "none"
	Stemmer string
	// Stopwords: "english", "german", "none", or custom table name
	Stopwords string
	// StripAccents removes diacritical marks (á → a)
	StripAccents bool
	// Lower converts text to lowercase
	Lower bool
	// MemoryLimit for DuckDB during indexing (e.g., "8GB")
	MemoryLimit string
	// Threads limits parallelism during indexing
	Threads int
	// TempDirectory for disk spilling during large index creation
	TempDirectory string
	// MaxTempDirectorySize limits temp directory usage (e.g., "50GB")
	MaxTempDirectorySize string
	// PreserveInsertionOrder can be disabled to reduce memory usage
	PreserveInsertionOrder bool
}

// DefaultFTSConfig returns sensible defaults for FTS indexing.
func DefaultFTSConfig() FTSConfig {
	return FTSConfig{
		Stemmer:                "porter",
		Stopwords:              "english",
		StripAccents:           true,
		Lower:                  true,
		MemoryLimit:            "4GB",
		Threads:                4,
		TempDirectory:          "",
		MaxTempDirectorySize:   "100GB",
		PreserveInsertionOrder: false, // Disable to reduce memory for large datasets
	}
}

// SearchResult contains search result with timing information.
type SearchResult struct {
	Documents []Document    `json:"documents"`
	Duration  time.Duration `json:"duration"`
	Method    string        `json:"method"` // Driver name or search method
	Total     int64         `json:"total"`  // Total matching documents (if known)
}

// Store manages DuckDB connection and queries for a language.
type Store struct {
	db      *sql.DB
	lang    string
	dataDir string
	dbPath  string
}

// NewStore creates a store for a language.
func NewStore(lang, dataDir string) (*Store, error) {
	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, lang+".duckdb")

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &Store{
		db:      db,
		lang:    lang,
		dataDir: dataDir,
		dbPath:  dbPath,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return store, nil
}

func (s *Store) initSchema() error {
	// Create documents table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id VARCHAR PRIMARY KEY,
			url VARCHAR,
			text VARCHAR,
			dump VARCHAR,
			date VARCHAR,
			language VARCHAR,
			language_score DOUBLE
		)
	`)
	if err != nil {
		return fmt.Errorf("creating documents table: %w", err)
	}

	// Create import state table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS import_state (
			parquet_file VARCHAR PRIMARY KEY,
			imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			row_count INTEGER
		)
	`)
	if err != nil {
		return fmt.Errorf("creating import_state table: %w", err)
	}

	return nil
}

// Import imports parquet files into DuckDB.
func (s *Store) Import(ctx context.Context, parquetDir string, progress func(file string, rows int64)) error {
	// Find parquet files
	entries, err := os.ReadDir(parquetDir)
	if err != nil {
		return fmt.Errorf("reading parquet directory: %w", err)
	}

	var parquetFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".parquet") {
			parquetFiles = append(parquetFiles, filepath.Join(parquetDir, entry.Name()))
		}
	}

	if len(parquetFiles) == 0 {
		return fmt.Errorf("no parquet files found in %s", parquetDir)
	}

	sort.Strings(parquetFiles)

	// Import each file
	for _, parquetPath := range parquetFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		filename := filepath.Base(parquetPath)

		// Check if already imported
		var exists bool
		err := s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) > 0 FROM import_state WHERE parquet_file = ?",
			filename,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking import state: %w", err)
		}
		if exists {
			continue // Skip already imported
		}

		// Import parquet file
		rows, err := s.importParquetFile(ctx, parquetPath)
		if err != nil {
			return fmt.Errorf("importing %s: %w", filename, err)
		}

		// Record import state
		_, err = s.db.ExecContext(ctx,
			"INSERT INTO import_state (parquet_file, row_count) VALUES (?, ?)",
			filename, rows,
		)
		if err != nil {
			return fmt.Errorf("recording import state: %w", err)
		}

		if progress != nil {
			progress(filename, rows)
		}
	}

	return nil
}

func (s *Store) importParquetFile(ctx context.Context, parquetPath string) (int64, error) {
	// Use INSERT INTO ... SELECT to append data
	query := fmt.Sprintf(`
		INSERT INTO documents (id, url, text, dump, date, language, language_score)
		SELECT
			id,
			url,
			text,
			dump,
			date,
			language,
			language_score
		FROM read_parquet('%s')
		ON CONFLICT (id) DO NOTHING
	`, parquetPath)

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

// CreateFTSIndex creates the full-text search index with default config.
func (s *Store) CreateFTSIndex(ctx context.Context) error {
	return s.CreateFTSIndexWithConfig(ctx, DefaultFTSConfig())
}

// CreateFTSIndexWithConfig creates the full-text search index with custom configuration.
// This configures DuckDB for handling large datasets by:
// - Setting memory limits and thread count
// - Configuring temp directory for disk spilling
// - Using appropriate stemmer and stopwords
func (s *Store) CreateFTSIndexWithConfig(ctx context.Context, cfg FTSConfig) error {
	// Configure DuckDB for large dataset handling
	if err := s.configureDuckDB(ctx, cfg); err != nil {
		return fmt.Errorf("configuring DuckDB: %w", err)
	}

	// Install and load FTS extension
	_, err := s.db.ExecContext(ctx, "INSTALL fts")
	if err != nil && !strings.Contains(err.Error(), "already installed") {
		return fmt.Errorf("installing FTS extension: %w", err)
	}

	_, err = s.db.ExecContext(ctx, "LOAD fts")
	if err != nil {
		return fmt.Errorf("loading FTS extension: %w", err)
	}

	// Build FTS index creation query with config
	stripAccents := 0
	if cfg.StripAccents {
		stripAccents = 1
	}
	lower := 0
	if cfg.Lower {
		lower = 1
	}

	// Create FTS index with configuration
	// Note: DuckDB FTS uses PRAGMA create_fts_index
	query := fmt.Sprintf(`
		PRAGMA create_fts_index(
			'documents', 'id', 'text', 'url',
			stemmer = '%s',
			stopwords = '%s',
			strip_accents = %d,
			lower = %d,
			overwrite = 1
		)
	`, cfg.Stemmer, cfg.Stopwords, stripAccents, lower)

	_, err = s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("creating FTS index: %w", err)
	}

	return nil
}

// configureDuckDB sets up DuckDB configuration for large dataset processing.
func (s *Store) configureDuckDB(ctx context.Context, cfg FTSConfig) error {
	// Set memory limit
	if cfg.MemoryLimit != "" {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("SET memory_limit = '%s'", cfg.MemoryLimit)); err != nil {
			return fmt.Errorf("setting memory_limit: %w", err)
		}
	}

	// Set thread count
	if cfg.Threads > 0 {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("SET threads = %d", cfg.Threads)); err != nil {
			return fmt.Errorf("setting threads: %w", err)
		}
	}

	// Set temp directory for disk spilling
	// Note: temp_directory can only be set before it's first used
	if cfg.TempDirectory != "" {
		// Ensure temp directory exists
		if err := os.MkdirAll(cfg.TempDirectory, 0755); err != nil {
			return fmt.Errorf("creating temp directory: %w", err)
		}
		// Try to set temp directory, ignore error if already in use
		_, _ = s.db.ExecContext(ctx, fmt.Sprintf("SET temp_directory = '%s'", cfg.TempDirectory))
	}

	// Set max temp directory size
	if cfg.MaxTempDirectorySize != "" {
		if _, err := s.db.ExecContext(ctx, fmt.Sprintf("SET max_temp_directory_size = '%s'", cfg.MaxTempDirectorySize)); err != nil {
			// Ignore error - might not be settable if temp already used
			_ = err
		}
	}

	// Disable insertion order preservation to reduce memory usage
	if !cfg.PreserveInsertionOrder {
		if _, err := s.db.ExecContext(ctx, "SET preserve_insertion_order = false"); err != nil {
			return fmt.Errorf("setting preserve_insertion_order: %w", err)
		}
	}

	return nil
}

// HasFTSIndex checks if the FTS index exists and is functional for this store.
func (s *Store) HasFTSIndex(ctx context.Context) bool {
	// Load FTS extension first
	if _, err := s.db.ExecContext(ctx, "LOAD fts"); err != nil {
		return false
	}

	// Check if the FTS schema exists
	var schemaCount int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.schemata
		WHERE schema_name = 'fts_main_documents'
	`).Scan(&schemaCount)
	if err != nil || schemaCount == 0 {
		return false
	}

	// Verify the FTS function actually works by running a test query
	// This catches cases where the schema exists but index is incomplete/corrupted
	var testScore sql.NullFloat64
	err = s.db.QueryRowContext(ctx, `
		SELECT fts_main_documents.match_bm25(id, 'test', fields := 'text') AS score
		FROM documents
		LIMIT 1
	`).Scan(&testScore)
	if err != nil {
		// FTS function doesn't work - index is not functional
		return false
	}

	return true
}

// DropFTSIndex removes the FTS index if it exists.
func (s *Store) DropFTSIndex(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "LOAD fts")
	if err != nil {
		return nil // FTS not loaded, nothing to drop
	}

	_, err = s.db.ExecContext(ctx, "PRAGMA drop_fts_index('documents')")
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return fmt.Errorf("dropping FTS index: %w", err)
	}
	return nil
}

// Search performs full-text search using BM25.
func (s *Store) Search(ctx context.Context, query string, limit, offset int) ([]Document, error) {
	result, err := s.SearchFTS(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// SearchFTS performs full-text search with BM25 scoring and returns timing info.
func (s *Store) SearchFTS(ctx context.Context, query string, limit, offset int) (*SearchResult, error) {
	return s.SearchFTSWithParams(ctx, query, limit, offset, 1.2, 0.75, false)
}

// SearchFTSWithParams performs FTS search with custom BM25 parameters.
// k controls term frequency saturation (default 1.2)
// b controls length normalization (default 0.75)
// conjunctive requires all query terms when true
func (s *Store) SearchFTSWithParams(ctx context.Context, query string, limit, offset int, k, b float64, conjunctive bool) (*SearchResult, error) {
	start := time.Now()

	// Load FTS extension
	if _, err := s.db.ExecContext(ctx, "LOAD fts"); err != nil {
		return nil, fmt.Errorf("loading FTS extension: %w", err)
	}

	// Build query with BM25 parameters
	conjunctiveVal := 0
	if conjunctive {
		conjunctiveVal = 1
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			d.id,
			d.url,
			d.text,
			d.dump,
			d.date,
			d.language,
			d.language_score,
			fts_main_documents.match_bm25(d.id, ?, fields := 'text', k := %.2f, b := %.2f, conjunctive := %d) AS score
		FROM documents d
		WHERE score IS NOT NULL
		ORDER BY score DESC
		LIMIT ? OFFSET ?
	`, k, b, conjunctiveVal)

	rows, err := s.db.QueryContext(ctx, sqlQuery, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("executing FTS search: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		var score sql.NullFloat64
		err := rows.Scan(
			&doc.ID,
			&doc.URL,
			&doc.Text,
			&doc.Dump,
			&doc.Date,
			&doc.Language,
			&doc.LanguageScore,
			&score,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		if score.Valid {
			doc.Score = score.Float64
		}
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "fts",
	}, nil
}

// SearchSimple performs simple LIKE-based search (fallback if FTS not available).
func (s *Store) SearchSimple(ctx context.Context, query string, limit, offset int) ([]Document, error) {
	result, err := s.SearchLike(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	return result.Documents, nil
}

// SearchLike performs LIKE-based search with timing info.
func (s *Store) SearchLike(ctx context.Context, query string, limit, offset int) (*SearchResult, error) {
	start := time.Now()

	// Simple LIKE search
	likePattern := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			url,
			text,
			dump,
			date,
			language,
			language_score
		FROM documents
		WHERE text LIKE ? OR url LIKE ?
		LIMIT ? OFFSET ?
	`, likePattern, likePattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("executing LIKE search: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(
			&doc.ID,
			&doc.URL,
			&doc.Text,
			&doc.Dump,
			&doc.Date,
			&doc.Language,
			&doc.LanguageScore,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		doc.Score = 1.0 // Default score for LIKE search
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "like",
	}, nil
}

// SearchComparison contains results from both search methods for comparison.
type SearchComparison struct {
	Query       string
	FTS         *SearchResult
	LIKE        *SearchResult
	FTSError    error
	LIKEError   error
	Overlap     int     // Number of common documents in results
	OverlapPct  float64 // Percentage overlap (based on smaller result set)
	SpeedupPct  float64 // How much faster FTS is vs LIKE (negative means LIKE is faster)
}

// CompareSearch runs both FTS and LIKE search and compares results.
func (s *Store) CompareSearch(ctx context.Context, query string, limit int) (*SearchComparison, error) {
	comp := &SearchComparison{
		Query: query,
	}

	// Run FTS search
	comp.FTS, comp.FTSError = s.SearchFTS(ctx, query, limit, 0)

	// Run LIKE search
	comp.LIKE, comp.LIKEError = s.SearchLike(ctx, query, limit, 0)

	// Calculate overlap if both succeeded
	if comp.FTSError == nil && comp.LIKEError == nil {
		comp.Overlap = calculateOverlap(comp.FTS.Documents, comp.LIKE.Documents)

		// Calculate overlap percentage based on smaller result set
		minLen := len(comp.FTS.Documents)
		if len(comp.LIKE.Documents) < minLen {
			minLen = len(comp.LIKE.Documents)
		}
		if minLen > 0 {
			comp.OverlapPct = float64(comp.Overlap) / float64(minLen) * 100
		}

		// Calculate speedup percentage
		if comp.LIKE.Duration > 0 {
			comp.SpeedupPct = (float64(comp.LIKE.Duration) - float64(comp.FTS.Duration)) / float64(comp.LIKE.Duration) * 100
		}
	}

	return comp, nil
}

// calculateOverlap counts documents that appear in both result sets.
func calculateOverlap(ftsResults, likeResults []Document) int {
	ftsIDs := make(map[string]bool)
	for _, doc := range ftsResults {
		ftsIDs[doc.ID] = true
	}

	overlap := 0
	for _, doc := range likeResults {
		if ftsIDs[doc.ID] {
			overlap++
		}
	}
	return overlap
}

// Count returns the number of documents.
func (s *Store) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	return count, err
}

// GetImportState returns the import state.
func (s *Store) GetImportState(ctx context.Context) ([]ImportState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT parquet_file, imported_at, row_count
		FROM import_state
		ORDER BY parquet_file
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []ImportState
	for rows.Next() {
		var state ImportState
		if err := rows.Scan(&state.ParquetFile, &state.ImportedAt, &state.RowCount); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
