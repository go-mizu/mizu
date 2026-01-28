// Package sqlite provides a SQLite FTS5-based driver for fineweb full-text search.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("sqlite", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// Driver implements the fineweb.Driver interface using SQLite FTS5.
type Driver struct {
	db       *sql.DB
	dbPath   string
	dataDir  string
	language string
}

// New creates a new SQLite FTS5 driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	dbName := "fineweb.sqlite"
	if cfg.Language != "" {
		dbName = cfg.Language + ".sqlite"
	}
	dbPath := filepath.Join(dataDir, dbName)

	// Use WAL mode for better concurrent read/write performance
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
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

	return d, nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "sqlite"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "sqlite",
		Description: "SQLite FTS5 with BM25 ranking",
		Features:    []string{"bm25", "fts5", "embedded", "lightweight"},
		External:    false,
	}
}

func (d *Driver) initSchema() error {
	// Create main documents table
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			url TEXT,
			text TEXT,
			dump TEXT,
			date TEXT,
			language TEXT,
			language_score REAL
		)
	`)
	if err != nil {
		return fmt.Errorf("creating documents table: %w", err)
	}

	// Create FTS5 virtual table for full-text search
	// Using content= to create an external content table (saves space)
	_, err = d.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			id,
			url,
			text,
			content='documents',
			content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 0'
		)
	`)
	if err != nil {
		return fmt.Errorf("creating FTS5 table: %w", err)
	}

	// Create triggers to keep FTS in sync with documents table
	_, err = d.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO documents_fts(rowid, id, url, text) VALUES (NEW.rowid, NEW.id, NEW.url, NEW.text);
		END
	`)
	if err != nil {
		return fmt.Errorf("creating insert trigger: %w", err)
	}

	_, err = d.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, id, url, text) VALUES ('delete', OLD.rowid, OLD.id, OLD.url, OLD.text);
		END
	`)
	if err != nil {
		return fmt.Errorf("creating delete trigger: %w", err)
	}

	_, err = d.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, id, url, text) VALUES ('delete', OLD.rowid, OLD.id, OLD.url, OLD.text);
			INSERT INTO documents_fts(rowid, id, url, text) VALUES (NEW.rowid, NEW.id, NEW.url, NEW.text);
		END
	`)
	if err != nil {
		return fmt.Errorf("creating update trigger: %w", err)
	}

	return nil
}

// Search performs full-text search using FTS5 BM25.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// FTS5 query with BM25 ranking
	// bm25(documents_fts) returns negative scores, so we negate for proper ordering
	rows, err := d.db.QueryContext(ctx, `
		SELECT
			d.id,
			d.url,
			d.text,
			d.dump,
			d.date,
			d.language,
			d.language_score,
			-bm25(documents_fts, 0, 0, 1) AS score
		FROM documents_fts
		JOIN documents d ON d.rowid = documents_fts.rowid
		WHERE documents_fts MATCH ?
		ORDER BY score DESC
		LIMIT ? OFFSET ?
	`, query, limit, offset)
	if err != nil {
		// Return error instead of falling back to slow LIKE search
		return nil, fmt.Errorf("FTS5 search failed (index may need rebuild): %w", err)
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
		Method:    "sqlite-fts5",
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
		INSERT OR IGNORE INTO documents (id, url, text, dump, date, language, language_score)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}

	var imported int64
	batchSize := 10000
	count := 0

	for doc, err := range docs {
		if err != nil {
			stmt.Close()
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			stmt.Close()
			return ctx.Err()
		default:
		}

		_, err = stmt.ExecContext(ctx, doc.ID, doc.URL, doc.Text, doc.Dump, doc.Date, doc.Language, doc.LanguageScore)
		if err != nil {
			stmt.Close()
			return fmt.Errorf("inserting document: %w", err)
		}

		imported++
		count++

		if count >= batchSize {
			stmt.Close() // Close old statement before commit
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
				INSERT OR IGNORE INTO documents (id, url, text, dump, date, language, language_score)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`)
			if err != nil {
				return fmt.Errorf("preparing statement: %w", err)
			}
			count = 0
		}
	}
	stmt.Close() // Close final statement

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing final batch: %w", err)
	}

	// Optimize FTS index for faster searches
	if err := d.OptimizeFTS(ctx); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: FTS optimization failed: %v\n", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// RebuildFTS rebuilds the FTS index from the documents table.
// Call this after bulk inserts without triggers or if the index is corrupted.
func (d *Driver) RebuildFTS(ctx context.Context) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO documents_fts(documents_fts) VALUES('rebuild')`)
	return err
}

// OptimizeFTS optimizes the FTS index for faster searches.
func (d *Driver) OptimizeFTS(ctx context.Context) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO documents_fts(documents_fts) VALUES('optimize')`)
	return err
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

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
