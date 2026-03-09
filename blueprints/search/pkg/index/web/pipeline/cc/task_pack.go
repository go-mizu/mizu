package cc

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

	"github.com/go-mizu/mizu/blueprints/search/pkg/core"
	"github.com/go-mizu/mizu/blueprints/search/pkg/index/web/pipeline/util"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// Compile-time check.
var _ core.Task[PackState, PackMetric] = (*PackTask)(nil)

// PackTask packs markdown files into a distributable format.
// Supported formats: "warc_md", "parquet".
type PackTask struct {
	crawlDir string
	paths    []string
	selected []int
	format   string
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
	return &PackTask{crawlDir: crawlDir, paths: paths, selected: selected, format: format}
}

func (t *PackTask) Run(ctx context.Context, emit func(*PackState)) (PackMetric, error) {
	start := time.Now()
	total := len(t.selected)
	var totalDocs, totalBytes int64

	for i, idx := range t.selected {
		if ctx.Err() != nil {
			return PackMetric{}, ctx.Err()
		}
		warcIdx := util.WARCFileIndex(t.paths[idx], idx)
		markdownDir := filepath.Join(t.crawlDir, "markdown", warcIdx)
		if !util.FileExists(markdownDir) {
			return PackMetric{}, fmt.Errorf("markdown dir not found: %s", markdownDir)
		}

		outPath, err := packOutputPath(t.crawlDir, t.format, warcIdx)
		if err != nil {
			return PackMetric{}, err
		}

		progress := func(docs, docsTotal, bytesRead, bytesWritten int64) {
			totalDocs = docs
			totalBytes = bytesWritten
			emitPackProgress(emit, i, total, warcIdx, t.format, docs, docsTotal, bytesRead, bytesWritten, start)
		}

		switch t.format {
		case "warc_md":
			err = packToWARCMd(ctx, markdownDir, outPath, progress)
		case "parquet":
			err = packToParquet(ctx, markdownDir, outPath, progress)
		default:
			return PackMetric{}, fmt.Errorf("unknown format %q (valid: warc_md, parquet)", t.format)
		}
		if err != nil {
			return PackMetric{}, fmt.Errorf("pack [%s] %s: %w", t.format, warcIdx, err)
		}
	}

	return PackMetric{
		Files:   total,
		Docs:    totalDocs,
		Bytes:   totalBytes,
		Elapsed: time.Since(start),
	}, nil
}

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

func emitPackProgress(emit func(*PackState), fileIdx, fileTotal int, warcIdx, format string,
	docs, docsTotal, bytesRead, bytesWritten int64, start time.Time) {
	if emit == nil {
		return
	}
	pct := util.PhaseProgress(docs, docsTotal)
	overall := util.FileProgress(fileIdx, fileTotal, pct)
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

// ── Pack to WARC-MD ──────────────────────────────────────────────────────────

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

type parquetDoc struct {
	DocID string `parquet:"doc_id"`
	Text  string `parquet:"text"`
}

const parquetRowGroupRows = 50_000

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

type mdFileEntry struct {
	path  string
	docID string
}

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

func docIDFromFilename(name string) string {
	name = strings.TrimSuffix(name, ".gz")
	name = strings.TrimSuffix(name, ".md")
	return name
}

func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "\uFFFD")
}
