package warc_md

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	kgzip "github.com/klauspost/compress/gzip"

	mdpkg "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// bodyBufPool recycles large byte buffers used for io.ReadAll of WARC record
// bodies. Each buffer grows to MaxBodySize+8192 once and is reused, eliminating
// ~50,000 allocations per shard.
var bodyBufPool = sync.Pool{
	New: func() any {
		b := bytes.NewBuffer(make([]byte, 0, 520*1024)) // 512 KB + 8 KB
		return b
	},
}

// maxPackWorkers caps the number of parallel converter workers. With the
// sequential reader path (RunPack), the single-threaded gzip reader is the
// bottleneck, not the converters. 8 workers provide 8000 conversions/s which
// far exceeds the reader's ~500 records/s throughput. Keeping this low
// reduces system-wide CPU contention when multiple sessions run concurrently.
const maxPackWorkers = 8

// PackConfig configures the pack pipeline: .warc.gz → .md.warc.gz
type PackConfig struct {
	InputFiles  []string // .warc.gz source files
	OutputPath  string   // output .md.warc.gz file path
	Workers     int      // parallel converter goroutines (0 = NumCPU*4)
	Force       bool     // overwrite existing output
	FastConvert  bool // use go-readability instead of trafilatura
	LightConvert bool // use lightweight extractor (fastest, less precise)
	StatusCode   int  // HTTP status filter (default: 200)
	MIMEFilter  string   // MIME type filter (default: "text/html")
	MaxBodySize int64    // max HTML body bytes per record (default: 512 KB)
}

// packItem is sent from reader to converter workers.
type packItem struct {
	targetURI string
	date      string
	recordID  string
	htmlBody  []byte
	htmlLen   int // original HTML body byte length before conversion
}

// packResult is sent from converter workers to the writer.
type packResult struct {
	targetURI  string
	date       string
	refersTo   string // original WARC-Record-ID
	markdown   string
	hasContent bool
	htmlLen    int // original HTML body byte length
}

// PackStats holds statistics for a completed pack run.
type PackStats struct {
	InputRecords  int64
	OutputRecords int64
	Skipped       int64
	Errors        int64
	ReadBytes     int64
	WriteBytes    int64
	PeakMemMB     float64 // peak RSS in MB (VmRSS on Linux)
	Duration      time.Duration
}

// PackRecord is an exported record produced by the pack pipeline.
// Used by RunPackDirect to stream results to callers without intermediate WARC files.
type PackRecord struct {
	TargetURI string
	Date      string
	RefersTo  string // original WARC-Record-ID
	Markdown  string
	HTMLLen   int
}

// PackWriterFunc receives converted records from the pack pipeline.
// Called from a single goroutine — does not need to be thread-safe.
// Return non-nil error to abort the pipeline.
type PackWriterFunc func(PackRecord) error

// RunPack executes the pack pipeline: read .warc.gz → convert HTML→Markdown → write .md.warc.gz.
//
// Architecture: reader goroutine → N converter workers → single writer goroutine.
// Records are NOT guaranteed to be in input order (parallel conversion reorders).
func RunPack(ctx context.Context, cfg PackConfig, progressFn ProgressFunc) (*PackStats, error) {
	if !cfg.Force {
		if _, err := os.Stat(cfg.OutputPath); err == nil {
			return &PackStats{Skipped: 1}, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0o755); err != nil {
		return nil, err
	}

	// Use internal pipeline with WARC file writer.
	return runPackPipeline(ctx, cfg, nil, progressFn)
}

// RunPackDirect executes the pack pipeline and calls writerFn for each converted
// record instead of writing to a .md.warc.gz file. This eliminates intermediate
// WARC serialization, gzip compression, and disk I/O — the caller writes directly
// to the target format (e.g., parquet).
//
// cfg.OutputPath is ignored. The caller is responsible for file creation/cleanup.
func RunPackDirect(ctx context.Context, cfg PackConfig, writerFn PackWriterFunc, progressFn ProgressFunc) (*PackStats, error) {
	return runPackPipeline(ctx, cfg, writerFn, progressFn)
}

// runPackPipeline is the shared reader→converter→writer pipeline.
// If writerFn is nil, results are written as WARC to cfg.OutputPath.
// If writerFn is non-nil, results are passed to the callback (direct mode).
func runPackPipeline(ctx context.Context, cfg PackConfig, writerFn PackWriterFunc, progressFn ProgressFunc) (*PackStats, error) {
	if len(cfg.InputFiles) == 0 {
		return &PackStats{}, nil
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU() * 4
	}
	if cfg.Workers > maxPackWorkers {
		cfg.Workers = maxPackWorkers
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = 200
	}
	if cfg.MIMEFilter == "" {
		cfg.MIMEFilter = "text/html"
	}
	if cfg.MaxBodySize == 0 {
		cfg.MaxBodySize = 512 * 1024
	}

	// GOGC=200: 2× live heap. Higher values waste memory as GC headroom;
	// pack is I/O-bound so the extra GC pauses are hidden by disk/network waits.
	prevGOGC := debug.SetGCPercent(200)
	defer debug.SetGCPercent(prevGOGC)

	// GOMEMLIMIT: hard ceiling prevents OOM kills (see pack_parallel.go).
	prevMemLimit := debug.SetMemoryLimit(600 << 20) // 600 MiB
	defer debug.SetMemoryLimit(prevMemLimit)

	var stats PackStats
	start := time.Now()

	stopMem := make(chan struct{})
	getPeakMB := trackPeakMem(stopMem)

	// Channels
	itemCh := make(chan packItem, cfg.Workers*2)
	resultCh := make(chan packResult, cfg.Workers*2)
	readerDone := make(chan error, 1)
	writerDone := make(chan error, 1)

	// Progress ticker
	var stopProgress chan struct{}
	if progressFn != nil {
		stopProgress = make(chan struct{})
		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					in := atomic.LoadInt64(&stats.InputRecords)
					out := atomic.LoadInt64(&stats.OutputRecords)
					errs := atomic.LoadInt64(&stats.Errors)
					progressFn(out, in, errs,
						atomic.LoadInt64(&stats.ReadBytes),
						atomic.LoadInt64(&stats.WriteBytes),
						time.Since(start), getPeakMB())
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// ── Reader goroutine ────────────────────────────────────────────────────
	go func() {
		defer close(itemCh)
		for _, warcPath := range cfg.InputFiles {
			if ctx.Err() != nil {
				readerDone <- ctx.Err()
				return
			}
			if err := packReadFile(ctx, warcPath, cfg, &stats, itemCh); err != nil {
				readerDone <- err
				return
			}
		}
		readerDone <- nil
	}()

	// ── Converter workers ───────────────────────────────────────────────────
	convertFn := mdpkg.Convert
	if cfg.LightConvert {
		convertFn = mdpkg.ConvertUltraLight // tokenizer-based, no DOM tree
	} else if cfg.FastConvert {
		convertFn = mdpkg.ConvertFast
	}

	converterDone := make(chan struct{})
	go func() {
		defer close(resultCh)
		defer close(converterDone)

		sem := make(chan struct{}, cfg.Workers)
		for item := range itemCh {
			if ctx.Err() != nil {
				return
			}
			sem <- struct{}{}
			go func(it packItem) {
				defer func() { <-sem }()

				res := convertFn(it.htmlBody, "")

				if res.HasContent && res.Markdown != "" {
					resultCh <- packResult{
						targetURI:  it.targetURI,
						date:       it.date,
						refersTo:   it.recordID,
						markdown:   res.Markdown,
						hasContent: true,
						htmlLen:    it.htmlLen,
					}
				} else {
					atomic.AddInt64(&stats.Errors, 1)
				}
			}(item)
		}
		// Wait for all in-flight conversions
		for range cfg.Workers {
			sem <- struct{}{}
		}
	}()

	// ── Writer goroutine ────────────────────────────────────────────────────
	go func() {
		if writerFn != nil {
			// Direct mode: pass results to caller's writer function.
			writerDone <- packWriteDirect(resultCh, &stats, writerFn)
		} else {
			// WARC mode: write to .md.warc.gz file.
			writerDone <- packWriteFile(cfg.OutputPath, resultCh, &stats)
		}
	}()

	// Wait for pipeline
	readErr := <-readerDone
	<-converterDone
	writeErr := <-writerDone

	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	stats.Duration = time.Since(start)
	stats.PeakMemMB = getPeakMB()

	if readErr != nil {
		return &stats, readErr
	}
	if writeErr != nil {
		return &stats, writeErr
	}
	if ctx.Err() != nil {
		return &stats, ctx.Err()
	}
	return &stats, nil
}

// packWriteDirect streams results to a caller-provided writer function.
func packWriteDirect(results <-chan packResult, stats *PackStats, writerFn PackWriterFunc) error {
	for res := range results {
		if !res.hasContent || len(res.markdown) == 0 {
			continue
		}
		if err := writerFn(PackRecord{
			TargetURI: res.targetURI,
			Date:      res.date,
			RefersTo:  res.refersTo,
			Markdown:  res.markdown,
			HTMLLen:   res.htmlLen,
		}); err != nil {
			return err
		}
		atomic.AddInt64(&stats.OutputRecords, 1)
		atomic.AddInt64(&stats.WriteBytes, int64(len(res.markdown)))
	}
	return nil
}

// packReadFile reads a single .warc.gz and sends matching HTML records to itemCh.
func packReadFile(ctx context.Context, warcPath string, cfg PackConfig, stats *PackStats, itemCh chan<- packItem) error {
	f, err := os.Open(warcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := warcpkg.NewReader(f)
	for r.Next() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rec := r.Record()
		if rec.Header.Type() != warcpkg.TypeResponse {
			io.Copy(io.Discard, rec.Body)
			continue
		}

		// Read body using pooled buffer to eliminate per-record allocations.
		buf := bodyBufPool.Get().(*bytes.Buffer)
		buf.Reset()
		var bodyReader io.Reader = rec.Body
		if cfg.MaxBodySize > 0 {
			bodyReader = io.LimitReader(rec.Body, cfg.MaxBodySize+8192)
		}
		_, err := buf.ReadFrom(bodyReader)
		if err != nil {
			bodyBufPool.Put(buf)
			atomic.AddInt64(&stats.Errors, 1)
			continue
		}
		bodyBytes := buf.Bytes()
		atomic.AddInt64(&stats.ReadBytes, int64(len(bodyBytes)))

		status, mime, htmlBody := parseHTTPResponseFast(bodyBytes)
		if cfg.StatusCode != 0 && status != cfg.StatusCode {
			bodyBufPool.Put(buf)
			continue
		}
		if cfg.MIMEFilter != "" && mime != cfg.MIMEFilter {
			if idx := strings.IndexByte(cfg.MIMEFilter, '/'); idx >= 0 {
				if !strings.HasPrefix(mime, cfg.MIMEFilter[:idx]) {
					bodyBufPool.Put(buf)
					continue
				}
			} else {
				bodyBufPool.Put(buf)
				continue
			}
		}

		if cfg.MaxBodySize > 0 && int64(len(htmlBody)) > cfg.MaxBodySize {
			htmlBody = htmlBody[:cfg.MaxBodySize]
		}

		if len(htmlBody) == 0 {
			bodyBufPool.Put(buf)
			continue
		}

		atomic.AddInt64(&stats.InputRecords, 1)

		// Copy htmlBody out of the pooled buffer so we can return it immediately.
		// The copy is max 512 KB; far cheaper than allocating a fresh buffer per record.
		htmlCopy := make([]byte, len(htmlBody))
		copy(htmlCopy, htmlBody)
		bodyBufPool.Put(buf)

		itemCh <- packItem{
			targetURI: rec.Header.TargetURI(),
			date:      rec.Header.Get("WARC-Date"),
			recordID:  rec.Header.RecordID(),
			htmlBody:  htmlCopy,
			htmlLen:   len(htmlCopy),
		}
	}
	return r.Err()
}

// packWriteFile writes WARC conversion records to a seekable .md.warc.gz.
// Each record is wrapped in its own gzip member.
func packWriteFile(outputPath string, results <-chan packResult, stats *PackStats) error {
	tmpPath := outputPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer func() {
		f.Close()
		os.Remove(tmpPath) // cleanup on error
	}()

	bw := bufio.NewWriterSize(f, 1024*1024) // 1 MB write buffer

	for res := range results {
		if !res.hasContent || len(res.markdown) == 0 {
			continue
		}

		// Build WARC record
		newID := fmt.Sprintf("<urn:uuid:%s>", uuid.New().String())
		contentLen := strconv.Itoa(len(res.markdown))

		hdr := warcpkg.Header{
			"WARC-Type":       warcpkg.TypeConversion,
			"WARC-Target-URI": res.targetURI,
			"WARC-Date":       res.date,
			"WARC-Record-ID":  newID,
			"WARC-Refers-To":  res.refersTo,
			"Content-Type":    "text/markdown",
			"Content-Length":  contentLen,
			"X-HTML-Length":   strconv.Itoa(res.htmlLen),
		}

		rec := &warcpkg.Record{
			Header: hdr,
			Body:   strings.NewReader(res.markdown),
		}

		// Each record in its own gzip member
		gz, err := kgzip.NewWriterLevel(bw, kgzip.BestSpeed)
		if err != nil {
			return err
		}

		w := warcpkg.NewWriter(gz)
		if err := w.WriteRecord(rec); err != nil {
			gz.Close()
			return err
		}
		if err := w.Close(); err != nil {
			gz.Close()
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}

		atomic.AddInt64(&stats.OutputRecords, 1)
		atomic.AddInt64(&stats.WriteBytes, int64(len(res.markdown)))
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, outputPath)
}

// parseHTTPResponseFast is a zero-allocation HTTP response parser for the pack
// hot path. It scans data in-place without creating bufio/textproto readers.
// Returns (statusCode, mimeType, bodySlice). bodySlice is a sub-slice of data.
func parseHTTPResponseFast(data []byte) (status int, mime string, body []byte) {
	// Find end of status line
	nl := bytes.IndexByte(data, '\n')
	if nl < 0 {
		return 0, "", nil
	}
	// Parse status code from "HTTP/1.x NNN ..."
	statusLine := data[:nl]
	if sp := bytes.IndexByte(statusLine, ' '); sp >= 0 {
		rest := statusLine[sp+1:]
		for i := 0; i < len(rest) && i < 3; i++ {
			c := rest[i]
			if c < '0' || c > '9' {
				break
			}
			status = status*10 + int(c-'0')
		}
	}
	pos := nl + 1

	// Scan headers for Content-Type and end-of-headers
	for pos < len(data) {
		lineEnd := bytes.IndexByte(data[pos:], '\n')
		if lineEnd < 0 {
			return status, mime, nil
		}
		line := data[pos : pos+lineEnd]
		line = bytes.TrimRight(line, "\r")
		pos += lineEnd + 1

		if len(line) == 0 {
			// Empty line = end of headers
			body = data[pos:]
			return
		}

		// Check for Content-Type header (case-insensitive first char)
		if len(line) > 14 && (line[0] == 'C' || line[0] == 'c') {
			lower := bytes.ToLower(line[:13])
			if bytes.Equal(lower, []byte("content-type:")) {
				val := bytes.TrimSpace(line[13:])
				if sc := bytes.IndexByte(val, ';'); sc >= 0 {
					val = bytes.TrimSpace(val[:sc])
				}
				mime = string(val)
			}
		}
	}
	return status, mime, nil
}
