package cc_v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkFileStoreClaim measures lock acquisition throughput.
func BenchmarkFileStoreClaim(b *testing.B) {
	dir := b.TempDir()
	s := NewFileStore(dir, "test-crawl")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 10000
		s.Claim(ctx, idx)
		s.Release(ctx, idx)
	}
}

// BenchmarkFileStoreMarkReady measures the mark-ready operation.
func BenchmarkFileStoreMarkReady(b *testing.B) {
	dir := b.TempDir()
	s := NewFileStore(dir, "test-crawl")
	ctx := context.Background()
	pqDir := filepath.Join(dir, "parquet")
	os.MkdirAll(pqDir, 0o755)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx := i % 10000
		s.Claim(ctx, idx)
		s.MarkReady(ctx, idx, filepath.Join(pqDir, fmt.Sprintf("%05d.parquet", idx)), "/tmp/test.warc.gz", &ShardStats{Rows: 100})
	}
}

// BenchmarkScanParquetDir measures directory scanning speed.
func BenchmarkScanParquetDir(b *testing.B) {
	dir := b.TempDir()
	// Create 100 parquet files.
	for i := 0; i < 100; i++ {
		path := filepath.Join(dir, fmt.Sprintf("%05d.parquet", i))
		os.WriteFile(path, make([]byte, 200), 0o644)
	}

	w := &Watcher{
		parquetDir: dir,
		committed:  make(map[int]bool),
		log:        NewLogger("bench", nil),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.scanParquetDir()
	}
}

// BenchmarkStatsCSVWrite measures CSV write throughput.
func BenchmarkStatsCSVWrite(b *testing.B) {
	dir := b.TempDir()
	csvPath := filepath.Join(dir, "stats.csv")

	rows := make([]StatsRow, 1000)
	for i := range rows {
		rows[i] = StatsRow{
			CrawlID:  "CC-MAIN-2026-12",
			FileIdx:  i,
			Rows:     int64(i * 1000),
			HTMLBytes: int64(i * 5000),
			MDBytes:  int64(i * 500),
			PqBytes:  int64(i * 200),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		writeStatsCSV(csvPath, rows)
	}
}

// BenchmarkStatsCSVRead measures CSV read throughput.
func BenchmarkStatsCSVRead(b *testing.B) {
	dir := b.TempDir()
	csvPath := filepath.Join(dir, "stats.csv")

	rows := make([]StatsRow, 1000)
	for i := range rows {
		rows[i] = StatsRow{
			CrawlID:  "CC-MAIN-2026-12",
			FileIdx:  i,
			Rows:     int64(i * 1000),
			HTMLBytes: int64(i * 5000),
			MDBytes:  int64(i * 500),
			PqBytes:  int64(i * 200),
		}
	}
	writeStatsCSV(csvPath, rows)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		readStatsCSV(csvPath)
	}
}

// BenchmarkCommittedSet measures how fast we can check committed status.
func BenchmarkCommittedSet(b *testing.B) {
	dir := b.TempDir()
	s := NewFileStore(dir, "test-crawl")
	ctx := context.Background()

	// Mark 500 shards as committed.
	for i := 0; i < 500; i++ {
		s.MarkCommitted(ctx, i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.CommittedSet(ctx)
	}
}
