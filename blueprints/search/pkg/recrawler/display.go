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
	domainSkip  atomic.Int64 // URLs skipped due to truly dead host/domain (not timeout-kill)
	timeoutSkip atomic.Int64 // URLs skipped due to timeout-based host kill optimization
	bytes       atomic.Int64
	fetchMs     atomic.Int64 // sum of fetch times for avg calculation

	// DNS pipeline counters (domain-level, not URL-level)
	dnsLive    atomic.Int64
	dnsDead    atomic.Int64
	dnsTimeout atomic.Int64

	// Two-pass probe counters (domain-level)
	probeReachable   atomic.Int64
	probeUnreachable atomic.Int64

	// HTTP status code distribution (lock-free per-key increments)
	// key: int status code, value: *atomic.Int64
	statuses sync.Map

	// Domain tracking (deduped exact counts without a global mutex)
	// key: domain string, value: true
	domainsOK        sync.Map
	domainsFail      sync.Map
	domainsOKCount   atomic.Int64
	domainsFailCount atomic.Int64

	// Config
	TotalURLs     int
	UniqueDomains int
	startTime     time.Time
	peakSpeed     float64
	Label         string

	// Speed tracking (rolling window)
	speedMu    sync.Mutex
	speedTicks []speedTick

	// Rolling speed values (updated by Speed())
	rollingFetchSpeed float64
	rollingDoneSpeed  float64
	rollingByteSpeed  float64

	// Adaptive timeout info (set by recrawler during fetch)
	adaptiveInfo atomic.Value // string

	// Frozen state for final display
	frozen   bool
	frozenAt time.Duration
}

type speedTick struct {
	time    time.Time
	fetched int64 // Fetched() — network I/O only
	done    int64 // Done() — includes skips
	bytes   int64 // total bytes received
}

// NewStats creates a new stats tracker.
func NewStats(totalURLs, uniqueDomains int, label string) *Stats {
	return &Stats{
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
	if code == 0 {
		return
	}
	v, _ := s.statuses.LoadOrStore(code, &atomic.Int64{})
	v.(*atomic.Int64).Add(1)
}

// recordDomainOK tracks successful domain with deduplication.
func (s *Stats) recordDomainOK(domain string) {
	if domain == "" {
		return
	}
	if _, loaded := s.domainsOK.LoadOrStore(domain, true); !loaded {
		s.domainsOKCount.Add(1)
	}
}

// recordDomainFail tracks failed domain.
func (s *Stats) recordDomainFail(domain string) {
	if domain == "" {
		return
	}
	if _, loaded := s.domainsFail.LoadOrStore(domain, true); !loaded {
		s.domainsFailCount.Add(1)
	}
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

// RecordDomainSkipBatch records N URLs skipped for a dead domain (batch atomic).
func (s *Stats) RecordDomainSkipBatch(n int) {
	s.domainSkip.Add(int64(n))
}

// RecordTimeoutKillSkip records a URL skipped due to timeout-based host kill.
func (s *Stats) RecordTimeoutKillSkip() {
	s.timeoutSkip.Add(1)
}

// RecordTimeoutKillSkipBatch records N URLs skipped due to timeout-based host kill.
func (s *Stats) RecordTimeoutKillSkipBatch(n int) {
	s.timeoutSkip.Add(int64(n))
}

// RecordDomainSkipReason routes skip accounting based on reason.
func (s *Stats) RecordDomainSkipReason(reason string) {
	if strings.Contains(reason, "timeout") {
		s.RecordTimeoutKillSkip()
		return
	}
	s.RecordDomainSkip()
}

// RecordDomainSkipBatchReason routes batch skip accounting based on reason.
func (s *Stats) RecordDomainSkipBatchReason(reason string, n int) {
	if strings.Contains(reason, "timeout") {
		s.RecordTimeoutKillSkipBatch(n)
		return
	}
	s.RecordDomainSkipBatch(n)
}

// RecordDNSLive records a domain resolved with live IPs.
func (s *Stats) RecordDNSLive() {
	s.dnsLive.Add(1)
}

// RecordDNSDead records a domain resolved as dead (NXDOMAIN).
func (s *Stats) RecordDNSDead() {
	s.dnsDead.Add(1)
}

// RecordDNSTimeout records a domain that timed out during DNS resolution.
func (s *Stats) RecordDNSTimeout() {
	s.dnsTimeout.Add(1)
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
	return s.success.Load() + s.failed.Load() + s.timeout.Load() + s.skipped.Load() + s.domainSkip.Load() + s.timeoutSkip.Load()
}

// Fetched returns only URLs that required actual network I/O.
func (s *Stats) Fetched() int64 {
	return s.success.Load() + s.failed.Load() + s.timeout.Load()
}

// Speed returns the current fetched pages/sec (rolling 10-second window).
// Also updates rollingDoneSpeed and rollingByteSpeed as side effects.
// Only counts pages that required actual network I/O (excludes skips).
func (s *Stats) Speed() float64 {
	fetched := s.Fetched()
	done := s.Done()
	bytesTotal := s.bytes.Load()
	now := time.Now()

	s.speedMu.Lock()
	s.speedTicks = append(s.speedTicks, speedTick{
		time:    now,
		fetched: fetched,
		done:    done,
		bytes:   bytesTotal,
	})

	// Keep only last 10 seconds for stable speed reading
	cutoff := now.Add(-10 * time.Second)
	start := 0
	for start < len(s.speedTicks) && s.speedTicks[start].time.Before(cutoff) {
		start++
	}
	if start > 0 && start < len(s.speedTicks) {
		s.speedTicks = s.speedTicks[start:]
	}

	var fetchSpeed, doneSpeed, byteSpeed float64
	if len(s.speedTicks) >= 2 {
		first := s.speedTicks[0]
		last := s.speedTicks[len(s.speedTicks)-1]
		dt := last.time.Sub(first.time).Seconds()
		if dt > 0 {
			fetchSpeed = float64(last.fetched-first.fetched) / dt
			doneSpeed = float64(last.done-first.done) / dt
			byteSpeed = float64(last.bytes-first.bytes) / dt
		}
	}
	s.rollingFetchSpeed = fetchSpeed
	s.rollingDoneSpeed = doneSpeed
	s.rollingByteSpeed = byteSpeed
	s.speedMu.Unlock()

	if fetchSpeed > s.peakSpeed {
		s.peakSpeed = fetchSpeed
	}
	return fetchSpeed
}

// DoneSpeed returns the rolling done pages/sec (includes skips).
func (s *Stats) DoneSpeed() float64 {
	s.speedMu.Lock()
	v := s.rollingDoneSpeed
	s.speedMu.Unlock()
	return v
}

// ByteSpeed returns the rolling bytes/sec.
func (s *Stats) ByteSpeed() float64 {
	s.speedMu.Lock()
	v := s.rollingByteSpeed
	s.speedMu.Unlock()
	return v
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
	tskip := s.timeoutSkip.Load()
	speed := s.Speed()
	elapsed := s.Elapsed()
	bytesTotal := s.bytes.Load()

	pct := float64(0)
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}

	// ETA based on rolling done speed (responsive), fallback to overall average
	eta := "---"
	if elapsed.Seconds() > 2 && done > 0 {
		doneSpeed := s.DoneSpeed()
		if doneSpeed <= 0 {
			doneSpeed = float64(done) / elapsed.Seconds()
		}
		remaining := total - done
		if remaining > 0 && doneSpeed > 0 {
			etaDur := time.Duration(float64(remaining)/doneSpeed) * time.Second
			eta = formatDuration(etaDur)
		} else {
			eta = "0s"
		}
	}

	// Progress bar
	barWidth := 40
	filled := min(int(pct/100*float64(barWidth)), barWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	// Domain stats (exact unique counts)
	domainsReached := int(s.domainsOKCount.Load())
	domainsFailed := int(s.domainsFailCount.Load())

	// HTTP status distribution
	statusLine := s.statusLine()

	// Build output
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  Recrawl: %s  │  %s URLs  │  %s domains\n",
		s.Label, fmtInt(s.TotalURLs), fmtInt(s.UniqueDomains)))
	b.WriteString(fmt.Sprintf("  %s  %5.1f%%  %s/%s\n",
		bar, pct, fmtInt64(done), fmtInt(s.TotalURLs)))
	b.WriteString("\n")
	rollingBW := s.ByteSpeed()
	avgBW := float64(0)
	if elapsed.Seconds() > 0 {
		avgBW = float64(bytesTotal) / elapsed.Seconds()
	}
	b.WriteString(fmt.Sprintf("  Speed   %s/s  │  Peak %s/s  │  %s/s  │  Total %s\n",
		fmtInt64(int64(speed)), fmtInt64(int64(s.peakSpeed)), fmtBytes(int64(rollingBW)), fmtBytes(bytesTotal)))
	b.WriteString(fmt.Sprintf("  ETA     %s  │  Elapsed %s  │  Avg %dms/req  │  Avg %s/s\n",
		eta, formatDuration(elapsed), int(s.AvgFetchMs()), fmtBytes(int64(avgBW))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  ✓ %s ok (%4.1f%%)  ✗ %s fail (%4.1f%%)  ⏱ %s timeout (%4.1f%%)\n",
		fmtInt64(succ), safePct(succ, done), fmtInt64(fail), safePct(fail, done),
		fmtInt64(tout), safePct(tout, done)))
	b.WriteString(fmt.Sprintf("  ⊘ %s skip  ☠ %s dead-skip (%4.1f%%)  ⌛ %s timeout-kill-skip (%4.1f%%)\n",
		fmtInt64(skip), fmtInt64(dskip), safePct(dskip, done), fmtInt64(tskip), safePct(tskip, done)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  HTTP  %s\n", statusLine))
	dnsLiveCount := s.dnsLive.Load()
	dnsDeadCount := s.dnsDead.Load()
	dnsTimeoutCount := s.dnsTimeout.Load()
	dnsTotal := dnsLiveCount + dnsDeadCount + dnsTimeoutCount
	if dnsTotal > 0 {
		b.WriteString(fmt.Sprintf("  DNS     %s/%s  │  %s live  │  %s dead  │  %s timeout (%4.1f%%)\n",
			fmtInt64(dnsTotal), fmtInt(s.UniqueDomains),
			fmtInt64(dnsLiveCount), fmtInt64(dnsDeadCount), fmtInt64(dnsTimeoutCount),
			safePct(dnsDeadCount+dnsTimeoutCount, dnsTotal)))
	}
	probeOK := s.probeReachable.Load()
	probeFail := s.probeUnreachable.Load()
	probeTotal := probeOK + probeFail
	if probeTotal > 0 {
		probeDenom := dnsLiveCount
		if probeDenom == 0 {
			probeDenom = probeTotal
		}
		b.WriteString(fmt.Sprintf("  Probe   %s/%s  │  %s reachable  │  %s unreachable (%4.1f%%)\n",
			fmtInt64(probeTotal), fmtInt64(probeDenom),
			fmtInt64(probeOK), fmtInt64(probeFail),
			safePct(probeFail, probeTotal)))
	}
	b.WriteString(fmt.Sprintf("  Domains  %s reached  │  %s had-failures  │  %s avg/page\n",
		fmtInt(domainsReached), fmtInt(domainsFailed), fmtBytes(avgBytes(bytesTotal, succ))))
	if info, ok := s.adaptiveInfo.Load().(string); ok && info != "" {
		b.WriteString(fmt.Sprintf("  Timeout  %s\n", info))
	}

	return b.String()
}

func (s *Stats) statusLine() string {
	type kv struct {
		code  int
		count int64
	}
	var pairs []kv
	s.statuses.Range(func(key, value any) bool {
		code, ok1 := key.(int)
		cnt, ok2 := value.(*atomic.Int64)
		if ok1 && ok2 {
			if n := cnt.Load(); n > 0 {
				pairs = append(pairs, kv{code, n})
			}
		}
		return true
	})
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	var parts []string
	for i, p := range pairs {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("%d:%s", p.code, fmtInt64(p.count)))
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
