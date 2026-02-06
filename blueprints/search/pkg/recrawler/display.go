package recrawler

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Stats tracks live statistics for the recrawl.
type Stats struct {
	// Counters (atomic for lock-free reads)
	success  atomic.Int64
	failed   atomic.Int64
	timeout  atomic.Int64
	skipped  atomic.Int64
	bytes    atomic.Int64
	fetchMs  atomic.Int64 // sum of fetch times for avg calculation

	// HTTP status code distribution
	statusMu sync.Mutex
	statuses map[int]int

	// Domain tracking
	domainMu    sync.Mutex
	domainsOK   map[string]bool
	domainsFail map[string]bool

	// Config
	TotalURLs     int
	UniqueDomains int
	startTime     time.Time
	peakSpeed     float64
	Label         string

	// Speed tracking (rolling window)
	speedMu    sync.Mutex
	speedTicks []speedTick
}

type speedTick struct {
	time  time.Time
	count int64
}

// NewStats creates a new stats tracker.
func NewStats(totalURLs, uniqueDomains int, label string) *Stats {
	return &Stats{
		statuses:      make(map[int]int),
		domainsOK:     make(map[string]bool),
		domainsFail:   make(map[string]bool),
		TotalURLs:     totalURLs,
		UniqueDomains: uniqueDomains,
		startTime:     time.Now(),
		Label:         label,
	}
}

// RecordSuccess records a successful fetch.
func (s *Stats) RecordSuccess(statusCode int, domain string, bytesRecv int64, fetchMs int64) {
	s.success.Add(1)
	s.bytes.Add(bytesRecv)
	s.fetchMs.Add(fetchMs)

	s.statusMu.Lock()
	s.statuses[statusCode]++
	s.statusMu.Unlock()

	s.domainMu.Lock()
	s.domainsOK[domain] = true
	s.domainMu.Unlock()
}

// RecordFailure records a failed fetch.
func (s *Stats) RecordFailure(statusCode int, domain string, isTimeout bool) {
	if isTimeout {
		s.timeout.Add(1)
	} else {
		s.failed.Add(1)
	}

	if statusCode > 0 {
		s.statusMu.Lock()
		s.statuses[statusCode]++
		s.statusMu.Unlock()
	}

	s.domainMu.Lock()
	s.domainsFail[domain] = true
	s.domainMu.Unlock()
}

// RecordSkip records a skipped URL.
func (s *Stats) RecordSkip() {
	s.skipped.Add(1)
}

// Done returns the total number of processed URLs.
func (s *Stats) Done() int64 {
	return s.success.Load() + s.failed.Load() + s.timeout.Load() + s.skipped.Load()
}

// Speed returns the current URLs/sec (rolling 5-second window).
func (s *Stats) Speed() float64 {
	done := s.Done()
	now := time.Now()

	s.speedMu.Lock()
	s.speedTicks = append(s.speedTicks, speedTick{time: now, count: done})

	// Keep only last 5 seconds
	cutoff := now.Add(-5 * time.Second)
	start := 0
	for start < len(s.speedTicks) && s.speedTicks[start].time.Before(cutoff) {
		start++
	}
	if start > 0 && start < len(s.speedTicks) {
		s.speedTicks = s.speedTicks[start:]
	}

	var speed float64
	if len(s.speedTicks) >= 2 {
		first := s.speedTicks[0]
		last := s.speedTicks[len(s.speedTicks)-1]
		dt := last.time.Sub(first.time).Seconds()
		if dt > 0 {
			speed = float64(last.count-first.count) / dt
		}
	}
	s.speedMu.Unlock()

	if speed > s.peakSpeed {
		s.peakSpeed = speed
	}
	return speed
}

// AvgSpeed returns the overall average speed.
func (s *Stats) AvgSpeed() float64 {
	elapsed := time.Since(s.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(s.Done()) / elapsed
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
	total := int64(s.TotalURLs)
	succ := s.success.Load()
	fail := s.failed.Load()
	tout := s.timeout.Load()
	skip := s.skipped.Load()
	speed := s.Speed()
	avgSpeed := s.AvgSpeed()
	elapsed := time.Since(s.startTime)
	bytesTotal := s.bytes.Load()

	pct := float64(0)
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}

	// ETA
	eta := "---"
	if speed > 0 {
		remaining := total - done
		if remaining > 0 {
			etaDur := time.Duration(float64(remaining)/speed) * time.Second
			eta = formatDuration(etaDur)
		} else {
			eta = "0s"
		}
	}

	// Progress bar
	barWidth := 40
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Domain stats
	s.domainMu.Lock()
	domainsReached := len(s.domainsOK)
	domainsFailed := len(s.domainsFail)
	s.domainMu.Unlock()

	// HTTP status distribution
	statusLine := s.statusLine()

	// Build output
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Recrawl: %s  │  %s URLs  │  %s domains\n",
		s.Label, fmtInt(s.TotalURLs), fmtInt(s.UniqueDomains)))
	b.WriteString(fmt.Sprintf("  %s  %5.1f%%  %s/%s\n",
		bar, pct, fmtInt64(done), fmtInt(s.TotalURLs)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Speed     %s/s  │  Peak %s/s  │  Avg %s/s\n",
		fmtInt64(int64(speed)), fmtInt64(int64(s.peakSpeed)), fmtInt64(int64(avgSpeed))))
	b.WriteString(fmt.Sprintf("  Elapsed   %s  │  ETA  %s  │  Avg fetch %dms\n",
		formatDuration(elapsed), eta, int(s.AvgFetchMs())))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  ✓ %s ok (%4.1f%%)  ✗ %s fail (%4.1f%%)\n",
		fmtInt64(succ), safePct(succ, done), fmtInt64(fail), safePct(fail, done)))
	b.WriteString(fmt.Sprintf("  ⏱ %s timeout (%4.1f%%)  ⊘ %s skip (%4.1f%%)\n",
		fmtInt64(tout), safePct(tout, done), fmtInt64(skip), safePct(skip, done)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  HTTP  %s\n", statusLine))
	b.WriteString(fmt.Sprintf("  Domains  %s reached  │  %s unreachable\n",
		fmtInt(domainsReached), fmtInt(domainsFailed)))
	b.WriteString(fmt.Sprintf("  Bytes  %s  │  %s avg  │  %s/s\n",
		fmtBytes(bytesTotal), fmtBytes(avgBytes(bytesTotal, succ)),
		fmtBytes(int64(float64(bytesTotal)/elapsed.Seconds()))))

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

// --- Formatting helpers ---

func fmtInt(n int) string {
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
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func safePct(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func avgBytes(total, count int64) int64 {
	if count == 0 {
		return 0
	}
	return total / count
}
