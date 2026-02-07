package cc

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FetchStats tracks live statistics for the WARC fetch pipeline.
type FetchStats struct {
	TotalPointers int

	fetched        atomic.Int64
	success        atomic.Int64
	failed         atomic.Int64
	skipped        atomic.Int64
	bytesFetched   atomic.Int64 // Compressed bytes from CDN
	bytesExtracted atomic.Int64 // Decompressed bytes
	fetchMs        atomic.Int64

	startTime time.Time
	peakSpeed float64
	Label     string

	speedMu    sync.Mutex
	speedTicks []speedTick

	frozen   bool
	frozenAt time.Duration
}

type speedTick struct {
	time  time.Time
	count int64
}

// NewFetchStats creates a new stats tracker.
func NewFetchStats(totalPointers int, label string) *FetchStats {
	return &FetchStats{
		TotalPointers: totalPointers,
		startTime:     time.Now(),
		Label:         label,
	}
}

// RecordSuccess records a successful WARC record fetch.
func (s *FetchStats) RecordSuccess(fetchedBytes, extractedBytes int64, fetchMs int64) {
	s.fetched.Add(1)
	s.success.Add(1)
	s.bytesFetched.Add(fetchedBytes)
	s.bytesExtracted.Add(extractedBytes)
	s.fetchMs.Add(fetchMs)
}

// RecordFailure records a failed fetch.
func (s *FetchStats) RecordFailure() {
	s.fetched.Add(1)
	s.failed.Add(1)
}

// RecordSkip records a skipped URL (resume mode).
func (s *FetchStats) RecordSkip() {
	s.skipped.Add(1)
}

// Done returns the total processed count.
func (s *FetchStats) Done() int64 {
	return s.fetched.Load() + s.skipped.Load()
}

// Fetched returns only URLs that required network I/O.
func (s *FetchStats) Fetched() int64 {
	return s.fetched.Load()
}

// Speed returns the rolling pages/sec.
func (s *FetchStats) Speed() float64 {
	done := s.Fetched()
	now := time.Now()

	s.speedMu.Lock()
	s.speedTicks = append(s.speedTicks, speedTick{time: now, count: done})

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

// Freeze locks in the elapsed time for final display.
func (s *FetchStats) Freeze() {
	if s.frozen {
		return
	}
	s.frozen = true
	s.frozenAt = time.Since(s.startTime)
}

// Elapsed returns the elapsed time.
func (s *FetchStats) Elapsed() time.Duration {
	if s.frozen {
		return s.frozenAt
	}
	return time.Since(s.startTime)
}

// Render returns a formatted stats display string.
func (s *FetchStats) Render() string {
	done := s.Done()
	total := int64(s.TotalPointers)
	succ := s.success.Load()
	fail := s.failed.Load()
	skip := s.skipped.Load()
	speed := s.Speed()
	elapsed := s.Elapsed()
	bytesFetched := s.bytesFetched.Load()
	bytesExtracted := s.bytesExtracted.Load()

	pct := float64(0)
	if total > 0 {
		pct = float64(done) / float64(total) * 100
	}

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

	barWidth := 40
	filled := int(pct / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  CC Fetch: %s  │  %s WARC records\n",
		s.Label, fmtInt(s.TotalPointers)))
	b.WriteString(fmt.Sprintf("  %s  %5.1f%%  %s/%s\n",
		bar, pct, fmtInt64(done), fmtInt(s.TotalPointers)))
	b.WriteString("\n")

	totalSpeed := float64(0)
	if elapsed.Seconds() > 0 {
		totalSpeed = float64(s.Fetched()) / elapsed.Seconds()
	}
	bytesPerSec := float64(0)
	if elapsed.Seconds() > 0 {
		bytesPerSec = float64(bytesFetched) / elapsed.Seconds()
	}
	b.WriteString(fmt.Sprintf("  Fetch     %s/s  │  Peak %s/s  │  Avg %s/s  │  %s/s\n",
		fmtInt64(int64(speed)), fmtInt64(int64(s.peakSpeed)), fmtInt64(int64(totalSpeed)), fmtBytes(int64(bytesPerSec))))
	b.WriteString(fmt.Sprintf("  Elapsed   %s  │  ETA  %s  │  Fetched %s  │  Extracted %s\n",
		formatDuration(elapsed), eta, fmtBytes(bytesFetched), fmtBytes(bytesExtracted)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  OK %s (%4.1f%%)  │  Failed %s (%4.1f%%)  │  Skipped %s\n",
		fmtInt64(succ), safePct(succ, succ+fail), fmtInt64(fail), safePct(fail, succ+fail), fmtInt64(skip)))

	return b.String()
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
