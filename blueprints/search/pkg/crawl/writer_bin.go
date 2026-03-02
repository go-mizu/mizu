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
	// binChanCap is the fallback channel capacity per shard when availMB=0.
	binChanCap = 32768 // 32K records

	// binSegQueueCap is the number of completed segment paths buffered for the drain goroutine.
	binSegQueueCap = 16

	// binSegDefaultMB is the segment rotation threshold in megabytes.
	binSegDefaultMB = 64

	// binFlushBufSize is the bufio.Writer buffer size for each segment file.
	binFlushBufSize = 512 * 1024 // 512 KB

	// binDefaultShards is the default number of parallel write/drain lanes.
	// N shards → N concurrent flushers + N drainers. Partial failure isolation:
	// if shard k's DuckDB write blocks, only 1/N of workers are affected.
	binDefaultShards = 4

	// binPauseCheckInterval is how often the flusher re-evaluates heap pressure.
	// ReadMemStats acquires the GC lock — calling it per-record at 3K/s adds
	// significant contention. Rate-limiting to once per second is sufficient.
	binPauseCheckInterval = time.Second
)

// pauser rate-limits expensive heap-pressure checks to once per second.
// The hot path (flusher goroutine) calls this per record; ReadMemStats must not
// be called on every record or it dominates flusher CPU under high throughput.
type pauser struct {
	lastNs  atomic.Int64 // last ReadMemStats call (unix nanoseconds)
	heapOK  atomic.Bool  // true when heap exceeds 70% of GOMEMLIMIT
}

// check returns true when heap exceeds 70% of GOMEMLIMIT.
// ReadMemStats is called at most once per binPauseCheckInterval.
func (p *pauser) check() bool {
	now := time.Now().UnixNano()
	if now-p.lastNs.Load() < int64(binPauseCheckInterval) {
		return p.heapOK.Load()
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	limit := uint64(debug.SetMemoryLimit(-1))
	over := limit > 0 && ms.HeapAlloc > limit*7/10
	p.lastNs.Store(now)
	p.heapOK.Store(over)
	return over
}

// binChanCapFromMem computes the result channel capacity per shard.
// Targets 5% of available RAM for the total write buffer, split across shards.
// avgRecordKB: estimated bytes per Result record (default 8).
func binChanCapFromMem(availMB, avgRecordKB, shards int) int {
	if shards <= 0 {
		shards = 1
	}
	if availMB <= 0 || avgRecordKB <= 0 {
		return max(binChanCap/shards, 4096)
	}
	total := availMB * 1024 / 20 / avgRecordKB
	perShard := total / shards
	return clamp(perShard, 4096, 65536)
}

// binWriterShard holds per-lane state for one parallel flusher/drainer pair.
// All flusher-state fields are accessed only by the flusher goroutine; no lock needed.
type binWriterShard struct {
	idx      int          // shard index, used in segment file naming
	segDir   string       // directory for this shard's segment files
	maxBytes int64        // rotate threshold
	rdb      ResultWriter // drain destination

	ch    chan Result // inbound results from workers
	segCh chan string // completed segment paths for this shard's drainer

	// flusher-only state
	cur      *os.File
	curEnc   *bseg.Encoder
	curBytes int64
	curPath  string
	segNum   int
	jr       bseg.Record // reused per-record buffer (zero alloc)
}

// BinSegWriter writes results to rotating binary segment files (.bseg2).
//
// Architecture: N parallel shards each have their own (channel → flusher → segCh → drainer → DuckDB)
// pipeline. Results are routed to shards by domain FNV-1a hash, so all URLs from one domain
// always go to the same shard (consistent routing, in-domain ordering preserved).
//
// Key properties:
//   - Partial-failure isolation: a slow DuckDB write blocks only 1/N of workers.
//   - Zero per-record heap alloc: bseg.Encoder writes directly to bufio.Writer; bseg.Record reused.
//   - Heap pressure checked at most once/second (not per-record) to avoid GC lock contention.
//   - Legacy .bseg (gob) files supported via drainSegGob fallback.
//
// BinSegWriter implements crawl.ResultWriter.
type BinSegWriter struct {
	shards   []*binWriterShard
	memPause pauser

	// Aggregated counters shared across shards (atomics — safe for concurrent use).
	written  atomic.Int64 // records written to segment files
	drained  atomic.Int64 // records successfully drained to DuckDB
	segCount atomic.Int32 // total segments created
	pendSeg  atomic.Int32 // segments queued for drain (not yet drained)

	wg sync.WaitGroup // waits for all 2×N goroutines (flusher + drainer per shard)
}

// NewBinSegWriter creates a BinSegWriter that writes to segDir.
//
//   - maxMB: segment size threshold (0 → default 64 MB).
//   - availMB: available RAM in MB for channel capacity tuning (0 → use default).
//   - rdb: the ResultDB to drain completed segments into (nil = accumulate on disk).
//
// Uses the default shard count (binDefaultShards = 4).
func NewBinSegWriter(segDir string, maxMB int, availMB int, rdb ResultWriter) (*BinSegWriter, error) {
	return NewBinSegWriterN(segDir, maxMB, availMB, 0, rdb)
}

// NewBinSegWriterN creates a BinSegWriter with an explicit shard count.
// shards=0 uses the default (binDefaultShards = 4).
func NewBinSegWriterN(segDir string, maxMB int, availMB int, shards int, rdb ResultWriter) (*BinSegWriter, error) {
	if err := os.MkdirAll(segDir, 0o755); err != nil {
		return nil, fmt.Errorf("bin writer: creating segment dir: %w", err)
	}
	if maxMB <= 0 {
		maxMB = binSegDefaultMB
	}
	if shards <= 0 {
		shards = binDefaultShards
	}
	chanPerShard := binChanCapFromMem(availMB, 8, shards)
	maxBytes := int64(maxMB) * 1024 * 1024

	w := &BinSegWriter{
		shards: make([]*binWriterShard, shards),
	}
	for i := range shards {
		s := &binWriterShard{
			idx:      i,
			segDir:   segDir,
			maxBytes: maxBytes,
			rdb:      rdb,
			ch:       make(chan Result, chanPerShard),
			segCh:    make(chan string, binSegQueueCap),
		}
		w.shards[i] = s
		w.wg.Add(2) // flusher + drainer
		go w.flusher(s)
		go w.drainer(s)
	}
	return w, nil
}

// Add enqueues a result for writing. Routes to the appropriate shard by domain FNV-1a hash.
// Blocks only when that shard's channel is full (flusher can't keep up with disk writes).
// With N shards, a slow DuckDB write on shard k blocks only domains mapped to shard k.
func (w *BinSegWriter) Add(r Result) {
	w.shards[domainShardIdx(r.Domain, len(w.shards))].ch <- r
}

// domainShardIdx returns the shard index for a domain using FNV-1a hash.
// Consistent routing: the same domain always goes to the same shard.
func domainShardIdx(domain string, n int) int {
	h := uint32(2166136261)
	for i := 0; i < len(domain); i++ {
		h ^= uint32(domain[i])
		h *= 16777619
	}
	return int(h % uint32(n))
}

// Flush is a no-op for BinSegWriter; flusher goroutines maintain continuous writes.
func (w *BinSegWriter) Flush(_ context.Context) error { return nil }

// Close drains all write channels, rotates final segments, and waits for all drain
// goroutines to finish. Returns only after all records are written to disk and drained.
func (w *BinSegWriter) Close() error {
	for _, s := range w.shards {
		close(s.ch) // each shard's flusher exits on channel close, then closes segCh
	}
	w.wg.Wait()
	return nil
}

// Written returns the total number of records serialized to segment files.
func (w *BinSegWriter) Written() int64 { return w.written.Load() }

// Drained returns the total number of records drained to DuckDB.
func (w *BinSegWriter) Drained() int64 { return w.drained.Load() }

// PendingSegs returns the total number of segment files waiting to be drained.
func (w *BinSegWriter) PendingSegs() int32 { return w.pendSeg.Load() }

// SegCount returns the total number of segment files created across all shards.
func (w *BinSegWriter) SegCount() int32 { return w.segCount.Load() }

// ChanFill returns the maximum channel fill level across all shards [0.0, 1.0].
// Values near 1.0 indicate at least one shard's flusher cannot keep up (disk I/O bottleneck).
func (w *BinSegWriter) ChanFill() float64 {
	var maxFill float64
	for _, s := range w.shards {
		c := cap(s.ch)
		if c == 0 {
			continue
		}
		f := float64(len(s.ch)) / float64(c)
		if f > maxFill {
			maxFill = f
		}
	}
	return maxFill
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

	var total int64
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || (filepath.Ext(name) != ".bseg2" && filepath.Ext(name) != ".bseg") {
			continue
		}
		path := filepath.Join(segDir, name)
		fmt.Fprintf(os.Stderr, "[binwriter] drain leftover: %s\n", name)
		n := drainSegFile(path, rdb)
		total += n
	}
	if total > 0 && rdb != nil {
		rdb.Flush(context.Background())
	}
	return total, nil
}

// ── flusher ──────────────────────────────────────────────────────────────────

// flusher drains s.ch, encodes each Result as a bseg record, and writes to
// the current segment file. When the segment reaches maxBytes, it's closed and
// its path sent to s.segCh for the drain goroutine.
func (w *BinSegWriter) flusher(s *binWriterShard) {
	defer func() {
		w.closeSeg(s)  // flush + close the final segment
		close(s.segCh) // signals drainer that no more segments are coming
		w.wg.Done()
	}()
	for r := range s.ch {
		// Apply back-pressure: sleep briefly when heap is near limit AND channel is near full.
		// memPause.check() is rate-limited to once/second to avoid GC lock overhead.
		if w.memPause.check() && w.ChanFill() > 0.9 {
			time.Sleep(100 * time.Millisecond)
		}
		w.writeOne(s, r)
	}
}

func (w *BinSegWriter) writeOne(s *binWriterShard, r Result) {
	// Rotate if current segment is at capacity or not yet opened.
	if s.cur == nil || s.curBytes >= s.maxBytes {
		w.rotateSeg(s)
		if s.cur == nil {
			return // failed to open new segment — skip record rather than block
		}
	}

	// Reuse s.jr to avoid a bseg.Record allocation per record.
	jr := &s.jr
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

	if err := s.curEnc.Encode(jr); err != nil {
		return
	}
	s.curBytes += recEstimatedSize(jr)
	w.written.Add(1)
}

// recEstimatedSize returns the estimated encoded byte size for a bseg record.
func recEstimatedSize(r *bseg.Record) int64 {
	const fixed = 4 + 1 + 4 + 8 + 8 + 8 // rec_len + flags + status + content_len + fetch_ms + crawled_ms
	const strOverhead = 9 * 2             // 9 string fields × 2 bytes each for uint16 length
	return int64(fixed + strOverhead +
		len(r.URL) + len(r.ContentType) + len(r.BodyCID) + len(r.Title) +
		len(r.Description) + len(r.Language) + len(r.Domain) + len(r.RedirectURL) + len(r.Error))
}

// rotateSeg closes the current segment (if any) and opens a new one.
// Segment filenames include the shard index to avoid collisions between shards.
func (w *BinSegWriter) rotateSeg(s *binWriterShard) {
	w.closeSeg(s)
	s.segNum++
	path := filepath.Join(s.segDir, fmt.Sprintf("s%02d_seg_%06d.bseg2", s.idx, s.segNum))
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[binwriter] failed to create segment %s: %v\n", path, err)
		s.cur = nil
		return
	}
	enc, err := bseg.NewEncoder(f, binFlushBufSize)
	if err != nil {
		f.Close()
		os.Remove(path)
		fmt.Fprintf(os.Stderr, "[binwriter] failed to init encoder %s: %v\n", path, err)
		s.cur = nil
		return
	}
	s.cur = f
	s.curEnc = enc
	s.curBytes = 0
	s.curPath = path
	w.segCount.Add(1)
}

// closeSeg flushes and closes the current segment file, then queues its path
// for the drain goroutine. Idempotent: safe to call when s.cur == nil.
func (w *BinSegWriter) closeSeg(s *binWriterShard) {
	if s.cur == nil {
		return
	}
	path := s.curPath
	curBytes := s.curBytes
	s.curEnc.Close() // flushes bufio + patches rec_count + closes file
	s.cur = nil
	s.curEnc = nil // release encoder's internal buffer (GC-eligible immediately)

	if curBytes > 0 {
		w.pendSeg.Add(1)
		s.segCh <- path // may block if drain is 16+ segments behind (expected: never)
	}
	s.curBytes = 0
}

// ── drainer ──────────────────────────────────────────────────────────────────

// drainer reads completed segment paths from s.segCh and drains each one into the
// destination ResultDB. Segment files are deleted after successful drain.
// rdb.Flush is called once after all segments are drained to amortize DuckDB overhead.
func (w *BinSegWriter) drainer(s *binWriterShard) {
	defer func() {
		if s.rdb != nil {
			s.rdb.Flush(context.Background())
		}
		w.wg.Done()
	}()
	for segPath := range s.segCh {
		count := drainSegFile(segPath, s.rdb)
		w.drained.Add(count)
		w.pendSeg.Add(-1)
	}
}

// drainSegFile reads a binary segment file (.bseg2 or legacy .bseg), calls rdb.Add
// for each record, then deletes the file. Returns the number of records successfully decoded.
// Package-level so DrainLeftovers can use it without a BinSegWriter instance.
func drainSegFile(path string, rdb ResultWriter) int64 {
	defer os.Remove(path) // always delete — even on partial read

	if rdb == nil {
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
			return drainSegGobFile(path, f, rdb)
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
		rdb.Add(bsegToResult(&rec))
		count++
	}
	return count
}

// drainSegGobFile reads a legacy gob-encoded .bseg segment file. f must be seekable.
// It seeks back to offset 0 before decoding.
func drainSegGobFile(path string, f *os.File, rdb ResultWriter) int64 {
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
		rdb.Add(rec.toResult())
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
// Kept only for backward-compatible drainSegGobFile reads.
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
