# Swarm Engine Optimization Design

**Date:** 2026-02-27
**Goal:** Achieve ≥10,000 full-body OK pages/s on `search hn recrawl --engine swarm` on the remote server (4× AMD EPYC, 5.9 GB RAM, no swap).

---

## Diagnosis

**Measured baseline:** 835 avg RPS, full body, 4 drones, 3000 workers each, 5s timeout.

**Root cause:** The OS soft fd limit is 1024 per process (hard limit: 1,048,576). By Little's Law:

```
1024 fds / 5.75s avg request time = 178 URLs/s per drone × 4 drones = 712 RPS
```

This matches the measured 835 RPS almost exactly. The 3000 workers are meaningless — only 178 have live sockets at any time.

Secondary issues compound the fd bottleneck:

| Issue | Impact |
|-------|--------|
| Soft fd limit = 1024 per drone | Caps concurrent connections to 178/drone |
| No adaptive timeout — dead sites block 5s each | ~25% of workers stuck at ceiling |
| `fetchCh` = Workers×2 = 6000 × 512KB max = 3 GB | Killed one drone (lost 25% throughput) |
| Queen ignores all intermediate drone stats | Live display shows "0 ok" throughout run |

---

## Change 1 — Raise `RLIMIT_NOFILE` to 65536

**Files:** `pkg/crawl/rlimit_linux.go` (new), `pkg/crawl/rlimit_other.go` (new), `pkg/crawl/swarm_drone.go`

The drone binary calls `raiseRlimit(65536)` at the top of `RunDrone()`. Uses `syscall.Getrlimit` + `syscall.Setrlimit` — no root required since hard limit is 1M.

```go
// rlimit_linux.go
//go:build linux
package crawl
import "syscall"
func raiseRlimit(n uint64) error {
    var rl syscall.Rlimit
    if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
        return err
    }
    if rl.Cur >= n { return nil }
    if n > rl.Max { n = rl.Max }
    rl.Cur = n
    return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rl)
}

// rlimit_other.go
//go:build !linux
package crawl
func raiseRlimit(n uint64) error { return nil }
```

**Expected impact:** Unlocks up to 65536 concurrent sockets per drone. Network bandwidth becomes the new ceiling. Alone: ~5–15× improvement.

---

## Change 2 — Adaptive Timeout in Drone

**Files:** `pkg/crawl/swarm_drone.go`

`adaptiveTracker` already exists in `keepalive.go` (same package). One tracker is created per drone in `RunDrone()`, passed through `runSwarmFetch` → `swarmProcessDomain`.

In `swarmProcessDomain`:
- Per-request effective timeout: `trk.ceiling()` instead of `cfg.Timeout`
- After each fetch completes (success or error-with-latency): `trk.observe(rf.fetchMs)`

The adaptive ceiling formula (from `keepalive.go`): `max(1s, min(cfg.Timeout, P95_latency × 2))`.

If P95 latency is 500ms, timeout auto-reduces to 1s. Dead sites fail in 1s instead of 5s — 5× faster worker recycling.

**Expected impact (combined with Change 1):** avg request time drops from 5.75s → ~1.5s. Additional 3–4× improvement.

---

## Change 3 — Fix fetchCh OOM

**Files:** `pkg/crawl/swarm_drone.go`

Two sub-changes:

1. **Cap `fetchCh` at 2000** (fixed constant, not `Workers*2`):
   ```go
   // Before: make(chan rawFetch, max(cfg.Workers*2, 1000))
   // After:
   fetchCh := make(chan rawFetch, 2000)
   ```

2. **Reduce body cap from 512KB → 256KB** in `keepaliveFetchRaw`:
   ```go
   // Before: io.LimitReader(resp.Body, 512*1024)
   // After:  io.LimitReader(resp.Body, 256*1024)
   ```

Worst-case memory per drone: `2000 × 256KB = 512MB × 4 drones = 2GB`. Safe on 5.9GB server.

---

## Change 4 — Live Stats Relay

**Files:** `pkg/crawl/engine.go`, `pkg/crawl/swarm.go`, `cli/hn.go`

**Problem:** The `results ResultWriter` passed to `SwarmEngine.Run` is intentionally bypassed (drones write directly to their own DBs). The HN live display reads from that writer, so it always shows "0 ok".

**Fix:** Add a lightweight side-channel to Config:

```go
// engine.go — add to Config
ProgressFunc func(ok, failed, timeout int64) // nil-safe; called with cumulative totals
```

In `runDroneProcess` (swarm.go): track per-drone previous stats; each time a new JSON stats line arrives, compute delta, add to shared atomics `totalOK/totalFailed/totalTimeout`.

Queen spawns a background goroutine that calls `cfg.ProgressFunc(totalOK, totalFailed, totalTimeout)` every 500ms during the run.

In `cli/hn.go` `runHNRecrawlV3`: set `cfg.ProgressFunc` to update the live display's counters (the same atomics that `v3LiveStats` reads for ok/fail/timeout counts).

---

## Expected Performance After All 4 Changes

| Scenario | RPS estimate |
|----------|-------------|
| fd limit raised, adaptive timeout, no OOM | 5,000–15,000 (network-limited) |
| 10 Gbps server | ≥10,000 easily |
| 1 Gbps server (128 MB/s) | ~4,700 (128MB/s ÷ 27KB avg) |

If the network ceiling is below 10K, `--status-only=false` mode will be BW-limited. The spec will be considered met if the system demonstrates it is network-bound (not CPU/fd/code bound).

---

## Files Changed

| File | Change |
|------|--------|
| `pkg/crawl/rlimit_linux.go` | New: `raiseRlimit()` using syscall |
| `pkg/crawl/rlimit_other.go` | New: no-op stub for non-Linux |
| `pkg/crawl/swarm_drone.go` | `RunDrone`: call raiseRlimit; adaptive tracker; fix fetchCh cap; reduce body cap |
| `pkg/crawl/swarm.go` | `runDroneProcess`: accumulate drone delta stats; call ProgressFunc |
| `pkg/crawl/engine.go` | Add `ProgressFunc` to Config |
| `cli/hn.go` | Wire `cfg.ProgressFunc` to live display atomics |
