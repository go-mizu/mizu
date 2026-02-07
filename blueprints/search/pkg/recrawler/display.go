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
	success     atomic.Int64
	failed      atomic.Int64
	timeout     atomic.Int64
	skipped     atomic.Int64
	domainSkip  atomic.Int64 // URLs skipped due to dead domain
	bytes       atomic.Int64
	fetchMs     atomic.Int64 // sum of fetch times for avg calculation

	// DNS pipeline counters (domain-level, not URL-level)
	dnsLive atomic.Int64
	dnsDead atomic.Int64

	// Two-pass probe counters (domain-level)
	probeReachable   atomic.Int64
	probeUnreachable atomic.Int64

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

	// Frozen state for final display
	frozen      bool
	frozenAt    time.Duration
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
	s.recordStatus(statusCode)
	s.recordDomainOK(domain)
}

// RecordFailure records a failed fetch.
func (s *Stats) RecordFailure(statusCode int, domain string, isTimeout bool) {
	if isTimeout {
		s.timeout.Add(1)
	} else {
		s.failed.Add(1)
	}
	if statusCode > 0 {
		s.recordStatus(statusCode)
	}
	s.recordDomainFail(domain)
}

// recordStatus updates status histogram with sampling to reduce lock contention.
func (s *Stats) recordStatus(code int) {
	s.statusMu.Lock()
	s.statuses[code]++
	s.statusMu.Unlock()
}

// recordDomainOK tracks successful domain with deduplication.
func (s *Stats) recordDomainOK(domain string) {
	// Fast path: already recorded (read lock)
	s.domainMu.Lock()
	s.domainsOK[domain] = true
	s.domainMu.Unlock()
}

// recordDomainFail tracks failed domain.
func (s *Stats) recordDomainFail(domain string) {
	s.domainMu.Lock()
	s.domainsFail[domain] = true
	s.domainMu.Unlock()
}

// Freeze locks in the elapsed time for the final stats display.
// Only takes effect on the first call; subsequent calls are no-ops.
func (s *Stats) Freeze() {
	if s.frozen {
		return
	}
	s.frozen = true
	s.frozenAt = time.Since(s.startTime)
}

// Elapsed returns the elapsed time, frozen if Freeze() was called.
func (s *Stats) Elapsed() time.Duration {
	if s.frozen {
		return s.frozenAt
	}
	return time.Since(s.startTime)
}

// RecordSkip records a skipped URL (resume).
func (s *Stats) RecordSkip() {
	s.skipped.Add(1)
}

// RecordDomainSkip records a URL skipped because its domain is dead.
func (s *Stats) RecordDomainSkip() {
	s.domainSkip.Add(1)
}

// RecordDNSLive records a domain resolved with live IPs.
func (s *Stats) RecordDNSLive() {
	s.dnsLive.Add(1)
}

// RecordDNSDead records a domain resolved as dead (NXDOMAIN).
func (s *Stats) RecordDNSDead() {
	s.dnsDead.Add(1)
}

// RecordProbeReachable records a domain that responded to the HTTP probe.
func (s *Stats) RecordProbeReachable() {
	s.probeReachable.Add(1)
}

// RecordProbeUnreachable records a domain that failed the HTTP probe.
func (s *Stats) RecordProbeUnreachable() {
	s.probeUnreachable.Add(1)
}

// Done returns the total number of processed URLs (including skips, for progress bar).
func (s *Stats) Done() int64 {
	return s.success.Load() + s.failed.Load() + s.timeout.Load() + s.skipped.Load() + s.domainSkip.Load()
}

// Fetched returns only URLs that required actual network I/O.
func (s *Stats) Fetched() int64 {
	return s.success.Load() + s.failed.Load() + s.timeout.Load()
}

// Speed returns the current fetched pages/sec (rolling 5-second window).
// Only counts pages that required actual network I/O (excludes skips).
func (s *Stats) Speed() float64 {
	done := s.Fetched()
	now := time.Now()

	s.speedMu.Lock()
	s.speedTicks = append(s.speedTicks, speedTick{time: now, count: done})

	// Keep only last 10 seconds for stable speed reading
	cutoff := now.Add(-10 * time.Second)
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

// AvgSpeed returns the overall average fetched pages/sec.
func (s *Stats) AvgSpeed() float64 {
	elapsed := s.Elapsed().Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(s.Fetched()) / elapsed
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
	dskip := s.domainSkip.Load()
	speed := s.Speed()
	elapsed := s.Elapsed()
	bytesTotal := s.bytes.Load()

	pct := float64(0)
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}

	// ETA based on overall average speed (stable, not jittery)
	eta := "---"
	if elapsed.Seconds() > 2 && done > 0 {
		avgSpeed := float64(done) / elapsed.Seconds()
		remaining := total - done
		if remaining > 0 {
			etaDur := time.Duration(float64(remaining)/avgSpeed) * time.Second
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
	totalSpeed := float64(0)
	if elapsed.Seconds() > 0 {
		totalSpeed = float64(done) / elapsed.Seconds()
	}
	bytesPerSec := float64(0)
	if elapsed.Seconds() > 0 {
		bytesPerSec = float64(bytesTotal) / elapsed.Seconds()
	}
	b.WriteString(fmt.Sprintf("  Fetch     %s/s  │  Peak %s/s  │  Avg %s/s  │  %s/s\n",
		fmtInt64(int64(speed)), fmtInt64(int64(s.peakSpeed)), fmtInt64(int64(totalSpeed)), fmtBytes(int64(bytesPerSec))))
	b.WriteString(fmt.Sprintf("  Elapsed   %s  │  ETA  %s  │  Avg fetch %dms  │  Total %s\n",
		formatDuration(elapsed), eta, int(s.AvgFetchMs()), fmtBytes(bytesTotal)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  ✓ %s ok (%4.1f%%)  ✗ %s fail (%4.1f%%)  ⏱ %s timeout (%4.1f%%)\n",
		fmtInt64(succ), safePct(succ, done), fmtInt64(fail), safePct(fail, done),
		fmtInt64(tout), safePct(tout, done)))
	b.WriteString(fmt.Sprintf("  ⊘ %s skip  ☠ %s domain-dead (%4.1f%%)\n",
		fmtInt64(skip), fmtInt64(dskip), safePct(dskip, done)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  HTTP  %s\n", statusLine))
	dnsLiveCount := s.dnsLive.Load()
	dnsDeadCount := s.dnsDead.Load()
	dnsTotal := dnsLiveCount + dnsDeadCount
	if dnsTotal > 0 {
		b.WriteString(fmt.Sprintf("  DNS     %s/%s  │  %s live  │  %s dead (%4.1f%%)\n",
			fmtInt64(dnsTotal), fmtInt(s.UniqueDomains),
			fmtInt64(dnsLiveCount), fmtInt64(dnsDeadCount),
			safePct(dnsDeadCount, dnsTotal)))
	}
	probeOK := s.probeReachable.Load()
	probeFail := s.probeUnreachable.Load()
	probeTotal := probeOK + probeFail
	if probeTotal > 0 {
		b.WriteString(fmt.Sprintf("  Probe   %s/%s  │  %s reachable  │  %s unreachable (%4.1f%%)\n",
			fmtInt64(probeTotal), fmtInt64(dnsLiveCount),
			fmtInt64(probeOK), fmtInt64(probeFail),
			safePct(probeFail, probeTotal)))
	}
	b.WriteString(fmt.Sprintf("  Domains  %s reached  │  %s unreachable  │  %s avg/page\n",
		fmtInt(domainsReached), fmtInt(domainsFailed), fmtBytes(avgBytes(bytesTotal, succ))))

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
