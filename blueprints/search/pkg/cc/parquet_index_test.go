package cc

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func TestParquetSubsetFromPath(t *testing.T) {
	path := "cc-index/table/cc-main/warc/crawl=CC-MAIN-2026-08/subset=crawldiagnostics/part-00000.parquet"
	if got := ParquetSubsetFromPath(path); got != "crawldiagnostics" {
		t.Fatalf("ParquetSubsetFromPath() = %q, want %q", got, "crawldiagnostics")
	}
}

func TestLocalParquetPathForRemotePreservesHivePartitions(t *testing.T) {
	tmp := t.TempDir()
	cfg := DefaultConfig()
	cfg.DataDir = tmp
	cfg.CrawlID = "CC-MAIN-TEST"

	remote := "cc-index/table/cc-main/warc/crawl=CC-MAIN-TEST/subset=warc/part-00000-test.parquet"
	local := LocalParquetPathForRemote(cfg, remote)
	wantSuffix := filepath.Join("crawl=CC-MAIN-TEST", "subset=warc", "part-00000-test.parquet")
	if !strings.HasSuffix(local, wantSuffix) {
		t.Fatalf("local path %q does not preserve hive partitions (want suffix %q)", local, wantSuffix)
	}
}

func TestImportParquetPathsWithProgressCreatesPerFileDuckDBAndCatalog(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	cfg := DefaultConfig()
	cfg.DataDir = tmp
	cfg.CrawlID = "CC-MAIN-TEST"

	remote := "cc-index/table/cc-main/warc/crawl=CC-MAIN-TEST/subset=warc/part-00000-localtest.parquet"
	parquetPath := LocalParquetPathForRemote(cfg, remote)
	if err := os.MkdirAll(filepath.Dir(parquetPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := writeTestParquet(parquetPath); err != nil {
		t.Fatalf("writeTestParquet: %v", err)
	}

	rowCount, err := ImportParquetPathsWithProgress(ctx, cfg, []string{parquetPath}, nil)
	if err != nil {
		t.Fatalf("ImportParquetPathsWithProgress: %v", err)
	}
	if rowCount != 2 {
		t.Fatalf("rowCount = %d, want 2", rowCount)
	}

	shardDBPath, err := IndexShardDBPathForParquet(cfg, parquetPath)
	if err != nil {
		t.Fatalf("IndexShardDBPathForParquet: %v", err)
	}
	if _, err := os.Stat(shardDBPath); err != nil {
		t.Fatalf("shard db missing: %v", err)
	}

	shardDB, err := sql.Open("duckdb", shardDBPath+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open shard db: %v", err)
	}
	defer shardDB.Close()

	cols := mustDescribeColumns(t, ctx, shardDB, "ccindex")
	for _, want := range []string{"custom_extra", "custom_num", "filename", "crawl", "subset"} {
		if !cols[want] {
			t.Fatalf("shard db missing imported column %q (columns=%v)", want, mapKeys(cols))
		}
	}

	catalogDB, err := sql.Open("duckdb", cfg.IndexDBPath()+"?access_mode=read_only")
	if err != nil {
		t.Fatalf("open catalog db: %v", err)
	}
	defer catalogDB.Close()

	var imports int
	if err := catalogDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM parquet_imports").Scan(&imports); err != nil {
		t.Fatalf("count parquet_imports: %v", err)
	}
	if imports != 1 {
		t.Fatalf("parquet_imports count = %d, want 1", imports)
	}

	var catalogRows int64
	if err := catalogDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM ccindex").Scan(&catalogRows); err != nil {
		t.Fatalf("count catalog ccindex rows: %v", err)
	}
	if catalogRows != 2 {
		t.Fatalf("catalog ccindex row count = %d, want 2", catalogRows)
	}
}

func TestLiveCCMain202608FirstParquetSampleImport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live Common Crawl import test in short mode")
	}
	if os.Getenv("CC_LIVE_IMPORT_TEST") == "" {
		t.Skip("set CC_LIVE_IMPORT_TEST=1 to run live Common Crawl sample import test")
	}

	ctx := context.Background()
	cfg := DefaultConfig()
	cfg.DataDir = t.TempDir()
	cfg.CrawlID = "CC-MAIN-2026-08"
	cfg.IndexWorkers = 1

	client := NewClient(cfg.BaseURL, 4)

	t.Log("loading parquet manifest for CC-MAIN-2026-08")
	paths, err := ParquetManifest(ctx, client, cfg)
	if err != nil {
		t.Fatalf("ParquetManifest: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("ParquetManifest returned no paths")
	}
	first := paths[0]
	t.Logf("first parquet manifest entry: %s", first)
	if !strings.Contains(first, "crawl=CC-MAIN-2026-08") {
		t.Fatalf("first path %q does not belong to CC-MAIN-2026-08", first)
	}

	t.Log("downloading first parquet file (manifest index 0)")
	localPath, err := DownloadManifestParquetFile(ctx, client, cfg, 0, nil)
	if err != nil {
		t.Fatalf("DownloadManifestParquetFile(0): %v", err)
	}
	if _, err := os.Stat(localPath); err != nil {
		t.Fatalf("downloaded parquet missing: %v", err)
	}

	t.Log("importing downloaded parquet to per-file DuckDB + catalog")
	rowCount, err := ImportParquetPathsWithProgress(ctx, cfg, []string{localPath}, nil)
	if err != nil {
		t.Fatalf("ImportParquetPathsWithProgress: %v", err)
	}
	if rowCount <= 0 {
		t.Fatalf("expected >0 rows after import, got %d", rowCount)
	}
}

func writeTestParquet(parquetPath string) error {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return err
	}
	defer db.Close()

	query := fmt.Sprintf(`
COPY (
	SELECT
		'https://example.com/' AS url,
		'crawl-data/CC-MAIN-TEST/segments/1.warc.gz' AS warc_filename,
		1::BIGINT AS warc_record_offset,
		100::BIGINT AS warc_record_length,
		200::INTEGER AS fetch_status,
		'text/html' AS content_mime_detected,
		'eng' AS content_languages,
		'example.com' AS url_host_registered_domain,
		'example.com' AS url_host_name,
		'com' AS url_host_tld,
		'alpha' AS custom_extra,
		7::INTEGER AS custom_num
	UNION ALL
	SELECT
		'https://example.org/' AS url,
		'crawl-data/CC-MAIN-TEST/segments/2.warc.gz' AS warc_filename,
		2::BIGINT AS warc_record_offset,
		200::BIGINT AS warc_record_length,
		404::INTEGER AS fetch_status,
		'text/plain' AS content_mime_detected,
		'eng,deu' AS content_languages,
		'example.org' AS url_host_registered_domain,
		'example.org' AS url_host_name,
		'org' AS url_host_tld,
		'beta' AS custom_extra,
		8::INTEGER AS custom_num
) TO %s (FORMAT PARQUET)`,
		duckQuoteString(filepath.ToSlash(parquetPath)),
	)
	_, err = db.Exec(query)
	return err
}

func mustDescribeColumns(t *testing.T, ctx context.Context, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.QueryContext(ctx, fmt.Sprintf("DESCRIBE SELECT * FROM %s", table))
	if err != nil {
		t.Fatalf("DESCRIBE %s: %v", table, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var name, typ string
		var nullable, key, def, extra sql.NullString
		if err := rows.Scan(&name, &typ, &nullable, &key, &def, &extra); err != nil {
			t.Fatalf("scan DESCRIBE row: %v", err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("DESCRIBE rows error: %v", err)
	}
	return cols
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
