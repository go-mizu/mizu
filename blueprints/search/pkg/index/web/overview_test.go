package web

import (
	"path/filepath"
	"testing"
)

func TestBuildOverviewResponse(t *testing.T) {
	root := t.TempDir()

	// Create data layout: 2 WARCs, 1 warc_md, 1 pack/parquet, 1 fts/dahlia
	mustMkdir(t, filepath.Join(root, "warc"))
	writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00000.warc.gz"), 1024*1024)
	writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00001.warc.gz"), 2048*1024)

	mustMkdir(t, filepath.Join(root, "warc_md"))
	writeFile(t, filepath.Join(root, "warc_md", "00000.md.warc.gz"), 512*1024)

	mustMkdir(t, filepath.Join(root, "pack", "parquet"))
	writeFile(t, filepath.Join(root, "pack", "parquet", "00000.parquet"), 256*1024)

	mustMkdir(t, filepath.Join(root, "fts", "dahlia", "00000"))
	writeFile(t, filepath.Join(root, "fts", "dahlia", "00000", "seg.bin"), 128*1024)

	resp := buildOverviewResponse("CC-TEST-2026", root, 1000, nil)

	// Manifest
	if resp.Manifest.TotalWARCs != 1000 {
		t.Fatalf("manifest total_warcs: got %d, want 1000", resp.Manifest.TotalWARCs)
	}

	// Downloaded
	if resp.Downloaded.Count != 2 {
		t.Fatalf("downloaded count: got %d, want 2", resp.Downloaded.Count)
	}
	if resp.Downloaded.SizeBytes != 3*1024*1024 {
		t.Fatalf("downloaded size: got %d", resp.Downloaded.SizeBytes)
	}
	if resp.Downloaded.AvgWARCBytes != (3*1024*1024)/2 {
		t.Fatalf("downloaded avg: got %d", resp.Downloaded.AvgWARCBytes)
	}

	// Markdown
	if resp.Markdown.Count != 1 {
		t.Fatalf("markdown count: got %d, want 1", resp.Markdown.Count)
	}
	if resp.Markdown.SizeBytes != 512*1024 {
		t.Fatalf("markdown size: got %d", resp.Markdown.SizeBytes)
	}

	// Pack
	if resp.Pack.Count != 1 {
		t.Fatalf("pack count: got %d, want 1", resp.Pack.Count)
	}
	if resp.Pack.ParquetBytes != 256*1024 {
		t.Fatalf("parquet bytes: got %d", resp.Pack.ParquetBytes)
	}

	// Indexed
	if resp.Indexed.Count != 1 {
		t.Fatalf("indexed count: got %d, want 1", resp.Indexed.Count)
	}
	if resp.Indexed.DahliaShards != 1 {
		t.Fatalf("dahlia shards: got %d", resp.Indexed.DahliaShards)
	}

	// Storage
	if resp.Storage.CrawlBytes <= 0 {
		t.Fatal("expected positive crawl bytes")
	}

	// System
	if resp.System.Goroutines <= 0 {
		t.Fatal("expected positive goroutines")
	}
	if resp.System.GoVersion == "" {
		t.Fatal("expected go version")
	}
	if resp.System.PID <= 0 {
		t.Fatal("expected positive PID")
	}
}

func TestBuildOverviewResponse_EmptyDir(t *testing.T) {
	root := t.TempDir()
	resp := buildOverviewResponse("CC-TEST-2026", root, 0, nil)

	if resp.CrawlID != "CC-TEST-2026" {
		t.Fatalf("crawl_id: got %q", resp.CrawlID)
	}
	if resp.Downloaded.Count != 0 {
		t.Fatalf("downloaded count: got %d, want 0", resp.Downloaded.Count)
	}
	if resp.Manifest.TotalWARCs != 0 {
		t.Fatalf("manifest total: got %d, want 0", resp.Manifest.TotalWARCs)
	}
}

func TestBuildOverviewResponse_ProjectedSize(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "warc"))
	// 1 WARC of 1 GB, manifest says 100K WARCs
	writeFile(t, filepath.Join(root, "warc", "CC-MAIN-x-00000.warc.gz"), 1024*1024*1024)

	resp := buildOverviewResponse("CC-TEST-2026", root, 100000, nil)

	// projected = avg_warc * total = 1GB * 100K = 100 TB
	expected := int64(1024) * 1024 * 1024 * 100000
	if resp.Storage.ProjectedFullBytes != expected {
		t.Fatalf("projected: got %d, want %d", resp.Storage.ProjectedFullBytes, expected)
	}
}
