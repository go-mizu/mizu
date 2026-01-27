// Package duckdb provides a DuckDB-based driver for fineweb full-text search.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("duckdb", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using DuckDB.
type Driver struct {
	db       *sql.DB
	dbPath   string
	dataDir  string
	language string
	ftsReady bool
}

// New creates a new DuckDB driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	// Use language in db filename if provided
	dbName := "fineweb.duckdb"
	if cfg.Language != "" {
		dbName = cfg.Language + ".duckdb"
	}
	dbPath := filepath.Join(dataDir, dbName)

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	d := &Driver{
		db:       db,
		dbPath:   dbPath,
		dataDir:  dataDir,
		language: cfg.Language,
	}

	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	// Check if FTS index exists
	d.ftsReady = d.hasFTSIndex(context.Background())

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "duckdb"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "duckdb",
		Description: "DuckDB with FTS extension for BM25 full-text search",
		Features:    []string{"bm25", "fts", "sql", "parquet-native"},
		External:    false,
	}
}

func (d *Driver) initSchema() error {
	_, err := d.db.Exec(`
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

	_, err = d.db.Exec(`
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

// Search performs full-text search.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	if d.ftsReady {
		return d.searchFTS(ctx, query, limit, offset)
	}
	return d.searchLike(ctx, query, limit, offset)
}

func (d *Driver) searchFTS(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	if _, err := d.db.ExecContext(ctx, "LOAD fts"); err != nil {
		return nil, fmt.Errorf("loading FTS extension: %w", err)
	}

	sqlQuery := `
		SELECT
			d.id,
			d.url,
			d.text,
			d.dump,
			d.date,
			d.language,
			d.language_score,
			fts_main_documents.match_bm25(d.id, ?, fields := 'text') AS score
		FROM documents d
		WHERE score IS NOT NULL
		ORDER BY score DESC
		LIMIT ? OFFSET ?
	`

	rows, err := d.db.QueryContext(ctx, sqlQuery, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("executing FTS search: %w", err)
	}
	defer rows.Close()

	var docs []fineweb.Document
	for rows.Next() {
		var doc fineweb.Document
		var score sql.NullFloat64
		err := rows.Scan(
			&doc.ID, &doc.URL, &doc.Text, &doc.Dump,
			&doc.Date, &doc.Language, &doc.LanguageScore, &score,
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

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "duckdb-fts",
	}, nil
}

func (d *Driver) searchLike(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	likePattern := "%" + query + "%"
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, url, text, dump, date, language, language_score
		FROM documents
		WHERE text LIKE ? OR url LIKE ?
		LIMIT ? OFFSET ?
	`, likePattern, likePattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("executing LIKE search: %w", err)
	}
	defer rows.Close()

	var docs []fineweb.Document
	for rows.Next() {
		var doc fineweb.Document
		err := rows.Scan(
			&doc.ID, &doc.URL, &doc.Text, &doc.Dump,
			&doc.Date, &doc.Language, &doc.LanguageScore,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		doc.Score = 1.0 // Default score for LIKE
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "duckdb-like",
	}, nil
}

// Import ingests documents from an iterator.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO documents (id, url, text, dump, date, language, language_score)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	var imported int64
	batchSize := 10000
	count := 0

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err = stmt.ExecContext(ctx, doc.ID, doc.URL, doc.Text, doc.Dump, doc.Date, doc.Language, doc.LanguageScore)
		if err != nil {
			return fmt.Errorf("inserting document: %w", err)
		}

		imported++
		count++

		if count >= batchSize {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("committing batch: %w", err)
			}

			if progress != nil {
				progress(imported, 0)
			}

			tx, err = d.db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("beginning new transaction: %w", err)
			}
			stmt, err = tx.PrepareContext(ctx, `
				INSERT INTO documents (id, url, text, dump, date, language, language_score)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT (id) DO NOTHING
			`)
			if err != nil {
				return fmt.Errorf("preparing statement: %w", err)
			}
			count = 0
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing final batch: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// ImportParquetDirect imports parquet files directly using DuckDB's native reader.
func (d *Driver) ImportParquetDirect(ctx context.Context, parquetPath string) (int64, error) {
	query := fmt.Sprintf(`
		INSERT INTO documents (id, url, text, dump, date, language, language_score)
		SELECT id, url, text, dump, date, language, language_score
		FROM read_parquet('%s')
		ON CONFLICT (id) DO NOTHING
	`, parquetPath)

	result, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

// CreateFTSIndex creates the full-text search index.
func (d *Driver) CreateFTSIndex(ctx context.Context, cfg FTSConfig) error {
	// Configure DuckDB
	if cfg.MemoryLimit != "" {
		d.db.ExecContext(ctx, fmt.Sprintf("SET memory_limit = '%s'", cfg.MemoryLimit))
	}
	if cfg.Threads > 0 {
		d.db.ExecContext(ctx, fmt.Sprintf("SET threads = %d", cfg.Threads))
	}
	if !cfg.PreserveInsertionOrder {
		d.db.ExecContext(ctx, "SET preserve_insertion_order = false")
	}

	// Install and load FTS
	if _, err := d.db.ExecContext(ctx, "INSTALL fts"); err != nil {
		if !strings.Contains(err.Error(), "already installed") {
			return fmt.Errorf("installing FTS: %w", err)
		}
	}

	if _, err := d.db.ExecContext(ctx, "LOAD fts"); err != nil {
		return fmt.Errorf("loading FTS: %w", err)
	}

	stripAccents := 0
	if cfg.StripAccents {
		stripAccents = 1
	}
	lower := 0
	if cfg.Lower {
		lower = 1
	}

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

	if _, err := d.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("creating FTS index: %w", err)
	}

	d.ftsReady = true
	return nil
}

// DropFTSIndex removes the FTS index.
func (d *Driver) DropFTSIndex(ctx context.Context) error {
	d.db.ExecContext(ctx, "LOAD fts")
	_, err := d.db.ExecContext(ctx, "PRAGMA drop_fts_index('documents')")
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return fmt.Errorf("dropping FTS index: %w", err)
	}
	d.ftsReady = false
	return nil
}

func (d *Driver) hasFTSIndex(ctx context.Context) bool {
	if _, err := d.db.ExecContext(ctx, "LOAD fts"); err != nil {
		return false
	}

	var count int
	err := d.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.schemata
		WHERE schema_name = 'fts_main_documents'
	`).Scan(&count)
	if err != nil || count == 0 {
		return false
	}

	// Verify it actually works
	var score sql.NullFloat64
	err = d.db.QueryRowContext(ctx, `
		SELECT fts_main_documents.match_bm25(id, 'test', fields := 'text') AS score
		FROM documents
		LIMIT 1
	`).Scan(&score)

	return err == nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	return count, err
}

// Close closes the database connection.
func (d *Driver) Close() error {
	return d.db.Close()
}

// FTSConfig contains configuration for FTS index creation.
type FTSConfig struct {
	Stemmer                string
	Stopwords              string
	StripAccents           bool
	Lower                  bool
	MemoryLimit            string
	Threads                int
	PreserveInsertionOrder bool
}

// DefaultFTSConfig returns sensible defaults.
func DefaultFTSConfig() FTSConfig {
	return FTSConfig{
		Stemmer:                "porter",
		Stopwords:              "english",
		StripAccents:           true,
		Lower:                  true,
		MemoryLimit:            "4GB",
		Threads:                4,
		PreserveInsertionOrder: false,
	}
}

// VietnameseFTSConfig returns config optimized for Vietnamese.
func VietnameseFTSConfig() FTSConfig {
	return FTSConfig{
		Stemmer:                "none", // No Vietnamese stemmer
		Stopwords:              "none", // No Vietnamese stopwords
		StripAccents:           false,  // Keep diacritics
		Lower:                  true,
		MemoryLimit:            "8GB",
		Threads:                4,
		PreserveInsertionOrder: false,
	}
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
