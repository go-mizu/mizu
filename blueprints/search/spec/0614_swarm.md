# Spec 0614: Swarm v2 — Multi-Process + Async Parse Pipeline

## Objective

Redesign `pkg/crawl/swarm.go` to achieve ≥10,000 OK pages/s on the remote server
(`search hn recrawl --engine swarm`), while correctly storing full response bodies.

The current swarm is broken in two ways:
1. **Drones discard all results** — `droneResultWriter` is a noop, nothing is written to DuckDB.
2. **Hardcoded `StatusOnly: true`** — bodies are never read.

---

## Remote Server Context

| Parameter     | Value                                         |
|---------------|-----------------------------------------------|
| CPUs          | 4 × AMD EPYC                                  |
| fd limit      | 65 536                                        |
| Network       | Unknown — benchmark will determine ceiling    |
| Baseline RPS  | ~2,400 avg (keepalive, status-only, 3000 workers) |
| Target        | ≥10,000 successful OK/s with full body stored |

---

## Root Cause Analysis

### Why the current swarm underperforms

| Problem | Impact |
|---------|--------|
| Drone `droneResultWriter.Add()` is a noop | 0% of results stored |
| Drones use `NoopDNS{}` | Every request resolves DNS live (slow) |
| Drone hardcodes `Workers=500, StatusOnly=true` | Ignores parent config |
| HTML parsing runs on HTTP fetch goroutines | Parses block I/O workers |
| `peakTracker` uses mutex every request | Contention at high RPS |

### Why single-process keepalive is capped

- **HTML parsing** (`crawler.Extract`) is CPU-bound: ~1 ms per page × 7 000 HTML/s = 7 CPU-s/s needed
  (we only have 4 CPUs). Parsing on fetch workers starves HTTP concurrency.
- **Single fd table**: 65 536 fds / 8 conns/domain = 8 192 concurrent domains max.
- **Single GC heap**: ~3 000 goroutines × body bytes in-flight → GC pauses at scale.

---

## Architecture: Swarm v2

```
Queen (swarm.go)
│
├── Hash seeds by domain → N buckets  (N = cfg.DroneCount, default 4)
├── For each drone i:
│    ├── exec(self, "cc recrawl-drone", --drone-id=i, --workers=..., --result-dir=.../d{i}, ...)
│    ├── stdin:  [4-byte gob length][gob dnsFrame][seed JSON lines…][EOF]
│    └── stdout: droneStats JSON lines every 500ms
│
└── Aggregate stats from all drones → display combined live RPS

Each Drone (swarm_drone.go)
│
├── stdin reader:
│    ├── [4-byte len] → read N bytes → gob.Decode(dnsFrame) → build staticDNSCache
│    └── remaining lines → []SeedURL (JSON decode)
│
├── Stage 1 — Fetch Pool  (cfg.Workers goroutines, domain-affine keep-alive)
│    └── keepaliveFetchRaw() → pushes rawFetch to fetchCh (cap=Workers×2)
│
├── Stage 2 — Parse Pool  (runtime.NumCPU() goroutines, CPU-bound)
│    └── crawler.Extract(bodyBytes) → builds recrawler.Result → writeCh (cap=256)
│
├── Stage 3 — DB Writer  (1 goroutine, batching)
│    └── recrawler.NewResultDB(cfg.SwarmResultDir+"/d{N}", 8, cfg.BatchSize)
│         writes results_{000..007}.duckdb
│
└── stdout: droneStats JSON every 500ms until done
```

### Key Innovations

1. **DNS frame over stdin** — queen serializes resolved IPs + dead set as gob; drone builds
   `staticDNSCache` from it. Zero live DNS lookups per drone.
2. **Decoupled parse pipeline** — fetch workers never call `crawler.Extract`. They push
   `rawFetch{bodyBytes}` to a buffered channel. Dedicated CPU-bound parse workers drain it.
3. **Per-drone result dirs** — drone 0 → `results/d0/`, drone 1 → `results/d1/`.
   No DB contention between processes. Existing tools scan `results/d*/` or all `*.duckdb`.
4. **Config inheritance** — queen passes all relevant flags to drone CLI; no hardcoded values.

---

## Data Structures

### dnsFrame (transferred via stdin gob)

```go
// dnsFrame is the DNS cache snapshot sent from queen to each drone.
type dnsFrame struct {
    Resolved map[string]string // host → first resolved IP
    Dead     map[string]bool   // host → true if NXDOMAIN
}
```

### rawFetch (fetch → parse channel)

```go
// rawFetch is the output of a fetch worker before HTML extraction.
type rawFetch struct {
    seed        recrawler.SeedURL
    statusCode  int
    bodyBytes   []byte   // nil if not 200 HTML, or if status-only
    contentType string
    contentLen  int64
    redirectURL string
    fetchMs     int64
    errStr      string   // non-empty on failure
}
```

---

## Stdin Protocol (Queen → Drone)

```
Byte layout:
  [4 bytes: uint32 big-endian = gob frame length N]
  [N bytes: gob-encoded dnsFrame]
  [newline-delimited JSON: one SeedURL per line]
  [EOF — signals end of seeds]
```

Queen writes this and closes the stdin pipe.
Drone: read 4-byte header → read exactly N bytes → gob decode → switch to json.Decoder.

---

## Config Changes (`engine.go`)

Add to `Config`:

```go
// Swarm engine: base directory for drone result DBs.
// Drones write to SwarmResultDir/d{N}/results_NNN.duckdb
// Required when engine=swarm; falls back to KeepAlive if empty.
SwarmResultDir string

// Swarm engine: base directory for drone failure DBs.
// Drones write to SwarmFailedDir/failed_{N}.duckdb
SwarmFailedDir string

// Swarm engine: DuckDB write batch size for drones.
BatchSize int
```

Set in `DefaultConfig()`: `BatchSize: 5000`.

---

## CLI Changes (`cli/cc.go`)

`newCCRecrawlDrone()` gains flags to replace hardcoded values:

```go
cmd.Flags().IntVar(&workers,          "workers",               500,   "Fetch worker count")
cmd.Flags().IntVar(&timeoutMs,        "timeout",               5000,  "Per-request timeout (ms)")
cmd.Flags().IntVar(&maxConns,         "max-conns-per-domain",  4,     "Max conns per domain")
cmd.Flags().BoolVar(&statusOnly,      "status-only",           false, "Status-only mode")
cmd.Flags().IntVar(&batchSize,        "batch-size",            5000,  "DB batch size")
cmd.Flags().StringVar(&resultDir,     "result-dir",            "",    "Result DB directory")
cmd.Flags().StringVar(&failedDB,      "failed-db",             "",    "Failed URL DB path")
cmd.Flags().IntVar(&domainFailThresh, "domain-fail-threshold", 3,     "Domain abandonment threshold")
cmd.Flags().IntVar(&domainTimeoutMs,  "domain-timeout",        0,     "Per-domain timeout (ms)")
```

All values flow into `crawl.Config` and the drone runner.

---

## Queen Changes (`swarm.go`)

```go
func (e *SwarmEngine) Run(...) {
    if cfg.SearchBinary == "" || cfg.DroneCount <= 1 || cfg.SwarmResultDir == "" {
        return (&KeepAliveEngine{}).Run(...)  // graceful fallback
    }

    n := cfg.DroneCount
    buckets := domainHashBuckets(seeds, n)

    // Build DNS frame once
    frame := buildDNSFrame(dns)

    var wg sync.WaitGroup
    for i := range n {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            droneResultDir := filepath.Join(cfg.SwarmResultDir, fmt.Sprintf("d%d", idx))
            droneFailedDB  := filepath.Join(cfg.SwarmFailedDir, fmt.Sprintf("failed_%d.duckdb", idx))
            runDroneProcess(ctx, cfg, idx, buckets[idx], frame,
                droneResultDir, droneFailedDB, &stats, peak)
        }(i)
    }
    wg.Wait()
}
```

Queen passes to each drone (via exec args):
```
search cc recrawl-drone
  --drone-id=0
  --workers=3000
  --timeout=5000
  --max-conns-per-domain=8
  --status-only=false
  --batch-size=5000
  --result-dir=/home/tam/data/hn/recrawl/results/d0
  --failed-db=/home/tam/data/hn/recrawl/failed_0.duckdb
  --domain-fail-threshold=3
  --domain-timeout=30000
```

---

## Drone Runner (`swarm_drone.go`)

```go
func RunDrone(ctx context.Context, cfg Config) error {
    // 1. Read DNS frame
    frame, seeds, err := readDroneStdin(os.Stdin)
    dns := frameToCache(frame)

    // 2. Build pipeline channels
    fetchCh := make(chan rawFetch, cfg.Workers*2)
    writeCh := make(chan recrawler.Result, 256)

    // 3. Open result DB
    rdb, _ := recrawler.NewResultDB(cfg.SwarmResultDir, 8, cfg.BatchSize)
    defer rdb.Close()
    failDB, _ := recrawler.OpenFailedDB(cfg.SwarmFailedDB)
    defer failDB.Close()

    // 4. Start parse workers (NumCPU goroutines)
    parseN := runtime.NumCPU()
    var parseWg sync.WaitGroup
    for range parseN {
        parseWg.Add(1)
        go func() {
            defer parseWg.Done()
            for rf := range fetchCh {
                writeCh <- parseRawFetch(rf, cfg)
            }
        }()
    }

    // 5. DB write goroutine
    var writeWg sync.WaitGroup
    writeWg.Add(1)
    go func() {
        defer writeWg.Done()
        for r := range writeCh {
            rdb.Add(r)
        }
    }()

    // 6. Stats reporter
    go reportStats(ctx, &okCount, &failCount, &timeoutCount)

    // 7. Run fetch pipeline (domain-affine, same grouping as KeepAlive)
    runSwarmFetch(ctx, seeds, dns, cfg, fetchCh, failDB, &okCount, &failCount, &timeoutCount)

    // 8. Shutdown pipeline
    close(fetchCh)
    parseWg.Wait()
    close(writeCh)
    writeWg.Wait()

    return nil
}
```

---

## HN CLI Wiring (`cli/hn.go`)

In `runHNRecrawlV3`, before `eng.Run`:

```go
cfg.SwarmResultDir = resultDir                         // e.g. ~/data/hn/recrawl/results
cfg.SwarmFailedDir = hnCfg.WithDefaults().RecrawlDir() // e.g. ~/data/hn/recrawl
cfg.BatchSize      = batchSize
```

---

## CC CLI Wiring (`cli/cc.go`)

Same pattern in `runCCRecrawlV3`:

```go
cfg.SwarmResultDir = resultDir
cfg.SwarmFailedDir = cfg.RecrawlDir()
cfg.BatchSize      = batchSize
```

---

## Performance Model

| Parameter             | Value                           |
|-----------------------|---------------------------------|
| Drones                | 4 (one per CPU)                 |
| Fetch workers/drone   | 3 000                           |
| Total concurrent conns| 12 000                          |
| Parse workers/drone   | 4 (= NumCPU)                    |
| Total parse workers   | 16                              |
| HTML parse capacity   | 16 × 1 000/s = 16 000 parses/s  |
| HTML needed at 10K OK | 10K × 70% = 7 000 HTML/s ✓      |
| Network needed at 10K | 10K × 30 KB avg = 300 MB/s      |

If the server has < 300 MB/s network, body download will be the ceiling.
With `--status-only=false` (body mode), expected RPS:

| Mode             | Expected avg RPS | Notes                              |
|------------------|------------------|------------------------------------|
| Status-only      | ~9 000–12 000    | 4 × keepalive baseline             |
| Full body        | ~6 000–10 000    | Body read adds latency; BW-limited |

---

## Files Changed

| File                                | Change                                          |
|-------------------------------------|-------------------------------------------------|
| `pkg/crawl/engine.go`               | Add `SwarmResultDir`, `SwarmFailedDir`, `BatchSize` to Config |
| `pkg/crawl/swarm.go`                | Rewrite queen: DNS gob frame + config flag passing |
| `pkg/crawl/swarm_drone.go`          | Rewrite drone: stdin protocol + 3-stage pipeline |
| `cli/cc.go` (newCCRecrawlDrone)     | Add all config flags to drone command           |
| `cli/hn.go` (runHNRecrawlV3)        | Set SwarmResultDir, SwarmFailedDir, BatchSize   |
| `cli/cc.go` (runCCRecrawlV3)        | Set SwarmResultDir, SwarmFailedDir, BatchSize   |

---

## Deployment & Verification

```bash
# Build Linux binary
make build-linux

# Deploy to remote
make deploy-linux

# Quick test: 10K seeds, body mode
ssh -i ~/.ssh/id_ed25519_deploy tam@server \
  "~/bin/search hn recrawl --engine swarm --workers 3000 --status-only=false --limit 10000"

# Full HN recrawl (body mode)
ssh -i ~/.ssh/id_ed25519_deploy tam@server \
  "~/bin/search hn recrawl --engine swarm --workers 3000 --status-only=false"

# Expected: ≥10,000 OK/s sustained (if bandwidth allows)
```

---

## Makefile Targets to Add

```makefile
.PHONY: remote-hn-recrawl-swarm
remote-hn-recrawl-swarm: ## Run hn recrawl with swarm engine (body mode) on remote
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search hn recrawl --engine swarm --workers 3000 --status-only=false"'

.PHONY: remote-hn-recrawl-swarm-bg
remote-hn-recrawl-swarm-bg: ## Run hn recrawl swarm in background
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search hn recrawl --engine swarm --workers 3000 --status-only=false >~/hn-swarm.log 2>&1 & echo PID:$$!"'
```

---

## Success Criteria

1. `results/d0/`, `results/d1/`, `results/d2/`, `results/d3/` exist with DuckDB files after run
2. Each DuckDB file has `body` column populated (not empty string) for 200-OK HTML pages
3. Combined RPS ≥ 10,000 OK/s OR demonstrates network is the bottleneck
4. `failed_0.duckdb` ... `failed_3.duckdb` exist with failure records
5. Build compiles cleanly: `make build-quick`
