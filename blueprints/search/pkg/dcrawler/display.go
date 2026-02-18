package dcrawler

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Stats tracks live statistics for the domain crawl.
type Stats struct {
	success  atomic.Int64
	failed   atomic.Int64
	timeout  atomic.Int64
	blocked  atomic.Int64 // pages detected as soft 404 / anti-bot
	skipped  atomic.Int64 // URLs skipped by adaptive URL class filter
	bytes    atomic.Int64
	fetchMs  atomic.Int64
	inFlight atomic.Int64

	// HTTP status distribution
	statusMu sync.Mutex
	statuses map[int]int

	// Depth tracking
	depthMu sync.Mutex
	depths  map[int]int

	// Config
	Label    string
	MaxPages int

	startTime time.Time
	peakSpeed float64

	// Rolling speed (10s window)
	speedMu    sync.Mutex
	speedTicks []speedTick

	rollingFetchSpeed float64
	rollingByteSpeed  float64

	// Frontier stats (set externally)
	frontierLen func() int
	bloomCount  func() uint32
	linksFound  atomic.Int64
	reseeds     atomic.Int64
	retries     atomic.Int64
	continuous  bool

	// Retry tracking
	retryExhausted atomic.Int64 // URLs that exhausted max retry attempts
	retryQLen      func() int   // function to query retry queue length

	// Rod worker phase tracking
	rodPhases       sync.Map // int -> string (worker ID -> phase name)
	rodPhaseStart   sync.Map // int -> time.Time (worker ID -> phase start time)
	rodWorkerURL    sync.Map // int -> string (worker ID -> current URL)
	rodWorkerStart  sync.Map // int -> time.Time (worker ID -> item start time)
	rodTotalWorkers int          // total rod workers
	rodRestarts     atomic.Int64 // browser restart count
	useRod          bool         // whether rod mode is active

	// Freeze
	frozen   bool
	frozenAt time.Duration

	// TUI init log channel (non-nil when RunWithDisplay is used)
	initLog chan string
}

type speedTick struct {
	time    time.Time
	fetched int64
	bytes   int64
}

// NewStats creates a new stats tracker.
func NewStats(label string, maxPages int, continuous bool) *Stats {
	return &Stats{
		statuses:   make(map[int]int),
		depths:     make(map[int]int),
		Label:      label,
		MaxPages:   maxPages,
		continuous: continuous,
		startTime:  time.Now(),
	}
}

// SetFrontierFuncs sets functions to query frontier state for display.
func (s *Stats) SetFrontierFuncs(lenFn func() int, bloomFn func() uint32) {
	s.frontierLen = lenFn
	s.bloomCount = bloomFn
}

// SetRetryQLen sets the function to query retry queue length.
func (s *Stats) SetRetryQLen(fn func() int) {
	s.retryQLen = fn
}

// SetRodPhase sets the current phase for a rod worker. Empty string clears it.
func (s *Stats) SetRodPhase(workerID int, phase string) {
	if phase == "" {
		s.rodPhases.Delete(workerID)
		s.rodPhaseStart.Delete(workerID)
		s.rodWorkerURL.Delete(workerID)
		s.rodWorkerStart.Delete(workerID)
	} else {
		s.rodPhases.Store(workerID, phase)
		s.rodPhaseStart.Store(workerID, time.Now())
	}
}

// SetRodWorkerItem sets the current URL being processed by a rod worker.
func (s *Stats) SetRodWorkerItem(workerID int, url string) {
	s.rodWorkerURL.Store(workerID, url)
	s.rodWorkerStart.Store(workerID, time.Now())
}

// SetRodTotalWorkers sets the total number of rod workers for display.
func (s *Stats) SetRodTotalWorkers(n int) {
	s.rodTotalWorkers = n
}

// SetUseRod marks that rod mode is active for display purposes.
func (s *Stats) SetUseRod(v bool) {
	s.useRod = v
}

// RecordSuccess records a successful fetch.
// bytes/fetchMs are added BEFORE success count so readers always see
// bytes >= what success accounts for (prevents avg from being wrong).
func (s *Stats) RecordSuccess(statusCode int, bytesRecv int64, fetchMs int64) {
	s.bytes.Add(bytesRecv)
	s.fetchMs.Add(fetchMs)
	s.recordStatus(statusCode)
	s.success.Add(1) // last: readers see consistent bytes/success ratio
}

// RecordFailure records a failed fetch.
func (s *Stats) RecordFailure(statusCode int, isTimeout bool) {
	if isTimeout {
		s.timeout.Add(1)
	} else {
		s.failed.Add(1)
	}
	if statusCode > 0 {
		s.recordStatus(statusCode)
	}
}

// RecordBlocked records a page detected as soft 404 / anti-bot.
func (s *Stats) RecordBlocked() {
	s.blocked.Add(1)
}

// Blocked returns the count of blocked pages.
func (s *Stats) Blocked() int64 {
	return s.blocked.Load()
}

// RecordSkipped increments the skipped counter (adaptive URL class filter).
func (s *Stats) RecordSkipped() {
	s.skipped.Add(1)
}

// Skipped returns the count of URLs skipped by adaptive filtering.
func (s *Stats) Skipped() int64 {
	return s.skipped.Load()
}

// RecordDepth records a page crawled at a given depth.
func (s *Stats) RecordDepth(depth int) {
	s.depthMu.Lock()
	s.depths[depth]++
	s.depthMu.Unlock()
}

// RecordLinks records the number of links found.
func (s *Stats) RecordLinks(n int) {
	s.linksFound.Add(int64(n))
}

func (s *Stats) recordStatus(code int) {
	s.statusMu.Lock()
	s.statuses[code]++
	s.statusMu.Unlock()
}

// Freeze locks in the elapsed time for final display.
func (s *Stats) Freeze() {
	if s.frozen {
		return
	}
	s.frozen = true
	s.frozenAt = time.Since(s.startTime)
}

// Elapsed returns the elapsed time.
func (s *Stats) Elapsed() time.Duration {
	if s.frozen {
		return s.frozenAt
	}
	return time.Since(s.startTime)
}

// Done returns the total processed count.
func (s *Stats) Done() int64 {
	return s.success.Load() + s.failed.Load() + s.timeout.Load() + s.blocked.Load() + s.skipped.Load()
}

// Speed returns the current pages/sec (rolling 10-second window).
func (s *Stats) Speed() float64 {
	fetched := s.Done()
	bytesTotal := s.bytes.Load()
	now := time.Now()

	s.speedMu.Lock()
	s.speedTicks = append(s.speedTicks, speedTick{
		time:    now,
		fetched: fetched,
		bytes:   bytesTotal,
	})

	cutoff := now.Add(-10 * time.Second)
	start := 0
	for start < len(s.speedTicks) && s.speedTicks[start].time.Before(cutoff) {
		start++
	}
	if start > 0 && start < len(s.speedTicks) {
		s.speedTicks = s.speedTicks[start:]
	}

	var fetchSpeed, byteSpeed float64
	if len(s.speedTicks) >= 2 {
		first := s.speedTicks[0]
		last := s.speedTicks[len(s.speedTicks)-1]
		dt := last.time.Sub(first.time).Seconds()
		if dt > 0 {
			fetchSpeed = float64(last.fetched-first.fetched) / dt
			byteSpeed = float64(last.bytes-first.bytes) / dt
		}
	}
	s.rollingFetchSpeed = fetchSpeed
	s.rollingByteSpeed = byteSpeed
	s.speedMu.Unlock()

	if fetchSpeed > s.peakSpeed {
		s.peakSpeed = fetchSpeed
	}
	return fetchSpeed
}

// ByteSpeed returns the rolling bytes/sec.
func (s *Stats) ByteSpeed() float64 {
	s.speedMu.Lock()
	v := s.rollingByteSpeed
	s.speedMu.Unlock()
	return v
}

// AvgFetchMs returns average fetch time across ALL completed requests (success + fail + timeout).
func (s *Stats) AvgFetchMs() float64 {
	done := s.Done()
	if done == 0 {
		return 0
	}
	return float64(s.fetchMs.Load()) / float64(done)
}


func (s *Stats) statusLine() string {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	type kv struct {
		code  int
		count int
	}
	var pairs []kv
	for k, v := range s.statuses {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	var parts []string
	for i, p := range pairs {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%s", p.code, fmtInt(p.count)))
	}
	if len(parts) == 0 {
		return "---"
	}
	return strings.Join(parts, "  ")
}

func (s *Stats) depthLine() string {
	s.depthMu.Lock()
	defer s.depthMu.Unlock()

	if len(s.depths) == 0 {
		return "---"
	}

	type kv struct {
		depth int
		count int
	}
	var pairs []kv
	for k, v := range s.depths {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].depth < pairs[j].depth
	})

	var parts []string
	for i, p := range pairs {
		if i >= 8 {
			remaining := 0
			for _, r := range pairs[i:] {
				remaining += r.count
			}
			parts = append(parts, fmt.Sprintf("%d+:%s", p.depth, fmtInt(remaining)))
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%s", p.depth, fmtInt(p.count)))
	}
	return strings.Join(parts, "  ")
}

// rodPhaseLine returns a summary of what rod workers are doing.
func (s *Stats) rodPhaseLine() string {
	phases := make(map[string]int)
	total := 0
	stuckCount := 0
	now := time.Now()

	s.rodPhases.Range(func(key, value any) bool {
		phase := value.(string)
		phases[phase]++
		total++

		if startVal, ok := s.rodPhaseStart.Load(key); ok {
			if now.Sub(startVal.(time.Time)) > 15*time.Second {
				stuckCount++
			}
		}
		return true
	})

	totalWorkers := s.rodTotalWorkers
	if totalWorkers <= 0 {
		totalWorkers = total
	}
	if total == 0 {
		return fmt.Sprintf("0/%d  all idle", totalWorkers)
	}

	phaseOrder := []string{"pool", "nav", "render", "cf-check", "scroll", "extract", "rate-limit", "restart"}
	var parts []string
	for _, p := range phaseOrder {
		if n, ok := phases[p]; ok {
			parts = append(parts, fmt.Sprintf("%s:%d", p, n))
		}
	}
	if len(parts) == 0 {
		for p, n := range phases {
			parts = append(parts, fmt.Sprintf("%s:%d", p, n))
		}
	}

	result := fmt.Sprintf("%d/%d  %s", total, totalWorkers, strings.Join(parts, "  "))
	if stuckCount > 0 {
		result += fmt.Sprintf("  \u2502  \u26a0 %d stuck >15s", stuckCount)
	}
	return result
}

// rodWorkerDetails returns detail lines for stuck workers (>15s in same phase).
func (s *Stats) rodWorkerDetails() string {
	type workerInfo struct {
		id       int
		phase    string
		phaseDur time.Duration
		url      string
		itemDur  time.Duration
	}

	var stuck []workerInfo
	now := time.Now()

	s.rodPhases.Range(func(key, value any) bool {
		wid := key.(int)
		phase := value.(string)

		var phaseDur time.Duration
		if startVal, ok := s.rodPhaseStart.Load(key); ok {
			phaseDur = now.Sub(startVal.(time.Time))
		}
		if phaseDur <= 15*time.Second {
			return true
		}

		var u string
		if urlVal, ok := s.rodWorkerURL.Load(wid); ok {
			u = urlVal.(string)
		}
		var itemDur time.Duration
		if startVal, ok := s.rodWorkerStart.Load(wid); ok {
			itemDur = now.Sub(startVal.(time.Time))
		}

		stuck = append(stuck, workerInfo{id: wid, phase: phase, phaseDur: phaseDur, url: u, itemDur: itemDur})
		return true
	})

	if len(stuck) == 0 {
		return ""
	}

	sort.Slice(stuck, func(i, j int) bool {
		return stuck[i].phaseDur > stuck[j].phaseDur
	})

	var b strings.Builder
	show := min(5, len(stuck))
	for i := range show {
		w := stuck[i]
		u := w.url
		if len(u) > 60 {
			u = u[:57] + "..."
		}
		b.WriteString(fmt.Sprintf("    W%02d  %-10s %3ds  %s\n", w.id, w.phase, int(w.phaseDur.Seconds()), u))
	}
	return b.String()
}

// --- Formatting helpers ---

func fmtInt(n int) string {
	if n < 0 {
		return fmt.Sprintf("-%s", fmtInt(-n))
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1_000_000, (n/1000)%1000, n%1000)
}

func fmtInt64(n int64) string {
	return fmtInt(int(n))
}

func fmtBytes(b int64) string {
	if b < 0 {
		return "0 B"
	}
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	case b < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024))
	}
}

func formatDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}

func safePct(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
