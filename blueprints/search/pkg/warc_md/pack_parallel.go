package warc_md

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gzip "github.com/klauspost/compress/gzip"

	mdpkg "github.com/go-mizu/mizu/blueprints/search/pkg/markdown"
	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// gzReaderPool recycles gzip readers to avoid per-offset allocation of internal
// decompression buffers (~32 KB each). On a shard with 50,000 offsets, this
// eliminates 50,000 gzip reader allocations.
var gzReaderPool sync.Pool

// GzipMemberOffset represents one gzip member's position in a .warc.gz file.
type GzipMemberOffset struct {
	Offset int64 // byte offset from start of file
	Size   int64 // compressed size of the gzip member
}

// ScanGzipOffsets scans a .warc.gz file and returns the byte offset and size
// of each gzip member. This decompresses each member to find its boundary
// (gzip members have no size in their header).
//
// For Common Crawl WARC files, each gzip member contains exactly one WARC record.
func ScanGzipOffsets(warcPath string) ([]GzipMemberOffset, error) {
	f, err := os.Open(warcPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cr := &countReader{r: f}
	br := bufio.NewReaderSize(cr, 64*1024)

	var offsets []GzipMemberOffset

	// The approach: after each gz.Reset(br) + gz.Multistream(false) + io.Copy(io.Discard, gz),
	// the file position (cr.n - br.Buffered()) points to the start of the NEXT gzip member.
	// We track positions between members to compute offset and size.

	gz, err := gzip.NewReader(br)
	if err != nil {
		return nil, fmt.Errorf("gzip init: %w", err)
	}
	gz.Multistream(false)

	// The first member starts at offset 0.
	// After gzip.NewReader, the reader has consumed the gzip header,
	// but the member started at file offset 0.
	prevEnd := int64(0) // start of first member

	for {
		// Drain current member fully
		io.Copy(io.Discard, gz)

		// After draining, the position points past this member
		memberEnd := cr.n - int64(br.Buffered())
		offsets = append(offsets, GzipMemberOffset{
			Offset: prevEnd,
			Size:   memberEnd - prevEnd,
		})

		// Try to advance to next gzip member
		// Before Reset, check for magic bytes
		peek, peekErr := br.Peek(2)
		if peekErr != nil || peek[0] != 0x1f || peek[1] != 0x8b {
			break
		}
		prevEnd = cr.n - int64(br.Buffered())

		if err := gz.Reset(br); err != nil {
			break
		}
		gz.Multistream(false)
	}

	return offsets, nil
}

type countReader struct {
	r io.Reader
	n int64
}

func (cr *countReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

// RunPackParallel executes the pack pipeline using pre-computed gzip member
// offsets for parallel reading. Each worker opens its own file descriptor,
// seeks to a gzip member, decompresses, parses WARC, extracts HTML,
// converts to markdown, and sends to the writer.
//
// This achieves near-linear scaling with CPU cores since there's no shared
// reader bottleneck.
func RunPackParallel(ctx context.Context, cfg PackConfig, offsets []GzipMemberOffset, progressFn ProgressFunc) (*PackStats, error) {
	if len(offsets) == 0 {
		return &PackStats{}, nil
	}
	if len(cfg.InputFiles) == 0 {
		return &PackStats{}, nil
	}
	warcPath := cfg.InputFiles[0] // parallel mode works on single files

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

	if !cfg.Force {
		if _, err := os.Stat(cfg.OutputPath); err == nil {
			return &PackStats{Skipped: 1}, nil
		}
	}
	if err := os.MkdirAll(osDir(cfg.OutputPath), 0o755); err != nil {
		return nil, err
	}

	prevGOGC := debug.SetGCPercent(200)
	defer debug.SetGCPercent(prevGOGC)

	// GOMEMLIMIT: hard ceiling prevents OOM kills. Without this, Go runtime
	// can grow to 3+ GB during gzip offset scanning bursts. 600 MiB leaves
	// ~200 MB for non-heap (file descriptors, OS buffers, goroutine stacks).
	// The runtime aggressively GCs near this limit, keeping RSS predictable.
	prevMemLimit := debug.SetMemoryLimit(600 << 20) // 600 MiB
	defer debug.SetMemoryLimit(prevMemLimit)

	var stats PackStats
	start := time.Now()
	stopMem := make(chan struct{})
	getPeakMB := trackPeakMem(stopMem)

	// Select converter
	convertFn := mdpkg.Convert
	if cfg.LightConvert {
		convertFn = mdpkg.ConvertUltraLight // tokenizer-based, no DOM tree
	} else if cfg.FastConvert {
		convertFn = mdpkg.ConvertFast
	}

	// Result channel for writer
	resultCh := make(chan packResult, cfg.Workers*2)
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

	// Writer goroutine (same as sequential)
	go func() {
		writerDone <- packWriteFile(cfg.OutputPath, resultCh, &stats)
	}()

	// Parallel workers — each processes a batch of offsets
	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.Workers)

	for _, off := range offsets {
		if ctx.Err() != nil {
			break
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(o GzipMemberOffset) {
			defer wg.Done()
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			result, err := processOneOffset(warcPath, o, cfg, convertFn)
			if err != nil {
				atomic.AddInt64(&stats.Errors, 1)
				return
			}
			if result == nil {
				return
			}

			atomic.AddInt64(&stats.InputRecords, 1)

			if result.hasContent && result.markdown != "" {
				resultCh <- *result
			} else {
				atomic.AddInt64(&stats.Errors, 1)
			}
		}(off)
	}

	wg.Wait()
	close(resultCh)

	writeErr := <-writerDone

	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	stats.Duration = time.Since(start)
	stats.PeakMemMB = getPeakMB()

	if writeErr != nil {
		return &stats, writeErr
	}
	if ctx.Err() != nil {
		return &stats, ctx.Err()
	}
	return &stats, nil
}

// processOneOffset reads and converts a single WARC record from a gzip member
// at the given offset. Opens its own file descriptor for true parallelism.
func processOneOffset(warcPath string, off GzipMemberOffset, cfg PackConfig, convertFn func([]byte, string) mdpkg.Result) (*packResult, error) {
	f, err := os.Open(warcPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Seek(off.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	// Reuse gzip reader from pool when possible.
	var gz *gzip.Reader
	if pooled, ok := gzReaderPool.Get().(*gzip.Reader); ok {
		if err := pooled.Reset(f); err != nil {
			gz, err = gzip.NewReader(f)
			if err != nil {
				return nil, err
			}
		} else {
			gz = pooled
		}
	} else {
		gz, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
	}
	defer func() {
		gz.Close()
		gzReaderPool.Put(gz)
	}()

	// Use the standard WARC reader to parse the record
	wr := warcpkg.NewReader(gz)
	if !wr.Next() {
		return nil, nil // no record in this member
	}
	rec := wr.Record()

	if rec.Header.Type() != warcpkg.TypeResponse {
		return nil, nil
	}

	// Read body using pooled buffer.
	buf := bodyBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	var bodyReader io.Reader = rec.Body
	if cfg.MaxBodySize > 0 {
		bodyReader = io.LimitReader(rec.Body, cfg.MaxBodySize+8192)
	}
	_, err = buf.ReadFrom(bodyReader)
	if err != nil {
		bodyBufPool.Put(buf)
		return nil, err
	}
	bodyBytes := buf.Bytes()

	status, mime, htmlBody := parseHTTPResponseFast(bodyBytes)
	if cfg.StatusCode != 0 && status != cfg.StatusCode {
		bodyBufPool.Put(buf)
		return nil, nil
	}
	if cfg.MIMEFilter != "" && mime != cfg.MIMEFilter {
		if idx := strings.IndexByte(cfg.MIMEFilter, '/'); idx >= 0 {
			if !strings.HasPrefix(mime, cfg.MIMEFilter[:idx]) {
				bodyBufPool.Put(buf)
				return nil, nil
			}
		} else {
			bodyBufPool.Put(buf)
			return nil, nil
		}
	}
	if cfg.MaxBodySize > 0 && int64(len(htmlBody)) > cfg.MaxBodySize {
		htmlBody = htmlBody[:cfg.MaxBodySize]
	}
	if len(htmlBody) == 0 {
		bodyBufPool.Put(buf)
		return nil, nil
	}

	// Convert HTML → Markdown (htmlBody is a sub-slice of the pooled buffer).
	res := convertFn(htmlBody, "")
	bodyBufPool.Put(buf) // return buffer now that conversion read the data

	if !res.HasContent || res.Markdown == "" {
		return nil, nil
	}

	return &packResult{
		targetURI:  rec.Header.TargetURI(),
		date:       rec.Header.Get("WARC-Date"),
		refersTo:   rec.Header.RecordID(),
		markdown:   res.Markdown,
		hasContent: true,
		htmlLen:    len(htmlBody),
	}, nil
}

// osDir is filepath.Dir without importing filepath (already imported in pack.go).
func osDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
