// Package manticore provides a Manticore Search-based driver for fineweb full-text search.
package manticore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/go-mizu/mizu/blueprints/search/pkg/engine/fineweb"
)

func init() {
	fineweb.Register("manticore", func(cfg fineweb.DriverConfig) (fineweb.Driver, error) {
		return New(cfg)
	})
}

// DefaultHost is the default Manticore MySQL protocol address.
const DefaultHost = "localhost:9306"

// DefaultHTTPHost is the default Manticore HTTP API address.
const DefaultHTTPHost = "http://localhost:9308"

// DefaultTableName is the default table name.
const DefaultTableName = "fineweb"

// Driver implements the fineweb.Driver interface using Manticore Search.
type Driver struct {
	db        *sql.DB
	httpHost  string
	tableName string
	language  string
}

// ManticoreDocument is the document structure for Manticore.
type ManticoreDocument struct {
	ID            int64   `json:"id"`
	URL           string  `json:"url"`
	Content       string  `json:"content"`
	Dump          string  `json:"dump"`
	Date          string  `json:"date"`
	Language      string  `json:"language"`
	LanguageScore float64 `json:"language_score"`
}

// New creates a new Manticore Search driver.
func New(cfg fineweb.DriverConfig) (*Driver, error) {
	host := cfg.GetString("host", DefaultHost)
	httpHost := cfg.GetString("http_host", DefaultHTTPHost)

	tableName := cfg.GetString("table", DefaultTableName)
	if cfg.Language != "" {
		// Replace non-alphanumeric characters for valid table name
		tableName = strings.ReplaceAll(cfg.Language, "-", "_")
		tableName = strings.ReplaceAll(tableName, ".", "_")
	}

	// Connect via MySQL protocol
	// DSN format: [user[:password]@][net[(addr)]]/dbname
	dsn := fmt.Sprintf("tcp(%s)/", host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to manticore: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging manticore: %w", err)
	}

	d := &Driver{
		db:        db,
		httpHost:  httpHost,
		tableName: tableName,
		language:  cfg.Language,
	}

	// Ensure table exists
	if err := d.ensureTable(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ensuring table: %w", err)
	}

	return d, nil
}

func (d *Driver) ensureTable() error {
	// Create real-time table with full-text index on content
	// Manticore SQL syntax for RT tables:
	// CREATE TABLE tablename (field_name field_type [options], ...) [table_options]
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT,
			url TEXT,
			content TEXT,
			dump TEXT,
			date STRING,
			language STRING,
			language_score FLOAT
		) morphology='stem_en' min_word_len='2'
	`, d.tableName)

	_, err := d.db.Exec(query)
	if err != nil {
		// Table might already exist with different schema, ignore error
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("creating table: %w", err)
		}
	}

	return nil
}

// Name returns the driver name.
func (d *Driver) Name() string {
	return "manticore"
}

// Info returns driver metadata.
func (d *Driver) Info() *fineweb.DriverInfo {
	return &fineweb.DriverInfo{
		Name:        "manticore",
		Description: "Manticore Search with BM25 ranking and real-time indexing",
		Features:    []string{"bm25", "real-time", "mysql-protocol", "http-api", "full-text-search"},
		External:    true,
	}
}

// Search performs full-text search using MATCH() and WEIGHT() for BM25 scoring.
func (d *Driver) Search(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Escape single quotes in query
	escapedQuery := strings.ReplaceAll(query, "'", "\\'")

	// Use MATCH() for full-text search with WEIGHT() for BM25 scoring
	// OPTION ranker=bm25 uses BM25 ranking algorithm
	sqlQuery := fmt.Sprintf(`
		SELECT id, url, content, dump, date, language, language_score, WEIGHT() as score
		FROM %s
		WHERE MATCH('%s')
		ORDER BY score DESC
		LIMIT %d, %d
		OPTION ranker=bm25
	`, d.tableName, escapedQuery, offset, limit)

	rows, err := d.db.QueryContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("executing search: %w", err)
	}
	defer rows.Close()

	var docs []fineweb.Document
	for rows.Next() {
		var (
			id            int64
			url           sql.NullString
			content       sql.NullString
			dump          sql.NullString
			date          sql.NullString
			language      sql.NullString
			languageScore sql.NullFloat64
			score         float64
		)

		if err := rows.Scan(&id, &url, &content, &dump, &date, &language, &languageScore, &score); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		docs = append(docs, fineweb.Document{
			ID:            strconv.FormatInt(id, 10),
			URL:           url.String,
			Text:          content.String,
			Dump:          dump.String,
			Date:          date.String,
			Language:      language.String,
			LanguageScore: languageScore.Float64,
			Score:         score,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	// Get total count using meta
	var total int64
	metaQuery := "SHOW META LIKE 'total_found'"
	metaRows, err := d.db.QueryContext(ctx, metaQuery)
	if err == nil {
		defer metaRows.Close()
		for metaRows.Next() {
			var name string
			var value int64
			if metaRows.Scan(&name, &value) == nil {
				total = value
			}
		}
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "manticore",
		Total:     total,
	}, nil
}

// Import ingests documents from an iterator using bulk INSERT.
func (d *Driver) Import(ctx context.Context, docs iter.Seq2[fineweb.Document, error], progress fineweb.ProgressFunc) error {
	batchSize := 1000
	batch := make([]ManticoreDocument, 0, batchSize)
	var imported int64
	var idCounter int64

	for doc, err := range docs {
		if err != nil {
			return fmt.Errorf("reading document: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Generate numeric ID from string ID (hash or sequential)
		idCounter++
		numericID := idCounter
		if doc.ID != "" {
			// Try to parse as number, otherwise use counter
			if parsed, err := strconv.ParseInt(doc.ID, 10, 64); err == nil {
				numericID = parsed
			}
		}

		batch = append(batch, ManticoreDocument{
			ID:            numericID,
			URL:           doc.URL,
			Content:       doc.Text,
			Dump:          doc.Dump,
			Date:          doc.Date,
			Language:      doc.Language,
			LanguageScore: doc.LanguageScore,
		})

		if len(batch) >= batchSize {
			if err := d.bulkInsert(ctx, batch); err != nil {
				return fmt.Errorf("bulk insert: %w", err)
			}

			imported += int64(len(batch))
			batch = batch[:0]

			if progress != nil {
				progress(imported, 0)
			}
		}
	}

	// Insert remaining documents
	if len(batch) > 0 {
		if err := d.bulkInsert(ctx, batch); err != nil {
			return fmt.Errorf("bulk insert final batch: %w", err)
		}
		imported += int64(len(batch))
	}

	if progress != nil {
		progress(imported, imported)
	}

	return nil
}

func (d *Driver) bulkInsert(ctx context.Context, docs []ManticoreDocument) error {
	if len(docs) == 0 {
		return nil
	}

	// Build bulk INSERT statement with direct value interpolation
	// Manticore doesn't support prepared statements, so we escape values manually
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO %s (id, url, content, dump, date, language, language_score) VALUES ", d.tableName))

	for i, doc := range docs {
		if i > 0 {
			sb.WriteString(", ")
		}
		// Escape single quotes in string values
		url := strings.ReplaceAll(doc.URL, "'", "\\'")
		content := strings.ReplaceAll(doc.Content, "'", "\\'")
		dump := strings.ReplaceAll(doc.Dump, "'", "\\'")
		date := strings.ReplaceAll(doc.Date, "'", "\\'")
		language := strings.ReplaceAll(doc.Language, "'", "\\'")

		sb.WriteString(fmt.Sprintf("(%d, '%s', '%s', '%s', '%s', '%s', %f)",
			doc.ID, url, content, dump, date, language, doc.LanguageScore))
	}

	_, err := d.db.ExecContext(ctx, sb.String())
	if err != nil {
		return fmt.Errorf("executing bulk insert: %w", err)
	}

	return nil
}

// Count returns the number of indexed documents.
func (d *Driver) Count(ctx context.Context) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", d.tableName)
	var count int64
	err := d.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting documents: %w", err)
	}
	return count, nil
}

// Close releases the database connection.
func (d *Driver) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// WaitForService waits for Manticore to be ready.
func WaitForService(ctx context.Context, host string, timeout time.Duration) error {
	if host == "" {
		host = DefaultHost
	}

	dsn := fmt.Sprintf("tcp(%s)/", host)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		db, err := sql.Open("mysql", dsn)
		if err == nil {
			if err := db.Ping(); err == nil {
				db.Close()
				return nil
			}
			db.Close()
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("manticore not ready after %v", timeout)
}

// IsServiceAvailable checks if Manticore is reachable via MySQL protocol.
func IsServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHost
	}

	dsn := fmt.Sprintf("tcp(%s)/", host)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false
	}
	defer db.Close()

	return db.Ping() == nil
}

// IsHTTPServiceAvailable checks if Manticore HTTP API is reachable.
func IsHTTPServiceAvailable(host string) bool {
	if host == "" {
		host = DefaultHTTPHost
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(host + "/sql?mode=raw")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest
}

// NewWithEnv creates a driver using environment variables.
func NewWithEnv(cfg fineweb.DriverConfig) (*Driver, error) {
	if cfg.Options == nil {
		cfg.Options = make(map[string]any)
	}
	if host := os.Getenv("MANTICORE_HOST"); host != "" {
		cfg.Options["host"] = host
	}
	if httpHost := os.Getenv("MANTICORE_HTTP_HOST"); httpHost != "" {
		cfg.Options["http_host"] = httpHost
	}
	return New(cfg)
}

// SearchViaHTTP performs search using the HTTP API as fallback.
// This is an alternative to the MySQL protocol search.
func (d *Driver) SearchViaHTTP(ctx context.Context, query string, limit, offset int) (*fineweb.SearchResult, error) {
	start := time.Now()

	// Build HTTP search request
	searchQuery := map[string]any{
		"index": d.tableName,
		"query": map[string]any{
			"match": map[string]any{
				"content": query,
			},
		},
		"limit":  limit,
		"offset": offset,
	}

	body, _ := json.Marshal(searchQuery)
	req, err := http.NewRequestWithContext(ctx, "POST", d.httpHost+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing HTTP search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP search failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Took int `json:"took"`
		Hits struct {
			Total int `json:"total"`
			Hits  []struct {
				ID     int64   `json:"_id"`
				Score  float64 `json:"_score"`
				Source struct {
					URL           string  `json:"url"`
					Content       string  `json:"content"`
					Dump          string  `json:"dump"`
					Date          string  `json:"date"`
					Language      string  `json:"language"`
					LanguageScore float64 `json:"language_score"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding HTTP response: %w", err)
	}

	docs := make([]fineweb.Document, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		docs = append(docs, fineweb.Document{
			ID:            strconv.FormatInt(hit.ID, 10),
			URL:           hit.Source.URL,
			Text:          hit.Source.Content,
			Dump:          hit.Source.Dump,
			Date:          hit.Source.Date,
			Language:      hit.Source.Language,
			LanguageScore: hit.Source.LanguageScore,
			Score:         hit.Score,
		})
	}

	return &fineweb.SearchResult{
		Documents: docs,
		Duration:  time.Since(start),
		Method:    "manticore-http",
		Total:     int64(result.Hits.Total),
	}, nil
}

// Ensure Driver implements all required interfaces
var (
	_ fineweb.Driver  = (*Driver)(nil)
	_ fineweb.Indexer = (*Driver)(nil)
	_ fineweb.Stats   = (*Driver)(nil)
)
