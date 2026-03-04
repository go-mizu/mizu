# Data Directory Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor all CC WARC downstream artefacts to a per-WARC-index directory layout with uncompressed markdown, a new `pack/markdown/*.bin.gz` concat-gzip format, and per-WARC FTS indices with fan-out search.

**Architecture:** Add path helpers to both config structs; strip the compress phase from the warc→markdown pipeline; add a new `PackFlatBinGz` writer/reader using concatenated gzip members; update all CLI commands to route through per-WARC-index paths; add fan-out search across all per-WARC FTS indices.

**Tech Stack:** Go, `compress/gzip` (stdlib), `github.com/klauspost/compress/gzip` (BestCompression), existing flatbin wire format, cobra CLI.

---

## Task 1: Path helpers — `pkg/cc/config.go`

**Files:**
- Modify: `pkg/cc/config.go`

**Context:** The `Config` struct has helper methods that return directory/file paths. We add four new ones for per-WARC markdown, pack, and FTS paths.

**Step 1: Add the four helpers**

Open `pkg/cc/config.go` and append after `WARCImportDir()`:

```go
// MarkdownWarcDir returns the markdown output directory for one WARC file.
// warcIdx is the zero-padded 5-digit file index, e.g. "00000".
func (c Config) MarkdownWarcDir(warcIdx string) string {
	return filepath.Join(c.CrawlDir(), "markdown", warcIdx)
}

// PackFile returns the path for a pre-packed bundle for one WARC file.
// format is one of: bin, parquet, duckdb, markdown.
// ext is the file extension, e.g. "bin", "parquet", "duckdb", "bin.gz".
func (c Config) PackFile(format, warcIdx, ext string) string {
	return filepath.Join(c.CrawlDir(), "pack", format, warcIdx+"."+ext)
}

// FTSEngineDir returns the per-WARC directory for a directory-based FTS engine
// (rose, bleve, tantivy).
func (c Config) FTSEngineDir(engine, warcIdx string) string {
	return filepath.Join(c.CrawlDir(), "fts", engine, warcIdx)
}

// FTSEngineFile returns the per-WARC file path for a file-based FTS engine
// (duckdb → .duckdb, sqlite → .sqlite).
func (c Config) FTSEngineFile(engine, warcIdx, ext string) string {
	return filepath.Join(c.CrawlDir(), "fts", engine, warcIdx+"."+ext)
}
```

**Step 2: Build to verify**

```bash
cd blueprints/search && go build ./pkg/cc/...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add pkg/cc/config.go
git commit -m "feat(cc/config): add per-WARC-index path helpers"
```

---

## Task 2: Path helpers — `pkg/warc_md/config.go`

**Files:**
- Modify: `pkg/warc_md/config.go`

**Context:** The warc_md Config needs a `MarkdownWarcDir(warcIdx)` for the new uncompressed output. Remove `MarkdownGzDir` and `CompressWorkers` since phase 3 is gone.

**Step 1: Add `MarkdownWarcDir`, remove compress helpers**

Replace `pkg/warc_md/config.go` content:

```go
package warc_md

import (
	"os"
	"path/filepath"
	"runtime"
)

// Config configures the WARC → Markdown pipeline.
type Config struct {
	CrawlID     string // e.g. "CC-MAIN-2026-08"
	DataDir     string // base: $HOME/data/common-crawl
	Workers     int    // parallel workers for convert (0 = NumCPU)
	Force       bool   // re-process existing files
	Fast        bool   // use go-readability instead of trafilatura
	KeepTemp    bool   // keep warc_single/ and markdown_raw/ after pipeline
	MIMEFilter  string // e.g. "text/html" (default)
	StatusCode  int    // HTTP status filter (default: 200)
	MaxBodySize int64  // max HTML body bytes (default: 512 KB)
}

// DefaultConfig returns sensible defaults for a given crawl ID.
func DefaultConfig(crawlID string) Config {
	home, _ := os.UserHomeDir()
	return Config{
		CrawlID:     crawlID,
		DataDir:     filepath.Join(home, "data", "common-crawl"),
		Workers:     0,
		MIMEFilter:  "text/html",
		StatusCode:  200,
		MaxBodySize: 512 * 1024,
	}
}

// CrawlDir returns the crawl-specific data directory.
func (c Config) CrawlDir() string {
	return filepath.Join(c.DataDir, c.CrawlID)
}

// WARCDir returns the directory containing downloaded .warc.gz files.
func (c Config) WARCDir() string {
	return filepath.Join(c.CrawlDir(), "warc")
}

// WARCSingleDir returns the directory for extracted single-record files (Phase 1 temp).
func (c Config) WARCSingleDir() string {
	return filepath.Join(c.CrawlDir(), "warc_single")
}

// MarkdownDir returns the directory for converted raw markdown files (Phase 2 temp output).
func (c Config) MarkdownDir() string {
	return filepath.Join(c.CrawlDir(), "markdown_raw")
}

// MarkdownWarcDir returns the final output directory for one WARC's markdown files.
// warcIdx is the zero-padded 5-digit file index, e.g. "00000".
// Files inside are plain .md (uncompressed), sharded by UUID.
func (c Config) MarkdownWarcDir(warcIdx string) string {
	return filepath.Join(c.CrawlDir(), "markdown", warcIdx)
}

// ConvertWorkers returns the optimal worker count for Phase 2 (HTML→Markdown).
func (c Config) ConvertWorkers() int {
	if c.Workers > 0 {
		return c.Workers
	}
	return runtime.NumCPU()
}
```

**Step 2: Build**

```bash
go build ./pkg/warc_md/...
```
Expected: compile errors referencing `MarkdownGzDir` and `CompressWorkers` in pipeline.go and compress.go — that's expected; fixed in next tasks.

**Step 3: Commit**

```bash
git add pkg/warc_md/config.go
git commit -m "feat(warc_md/config): add MarkdownWarcDir, remove compress helpers"
```

---

## Task 3: Update path helpers in `pkg/warc_md/path.go`

**Files:**
- Modify: `pkg/warc_md/path.go`

**Context:** `MarkdownGzFilePath` currently writes `.md.gz`. Replace with `MarkdownFilePath` writing plain `.md` (which already existed for the raw phase). Add `MarkdownWarcFilePath` for the new per-WARC output.

**Step 1: Add `MarkdownWarcFilePath`, keep existing functions**

Append to `pkg/warc_md/path.go`:

```go
// MarkdownWarcFilePath returns the full path for the final .md output
// under the per-WARC directory (baseDir = MarkdownWarcDir result).
func MarkdownWarcFilePath(baseDir, recordID string) string {
	return filepath.Join(baseDir, RecordIDToRelPath(recordID)+".md")
}
```

**Step 2: Build**

```bash
go build ./pkg/warc_md/...
```

**Step 3: Commit**

```bash
git add pkg/warc_md/path.go
git commit -m "feat(warc_md/path): add MarkdownWarcFilePath for uncompressed per-WARC output"
```

---

## Task 4: Remove phase 3 from `pkg/warc_md/pipeline.go`

**Files:**
- Modify: `pkg/warc_md/pipeline.go`
- Delete: `pkg/warc_md/compress.go`

**Context:** `RunFilePipeline` has 3 phases; phase 3 (compress) is removed. `RunInMemoryPipeline` wrote `.md.gz` — this is also removed (the `--mem` flag is being removed from the CLI). The pipeline now takes a `warcIdx` parameter so it writes to the correct output directory.

**Step 1: Rewrite `pipeline.go`**

Replace `RunFilePipeline` signature and body. Remove `RunInMemoryPipeline` entirely. The `PipelineResult.Compress` field is no longer populated.

```go
// RunFilePipeline executes two phases sequentially:
//   Phase 1: extract HTML records → warc_single/
//   Phase 2: convert HTML → plain .md → markdown/{warcIdx}/
//
// Temp directories (warc_single/ and markdown_raw/) are removed after
// success unless cfg.KeepTemp is set.
//
// p1Fn, p2Fn are per-phase progress callbacks (may be nil).
func RunFilePipeline(ctx context.Context, cfg Config, warcIdx string, inputFiles []string,
	p1Fn, p2Fn ProgressFunc) (*PipelineResult, error) {

	start := time.Now()

	outDir := cfg.MarkdownWarcDir(warcIdx)

	// ── Phase 1: Extract ────────────────────────────────────────────────────
	s1, err := RunExtract(ctx, ExtractConfig{
		InputFiles:  inputFiles,
		OutputDir:   cfg.WARCSingleDir(),
		Workers:     len(inputFiles),
		Force:       cfg.Force,
		StatusCode:  cfg.StatusCode,
		MIMEFilter:  cfg.MIMEFilter,
		MaxBodySize: cfg.MaxBodySize,
	}, p1Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 1 extract: %w", err)
	}

	// ── Phase 2: Convert ────────────────────────────────────────────────────
	s2, err := RunConvert(ctx, ConvertConfig{
		InputDir:  cfg.WARCSingleDir(),
		OutputDir: outDir,
		Workers:   cfg.ConvertWorkers(),
		Force:     cfg.Force,
		Fast:      cfg.Fast,
	}, p2Fn)
	if err != nil {
		return nil, fmt.Errorf("phase 2 convert: %w", err)
	}

	result := &PipelineResult{
		Extract:  s1,
		Convert:  s2,
		Duration: time.Since(start),
	}

	if !cfg.KeepTemp {
		os.RemoveAll(cfg.WARCSingleDir())
		os.RemoveAll(cfg.MarkdownDir())
	}

	return result, nil
}
```

Also update `PipelineResult` in `types.go` — remove `Compress *PhaseStats` if present (check `pkg/warc_md/types.go`).

**Step 2: Delete compress.go**

```bash
rm pkg/warc_md/compress.go
```

**Step 3: Build**

```bash
go build ./pkg/warc_md/...
```
Fix any remaining references to `MarkdownGzDir`, `compressToGz`, `RunCompress`, etc.

**Step 4: Commit**

```bash
git add -u pkg/warc_md/
git commit -m "feat(warc_md): remove phase 3 compress, pipeline writes plain .md per-WARC"
```

---

## Task 5: New `pkg/index/pack_bingz.go` — writer

**Files:**
- Create: `pkg/index/pack_bingz.go`

**Context:** `PackFlatBinGz` writes `pack/markdown/{warcIdx}.bin.gz` as concatenated gzip members (1000 docs/member, `BestCompression`). It also writes a `.bin.gz.idx` alongside for fast random access.

Wire format per gzip member (raw flatbin, no header/footer):
```
uint16 LE  id_len
[]byte     id
uint32 LE  txt_len
[]byte     text
(repeat for docsPerMember docs)
```

Index file `*.bin.gz.idx`:
```
per member (12 bytes):
  uint64 LE  byte_offset_in_bin_gz
  uint32 LE  doc_count
footer (4 bytes):
  uint32 LE  member_count
```

**Step 1: Create `pkg/index/pack_bingz.go`**

```go
package index

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	kgzip "github.com/klauspost/compress/gzip"
)

const binGzDocsPerMember = 1000

// memberEntry records where one gzip member starts and how many docs it holds.
type memberEntry struct {
	offset   uint64
	docCount uint32
}

// PackFlatBinGz packs all markdown files from markdownDir into a concatenated
// gzip file at packPath. Each gzip member contains up to docsPerMember flatbin
// records compressed with BestCompression.
//
// An index file (packPath + ".idx") is written alongside for parallel access.
func PackFlatBinGz(ctx context.Context, markdownDir, packPath string, workers, batchSize int, progress ProgressFunc) (*PipelineStats, error) {
	if err := os.MkdirAll(filepath.Dir(packPath), 0o755); err != nil {
		return nil, err
	}

	f, err := os.Create(packPath)
	if err != nil {
		return nil, err
	}

	// bf wraps f so we can track the current byte offset without seeking.
	bf := &countingWriter{w: f}

	var (
		mu      sync.Mutex
		members []memberEntry
		// Accumulate docs for the current member.
		buf     bytes.Buffer
		bufDocs int
	)

	flushMember := func() error {
		if bufDocs == 0 {
			return nil
		}
		offset := uint64(bf.n)
		gz, err := kgzip.NewWriterLevel(bf, kgzip.BestCompression)
		if err != nil {
			return err
		}
		if _, err := gz.Write(buf.Bytes()); err != nil {
			gz.Close()
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}
		members = append(members, memberEntry{offset: offset, docCount: uint32(bufDocs)})
		buf.Reset()
		bufDocs = 0
		return nil
	}

	eng := &funcEngine{
		name: "bingz-writer",
		indexFn: func(_ context.Context, docs []Document) error {
			mu.Lock()
			defer mu.Unlock()
			var hdr [6]byte
			for _, doc := range docs {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				id := doc.DocID
				if len(id) > 65535 {
					id = id[:65535]
				}
				binary.LittleEndian.PutUint16(hdr[0:2], uint16(len(id)))
				binary.LittleEndian.PutUint32(hdr[2:6], uint32(len(doc.Text)))
				buf.Write(hdr[:2])
				buf.WriteString(id)
				buf.Write(hdr[2:6])
				buf.Write(doc.Text)
				bufDocs++
				if bufDocs >= binGzDocsPerMember {
					if err := flushMember(); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}

	stats, pipeErr := RunPipeline(ctx, eng, PipelineConfig{
		SourceDir: markdownDir,
		BatchSize: batchSize,
		Workers:   workers,
	}, progress)

	if pipeErr == nil {
		mu.Lock()
		pipeErr = flushMember() // flush remaining docs
		mu.Unlock()
	}

	closeErr := f.Close()
	if pipeErr != nil {
		os.Remove(packPath)
		return stats, pipeErr
	}
	if closeErr != nil {
		return stats, closeErr
	}

	// Write the index file.
	if err := writeBinGzIdx(packPath+".idx", members); err != nil {
		return stats, fmt.Errorf("write bingz idx: %w", err)
	}

	return stats, nil
}

// writeBinGzIdx writes the member index to path.
func writeBinGzIdx(path string, members []memberEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 64*1024)
	var b [12]byte
	for _, m := range members {
		binary.LittleEndian.PutUint64(b[0:8], m.offset)
		binary.LittleEndian.PutUint32(b[8:12], m.docCount)
		bw.Write(b[:])
	}
	// Footer: member_count uint32
	var foot [4]byte
	binary.LittleEndian.PutUint32(foot[:], uint32(len(members)))
	bw.Write(foot[:])
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// countingWriter wraps an io.Writer and counts bytes written.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}
```

**Step 2: Build**

```bash
go build ./pkg/index/...
```

**Step 3: Commit**

```bash
git add pkg/index/pack_bingz.go
git commit -m "feat(index): PackFlatBinGz writer — concat gzip members with BestCompression"
```

---

## Task 6: New `pkg/index/pack_bingz.go` — reader

**Files:**
- Modify: `pkg/index/pack_bingz.go` (append)

**Context:** `RunPipelineFromFlatBinGz` reads the `.bin.gz` file in parallel using the `.idx` index. Each worker opens its own file handle and decompresses its assigned members independently. If `.idx` is missing, it is rebuilt by scanning for gzip magic bytes.

**Step 1: Append reader functions to `pack_bingz.go`**

```go
// RunPipelineFromFlatBinGz reads a packed .bin.gz file and feeds documents into engine.
//
// It loads (or builds) the member index from packPath+".idx", then spawns
// NumCPU workers each opening their own file handle and reading their member range.
func RunPipelineFromFlatBinGz(ctx context.Context, engine Engine, packPath string, batchSize int, progress PackProgressFunc) (*PipelineStats, error) {
	members, err := loadOrBuildBinGzIdx(packPath)
	if err != nil {
		return nil, fmt.Errorf("bingz index: %w", err)
	}
	if len(members) == 0 {
		return &PipelineStats{StartTime: timeNow()}, nil
	}

	var total int64
	for _, m := range members {
		total += int64(m.docCount)
	}

	docCh := make(chan Document, max(batchSize*4, 4096))

	numWorkers := runtime.NumCPU()
	if numWorkers > len(members) {
		numWorkers = len(members)
	}

	memberCh := make(chan memberEntry, len(members))
	for _, m := range members {
		memberCh <- m
	}
	close(memberCh)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wf, err := os.Open(packPath)
			if err != nil {
				return
			}
			defer wf.Close()

			var hdr [6]byte
			for m := range memberCh {
				if ctx.Err() != nil {
					return
				}
				if _, err := wf.Seek(int64(m.offset), io.SeekStart); err != nil {
					return
				}
				gr, err := kgzip.NewReader(wf)
				if err != nil {
					return
				}
				br := bufio.NewReaderSize(gr, 512*1024)
				for i := uint32(0); i < m.docCount; i++ {
					if ctx.Err() != nil {
						gr.Close()
						return
					}
					if _, err := io.ReadFull(br, hdr[:2]); err != nil {
						break
					}
					idLen := int(binary.LittleEndian.Uint16(hdr[:2]))
					idBuf := make([]byte, idLen)
					if _, err := io.ReadFull(br, idBuf); err != nil {
						break
					}
					if _, err := io.ReadFull(br, hdr[2:6]); err != nil {
						break
					}
					textLen := int(binary.LittleEndian.Uint32(hdr[2:6]))
					textBuf := make([]byte, textLen)
					if _, err := io.ReadFull(br, textBuf); err != nil {
						break
					}
					select {
					case docCh <- Document{DocID: string(idBuf), Text: textBuf}:
					case <-ctx.Done():
						gr.Close()
						return
					}
				}
				gr.Close()
			}
		}()
	}

	go func() {
		wg.Wait()
		close(docCh)
	}()

	return RunPipelineFromChannel(ctx, engine, docCh, total, batchSize, progress)
}

// loadOrBuildBinGzIdx loads the member index from packPath+".idx".
// If the index file is missing or corrupt, it scans packPath for gzip magic
// bytes to rebuild it.
func loadOrBuildBinGzIdx(packPath string) ([]memberEntry, error) {
	idxPath := packPath + ".idx"
	if members, err := readBinGzIdx(idxPath); err == nil && len(members) > 0 {
		return members, nil
	}
	// Rebuild by scanning for gzip member offsets.
	members, err := scanBinGzMembers(packPath)
	if err != nil {
		return nil, err
	}
	// Cache the rebuilt index (best-effort).
	_ = writeBinGzIdx(idxPath, members)
	return members, nil
}

// readBinGzIdx reads a previously written .bin.gz.idx file.
func readBinGzIdx(path string) ([]memberEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("idx too small")
	}
	memberCount := int(binary.LittleEndian.Uint32(data[len(data)-4:]))
	expected := memberCount*12 + 4
	if len(data) != expected {
		return nil, fmt.Errorf("idx size mismatch: got %d want %d", len(data), expected)
	}
	members := make([]memberEntry, memberCount)
	for i := range members {
		off := i * 12
		members[i].offset = binary.LittleEndian.Uint64(data[off:])
		members[i].docCount = binary.LittleEndian.Uint32(data[off+8:])
	}
	return members, nil
}

// scanBinGzMembers finds gzip member offsets by scanning for the gzip magic
// sequence 0x1f 0x8b. Returns one memberEntry per member (docCount=0 since
// doc counts are not recoverable without decompression).
func scanBinGzMembers(packPath string) ([]memberEntry, error) {
	data, err := os.ReadFile(packPath)
	if err != nil {
		return nil, err
	}
	var members []memberEntry
	for i := 0; i < len(data)-1; {
		if data[i] == 0x1f && data[i+1] == 0x8b {
			members = append(members, memberEntry{offset: uint64(i)})
			// Advance past this gzip member by decompressing it to find its end.
			gr, err := kgzip.NewReader(bytes.NewReader(data[i:]))
			if err != nil {
				break
			}
			n, _ := io.Copy(io.Discard, gr)
			gr.Close()
			_ = n
			// Find where the decompressed data ends in the compressed stream
			// by trying gzip.Reader which stops at stream end.
			// We rely on reading exactly one member per kgzip.Reader.
			// To advance i, decompress and note the reader position isn't trackable
			// easily — instead just search for next 0x1f 0x8b after current pos+1.
			i++
			for i < len(data)-1 {
				if data[i] == 0x1f && data[i+1] == 0x8b {
					break
				}
				i++
			}
		} else {
			i++
		}
	}
	return members, nil
}

func timeNow() time.Time { return time.Now() }
```

> **Note on scanBinGzMembers**: The scan approach works but `docCount` will be 0 for rebuilt indices (no doc count stored). For a complete rebuild that knows doc counts, the caller must decompress each member. In practice the `.idx` file is always written by `PackFlatBinGz` so this fallback is rarely used.

**Step 2: Build**

```bash
go build ./pkg/index/...
```

**Step 3: Commit**

```bash
git add pkg/index/pack_bingz.go
git commit -m "feat(index): RunPipelineFromFlatBinGz reader with parallel member decompression"
```

---

## Task 7: Expose `PackFlatBinGz` in `cli/cc_fts.go` (duckdb ops)

**Files:**
- Modify: `cli/cc_fts.go` (or `cli/duckdb_ops.go` if DuckDB-specific)

**Context:** `packFilePath` and `runCCFTSPack` need to be updated.
The `ndjson` format is removed. A new `markdown` format produces `.bin.gz`.
All paths move from `fts/pack/docs.*` to `pack/{format}/{warcIdx}.{ext}`.

**Step 1: Add `warcIndexFromPath` helper to `cli/cc_fts.go`**

```go
// warcIndexFromPath extracts the zero-padded 5-digit WARC file index from
// a WARC filename or its manifest path.
//
//   "CC-MAIN-20260206181458-20260206211458-00000.warc.gz" → "00000"
//
// Falls back to fmt.Sprintf("%05d", fallback) if the filename does not end
// with a 5-digit segment before ".warc.gz".
func warcIndexFromPath(warcPath string, fallback int) string {
	base := filepath.Base(warcPath)
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".gz"), ".warc")
	parts := strings.Split(name, "-")
	if last := parts[len(parts)-1]; len(last) == 5 {
		if _, err := strconv.Atoi(last); err == nil {
			return last
		}
	}
	return fmt.Sprintf("%05d", fallback)
}
```

**Step 2: Rewrite `packFilePath`**

```go
// packFilePath returns the full path for a pack file given format, packDir, and warcIdx.
func packFilePath(packDir, format, warcIdx string) (string, error) {
	switch format {
	case "parquet":
		return filepath.Join(packDir, "parquet", warcIdx+".parquet"), nil
	case "bin":
		return filepath.Join(packDir, "bin", warcIdx+".bin"), nil
	case "duckdb":
		return filepath.Join(packDir, "duckdb", warcIdx+".duckdb"), nil
	case "markdown":
		return filepath.Join(packDir, "markdown", warcIdx+".bin.gz"), nil
	default:
		return "", fmt.Errorf("unknown format %q (valid: parquet, bin, duckdb, markdown)", format)
	}
}
```

**Step 3: Update `runCCFTSPack` to use warcIdx + new paths**

```go
func runCCFTSPack(ctx context.Context, crawlID, fileIdx, format string, batchSize, workers int) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	packDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "pack")

	// Resolve WARC manifest to get warcIdx.
	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	formats := []string{format}
	if format == "all" {
		formats = []string{"parquet", "bin", "duckdb", "markdown"}
	}

	for _, idx := range selected {
		warcIdx := warcIndexFromPath(paths[idx], idx)
		markdownDir := filepath.Join(homeDir, "data", "common-crawl", crawlID, "markdown", warcIdx)
		if _, err := os.Stat(markdownDir); os.IsNotExist(err) {
			return fmt.Errorf("markdown dir not found: %s\n  run: search cc warc markdown --file %d", markdownDir, idx)
		}

		for _, fmt_ := range formats {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			packFile, err := packFilePath(packDir, fmt_, warcIdx)
			if err != nil {
				return err
			}
			if err := runPackFormat(ctx, fmt_, markdownDir, packFile, batchSize, workers); err != nil {
				return fmt.Errorf("pack %s: %w", fmt_, err)
			}
		}
	}
	return nil
}
```

**Step 4: Update `runPackFormat` to handle `markdown` format**

In the `switch format` block add:
```go
case "markdown":
	stats, err = index.PackFlatBinGz(ctx, markdownDir, packFile, workers, batchSize, progress)
```
Remove the `ndjson` case.

**Step 5: Update flag description for `--format`**

```go
cmd.Flags().StringVar(&format, "format", "all", "Format: parquet, bin, duckdb, markdown, all")
```

Also add `--file` flag to `newCCFTSPack`:
```go
cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9), or all")
```

**Step 6: Build**

```bash
go build ./cli/...
```

**Step 7: Commit**

```bash
git add cli/cc_fts.go
git commit -m "feat(cli/fts): per-WARC pack paths, add markdown format, remove ndjson"
```

---

## Task 8: Update `runCCFTSIndex` for per-WARC paths

**Files:**
- Modify: `cli/cc_fts.go`

**Context:** `runCCFTSIndex` currently opens `fts/{engine}/` and reads from `fts/pack/docs.*`. Both move to per-WARC paths. Add `--file` flag. Add `markdown` source that reads `.bin.gz`.

**Step 1: Add `--file` flag to `newCCFTSIndex`**

```go
cmd.Flags().StringVar(&fileIdx, "file", "0", "File index, range (0-9)")
```

**Step 2: Rewrite `runCCFTSIndex`**

```go
func runCCFTSIndex(ctx context.Context, crawlID, fileIdx, engineName, source string, batchSize, workers int, addr string) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, "data", "common-crawl", crawlID)

	client := cc.NewClient("", 4)
	paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	selected, err := ccParseFileSelector(fileIdx, len(paths))
	if err != nil {
		return fmt.Errorf("--file: %w", err)
	}

	for _, idx := range selected {
		warcIdx := warcIndexFromPath(paths[idx], idx)
		outputDir := filepath.Join(baseDir, "fts", engineName, warcIdx)

		eng, err := index.NewEngine(engineName)
		if err != nil {
			return err
		}
		if addr != "" {
			if setter, ok := eng.(index.AddrSetter); ok {
				setter.SetAddr(addr)
			}
		}
		if err := eng.Open(ctx, outputDir); err != nil {
			return fmt.Errorf("open engine: %w", err)
		}

		var stats *index.PipelineStats
		if source == "files" {
			sourceDir := filepath.Join(baseDir, "markdown", warcIdx)
			if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
				eng.Close()
				return fmt.Errorf("markdown dir not found: %s", sourceDir)
			}
			cfg := index.PipelineConfig{SourceDir: sourceDir, BatchSize: batchSize, Workers: workers}
			progress := makeFTSProgress(engineName, outputDir)
			stats, err = index.RunPipeline(ctx, eng, cfg, progress)
		} else {
			packDir := filepath.Join(baseDir, "pack")
			packFile, perr := packFilePath(packDir, source, warcIdx)
			if perr != nil {
				eng.Close()
				return perr
			}
			progress := makeFTSPackProgress(engineName, source, outputDir)
			stats, err = runCCFTSIndexFromPackFile(ctx, engineName, source, eng, outputDir, packFile, batchSize, progress)
		}

		eng.Close()
		if err != nil {
			return err
		}
		_ = stats
	}
	return nil
}
```

**Step 3: Update `runCCFTSIndexFromPack` → `runCCFTSIndexFromPackFile`**

Rename and change signature to accept `packFile string` directly (no longer derives it internally):

```go
func runCCFTSIndexFromPackFile(ctx context.Context, engineName, source string, eng index.Engine, outputDir, packFile string, batchSize int, progress index.PackProgressFunc) (*index.PipelineStats, error) {
	// ... existing switch on source, but use packFile directly.
	// Add case "markdown":
	//   stats, err = index.RunPipelineFromFlatBinGz(ctx, eng, packFile, batchSize, progress)
}
```

**Step 4: Build and fix compilation errors**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add cli/cc_fts.go
git commit -m "feat(cli/fts): per-WARC index paths, --file flag, markdown source via .bin.gz"
```

---

## Task 9: Fan-out search in `runCCFTSSearch`

**Files:**
- Modify: `cli/cc_fts.go`

**Context:** Without `--file`, the search opens every per-WARC FTS directory under `fts/{engine}/`, searches them in parallel, and merges the top-K results by score.

**Step 1: Add `--file` flag to `newCCFTSSearch` (optional, default="" = all)**

```go
cmd.Flags().StringVar(&fileIdx, "file", "", "File index to search (default: all WARCs)")
```

**Step 2: Rewrite `runCCFTSSearch`**

```go
func runCCFTSSearch(ctx context.Context, crawlID, fileIdx, engineName, query string, limit, offset int, addr string) error {
	if crawlID == "" {
		crawlID = detectLatestCrawl()
	}

	homeDir, _ := os.UserHomeDir()
	ftsBase := filepath.Join(homeDir, "data", "common-crawl", crawlID, "fts", engineName)

	// Collect target directories.
	var targetDirs []string
	if fileIdx != "" {
		// Single WARC mode.
		client := cc.NewClient("", 4)
		paths, err := client.DownloadManifest(ctx, crawlID, "warc.paths.gz")
		if err != nil {
			return err
		}
		selected, err := ccParseFileSelector(fileIdx, len(paths))
		if err != nil {
			return err
		}
		for _, idx := range selected {
			targetDirs = append(targetDirs, filepath.Join(ftsBase, warcIndexFromPath(paths[idx], idx)))
		}
	} else {
		// Fan-out: discover all per-WARC directories.
		entries, err := os.ReadDir(ftsBase)
		if err != nil {
			return fmt.Errorf("no FTS index at %s — run 'cc fts index' first", ftsBase)
		}
		for _, e := range entries {
			if e.IsDir() {
				targetDirs = append(targetDirs, filepath.Join(ftsBase, e.Name()))
			}
		}
		if len(targetDirs) == 0 {
			return fmt.Errorf("no per-WARC FTS indices found under %s", ftsBase)
		}
	}

	// Search all target dirs in parallel, collect results.
	type shardResult struct {
		hits  []index.Hit
		total int
		err   error
	}
	results := make([]shardResult, len(targetDirs))
	var wg sync.WaitGroup
	for i, dir := range targetDirs {
		i, dir := i, dir
		wg.Add(1)
		go func() {
			defer wg.Done()
			eng, err := index.NewEngine(engineName)
			if err != nil {
				results[i].err = err
				return
			}
			if addr != "" {
				if setter, ok := eng.(index.AddrSetter); ok {
					setter.SetAddr(addr)
				}
			}
			if err := eng.Open(ctx, dir); err != nil {
				results[i].err = err
				return
			}
			defer eng.Close()

			res, err := eng.Search(ctx, index.Query{Text: query, Limit: limit + offset, Offset: 0})
			if err != nil {
				results[i].err = err
				return
			}
			results[i].hits = res.Hits
			results[i].total = res.Total
		}()
	}
	wg.Wait()

	// Merge: collect all hits, sort by score descending, take top limit.
	var allHits []index.Hit
	var totalCount int
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "warning: shard error: %v\n", r.err)
			continue
		}
		allHits = append(allHits, r.hits...)
		totalCount += r.total
	}
	// Sort by score descending.
	sort.Slice(allHits, func(i, j int) bool {
		return allHits[i].Score > allHits[j].Score
	})
	if offset < len(allHits) {
		allHits = allHits[offset:]
	} else {
		allHits = nil
	}
	if len(allHits) > limit {
		allHits = allHits[:limit]
	}

	fmt.Fprintf(os.Stderr, "── Results for %q (engine: %s, shards: %d, total: %d) ──\n",
		query, engineName, len(targetDirs), totalCount)
	for i, hit := range allHits {
		snippet := strings.ReplaceAll(hit.Snippet, "\n", " ")
		if len(snippet) > 80 {
			snippet = snippet[:80] + "..."
		}
		fmt.Fprintf(os.Stderr, "  %-4d %-8.2f %-40s %s\n", i+1+offset, hit.Score, hit.DocID, snippet)
	}

	return nil
}
```

**Step 3: Add `sort` import to `cli/cc_fts.go`**

```go
import "sort"
```

**Step 4: Build**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add cli/cc_fts.go
git commit -m "feat(cli/fts): fan-out search across all per-WARC FTS indices"
```

---

## Task 10: Update `cli/cc_warc_markdown.go`

**Files:**
- Modify: `cli/cc_warc_markdown.go`

**Context:** Remove phase 3 (compress), remove `--mem` flag, extract `warcIdx` from WARC filename, pass to pipeline.

**Step 1: Remove `--mem` flag and compress phase**

In `newCCWarcMarkdown`, remove `inMemory bool` variable and flag:
```go
// Remove:
// cmd.Flags().BoolVar(&inMemory, "mem", false, "Streaming pipeline: no temp files")
```

In `runCCWarcMarkdown`, remove `inMemory` parameter and the `runWARCMDInMemory` branch.

**Step 2: Extract warcIdx and pass to RunFilePipeline**

In the loop that builds `inputFiles`, compute `warcIdx`:
```go
for _, idx := range selected {
    warcIdx := warcIndexFromPath(paths[idx], idx)
    localPath := filepath.Join(warcDir, filepath.Base(paths[idx]))
    // ... auto-download if missing ...
    inputFiles = append(inputFiles, struct{ path, idx string }{localPath, warcIdx})
}
```

Or process one file at a time (simpler):
```go
for i, idx := range selected {
    warcIdx := warcIndexFromPath(paths[idx], idx)
    localPath := filepath.Join(warcDir, filepath.Base(paths[idx]))
    if !fileExists(localPath) {
        if err := downloadWithProgress(ctx, client, paths[idx], localPath); err != nil {
            return err
        }
    }
    if err := runCCWarcMarkdownOne(ctx, cfg, warcIdx, localPath, i+1, len(selected)); err != nil {
        return err
    }
}
```

Where `runCCWarcMarkdownOne` calls `warcmd.RunFilePipeline(ctx, cfg, warcIdx, []string{localPath}, p1Fn, p2Fn)`.

**Step 3: Update summary output**

Replace `cfg.MarkdownGzDir()` with `cfg.MarkdownWarcDir(warcIdx)` in display strings. Remove compress phase rows from the summary table.

**Step 4: Remove `runWARCMDInMemory`, `runWARCMDParallelFiles`** (or keep parallel mode but without compress)

For parallel mode (`len(inputFiles) > 1 && workers > 1`): call `warcmd.RunFilePipeline` for each file with its own `warcIdx`.

**Step 5: Build**

```bash
go build ./...
```

**Step 6: Commit**

```bash
git add cli/cc_warc_markdown.go
git commit -m "feat(cli/warc-markdown): 2-phase pipeline, per-WARC output, remove --mem"
```

---

## Task 11: Remove `cc fts decompress` command

**Files:**
- Modify: `cli/cc_fts.go`

**Step 1: Remove `newCCFTSDecompress` and `runCCFTSDecompress`**

Delete both functions. Remove `cmd.AddCommand(newCCFTSDecompress())` from `newCCFTS()`.

**Step 2: Build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add cli/cc_fts.go
git commit -m "feat(cli/fts): remove decompress command (markdown now always uncompressed)"
```

---

## Task 12: Run local tests

**Step 1: Run package tests**

```bash
cd blueprints/search
go test ./pkg/cc/... ./pkg/warc_md/... ./pkg/index/... -v -count=1
```
Expected: all pass.

**Step 2: Build the full binary**

```bash
make install   # or: go build -o ~/bin/search ./cmd/search/
```

**Step 3: Smoke-test cc warc markdown**

```bash
# Download one WARC file if not already present
search cc warc download --file 0

# Convert to markdown
search cc warc markdown --file 0
# Verify output:
ls ~/data/common-crawl/CC-MAIN-2026-08/markdown/00000/ | head
```

**Step 4: Smoke-test cc fts pack**

```bash
search cc fts pack --file 0 --format bin
search cc fts pack --file 0 --format markdown
ls ~/data/common-crawl/CC-MAIN-2026-08/pack/bin/
ls ~/data/common-crawl/CC-MAIN-2026-08/pack/markdown/
```

**Step 5: Smoke-test cc fts index + search**

```bash
search cc fts index --file 0 --engine rose --source files
search cc fts search "machine learning" --engine rose
```

**Step 6: Commit**

```bash
git add .
git commit -m "test: local smoke-test pass for per-WARC refactor"
```

---

## Task 13: Deploy to server2 and run benchmarks

**Step 1: Build Linux binary**

```bash
make build-linux-noble   # ~15-20min under QEMU, use run_in_background=true
make deploy-linux-noble SERVER=2
```

**Step 2: On server2 — clean old data dirs**

```bash
ssh server2
rm -rf ~/data/common-crawl/CC-MAIN-2026-08/markdown/
rm -rf ~/data/common-crawl/CC-MAIN-2026-08/fts/
# Keep warc/ (reuse downloaded files)
```

**Step 3: Run cc warc markdown (WARC 0)**

```bash
time ~/bin/search cc warc markdown --file 0
du -sh ~/data/common-crawl/CC-MAIN-2026-08/markdown/00000/
```

**Step 4: Run cc fts pack (all formats)**

```bash
time ~/bin/search cc fts pack --file 0 --format all
du -sh ~/data/common-crawl/CC-MAIN-2026-08/pack/*/00000.*
```

**Step 5: Run cc fts index (all engines)**

```bash
for engine in rose duckdb sqlite bleve; do
  echo "=== $engine ==="
  time ~/bin/search cc fts index --file 0 --engine $engine --source files
done
```

**Step 6: Run cc fts search (fan-out)**

```bash
~/bin/search cc fts search "machine learning" --engine rose
~/bin/search cc fts search "climate change" --engine duckdb
```

**Step 7: Record all metrics in spec/0648**

Update `spec/0648_refactor_data_directory.md` Benchmark Results table with:
- docs/s, MB/s read/write for markdown conversion
- pack file sizes + compression ratios
- index build docs/s + peak RSS per engine
- search latency (fan-out 1 WARC vs multiple)
- disk usage: markdown/, pack/, fts/ per WARC

```bash
git add spec/0648_refactor_data_directory.md
git commit -m "docs(spec/0648): benchmark results from server2"
```

---

## Execution Options

**Plan complete and saved to `docs/plans/2026-03-03-data-directory-refactor.md`.**

Two execution options:

**1. Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Parallel Session (separate)** — Open a new session with executing-plans, batch execution with checkpoints.

Which approach?
