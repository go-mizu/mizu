package crawl

import "fmt"

// AutoConfigKeepAlive returns an auto-tuned Config and a human-readable reason string.
// It raises the fd limit (idempotent) before computing worker counts.
//
// Formula:
//
//	bodyKB = 256 (fullBody) or 4 (status-only)
//
//	# Step 1: compute max workers RAM can support at innerN=4 (minimum useful innerN)
//	wMemUncapped = min(availKB×0.70 / (4×bodyKB/4),
//	                   availKB×0.80 / (4×bodyKB))
//
//	# Step 2: choose innerN
//	if fdSoft/(4×2) <= wMemUncapped:   # fd-capped even at innerN=4
//	    innerN = 4                      # maximize workers
//	else:                              # mem-capped → can afford more conns/domain
//	    innerN = clamp(CPUCount×2, 4, min(16, fdSoft/(2×wMemUncapped)))
//
//	# Step 3: recompute with chosen innerN
//	wMem    = min(availKB×0.70 / (innerN×bodyKB/4),
//	              availKB×0.80 / (innerN×bodyKB))
//	wFd     = fdSoft / (innerN×2)
//	workers = max(min(wMem, wFd, 10000), 200)
func AutoConfigKeepAlive(si SysInfo, fullBody bool) (Config, string) {
	cfg := DefaultConfig()

	availKB := si.MemAvailableMB * 1024
	if availKB <= 0 {
		availKB = 2048 * 1024 // fallback: 2 GB when platform doesn't report memory
	}

	var bodyKB int64 = 256
	if !fullBody {
		bodyKB = 4
	}

	fdSoft := int64(si.FdSoftAfter)
	if fdSoft <= 0 {
		fdSoft = 65536
	}

	const minInnerN = 4

	// Step 1: how many workers can RAM support at the minimum innerN?
	uncappedExpKB := max(int64(minInnerN)*bodyKB/4, 1)
	uncappedWrstKB := max(int64(minInnerN)*bodyKB, 1)
	wMemUncapped := min(availKB*70/100/uncappedExpKB, availKB*80/100/uncappedWrstKB)

	// Step 2: choose innerN.
	// If even at innerN=4 the fd budget is the limiting factor, use innerN=4 to
	// maximise worker count (more domains in parallel beats deeper per-domain).
	// Otherwise use a CPU-proportional innerN, capped so workers stay ≥ wMemUncapped.
	wFdMin := fdSoft / int64(minInnerN*2) // max workers achievable at innerN=4
	var innerN int
	if wFdMin <= wMemUncapped {
		// fd-capped: minimize innerN to squeeze out the most workers
		innerN = minInnerN
	} else {
		// mem-capped: CPU-proportional innerN, but don't over-allocate fds
		cpuInnerN := max(min(si.CPUCount*2, 16), minInnerN)
		// cap so wFd doesn't drop below wMemUncapped (avoid fd becoming the new cap)
		maxInnerNForFd := max(int(fdSoft/(2*max(wMemUncapped, 1))), minInnerN)
		innerN = min(cpuInnerN, maxInnerNForFd)
	}

	// Step 3: recompute memory/fd budgets with the chosen innerN
	memExpKB := max(int64(innerN)*bodyKB/4, 1)
	memWrstKB := max(int64(innerN)*bodyKB, 1)
	wMem := min(availKB*70/100/memExpKB, availKB*80/100/memWrstKB)
	wFd := fdSoft / int64(innerN*2)

	workers := max(min(wMem, wFd, 10000), 200)

	// Human-readable reason
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
