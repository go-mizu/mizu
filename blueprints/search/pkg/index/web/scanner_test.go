package web

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDataDir(t *testing.T) {
	// Create a temporary directory tree that mimics the crawl data layout:
	//   {crawlDir}/warc/          → WARC files
	//   {crawlDir}/markdown/{n}/  → markdown shards with .md files
	//   {crawlDir}/pack/{fmt}/    → packed bundles per format
	//   {crawlDir}/fts/{eng}/{n}/ → FTS index shards per engine
	root := t.TempDir()

	// ── warc/ ────────────────────────────────────────
	warcDir := filepath.Join(root, "warc")
	mustMkdir(t, warcDir)
	writeFile(t, filepath.Join(warcDir, "00000.warc.gz"), 1024*1024)     // 1 MB
	writeFile(t, filepath.Join(warcDir, "00001.warc.gz"), 2*1024*1024)   // 2 MB
	writeFile(t, filepath.Join(warcDir, "00002.warc.gz"), 512*1024)      // 512 KB

	// ── markdown/ ────────────────────────────────────
	md0 := filepath.Join(root, "markdown", "00000")
	md1 := filepath.Join(root, "markdown", "00001")
	mustMkdir(t, md0)
	mustMkdir(t, md1)
	// 3 docs in shard 00000, 2 in shard 00001
	writeFile(t, filepath.Join(md0, "aaa.md"), 500)
	writeFile(t, filepath.Join(md0, "bbb.md"), 600)
	writeFile(t, filepath.Join(md0, "ccc.md"), 700)
	writeFile(t, filepath.Join(md1, "ddd.md"), 400)
	writeFile(t, filepath.Join(md1, "eee.md"), 800)

	// ── pack/ ────────────────────────────────────────
	packPQ := filepath.Join(root, "pack", "parquet")
	packBin := filepath.Join(root, "pack", "bin")
	mustMkdir(t, packPQ)
	mustMkdir(t, packBin)
	writeFile(t, filepath.Join(packPQ, "00000.parquet"), 4096)
	writeFile(t, filepath.Join(packPQ, "00001.parquet"), 8192)
	writeFile(t, filepath.Join(packBin, "00000.bin"), 2048)

	// ── fts/ ─────────────────────────────────────────
	ftsBleve0 := filepath.Join(root, "fts", "bleve", "00000")
	ftsBleve1 := filepath.Join(root, "fts", "bleve", "00001")
	ftsTantivy0 := filepath.Join(root, "fts", "tantivy", "00000")
	mustMkdir(t, ftsBleve0)
	mustMkdir(t, ftsBleve1)
	mustMkdir(t, ftsTantivy0)
	writeFile(t, filepath.Join(ftsBleve0, "store.bolt"), 10000)
	writeFile(t, filepath.Join(ftsBleve0, "index.bolt"), 5000)
	writeFile(t, filepath.Join(ftsBleve1, "store.bolt"), 12000)
	writeFile(t, filepath.Join(ftsTantivy0, "meta.json"), 100)
	writeFile(t, filepath.Join(ftsTantivy0, "segments"), 30000)

	// ── Run scanner ──────────────────────────────────
	ds := ScanDataDir(root)

	// WARC assertions
	if ds.WARCCount != 3 {
		t.Errorf("WARCCount = %d, want 3", ds.WARCCount)
	}
	wantWARCSize := int64(1024*1024 + 2*1024*1024 + 512*1024)
	if ds.WARCTotalSize != wantWARCSize {
		t.Errorf("WARCTotalSize = %d, want %d", ds.WARCTotalSize, wantWARCSize)
	}

	// Markdown assertions
	if ds.MDShards != 2 {
		t.Errorf("MDShards = %d, want 2", ds.MDShards)
	}
	wantMDSize := int64(500 + 600 + 700 + 400 + 800)
	if ds.MDTotalSize != wantMDSize {
		t.Errorf("MDTotalSize = %d, want %d", ds.MDTotalSize, wantMDSize)
	}
	if ds.MDDocEstimate != 5 {
		t.Errorf("MDDocEstimate = %d, want 5", ds.MDDocEstimate)
	}

	// Pack assertions
	if len(ds.PackFormats) != 2 {
		t.Errorf("PackFormats has %d entries, want 2", len(ds.PackFormats))
	}
	if ds.PackFormats["parquet"] != 4096+8192 {
		t.Errorf("PackFormats[parquet] = %d, want %d", ds.PackFormats["parquet"], 4096+8192)
	}
	if ds.PackFormats["bin"] != 2048 {
		t.Errorf("PackFormats[bin] = %d, want %d", ds.PackFormats["bin"], 2048)
	}

	// FTS assertions
	if len(ds.FTSEngines) != 2 {
		t.Errorf("FTSEngines has %d entries, want 2", len(ds.FTSEngines))
	}
	if ds.FTSEngines["bleve"] != 10000+5000+12000 {
		t.Errorf("FTSEngines[bleve] = %d, want %d", ds.FTSEngines["bleve"], 10000+5000+12000)
	}
	if ds.FTSEngines["tantivy"] != 100+30000 {
		t.Errorf("FTSEngines[tantivy] = %d, want %d", ds.FTSEngines["tantivy"], 100+30000)
	}
	if ds.FTSShardCount["bleve"] != 2 {
		t.Errorf("FTSShardCount[bleve] = %d, want 2", ds.FTSShardCount["bleve"])
	}
	if ds.FTSShardCount["tantivy"] != 1 {
		t.Errorf("FTSShardCount[tantivy] = %d, want 1", ds.FTSShardCount["tantivy"])
	}

	// CrawlID should be empty (not derived from dir name by scanner).
	if ds.CrawlID != "" {
		t.Errorf("CrawlID = %q, want empty", ds.CrawlID)
	}
}

func TestScanDataDir_Empty(t *testing.T) {
	root := t.TempDir()
	ds := ScanDataDir(root)

	// All maps must be non-nil (initialized) for clean JSON marshaling.
	if ds.PackFormats == nil {
		t.Error("PackFormats is nil, want initialized empty map")
	}
	if ds.FTSEngines == nil {
		t.Error("FTSEngines is nil, want initialized empty map")
	}
	if ds.FTSShardCount == nil {
		t.Error("FTSShardCount is nil, want initialized empty map")
	}

	// All numeric fields should be zero.
	if ds.WARCCount != 0 {
		t.Errorf("WARCCount = %d, want 0", ds.WARCCount)
	}
	if ds.WARCTotalSize != 0 {
		t.Errorf("WARCTotalSize = %d, want 0", ds.WARCTotalSize)
	}
	if ds.MDShards != 0 {
		t.Errorf("MDShards = %d, want 0", ds.MDShards)
	}
	if ds.MDTotalSize != 0 {
		t.Errorf("MDTotalSize = %d, want 0", ds.MDTotalSize)
	}
	if ds.MDDocEstimate != 0 {
		t.Errorf("MDDocEstimate = %d, want 0", ds.MDDocEstimate)
	}

	// JSON marshaling should produce {} for empty maps, not null.
	data, err := json.Marshal(ds)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	for _, key := range []string{"pack_formats", "fts_engines", "fts_shard_count"} {
		v, ok := raw[key]
		if !ok {
			t.Errorf("JSON missing key %q", key)
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			t.Errorf("JSON key %q is not an object: %T", key, v)
			continue
		}
		if len(m) != 0 {
			t.Errorf("JSON key %q should be empty object, got %v", key, m)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{2411724, "2.3 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}
	for _, tc := range tests {
		got := FormatBytes(tc.input)
		if got != tc.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ── Test Helpers ─────────────────────────────────────────────────────────

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

// writeFile creates a file with exactly `size` bytes of zero data.
func writeFile(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%s): %v", path, err)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		t.Fatalf("Truncate(%s, %d): %v", path, size, err)
	}
	f.Close()
}
