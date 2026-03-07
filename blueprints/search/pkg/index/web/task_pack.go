package web

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// PackTask packs markdown files into a distributable format.
// Supported formats:
//   - "warc_md": .md files → single .md.warc.gz (concatenated gzip WARC)
//   - "parquet": .md files → .parquet (ZSTD-compressed, doc_id+text schema)
//
// It is a self-contained core.Task with no dependency on Manager.
type PackTask struct {
	CrawlDir string   `json:"crawl_dir"`
	Paths    []string `json:"paths"`    // manifest paths
	Selected []int    `json:"selected"` // indices into Paths
	Format   string   `json:"format"`   // "warc_md" or "parquet"
}

// PackState is emitted during packing with per-file detail.
type PackState struct {
	FileIndex     int     `json:"file_index"`
	FileTotal     int     `json:"file_total"`
	WARCIndex     string  `json:"warc_index"`
	Format        string  `json:"format"`
	DocsProcessed int64   `json:"docs_processed"`
	DocsTotal     int64   `json:"docs_total"`
	BytesRead     int64   `json:"bytes_read"`
	BytesWritten  int64   `json:"bytes_written"`
	Progress      float64 `json:"progress"`
	DocsPerSec    float64 `json:"docs_per_sec,omitempty"`
}

// PackMetric is the final result after packing completes.
type PackMetric struct {
	Files   int           `json:"files"`
	Docs    int64         `json:"docs"`
	Bytes   int64         `json:"bytes_written"`
	Elapsed time.Duration `json:"elapsed_ns"`
}

// NewPackTask creates a pack task for the given format and WARC files.
func NewPackTask(crawlDir string, paths []string, selected []int, format string) *PackTask {
	if format == "" {
		format = "parquet"
	}
	return &PackTask{CrawlDir: crawlDir, Paths: paths, Selected: selected, Format: format}
}

func (t *PackTask) Run(ctx context.Context, emit func(*PackState)) (PackMetric, error) {
	start := time.Now()
	total := len(t.Selected)
	var totalDocs, totalBytes int64

	for i, idx := range t.Selected {
		if ctx.Err() != nil {
			return PackMetric{}, ctx.Err()
		}
		warcIdx := warcFileIndex(t.Paths[idx], idx)
		markdownDir := filepath.Join(t.CrawlDir, "markdown", warcIdx)
		if !fileExists(markdownDir) {
			return PackMetric{}, fmt.Errorf("markdown dir not found: %s", markdownDir)
		}

		outPath, err := packOutputPath(t.CrawlDir, t.Format, warcIdx)
		if err != nil {
			return PackMetric{}, err
		}

		progress := func(docs, docsTotal, bytesRead, bytesWritten int64) {
			totalDocs = docs
			totalBytes = bytesWritten
			emitPackProgress(emit, i, total, warcIdx, t.Format, docs, docsTotal, bytesRead, bytesWritten, start)
		}

		switch t.Format {
		case "warc_md":
			err = packToWARCMd(ctx, markdownDir, outPath, progress)
		case "parquet":
			err = packToParquet(ctx, markdownDir, outPath, progress)
		default:
			return PackMetric{}, fmt.Errorf("unknown format %q (valid: warc_md, parquet)", t.Format)
		}
		if err != nil {
			return PackMetric{}, fmt.Errorf("pack [%s] %s: %w", t.Format, warcIdx, err)
		}
	}

	return PackMetric{
		Files:   total,
		Docs:    totalDocs,
		Bytes:   totalBytes,
		Elapsed: time.Since(start),
	}, nil
}

// packOutputPath returns the output file path for a given format and WARC index.
func packOutputPath(crawlDir, format, warcIdx string) (string, error) {
	switch format {
	case "warc_md":
		return filepath.Join(crawlDir, "warc_md", warcIdx+".md.warc.gz"), nil
	case "parquet":
		return filepath.Join(crawlDir, "pack", "parquet", warcIdx+".parquet"), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}

// emitPackProgress emits a detailed pack state snapshot.
func emitPackProgress(emit func(*PackState), fileIdx, fileTotal int, warcIdx, format string,
	docs, docsTotal, bytesRead, bytesWritten int64, start time.Time) {
	if emit == nil {
		return
	}
	pct := phaseProgress(docs, docsTotal)
	overall := fileProgress(fileIdx, fileTotal, pct)
	var dps float64
	if elapsed := time.Since(start); elapsed > 0 && docs > 0 {
		dps = float64(docs) / elapsed.Seconds()
	}
	emit(&PackState{
		FileIndex:     fileIdx,
		FileTotal:     fileTotal,
		WARCIndex:     warcIdx,
		Format:        format,
		DocsProcessed: docs,
		DocsTotal:     docsTotal,
		BytesRead:     bytesRead,
		BytesWritten:  bytesWritten,
		Progress:      overall,
		DocsPerSec:    dps,
	})
}

// ── Pack to WARC-MD (.md.warc.gz) ────────────────────────────────────────────

// packToWARCMd walks a markdown directory and writes all .md files as WARC
// conversion records into a single concatenated-gzip file. Each record is its
// own gzip member for random-access reading.
func packToWARCMd(ctx context.Context, markdownDir, outPath string, progress func(docs, total, bytesRead, bytesWritten int64)) error {
	files, err := collectMarkdownFiles(markdownDir)
	if err != nil {
		return fmt.Errorf("walk %s: %w", markdownDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	total := int64(len(files))
	var docs, bytesRead, bytesWritten int64

	for _, mdFile := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		body, err := os.ReadFile(mdFile.path)
		if err != nil {
			continue
		}
		bytesRead += int64(len(body))

		// Each record in its own gzip member for random-access reads.
		gz, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
		if err != nil {
			return err
		}

		w := warcpkg.NewWriter(gz)
		recordID := "<urn:uuid:" + uuid.New().String() + ">"
		rec := &warcpkg.Record{
			Header: warcpkg.Header{
				"WARC-Type":      warcpkg.TypeConversion,
				"WARC-Record-ID": recordID,
				"Content-Length": strconv.FormatInt(int64(len(body)), 10),
				"Content-Type":   "text/markdown",
			},
			Body: bytes.NewReader(body),
		}
		if err := w.WriteRecord(rec); err != nil {
			gz.Close()
			return fmt.Errorf("write record: %w", err)
		}
		if err := w.Close(); err != nil {
			gz.Close()
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}

		docs++
		bytesWritten += int64(len(body))
		if progress != nil {
			progress(docs, total, bytesRead, bytesWritten)
		}
	}

	return nil
}

// ── Pack to Parquet ──────────────────────────────────────────────────────────

// parquetDoc is the schema for the pack parquet file.
type parquetDoc struct {
	DocID string `parquet:"doc_id"`
	Text  string `parquet:"text"`
}

const parquetRowGroupRows = 50_000

// packToParquet walks a markdown directory and writes all .md files to a
// ZSTD-compressed Parquet file with doc_id+text schema.
func packToParquet(ctx context.Context, markdownDir, outPath string, progress func(docs, total, bytesRead, bytesWritten int64)) error {
	files, err := collectMarkdownFiles(markdownDir)
	if err != nil {
		return fmt.Errorf("walk %s: %w", markdownDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}

	pw := parquet.NewGenericWriter[parquetDoc](f,
		parquet.Compression(&parquet.Zstd),
		parquet.MaxRowsPerRowGroup(parquetRowGroupRows),
		parquet.PageBufferSize(1*1024*1024),
	)

	total := int64(len(files))
	var docs, bytesRead int64
	batch := make([]parquetDoc, 0, 1000)

	for _, mdFile := range files {
		if ctx.Err() != nil {
			pw.Close()
			f.Close()
			os.Remove(outPath)
			return ctx.Err()
		}

		body, err := os.ReadFile(mdFile.path)
		if err != nil {
			continue
		}
		bytesRead += int64(len(body))

		text := sanitizeUTF8(string(body))
		batch = append(batch, parquetDoc{DocID: mdFile.docID, Text: text})

		if len(batch) >= 1000 {
			if _, err := pw.Write(batch); err != nil {
				pw.Close()
				f.Close()
				os.Remove(outPath)
				return fmt.Errorf("parquet write: %w", err)
			}
			batch = batch[:0]
		}

		docs++
		if progress != nil {
			progress(docs, total, bytesRead, 0)
		}
	}

	// Flush remaining batch.
	if len(batch) > 0 {
		if _, err := pw.Write(batch); err != nil {
			pw.Close()
			f.Close()
			os.Remove(outPath)
			return fmt.Errorf("parquet write: %w", err)
		}
	}

	if err := pw.Close(); err != nil {
		f.Close()
		os.Remove(outPath)
		return fmt.Errorf("parquet close: %w", err)
	}
	return f.Close()
}

// ── Shared helpers ───────────────────────────────────────────────────────────

// mdFileEntry holds a discovered markdown file's path and derived doc ID.
type mdFileEntry struct {
	path  string
	docID string
}

// collectMarkdownFiles walks a directory for .md files and returns entries
// with doc IDs derived from the filename (UUID or sanitized base).
func collectMarkdownFiles(dir string) ([]mdFileEntry, error) {
	var files []mdFileEntry
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		docID := docIDFromFilename(d.Name())
		files = append(files, mdFileEntry{path: path, docID: docID})
		return nil
	})
	return files, err
}

// docIDFromFilename extracts a doc ID from a markdown filename.
// "abc123.md" → "abc123", "abc123.md.gz" → "abc123".
func docIDFromFilename(name string) string {
	name = strings.TrimSuffix(name, ".gz")
	name = strings.TrimSuffix(name, ".md")
	return name
}

// sanitizeUTF8 replaces invalid UTF-8 sequences with the replacement character.
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "\uFFFD")
}
