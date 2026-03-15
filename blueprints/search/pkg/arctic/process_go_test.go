package arctic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// generateChunk creates a synthetic JSONL file with nLines comment records.
// Returns the file path and size in bytes.
func generateChunk(t *testing.T, dir string, nLines int) (string, int64) {
	t.Helper()
	chunkPath := filepath.Join(dir, "chunk_0000.jsonl")
	f, err := os.Create(chunkPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < nLines; i++ {
		row := map[string]any{
			"id":                fmt.Sprintf("abc%d", i),
			"author":           fmt.Sprintf("user_%d", i%1000),
			"subreddit":        fmt.Sprintf("subreddit_%d", i%200),
			"body":             fmt.Sprintf("This is comment body number %d with some extra text to make it realistic and test compression ratios properly. Adding more words to simulate average Reddit comment length which is typically 100-300 characters.", i),
			"score":            i % 500,
			"created_utc":      1300000000 + i,
			"link_id":          fmt.Sprintf("t3_link%d", i%100),
			"parent_id":        fmt.Sprintf("t1_parent%d", i%200),
			"distinguished":    nil,
			"author_flair_text": nil,
		}
		b, _ := json.Marshal(row)
		f.Write(b)
		f.Write([]byte{'\n'})
	}
	f.Close()
	fi, _ := os.Stat(chunkPath)
	return chunkPath, fi.Size()
}

func memStatsMB() (alloc, sys float64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024, float64(m.Sys) / 1024 / 1024
}

// TestBenchmarkAllEngines benchmarks Go (level 11), Go in-memory, and DuckDB (level 3 + parquet v2)
// side by side with memory tracking.
// Run: go test -run TestBenchmarkAllEngines -v -count=1 ./pkg/arctic/
func TestBenchmarkAllEngines(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping benchmark in short mode")
	}

	tmp := t.TempDir()
	const nLines = 100_000 // 100K = one full chunk at new ChunkLines

	t.Logf("Generating %d synthetic comment lines...", nLines)
	chunkPath, chunkSize := generateChunk(t, tmp, nLines)
	t.Logf("Chunk file: %.1f MB (%d lines)", float64(chunkSize)/1024/1024, nLines)

	ctx := context.Background()

	type result struct {
		name    string
		rows    int64
		sizeMB  float64
		dur     time.Duration
		allocMB float64 // heap alloc delta
	}
	var results []result

	// --- Go engine (disk chunk, SpeedBestCompression ~level 11) ---
	{
		cfg := Config{WorkDir: filepath.Join(tmp, "go_disk")}.WithDefaults()
		cfg.WorkDir = filepath.Join(tmp, "go_disk")
		goChunk := filepath.Join(tmp, "chunk_go_disk.jsonl")
		copyFile(t, chunkPath, goChunk)

		runtime.GC()
		allocBefore, _ := memStatsMB()
		start := time.Now()
		sr, err := convertChunkToShardGo(ctx, cfg, goChunk, "comments", "2011", "03", 0)
		dur := time.Since(start)
		allocAfter, _ := memStatsMB()
		if err != nil {
			t.Fatalf("Go disk failed: %v", err)
		}
		results = append(results, result{
			name: "Go (disk chunk, ZSTD ~11)", rows: sr.Rows,
			sizeMB: float64(sr.SizeBytes) / 1024 / 1024, dur: dur,
			allocMB: allocAfter - allocBefore,
		})
	}

	// --- Go engine (in-memory, SpeedBestCompression ~level 11) ---
	{
		cfg := Config{WorkDir: filepath.Join(tmp, "go_mem")}.WithDefaults()
		cfg.WorkDir = filepath.Join(tmp, "go_mem")

		// Read lines into memory.
		data, _ := os.ReadFile(chunkPath)
		var lines [][]byte
		start := 0
		for i, b := range data {
			if b == '\n' {
				lines = append(lines, append([]byte(nil), data[start:i]...))
				start = i + 1
			}
		}

		runtime.GC()
		allocBefore, _ := memStatsMB()
		tStart := time.Now()
		sr, err := convertChunkToShardGoMem(ctx, cfg, lines, "comments", "2011", "03", 0)
		dur := time.Since(tStart)
		allocAfter, _ := memStatsMB()
		if err != nil {
			t.Fatalf("Go mem failed: %v", err)
		}
		results = append(results, result{
			name: "Go (in-memory, ZSTD ~11)", rows: sr.Rows,
			sizeMB: float64(sr.SizeBytes) / 1024 / 1024, dur: dur,
			allocMB: allocAfter - allocBefore,
		})
	}

	// --- DuckDB engine (Parquet v2, ZSTD level 3) ---
	{
		cfg := Config{WorkDir: filepath.Join(tmp, "duckdb")}.WithDefaults()
		cfg.WorkDir = filepath.Join(tmp, "duckdb")
		duckChunk := filepath.Join(tmp, "chunk_duck.jsonl")
		copyFile(t, chunkPath, duckChunk)

		runtime.GC()
		allocBefore, _ := memStatsMB()
		start := time.Now()
		sr, err := convertChunkToShard(ctx, cfg, duckChunk, "comments", "2011", "03", 0)
		dur := time.Since(start)
		allocAfter, _ := memStatsMB()
		if err != nil {
			t.Fatalf("DuckDB failed: %v", err)
		}
		results = append(results, result{
			name: "DuckDB (Parquet v2, ZSTD 3)", rows: sr.Rows,
			sizeMB: float64(sr.SizeBytes) / 1024 / 1024, dur: dur,
			allocMB: allocAfter - allocBefore,
		})
	}

	// --- Print comparison table ---
	t.Log("")
	t.Log("╔══════════════════════════════════╤═════════╤══════════╤══════════╤═══════════╤═══════════╗")
	t.Log("║ Engine                           │ Rows    │ Size MB  │ Duration │ Rows/s    │ Alloc MB  ║")
	t.Log("╠══════════════════════════════════╪═════════╪══════════╪══════════╪═══════════╪═══════════╣")
	for _, r := range results {
		t.Logf("║ %-32s │ %7d │ %8.2f │ %8s │ %9.0f │ %9.1f ║",
			r.name, r.rows, r.sizeMB, r.dur.Round(time.Millisecond),
			float64(r.rows)/r.dur.Seconds(), r.allocMB)
	}
	t.Log("╚══════════════════════════════════╧═════════╧══════════╧══════════╧═══════════╧═══════════╝")

	// Validate all produce the same row count.
	for i := 1; i < len(results); i++ {
		if results[i].rows != results[0].rows {
			t.Errorf("row count mismatch: %s=%d vs %s=%d",
				results[0].name, results[0].rows, results[i].name, results[i].rows)
		}
	}
}

// TestGoParquetSubmissions verifies the submission schema path works.
func TestGoParquetSubmissions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	tmp := t.TempDir()
	chunkPath := filepath.Join(tmp, "chunk_0000.jsonl")
	const nLines = 10_000

	f, err := os.Create(chunkPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < nLines; i++ {
		row := map[string]any{
			"id":              fmt.Sprintf("sub%d", i),
			"author":          fmt.Sprintf("poster_%d", i%500),
			"subreddit":       "test_sub",
			"title":           fmt.Sprintf("Post title number %d", i),
			"selftext":        fmt.Sprintf("Body of submission %d with some content.", i),
			"score":           i % 1000,
			"created_utc":     1300000000 + i,
			"num_comments":    i % 50,
			"url":             fmt.Sprintf("https://reddit.com/r/test/%d", i),
			"over_18":         i%10 == 0,
			"link_flair_text": nil,
			"author_flair_text": nil,
		}
		b, _ := json.Marshal(row)
		f.Write(b)
		f.Write([]byte{'\n'})
	}
	f.Close()

	cfg := Config{WorkDir: tmp}.WithDefaults()
	cfg.WorkDir = tmp
	ctx := context.Background()

	// Test disk path.
	goChunk := filepath.Join(tmp, "chunk_go.jsonl")
	copyFile(t, chunkPath, goChunk)
	goCfg := cfg
	goCfg.WorkDir = filepath.Join(tmp, "go_output")
	goResult, err := convertChunkToShardGo(ctx, goCfg, goChunk, "submissions", "2011", "03", 0)
	if err != nil {
		t.Fatalf("Go submissions (disk) failed: %v", err)
	}
	t.Logf("Go submissions (disk): %d rows, %.2f MB", goResult.Rows, float64(goResult.SizeBytes)/1024/1024)

	// Test in-memory path.
	data, _ := os.ReadFile(chunkPath)
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, append([]byte(nil), data[start:i]...))
			start = i + 1
		}
	}
	memCfg := cfg
	memCfg.WorkDir = filepath.Join(tmp, "go_mem_output")
	memResult, err := convertChunkToShardGoMem(ctx, memCfg, lines, "submissions", "2011", "03", 0)
	if err != nil {
		t.Fatalf("Go submissions (mem) failed: %v", err)
	}
	t.Logf("Go submissions (mem):  %d rows, %.2f MB", memResult.Rows, float64(memResult.SizeBytes)/1024/1024)

	if goResult.Rows != nLines || memResult.Rows != nLines {
		t.Errorf("expected %d rows, got disk=%d mem=%d", nLines, goResult.Rows, memResult.Rows)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
