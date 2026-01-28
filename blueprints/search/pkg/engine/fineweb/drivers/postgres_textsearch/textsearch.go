// Package postgres_textsearch provides a PostgreSQL driver using Tiger Data's pg_textsearch extension.
// pg_textsearch implements BM25 full-text search with a memtable architecture.
package postgres_textsearch

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("postgres_textsearch", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultDSN is the default PostgreSQL connection string for pg_textsearch.
// Note: pg_textsearch requires building from source; use port 5437 by convention.
const DefaultDSN = "postgres://fineweb:fineweb@localhost:5437/fineweb?sslmode=disable"

// DefaultTableName is the default table name.
const DefaultTableName = "documents"

// Driver implements the fineweb.Driver interface using Tiger Data pg_textsearch.
type Driver struct {
	pool       *pgxpool.Pool
	tableName  string
	dsn        string
	language   string
	indexName  string
	textConfig string
}

// New creates a new Tiger Data pg_textsearch driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dsn := cfg.GetString("dsn", DefaultDSN)
	tableName := cfg.GetString("table", DefaultTableName)
	textConfig := cfg.GetString("text_config", "simple")
	if cfg.Language != "" {
		tableName = strings.ToLower(strings.ReplaceAll(cfg.Language, "-", "_"))
		// Map language to PostgreSQL text search config
		switch cfg.Language {
		case "english", "en":
			textConfig = "english"
		case "german", "de":
			textConfig = "german"
		case "french", "fr":
			textConfig = "french"
		case "spanish", "es":
			textConfig = "spanish"
		default:
			textConfig = "simple"
		}
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	d := &Driver{
		pool:       pool,
		tableName:  tableName,
		dsn:        dsn,
		language:   cfg.Language,
		indexName:  tableName + "_bm25_idx",
		textConfig: textConfig,
	}

	// Ensure extension and table exist
	if err := d.ensureExtension(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensuring extension: %w", err)
	}

	if err := d.ensureTable(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensuring table: %w", err)
	}

	return d, nil
}

// ensureExtension creates the pg_textsearch extension if it doesn't exist.
func (d *Driver) ensureExtension(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_textsearch`)
	if err != nil {
		return fmt.Errorf("creating pg_textsearch extension: %w", err)
	}
	return nil
}

// ensureTable creates the documents table if it doesn't exist.
func (d *Driver) ensureTable(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			url TEXT,
			content TEXT,
			dump TEXT,
			date TEXT,
			language TEXT,
			language_score FLOAT8
		)
	`, d.tableName)

	_, err := d.pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	return nil
}

// ensureBM25Index creates the BM25 index after data is loaded.
func (d *Driver) ensureBM25Index(ctx context.Context) error {
	// Drop existing index if any
	dropIndexSQL := fmt.Sprintf(`DROP INDEX IF EXISTS %s`, d.indexName)
	_, _ = d.pool.Exec(ctx, dropIndexSQL)

	// Create BM25 index using pg_textsearch syntax
	// Note: pg_textsearch uses text_config parameter for tokenization
	createIndexSQL := fmt.Sprintf(`
		CREATE INDEX %s ON %s
		USING bm25 (content)
		WITH (text_config = '%s')
	`, d.indexName, d.tableName, d.textConfig)

	_, err := d.pool.Exec(ctx, createIndexSQL)
	if err != nil {
		return fmt.Errorf("creating BM25 index: %w", err)
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "postgres_textsearch"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "postgres_textsearch",
		Description: "Tiger Data pg_textsearch with BM25 scoring and memtable architecture",
		Features:    []string{"bm25", "full-text-search", "memtable", "block-max-wand"},
		External:    true,
	}
}

// Search performs full-text search using BM25 with <@> operator.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	if strings.TrimSpace(query) == "" {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "postgres_textsearch",
			Total:     0,
		}, nil
	}

	// Escape single quotes in query
	escapedQuery := strings.ReplaceAll(query, "'", "''")

	// Search query using pg_textsearch <@> operator
	// Note: pg_textsearch returns negative scores (lower = better match)
	// We negate to get positive scores for consistency
	searchSQL := fmt.Sprintf(`
		SELECT id, url, content, dump, date, language, language_score,
		       -(content <@> '%s') as score
		FROM %s
		ORDER BY content <@> '%s'
		LIMIT $1 OFFSET $2
	`, escapedQuery, d.tableName, escapedQuery)

	rows, err := d.pool.Query(ctx, searchSQL, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}
	defer rows.Close()

	docs := make([]fineweb.Document, 0)
	for rows.Next() {
		var doc fineweb.Document
		var content string
		err := rows.Scan(
			&doc.ID,
			&doc.URL,
			&content,
			&doc.Dump,
			&doc.Date,
			&doc.Language,
			&doc.LanguageScore,
			&doc.Score,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		doc.Text = content
		docs = append(docs, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	// Get total count (note: expensive for large result sets)
	var total int64
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, d.tableName)
	_ = d.pool.QueryRow(ctx, countSQL).Scan(&total)

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "postgres_textsearch",
		Total:     total,
	}, nil
}

// Import ingests documents from an iterator using COPY protocol for fast bulk loading.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 10000
	batch := make([][]any, 0, batchSize)
	var imported int64

	columns := []string{"id", "url", "content", "dump", "date", "language", "language_score"}

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		row := []any{
			doc.ID,
			doc.URL,
			doc.Text,
			doc.Dump,
			doc.Date,
			doc.Language,
			doc.LanguageScore,
		}
		batch = append(batch, row)

		if len(batch) >= batchSize {
			if err := d.copyBatch(ctx, columns, batch); err != nil {
				return fmt.Errorf("copying batch: %w", err)
			}
			imported += int64(len(batch))
			batch = batch[:0]

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Copy remaining documents
	if len(batch) > 0 {
		if err := d.copyBatch(ctx, columns, batch); err != nil {
			return fmt.Errorf("copying final batch: %w", err)
		}
		imported += int64(len(batch))
	}

	// Create BM25 index after loading data
	if err := d.ensureBM25Index(ctx); err != nil {
		return fmt.Errorf("creating BM25 index: %w", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// copyBatch uses COPY protocol to insert rows efficiently.
func (d *Driver) copyBatch(ctx context.Context, columns []string, rows [][]any) error {
	_, err := d.pool.CopyFrom(
		ctx,
		pgx.Identifier{d.tableName},
		columns,
		pgx.CopyFromRows(rows),
	)
	return err
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	var count int64
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, d.tableName)
	err := d.pool.QueryRow(ctx, countSQL).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting documents: %w", err)
	}
	return count, nil
}

// DeleteIndex drops the table and index for fresh benchmark runs.
func (d *Driver) DeleteIndex(ctx context.Context) error {
	dropSQL := fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE`, d.tableName)
	_, err := d.pool.Exec(ctx, dropSQL)
	if err != nil {
		return fmt.Errorf("dropping table: %w", err)
	}

	// Recreate empty table
	return d.ensureTable(ctx)
}

// Close releases the connection pool.
func (d *Driver) Close() error {
	d.pool.Close()
	return nil
}

// WaitForService waits for PostgreSQL to be ready.
func WaitForService(ctx context.Context, dsn string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pool, err := pgxpool.New(ctx, dsn)
		if err == nil {
			pool.Close()
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("postgres (pg_textsearch) not ready after %v", timeout)
}

// IsServiceAvailable checks if pg_textsearch PostgreSQL is reachable.
func IsServiceAvailable(dsn string) bool {
	if dsn == "" {
		dsn = DefaultDSN
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return false
	}
	defer pool.Close()

	return pool.Ping(ctx) == nil
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if dsn := os.Getenv("PG_TEXTSEARCH_DSN"); dsn != "" {
		cfg.Options["dsn"] = dsn
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
