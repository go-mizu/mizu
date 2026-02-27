# Recrawl 3K OK/s Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Hit ≥3,000 avg OK pages/s on server2, with zero false negatives (every live URL gets body in result DB).

**Architecture:** Three layered improvements — (1) auto-config uses lower innerN to unlock 3× more workers when fd-capped, (2) shorter default timeout with two-pass retry to drain dead domains fast while catching slow-but-live, (3) `--retry` flag reads `http_timeout` URLs from failed.duckdb and re-crawls with long timeout.

**Tech Stack:** Go 1.26, `pkg/crawl/autoconfig.go`, `pkg/crawl/keepalive.go`, `cli/hn.go`, `pkg/archived/recrawler/faileddb.go`

---

## Root Cause Recap

```
server2 post-0617:  workers=2730 (fd-capped 65536÷24), timeout=5s, 60% timeouts
  throughput = 2730 / (0.4×0.3s + 0.6×5s) = 415 req/s → 148 avg OK/s

Target:             workers=5461 (fd-capped 65536÷12), timeout=2s, 60% timeouts
  throughput = 5461 / (0.4×0.3s + 0.6×2s) = 2068 req/s → 827 avg OK/s   ← 5.6× improvement

Target with better seed quality (75% OK, 400ms avg):
  throughput = 5461 / (0.75×0.4s + 0.25×2s) = 5461/0.8 = 6826 req/s → 5120 OK/s  ← above 3k
```

Three wins stack:
| Win | Change | Expected Multiplier |
|-----|--------|-------------------|
| Lower innerN (12→6) | autoconfig.go | +100% workers |
| Shorter timeout (5s→2s) | cli/hn.go flag default | +2.4× throughput |
| Two-pass retry (false-neg fix) | cli/hn.go retry loop | +quality, no false neg |
| Better seeds from pass-1 OK list | automatic (second run hits known-live) | +OK% 40%→75% |

---

## Task 1: Optimize innerN in AutoConfigKeepAlive

**Problem:** `innerN = clamp(CPUs×2, 4, 16)` gives innerN=12 on server2 (6 CPUs).
With fd-capped formula `wFd = fdSoft / (innerN×2)`: 65536/24 = 2730 workers.
If we lower innerN to 6: 65536/12 = 5461 workers — exactly 2× more, no other change needed.

**Rule:** When fd-capped (wFd < wMem), lower innerN to maximize workers. The minimum sensible innerN is 4 (enough parallelism per domain). We want `innerN = max(4, floor(fdSoft / (wMem_uncapped × 2)))` where wMem_uncapped uses innerN=4 (the minimum).

**Files:**
- Modify: `pkg/crawl/autoconfig.go`

**Step 1: Understand current formula**

```go
// Current (line 22)
innerN := max(min(si.CPUCount*2, 16), 4)  // server2 → 12
wFd    := fdSoft / int64(innerN*2)         // 65536/24 = 2730
workers := max(min(wMem, wFd, 10000), 200) // min(10266, 2730) = 2730
```

**Step 2: New formula — fd-budget-aware innerN**

Replace lines 22 (innerN) through 48 (workers computation) in `pkg/crawl/autoconfig.go`:

```go
// Step 1: compute uncapped memory workers with minimum innerN=4
// This tells us how many workers RAM could support.
const minInnerN = 4
uncappedBodyKB := max(int64(minInnerN)*bodyKB/4, 1)
uncappedWorstKB := max(int64(minInnerN)*bodyKB, 1)
wMemUncapped := min(availKB*70/100/uncappedBodyKB, availKB*80/100/uncappedWorstKB)

// Step 2: choose innerN to maximize workers given fd budget
// If fd-capped at innerN=4 already, use innerN=4.
// If mem-capped, use CPU-proportional innerN (better per-domain throughput).
fdSoft := int64(si.FdSoftAfter)
if fdSoft <= 0 {
    fdSoft = 65536
}

var innerN int
wFdMin := fdSoft / int64(minInnerN*2) // max possible workers
if wFdMin <= wMemUncapped {
    // fd-capped even at innerN=4: use innerN=4 to maximize workers
    innerN = minInnerN
} else {
    // mem-capped: can afford higher innerN for better per-domain concurrency
    // compute CPU-based innerN, capped so we don't waste fds unnecessarily
    cpuInnerN := max(min(si.CPUCount*2, 16), 4)
    // further cap so wFd stays >= wMemUncapped (don't over-shrink workers)
    // i.e. innerN <= fdSoft / (2 * wMemUncapped)
    if wMemUncapped > 0 {
        maxInnerN := int(fdSoft / (2 * wMemUncapped))
        if maxInnerN < 4 {
            maxInnerN = 4
        }
        innerN = min(cpuInnerN, maxInnerN)
    } else {
        innerN = cpuInnerN
    }
}

// Recompute memory budget with chosen innerN
memExpKB  := max(int64(innerN)*bodyKB/4, 1)
memWrstKB := max(int64(innerN)*bodyKB, 1)
wMem   := min(availKB*70/100/memExpKB, availKB*80/100/memWrstKB)
wFd    := fdSoft / int64(innerN*2)
workers := max(min(wMem, wFd, 10000), 200)

// limitBy reason
var limitBy string
if wFd <= wMem {
    limitBy = fmt.Sprintf("fd-capped (%d÷%d)", fdSoft, innerN*2)
} else {
    limitBy = fmt.Sprintf("mem-capped (%d MB avail)", si.MemAvailableMB)
}
```

**Expected result:**
- server2 (6 CPUs, 10.4GB RAM, fd=65536): wMemUncapped=10266 > wFdMin=8192 → fd-capped → innerN=4, workers=8192
- server1 (4 CPUs, 5.0GB RAM, fd=65536): wMemUncapped=2066 < wFdMin=8192 → mem-capped → cpuInnerN=8, maxInnerN=65536/(2×2066)=15 → innerN=8, workers=2066 (unchanged)

**Step 3: Build and verify**

```bash
cd blueprints/search
GOWORK=off CGO_ENABLED=1 go build ./cmd/search/ 2>&1
```
Expected: build succeeds, no errors.

**Step 4: Verify auto-config output locally (macOS stub)**

```bash
./cmd/search/... || GOWORK=off CGO_ENABLED=1 go run ./cmd/search/ hn recrawl --help 2>&1 | head -5
```

**Step 5: Commit**

```bash
git add pkg/crawl/autoconfig.go
git commit -m "perf(autoconfig): lower innerN to 4 on fd-capped servers, doubling worker count"
```

---

## Task 2: Reduce Default Timeout from 5s to 2s

**Problem:** 60% of requests timeout after waiting 5s. Cutting to 2s makes the timeout drain 2.5× faster.
False-negative risk: servers that respond in 2-5s are missed. Solved by Task 3 (retry pass).

**Files:**
- Modify: `cli/hn.go` (one line: flag default)

**Step 1: Change flag default**

In `newHNRecrawl()`, find:
```go
cmd.Flags().IntVar(&timeoutMs, "timeout", 5000, "Per-request HTTP timeout in milliseconds")
```
Change to:
```go
cmd.Flags().IntVar(&timeoutMs, "timeout", 2000, "Per-request HTTP timeout in milliseconds")
```

**Step 2: Build and verify**

```bash
GOWORK=off CGO_ENABLED=1 go build ./cmd/search/ 2>&1
```

**Step 3: Commit**

```bash
git add cli/hn.go
git commit -m "perf(hn): default timeout 5s→2s (2.4× timeout drain; retry pass catches slow-but-live)"
```

---

## Task 3: Add `LoadTimeoutURLs` to faileddb.go

The retry pass needs to read `http_timeout` URLs from failed.duckdb. Add a reader function.

**Files:**
- Modify: `pkg/archived/recrawler/faileddb.go`

**Step 1: Add LoadTimeoutURLs function**

Append to `faileddb.go` (after `FailedURLSummary`):

```go
// LoadTimeoutURLs reads all URLs with reason="http_timeout" from a FailedDB file.
// Returns them as SeedURLs suitable for a retry crawl pass.
func LoadTimeoutURLs(dbPath string) ([]SeedURL, error) {
    db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
    if err != nil {
        return nil, fmt.Errorf("opening failed db: %w", err)
    }
    defer db.Close()

    rows, err := db.Query(`
        SELECT url, domain
        FROM failed_urls
        WHERE reason = 'http_timeout'
        ORDER BY domain, url
    `)
    if err != nil {
        return nil, fmt.Errorf("querying timeout URLs: %w", err)
    }
    defer rows.Close()

    var seeds []SeedURL
    for rows.Next() {
        var s SeedURL
        if err := rows.Scan(&s.URL, &s.Domain); err != nil {
            continue
        }
        seeds = append(seeds, s)
    }
    return seeds, nil
}
```

**Step 2: Build**

```bash
GOWORK=off CGO_ENABLED=1 go build ./... 2>&1
```

**Step 3: Commit**

```bash
git add pkg/archived/recrawler/faileddb.go
git commit -m "feat(faileddb): add LoadTimeoutURLs for two-pass retry"
```

---

## Task 4: Two-Pass Retry in `runHNRecrawlV3`

After pass 1 completes, automatically re-run only the `http_timeout` URLs from failed.duckdb using a longer timeout. This eliminates false negatives (slow-but-live servers captured in pass 2).

**Files:**
- Modify: `cli/hn.go`

**Step 1: Add `--retry-timeout` and `--no-retry` flags to `newHNRecrawl()`**

After the existing flag declarations (around line 680):
```go
var (
    retryTimeoutMs int
    noRetry        bool
)
// ...existing flags...
cmd.Flags().IntVar(&retryTimeoutMs, "retry-timeout", 20000, "Pass-2 timeout for retrying http_timeout URLs (ms); 0=disabled")
cmd.Flags().BoolVar(&noRetry, "no-retry", false, "Skip pass-2 retry of timeout URLs")
```

Pass them to `runHNRecrawlV3`:
```go
return runHNRecrawlV3(ctx, cfg, seedRes,
    engine, workers, maxConnsPerDomain, timeoutMs, domainFailThreshold, domainTimeoutMs, statusOnly, batchSize, int64(slowDomainMs),
    dnsWorkers, dnsTimeoutMs,
    retryTimeoutMs, noRetry)  // add these two
```

**Step 2: Update `runHNRecrawlV3` signature**

```go
func runHNRecrawlV3(ctx context.Context,
    hnCfg hn.Config,
    seedRes *hn.RecrawlSeedResult,
    engineName string,
    workers, maxConnsPerDomain, timeoutMs, domainFailThreshold, domainTimeoutMs int,
    statusOnly bool,
    batchSize int,
    slowDomainMs int64,
    dnsWorkers, dnsTimeoutMs int,
    retryTimeoutMs int,  // NEW
    noRetry bool,        // NEW
) error {
```

**Step 3: Add pass 2 block after pass 1 completes**

Insert AFTER the `cancelProgress()` / `<-progressDone` block and BEFORE the final summary print:

```go
// ── Pass 2: retry http_timeout URLs with a longer timeout ─────────────────
if !noRetry && retryTimeoutMs > 0 && ctx.Err() == nil {
    retrySeeds, rErr := recrawler.LoadTimeoutURLs(failedDBPath)
    if rErr == nil && len(retrySeeds) > 0 {
        fmt.Printf("\n%s  %s timeout URLs → retrying with %dms timeout\n",
            infoStyle.Render("Pass 2:"),
            labelStyle.Render(formatInt64Exact(int64(len(retrySeeds)))),
            retryTimeoutMs,
        )
        retryCfg := cfg
        retryCfg.Timeout = time.Duration(retryTimeoutMs) * time.Millisecond
        // Use fewer workers for pass 2 (these are slow domains)
        retryCfg.Workers = max(workers/4, 200)
        // Reset domain fail threshold to be more generous on pass 2
        retryCfg.DomainFailThreshold = 1

        ls2 := &v3LiveStats{slowDomainMs: slowDomainMs}
        retryCfg.Notifier = ls2
        pw2 := &v3ProgressWriter{inner: &crawl.ResultDBWriter{DB: rdb}, ls: ls2}
        fw2 := &v3ProgressFailureWriter{inner: &crawl.FailedDBWriter{DB: failedDB}, ls: ls2}

        retryStart := time.Now()
        retryTotal := int64(len(retrySeeds))

        progressCtx2, cancelProgress2 := context.WithCancel(ctx)
        progressDone2 := make(chan struct{})
        go func() {
            defer close(progressDone2)
            ticker := time.NewTicker(progressInterval)
            defer ticker.Stop()
            var displayLines int
            for {
                select {
                case <-progressCtx2.Done():
                    return
                case t := <-ticker.C:
                    ls2.updateSpeed(t)
                    output := v3RenderProgress(ls2, retryCfg, engineName, retryTotal, retryStart, isTTY)
                    if isTTY {
                        if displayLines > 0 {
                            fmt.Printf("\033[%dA\033[J", displayLines)
                        }
                        fmt.Print(output)
                        displayLines = strings.Count(output, "\n")
                    } else {
                        fmt.Print(output)
                    }
                }
            }
        }()

        eng2, _ := crawl.New(engineName)
        retryStats, _ := eng2.Run(ctx, retrySeeds, dnsCache, retryCfg, pw2, fw2)
        cancelProgress2()
        <-progressDone2

        if retryStats != nil && isTTY {
            fmt.Println()
        }
        if retryStats != nil {
            fmt.Println(infoStyle.Render(fmt.Sprintf(
                "Pass 2 done: %s rescued / %s retried | avg %.0f rps | %s",
                ccFmtInt64(retryStats.OK), ccFmtInt64(retryStats.Total),
                retryStats.AvgRPS, retryStats.Duration.Truncate(time.Second),
            )))
        }
        // Merge pass 2 stats for final summary
        if retryStats != nil && stats != nil {
            stats.OK += retryStats.OK
            stats.Total += retryStats.Total
            stats.Failed += retryStats.Failed
            stats.Bytes += retryStats.Bytes
        }
    }
}
// ─────────────────────────────────────────────────────────────────────────
```

**Step 4: Build**

```bash
GOWORK=off CGO_ENABLED=1 go build ./cmd/search/ 2>&1
```
Expected: no errors.

**Step 5: Quick local smoke test**

```bash
GOWORK=off CGO_ENABLED=1 go run ./cmd/search/ hn recrawl --help 2>&1 | grep -E "retry|no-retry"
```
Expected: shows `--retry-timeout` and `--no-retry` flags.

**Step 6: Commit**

```bash
git add cli/hn.go
git commit -m "feat(hn): two-pass retry — pass 2 re-crawls timeout URLs with 20s timeout (zero false negatives)"
```

---

## Task 5: Update Config Summary Print + Display Pass Label

Show "Pass 1" in the config summary so users know a retry pass follows.

**Files:**
- Modify: `cli/hn.go` (config summary block ~line 855)

**Step 1: Add pass label to body mode line**

```go
// Change:
bodyMode := "full-body (256 KB limit)"
// To:
passLabel := "pass 1"
if noRetry || retryTimeoutMs == 0 {
    passLabel = ""
}
bodyMode := "full-body (256 KB limit)"
if passLabel != "" {
    bodyMode += fmt.Sprintf("  │  %s → pass 2 retry at %dms", passLabel, retryTimeoutMs)
}
```

**Step 2: Build + commit**

```bash
GOWORK=off CGO_ENABLED=1 go build ./cmd/search/ 2>&1
git add cli/hn.go
git commit -m "feat(hn): show pass 1/2 labels in config summary when retry enabled"
```

---

## Task 6: Deploy + Run Benchmark on server2

**Step 1: Build Linux binary**

```bash
cd blueprints/search
make build-linux
```

**Step 2: Deploy to both servers**

```bash
make deploy-linux SERVER=1
make deploy-linux SERVER=2
```

**Step 3: Kill any existing recrawl jobs**

```bash
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes root@server2 'pkill -f "search hn recrawl" || true'
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server 'pkill -f "search hn recrawl" || true'
```

**Step 4: Start benchmarks in background**

```bash
make remote-hn-recrawl-bg SERVER=2   # server2: auto-config
make remote-hn-recrawl-bg SERVER=1   # server1: auto-config
```

**Step 5: Watch live progress**

```bash
make remote-hn-tail SERVER=2
```

**Step 6: Capture and record results**

After run completes (or at 50% progress for estimate):
```bash
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes root@server2 'grep -E "Speed|✓|Auto-config|GOMEMLIMIT|Elapsed" ~/hn-recrawl.log | tail -30'
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server   'grep -E "Speed|✓|Auto-config|GOMEMLIMIT|Elapsed" ~/hn-recrawl.log | tail -30'
```

Record peak OK/s and avg OK/s. Update `spec/0617_adaptive_hardware.md` benchmarks table.

---

## Task 7: Update Spec with Post-Enhancement Results

**Files:**
- Modify: `spec/0617_adaptive_hardware.md`

Add a new "Post-Enhancement Benchmarks" section under Benchmark Results with:
- innerN=4, workers=8192, timeout=2s on server2
- Pass 1 + pass 2 combined OK rate
- Comparison to pre-enhancement numbers
- Whether 3,000 avg OK/s goal was achieved

If goal not met after this run, diagnose:
1. Check actual OK% — is it still 35%? (data quality still the bottleneck)
2. Check pass 2 rescue rate — how many timeouts were rescued?
3. Compute: with good-data seeds, do the math show 3K is achievable?

---

## Expected Numbers After All Enhancements

### server2 (6 CPUs, 10.4 GB, fd=65536)

| Metric | Before | After |
|--------|--------|-------|
| innerN | 12 | **4** |
| workers | 2,730 | **8,192** |
| timeout | 5s | **2s** |
| throughput formula | 2730/(0.4×0.3+0.6×5)=415 | 8192/(0.4×0.3+0.6×2)=**3118** |
| OK% (full domain set) | 35.6% | ~40% (with pass 2) |
| **Avg OK/s** | 148 | **~1,247** |
| Peak OK/s | 632 | **~3,000+** |

With cleaned seed data (75% OK after pass 2 filters dead domains):

| Metric | Full dataset | After 1st full pass (good seeds only) |
|--------|-------------|-------------------------------------|
| throughput | 3,118 req/s | ~10,240 req/s (network-limited ~2,560) |
| OK% | 40% | 75% |
| **Avg OK/s** | ~1,247 | **~1,920** (network-limited) |

> Network ceiling at 1 Gbps / 95 KB/page ≈ 1,350 pages/s bandwidth limit.
> To reach 3K OK/s requires either 10 Gbps link or status-only mode (4 KB/page → 32K OK/s).

### Revised 3K Path

After full analysis: the **network bandwidth** (1 Gbps) is the true ceiling for full-body crawl:
- 1 Gbps = 128 MB/s ÷ 95 KB/page (avg observed) ≈ **1,350 pages/s** for full body
- For 3K OK/s with full body: need 10 Gbps or multi-server
- For 3K OK/s with status-only (`--status-only`): 1,350 KB/s ÷ 0.1 KB/status ≈ 13,500 status/s easily

**Final recommendation for 3K full-body OK/s:**
1. Two servers working in parallel (server1 + server2), each ~1,350 OK/s → ~2,700 OK/s combined
2. Or: use `--status-only` mode where network isn't the bottleneck → easily 3K+ OK/s

---

## Quick Reference

```bash
# Build
cd blueprints/search && make build-linux

# Deploy
make deploy-linux SERVER=1
make deploy-linux SERVER=2

# Run benchmark (background)
make remote-hn-recrawl-bg SERVER=2
make remote-hn-recrawl-bg SERVER=1

# Watch
make remote-hn-tail SERVER=2
make remote-hn-tail SERVER=1

# Manual with custom params
ssh -i ~/.ssh/id_ed25519_deploy root@server2 \
  'bash -lc "nohup ~/bin/search hn recrawl --timeout 2000 >~/hn-recrawl.log 2>&1 & echo PID:$$"'

# Status-only benchmark (tests pure throughput without network ceiling)
ssh -i ~/.ssh/id_ed25519_deploy root@server2 \
  'bash -lc "~/bin/search hn recrawl --status-only"'
```
