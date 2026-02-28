package crawl

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/bseg"
)

const (
	// binChanCap is the fallback channel capacity when availMB=0.
	// At 10K writes/s this provides ~3.2 seconds of headroom before any worker blocks.
	binChanCap = 32768 // 32K records

	// binSegQueueCap is the number of completed segment paths buffered for the drain goroutine.
	binSegQueueCap = 16

	// binSegDefaultMB is the segment rotation threshold in megabytes.
	binSegDefaultMB = 64

	// binFlushBufSize is the bufio.Writer buffer size for each segment file.
	binFlushBufSize = 512 * 1024 // 512 KB
)

// binChanCapFromMem computes the result channel capacity.
// Targets 5% of available RAM for the write buffer.
// avgRecordKB: estimated bytes per Result record (default 8).
func binChanCapFromMem(availMB, avgRecordKB int) int {
	if availMB <= 0 || avgRecordKB <= 0 {
		return binChanCap // default
	}
	v := availMB * 1024 / 20 / avgRecordKB
	return clamp(v, 4096, 65536)
}

// BinSegWriter writes results to rotating binary segment files (.bseg2).
//
// Write path: HTTP workers → ch (RAM-proportional buffer) → flusher goroutine → bseg-encoded file write.
// Drain path: completed segments → drain goroutine → rdb.Add() → DuckDB (background).
//
// Key design properties:
//   - Zero per-record heap alloc: bseg.Encoder writes directly to bufio.Writer; bseg.Record reused.
//   - After segment close: curEnc=nil — buffers are GC-eligible immediately.
//   - Channel memory bounded: cap × ~8KB avg record.
//   - Legacy .bseg (gob) files supported via drainSegGob fallback.
//
// BinSegWriter implements crawl.ResultWriter.
type BinSegWriter struct {
	segDir   string              // directory for binary segment files
	maxBytes int64               // rotate segment when it reaches this size
	rdb      ResultWriter // drain destination (nil = no drain, segments left on disk)

	ch    chan Result // primary write channel, buffered
	segCh chan string           // completed segment paths for drain goroutine

	// flusher state — accessed only by the flusher goroutine, no lock needed.
	cur      *os.File
	curEnc   *bseg.Encoder // one encoder per segment; nil when no segment is open
	curBytes int64
	curPath  string
	segNum   int
	jr       bseg.Record // reused record buffer; avoids a bseg.Record allocation per writeOne call

	written  atomic.Int64 // records written to segment files
	drained  atomic.Int64 // records successfully drained to DuckDB
	segCount atomic.Int32 // total segments created
	pendSeg  atomic.Int32 // segments queued for drain (not yet drained)

	wg sync.WaitGroup
}

// NewBinSegWriter creates a BinSegWriter that writes to segDir.
//
//   - maxMB: segment size threshold (0 → default 64 MB).
//   - availMB: available RAM in MB for channel capacity tuning (0 → use default 32768).
//   - rdb: the ResultDB to drain completed segments into (nil = accumulate on disk).
func NewBinSegWriter(segDir string, maxMB int, availMB int, rdb ResultWriter) (*BinSegWriter, error) {
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return nil, fmt.Errorf("bin writer: creating segment dir: %w", err)
	}
	if maxMB <= 0 {
		maxMB = binSegDefaultMB
	}
	chanCap := binChanCapFromMem(availMB, 8)
	w := &BinSegWriter{
		segDir:   segDir,
		maxBytes: int64(maxMB) * 1024 * 1024,
		rdb:      rdb,
		ch:       make(chan Result, chanCap),
		segCh:    make(chan string, binSegQueueCap),
	}
	w.wg.Add(2) // flusher + drainer
	go w.flusher()
	go w.drainer()
	return w, nil
}

// Add enqueues a result for writing. It blocks only when the channel is full
// (which only occurs if the flusher goroutine cannot keep up with disk writes).
// Under normal operation this channel stays near-empty.
func (w *BinSegWriter) Add(r Result) {
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

// ChanFill returns the fractional fill level of the write channel [0.0, 1.0].
// Values near 1.0 indicate the flusher cannot keep up (disk I/O bottleneck).
func (w *BinSegWriter) ChanFill() float64 {
	c := cap(w.ch)
	if c == 0 {
		return 0
	}
	return float64(len(w.ch)) / float64(c)
}

// DrainLeftovers drains any leftover .bseg2 and .bseg segment files from a previous
// crashed run into rdb, then deletes them. Call before starting a new crawl to ensure
// no results are lost from the prior run.
//
// Returns the total number of records drained.
func DrainLeftovers(segDir string, rdb ResultWriter) (int64, error) {
	entries, err := os.ReadDir(segDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // no segment dir — nothing to drain
		}
		return 0, fmt.Errorf("drain leftovers: readdir %s: %w", segDir, err)
	}

	// Reuse a temporary BinSegWriter shell for its drainSeg method.
	w := &BinSegWriter{rdb: rdb}
	var total int64
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || (filepath.Ext(name) != ".bseg2" && filepath.Ext(name) != ".bseg") {
			continue
		}
		path := filepath.Join(segDir, name)
		fmt.Fprintf(os.Stderr, "[binwriter] drain leftover: %s\n", name)
		n := w.drainSeg(path) // drainSeg deletes the file on return
		total += n
	}
	if total > 0 && rdb != nil {
		rdb.Flush(context.Background())
	}
	return total, nil
}

// shouldPause returns true when heap usage exceeds 70% of GOMEMLIMIT and the
// write channel is more than 90% full. Used by the flusher to apply back-pressure.
func (w *BinSegWriter) shouldPause() bool {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	limit := uint64(debug.SetMemoryLimit(-1))
	return limit > 0 && ms.HeapAlloc > limit*7/10 && w.ChanFill() > 0.9
}

// ── flusher ──────────────────────────────────────────────────────────────────

// flusher drains w.ch, encodes each Result as a bseg record, and writes to
// the current segment file. When the segment reaches maxBytes, it's closed and
// its path sent to segCh for the drain goroutine.
func (w *BinSegWriter) flusher() {
	defer func() {
		w.closeCurrentSeg() // flush + close the final segment
		close(w.segCh)      // signals drainer that no more segments are coming
		w.wg.Done()
	}()
	for r := range w.ch {
		if w.shouldPause() {
			time.Sleep(100 * time.Millisecond)
		}
		w.writeOne(r)
	}
}

func (w *BinSegWriter) writeOne(r Result) {
	// Rotate if current segment is at capacity or not yet opened.
	if w.cur == nil || w.curBytes >= w.maxBytes {
		w.rotateSeg()
		if w.cur == nil {
			return // failed to open new segment — skip record rather than block
		}
	}

	// Reuse w.jr to avoid a bseg.Record allocation per record.
	jr := &w.jr
	jr.URL         = binSanitize(r.URL)
	jr.StatusCode  = int32(r.StatusCode)
	jr.ContentLen  = r.ContentLength
	jr.BodyCID     = r.BodyCID
	jr.Title       = binSanitize(r.Title)
	jr.Description = binSanitize(r.Description)
	jr.Language    = binSanitize(r.Language)
	jr.Domain      = binSanitize(r.Domain)
	jr.RedirectURL = binSanitize(r.RedirectURL)
	jr.FetchMs     = r.FetchTimeMs
	jr.CrawledMs   = r.CrawledAt.UnixMilli()
	jr.Error       = binSanitize(r.Error)
	jr.ContentType = binSanitize(r.ContentType)
	jr.Failed      = r.Error != ""

	if err := w.curEnc.Encode(jr); err != nil {
		return
	}
	w.curBytes += recEstimatedSize(jr)
	w.written.Add(1)
}

// recEstimatedSize returns the estimated encoded byte size for a bseg record.
// This is exact for the bseg format:
//
//	4 (rec_len) + 1 (flags) + 4 (status) + 8 (content_len) + 8 (fetch_ms) + 8 (crawled_ms)
//	+ 9 string fields × 2 (uint16 length prefix) + sum of string byte lengths
func recEstimatedSize(r *bseg.Record) int64 {
	const fixed = 4 + 1 + 4 + 8 + 8 + 8 // rec_len + flags + status + content_len + fetch_ms + crawled_ms
	const strOverhead = 9 * 2             // 9 string fields × 2 bytes each for uint16 length
	return int64(fixed + strOverhead +
		len(r.URL) + len(r.ContentType) + len(r.BodyCID) + len(r.Title) +
		len(r.Description) + len(r.Language) + len(r.Domain) + len(r.RedirectURL) + len(r.Error))
}

// rotateSeg closes the current segment (if any) and opens a new one.
func (w *BinSegWriter) rotateSeg() {
	w.closeCurrentSeg()
	w.segNum++
	path := filepath.Join(w.segDir, fmt.Sprintf("seg_%06d.bseg2", w.segNum))
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[binwriter] failed to create segment %s: %v\n", path, err)
		w.cur = nil
		return
	}
	enc, err := bseg.NewEncoder(f, binFlushBufSize)
	if err != nil {
		f.Close()
		os.Remove(path)
		fmt.Fprintf(os.Stderr, "[binwriter] failed to init encoder %s: %v\n", path, err)
		w.cur = nil
		return
	}
	w.cur = f
	w.curEnc = enc
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
	path := w.curPath
	curBytes := w.curBytes
	w.curEnc.Close() // flushes bufio + patches rec_count + closes file
	w.cur = nil
	w.curEnc = nil // release encoder's internal buffer (GC-eligible immediately)

	if curBytes > 0 {
		w.pendSeg.Add(1)
		w.segCh <- path // may block if drain is 16+ segments behind (expected: never)
	}
	w.curBytes = 0
}

// ── drainer ──────────────────────────────────────────────────────────────────

// drainer reads completed segment paths from segCh and drains each one into the
// destination ResultDB. Segment files are deleted after successful drain.
// rdb.Flush is called once after all segments are drained to amortize DuckDB overhead.
// Calling Flush per-segment (inside drainSeg) caused ~16s blocking per segment across
// 16 shards, back-pressuring the flusher and filling the worker channel to 100%.
func (w *BinSegWriter) drainer() {
	defer func() {
		if w.rdb != nil {
			w.rdb.Flush(context.Background())
		}
		w.wg.Done()
	}()
	for segPath := range w.segCh {
		count := w.drainSeg(segPath)
		w.drained.Add(count)
		w.pendSeg.Add(-1)
	}
}

// drainSeg reads a binary segment file (.bseg2 or legacy .bseg), calls rdb.Add
// for each record, then deletes the file. Returns the number of records successfully decoded.
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

	dec, err := bseg.NewDecoder(f)
	if err != nil {
		if errors.Is(err, bseg.ErrBadMagic) || errors.Is(err, bseg.ErrBadVersion) {
			// Fallback: try legacy gob format (old .bseg files).
			return w.drainSegGob(path, f)
		}
		fmt.Fprintf(os.Stderr, "[binwriter] drain header %s: %v\n", path, err)
		return 0
	}

	var rec bseg.Record
	var count int64
	for {
		if err := dec.Decode(&rec); err != nil {
			break // EOF or corrupt
		}
		w.rdb.Add(bsegToResult(&rec))
		count++
	}
	return count
}

// drainSegGob reads a legacy gob-encoded .bseg segment file. f must be seekable.
// It seeks back to offset 0 before decoding.
func (w *BinSegWriter) drainSegGob(path string, f *os.File) int64 {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		fmt.Fprintf(os.Stderr, "[binwriter] drain gob seek %s: %v\n", path, err)
		return 0
	}

	dec := gob.NewDecoder(f)
	var rec binRecord
	var count int64
	for {
		if err := dec.Decode(&rec); err != nil {
			break // EOF or corrupt record
		}
		w.rdb.Add(rec.toResult())
		count++
	}
	return count
}

// bsegToResult converts a bseg.Record to a crawl.Result.
func bsegToResult(r *bseg.Record) Result {
	return Result{
		URL:           r.URL,
		StatusCode:    int(r.StatusCode),
		ContentType:   r.ContentType,
		ContentLength: r.ContentLen,
		BodyCID:       r.BodyCID,
		Title:         r.Title,
		Description:   r.Description,
		Language:      r.Language,
		Domain:        r.Domain,
		RedirectURL:   r.RedirectURL,
		FetchTimeMs:   r.FetchMs,
		CrawledAt:     time.UnixMilli(r.CrawledMs),
		Error:         r.Error,
	}
}

// ── legacy gob support ────────────────────────────────────────────────────────

// binRecord is the legacy gob serialization type for old .bseg files.
// Kept only for backward-compatible drainSegGob reads.
type binRecord struct {
	URL           string
	StatusCode    int
	ContentType   string
	ContentLength int64
	BodyCID       string
	Title         string
	Description   string
	Language      string
	Domain        string
	RedirectURL   string
	FetchTimeMs   int64
	CrawledAtMs   int64
	Error         string
	Failed        bool
}

func (r *binRecord) toResult() Result {
	return Result{
		URL:           r.URL,
		StatusCode:    r.StatusCode,
		ContentType:   r.ContentType,
		ContentLength: r.ContentLength,
		BodyCID:       r.BodyCID,
		Title:         r.Title,
		Description:   r.Description,
		Language:      r.Language,
		Domain:        r.Domain,
		RedirectURL:   r.RedirectURL,
		FetchTimeMs:   r.FetchTimeMs,
		CrawledAt:     time.UnixMilli(r.CrawledAtMs),
		Error:         r.Error,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// binSanitize removes null bytes and invalid UTF-8 sequences.
// Mirrors the sanitizeStr helper from the recrawler package.
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
