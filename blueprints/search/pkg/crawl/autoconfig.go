package crawl

import "fmt"

// AutoConfigKeepAlive returns an auto-tuned Config and a human-readable reason string.
// It raises the fd limit (idempotent) before computing worker counts.
//
// Formula:
//
//	innerN = clamp(CPUCount×2, 4, 16)
//	bodyKB = 256 (fullBody) or 4 (status-only)
//
//	wMem  = min(availKB×0.70 / (innerN×bodyKB/4),    ← expected use, 25% saturation
//	            availKB×0.80 / (innerN×bodyKB))        ← worst-case OOM guard
//	wFd   = fdSoftAfter / (innerN×2)                  ← fd safety factor 2×
//
//	workers = max(min(wMem, wFd, 10000), 200)
func AutoConfigKeepAlive(si SysInfo, fullBody bool) (Config, string) {
	cfg := DefaultConfig()

	// Inner connections per domain: 2 per CPU, clamped [4,16]
	innerN := max(min(si.CPUCount*2, 16), 4)

	availKB := si.MemAvailableMB * 1024
	if availKB <= 0 {
		availKB = 2048 * 1024 // fallback: 2 GB when platform doesn't report memory
	}

	var bodyKB int64 = 256
	if !fullBody {
		bodyKB = 4
	}

	// Memory per worker: expected (25% saturation) and worst-case (100%)
	memExpKB := max(int64(innerN)*bodyKB/4, 1)
	memWrstKB := max(int64(innerN)*bodyKB, 1)

	// Workers from memory constraints (soft = expected use, hard = worst-case OOM guard)
	wMem := min(availKB*70/100/memExpKB, availKB*80/100/memWrstKB)

	// Workers from fd limit
	fdSoft := int64(si.FdSoftAfter)
	if fdSoft <= 0 {
		fdSoft = 65536
	}
	wFd := fdSoft / int64(innerN*2)

	workers := max(min(wMem, wFd, 10000), 200)

	// Build human-readable reason
	var limitBy string
	if wFd <= wMem {
		limitBy = fmt.Sprintf("fd-capped (%d÷%d)", fdSoft, innerN*2)
	} else {
		limitBy = fmt.Sprintf("mem-capped (%d MB avail)", si.MemAvailableMB)
	}

	reason := fmt.Sprintf("workers=%d  innerN=%d  (%s)", workers, innerN, limitBy)

	cfg.Workers = int(workers)
	cfg.MaxConnsPerDomain = innerN
	cfg.StatusOnly = !fullBody

	return cfg, reason
}
