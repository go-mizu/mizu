# 0733 — Auto Smart CC Publish: Dynamic Budget + Disk Cleanup

The CC publish scheduler uses a fixed `--max-sessions=6` regardless of hardware.
This causes severe overload on constrained servers:

- **Server 1** (4 cores, 5.8 GB RAM): 7 sessions running, only 275 MB available → near OOM
- **Server 2** (6 cores, 12 GB RAM): 6 sessions, load avg 15.69 (2.6× cores)

Additionally, leftover files from completed/crashed pipeline sessions accumulate on
disk: raw `.warc.gz` files, packed `.md.warc.gz`, orphaned `.tmp` files, and `.meta`
sidecars for already-committed shards.

This spec introduces **dynamic hardware-aware budgeting** for the scheduler and
**automatic disk cleanup** of leftovers.

---

## Design Principles

1. **No OOM** — compute max sessions from actual RAM, not a fixed constant
2. **No crash from overload** — respect CPU cores when computing budget
3. **No full disk** — clean up committed/orphaned files, pause when disk is low
4. **Backward compatible** — explicit `--max-sessions=N` (N>0) overrides auto-detection
5. **Observable** — log hardware profile, budget, and cleanup actions every round
6. **Dynamic** — re-evaluate resources every 2 minutes, scale sessions up/down

---

## 1. Hardware Detection

Reuse `arctic.DetectHardware(diskPath)` and `arctic.HardwareProfile` from `pkg/arctic/`.
The CLI package already imports `pkg/arctic` for arctic publish.

At scheduler startup:
```
hw := arctic.DetectHardware(repoRoot)
log: "Hardware: server2 (linux): 6 cores, 12 GB RAM (5 avail), 193 GB disk (126 free)"
```

---

## 2. CC Budget Computation

New type and function in `cli/cc_publish_schedule.go`:

```go
type ccBudget struct {
    MaxSessions    int     // max concurrent screen sessions
    RAMPerSession  float64 // estimated GB per pipeline session
    Reason         string  // human explanation of how budget was derived
}
```

### Budget Rules

Each pipeline session (download → pack → export) uses ~600–900 MB observed.
Use 0.9 GB as the budget per session.

| Constraint | Formula | Server 1 | Server 2 |
|------------|---------|----------|----------|
| By RAM     | `(totalRAM - 2.0) / 0.9` | 4 | 11 |
| By CPU     | `cores × 1.5` | 6 | 9 |
| Final      | `min(ramLimit, cpuLimit)`, clamped [1, 8] | **4** | **8** |

Reserve 2 GB for OS + watcher + other services (Kubernetes on server 1, arctic on server 2).

Hard cap at 8: more sessions yield diminishing returns due to disk I/O contention.

### Environment Override

`MIZU_CC_MAX_SESSIONS=N` overrides auto-detection (same as explicit `--max-sessions=N`).

### Auto Mode

When `--max-sessions=0` (new default), the scheduler auto-detects. Any positive value
uses the fixed value as before (backward compatible).

---

## 3. Adaptive Dynamic Scaling (Each 2-Minute Round)

Hardware is **re-detected every round** (not just at startup) via `arctic.DetectHardware`.
This captures live RAM availability, disk free space, and combined with load average
reading from `/proc/loadavg`, gives a full picture of system pressure — including
pressure from non-CC processes (Kubernetes, arctic, hn-live, chrome, etc).

```go
func dynamicMaxSessions(hw arctic.HardwareProfile, initialMax, nRunning int, tracker *ccResourceTracker) (int, string)
```

### Resource Pressure Response

| Condition | Action |
|-----------|--------|
| RAM available < 300 MB | **CRITICAL**: reduce to nRunning-2 (emergency shed) |
| RAM available < 500 MB | Reduce to nRunning-1 |
| RAM available < 1 GB | Hold at current (don't grow) |
| Load avg > 3 × cores | Shed proportionally: -(load - 3×cores)/cores sessions |
| Load avg > 2 × cores | Hold at current (don't grow) |
| Disk free < 20 GB | Pause all (effective max = 0) |
| Disk free < 50 GB | Reduce by 1 |
| Disk filling > 20GB in 10 rounds | Reduce by 1 (predictive) |

### Throughput-Based Bottleneck Detection

A `ccResourceTracker` maintains a sliding window of the last 30 rounds (~1 hour):

```go
type ccRoundSnapshot struct {
    round, committed, running int
    ramAvail, loadAvg, diskFree float64
}
```

**Bottleneck detection**: Compares throughput-per-session (commits/round/session) in
the first half vs second half of the history window. If the second half has more
sessions but throughput-per-session dropped >30%, we've hit an I/O bottleneck
(disk or network) — adding more sessions won't help. The scheduler holds at current
count instead of growing.

**Disk fill rate**: Tracks GB/round consumption. If the disk will hit 20 GB free
within 10 rounds (~20 min), proactively reduces sessions before the hard limit is hit.

### Session Shedding

When effective max drops below current running count:
- Kill the session with the **highest stall count** first (least productive)
- On CRITICAL RAM (< 300MB), shed 2 sessions immediately
- On proportional CPU overload, shed proportionally to the excess

---

## 4. Disk Cleanup

New function called at the start of each scheduler round and at startup:

```go
func ccCleanupLeftovers(repoRoot, crawlID string, committed map[int]struct{}) (freed int64)
```

### What Gets Cleaned

| File Type | Location | Condition |
|-----------|----------|-----------|
| `.warc.gz` (raw WARC) | `~/data/common-crawl/{crawl}/warc/` | Shard index in committed set |
| `.md.warc.gz` (packed) | `~/data/common-crawl/{crawl}/warc_md/` | Shard index in committed set |
| `.md.warc.gz.tmp` | `~/data/common-crawl/{crawl}/warc_md/` | Always (orphaned crash artifacts) |
| `.warc.path` (sidecar) | `~/data/common-crawl/{crawl}/warc_md/` | Shard index in committed set |
| `.meta` (timing sidecar) | `~/data/common-crawl/{crawl}/export/repo/data/{crawl}/` | Shard index in committed set |
| `.parquet` (exported) | `~/data/common-crawl/{crawl}/export/repo/data/{crawl}/` | Shard index in committed set AND file older than 10 min (safety: let watcher finish) |

### Safety Rules

- Never delete files for shards not in the committed set (still in pipeline)
- Never delete `.parquet` files younger than 10 minutes (watcher may be uploading)
- Log each deletion with size freed
- Run cleanup before budget evaluation so disk metrics reflect cleaned state

### Frequency

- Full cleanup at scheduler startup
- Incremental cleanup every round (only check newly committed shards)

---

## 5. Implementation Changes

### Modified Files

1. **`cli/cc_publish_schedule.go`** — all changes:
   - Import `arctic` package
   - Add `ccBudget` type and `computeCCBudget(hw)` function
   - Add `dynamicMaxSessions(hw, initial)` function
   - Add `ccCleanupLeftovers(repoRoot, crawlID, committed)` function
   - Modify `runCCSchedule()` to detect hardware, compute budget, run cleanup each round
   - Modify `ccScheduleConfig` to add `RepoRoot` field for disk detection path

2. **`cli/cc_publish.go`** — wire changes:
   - Change `--max-sessions` default from 6 to 0 (auto)
   - Log hardware profile and computed budget at startup

### No New Files

All logic lives in existing files. No new packages, no new dependencies.

---

## 6. Logging

Each round logs (to both stdout and log file):

```
Round 42 | hw: 5.8GB RAM (0.3 avail), 164GB disk (160 free), load 10.56/4 cores
         | budget: 4 max (auto: ram=4 cpu=6 → 4), effective: 3 (ram<500MB: -1)
         | cleanup: freed 1.6 GB (2 .warc.gz, 3 .tmp)
         | committed=4820 | done=8/20 chunks | running=3 | todo=6 | slots=0
```

---

## 7. CLI Changes

```
search cc publish --schedule --start 0 --end 4999
  # auto-detects hardware, computes budget, runs cleanup

search cc publish --schedule --start 0 --end 4999 --max-sessions 4
  # explicit override, skips auto-detection

MIZU_CC_MAX_SESSIONS=3 search cc publish --schedule ...
  # environment override
```
