// Package postgres provides a PostgreSQL full-text search driver for fineweb.
package postgres

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
	fineweb.Register("postgres", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultDSN is the default PostgreSQL connection string.
const DefaultDSN = "postgres://fineweb:fineweb@localhost:5432/fineweb?sslmode=disable"

// DefaultTableName is the default table name.
const DefaultTableName = "documents"

// Driver implements the fineweb.Driver interface using PostgreSQL full-text search.
type Driver struct {
	pool      *pgxpool.Pool
	tableName string
	dsn       string
	language  string
}

// New creates a new PostgreSQL driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dsn := cfg.GetString("dsn", DefaultDSN)
	tableName := cfg.GetString("table", DefaultTableName)
	if cfg.Language != "" {
		tableName = strings.ToLower(strings.ReplaceAll(cfg.Language, "-", "_"))
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	d := &Driver{
		pool:      pool,
		tableName: tableName,
		dsn:       dsn,
		language:  cfg.Language,
	}

	// Ensure table exists with tsvector column
	if err := d.ensureTable(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensuring table: %w", err)
	}

	return d, nil
}

// ensureTable creates the documents table with tsvector and GIN index if it doesn't exist.
func (d *Driver) ensureTable(ctx context.Context) error {
	// Create table with generated tsvector column
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			url TEXT,
			content TEXT,
			dump TEXT,
			date TEXT,
			language TEXT,
			language_score FLOAT8,
			content_tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', content)) STORED
		)
	`, d.tableName)

	_, err := d.pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	// Create GIN index on tsvector column
	createIndexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_content_tsv_idx ON %s USING GIN (content_tsv)
	`, d.tableName, d.tableName)

	_, err = d.pool.Exec(ctx, createIndexSQL)
	if err != nil {
		return fmt.Errorf("creating index: %w", err)
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "postgres"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "postgres",
		Description: "PostgreSQL 18 with tsvector, ts_query, and GIN index",
		Features:    []string{"full-text-search", "tsvector", "gin-index", "ts_rank", "copy-protocol"},
		External:    true,
	}
}

// Search performs full-text search using ts_query and ts_rank.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Prepare the query - convert to tsquery format
	// Split query into words and join with &
	words := strings.Fields(query)
	if len(words) == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "postgres",
			Total:     0,
		}, nil
	}

	tsQueryTerms := make([]string, len(words))
	for i, word := range words {
		// Escape single quotes and create prefix match
		escaped := strings.ReplaceAll(word, "'", "''")
		tsQueryTerms[i] = escaped + ":*"
	}
	tsQuery := strings.Join(tsQueryTerms, " & ")

	// Search query with ts_rank for scoring
	searchSQL := fmt.Sprintf(`
		SELECT id, url, content, dump, date, language, language_score,
		       ts_rank(content_tsv, to_tsquery('simple', $1)) as score
		FROM %s
		WHERE content_tsv @@ to_tsquery('simple', $1)
		ORDER BY score DESC
		LIMIT $2 OFFSET $3
	`, d.tableName)

	rows, err := d.pool.Query(ctx, searchSQL, tsQuery, limit, offset)
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

	// Get total count (optional, can be slow for large result sets)
	var total int64
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s WHERE content_tsv @@ to_tsquery('simple', $1)
	`, d.tableName)
	_ = d.pool.QueryRow(ctx, countSQL, tsQuery).Scan(&total)

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "postgres",
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
			doc.Text, // Map Text to content column
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

	return fmt.Errorf("postgres not ready after %v", timeout)
}

// IsServiceAvailable checks if PostgreSQL is reachable.
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
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
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
