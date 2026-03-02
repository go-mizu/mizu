package markdown

import (
	"runtime"
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

// PhaseProgressFunc is called periodically during a phase.
// Params: done, total, errors, readBytes, writeBytes, elapsed, peakMemMB
type PhaseProgressFunc func(done, total, errors, readBytes, writeBytes int64, elapsed time.Duration, peakMemMB float64)

// trackPeakMem samples runtime.MemStats.Sys every 2s and returns a getter for the peak MB.
// Close stop to terminate.
func trackPeakMem(stop <-chan struct{}) func() float64 {
	var peakMB int64 // MB stored as int64 for atomic CAS
	// Initial sample
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	atomic.StoreInt64(&peakMB, int64(ms.Sys>>20))

	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				runtime.ReadMemStats(&ms)
				mb := int64(ms.Sys >> 20)
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

// cidFromRelPath reconstructs a CID from a relative path with any extension.
// Works for .gz, .html, .md, .md.gz etc.
// ab/cd/ef0123...rest.ext → sha256:abcdef0123...rest
func cidFromRelPath(relPath string) string {
	// normalise separators
	s := relPath
	for i := range s {
		if s[i] == '\\' {
			b := []byte(s)
			b[i] = '/'
			s = string(b)
		}
	}
	// split into at most 3 parts: dir1/dir2/filename
	slash1 := -1
	slash2 := -1
	for i, c := range s {
		if c == '/' {
			if slash1 < 0 {
				slash1 = i
			} else if slash2 < 0 {
				slash2 = i
				break
			}
		}
	}
	if slash1 < 0 || slash2 < 0 {
		return "unknown:" + relPath
	}
	dir1 := s[:slash1]
	dir2 := s[slash1+1 : slash2]
	name := s[slash2+1:]
	// strip all extensions (.gz, .html, .md, .md.gz)
	for {
		dot := -1
		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '.' {
				dot = i
				break
			}
		}
		if dot < 0 {
			break
		}
		name = name[:dot]
	}
	return "sha256:" + dir1 + dir2 + name
}
