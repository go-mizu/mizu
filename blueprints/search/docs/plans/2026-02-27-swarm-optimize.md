# Swarm Engine Optimization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Achieve ≥10,000 full-body OK pages/s on `search hn recrawl --engine swarm` by removing the fd-limit bottleneck and fixing OOM/display bugs.

**Architecture:** Four targeted changes: (1) raise drone RLIMIT_NOFILE from 1024 → 65536, (2) add adaptive timeout (existing `adaptiveTracker` from keepalive.go) into the drone fetch loop, (3) fix channel OOM by capping `fetchCh` at 2000 + halving body buffer to 256KB, (4) relay live drone stats to the queen's progress display via a `ProgressFunc` callback.

**Tech Stack:** Go, `syscall.Setrlimit` (Linux), existing `adaptiveTracker` (same package), `atomic.Int64`, `cli/hn.go` `v3LiveStats`.

---

### Task 1: Add RLIMIT_NOFILE raise (rlimit files + RunDrone call)

**Files:**
- Create: `pkg/crawl/rlimit_linux.go`
- Create: `pkg/crawl/rlimit_other.go`
- Modify: `pkg/crawl/swarm_drone.go` (top of `RunDrone`)

**Step 1: Create `pkg/crawl/rlimit_linux.go`**

```go
//go:build linux

package crawl

import (
	"fmt"
	"syscall"
)

func raiseRlimit(n uint64) error {
	var rl syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rl); err != nil {
		return fmt.Errorf("getrlimit: %w", err)
	}
	if rl.Cur >= n {
		return nil
	}
	if n > rl.Max {
		n = rl.Max
	}
	rl.Cur = n
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rl)
}
```

**Step 2: Create `pkg/crawl/rlimit_other.go`**

```go
//go:build !linux

package crawl

func raiseRlimit(_ uint64) error { return nil }
```

**Step 3: Add `raiseRlimit` call at the top of `RunDrone`**

In `pkg/crawl/swarm_drone.go`, the function `RunDrone` starts at line 173. Add the call immediately after `readDroneInput`:

```go
func RunDrone(ctx context.Context, cfg Config) error {
	frame, seeds, err := readDroneInput(os.Stdin)
	if err != nil {
		return fmt.Errorf("read drone input: %w", err)
	}
	if len(seeds) == 0 {
		return nil
	}

	// Raise fd limit to allow more concurrent connections.
	if err := raiseRlimit(65536); err != nil {
		fmt.Fprintf(os.Stderr, "[drone] raiseRlimit: %v (continuing)\n", err)
	}

	dns := &staticFrameCache{...}
	...
}
```

The exact edit is: insert the 4-line `raiseRlimit` block **after** `dns := &staticFrameCache{resolved: frame.Resolved, dead: frame.Dead}` (line 182).

**Step 4: Build to verify compilation**

Run: `go build ./pkg/crawl/...`
Expected: no errors

**Step 5: Commit**

```bash
git add pkg/crawl/rlimit_linux.go pkg/crawl/rlimit_other.go pkg/crawl/swarm_drone.go
git commit -m "feat(swarm): raise RLIMIT_NOFILE to 65536 in drone"
```

---

### Task 2: Fix fetchCh OOM (cap 2000 + body cap 256KB)

**Files:**
- Modify: `pkg/crawl/swarm_drone.go`

This is a two-line change. No new functions needed.

**Step 1: Change fetchCh buffer size**

In `swarm_drone.go` line 226, change:
```go
// Before:
fetchCh := make(chan rawFetch, max(cfg.Workers*2, 1000))

// After:
fetchCh := make(chan rawFetch, 2000)
```

**Step 2: Change body read cap**

In `keepaliveFetchRaw` (line ~121), change:
```go
// Before:
bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 512*1024))

// After:
bodyBytes, _ = io.ReadAll(io.LimitReader(resp.Body, 256*1024))
```

**Step 3: Build to verify**

Run: `go build ./pkg/crawl/...`
Expected: no errors

**Step 4: Commit**

```bash
git add pkg/crawl/swarm_drone.go
git commit -m "fix(swarm): cap fetchCh at 2000, body buffer 256KB to prevent OOM"
```

---

### Task 3: Add adaptive timeout to drone fetch loop

**Files:**
- Modify: `pkg/crawl/swarm_drone.go`

The `adaptiveTracker` type already exists in `keepalive.go` (same package). We add one instance per drone and thread it through `runSwarmFetch` → `swarmProcessDomain`.

**Step 1: Create the tracker in `RunDrone` and pass to `runSwarmFetch`**

In `RunDrone` (line ~282), the call is:
```go
runSwarmFetch(ctx, seeds, dns, cfg, fetchCh, failDB)
```

Change to:
```go
trk := &adaptiveTracker{}
runSwarmFetch(ctx, seeds, dns, cfg, trk, fetchCh, failDB)
```

**Step 2: Update `runSwarmFetch` signature**

Current signature (line 307):
```go
func runSwarmFetch(ctx context.Context, seeds []recrawler.SeedURL,
	dns DNSCache, cfg Config, fetchCh chan<- rawFetch, failDB *recrawler.FailedDB) {
```

New signature — add `trk *adaptiveTracker` before `fetchCh`:
```go
func runSwarmFetch(ctx context.Context, seeds []recrawler.SeedURL,
	dns DNSCache, cfg Config, trk *adaptiveTracker, fetchCh chan<- rawFetch, failDB *recrawler.FailedDB) {
```

Inside `runSwarmFetch`, the call to `swarmProcessDomain` (line ~354) becomes:
```go
swarmProcessDomain(gctx, urls, dns, cfg, trk, innerN, fetchCh, failDB)
```

**Step 3: Update `swarmProcessDomain` signature and apply adaptive timeout**

Current signature (line 364):
```go
func swarmProcessDomain(ctx context.Context, urls []recrawler.SeedURL,
	dns DNSCache, cfg Config, innerN int, fetchCh chan<- rawFetch,
	failDB *recrawler.FailedDB) {
```

New signature — add `trk *adaptiveTracker` after `cfg Config`:
```go
func swarmProcessDomain(ctx context.Context, urls []recrawler.SeedURL,
	dns DNSCache, cfg Config, trk *adaptiveTracker, innerN int, fetchCh chan<- rawFetch,
	failDB *recrawler.FailedDB) {
```

Inside the per-URL worker loop (where `client` is created and used), apply adaptive timeout **before** each fetch and record latency **after** each fetch.

The current inner goroutine (line ~416):
```go
client := &http.Client{
    Transport: transport,
    Timeout:   cfg.Timeout,
}
for seed := range urlCh {
    // ... context checks ...
    rf := keepaliveFetchRaw(domainCtx, client, seed, cfg)
    // ... timeout detection, fetchCh send ...
}
```

Change to apply adaptive timeout per iteration (same pattern as `processOneDomain` in keepalive.go):
```go
client := &http.Client{
    Transport: transport,
    Timeout:   cfg.Timeout,
}
for seed := range urlCh {
    // ... existing context checks (abandon/domain-deadline) ...

    // Apply adaptive timeout if we have enough samples.
    if t := trk.Timeout(cfg.Timeout); t > 0 {
        client.Timeout = t
    } else {
        client.Timeout = cfg.Timeout
    }

    rf := keepaliveFetchRaw(domainCtx, client, seed, cfg)

    // Record successful latencies for adaptive tracker.
    if rf.errStr == "" {
        trk.record(rf.fetchMs)
    }

    // ... existing isTimeout detection, abandonCh close, fetchCh send ...
}
```

**Step 4: Build to verify**

Run: `go build ./pkg/crawl/...`
Expected: no errors

**Step 5: Commit**

```bash
git add pkg/crawl/swarm_drone.go
git commit -m "feat(swarm): add adaptive P95×2 timeout in drone fetch loop"
```

---

### Task 4: Add ProgressFunc to Config + live stats relay in swarm.go

**Files:**
- Modify: `pkg/crawl/engine.go`
- Modify: `pkg/crawl/swarm.go`

**Step 1: Add `ProgressFunc` field to `Config` in `engine.go`**

In `engine.go`, add to the `Config` struct after the `BatchSize` field (line 56):

```go
// ProgressFunc is called every 500ms by the swarm engine with cumulative
// ok/failed/timeout totals from all drones. Nil-safe.
ProgressFunc func(ok, failed, timeout int64)
```

**Step 2: Change `runDroneProcess` to accumulate stats live (delta per JSON line)**

In `swarm.go`, the current `runDroneProcess` accumulates stats only at the very end:

```go
// CURRENT (broken — only final values added):
ok.Add(finalStats.OK)
failed.Add(finalStats.Failed)
timeout.Add(finalStats.Timeout)
total.Add(finalStats.Total)
```

Replace the entire stdout-drain loop and final accumulation with delta accumulation:

```go
// Replace the scanner loop + final Add calls with this:
var prevOK, prevFailed, prevTimeout int64
scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    var ds droneStats
    if json.Unmarshal(scanner.Bytes(), &ds) == nil {
        ok.Add(ds.OK - prevOK)
        failed.Add(ds.Failed - prevFailed)
        timeout.Add(ds.Timeout - prevTimeout)
        total.Add((ds.OK + ds.Failed + ds.Timeout) - (prevOK + prevFailed + prevTimeout))
        prevOK = ds.OK
        prevFailed = ds.Failed
        prevTimeout = ds.Timeout
    }
}
// Note: remove the old ok.Add/failed.Add/timeout.Add/total.Add lines after cmd.Wait()
```

Also **remove** the `peak.Record()` call from inside the scanner loop — it was firing at JSON-line rate (~2/s) rather than per-request.

**Step 3: Add ProgressFunc goroutine in `SwarmEngine.Run`**

In `swarm.go`, after the `var wg sync.WaitGroup` declaration (line ~105), add a background goroutine that calls `cfg.ProgressFunc` every 500ms while the drones run:

```go
// ProgressFunc relay: call every 500ms with cumulative totals.
var progressDone chan struct{}
if cfg.ProgressFunc != nil {
    progressDone = make(chan struct{})
    go func() {
        defer close(progressDone)
        ticker := time.NewTicker(500 * time.Millisecond)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                cfg.ProgressFunc(totalOK.Load(), totalFailed.Load(), totalTimeout.Load())
            }
        }
    }()
}

wg.Wait()  // drones finish

if progressDone != nil {
    <-progressDone  // wait for goroutine — it exits on ctx.Done() or we cancel
}
```

Wait — the goroutine exits on `ctx.Done()` but after `wg.Wait()` we need it to stop. Use a cancel context local to the goroutine:

```go
if cfg.ProgressFunc != nil {
    progressCtx, cancelProgress := context.WithCancel(ctx)
    progressDone = make(chan struct{})
    go func() {
        defer close(progressDone)
        ticker := time.NewTicker(500 * time.Millisecond)
        defer ticker.Stop()
        for {
            select {
            case <-progressCtx.Done():
                return
            case <-ticker.C:
                cfg.ProgressFunc(totalOK.Load(), totalFailed.Load(), totalTimeout.Load())
            }
        }
    }()
    defer func() {
        cancelProgress()
        <-progressDone
    }()
}
```

Place the `defer` before `wg.Wait()` so it runs after `wg.Wait()` returns.

**Step 4: Build to verify**

Run: `go build ./pkg/crawl/...`
Expected: no errors

**Step 5: Commit**

```bash
git add pkg/crawl/engine.go pkg/crawl/swarm.go
git commit -m "feat(swarm): ProgressFunc callback + live delta stats accumulation"
```

---

### Task 5: Wire ProgressFunc in cli/hn.go

**Files:**
- Modify: `cli/hn.go`

**Step 1: Add ProgressFunc wiring after `cfg.Notifier = ls`**

In `cli/hn.go`, after line 832 (`cfg.Notifier = ls`), add:

```go
// For swarm engine: relay drone stats to live display atomics.
if engineName == "swarm" {
    cfg.ProgressFunc = func(ok, failed, timeout int64) {
        ls.ok.Store(ok)
        ls.failed.Store(failed)
        ls.timeout.Store(timeout)
        ls.total.Store(ok + failed + timeout)
    }
}
```

**Step 2: Build to verify**

Run: `go build ./cmd/search/`
Expected: no errors

**Step 3: Commit**

```bash
git add cli/hn.go
git commit -m "feat(swarm): wire ProgressFunc to live display in hn recrawl"
```

---

### Task 6: Build Linux binary and deploy

**Step 1: Check Makefile targets**

Run: `cat Makefile | grep -E '(linux|deploy|build)'`

Identify the cross-compilation target (likely `make build-linux` or similar).

**Step 2: Build Linux binary**

```bash
GOOS=linux GOARCH=amd64 go build -o /tmp/search-linux ./cmd/search/
```

Or use the Makefile target if one exists.

**Step 3: Deploy to remote server**

Check Makefile for deploy target. Typical pattern:
```bash
make deploy-linux
```

Or manually:
```bash
scp /tmp/search-linux remote-server:~/bin/search
```

**Step 4: Verify binary uploaded**

```bash
ssh remote-server "~/bin/search --version || ~/bin/search --help | head -3"
```

---

### Task 7: Benchmark on remote server

**Step 1: SSH to remote and run benchmark**

```bash
ssh remote-server
cd ~
./bin/search hn recrawl --engine swarm --workers 2000 --drones 4 --timeout 5000 --domain-timeout 30000 --limit 200000
```

Observe live display — should show non-zero ok/fail/timeout now.

**Step 2: Verify body is stored**

After a run completes, check result DB has body content:
```bash
duckdb /path/to/results/d0/*.duckdb -c "SELECT url, length(body) as body_len FROM results WHERE body_len > 0 LIMIT 5;"
```

**Step 3: Confirm RPS target**

Expected outcomes:
- fd-unlimited (65536): 5,000–15,000 RPS (network-bound)
- Live display shows non-zero ok/failed/timeout during run
- No drone OOM kills (fetchCh capped at 2000, body at 256KB)

If server is bandwidth-limited at <10K RPS, confirm `avg_rps > worker_count × 1/timeout` to show network (not fd) is the new ceiling.

---

## Reference: Root Cause

**Little's Law proof that fd limit was the bottleneck:**
```
1024 fds / 5.75s avg request time = 178 URLs/s per drone × 4 drones = 712 RPS ≈ measured 835 RPS
```

After raising to 65536:
```
65536 fds / 1.5s adaptive timeout = 43,690 concurrent requests / 4 drones = could hit 10K+ easily
```

Network bandwidth ceiling at 1 Gbps: `128 MB/s ÷ 27KB avg = ~4,700 RPS` (full body mode).
