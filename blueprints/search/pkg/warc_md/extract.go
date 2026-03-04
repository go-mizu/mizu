package warc_md

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	warcpkg "github.com/go-mizu/mizu/blueprints/search/pkg/warc"
)

// ExtractConfig configures Phase 1: .warc.gz → warc_single/**/*.warc
type ExtractConfig struct {
	InputFiles  []string // .warc.gz files to process (each worker handles one file)
	OutputDir   string   // warc_single/ base directory
	Workers     int      // parallel workers (one per input file; 0 = min(len(files), NumCPU))
	Force       bool     // re-extract existing files
	StatusCode  int      // HTTP status filter (default: 200)
	MIMEFilter  string   // MIME type filter (default: "text/html")
	MaxBodySize int64    // max body bytes per record (default: no limit)
}

// rawRecord holds an extracted HTML record ready for writing.
type rawRecord struct {
	recordID string
	htmlBody []byte
}

// RunExtract streams each .warc.gz file and writes filtered HTML response
// records to individual files in OutputDir.
//
// Each output file (*.warc) contains the raw HTML body bytes — NOT a full WARC
// record. The .warc extension is a naming convention only.
//
// Multiple input files are processed in parallel (bounded by Workers).
// Within each file, reading is sequential (gzip stream constraint).
func RunExtract(ctx context.Context, cfg ExtractConfig, progressFn ProgressFunc) (*PhaseStats, error) {
	if len(cfg.InputFiles) == 0 {
		return &PhaseStats{}, nil
	}
	if cfg.Workers <= 0 {
		cfg.Workers = min(len(cfg.InputFiles), runtime.NumCPU())
		if cfg.Workers < 1 {
			cfg.Workers = 1
		}
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = 200
	}
	if cfg.MIMEFilter == "" {
		cfg.MIMEFilter = "text/html"
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return nil, err
	}

	var stats PhaseStats
	start := time.Now()

	stopMem := make(chan struct{})
	getPeakMB := trackPeakMem(stopMem)

	var stopProgress chan struct{}
	if progressFn != nil {
		stopProgress = make(chan struct{})
		go func() {
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
					done := atomic.LoadInt64(&stats.Files) + atomic.LoadInt64(&stats.Skipped) + atomic.LoadInt64(&stats.Errors)
					progressFn(done, 0, atomic.LoadInt64(&stats.Errors),
						atomic.LoadInt64(&stats.ReadBytes), atomic.LoadInt64(&stats.WriteBytes),
						time.Since(start), getPeakMB())
				case <-stopProgress:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// semaphore for bounding file-level parallelism
	sem := make(chan struct{}, cfg.Workers)
	var wg sync.WaitGroup
	var firstErr atomic.Value // stores first error (type error)

	for _, warcPath := range cfg.InputFiles {
		if ctx.Err() != nil {
			break
		}
		warcPath := warcPath
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := extractOneFile(ctx, warcPath, cfg, &stats); err != nil && ctx.Err() == nil {
				firstErr.Store(err)
			}
		}()
	}
	wg.Wait()

	if stopProgress != nil {
		close(stopProgress)
	}
	close(stopMem)
	stats.Duration = time.Since(start)
	stats.PeakMemMB = getPeakMB()

	var retErr error
	if v := firstErr.Load(); v != nil {
		retErr = v.(error)
	}
	if ctx.Err() != nil {
		retErr = ctx.Err()
	}
	return &stats, retErr
}

// extractOneFile processes a single .warc.gz file, writing matching HTML records.
func extractOneFile(ctx context.Context, warcPath string, cfg ExtractConfig, stats *PhaseStats) error {
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
			bodyReader = io.LimitReader(rec.Body, cfg.MaxBodySize+8192) // +8192 for HTTP headers
		}
		bodyBytes, err := io.ReadAll(bodyReader)
		if err != nil {
			atomic.AddInt64(&stats.Errors, 1)
			continue
		}
		atomic.AddInt64(&stats.ReadBytes, int64(len(bodyBytes)))

		// Parse HTTP response: status, MIME, HTML body
		status, mime, htmlBody := parseHTTPResponse(bodyBytes)
		if cfg.StatusCode != 0 && status != cfg.StatusCode {
			continue
		}
		if cfg.MIMEFilter != "" && !strings.Contains(mime, strings.SplitN(cfg.MIMEFilter, "/", 2)[0]) {
			continue
		}

		// Cap HTML body at MaxBodySize
		if cfg.MaxBodySize > 0 && int64(len(htmlBody)) > cfg.MaxBodySize {
			htmlBody = htmlBody[:cfg.MaxBodySize]
		}

		recordID := rec.Header.RecordID()
		outPath := WARCSingleFilePath(cfg.OutputDir, recordID)

		if !cfg.Force {
			if _, err := os.Stat(outPath); err == nil {
				atomic.AddInt64(&stats.Skipped, 1)
				continue
			}
		}

		if err := writeRawFile(outPath, htmlBody); err != nil {
			atomic.AddInt64(&stats.Errors, 1)
			continue
		}
		atomic.AddInt64(&stats.WriteBytes, int64(len(htmlBody)))
		atomic.AddInt64(&stats.Files, 1)
	}
	return r.Err()
}

// parseHTTPResponse extracts (statusCode, mimeType, htmlBody) from a raw HTTP
// response byte slice (status line + headers + body).
// Returns ("", 0, nil) on parse failure.
func parseHTTPResponse(data []byte) (status int, mime string, body []byte) {
	r := bufio.NewReader(bytes.NewReader(data))

	// Status line: "HTTP/1.1 200 OK\r\n"
	line, err := r.ReadString('\n')
	if err != nil {
		return 0, "", nil
	}
	parts := strings.SplitN(strings.TrimRight(line, "\r\n"), " ", 3)
	if len(parts) >= 2 {
		status, _ = strconv.Atoi(parts[1])
	}

	// HTTP headers
	tp := textproto.NewReader(r)
	hdrs, _ := tp.ReadMIMEHeader()
	ct := hdrs.Get("Content-Type")
	mime = strings.SplitN(ct, ";", 2)[0]
	mime = strings.TrimSpace(mime)

	// Remaining bytes are the HTML body
	body, _ = io.ReadAll(r)
	return
}

// writeRawFile writes data to path atomically (via temp file + rename).
func writeRawFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}
