package cli

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
	"github.com/parquet-go/parquet-go"
)

func TestExportWARCMdShardToParquet(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "00000.md.warc.gz")
	outPath := filepath.Join(dir, "00000.parquet")

	f, err := os.Create(inPath)
	if err != nil {
		t.Fatalf("create input: %v", err)
	}
	gz := gzip.NewWriter(f)
	w := warcpkg.NewWriter(gz)
	rec := &warcpkg.Record{
		Header: warcpkg.Header{
			"WARC-Type":       warcpkg.TypeConversion,
			"WARC-Target-URI": "https://example.com/post",
			"WARC-Date":       "2026-03-13T00:00:00Z",
			"WARC-Record-ID":  "<urn:uuid:11111111-1111-1111-1111-111111111111>",
			"WARC-Refers-To":  "<urn:uuid:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa>",
			"Content-Type":    "text/markdown",
			"Content-Length":  "12",
			"X-Test":          "value",
		},
		Body: strings.NewReader("# Hello\nBody"),
	}
	if err := w.WriteRecord(rec); err != nil {
		t.Fatalf("write record: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	rows, err := exportWARCMdShardToParquet(inPath, outPath, 0)
	if err != nil {
		t.Fatalf("export shard: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row, got %d", rows)
	}

	rf, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("open parquet: %v", err)
	}
	defer rf.Close()
	info, err := rf.Stat()
	if err != nil {
		t.Fatalf("stat parquet: %v", err)
	}
	pf, err := parquet.OpenFile(rf, info.Size())
	if err != nil {
		t.Fatalf("open parquet file: %v", err)
	}
	pr := parquet.NewGenericReader[ccWARCExportRow](pf)
	defer pr.Close()

	var got [1]ccWARCExportRow
	n, err := pr.Read(got[:])
	if err != nil && err != io.EOF {
		t.Fatalf("read parquet rows: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row read, got %d", n)
	}
	if got[0].DocID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected doc_id: %q", got[0].DocID)
	}
	if got[0].Host != "example.com" {
		t.Fatalf("unexpected host: %q", got[0].Host)
	}
	if got[0].MarkdownBody != "# Hello\nBody" {
		t.Fatalf("unexpected markdown body: %q", got[0].MarkdownBody)
	}
	if !strings.Contains(got[0].WARCHeadersJSON, `"X-Test":"value"`) {
		t.Fatalf("expected headers json to contain X-Test, got %q", got[0].WARCHeadersJSON)
	}
}

func TestCCResolvePublishUploadFiles(t *testing.T) {
	repoRoot := t.TempDir()
	dataDir := filepath.Join(repoRoot, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	for _, name := range []string{"00000.parquet", "00002.parquet"} {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	all, err := ccResolvePublishUploadFiles(repoRoot, "all")
	if err != nil {
		t.Fatalf("resolve all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 files, got %d", len(all))
	}

	one, err := ccResolvePublishUploadFiles(repoRoot, "0")
	if err != nil {
		t.Fatalf("resolve file 0: %v", err)
	}
	if len(one) != 1 || one[0].PathInRepo != "data/00000.parquet" {
		t.Fatalf("unexpected single selection: %+v", one)
	}
}
