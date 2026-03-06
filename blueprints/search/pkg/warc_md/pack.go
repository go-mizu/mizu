package warc_md

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	kgzip "github.com/klauspost/compress/gzip"

	mdpkg "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// PackConfig configures the pack pipeline: .warc.gz → .md.warc.gz
type PackConfig struct {
	InputFiles  []string // .warc.gz source files
	OutputPath  string   // output .md.warc.gz file path
	Workers     int      // parallel converter goroutines (0 = NumCPU)
	Force       bool     // overwrite existing output
	Fast        bool     // use go-readability instead of trafilatura
	StatusCode  int      // HTTP status filter (default: 200)
	MIMEFilter  string   // MIME type filter (default: "text/html")
	MaxBodySize int64    // max HTML body bytes per record (default: 512 KB)
}

// packItem is sent from reader to converter workers.
type packItem struct {
	targetURI string
	date      string
	recordID  string
	htmlBody  []byte
}

// packResult is sent from converter workers to the writer.
type packResult struct {
	targetURI  string
	date       string
	refersTo   string // original WARC-Record-ID
	markdown   []byte
	hasContent bool
}

// PackStats holds statistics for a completed pack run.
type PackStats struct {
	InputRecords  int64
	OutputRecords int64
	Skipped       int64
	Errors        int64
	ReadBytes     int64
	WriteBytes    int64
	PeakMemMB     float64
	Duration      time.Duration
}

// RunPack executes the pack pipeline: read .warc.gz → convert HTML→Markdown → write .md.warc.gz.
//
// Architecture: reader goroutine → N converter workers → single writer goroutine.
// Records are NOT guaranteed to be in input order (parallel conversion reorders).
func RunPack(ctx context.Context, cfg PackConfig, progressFn ProgressFunc) (*PackStats, error) {
	if len(cfg.InputFiles) == 0 {
		return &PackStats{}, nil
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
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

	if !cfg.Force {
		if _, err := os.Stat(cfg.OutputPath); err == nil {
			return &PackStats{Skipped: 1}, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0o755); err != nil {
		return nil, err
	}

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

				var res mdpkg.Result
				if cfg.Fast {
					res = mdpkg.ConvertFast(it.htmlBody, "")
				} else {
					res = mdpkg.Convert(it.htmlBody, "")
				}

				if res.HasContent && res.Markdown != "" {
					resultCh <- packResult{
						targetURI:  it.targetURI,
						date:       it.date,
						refersTo:   it.recordID,
						markdown:   []byte(res.Markdown),
						hasContent: true,
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
		writerDone <- packWriteFile(cfg.OutputPath, resultCh, &stats)
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

		// Read body (HTTP status line + headers + HTML body)
		var bodyReader io.Reader = rec.Body
		if cfg.MaxBodySize > 0 {
			bodyReader = io.LimitReader(rec.Body, cfg.MaxBodySize+8192)
		}
		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			atomic.AddInt64(&stats.Errors, 1)
			continue
		}
		atomic.AddInt64(&stats.ReadBytes, int64(len(bodyBytes)))

		status, mime, htmlBody := parseHTTPResponse(bodyBytes)
		if cfg.StatusCode != 0 && status != cfg.StatusCode {
			continue
		}
		if cfg.MIMEFilter != "" {
			wantType := cfg.MIMEFilter
			if idx := bytes.IndexByte([]byte(wantType), '/'); idx >= 0 {
				// Check if mime starts with the type prefix
				if mime == "" || (mime != wantType && !bytes.HasPrefix([]byte(mime), []byte(wantType[:idx]))) {
					continue
				}
			}
		}

		if cfg.MaxBodySize > 0 && int64(len(htmlBody)) > cfg.MaxBodySize {
			htmlBody = htmlBody[:cfg.MaxBodySize]
		}

		if len(htmlBody) == 0 {
			continue
		}

		atomic.AddInt64(&stats.InputRecords, 1)

		itemCh <- packItem{
			targetURI: rec.Header.TargetURI(),
			date:      rec.Header.Get("WARC-Date"),
			recordID:  rec.Header.RecordID(),
			htmlBody:  htmlBody,
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
		}

		rec := &warcpkg.Record{
			Header: hdr,
			Body:   bytes.NewReader(res.markdown),
		}

		// Each record in its own gzip member
		gz, err := kgzip.NewWriterLevel(f, kgzip.BestSpeed)
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

	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, outputPath)
}
