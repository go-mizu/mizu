package cc_v2

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatsCSVRoundtrip(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "stats.csv")

	rows := []StatsRow{
		{CrawlID: "CC-MAIN-2026-12", FileIdx: 0, Rows: 1000, HTMLBytes: 5000, MDBytes: 500, PqBytes: 200, CreatedAt: "2026-03-22T00:00:00Z"},
		{CrawlID: "CC-MAIN-2026-12", FileIdx: 1, Rows: 2000, HTMLBytes: 10000, MDBytes: 1000, PqBytes: 400, CreatedAt: "2026-03-22T01:00:00Z", DurDlS: 30, DurPackS: 45},
	}

	if err := writeStatsCSV(csvPath, rows); err != nil {
		t.Fatal(err)
	}

	read, err := readStatsCSV(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != 2 {
		t.Fatalf("got %d rows, want 2", len(read))
	}
	if read[0].Rows != 1000 || read[1].Rows != 2000 {
		t.Fatalf("rows mismatch: %d, %d", read[0].Rows, read[1].Rows)
	}
	if read[1].DurDlS != 30 || read[1].DurPackS != 45 {
		t.Fatalf("timing mismatch: dl=%d pack=%d", read[1].DurDlS, read[1].DurPackS)
	}
}

func TestUpsertStats(t *testing.T) {
	rows := []StatsRow{
		{CrawlID: "CC-MAIN-2026-12", FileIdx: 0, Rows: 1000},
		{CrawlID: "CC-MAIN-2026-12", FileIdx: 1, Rows: 2000},
	}

	// Upsert existing row — should update.
	rows = upsertStats(rows, StatsRow{CrawlID: "CC-MAIN-2026-12", FileIdx: 0, Rows: 1500})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Rows != 1500 {
		t.Fatalf("expected 1500, got %d", rows[0].Rows)
	}

	// Upsert new row — should append.
	rows = upsertStats(rows, StatsRow{CrawlID: "CC-MAIN-2026-12", FileIdx: 2, Rows: 3000})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestMergeStatsFromRemote(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "stats.csv")

	// Write local stats.
	local := []StatsRow{
		{CrawlID: "CC-MAIN-2026-12", FileIdx: 0, Rows: 1000},
	}
	writeStatsCSV(localPath, local)

	// Remote has local row 0 (should not overwrite) and new row 1.
	remoteCSV := `crawl_id,file_idx,rows,html_bytes,md_bytes,parquet_bytes,created_at,dur_download_s,dur_convert_s,dur_publish_s,peak_rss_mb
CC-MAIN-2026-12,0,999,0,0,0,,,,,
CC-MAIN-2026-12,1,2000,0,0,0,,,,,
`
	mergeStatsFromRemote(localPath, []byte(remoteCSV), "CC-MAIN-2026-12")

	merged, err := readStatsCSV(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(merged))
	}
	// Row 0 should keep local value (local wins).
	if merged[0].Rows != 1000 {
		t.Fatalf("row 0 should be local (1000), got %d", merged[0].Rows)
	}
	// Row 1 should be from remote.
	if merged[1].Rows != 2000 {
		t.Fatalf("row 1 should be remote (2000), got %d", merged[1].Rows)
	}
}

func TestParseFileSelector(t *testing.T) {
	tests := []struct {
		input   string
		want    []int
		wantErr bool
	}{
		{"0", []int{0}, false},
		{"0-3", []int{0, 1, 2, 3}, false},
		{"1,3,5", []int{1, 3, 5}, false},
		{"0-2,5", []int{0, 1, 2, 5}, false},
		{"all", nil, false},
		{"", nil, false},
		{"abc", nil, true},
	}
	for _, tt := range tests {
		got, err := ParseFileSelector(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFileSelector(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if tt.wantErr {
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("ParseFileSelector(%q): got %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseFileSelector(%q)[%d]: got %d, want %d", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestReadStatsCSVMissing(t *testing.T) {
	rows, err := readStatsCSV("/nonexistent/path/stats.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows for missing file, got %d", len(rows))
	}
}

func TestGenerateREADME(t *testing.T) {
	rows := []StatsRow{
		{CrawlID: "CC-MAIN-2026-12", FileIdx: 0, Rows: 1000, HTMLBytes: 5000, MDBytes: 500, PqBytes: 200},
	}
	readme := generateREADME("CC-MAIN-2026-12", rows, 1)
	if len(readme) == 0 {
		t.Fatal("README should not be empty")
	}
	// Should contain crawl ID.
	if !contains(readme, "CC-MAIN-2026-12") {
		t.Fatal("README should contain crawl ID")
	}
	// Should contain shard count.
	if !contains(readme, "1") {
		t.Fatal("README should contain shard count")
	}

	// When committed > csv shards, should use committed count and scale estimates.
	readme2 := generateREADME("CC-MAIN-2026-12", rows, 100)
	if !contains(readme2, "| Shards | 100 |") {
		t.Fatal("README should use committed count (100), not csv count (1)")
	}
	// Scaled docs should have ~ prefix.
	if !contains(readme2, "~") {
		t.Fatal("README should use ~ prefix for estimated docs")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr) >= 0
}

func searchString(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func init() {
	// Suppress any test output noise.
	_ = os.Setenv("REDIS_PASSWORD", "")
}
