# CC FTS Index Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Index 154,990 markdown files from Common Crawl into pluggable FTS engines (devnull, duckdb, sqlite, chdb) and provide keyword search with ranked results.

**Architecture:** Three-stage pipeline (walker → readers → batcher) feeds documents into a pluggable Engine interface. Each driver self-registers via `init()`. CLI adds `search cc fts index` and `search cc fts "query"` subcommands.

**Tech Stack:** Go 1.25, DuckDB FTS extension (duckdb-go/v2), modernc.org/sqlite FTS5, chdb-go (CGO, build-tagged), Cobra CLI.

---

### Task 1: Engine Interface and Registry

**Files:**
- Create: `pkg/index/engine.go`

**Step 1: Write engine.go**

```go
package index

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Engine is a pluggable FTS backend with lifecycle management.
type Engine interface {
	Name() string
	Open(ctx context.Context, dir string) error
	Close() error
	Stats(ctx context.Context) (EngineStats, error)
	Index(ctx context.Context, docs []Document) error
	Search(ctx context.Context, q Query) (Results, error)
}

// EngineStats reports index metadata.
type EngineStats struct {
	DocCount  int64
	DiskBytes int64
}

// --- registry ---

type EngineFactory func() Engine

var (
	registry   = make(map[string]EngineFactory)
	registryMu sync.RWMutex
)

func Register(name string, factory EngineFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("index: driver %q already registered", name))
	}
	registry[name] = factory
}

func NewEngine(name string) (Engine, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("index: unknown driver %q (available: %v)", name, List())
	}
	return factory(), nil
}

func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build ./pkg/index/...`
Expected: success, no output

**Step 3: Commit**

```bash
git add pkg/index/engine.go
git commit -m "feat(index): add Engine interface and driver registry"
```

---

### Task 2: devnull Driver

**Files:**
- Create: `pkg/index/driver/devnull/devnull.go`

**Step 1: Write devnull.go**

```go
package devnull

import (
	"context"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("devnull", func() index.Engine { return &Engine{} })
}

type Engine struct{}

func (e *Engine) Name() string                                         { return "devnull" }
func (e *Engine) Open(_ context.Context, _ string) error               { return nil }
func (e *Engine) Close() error                                         { return nil }
func (e *Engine) Stats(_ context.Context) (index.EngineStats, error)   { return index.EngineStats{}, nil }
func (e *Engine) Index(_ context.Context, _ []index.Document) error    { return nil }
func (e *Engine) Search(_ context.Context, _ index.Query) (index.Results, error) {
	return index.Results{}, nil
}

var _ index.Engine = (*Engine)(nil)
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/index/driver/devnull/...`
Expected: success

**Step 3: Commit**

```bash
git add pkg/index/driver/devnull/devnull.go
git commit -m "feat(index): add devnull driver (no-op baseline)"
```

---

### Task 3: Index Pipeline

**Files:**
- Create: `pkg/index/pipeline.go`

**Step 1: Write pipeline.go**

This implements the 3-stage pipeline: walker → readers → batcher.

```go
package index

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PipelineConfig controls the indexing pipeline.
type PipelineConfig struct {
	SourceDir string // markdown/ directory path
	BatchSize int    // docs per Engine.Index call (default 5000)
	Workers   int    // parallel file readers (default 4)
}

// PipelineStats tracks pipeline progress.
type PipelineStats struct {
	TotalFiles  atomic.Int64
	DocsIndexed atomic.Int64
	Errors      atomic.Int64
	StartTime   time.Time
	PeakRSSMB   atomic.Int64 // peak RSS in MB
}

// ProgressFunc is called periodically with current stats.
type ProgressFunc func(stats *PipelineStats)

// RunPipeline indexes all markdown files from sourceDir into engine.
func RunPipeline(ctx context.Context, engine Engine, cfg PipelineConfig, progress ProgressFunc) (*PipelineStats, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 5000
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}

	stats := &PipelineStats{StartTime: time.Now()}

	// Start memory tracker
	memStop := make(chan struct{})
	go trackPeakMem(stats, memStop)
	defer close(memStop)

	// Start progress ticker
	if progress != nil {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		go func() {
			for {
				select {
				case <-ticker.C:
					progress(stats)
				case <-ctx.Done():
					return
				case <-memStop:
					return
				}
			}
		}()
	}

	fileCh := make(chan string, 1000)
	docCh := make(chan Document, 5000)

	// Stage 1: Walker
	var walkErr error
	go func() {
		defer close(fileCh)
		walkErr = walkMarkdown(ctx, cfg.SourceDir, fileCh, stats)
	}()

	// Stage 2: Readers
	var readerWg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		readerWg.Add(1)
		go func() {
			defer readerWg.Done()
			readFiles(ctx, fileCh, docCh, stats)
		}()
	}
	go func() {
		readerWg.Wait()
		close(docCh)
	}()

	// Stage 3: Batcher
	if err := batchIndex(ctx, engine, docCh, cfg.BatchSize, stats); err != nil {
		return stats, err
	}

	if walkErr != nil {
		return stats, walkErr
	}

	// Final progress call
	if progress != nil {
		progress(stats)
	}

	return stats, nil
}

func walkMarkdown(ctx context.Context, dir string, out chan<- string, stats *PipelineStats) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable dirs
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".md.gz") || strings.HasSuffix(name, ".md") {
			stats.TotalFiles.Add(1)
			select {
			case out <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
}

func readFiles(ctx context.Context, in <-chan string, out chan<- Document, stats *PipelineStats) {
	for path := range in {
		if ctx.Err() != nil {
			return
		}
		doc, err := readMarkdownFile(path)
		if err != nil {
			stats.Errors.Add(1)
			continue
		}
		if doc.Text == "" {
			continue
		}
		select {
		case out <- doc:
		case <-ctx.Done():
			return
		}
	}
}

func readMarkdownFile(path string) (Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()

	var r io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return Document{}, err
		}
		defer gr.Close()
		r = gr
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return Document{}, err
	}

	// Extract UUID from filename: strip directory and extensions
	base := filepath.Base(path)
	docID := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".md")

	return Document{
		DocID: docID,
		Text:  string(data),
	}, nil
}

func batchIndex(ctx context.Context, engine Engine, docs <-chan Document, batchSize int, stats *PipelineStats) error {
	batch := make([]Document, 0, batchSize)
	for doc := range docs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		batch = append(batch, doc)
		if len(batch) >= batchSize {
			if err := engine.Index(ctx, batch); err != nil {
				return fmt.Errorf("index batch: %w", err)
			}
			stats.DocsIndexed.Add(int64(len(batch)))
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := engine.Index(ctx, batch); err != nil {
			return fmt.Errorf("index final batch: %w", err)
		}
		stats.DocsIndexed.Add(int64(len(batch)))
	}
	return nil
}

func trackPeakMem(stats *PipelineStats, stop <-chan struct{}) {
	var m runtime.MemStats
	for {
		select {
		case <-stop:
			return
		case <-time.After(500 * time.Millisecond):
			runtime.ReadMemStats(&m)
			mb := int64(m.Sys / (1024 * 1024))
			for {
				cur := stats.PeakRSSMB.Load()
				if mb <= cur || stats.PeakRSSMB.CompareAndSwap(cur, mb) {
					break
				}
			}
		}
	}
}

// DirSizeBytes returns total size of files in dir (non-recursive is fine for single-level).
func DirSizeBytes(dir string) int64 {
	var total int64
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/index/...`
Expected: success

**Step 3: Commit**

```bash
git add pkg/index/pipeline.go
git commit -m "feat(index): add 3-stage indexing pipeline (walk → read → batch)"
```

---

### Task 4: DuckDB Driver

**Files:**
- Create: `pkg/index/driver/duckdb/duckdb.go`

**Step 1: Write duckdb.go**

```go
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("duckdb", func() index.Engine { return &Engine{} })
}

type Engine struct {
	db     *sql.DB
	dbPath string
	dir    string
}

func (e *Engine) Name() string { return "duckdb" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.dbPath = filepath.Join(dir, "fts.duckdb")

	db, err := sql.Open("duckdb", e.dbPath)
	if err != nil {
		return fmt.Errorf("duckdb open %s: %w", e.dbPath, err)
	}
	e.db = db

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS documents (
		doc_id VARCHAR PRIMARY KEY,
		text   VARCHAR
	)`)
	return err
}

func (e *Engine) Close() error {
	if e.db == nil {
		return nil
	}
	return e.db.Close()
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var count int64
	if err := e.db.QueryRowContext(ctx, "SELECT count(*) FROM documents").Scan(&count); err != nil {
		return index.EngineStats{}, err
	}
	disk := index.DirSizeBytes(e.dir)
	return index.EngineStats{DocCount: count, DiskBytes: disk}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO documents (doc_id, text) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, doc := range docs {
		if _, err := stmt.ExecContext(ctx, doc.DocID, doc.Text); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	// Try FTS search first
	rows, err := e.searchFTS(ctx, q.Text, limit, q.Offset)
	if err != nil {
		// Fallback to LIKE
		return e.searchLike(ctx, q.Text, limit, q.Offset)
	}
	return rows, nil
}

func (e *Engine) searchFTS(ctx context.Context, query string, limit, offset int) (index.Results, error) {
	// Load FTS extension
	if _, err := e.db.ExecContext(ctx, "INSTALL fts; LOAD fts"); err != nil {
		return index.Results{}, err
	}

	// Check if FTS index exists
	var n int
	err := e.db.QueryRowContext(ctx,
		"SELECT count(*) FROM information_schema.schemata WHERE schema_name = 'fts_main_documents'").Scan(&n)
	if err != nil || n == 0 {
		return index.Results{}, fmt.Errorf("no FTS index")
	}

	sql := `SELECT doc_id, substring(text, 1, 200) AS snippet,
	        fts_main_documents.match_bm25(doc_id, ?, fields := 'text') AS score
	        FROM documents WHERE score IS NOT NULL
	        ORDER BY score DESC LIMIT ? OFFSET ?`

	rows, err := e.db.QueryContext(ctx, sql, query, limit, offset)
	if err != nil {
		return index.Results{}, err
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

func (e *Engine) searchLike(ctx context.Context, query string, limit, offset int) (index.Results, error) {
	sql := `SELECT doc_id, substring(text, 1, 200) AS snippet, 1.0 AS score
	        FROM documents WHERE text ILIKE '%' || ? || '%'
	        LIMIT ? OFFSET ?`

	rows, err := e.db.QueryContext(ctx, sql, query, limit, offset)
	if err != nil {
		return index.Results{}, err
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

// CreateFTSIndex builds the DuckDB FTS index. Call after all documents are indexed.
func (e *Engine) CreateFTSIndex(ctx context.Context) error {
	if _, err := e.db.ExecContext(ctx, "INSTALL fts; LOAD fts"); err != nil {
		return fmt.Errorf("load fts: %w", err)
	}
	_, err := e.db.ExecContext(ctx,
		`PRAGMA create_fts_index('documents', 'doc_id', 'text',
		 stemmer='english', stopwords='english', lower=1, strip_accents=1, overwrite=1)`)
	if err != nil {
		return fmt.Errorf("create fts index: %w", err)
	}
	return nil
}

var _ index.Engine = (*Engine)(nil)
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/index/driver/duckdb/...`
Expected: success

**Step 3: Commit**

```bash
git add pkg/index/driver/duckdb/duckdb.go
git commit -m "feat(index): add DuckDB FTS driver with BM25 scoring"
```

---

### Task 5: SQLite Driver

**Files:**
- Create: `pkg/index/driver/sqlite/sqlite.go`

**Step 1: Write sqlite.go**

```go
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("sqlite", func() index.Engine { return &Engine{} })
}

type Engine struct {
	db     *sql.DB
	dbPath string
	dir    string
}

func (e *Engine) Name() string { return "sqlite" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir
	e.dbPath = filepath.Join(dir, "fts.db")
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", e.dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("sqlite open %s: %w", e.dbPath, err)
	}
	e.db = db

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			doc_id TEXT PRIMARY KEY,
			text   TEXT
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			text,
			content='documents',
			content_rowid='rowid',
			tokenize='unicode61 remove_diacritics 0'
		)`,
		`CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO documents_fts(rowid, text) VALUES (new.rowid, new.text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, text) VALUES ('delete', old.rowid, old.text);
		END`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("sqlite init: %w", err)
		}
	}
	return nil
}

func (e *Engine) Close() error {
	if e.db == nil {
		return nil
	}
	// Optimize FTS before closing
	e.db.Exec("INSERT INTO documents_fts(documents_fts) VALUES ('optimize')")
	return e.db.Close()
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	var count int64
	if err := e.db.QueryRowContext(ctx, "SELECT count(*) FROM documents").Scan(&count); err != nil {
		return index.EngineStats{}, err
	}
	disk := index.DirSizeBytes(e.dir)
	return index.EngineStats{DocCount: count, DiskBytes: disk}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO documents (doc_id, text) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, doc := range docs {
		if _, err := stmt.ExecContext(ctx, doc.DocID, doc.Text); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	sql := `SELECT d.doc_id, snippet(f, 0, '', '', '...', 40) AS snippet,
	        -bm25(f) AS score
	        FROM documents_fts f
	        JOIN documents d ON d.rowid = f.rowid
	        WHERE f.text MATCH ?
	        ORDER BY bm25(f)
	        LIMIT ? OFFSET ?`

	rows, err := e.db.QueryContext(ctx, sql, q.Text, limit, q.Offset)
	if err != nil {
		return index.Results{}, fmt.Errorf("sqlite search: %w", err)
	}
	defer rows.Close()

	var results index.Results
	for rows.Next() {
		var h index.Hit
		if err := rows.Scan(&h.DocID, &h.Snippet, &h.Score); err != nil {
			return results, err
		}
		results.Hits = append(results.Hits, h)
	}
	results.Total = len(results.Hits)
	return results, rows.Err()
}

var _ index.Engine = (*Engine)(nil)
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/index/driver/sqlite/...`
Expected: success

**Step 3: Commit**

```bash
git add pkg/index/driver/sqlite/sqlite.go
git commit -m "feat(index): add SQLite FTS5 driver with BM25 scoring"
```

---

### Task 6: chdb Driver (build-tagged)

**Files:**
- Create: `pkg/index/driver/chdb/chdb.go`

**Step 1: Add chdb-go dependency**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && go get github.com/chdb-io/chdb-go@latest`

**Step 2: Write chdb.go**

```go
//go:build chdb

package chdb

import (
	"context"
	"fmt"
	"os"
	"strings"

	chdb_api "github.com/chdb-io/chdb-go/chdb"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
)

func init() {
	index.Register("chdb", func() index.Engine { return &Engine{} })
}

type Engine struct {
	session *chdb_api.Session
	dir     string
}

func (e *Engine) Name() string { return "chdb" }

func (e *Engine) Open(ctx context.Context, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e.dir = dir

	session, err := chdb_api.NewSession(dir)
	if err != nil {
		return fmt.Errorf("chdb open %s: %w", dir, err)
	}
	e.session = session

	_, err = session.Query(`CREATE TABLE IF NOT EXISTS documents (
		doc_id String,
		text   String,
		INDEX text_idx text TYPE inverted()
	) ENGINE = MergeTree() ORDER BY doc_id
	SETTINGS index_granularity = 8192`)
	if err != nil {
		return fmt.Errorf("chdb create table: %w", err)
	}
	return nil
}

func (e *Engine) Close() error {
	if e.session != nil {
		e.session.Cleanup()
	}
	return nil
}

func (e *Engine) Stats(ctx context.Context) (index.EngineStats, error) {
	result, err := e.session.Query("SELECT count(*) FROM documents")
	if err != nil {
		return index.EngineStats{}, err
	}
	var count int64
	fmt.Sscan(strings.TrimSpace(result.String()), &count)
	disk := index.DirSizeBytes(e.dir)
	return index.EngineStats{DocCount: count, DiskBytes: disk}, nil
}

func (e *Engine) Index(ctx context.Context, docs []index.Document) error {
	if len(docs) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO documents (doc_id, text) VALUES ")
	for i, doc := range docs {
		if i > 0 {
			sb.WriteString(", ")
		}
		// Escape single quotes
		id := strings.ReplaceAll(doc.DocID, "'", "''")
		text := strings.ReplaceAll(doc.Text, "'", "''")
		// Escape backslashes for ClickHouse
		text = strings.ReplaceAll(text, `\`, `\\`)
		fmt.Fprintf(&sb, "('%s', '%s')", id, text)
	}
	_, err := e.session.Query(sb.String())
	return err
}

func (e *Engine) Search(ctx context.Context, q index.Query) (index.Results, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}

	// Escape query for safety
	query := strings.ReplaceAll(q.Text, "'", "''")
	query = strings.ReplaceAll(query, `\`, `\\`)

	sql := fmt.Sprintf(`SELECT doc_id, substring(text, 1, 200) AS snippet
		FROM documents
		WHERE hasAllTokens(lower(text), lower('%s'))
		ORDER BY length(text) ASC
		LIMIT %d OFFSET %d FORMAT TabSeparated`, query, limit, q.Offset)

	result, err := e.session.Query(sql)
	if err != nil {
		return index.Results{}, fmt.Errorf("chdb search: %w", err)
	}

	var results index.Results
	lines := strings.Split(strings.TrimSpace(result.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		results.Hits = append(results.Hits, index.Hit{
			DocID:   parts[0],
			Score:   1.0,
			Snippet: parts[1],
		})
	}
	results.Total = len(results.Hits)
	return results, nil
}

var _ index.Engine = (*Engine)(nil)
```

**Step 3: Create stub for non-chdb builds**

Create `pkg/index/driver/chdb/stub.go`:

```go
//go:build !chdb

package chdb

// When built without the chdb tag, this package is empty.
// The chdb driver is not registered.
```

**Step 4: Verify it compiles (without chdb tag)**

Run: `go build ./pkg/index/driver/chdb/...`
Expected: success (empty package, no registration)

**Step 5: Commit**

```bash
git add pkg/index/driver/chdb/
git commit -m "feat(index): add chdb driver with inverted index (build-tagged)"
```

---

### Task 7: CLI Commands (cc fts)

**Files:**
- Create: `cli/cc_fts.go`
- Modify: `cli/cc.go` (add `cmd.AddCommand(newCCFTS())`)

**Step 1: Write cc_fts.go**

```go
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/index"
	// Import all drivers for registration
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/devnull"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/duckdb"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/sqlite"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/index/driver/chdb"
	"github.com/spf13/cobra"
)

func newCCFTS() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fts",
		Short: "Full-text search index and query for CC markdown",
		Long:  `Build FTS indexes from Common Crawl markdown files and search them.`,
		Example: `  search cc fts index --engine duckdb
  search cc fts index --engine sqlite --workers 8 --batch-size 10000
  search cc fts "machine learning" --engine duckdb --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newCCFTSIndex())
	cmd.AddCommand(newCCFTSSearch())
	return cmd
}

func newCCFTSIndex() *cobra.Command {
	var (
		crawlID   string
		engine    string
		batchSize int
		workers   int
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Build FTS index from CC markdown files",
		Example: `  search cc fts index --engine duckdb
  search cc fts index --engine sqlite --crawl CC-MAIN-2026-08
  search cc fts index --engine devnull  # benchmark I/O only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCCFTSIndex(cmd.Context(), crawlID, engine, batchSize, workers)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine: "+strings.Join(index.List(), ", "))
	cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Documents per batch insert")
	cmd.Flags().IntVar(&workers, "workers", 4, "Parallel file readers")
	return cmd
}

func newCCFTSSearch() *cobra.Command {
	var (
		crawlID string
		engine  string
		limit   int
		offset  int
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search the FTS index",
		Args:  cobra.MinimumNArgs(1),
		Example: `  search cc fts search "machine learning" --engine duckdb
  search cc fts search "climate change" --limit 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return runCCFTSSearch(cmd.Context(), crawlID, engine, query, limit, offset)
		},
	}

	cmd.Flags().StringVar(&crawlID, "crawl", "", "Crawl ID (default: latest)")
	cmd.Flags().StringVar(&engine, "engine", "duckdb", "FTS engine")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "Result offset")
	return cmd
}

func runCCFTSIndex(ctx context.Context, crawlID, engineName string, batchSize, workers int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	sourceDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown")
	outputDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("markdown dir not found: %s", sourceDir)
	}

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}

	if err := eng.Open(ctx, outputDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	fmt.Fprintf(os.Stderr, "indexing %s → %s (engine=%s, batch=%d, workers=%d)\n",
		sourceDir, outputDir, engineName, batchSize, workers)

	cfg := index.PipelineConfig{
		SourceDir: sourceDir,
		BatchSize: batchSize,
		Workers:   workers,
	}

	progress := func(stats *index.PipelineStats) {
		total := stats.TotalFiles.Load()
		done := stats.DocsIndexed.Load()
		elapsed := time.Since(stats.StartTime).Seconds()
		rate := float64(0)
		if elapsed > 0 {
			rate = float64(done) / elapsed
		}
		disk := index.DirSizeBytes(outputDir)
		peakMB := stats.PeakRSSMB.Load()

		// Progress bar
		pct := float64(0)
		if total > 0 {
			pct = float64(done) / float64(total)
		}
		barWidth := 20
		filled := int(pct * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		fmt.Fprintf(os.Stderr, "\r\033[Kindexing [%s] %s %d/%d docs │ %.0f docs/s │ %.1fs │ RSS %d MB │ disk %s",
			engineName, bar, done, total, rate, elapsed, peakMB, formatBytes(disk))
	}

	stats, err := index.RunPipeline(ctx, eng, cfg, progress)
	fmt.Fprintln(os.Stderr) // newline after progress

	if err != nil {
		return err
	}

	// Create FTS index for DuckDB (post-insert step)
	if engineName == "duckdb" {
		fmt.Fprintf(os.Stderr, "creating FTS index (BM25)...\n")
		if ddb, ok := eng.(interface{ CreateFTSIndex(context.Context) error }); ok {
			if err := ddb.CreateFTSIndex(ctx); err != nil {
				return fmt.Errorf("create FTS index: %w", err)
			}
		}
	}

	// Final summary
	engineStats, _ := eng.Stats(ctx)
	elapsed := time.Since(stats.StartTime)
	avgRate := float64(stats.DocsIndexed.Load()) / elapsed.Seconds()

	fmt.Fprintf(os.Stderr, "\n── FTS Index Complete ──────────────────────────\n")
	fmt.Fprintf(os.Stderr, "  engine:    %s\n", engineName)
	fmt.Fprintf(os.Stderr, "  docs:      %d\n", engineStats.DocCount)
	fmt.Fprintf(os.Stderr, "  errors:    %d\n", stats.Errors.Load())
	fmt.Fprintf(os.Stderr, "  elapsed:   %s\n", elapsed.Round(100*time.Millisecond))
	fmt.Fprintf(os.Stderr, "  avg rate:  %.0f docs/s\n", avgRate)
	fmt.Fprintf(os.Stderr, "  peak RSS:  %d MB\n", stats.PeakRSSMB.Load())
	fmt.Fprintf(os.Stderr, "  disk:      %s\n", formatBytes(engineStats.DiskBytes))
	fmt.Fprintf(os.Stderr, "  path:      %s\n", outputDir)

	return nil
}

func runCCFTSSearch(ctx context.Context, crawlID, engineName, query string, limit, offset int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	eng, err := index.NewEngine(engineName)
	if err != nil {
		return err
	}

	if err := eng.Open(ctx, outputDir); err != nil {
		return fmt.Errorf("open engine: %w", err)
	}
	defer eng.Close()

	start := time.Now()
	results, err := eng.Search(ctx, index.Query{
		Text:   query,
		Limit:  limit,
		Offset: offset,
	})
	elapsed := time.Since(start)

	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	fmt.Fprintf(os.Stderr, "── Results for %q (engine: %s, %d hits, %s) ──\n",
		query, engineName, results.Total, elapsed.Round(time.Microsecond))
	fmt.Fprintf(os.Stderr, "  %-4s %-8s %-40s %s\n", "#", "Score", "DocID", "Snippet")
	fmt.Fprintf(os.Stderr, "  %-4s %-8s %-40s %s\n", "──", "────────", strings.Repeat("─", 40), strings.Repeat("─", 40))

	for i, hit := range results.Hits {
		snippet := hit.Snippet
		if len(snippet) > 80 {
			snippet = snippet[:80] + "..."
		}
		// Replace newlines with spaces for display
		snippet = strings.ReplaceAll(snippet, "\n", " ")
		snippet = strings.ReplaceAll(snippet, "\r", "")
		fmt.Fprintf(os.Stderr, "  %-4d %-8.2f %-40s %s\n",
			i+1+offset, hit.Score, hit.DocID, snippet)
	}

	return nil
}

func detectLatestCrawl() string {
	homeDir, _ := os.UserHomeDir()
	ccDir := filepath.Join(homeDir, "data", "common-crawl")
	entries, err := os.ReadDir(ccDir)
	if err != nil {
		return "CC-MAIN-2026-08"
	}
	// Find latest CC-MAIN-* directory
	latest := ""
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "CC-MAIN-") {
			if e.Name() > latest {
				latest = e.Name()
			}
		}
	}
	if latest == "" {
		return "CC-MAIN-2026-08"
	}
	return latest
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
```

**Step 2: Register in cc.go**

Add `cmd.AddCommand(newCCFTS())` in `NewCC()` function in `cli/cc.go`, after the last `cmd.AddCommand(...)` line.

**Step 3: Verify it compiles**

Run: `go build ./cli/... && go build ./cmd/search/...`
Expected: success

**Step 4: Commit**

```bash
git add cli/cc_fts.go cli/cc.go
git commit -m "feat(cli): add 'search cc fts index' and 'search cc fts search' commands"
```

---

### Task 8: Test Locally on macOS

**Step 1: Build**

Run: `cd /Users/apple/github/go-mizu/mizu/blueprints/search && make install`

**Step 2: Test devnull driver (I/O baseline)**

Run: `search cc fts index --engine devnull --crawl CC-MAIN-2026-08`
Expected: indexes all 154K files, shows progress, completes with summary. Disk = 0.

**Step 3: Test DuckDB driver**

Run: `search cc fts index --engine duckdb --crawl CC-MAIN-2026-08`
Expected: creates `~/data/common-crawl/CC-MAIN-2026-08/fts/duckdb/fts.duckdb`, builds FTS index.

**Step 4: Test DuckDB search**

Run: `search cc fts search "machine learning" --engine duckdb`
Expected: shows ranked results with BM25 scores.

**Step 5: Test SQLite driver**

Run: `search cc fts index --engine sqlite --crawl CC-MAIN-2026-08`
Expected: creates `~/data/common-crawl/CC-MAIN-2026-08/fts/sqlite/fts.db`.

**Step 6: Test SQLite search**

Run: `search cc fts search "machine learning" --engine sqlite`
Expected: shows ranked results with BM25 scores.

**Step 7: Fix any issues discovered during testing**

---

### Task 9: Deploy and Benchmark on Server 2

**Step 1: Build for Linux**

Run: `make build-linux-noble`
Expected: produces Linux amd64 binary.

**Step 2: Deploy to server2**

Run: `make deploy-linux-noble SERVER=2`

**Step 3: Run benchmark on server2**

SSH into server2 and run each engine with `time`:

```bash
# Ensure markdown data exists
ls ~/data/common-crawl/CC-MAIN-2026-08/markdown/ | head

# devnull baseline
time search cc fts index --engine devnull --crawl CC-MAIN-2026-08

# DuckDB
rm -rf ~/data/common-crawl/CC-MAIN-2026-08/fts/duckdb/
time search cc fts index --engine duckdb --crawl CC-MAIN-2026-08

# SQLite
rm -rf ~/data/common-crawl/CC-MAIN-2026-08/fts/sqlite/
time search cc fts index --engine sqlite --crawl CC-MAIN-2026-08

# Search benchmarks (10 queries × 3 runs each)
for q in "machine learning" "climate change" "artificial intelligence" "United States" "open source software" "COVID-19 pandemic" "data privacy" "renewable energy" "blockchain technology" "neural network"; do
  echo "--- $q ---"
  for i in 1 2 3; do
    time search cc fts search "$q" --engine duckdb 2>&1 | tail -1
  done
done
```

**Step 4: Fill in benchmark tables in spec/0642_index_fts.md**

Update the benchmark tables with actual numbers from server2.

**Step 5: Commit benchmark results**

```bash
git add spec/0642_index_fts.md
git commit -m "bench(index): add FTS benchmark results from server2"
```

---

### Task 10: chdb Driver Testing (if libchdb available)

**Step 1: Install libchdb on server2**

Run: `curl -sL https://lib.chdb.io | bash`

**Step 2: Build with chdb tag**

Run: `CGO_ENABLED=1 go build -tags chdb -o ~/bin/search ./cmd/search/`

**Step 3: Test chdb index + search**

```bash
rm -rf ~/data/common-crawl/CC-MAIN-2026-08/fts/chdb/
time search cc fts index --engine chdb --crawl CC-MAIN-2026-08
search cc fts search "machine learning" --engine chdb
```

**Step 4: Add chdb benchmarks to spec**

Update `spec/0642_index_fts.md` with chdb numbers.
