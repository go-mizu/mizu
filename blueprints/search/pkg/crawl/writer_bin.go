package crawl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
)

const (
	// binChanCap is the number of Result records buffered in the write channel.
	// At 10K writes/s this provides ~13 seconds of headroom before any worker blocks.
	binChanCap = 131072 // 128K records

	// binSegQueueCap is the number of completed segment paths buffered for the drain goroutine.
	// 16 segments × ~16s each = ~256s of drain lag before the flusher backs up.
	binSegQueueCap = 16

	// binSegDefaultMB is the segment rotation threshold in megabytes.
	binSegDefaultMB = 64

	// binFlushBufSize is the bufio.Writer buffer size for each segment file.
	binFlushBufSize = 512 * 1024 // 512 KB
)

// binResultJSON is the JSON serialization format for NDJSON segments.
// snake_case tags match DuckDB column names for direct read_json_auto ingestion.
// crawled_at is stored as Unix milliseconds to avoid RFC3339 parsing ambiguity.
type binResultJSON struct {
	URL           string `json:"url"`
	StatusCode    int    `json:"status_code"`
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
	Body          string `json:"body"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Language      string `json:"language"`
	Domain        string `json:"domain"`
	RedirectURL   string `json:"redirect_url"`
	FetchTimeMs   int64  `json:"fetch_time_ms"`
	CrawledAtMs   int64  `json:"crawled_at_ms"` // Unix milliseconds
	Error         string `json:"error"`
	Status        string `json:"status"` // "done" or "failed"
}

func (j *binResultJSON) toResult() recrawler.Result {
	return recrawler.Result{
		URL:           j.URL,
		StatusCode:    j.StatusCode,
		ContentType:   j.ContentType,
		ContentLength: j.ContentLength,
		Body:          j.Body,
		Title:         j.Title,
		Description:   j.Description,
		Language:      j.Language,
		Domain:        j.Domain,
		RedirectURL:   j.RedirectURL,
		FetchTimeMs:   j.FetchTimeMs,
		CrawledAt:     time.UnixMilli(j.CrawledAtMs),
		Error:         j.Error,
	}
}

// BinSegWriter writes results to rotating NDJSON segment files.
//
// Write path: HTTP workers → ch (128K buffer) → flusher goroutine → buffered file write.
// Drain path: completed segments → drain goroutine → rdb.Add() → DuckDB (background).
//
// HTTP workers are never blocked by DuckDB checkpoint pauses. The only backpressure is
// through the 128K channel (~13s buffer at 10K/s), giving drain ~256s of lag headroom
// before any worker blocks. Compare to the current DuckDB direct path (flushCh=2, blocks
// immediately when checkpoint fires).
//
// BinSegWriter implements crawl.ResultWriter.
type BinSegWriter struct {
	segDir   string               // directory for NDJSON segment files
	maxBytes int64                // rotate segment when it reaches this size
	rdb      *recrawler.ResultDB  // drain destination (nil = no drain, segments left on disk)

	ch    chan recrawler.Result // primary write channel, large buffer
	segCh chan string           // completed segment paths for drain goroutine

	// flusher state — accessed only by the flusher goroutine, no lock needed
	cur      *os.File
	curBuf   *bufio.Writer
	curBytes int64
	curPath  string
	segNum   int

	written  atomic.Int64 // records written to segment files
	drained  atomic.Int64 // records successfully drained to DuckDB
	segCount atomic.Int32 // total segments created
	pendSeg  atomic.Int32 // segments queued for drain (not yet drained)

	wg sync.WaitGroup
}

// NewBinSegWriter creates a BinSegWriter that writes to segDir.
//
//   - maxMB: segment size threshold (0 → default 64 MB).
//   - rdb: the ResultDB to drain completed segments into (nil = accumulate on disk).
func NewBinSegWriter(segDir string, maxMB int, rdb *recrawler.ResultDB) (*BinSegWriter, error) {
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return nil, fmt.Errorf("bin writer: creating segment dir: %w", err)
	}
	if maxMB <= 0 {
		maxMB = binSegDefaultMB
	}
	w := &BinSegWriter{
		segDir:   segDir,
		maxBytes: int64(maxMB) * 1024 * 1024,
		rdb:      rdb,
		ch:       make(chan recrawler.Result, binChanCap),
		segCh:    make(chan string, binSegQueueCap),
	}
	w.wg.Add(2) // flusher + drainer
	go w.flusher()
	go w.drainer()
	return w, nil
}

// Add enqueues a result for writing. It blocks only when the 128K channel is full
// (which only occurs if the flusher goroutine cannot keep up with disk writes).
// Under normal operation this channel stays near-empty.
func (w *BinSegWriter) Add(r recrawler.Result) {
	w.ch <- r
}

// Flush is a no-op for BinSegWriter; the flusher goroutine maintains continuous writes.
func (w *BinSegWriter) Flush(_ context.Context) error { return nil }

// Close drains the write channel, rotates the final segment, and waits for the drain
// goroutine to finish. Returns only after all records are written to disk and drained
// to the destination ResultDB (if configured).
func (w *BinSegWriter) Close() error {
	close(w.ch)  // signals flusher to finish; flusher will close(segCh) on exit
	w.wg.Wait()  // waits for both flusher and drainer to complete
	return nil
}

// Written returns the total number of records serialized to segment files.
func (w *BinSegWriter) Written() int64 { return w.written.Load() }

// Drained returns the total number of records drained to DuckDB.
func (w *BinSegWriter) Drained() int64 { return w.drained.Load() }

// PendingSegs returns the number of segment files waiting to be drained.
func (w *BinSegWriter) PendingSegs() int32 { return w.pendSeg.Load() }

// SegCount returns the total number of segment files created.
func (w *BinSegWriter) SegCount() int32 { return w.segCount.Load() }

// ── flusher ──────────────────────────────────────────────────────────────────

// flusher drains w.ch, serializes each Result as a JSON line, and writes to
// the current segment file. When the segment reaches maxBytes, it's closed and
// its path sent to segCh for the drain goroutine.
func (w *BinSegWriter) flusher() {
	defer func() {
		w.closeCurrentSeg() // flush + close the final segment
		close(w.segCh)      // signals drainer that no more segments are coming
		w.wg.Done()
	}()
	for r := range w.ch {
		w.writeOne(r)
	}
}

func (w *BinSegWriter) writeOne(r recrawler.Result) {
	// Rotate if current segment is at capacity or not yet opened.
	if w.cur == nil || w.curBytes >= w.maxBytes {
		w.rotateSeg()
		if w.cur == nil {
			return // failed to open new segment — skip record rather than block
		}
	}

	status := "done"
	if r.Error != "" {
		status = "failed"
	}
	jr := binResultJSON{
		URL:           binSanitize(r.URL),
		StatusCode:    r.StatusCode,
		ContentType:   binSanitize(r.ContentType),
		ContentLength: r.ContentLength,
		Body:          binSanitize(r.Body),
		Title:         binSanitize(r.Title),
		Description:   binSanitize(r.Description),
		Language:      binSanitize(r.Language),
		Domain:        binSanitize(r.Domain),
		RedirectURL:   binSanitize(r.RedirectURL),
		FetchTimeMs:   r.FetchTimeMs,
		CrawledAtMs:   r.CrawledAt.UnixMilli(),
		Error:         binSanitize(r.Error),
		Status:        status,
	}
	b, err := json.Marshal(jr)
	if err != nil {
		return
	}
	b = append(b, '\n')
	n, _ := w.curBuf.Write(b)
	w.curBytes += int64(n)
	w.written.Add(1)
}

// rotateSeg closes the current segment (if any) and opens a new one.
func (w *BinSegWriter) rotateSeg() {
	w.closeCurrentSeg()
	w.segNum++
	path := filepath.Join(w.segDir, fmt.Sprintf("seg_%06d.jsonl", w.segNum))
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[binwriter] failed to create segment %s: %v\n", path, err)
		w.cur = nil
		return
	}
	w.cur = f
	w.curBuf = bufio.NewWriterSize(f, binFlushBufSize)
	w.curBytes = 0
	w.curPath = path
	w.segCount.Add(1)
}

// closeCurrentSeg flushes and closes the current segment file, then queues its path
// for the drain goroutine. Idempotent: safe to call when w.cur == nil.
func (w *BinSegWriter) closeCurrentSeg() {
	if w.cur == nil {
		return
	}
	w.curBuf.Flush()
	w.cur.Close()
	path := w.curPath
	w.cur = nil
	w.curBuf = nil

	if w.curBytes > 0 {
		w.pendSeg.Add(1)
		w.segCh <- path // may block if drain is 16+ segments behind (expected: never)
	}
	w.curBytes = 0
}

// ── drainer ──────────────────────────────────────────────────────────────────

// drainer reads completed segment paths from segCh and drains each one into the
// destination ResultDB. Segment files are deleted after successful drain.
func (w *BinSegWriter) drainer() {
	defer w.wg.Done()
	for segPath := range w.segCh {
		count := w.drainSeg(segPath)
		w.drained.Add(count)
		w.pendSeg.Add(-1)
	}
}

// drainSeg reads a NDJSON segment file, calls rdb.Add for each record, then deletes
// the file. Returns the number of records successfully decoded.
func (w *BinSegWriter) drainSeg(path string) int64 {
	defer os.Remove(path) // always delete — even on partial read

	if w.rdb == nil {
		return 0 // drain disabled, just clean up the file
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[binwriter] drain open %s: %v\n", path, err)
		return 0
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var count int64
	for {
		var jr binResultJSON
		if err := dec.Decode(&jr); err != nil {
			break // EOF or corrupt record
		}
		w.rdb.Add(jr.toResult())
		count++
	}

	// Flush any accumulated batch in the ResultDB shards for this segment.
	w.rdb.Flush(context.Background())
	return count
}

// ── helpers ───────────────────────────────────────────────────────────────────

// binSanitize removes null bytes and invalid UTF-8 sequences.
// Mirrors recrawler.sanitizeStr but accessible in this package.
func binSanitize(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x00", "")
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	return s
}
