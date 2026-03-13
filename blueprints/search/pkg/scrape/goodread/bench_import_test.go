package goodread_test

// Benchmark: compare three strategies for importing sitemap URLs into DuckDB queue.
//
// Usage:
//   go test -bench=BenchmarkImport -benchtime=1x -v ./pkg/scrape/goodread/
//
// Strategies:
//   A. BatchSQL        – current: batch INSERT OR IGNORE, 5000/tx
//   B. AppenderStaging – DuckDB Appender → temp table → single bulk INSERT OR IGNORE
//   C. ReadCSV         – write CSV to /tmp → INSERT OR IGNORE ... FROM read_csv_auto()

import (
	"context"
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

	// Bulk INSERT OR IGNORE from staging (one vectorized DuckDB operation).
	_, err = conn.ExecContext(ctx, `
		INSERT OR IGNORE INTO queue (url, entity_type, priority)
		SELECT url, entity_type, priority FROM _stage
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

