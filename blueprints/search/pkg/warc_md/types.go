package warc_md

import (
	"sync/atomic"
	"time"
)

// PhaseStats holds final statistics for a single pipeline phase.
type PhaseStats struct {
	Files      int64
	Skipped    int64
	Errors     int64
	ReadBytes  int64
	WriteBytes int64
	PeakMemMB  float64
	Duration   time.Duration
}

// PipelineResult holds stats for the two pipeline phases.
type PipelineResult struct {
	Extract  *PhaseStats
	Convert  *PhaseStats
	Duration time.Duration
}

// ProgressFunc is called periodically during a phase.
// Parameters: done, total, errors, readBytes, writeBytes, elapsed, peakMemMB
// total may be 0 when unknown (streaming Phase 1).
type ProgressFunc func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64)

// WARCItem is a single HTML record extracted from a .warc.gz for the pipeline.
type WARCItem struct {
	RecordID string
	HTMLBody []byte
}

// MarkdownItem is the output of the convert phase.
type MarkdownItem struct {
	RecordID   string
	Markdown   string
	Title      string
	Language   string
	HasContent bool
}

// trackPeakMem samples real RSS (VmRSS on Linux, MemStats.Sys elsewhere)
// every 2s and returns a getter for the peak MB seen so far.
// Close stop to terminate the background goroutine.
func trackPeakMem(stop <-chan struct{}) func() float64 {
	var peakMB int64
	mb := int64(readRSSMB())
	if mb < 1 {
		mb = 1
	}
	atomic.StoreInt64(&peakMB, mb)

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				mb := int64(readRSSMB())
				for {
					old := atomic.LoadInt64(&peakMB)
					if old >= mb {
						break
					}
					if atomic.CompareAndSwapInt64(&peakMB, old, mb) {
						break
					}
				}
			case <-stop:
				return
			}
		}
	}()
	return func() float64 { return float64(atomic.LoadInt64(&peakMB)) }
}

// ReadRSSMB is exported for use by the scheduler to read live RSS of running sessions.
func ReadRSSMB() float64 {
	return readRSSMB()
}
