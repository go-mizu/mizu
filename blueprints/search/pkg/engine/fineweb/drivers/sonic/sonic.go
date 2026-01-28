// Package sonic provides a Sonic-based driver for fineweb full-text search.
// Sonic is a search index only - it stores document IDs, not full documents.
// Documents are stored in a local SQLite database.
package sonic

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/expectedsh/go-sonic/sonic"
	_ "modernc.org/sqlite"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("sonic", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default Sonic host.
const DefaultHost = "localhost"

// DefaultPort is the default Sonic port.
const DefaultPort = 1491

// DefaultPassword is the default Sonic password.
const DefaultPassword = "fineweb"

// DefaultCollection is the default Sonic collection.
const DefaultCollection = "fineweb"

// DefaultBucket is the default Sonic bucket.
const DefaultBucket = "documents"

// Driver implements the fineweb.Driver interface using Sonic for search
// and SQLite for document storage.
type Driver struct {
	// Sonic connections (one per mode)
	ingester sonic.Ingestable
	searcher sonic.Searchable

	// SQLite for document storage
	db     *sql.DB
	dbPath string

	// Configuration
	host       string
	port       int
	password   string
	collection string
	bucket     string
	dataDir    string
	language   string
}

// New creates a new Sonic driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	port := cfg.GetInt("port", DefaultPort)
	password := cfg.GetString("password", DefaultPassword)
	collection := cfg.GetString("collection", DefaultCollection)
	bucket := cfg.GetString("bucket", DefaultBucket)

	// Parse host:port format if provided
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				port = p
			}
		}
	}

	// Use language-specific bucket if provided
	if cfg.Language != "" {
		bucket = cfg.Language
	}

	dataDir := cfg.DataDir
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "data", "blueprints", "search", "fineweb-2")
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	// Create SQLite database for document storage
	dbName := "fineweb.sonicdb"
	if cfg.Language != "" {
		dbName = cfg.Language + ".sonicdb"
	}
	dbPath := filepath.Join(dataDir, dbName)

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}

	d := &Driver{
		db:         db,
		dbPath:     dbPath,
		host:       host,
		port:       port,
		password:   password,
		collection: collection,
		bucket:     bucket,
		dataDir:    dataDir,
		language:   cfg.Language,
	}

	// Initialize SQLite schema for document storage
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	// Connect to Sonic for ingest operations
	ingester, err := sonic.NewIngester(host, port, password)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to sonic ingest: %w", err)
	}
	d.ingester = ingester

	// Connect to Sonic for search operations
	searcher, err := sonic.NewSearch(host, port, password)
	if err != nil {
		ingester.Quit()
		db.Close()
		return nil, fmt.Errorf("connecting to sonic search: %w", err)
	}
	d.searcher = searcher

	return d, nil
}

func (d *Driver) initSchema() error {
	// Create documents table for full document storage
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

	// Create index for faster lookups
	_, err = d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_documents_id ON documents(id)`)
	if err != nil {
		return fmt.Errorf("creating id index: %w", err)
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "sonic"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "sonic",
		Description: "Sonic fast search backend with SQLite document store",
		Features:    []string{"fast-indexing", "low-memory", "phonetic-search", "typo-tolerance"},
		External:    true,
	}
}

// Search performs full-text search using Sonic and fetches documents from SQLite.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Query Sonic to get matching document IDs
	// Sonic QUERY returns IDs of matching documents
	// Note: Sonic supports both limit and offset natively
	ids, err := d.searcher.Query(d.collection, d.bucket, query, limit, offset, sonic.LangAutoDetect)
	if err != nil {
		return nil, fmt.Errorf("sonic query failed: %w", err)
	}

	if len(ids) == 0 {
		return &fineweb.SearchResult{
			Documents: []fineweb.Document{},
			Duration:  time.Since(start),
			Method:    "sonic",
			Total:     0,
		}, nil
	}

	// Fetch full documents from SQLite
	docs, err := d.fetchDocuments(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetching documents: %w", err)
	}

	// Assign scores based on position (Sonic returns results in relevance order)
	for i := range docs {
		docs[i].Score = 1.0 / float64(i+1)
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "sonic",
		Total:     int64(len(docs)),
	}, nil
}

// fetchDocuments retrieves full documents from SQLite by their IDs.
func (d *Driver) fetchDocuments(ctx context.Context, ids []string) ([]fineweb.Document, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build query with placeholders
	query := "SELECT id, url, text, dump, date, language, language_score FROM documents WHERE id IN ("
	args := make([]any, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ")"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying documents: %w", err)
	}
	defer rows.Close()

	// Create a map to preserve order
	docMap := make(map[string]fineweb.Document, len(ids))
	for rows.Next() {
		var doc fineweb.Document
		err := rows.Scan(&doc.ID, &doc.URL, &doc.Text, &doc.Dump, &doc.Date, &doc.Language, &doc.LanguageScore)
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		docMap[doc.ID] = doc
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Return documents in the same order as IDs (preserves Sonic ranking)
	docs := make([]fineweb.Document, 0, len(ids))
	for _, id := range ids {
		if doc, ok := docMap[id]; ok {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

// Import ingests documents from an iterator.
// Documents are stored in SQLite and their text is pushed to Sonic for indexing.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 1000
	var imported int64

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO documents (id, url, text, dump, date, language, language_score)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

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

		// Store document in SQLite
		_, err = stmt.ExecContext(ctx, doc.ID, doc.URL, doc.Text, doc.Dump, doc.Date, doc.Language, doc.LanguageScore)
		if err != nil {
			return fmt.Errorf("inserting document: %w", err)
		}

		// Push document text to Sonic for indexing
		// PUSH collection bucket object "text content" [LANG]
		err = d.ingester.Push(d.collection, d.bucket, doc.ID, doc.Text, sonic.LangAutoDetect)
		if err != nil {
			return fmt.Errorf("pushing to sonic: %w", err)
		}

		imported++
		count++

		if count >= batchSize {
			// Commit SQLite batch
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("committing batch: %w", err)
			}

			// Flush Sonic collection to ensure data is persisted
			if err := d.ingester.FlushCollection(d.collection); err != nil {
				// Non-fatal, just warn
				fmt.Printf("Warning: sonic flush failed: %v\n", err)
			}

			if progress != nil {
				progress(imported, 0)
			}

			// Start new transaction
			tx, err = d.db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("beginning new transaction: %w", err)
			}
			stmt, err = tx.PrepareContext(ctx, `
				INSERT OR REPLACE INTO documents (id, url, text, dump, date, language, language_score)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`)
			if err != nil {
				return fmt.Errorf("preparing statement: %w", err)
			}
			count = 0
		}
	}

	// Commit final batch
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing final batch: %w", err)
	}

	// Final flush to Sonic
	if err := d.ingester.FlushCollection(d.collection); err != nil {
		fmt.Printf("Warning: final sonic flush failed: %v\n", err)
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	var count int64
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	return count, err
}

// Close releases all resources.
func (d *Driver) Close() error {
	var errs []error

	if d.ingester != nil {
		if err := d.ingester.Quit(); err != nil {
			errs = append(errs, fmt.Errorf("closing ingest connection: %w", err))
		}
	}

	if d.searcher != nil {
		if err := d.searcher.Quit(); err != nil {
			errs = append(errs, fmt.Errorf("closing search connection: %w", err))
		}
	}

	if d.db != nil {
		if err := d.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing sqlite: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}
	return nil
}

// FlushIndex flushes pending writes to the Sonic index.
func (d *Driver) FlushIndex() error {
	return d.ingester.FlushCollection(d.collection)
}

// ClearBucket clears all data from the Sonic index for this bucket.
func (d *Driver) ClearBucket() error {
	return d.ingester.FlushBucket(d.collection, d.bucket)
}

// WaitForService waits for Sonic to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	password := DefaultPassword
	port := DefaultPort

	// Parse host:port format if provided
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				port = p
			}
		}
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to establish a search connection
		conn, err := sonic.NewSearch(host, port, password)
		if err == nil {
			conn.Quit()
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("sonic not ready after %v", timeout)
}

// IsServiceAvailable checks if Sonic is reachable.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}
	port := DefaultPort

	// Parse host:port format if provided
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil {
				port = p
			}
		}
	}

	conn, err := sonic.NewSearch(host, port, DefaultPassword)
	if err != nil {
		return false
	}
	conn.Quit()
	return true
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if host := os.Getenv("SONIC_HOST"); host != "" {
		cfg.Options["host"] = host
	}
	if portStr := os.Getenv("SONIC_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Options["port"] = port
		}
	}
	if password := os.Getenv("SONIC_PASSWORD"); password != "" {
		cfg.Options["password"] = password
	}
	return New(cfg)
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
