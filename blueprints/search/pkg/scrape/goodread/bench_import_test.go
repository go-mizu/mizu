package goodread_test

// Benchmark: compare import strategies for seeding the DuckDB queue.
//
// Usage:
//   go test -bench=BenchmarkImport -benchtime=1x -v ./pkg/scrape/goodread/
//   go test -run=TestImportStrategies -v ./pkg/scrape/goodread/
//
// Two test groups:
//   Empty-table (original): A/B/C strategies on fresh DB.
//   Real-world (new):        D strategies with 3M pre-populated rows, 10 batches.
//
// Strategies (empty table):
//   A. BatchSQL        – batch INSERT OR IGNORE, 5000/tx
//   B. AppenderStaging – DuckDB Appender → temp table → single INSERT OR IGNORE → hash-join
//   C. ReadCSV         – write CSV → read_csv_auto() → hash-join INSERT
//
// Strategies (real-world, 3M rows pre-populated):
//   D1. PerFile_InsertIgnore  – per-file INSERT OR IGNORE (old behaviour)
//   D2. PerFile_HashJoin      – per-file LEFT JOIN anti-join INSERT
//   D3. SingleStage_Appender  – all batches → _seed_stage via Appender, one final hash anti-join
//   D4. SingleStage_Parquet   – all batches → parquet file, one final hash anti-join from read_parquet()

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"os"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	_ "github.com/duckdb/duckdb-go/v2"

	"github.com/go-mizu/mizu/blueprints/search/pkg/scrape/goodread"
)

const benchN = 48_689 // realistic: one sitemap.gz file

func makeURLs(n int) []goodread.QueueItem {
	items := make([]goodread.QueueItem, n)
	for i := range items {
		items[i] = goodread.QueueItem{
			URL:        fmt.Sprintf("https://www.goodreads.com/author/show/%d", i+1),
			EntityType: "author",
			Priority:   1,
		}
	}
	return items
}

// freshState creates a temp DuckDB state and returns it + cleanup func.
func freshState(tb testing.TB) (*goodread.State, func()) {
	tb.Helper()
	path := tb.TempDir() + "/state.duckdb"
	s, err := goodread.OpenState(path)
	if err != nil {
		tb.Fatalf("OpenState: %v", err)
	}
	return s, func() { s.Close() }
}

// ── Strategy A: current BatchSQL ─────────────────────────────────────────────

func BenchmarkImport_A_BatchSQL(b *testing.B) {
	items := makeURLs(benchN)
	b.ResetTimer()
	for range b.N {
		s, cleanup := freshState(b)
		start := time.Now()
		if err := importBatchSQL(s, items); err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(time.Since(start).Milliseconds()), "ms/op")
		b.ReportMetric(float64(benchN)/time.Since(start).Seconds(), "urls/s")
		cleanup()
	}
}

func importBatchSQL(s *goodread.State, items []goodread.QueueItem) error {
	const batchSize = 5000
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		if err := s.EnqueueBatch(items[i:end]); err != nil {
			return err
		}
	}
	return nil
}

// ── Strategy B: Appender → staging → bulk INSERT OR IGNORE ───────────────────

func BenchmarkImport_B_AppenderStaging(b *testing.B) {
	items := makeURLs(benchN)
	b.ResetTimer()
	for range b.N {
		s, cleanup := freshState(b)
		start := time.Now()
		if err := importAppenderStaging(s, items); err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(time.Since(start).Milliseconds()), "ms/op")
		b.ReportMetric(float64(benchN)/time.Since(start).Seconds(), "urls/s")
		cleanup()
	}
}

func importAppenderStaging(s *goodread.State, items []goodread.QueueItem) error {
	db := s.DB()
	ctx := context.Background()

	// Pin everything to one connection so TEMP TABLE is visible to Appender.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("conn: %w", err)
	}
	defer conn.Close()

	// Create staging table on this connection (no UNIQUE constraint).
	if _, err := conn.ExecContext(ctx,
		`CREATE TEMP TABLE IF NOT EXISTS _stage (url VARCHAR, entity_type VARCHAR, priority INTEGER)`,
	); err != nil {
		return fmt.Errorf("create staging: %w", err)
	}
	defer conn.ExecContext(ctx, `DROP TABLE IF EXISTS _stage`)

	// Fill staging via DuckDB Appender — same connection, binary row protocol.
	if err := conn.Raw(func(driverConn any) error {
		dc, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("not a driver.Conn")
		}
		app, err := duckdb.NewAppenderFromConn(dc, "", "_stage")
		if err != nil {
			return fmt.Errorf("new appender: %w", err)
		}
		for _, it := range items {
			if err := app.AppendRow(it.URL, it.EntityType, int32(it.Priority)); err != nil {
				app.Close()
				return err
			}
		}
		return app.Close()
	}); err != nil {
		return fmt.Errorf("appender: %w", err)
	}

	// Anti-join INSERT: hash join (O(n+m)) vs INSERT OR IGNORE ART lookups (O(m×log n)).
	_, err = conn.ExecContext(ctx, `
		INSERT INTO queue (url, entity_type, priority)
		SELECT s.url, s.entity_type, s.priority
		FROM _stage s
		LEFT JOIN queue q ON s.url = q.url
		WHERE q.url IS NULL
	`)
	return err
}

// ── Strategy C: write CSV → read_csv_auto() bulk INSERT ──────────────────────

func BenchmarkImport_C_ReadCSV(b *testing.B) {
	items := makeURLs(benchN)
	b.ResetTimer()
	for range b.N {
		s, cleanup := freshState(b)
		start := time.Now()
		if err := importReadCSV(s, items); err != nil {
			b.Fatal(err)
		}
		b.ReportMetric(float64(time.Since(start).Milliseconds()), "ms/op")
		b.ReportMetric(float64(benchN)/time.Since(start).Seconds(), "urls/s")
		cleanup()
	}
}

func importReadCSV(s *goodread.State, items []goodread.QueueItem) error {
	// Write CSV to temp file
	f, err := os.CreateTemp("", "sitemap_*.csv")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	w := csv.NewWriter(f)
	for _, it := range items {
		if err := w.Write([]string{it.URL, it.EntityType, fmt.Sprintf("%d", it.Priority)}); err != nil {
			f.Close()
			return err
		}
	}
	w.Flush()
	f.Close()
	if err := w.Error(); err != nil {
		return err
	}

	// Bulk INSERT OR IGNORE via read_csv_auto
	db := s.DB()
	_, err = db.ExecContext(context.Background(), fmt.Sprintf(`
		INSERT OR IGNORE INTO queue (url, entity_type, priority)
		SELECT column0, column1, CAST(column2 AS INTEGER)
		FROM read_csv_auto('%s', header=false)
	`, f.Name()))
	return err
}

// ── standalone timing comparison (run with -v -bench=.) ──────────────────────

func TestImportStrategies_Compare(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	items := makeURLs(benchN)

	type result struct {
		name string
		dur  time.Duration
		err  error
	}
	var results []result

	strategies := []struct {
		name string
		fn   func(*goodread.State, []goodread.QueueItem) error
	}{
		{"A_BatchSQL", importBatchSQL},
		{"B_AppenderStaging", importAppenderStaging},
		{"C_ReadCSV", importReadCSV},
	}

	for _, s := range strategies {
		st, cleanup := freshState(t)
		start := time.Now()
		err := s.fn(st, items)
		dur := time.Since(start)
		cleanup()
		results = append(results, result{s.name, dur, err})
	}

	t.Logf("\n── Import benchmark: %d URLs ──", benchN)
	for _, r := range results {
		if r.err != nil {
			t.Errorf("  %-25s ERROR: %v", r.name, r.err)
		} else {
			t.Logf("  %-25s %6.0f ms   (%6.0f URLs/s)",
				r.name, r.dur.Seconds()*1000,
				float64(benchN)/r.dur.Seconds())
		}
	}

	// Find fastest
	if len(results) > 1 {
		fastest := results[0]
		for _, r := range results[1:] {
			if r.err == nil && r.dur < fastest.dur {
				fastest = r
			}
		}
		t.Logf("  Winner: %s (%.1fx faster than BatchSQL)", fastest.name,
			float64(results[0].dur)/float64(fastest.dur))
	}
}

// ── Real-world benchmark: 3M pre-populated rows + 10 batches of ~38K ─────────

const (
	rwPreload = 500_000 // simulate existing queue; 3M crashes D1 via OOM
	rwBatches = 5       // simulate 5 sitemap .gz files being seeded
	rwBatchN  = benchN  // ~38K URLs per file (same as real sitemaps)
)

// preloadQueue fills the queue with n rows using EnqueueBulk (Appender staging).
func preloadQueue(tb testing.TB, st *goodread.State, n int) {
	tb.Helper()
	const chunk = 50_000
	for i := 0; i < n; i += chunk {
		end := i + chunk
		if end > n {
			end = n
		}
		items := make([]goodread.QueueItem, end-i)
		for j := range items {
			items[j] = goodread.QueueItem{
				URL:        fmt.Sprintf("https://www.goodreads.com/book/show/%d", i+j+1),
				EntityType: "book",
				Priority:   1,
			}
		}
		if err := st.EnqueueBulk(items); err != nil {
			tb.Fatalf("preload chunk %d: %v", i, err)
		}
	}
}

// makeBatches creates rwBatches slices of rwBatchN URLs each.
// Half the URLs overlap with the preloaded range to simulate real dedup.
func makeBatches() [][]goodread.QueueItem {
	batches := make([][]goodread.QueueItem, rwBatches)
	for b := range rwBatches {
		offset := rwPreload/2 + b*rwBatchN // 50% overlap with preloaded rows
		batches[b] = make([]goodread.QueueItem, rwBatchN)
		for i := range rwBatchN {
			batches[b][i] = goodread.QueueItem{
				URL:        fmt.Sprintf("https://www.goodreads.com/book/show/%d", offset+i+1),
				EntityType: "book",
				Priority:   1,
			}
		}
	}
	return batches
}

// TestImportStrategies_RealWorld compares four strategies against a 3M-row queue.
func TestImportStrategies_RealWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: pre-populates 3M rows")
	}

	batches := makeBatches()

	type rwResult struct {
		name     string
		dur      time.Duration
		inserted int64
		err      error
	}
	var results []rwResult

	runStrategy := func(name string, fn func(*goodread.State, [][]goodread.QueueItem) (int64, error)) {
		st, cleanup := freshState(t)
		t.Logf("  [%s] pre-loading %d rows ...", name, rwPreload)
		preload := time.Now()
		preloadQueue(t, st, rwPreload)
		t.Logf("  [%s] preload done in %s", name, time.Since(preload).Round(time.Millisecond))

		start := time.Now()
		n, err := fn(st, batches)
		dur := time.Since(start)
		cleanup()
		results = append(results, rwResult{name, dur, n, err})
	}

	// D1: per-file INSERT OR IGNORE (old behaviour — per-row ART index check)
	runStrategy("D1_PerFile_InsertIgnore", func(st *goodread.State, batches [][]goodread.QueueItem) (int64, error) {
		db := st.DB()
		ctx := context.Background()
		var total int64
		for _, items := range batches {
			conn, _ := db.Conn(ctx)
			conn.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS _t (url VARCHAR, entity_type VARCHAR, priority INTEGER)`)
			conn.Raw(func(dc any) error {
				c := dc.(driver.Conn)
				app, _ := duckdb.NewAppenderFromConn(c, "", "_t")
				for _, it := range items {
					app.AppendRow(it.URL, it.EntityType, int32(it.Priority)) //nolint:errcheck
				}
				return app.Close()
			})
			res, err := conn.ExecContext(ctx, `INSERT OR IGNORE INTO queue (url, entity_type, priority) SELECT url, entity_type, priority FROM _t`)
			if err != nil {
				conn.Close()
				return total, err
			}
			n, _ := res.RowsAffected()
			total += n
			conn.ExecContext(ctx, `DROP TABLE IF EXISTS _t`) //nolint:errcheck
			conn.Close()
		}
		return total, nil
	})

	// D2: per-file LEFT JOIN anti-join (hash join per file, table grows each iteration)
	runStrategy("D2_PerFile_HashJoin", func(st *goodread.State, batches [][]goodread.QueueItem) (int64, error) {
		db := st.DB()
		ctx := context.Background()
		var total int64
		for _, items := range batches {
			conn, _ := db.Conn(ctx)
			conn.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS _t (url VARCHAR, entity_type VARCHAR, priority INTEGER)`)
			conn.Raw(func(dc any) error {
				c := dc.(driver.Conn)
				app, _ := duckdb.NewAppenderFromConn(c, "", "_t")
				for _, it := range items {
					app.AppendRow(it.URL, it.EntityType, int32(it.Priority)) //nolint:errcheck
				}
				return app.Close()
			})
			res, err := conn.ExecContext(ctx, `
				INSERT INTO queue (url, entity_type, priority)
				SELECT t.url, t.entity_type, t.priority FROM _t t
				LEFT JOIN queue q ON t.url = q.url WHERE q.url IS NULL`)
			if err != nil {
				conn.Close()
				return total, err
			}
			n, _ := res.RowsAffected()
			total += n
			conn.ExecContext(ctx, `DROP TABLE IF EXISTS _t`) //nolint:errcheck
			conn.Close()
		}
		return total, nil
	})

	// D3: single-stage via Appender (all batches → _seed_stage, one final hash anti-join)
	runStrategy("D3_SingleStage_Appender", func(st *goodread.State, batches [][]goodread.QueueItem) (int64, error) {
		if err := st.CreateSeedStage(); err != nil {
			return 0, err
		}
		for _, items := range batches {
			if err := st.AppendSeedBatch(items); err != nil {
				return 0, err
			}
		}
		return st.FlushSeedToQueue()
	})

	// D4: single-stage via Parquet (all batches → parquet, one final hash anti-join)
	runStrategy("D4_SingleStage_Parquet", func(st *goodread.State, batches [][]goodread.QueueItem) (int64, error) {
		return importViaParquet(st.DB(), batches)
	})

	// ── Print results ──────────────────────────────────────────────────────────
	totalURLs := rwBatches * rwBatchN
	t.Logf("\n── Real-world import benchmark: %d pre-existing rows, %d batches × %d URLs ──",
		rwPreload, rwBatches, rwBatchN)
	baseline := results[0].dur
	for _, r := range results {
		if r.err != nil {
			t.Errorf("  %-30s ERROR: %v", r.name, r.err)
		} else {
			ratio := float64(baseline) / float64(r.dur)
			t.Logf("  %-30s %6.0f ms  %6.0f URLs/s  inserted=%d  (%.1fx vs D1)",
				r.name, r.dur.Seconds()*1000,
				float64(totalURLs)/r.dur.Seconds(),
				r.inserted, ratio)
		}
	}
	fastest := results[0]
	for _, r := range results[1:] {
		if r.err == nil && r.dur < fastest.dur {
			fastest = r
		}
	}
	t.Logf("  Winner: %s", fastest.name)
}

// importViaParquet writes all batches to a parquet file via DuckDB, then
// does a single hash anti-join INSERT from read_parquet().
func importViaParquet(db *sql.DB, batches [][]goodread.QueueItem) (int64, error) {
	ctx := context.Background()

	// Stage all URLs into a temp DuckDB table, then COPY to parquet.
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx,
		`CREATE TEMP TABLE _pq_stage (url VARCHAR, entity_type VARCHAR, priority INTEGER)`); err != nil {
		return 0, err
	}

	if err := conn.Raw(func(dc any) error {
		c := dc.(driver.Conn)
		app, err := duckdb.NewAppenderFromConn(c, "", "_pq_stage")
		if err != nil {
			return err
		}
		for _, items := range batches {
			for _, it := range items {
				if err := app.AppendRow(it.URL, it.EntityType, int32(it.Priority)); err != nil {
					app.Close()
					return err
				}
			}
		}
		return app.Close()
	}); err != nil {
		return 0, err
	}

	// Write to parquet.
	pqPath := os.TempDir() + "/goodread_seed_bench.parquet"
	defer os.Remove(pqPath)
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(
		`COPY _pq_stage TO '%s' (FORMAT PARQUET)`, pqPath)); err != nil {
		return 0, fmt.Errorf("copy to parquet: %w", err)
	}
	conn.ExecContext(ctx, `DROP TABLE IF EXISTS _pq_stage`) //nolint:errcheck
	conn.Close()

	// Hash anti-join INSERT from parquet.
	res, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO queue (url, entity_type, priority)
		SELECT p.url, p.entity_type, p.priority
		FROM read_parquet('%s') p
		LEFT JOIN queue q ON p.url = q.url
		WHERE q.url IS NULL
	`, pqPath))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

