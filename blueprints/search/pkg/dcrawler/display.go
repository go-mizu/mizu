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
	reseeds    atomic.Int64
	continuous bool

	// Freeze
	frozen   bool
	frozenAt time.Duration
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

// RecordSuccess records a successful fetch.
func (s *Stats) RecordSuccess(statusCode int, bytesRecv int64, fetchMs int64) {
	s.success.Add(1)
	s.bytes.Add(bytesRecv)
	s.fetchMs.Add(fetchMs)
	s.recordStatus(statusCode)
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
	return s.success.Load() + s.failed.Load() + s.timeout.Load()
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

// AvgFetchMs returns average fetch time in milliseconds.
func (s *Stats) AvgFetchMs() float64 {
	succ := s.success.Load()
	if succ == 0 {
		return 0
	}
	return float64(s.fetchMs.Load()) / float64(succ)
}

// Render returns a formatted stats display string.
func (s *Stats) Render() string {
	done := s.Done()
	succ := s.success.Load()
	fail := s.failed.Load()
	tout := s.timeout.Load()
	speed := s.Speed()
	elapsed := s.Elapsed()
	bytesTotal := s.bytes.Load()
	bw := s.ByteSpeed()
	inflight := s.inFlight.Load()

	// Progress bar or open-ended counter
	var progressLine string
	if s.MaxPages > 0 {
		pct := float64(done) / float64(s.MaxPages) * 100
		if pct > 100 {
			pct = 100
		}
		barWidth := 40
		filled := int(pct / 100 * float64(barWidth))
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barWidth-filled)
		progressLine = fmt.Sprintf("  %s  %5.1f%%  %s/%s", bar, pct, fmtInt64(done), fmtInt(s.MaxPages))
	} else {
		barWidth := 40
		// Pulsing bar for open-ended crawl
		pos := int(elapsed.Seconds()*2) % (barWidth * 2)
		if pos >= barWidth {
			pos = barWidth*2 - pos - 1
		}
		var bar strings.Builder
		for i := range barWidth {
			if i >= pos-1 && i <= pos+1 {
				bar.WriteString("\u2588")
			} else {
				bar.WriteString("\u2591")
			}
		}
		mode := ""
		if s.continuous {
			mode = " [continuous]"
		}
		progressLine = fmt.Sprintf("  %s  %s pages%s", bar.String(), fmtInt64(done), mode)
	}

	// ETA
	eta := "---"
	if s.continuous {
		eta = "continuous"
	} else if s.MaxPages > 0 && elapsed.Seconds() > 2 && done > 0 && speed > 0 {
		remaining := int64(s.MaxPages) - done
		if remaining > 0 {
			etaDur := time.Duration(float64(remaining)/speed) * time.Second
			eta = formatDuration(etaDur)
		} else {
			eta = "done"
		}
	}

	// Frontier stats
	frontierQ := "---"
	bloomN := "---"
	if s.frontierLen != nil {
		frontierQ = fmtInt(s.frontierLen())
	}
	if s.bloomCount != nil {
		bloomN = fmtInt(int(s.bloomCount()))
	}

	statusLine := s.statusLine()
	depthLine := s.depthLine()

	avgPage := int64(0)
	if succ > 0 {
		avgPage = bytesTotal / succ
	}

	var b strings.Builder

	// === Header: domain + key metrics ===
	b.WriteString(fmt.Sprintf("  Crawl: %s\n", s.Label))
	b.WriteString(progressLine + "\n")
	b.WriteString("\n")

	// === SPEED (the important line) ===
	b.WriteString(fmt.Sprintf("  Speed     %s pages/s  \u2502  Peak %s/s  \u2502  Bandwidth %s/s\n",
		fmtInt64(int64(speed)), fmtInt64(int64(s.peakSpeed)), fmtBytes(int64(bw))))

	// === TOTALS ===
	b.WriteString(fmt.Sprintf("  Pages     %s downloaded  \u2502  %s total size  \u2502  avg %s/page\n",
		fmtInt64(succ), fmtBytes(bytesTotal), fmtBytes(avgPage)))

	// === TIMING ===
	b.WriteString(fmt.Sprintf("  Elapsed   %s  \u2502  ETA %s  \u2502  Avg %dms/req  \u2502  In-flight %s\n",
		formatDuration(elapsed), eta, int(s.AvgFetchMs()), fmtInt64(inflight)))
	b.WriteString("\n")

	// === Results breakdown ===
	b.WriteString(fmt.Sprintf("  \u2713 %s ok (%4.1f%%)  \u2717 %s fail (%4.1f%%)  \u23f1 %s timeout (%4.1f%%)\n",
		fmtInt64(succ), safePct(succ, done),
		fmtInt64(fail), safePct(fail, done),
		fmtInt64(tout), safePct(tout, done)))

	// === Frontier ===
	frontierLine := fmt.Sprintf("  Frontier  %s queued  \u2502  %s seen  \u2502  %s links found",
		frontierQ, bloomN, fmtInt64(s.linksFound.Load()))
	if reseeds := s.reseeds.Load(); reseeds > 0 {
		frontierLine += fmt.Sprintf("  \u2502  %s reseeds", fmtInt64(reseeds))
	}
	b.WriteString(frontierLine + "\n")
	b.WriteString("\n")

	// === HTTP + Depth ===
	b.WriteString(fmt.Sprintf("  HTTP      %s\n", statusLine))
	b.WriteString(fmt.Sprintf("  Depth     %s\n", depthLine))

	return b.String()
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
