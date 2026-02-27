# Recrawl Remote + V3 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix `search cc recrawl --file p:0` stability and performance on remote server (1000 pages/s, no crash), then implement `pkg/recrawl_v3` with four independent engines (KeepAlive, Epoll, Swarm, RawHTTP).

**Architecture:** Spec 0612 fixes fd limit and DuckDB stale lock before open; spec 0613 adds `pkg/recrawl_v3` with a common `Engine` interface implemented by four strategies, all sharing existing `ResultDB`/`FailedDB` writers.

**Tech Stack:** Go 1.22+, DuckDB (duckdb-go/v2), crypto/tls, net/http, os/exec (swarm queen), existing `pkg/recrawler` types (reused).

---

## Phase 1: Remote Server Fixes (Spec 0612)

### Task 1: Add `ulimit -n 65536` to the deploy-linux wrapper script

**Files:**
- Modify: `Makefile` (deploy-linux target, wrapper printf line)

**Step 1: Read the current printf line in deploy-linux**

In `Makefile`, find the line:
```
printf '"'"'#!/usr/bin/env bash\nexport LD_LIBRARY_PATH=...
```

**Step 2: Update the printf to prepend `ulimit -n 65536`**

Change the wrapper content from:
```bash
#!/usr/bin/env bash
export LD_LIBRARY_PATH="..."
exec "..." "$@"
```
to:
```bash
#!/usr/bin/env bash
ulimit -n 65536 2>/dev/null || true
export LD_LIBRARY_PATH="..."
exec "..." "$@"
```

In the Makefile escaped form, the `printf` argument becomes:
```
'#!/usr/bin/env bash\nulimit -n 65536 2>/dev/null || true\nexport LD_LIBRARY_PATH=\"$$HOME/bin/search-libs$${LD_LIBRARY_PATH:+:$$LD_LIBRARY_PATH}\"\nexec \"$$HOME/bin/search-linux\" \"\$$@\"\n'
```

**Step 3: Deploy and verify**

```bash
make deploy-linux
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server 'bash -lc "ulimit -n"'
```
Expected: `65536`

**Step 4: Add `remote-recrawl`, `remote-recrawl-bg`, `remote-tail` targets to Makefile**

```makefile
.PHONY: remote-recrawl
remote-recrawl: ## Run cc recrawl --file p:0 --status-only on remote (foreground, 1500 workers)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search cc recrawl --file p:0 --status-only --workers 1500"'

.PHONY: remote-recrawl-bg
remote-recrawl-bg: ## Run cc recrawl --file p:0 --status-only on remote (background, ~/recrawl.log)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:0 --status-only --workers 1500 >~/recrawl.log 2>&1 & echo PID:$$!"'

.PHONY: remote-tail
remote-tail: ## Tail recrawl log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/recrawl.log'
```

**Step 5: Verify help target**
```bash
make help | grep remote
```
Expected: shows remote-recrawl, remote-recrawl-bg, remote-tail, remote-search

**Step 6: Commit**
```bash
git add Makefile
git commit -m "fix: raise ulimit to 65536 in remote wrapper, add remote-recrawl targets"
```

---

### Task 2: Stale DuckDB lock detection and cleanup

**Files:**
- Modify: `pkg/recrawler/faileddb.go` (add `OpenFailedDB`, `removeIfStaleLocked`)
- Modify: `cli/cc.go` (use `OpenFailedDB` instead of `NewFailedDB`)

**Step 1: Write the failing test**

Add `pkg/recrawler/faileddb_lock_test.go`:
```go
package recrawler

import (
    "os"
    "testing"
    "path/filepath"
)

func TestOpenFailedDB_RemovesStaleLock(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "failed.duckdb")

    // Create a fake .lock file with a dead PID
    lockPath := path + ".lock"
    // PID 99999999 almost certainly doesn't exist
    os.WriteFile(lockPath, []byte("PID=99999999\n"), 0644)
    // Create a dummy db file
    os.WriteFile(path, []byte("not a real db"), 0644)

    db, err := OpenFailedDB(path)
    if err != nil {
        t.Fatalf("OpenFailedDB should succeed after removing stale lock, got: %v", err)
    }
    db.Close()

    if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
        t.Error("stale .lock file should have been removed")
    }
}

func TestOpenFailedDB_NoCrashWithNoLockFile(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "failed.duckdb")
    db, err := OpenFailedDB(path)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    db.Close()
}
```

**Step 2: Run test to verify it fails**

```bash
cd blueprints/search && go test ./pkg/recrawler/... -run TestOpenFailedDB -v
```
Expected: `FAIL — OpenFailedDB undefined`

**Step 3: Implement `OpenFailedDB` and `removeIfStaleLocked` in `faileddb.go`**

Add after the existing `NewFailedDB` function:

```go
// OpenFailedDB is like NewFailedDB but detects and removes stale DuckDB locks
// left by dead processes before attempting to open the file.
// Safe to call even if the database file does not exist.
func OpenFailedDB(path string) (*FailedDB, error) {
    if err := removeIfStaleLocked(path); err != nil {
        return nil, fmt.Errorf("clearing stale lock: %w", err)
    }
    return NewFailedDB(path)
}

// removeIfStaleLocked checks whether the DuckDB .lock file beside dbPath
// is held by a dead process, and if so removes both the lock and the db file.
// DuckDB lock files contain lines like "PID=<n>" on Linux.
func removeIfStaleLocked(dbPath string) error {
    lockPath := dbPath + ".lock"
    data, err := os.ReadFile(lockPath)
    if errors.Is(err, os.ErrNotExist) {
        return nil // no lock → nothing to do
    }
    if err != nil {
        return nil // unreadable → let DuckDB decide
    }
    pid := parseLockFilePID(data)
    if pid <= 0 {
        return nil // can't parse PID → let DuckDB decide
    }
    if processIsAlive(pid) {
        return nil // legitimate live lock
    }
    // Dead process: remove stale lock + db so next open succeeds
    os.Remove(lockPath)
    os.Remove(dbPath)
    return nil
}

// parseLockFilePID extracts the PID from DuckDB lock file content.
// DuckDB writes "PID=<n>\n" (Linux) or may differ by version; we scan for digits.
func parseLockFilePID(data []byte) int {
    s := string(data)
    for _, line := range strings.Split(s, "\n") {
        line = strings.TrimSpace(line)
        if after, ok := strings.CutPrefix(line, "PID="); ok {
            var pid int
            fmt.Sscanf(after, "%d", &pid)
            return pid
        }
    }
    // fallback: first integer token in file
    var pid int
    fmt.Sscanf(s, "%d", &pid)
    return pid
}

// processIsAlive returns true if the given PID has a running process.
// Uses kill(pid, 0) semantics: no signal sent, just checks existence.
func processIsAlive(pid int) bool {
    if pid <= 0 {
        return false
    }
    proc, err := os.FindProcess(pid)
    if err != nil {
        return false
    }
    // On Linux, FindProcess always succeeds; send signal 0 to test liveness.
    err = proc.Signal(syscall.Signal(0))
    return err == nil
}
```

Add required imports to faileddb.go: `"errors"`, `"fmt"`, `"os"`, `"strings"`, `"syscall"`.

**Step 4: Update `cli/cc.go` — replace `NewFailedDB` call with `OpenFailedDB`**

At line ~1403:
```go
// Before:
failedDB, err := recrawler.NewFailedDB(failedDBPath)
// After:
failedDB, err := recrawler.OpenFailedDB(failedDBPath)
```

**Step 5: Run tests**

```bash
cd blueprints/search && go test ./pkg/recrawler/... -run TestOpenFailedDB -v
```
Expected: PASS

**Step 6: Build and verify no compile errors**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go build ./cmd/search/
```

**Step 7: Commit**
```bash
git add pkg/recrawler/faileddb.go pkg/recrawler/faileddb_lock_test.go cli/cc.go
git commit -m "fix: detect and remove stale DuckDB locks before opening failed.duckdb"
```

---

### Task 3: Deploy and benchmark Phase 1

**Step 1: Build and deploy**
```bash
make deploy-linux
```
Expected: "Deployed: tam@server:~/bin/search"

**Step 2: Verify ulimit**
```bash
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server 'bash -lc "source ~/bin/search 2>/dev/null; ulimit -n"' 2>&1 || \
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server '~/bin/search --version 2>&1 | head -1; cat ~/bin/search'
```

**Step 3: Kill any stale process, run benchmark with 1500 workers**
```bash
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server \
  'pkill -f search-linux 2>/dev/null; sleep 1; timeout 120 ~/bin/search cc recrawl --file p:0 --status-only --workers 1500 --limit 100000 2>&1' 2>&1
```
Expected: Peak >= 1000 pages/s

**Step 4: Test crash recovery — kill mid-run, re-run**
```bash
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server \
  'nohup ~/bin/search cc recrawl --file p:0 --status-only --workers 1500 --limit 5000 >/tmp/r.log 2>&1 & PID=$!; sleep 5; kill $PID; sleep 2; timeout 30 ~/bin/search cc recrawl --file p:0 --status-only --workers 100 --limit 1000 2>&1'
```
Expected: second run starts without "Conflicting lock" error

---

## Phase 2: pkg/recrawl_v3 Package (Spec 0613)

### Task 4: Create package skeleton with interface, types, and config

**Files:**
- Create: `pkg/recrawl_v3/engine.go`
- Create: `pkg/recrawl_v3/types.go`

**Step 1: Create `engine.go`**

```go
// Package recrawl_v3 implements four independent high-performance recrawl engines.
// All engines implement the Engine interface and share ResultWriter / FailureWriter.
package recrawl_v3

import (
    "context"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

// Engine is implemented by all four v3 strategies.
type Engine interface {
    Run(ctx context.Context, seeds []recrawler.SeedURL, dns DNSCache, cfg Config,
        results ResultWriter, failures FailureWriter) (*Stats, error)
}

// Stats holds performance counters returned after a Run.
type Stats struct {
    Total    int64
    OK       int64
    Failed   int64
    Timeout  int64
    PeakRPS  float64
    AvgRPS   float64
    Duration time.Duration
    P95LatMs int64
    MemRSS   int64 // bytes at end of run
}

// Config configures any engine.
type Config struct {
    Workers           int           // concurrent workers (engines A, D, C-drone)
    Timeout           time.Duration // per-request HTTP timeout
    StatusOnly        bool          // discard body, read status line only
    MaxConnsPerDomain int           // max simultaneous connections per domain (engine A)
    UserAgent         string
    InsecureTLS       bool   // skip TLS verification
    DroneCount        int    // swarm engine: number of drone processes (engine C)
    SearchBinary      string // path to self binary (engine C drones re-exec it)
}

// DefaultConfig returns sensible defaults for the remote server.
func DefaultConfig() Config {
    return Config{
        Workers:           1500,
        Timeout:           5 * time.Second,
        StatusOnly:        true,
        MaxConnsPerDomain: 4,
        UserAgent:         "MizuCrawler/3.0",
        InsecureTLS:       true,
        DroneCount:        4,
    }
}

// DNSCache is a read-only pre-resolved host→IP mapping.
type DNSCache interface {
    // Lookup returns the first resolved IP for host, or ok=false.
    Lookup(host string) (ip string, ok bool)
    // IsDead returns true if host resolved to NXDOMAIN.
    IsDead(host string) bool
}

// ResultWriter accepts crawl results.
type ResultWriter interface {
    Add(r recrawler.Result)
    Flush(ctx context.Context) error
    Close() error
}

// FailureWriter accepts failed URLs.
type FailureWriter interface {
    AddURL(u recrawler.FailedURL)
    Close() error
}

// New returns the named engine. Valid names: "keepalive", "epoll", "swarm", "rawhttp".
// Returns an error for unknown names.
func New(name string) (Engine, error) {
    switch name {
    case "keepalive":
        return &KeepAliveEngine{}, nil
    case "epoll":
        return &EpollEngine{}, nil
    case "swarm":
        return &SwarmEngine{}, nil
    case "rawhttp":
        return &RawHTTPEngine{}, nil
    default:
        return nil, fmt.Errorf("unknown engine %q (valid: keepalive, epoll, swarm, rawhttp)", name)
    }
}
```

**Step 2: Create `types.go`**

```go
package recrawl_v3

import (
    "runtime"
    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

// staticDNSCache wraps recrawler.DNSResolver for the DNSCache interface.
type staticDNSCache struct {
    r *recrawler.DNSResolver
}

func (s *staticDNSCache) Lookup(host string) (string, bool) {
    ips := s.r.ResolvedIPs(host)
    if len(ips) == 0 {
        return "", false
    }
    return ips[0], true
}

func (s *staticDNSCache) IsDead(host string) bool {
    return s.r.IsDead(host)
}

// WrapDNSResolver adapts a *recrawler.DNSResolver to DNSCache.
func WrapDNSResolver(r *recrawler.DNSResolver) DNSCache {
    return &staticDNSCache{r: r}
}

// rssNow returns current process RSS memory in bytes.
func rssNow() int64 {
    var ms runtime.MemStats
    runtime.ReadMemStats(&ms)
    return int64(ms.Sys)
}
```

**Step 3: Verify package compiles**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go build ./pkg/recrawl_v3/
```
Expected: no errors (stubs can return nil for now)

**Step 4: Commit**
```bash
git add pkg/recrawl_v3/
git commit -m "feat: add pkg/recrawl_v3 skeleton — Engine interface, Config, DNSCache"
```

---

### Task 5: Engine A — KeepAlive (domain-affine keep-alive pools)

**Files:**
- Create: `pkg/recrawl_v3/keepalive.go`
- Create: `pkg/recrawl_v3/keepalive_test.go`

**Step 1: Write failing test**

```go
// keepalive_test.go
package recrawl_v3

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

func TestKeepAliveEngine_BasicCrawl(t *testing.T) {
    // Start a test HTTP server that counts requests
    var reqCount int
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        reqCount++
        w.WriteHeader(200)
    }))
    defer srv.Close()

    seeds := make([]recrawler.SeedURL, 20)
    for i := range seeds {
        seeds[i] = recrawler.SeedURL{
            URL:    srv.URL + "/page/" + string(rune('a'+i)),
            Domain: "localhost",
            Host:   "localhost",
        }
    }

    cfg := DefaultConfig()
    cfg.Workers = 4
    cfg.Timeout = 2 * time.Second
    cfg.InsecureTLS = false // test server is HTTP

    eng := &KeepAliveEngine{}
    rw := &noopResultWriter{}
    fw := &noopFailureWriter{}
    stats, err := eng.Run(context.Background(), seeds, &noopDNS{}, cfg, rw, fw)
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }
    if stats.OK != 20 {
        t.Errorf("want 20 OK, got %d (failed=%d)", stats.OK, stats.Failed)
    }
    if reqCount != 20 {
        t.Errorf("server saw %d requests, want 20", reqCount)
    }
}

// noopResultWriter, noopFailureWriter, noopDNS are test stubs
type noopResultWriter struct{}
func (n *noopResultWriter) Add(_ recrawler.Result) {}
func (n *noopResultWriter) Flush(_ context.Context) error { return nil }
func (n *noopResultWriter) Close() error { return nil }

type noopFailureWriter struct{}
func (n *noopFailureWriter) AddURL(_ recrawler.FailedURL) {}
func (n *noopFailureWriter) Close() error { return nil }

type noopDNS struct{}
func (n *noopDNS) Lookup(_ string) (string, bool) { return "", false }
func (n *noopDNS) IsDead(_ string) bool { return false }
```

**Step 2: Run test to verify it fails**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -run TestKeepAliveEngine -v
```
Expected: FAIL — `KeepAliveEngine` undefined

**Step 3: Implement `keepalive.go`**

```go
package recrawl_v3

import (
    "context"
    "crypto/tls"
    "net/http"
    "strings"
    "sync"
    "sync/atomic"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
    "golang.org/x/sync/errgroup"
)

// KeepAliveEngine groups URLs by domain and processes each domain's URLs
// with a single http.Client that reuses keep-alive connections.
// Workers are domain-keyed: one goroutine handles all URLs for one domain.
type KeepAliveEngine struct{}

func (e *KeepAliveEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
    dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

    // Group URLs by domain
    byDomain := make(map[string][]recrawler.SeedURL, 1024)
    for _, s := range seeds {
        if dns.IsDead(s.Host) {
            failures.AddURL(recrawler.FailedURL{
                URL: s.URL, Domain: s.Domain, Reason: "domain_dead",
            })
            continue
        }
        byDomain[s.Domain] = append(byDomain[s.Domain], s)
    }

    type domainWork struct {
        domain string
        urls   []recrawler.SeedURL
    }

    workCh := make(chan domainWork, len(byDomain))
    for d, us := range byDomain {
        workCh <- domainWork{d, us}
    }
    close(workCh)

    maxWorkers := cfg.Workers
    if maxWorkers <= 0 {
        maxWorkers = 500
    }
    if maxWorkers > len(byDomain) {
        maxWorkers = len(byDomain)
    }

    var (
        ok      atomic.Int64
        failed  atomic.Int64
        timeout atomic.Int64
        total   atomic.Int64
    )

    start := time.Now()
    peakRPS := &peakTracker{}

    g, gctx := errgroup.WithContext(ctx)

    for range maxWorkers {
        g.Go(func() error {
            for work := range workCh {
                if gctx.Err() != nil {
                    return nil
                }
                processOneDomain(gctx, work.domain, work.urls, dns, cfg,
                    results, failures, &ok, &failed, &timeout, &total, peakRPS)
            }
            return nil
        })
    }

    g.Wait()

    dur := time.Since(start)
    tot := total.Load()
    avgRPS := 0.0
    if dur.Seconds() > 0 {
        avgRPS = float64(tot) / dur.Seconds()
    }

    return &Stats{
        Total:    tot,
        OK:       ok.Load(),
        Failed:   failed.Load(),
        Timeout:  timeout.Load(),
        PeakRPS:  peakRPS.Peak(),
        AvgRPS:   avgRPS,
        Duration: dur,
        MemRSS:   rssNow(),
    }, nil
}

func processOneDomain(ctx context.Context, domain string, urls []recrawler.SeedURL,
    dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter,
    ok, failed, timeout, total *atomic.Int64, peak *peakTracker) {

    tlsCfg := &tls.Config{InsecureSkipVerify: cfg.InsecureTLS, ServerName: domain}
    transport := &http.Transport{
        TLSClientConfig:     tlsCfg,
        MaxIdleConnsPerHost: cfg.MaxConnsPerDomain,
        IdleConnTimeout:     15 * time.Second,
        DisableCompression:  true,
    }
    if ip, found := dns.Lookup(domain); found {
        transport.DialContext = dialWithIP(ip)
    }
    client := &http.Client{
        Transport: transport,
        Timeout:   cfg.Timeout,
        CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
            return http.ErrUseLastResponse
        },
    }

    for _, seed := range urls {
        if ctx.Err() != nil {
            return
        }
        r := fetchOne(ctx, client, seed, cfg)
        total.Add(1)
        peak.Record()
        if r.Error != "" && strings.Contains(r.Error, "timeout") {
            timeout.Add(1)
        } else if r.Error != "" {
            failed.Add(1)
            failures.AddURL(recrawler.FailedURL{
                URL: seed.URL, Domain: seed.Domain, Reason: "http_error",
                Error: r.Error, FetchTimeMs: r.FetchTimeMs,
            })
        } else {
            ok.Add(1)
        }
        results.Add(r)
    }
    transport.CloseIdleConnections()
}

func fetchOne(ctx context.Context, client *http.Client, seed recrawler.SeedURL, cfg Config) recrawler.Result {
    start := time.Now()
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, seed.URL, nil)
    if err != nil {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: time.Since(start).Milliseconds()}
    }
    req.Header.Set("User-Agent", cfg.UserAgent)

    resp, err := client.Do(req)
    ms := time.Since(start).Milliseconds()
    if err != nil {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms}
    }
    defer resp.Body.Close()
    if cfg.StatusOnly {
        // Drain minimal body to allow connection reuse
        buf := make([]byte, 1)
        resp.Body.Read(buf) //nolint:errcheck
    }
    return recrawler.Result{
        URL:         seed.URL,
        Domain:      seed.Domain,
        StatusCode:  resp.StatusCode,
        ContentType: resp.Header.Get("Content-Type"),
        RedirectURL: resp.Header.Get("Location"),
        FetchTimeMs: ms,
        CrawledAt:   time.Now(),
    }
}

// peakTracker computes peak RPS over a sliding 1-second window.
type peakTracker struct {
    mu      sync.Mutex
    windows []int64
    last    time.Time
    cur     int64
    peak    float64
}

func (p *peakTracker) Record() {
    p.mu.Lock()
    defer p.mu.Unlock()
    now := time.Now()
    if p.last.IsZero() {
        p.last = now
    }
    p.cur++
    if now.Sub(p.last) >= time.Second {
        rps := float64(p.cur) / now.Sub(p.last).Seconds()
        if rps > p.peak {
            p.peak = rps
        }
        p.cur = 0
        p.last = now
    }
}

func (p *peakTracker) Peak() float64 {
    p.mu.Lock()
    defer p.mu.Unlock()
    return p.peak
}
```

Add `dialWithIP` helper to `types.go`:
```go
import (
    "context"
    "net"
)

// dialWithIP returns a DialContext func that connects to a pre-resolved IP
// but sends the original hostname in TLS SNI / HTTP Host header.
func dialWithIP(ip string) func(ctx context.Context, network, addr string) (net.Conn, error) {
    return func(ctx context.Context, network, addr string) (net.Conn, error) {
        _, port, err := net.SplitHostPort(addr)
        if err != nil {
            port = "443"
        }
        d := &net.Dialer{Timeout: 5 * time.Second}
        return d.DialContext(ctx, "tcp", net.JoinHostPort(ip, port))
    }
}
```

**Step 4: Run test**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -run TestKeepAliveEngine -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add pkg/recrawl_v3/keepalive.go pkg/recrawl_v3/keepalive_test.go pkg/recrawl_v3/types.go
git commit -m "feat: add recrawl_v3 Engine A — domain-affine keep-alive pool"
```

---

### Task 6: Engine B — Epoll (small goroutine pool with raw net.Conn)

**Files:**
- Create: `pkg/recrawl_v3/epoll.go`
- Add tests to: `pkg/recrawl_v3/keepalive_test.go` (shared stubs)

**Step 1: Write failing test**

Add to `keepalive_test.go` (reuses same test server setup):
```go
func TestEpollEngine_BasicCrawl(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
    }))
    defer srv.Close()

    seeds := make([]recrawler.SeedURL, 20)
    for i := range seeds {
        seeds[i] = recrawler.SeedURL{URL: srv.URL + "/e/" + string(rune('a'+i)),
            Domain: "localhost", Host: "localhost"}
    }
    cfg := DefaultConfig()
    cfg.Workers = 4
    cfg.Timeout = 2 * time.Second
    cfg.InsecureTLS = false

    eng := &EpollEngine{}
    stats, err := eng.Run(context.Background(), seeds, &noopDNS{}, cfg,
        &noopResultWriter{}, &noopFailureWriter{})
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }
    if stats.OK != 20 {
        t.Errorf("want 20 OK, got %d", stats.OK)
    }
}
```

**Step 2: Run to verify fail**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -run TestEpollEngine -v
```
Expected: FAIL

**Step 3: Implement `epoll.go`**

```go
package recrawl_v3

import (
    "bufio"
    "bytes"
    "context"
    "crypto/tls"
    "fmt"
    "net"
    "runtime"
    "strconv"
    "sync/atomic"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
    "golang.org/x/sync/errgroup"
)

// EpollEngine uses a fixed goroutine pool (4×nCPU) where each goroutine
// handles requests sequentially using raw net.Conn + explicit SetDeadline.
// No net/http, no goroutine-per-connection overhead.
type EpollEngine struct{}

func (e *EpollEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
    dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

    numWorkers := 4 * runtime.NumCPU()
    if cfg.Workers > 0 && cfg.Workers < numWorkers {
        numWorkers = cfg.Workers
    }

    workCh := make(chan recrawler.SeedURL, len(seeds))
    for _, s := range seeds {
        workCh <- s
    }
    close(workCh)

    var ok, failed, timeout, total atomic.Int64
    start := time.Now()
    peak := &peakTracker{}

    g, gctx := errgroup.WithContext(ctx)
    for range numWorkers {
        g.Go(func() error {
            for seed := range workCh {
                if gctx.Err() != nil {
                    return nil
                }
                r := rawFetch(gctx, seed, dns, cfg)
                total.Add(1)
                peak.Record()
                switch {
                case r.Error != "" && isTimeout(r.Error):
                    timeout.Add(1)
                    failures.AddURL(recrawler.FailedURL{URL: seed.URL, Domain: seed.Domain,
                        Reason: "http_timeout", Error: r.Error})
                case r.Error != "":
                    failed.Add(1)
                    failures.AddURL(recrawler.FailedURL{URL: seed.URL, Domain: seed.Domain,
                        Reason: "http_error", Error: r.Error})
                default:
                    ok.Add(1)
                }
                results.Add(r)
            }
            return nil
        })
    }
    g.Wait()

    dur := time.Since(start)
    tot := total.Load()
    avgRPS := 0.0
    if dur.Seconds() > 0 {
        avgRPS = float64(tot) / dur.Seconds()
    }
    return &Stats{
        Total: tot, OK: ok.Load(), Failed: failed.Load(), Timeout: timeout.Load(),
        PeakRPS: peak.Peak(), AvgRPS: avgRPS, Duration: dur, MemRSS: rssNow(),
    }, nil
}

// rawFetch dials a TCP connection, sends a minimal HTTP/1.1 GET, reads the status line.
func rawFetch(ctx context.Context, seed recrawler.SeedURL, dns DNSCache, cfg Config) recrawler.Result {
    start := time.Now()
    ms := func() int64 { return time.Since(start).Milliseconds() }

    u, err := parseURL(seed.URL)
    if err != nil {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms()}
    }

    host, port := u.Hostname(), u.Port()
    if port == "" {
        if u.Scheme == "https" {
            port = "443"
        } else {
            port = "80"
        }
    }

    // Prefer pre-resolved IP
    dialAddr := net.JoinHostPort(host, port)
    if ip, ok := dns.Lookup(host); ok {
        dialAddr = net.JoinHostPort(ip, port)
    }

    deadline := time.Now().Add(cfg.Timeout)
    dialCtx, cancel := context.WithDeadline(ctx, deadline)
    defer cancel()

    conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", dialAddr)
    if err != nil {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms()}
    }
    defer conn.Close()
    conn.SetDeadline(deadline)

    var rwConn net.Conn = conn
    if u.Scheme == "https" {
        tlsConn := tls.Client(conn, &tls.Config{
            InsecureSkipVerify: cfg.InsecureTLS,
            ServerName:         host,
        })
        if err := tlsConn.Handshake(); err != nil {
            return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
                Error: "tls: " + err.Error(), FetchTimeMs: ms()}
        }
        rwConn = tlsConn
    }

    // Build minimal HTTP request
    path := u.RequestURI()
    req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nConnection: close\r\n\r\n",
        path, host, cfg.UserAgent)
    if _, err := rwConn.Write([]byte(req)); err != nil {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms()}
    }

    // Read only status line
    br := bufio.NewReaderSize(rwConn, 512)
    line, err := br.ReadString('\n')
    if err != nil && len(line) < 12 {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms()}
    }

    // Parse: "HTTP/1.1 200 OK\r\n" → status code at bytes 9–11
    code := 0
    if len(line) >= 12 {
        code, _ = strconv.Atoi(string(bytes.TrimSpace([]byte(line[9:12]))))
    }

    return recrawler.Result{
        URL: seed.URL, Domain: seed.Domain, StatusCode: code,
        FetchTimeMs: ms(), CrawledAt: time.Now(),
    }
}

func isTimeout(errStr string) bool {
    return strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline")
}
```

Add `parseURL` helper in `types.go`:
```go
import "net/url"

func parseURL(rawURL string) (*url.URL, error) {
    return url.Parse(rawURL)
}
```

**Step 4: Run test**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -run TestEpollEngine -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add pkg/recrawl_v3/epoll.go
git commit -m "feat: add recrawl_v3 Engine B — epoll-style fixed goroutine pool with raw net.Conn"
```

---

### Task 7: Engine C — Swarm (queen/drone multi-process)

**Files:**
- Create: `pkg/recrawl_v3/swarm.go`
- Create: `pkg/recrawl_v3/swarm_drone.go`

**Step 1: Write failing test**

```go
func TestSwarmEngine_BasicCrawl(t *testing.T) {
    // Swarm spawns child processes; we test the queen logic with DroneCount=1
    // by stubbing out exec.Command in tests (skip if no binary path available)
    if os.Getenv("SEARCH_BINARY") == "" {
        t.Skip("SEARCH_BINARY not set; set to search binary path for swarm test")
    }
    // ... integration test only via SEARCH_BINARY env
}
```

Note: Swarm engine's unit test is an integration test (requires compiled binary).
The Run() method degrades gracefully if SearchBinary is empty: falls back to KeepAlive.

**Step 2: Implement `swarm.go`**

```go
package recrawl_v3

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "os/exec"
    "sync"
    "sync/atomic"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

// SwarmEngine spawns N drone sub-processes, distributes seeds by domain hash,
// and aggregates stats. Each drone writes its own DuckDB shard.
// If SearchBinary is empty, falls back to KeepAliveEngine.
type SwarmEngine struct{}

// droneStats is the JSON line each drone writes to stdout.
type droneStats struct {
    OK      int64   `json:"ok"`
    Failed  int64   `json:"failed"`
    Timeout int64   `json:"timeout"`
    RPS     float64 `json:"rps"`
}

func (e *SwarmEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
    dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

    if cfg.SearchBinary == "" || cfg.DroneCount <= 1 {
        // Graceful fallback: run KeepAlive in-process
        return (&KeepAliveEngine{}).Run(ctx, seeds, dns, cfg, results, failures)
    }

    n := cfg.DroneCount
    // Partition seeds by domain hash into N buckets
    buckets := make([][]recrawler.SeedURL, n)
    for _, s := range seeds {
        h := fnvHash(s.Domain)
        buckets[h%uint32(n)] = append(buckets[h%uint32(n)], s)
    }

    var (
        totalOK      atomic.Int64
        totalFailed  atomic.Int64
        totalTimeout atomic.Int64
        totalReqs    atomic.Int64
    )

    start := time.Now()
    peak := &peakTracker{}
    var wg sync.WaitGroup

    for i := range n {
        wg.Add(1)
        go func(droneIdx int, droneSeeds []recrawler.SeedURL) {
            defer wg.Done()
            if err := runDrone(ctx, cfg.SearchBinary, droneIdx, droneSeeds,
                &totalOK, &totalFailed, &totalTimeout, &totalReqs, peak); err != nil {
                fmt.Fprintf(os.Stderr, "drone %d error: %v\n", droneIdx, err)
            }
        }(i, buckets[i])
    }

    wg.Wait()
    dur := time.Since(start)
    tot := totalReqs.Load()
    avgRPS := 0.0
    if dur.Seconds() > 0 {
        avgRPS = float64(tot) / dur.Seconds()
    }

    return &Stats{
        Total: tot, OK: totalOK.Load(), Failed: totalFailed.Load(), Timeout: totalTimeout.Load(),
        PeakRPS: peak.Peak(), AvgRPS: avgRPS, Duration: dur, MemRSS: rssNow(),
    }, nil
}

func runDrone(ctx context.Context, binary string, idx int, seeds []recrawler.SeedURL,
    ok, failed, timeout, total *atomic.Int64, peak *peakTracker) error {

    cmd := exec.CommandContext(ctx, binary, "cc", "recrawl-drone",
        fmt.Sprintf("--drone-id=%d", idx))
    stdin, err := cmd.StdinPipe()
    if err != nil {
        return err
    }
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return err
    }
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return err
    }

    // Write seeds as JSON lines to drone stdin
    enc := json.NewEncoder(stdin)
    for _, s := range seeds {
        enc.Encode(s)
    }
    stdin.Close()

    // Read stats lines from drone stdout
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        var ds droneStats
        if err := json.Unmarshal(scanner.Bytes(), &ds); err != nil {
            continue
        }
        ok.Add(ds.OK)
        failed.Add(ds.Failed)
        timeout.Add(ds.Timeout)
        total.Add(ds.OK + ds.Failed + ds.Timeout)
        peak.Record()
    }

    return cmd.Wait()
}

func fnvHash(s string) uint32 {
    h := uint32(2166136261)
    for i := 0; i < len(s); i++ {
        h ^= uint32(s[i])
        h *= 16777619
    }
    return h
}
```

**Step 3: Implement `swarm_drone.go`** (drone subcommand handler)

```go
package recrawl_v3

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

// RunDrone is called by the "cc recrawl-drone" subcommand.
// Reads SeedURL JSON lines from stdin, crawls with KeepAliveEngine,
// writes droneStats JSON lines to stdout every 500ms.
func RunDrone(ctx context.Context, cfg Config) error {
    var seeds []recrawler.SeedURL
    scanner := bufio.NewScanner(os.Stdin)
    scanner.Buffer(make([]byte, 1<<20), 1<<20)
    for scanner.Scan() {
        var s recrawler.SeedURL
        if err := json.Unmarshal(scanner.Bytes(), &s); err == nil {
            seeds = append(seeds, s)
        }
    }

    var lastOK, lastFailed, lastTimeout int64
    ticker := time.NewTicker(500 * time.Millisecond)
    statCh := make(chan droneStats, 100)

    go func() {
        defer ticker.Stop()
        for range ticker.C {
            select {
            case ds := <-statCh:
                enc := json.NewEncoder(os.Stdout)
                enc.Encode(ds)
            case <-ctx.Done():
                return
            }
        }
    }()

    rw := &countingWriter{statCh: statCh, lastOK: &lastOK,
        lastFailed: &lastFailed, lastTimeout: &lastTimeout}
    fw := &noopFailureWriter{}

    _, err := (&KeepAliveEngine{}).Run(ctx, seeds, &noopDNS{}, cfg, rw, fw)

    // flush final stats
    fmt.Fprintf(os.Stdout, `{"ok":%d,"failed":%d,"timeout":%d,"rps":0}`+"\n",
        lastOK, lastFailed, lastTimeout)
    return err
}

type countingWriter struct {
    statCh                       chan droneStats
    lastOK, lastFailed, lastTimeout *int64
    buf                          []recrawler.Result
}
func (c *countingWriter) Add(r recrawler.Result) {
    if r.Error == "" { *c.lastOK++ } else { *c.lastFailed++ }
}
func (c *countingWriter) Flush(_ context.Context) error { return nil }
func (c *countingWriter) Close() error                  { return nil }
```

**Step 4: Run tests**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -run TestSwarm -v
```
Expected: SKIP (SEARCH_BINARY not set) — OK

**Step 5: Commit**
```bash
git add pkg/recrawl_v3/swarm.go pkg/recrawl_v3/swarm_drone.go
git commit -m "feat: add recrawl_v3 Engine C — multi-process swarm (queen/drone)"
```

---

### Task 8: Engine D — RawHTTP (bypass net/http, custom HTTP/1.1)

**Files:**
- Create: `pkg/recrawl_v3/rawhttp.go`
- Create: `pkg/recrawl_v3/rawhttp_pool.go`

**Step 1: Write failing test**

Add to test file:
```go
func TestRawHTTPEngine_BasicCrawl(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(200)
    }))
    defer srv.Close()

    seeds := make([]recrawler.SeedURL, 20)
    for i := range seeds {
        seeds[i] = recrawler.SeedURL{URL: srv.URL + "/r/" + string(rune('a'+i)),
            Domain: "localhost", Host: "localhost"}
    }
    cfg := DefaultConfig()
    cfg.Workers = 4
    cfg.Timeout = 2 * time.Second
    cfg.InsecureTLS = false

    eng := &RawHTTPEngine{}
    stats, err := eng.Run(context.Background(), seeds, &noopDNS{}, cfg,
        &noopResultWriter{}, &noopFailureWriter{})
    if err != nil {
        t.Fatalf("Run failed: %v", err)
    }
    if stats.OK != 20 {
        t.Errorf("want 20 OK, got %d", stats.OK)
    }
}
```

**Step 2: Implement `rawhttp.go`**

RawHTTPEngine wraps EpollEngine's `rawFetch` with optional connection pooling:

```go
package recrawl_v3

import (
    "context"
    "sync/atomic"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
    "golang.org/x/sync/errgroup"
)

// RawHTTPEngine uses raw net.Conn (bypassing net/http) with an optional
// per-host connection pool for keep-alive reuse.
// Unlike EpollEngine (fixed tiny pool), RawHTTPEngine scales workers like KeepAlive
// but avoids all net/http allocation overhead.
type RawHTTPEngine struct{}

func (e *RawHTTPEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
    dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

    pool := newRawConnPool(cfg.MaxConnsPerDomain, cfg.Timeout)
    defer pool.CloseAll()

    maxWorkers := cfg.Workers
    if maxWorkers <= 0 {
        maxWorkers = 500
    }

    workCh := make(chan recrawler.SeedURL, min(len(seeds), 10000))
    go func() {
        for _, s := range seeds {
            workCh <- s
        }
        close(workCh)
    }()

    var ok, failed, timeout, total atomic.Int64
    start := time.Now()
    peak := &peakTracker{}

    g, gctx := errgroup.WithContext(ctx)
    for range maxWorkers {
        g.Go(func() error {
            for seed := range workCh {
                if gctx.Err() != nil {
                    return nil
                }
                r := rawFetchPooled(gctx, seed, dns, cfg, pool)
                total.Add(1)
                peak.Record()
                switch {
                case r.Error != "" && isTimeout(r.Error):
                    timeout.Add(1)
                    failures.AddURL(recrawler.FailedURL{URL: seed.URL, Domain: seed.Domain,
                        Reason: "http_timeout", Error: r.Error})
                case r.Error != "":
                    failed.Add(1)
                    failures.AddURL(recrawler.FailedURL{URL: seed.URL, Domain: seed.Domain,
                        Reason: "http_error", Error: r.Error})
                default:
                    ok.Add(1)
                }
                results.Add(r)
            }
            return nil
        })
    }
    g.Wait()

    dur := time.Since(start)
    tot := total.Load()
    avgRPS := 0.0
    if dur.Seconds() > 0 {
        avgRPS = float64(tot) / dur.Seconds()
    }
    return &Stats{
        Total: tot, OK: ok.Load(), Failed: failed.Load(), Timeout: timeout.Load(),
        PeakRPS: peak.Peak(), AvgRPS: avgRPS, Duration: dur, MemRSS: rssNow(),
    }, nil
}
```

**Step 3: Implement `rawhttp_pool.go`**

```go
package recrawl_v3

import (
    "context"
    "crypto/tls"
    "fmt"
    "net"
    "sync"
    "time"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

// rawConnPool maintains a per-host pool of reusable net.Conn connections.
// Connections are returned via Put() after a successful request.
type rawConnPool struct {
    mu      sync.Mutex
    pools   map[string][]net.Conn
    maxPer  int
    timeout time.Duration
}

func newRawConnPool(maxPerHost int, timeout time.Duration) *rawConnPool {
    if maxPerHost <= 0 {
        maxPerHost = 4
    }
    return &rawConnPool{
        pools:   make(map[string][]net.Conn),
        maxPer:  maxPerHost,
        timeout: timeout,
    }
}

func (p *rawConnPool) Get(key string) (net.Conn, bool) {
    p.mu.Lock()
    defer p.mu.Unlock()
    conns := p.pools[key]
    if len(conns) == 0 {
        return nil, false
    }
    c := conns[len(conns)-1]
    p.pools[key] = conns[:len(conns)-1]
    return c, true
}

func (p *rawConnPool) Put(key string, c net.Conn) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if len(p.pools[key]) >= p.maxPer {
        c.Close()
        return
    }
    p.pools[key] = append(p.pools[key], c)
}

func (p *rawConnPool) CloseAll() {
    p.mu.Lock()
    defer p.mu.Unlock()
    for _, conns := range p.pools {
        for _, c := range conns {
            c.Close()
        }
    }
}

// rawFetchPooled is like rawFetch but tries to reuse a pooled connection.
func rawFetchPooled(ctx context.Context, seed recrawler.SeedURL, dns DNSCache,
    cfg Config, pool *rawConnPool) recrawler.Result {

    start := time.Now()
    ms := func() int64 { return time.Since(start).Milliseconds() }

    u, err := parseURL(seed.URL)
    if err != nil {
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms()}
    }

    host, port := u.Hostname(), u.Port()
    if port == "" {
        if u.Scheme == "https" { port = "443" } else { port = "80" }
    }
    poolKey := fmt.Sprintf("%s://%s:%s", u.Scheme, host, port)

    var conn net.Conn
    if c, ok := pool.Get(poolKey); ok {
        conn = c
    } else {
        dialAddr := net.JoinHostPort(host, port)
        if ip, ok := dns.Lookup(host); ok {
            dialAddr = net.JoinHostPort(ip, port)
        }
        dialCtx, cancel := context.WithDeadline(ctx, time.Now().Add(cfg.Timeout))
        defer cancel()
        conn, err = (&net.Dialer{}).DialContext(dialCtx, "tcp", dialAddr)
        if err != nil {
            return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
                Error: err.Error(), FetchTimeMs: ms()}
        }

        if u.Scheme == "https" {
            tlsConn := tls.Client(conn, &tls.Config{
                InsecureSkipVerify: cfg.InsecureTLS,
                ServerName:         host,
            })
            if err := tlsConn.Handshake(); err != nil {
                conn.Close()
                return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
                    Error: "tls: " + err.Error(), FetchTimeMs: ms()}
            }
            conn = tlsConn
        }
    }

    conn.SetDeadline(time.Now().Add(cfg.Timeout))
    path := u.RequestURI()
    reqBytes := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nConnection: keep-alive\r\n\r\n",
        path, host, cfg.UserAgent)

    if _, err := conn.Write([]byte(reqBytes)); err != nil {
        conn.Close()
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: err.Error(), FetchTimeMs: ms()}
    }

    // Read status line only
    buf := make([]byte, 512)
    n, err := conn.Read(buf)
    if err != nil || n < 12 {
        conn.Close()
        return recrawler.Result{URL: seed.URL, Domain: seed.Domain,
            Error: fmt.Sprintf("read: %v (n=%d)", err, n), FetchTimeMs: ms()}
    }

    code := 0
    if len(buf) >= 12 {
        var c int
        fmt.Sscanf(string(buf[9:12]), "%d", &c)
        code = c
    }

    // Drain remaining response before returning conn to pool
    drainConn(conn)
    pool.Put(poolKey, conn)

    return recrawler.Result{
        URL: seed.URL, Domain: seed.Domain, StatusCode: code,
        FetchTimeMs: ms(), CrawledAt: time.Now(),
    }
}

// drainConn reads the full response body from a connection before returning it to the pool.
// For status-only mode, reads up to 64KB to clear the response stream.
func drainConn(conn net.Conn) {
    conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
    buf := make([]byte, 64<<10)
    conn.Read(buf) //nolint:errcheck
    conn.SetDeadline(time.Time{})
}
```

**Step 4: Run tests**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -run TestRawHTTP -v
```
Expected: PASS

**Step 5: Commit**
```bash
git add pkg/recrawl_v3/rawhttp.go pkg/recrawl_v3/rawhttp_pool.go
git commit -m "feat: add recrawl_v3 Engine D — raw HTTP/1.1 with custom net.Conn + connection pool"
```

---

### Task 9: Benchmark harness

**Files:**
- Create: `pkg/recrawl_v3/bench_test.go`

**Step 1: Write benchmark**

```go
// bench_test.go
package recrawl_v3

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-mizu/mizu/blueprints/search/pkg/recrawler"
)

func makeSeeds(srv *httptest.Server, n int) []recrawler.SeedURL {
    seeds := make([]recrawler.SeedURL, n)
    for i := range seeds {
        seeds[i] = recrawler.SeedURL{
            URL:    srv.URL + "/bench/" + fmt.Sprintf("%d", i),
            Domain: "localhost",
            Host:   "localhost",
        }
    }
    return seeds
}

func BenchmarkEngineKeepAlive(b *testing.B) { benchEngine(b, "keepalive") }
func BenchmarkEngineEpoll(b *testing.B)     { benchEngine(b, "epoll") }
func BenchmarkEngineRawHTTP(b *testing.B)   { benchEngine(b, "rawhttp") }

func benchEngine(b *testing.B, name string) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.WriteHeader(200)
    }))
    defer srv.Close()

    seeds := makeSeeds(srv, 1000)
    cfg := DefaultConfig()
    cfg.Workers = 50
    cfg.InsecureTLS = false

    eng, err := New(name)
    if err != nil {
        b.Fatal(err)
    }

    b.ResetTimer()
    b.ReportAllocs()
    for range b.N {
        stats, err := eng.Run(context.Background(), seeds, &noopDNS{}, cfg,
            &noopResultWriter{}, &noopFailureWriter{})
        if err != nil {
            b.Fatal(err)
        }
        b.ReportMetric(stats.AvgRPS, "rps")
    }
}
```

**Step 2: Run benchmarks locally**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go test ./pkg/recrawl_v3/... -bench=. -benchtime=10s -v
```
Expected: all three benchmarks run, print rps metric

**Step 3: Commit**
```bash
git add pkg/recrawl_v3/bench_test.go
git commit -m "test: add engine benchmark harness for recrawl_v3"
```

---

### Task 10: CLI integration — `--engine` flag on `cc recrawl`

**Files:**
- Modify: `cli/cc.go` (add `--engine` flag, route to recrawl_v3)

**Step 1: Add `--engine` flag to `newCCRecrawl()`**

In the var block at ~line 919:
```go
engine string
```

In cmd.Flags() section:
```go
cmd.Flags().StringVar(&engine, "engine", "", "v3 engine: keepalive|epoll|swarm|rawhttp|auto (empty=use v1/v2)")
```

In opts struct:
```go
engine string
```

**Step 2: Add engine dispatch in `runCCRecrawl` before Step 5**

After DNS resolution and before opening FailedDB (around line 1395), add:
```go
// If a v3 engine is requested, delegate to recrawl_v3 and return.
if opts.engine != "" {
    return runCCRecrawlV3(ctx, opts, seeds, dnsResolver, resultDir, failedDBPath)
}
```

**Step 3: Implement `runCCRecrawlV3` helper**

```go
func runCCRecrawlV3(ctx context.Context, opts ccRecrawlOpts, seeds []recrawler.SeedURL,
    dnsResolver *recrawler.DNSResolver, resultDir, failedDBPath string) error {

    fmt.Println(infoStyle.Render(fmt.Sprintf("Using recrawl_v3 engine: %s", opts.engine)))

    engineName := opts.engine
    if engineName == "auto" {
        engineName = "keepalive" // TODO: warmup all and pick best
    }

    eng, err := recrawl_v3.New(engineName)
    if err != nil {
        return fmt.Errorf("engine %q: %w", engineName, err)
    }

    cfg := recrawl_v3.DefaultConfig()
    cfg.Workers = opts.workers
    cfg.Timeout = time.Duration(opts.timeout) * time.Millisecond
    cfg.StatusOnly = opts.statusOnly
    cfg.InsecureTLS = true
    cfg.SearchBinary = os.Executable() // for swarm engine

    var dnsCache recrawl_v3.DNSCache
    if dnsResolver != nil {
        dnsCache = recrawl_v3.WrapDNSResolver(dnsResolver)
    } else {
        dnsCache = &recrawl_v3.NoopDNS{}
    }

    if err := recrawler.OpenFailedDB(failedDBPath); ... // reuse same pattern
    rdb, err := recrawler.NewResultDB(resultDir, 16, opts.batchSize)
    ...

    stats, err := eng.Run(ctx, seeds, dnsCache, cfg,
        &recrawl_v3.ResultDBWriter{DB: rdb},
        &recrawl_v3.FailedDBWriter{DB: failedDB})
    ...
    fmt.Printf("Engine %s: %.0f avg rps, %.0f peak rps, %d ok / %d total\n",
        engineName, stats.AvgRPS, stats.PeakRPS, stats.OK, stats.Total)
    return err
}
```

Also add `ResultDBWriter` and `FailedDBWriter` adapters to `pkg/recrawl_v3/types.go`:
```go
// ResultDBWriter adapts recrawler.ResultDB to ResultWriter.
type ResultDBWriter struct{ DB *recrawler.ResultDB }
func (r *ResultDBWriter) Add(result recrawler.Result)               { r.DB.Add(result) }
func (r *ResultDBWriter) Flush(ctx context.Context) error           { return r.DB.Flush(ctx) }
func (r *ResultDBWriter) Close() error                              { return r.DB.Close() }

// FailedDBWriter adapts recrawler.FailedDB to FailureWriter.
type FailedDBWriter struct{ DB *recrawler.FailedDB }
func (f *FailedDBWriter) AddURL(u recrawler.FailedURL) { f.DB.AddURL(u) }
func (f *FailedDBWriter) Close() error                 { return f.DB.Close() }

// NoopDNS implements DNSCache with no pre-resolved IPs.
type NoopDNS struct{}
func (n *NoopDNS) Lookup(_ string) (string, bool) { return "", false }
func (n *NoopDNS) IsDead(_ string) bool           { return false }
```

**Step 4: Build to verify**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go build ./cmd/search/
```

**Step 5: Commit**
```bash
git add cli/cc.go pkg/recrawl_v3/types.go
git commit -m "feat: add --engine flag to cc recrawl, routing to recrawl_v3 engines"
```

---

### Task 11: Add `cc recrawl-drone` subcommand (Swarm engine support)

**Files:**
- Modify: `cli/cc.go` (add `newCCRecrawlDrone()`)

**Step 1: Add drone subcommand**

```go
func newCCRecrawlDrone() *cobra.Command {
    var droneID int
    cmd := &cobra.Command{
        Use:    "recrawl-drone",
        Hidden: true,
        Short:  "Internal: drone worker for swarm engine",
        RunE: func(cmd *cobra.Command, args []string) error {
            cfg := recrawl_v3.DefaultConfig()
            cfg.Workers = 500
            cfg.StatusOnly = true
            cfg.InsecureTLS = true
            return recrawl_v3.RunDrone(cmd.Context(), cfg)
        },
    }
    cmd.Flags().IntVar(&droneID, "drone-id", 0, "Drone index (used for log prefix)")
    return cmd
}
```

Add to `NewCC()`:
```go
cmd.AddCommand(newCCRecrawlDrone())
```

**Step 2: Build and verify hidden command exists**
```bash
cd blueprints/search && CGO_ENABLED=1 GOWORK=off go build -o /tmp/search-test ./cmd/search/ && /tmp/search-test cc recrawl-drone --help
```

**Step 3: Commit**
```bash
git add cli/cc.go
git commit -m "feat: add hidden cc recrawl-drone subcommand for swarm engine"
```

---

### Task 12: Deploy and benchmark all engines on remote server

**Step 1: Final build and deploy**
```bash
make deploy-linux
make remote-search
```
Expected: help page visible

**Step 2: Benchmark each engine on remote (100K URLs)**
```bash
ssh -i ~/.ssh/id_ed25519_deploy -o BatchMode=yes tam@server \
  'timeout 120 ~/bin/search cc recrawl --file p:0 --status-only --limit 100000 --engine keepalive --workers 1500 2>&1' 2>&1
```
Then repeat for `--engine epoll`, `--engine rawhttp`, `--engine swarm`.
Also run without `--engine` (baseline v1/v2).

Record results in `spec/0613_recrawl_v3.md` Verification Plan table.

**Step 3: Run full no-limit job with best engine**
```bash
make remote-recrawl-bg
make remote-tail
```
Expected: sustained 1000+ pages/s for the full ~2.5M URL run

**Step 4: Commit benchmark results**
```bash
git add spec/0612_recrawl_remote.md spec/0613_recrawl_v3.md
git commit -m "docs: record benchmark results for recrawl v3 engines on doge-01"
```

---

## Summary of All Files Changed/Created

| File | Change |
|------|--------|
| `Makefile` | `ulimit -n 65536` in wrapper; `remote-recrawl`, `remote-recrawl-bg`, `remote-tail` targets |
| `pkg/recrawler/faileddb.go` | `OpenFailedDB()`, `removeIfStaleLocked()`, `parseLockFilePID()`, `processIsAlive()` |
| `pkg/recrawler/faileddb_lock_test.go` | Tests for stale lock detection |
| `cli/cc.go` | `NewFailedDB` → `OpenFailedDB`; `--engine` flag; `runCCRecrawlV3`; `newCCRecrawlDrone` |
| `pkg/recrawl_v3/engine.go` | `Engine` interface, `Stats`, `Config`, `New()` |
| `pkg/recrawl_v3/types.go` | `staticDNSCache`, `dialWithIP`, `parseURL`, adapters |
| `pkg/recrawl_v3/keepalive.go` | Engine A |
| `pkg/recrawl_v3/epoll.go` | Engine B |
| `pkg/recrawl_v3/swarm.go` | Engine C (queen) |
| `pkg/recrawl_v3/swarm_drone.go` | Engine C (drone) |
| `pkg/recrawl_v3/rawhttp.go` | Engine D |
| `pkg/recrawl_v3/rawhttp_pool.go` | Engine D connection pool |
| `pkg/recrawl_v3/keepalive_test.go` | Tests for A, B, D |
| `pkg/recrawl_v3/bench_test.go` | Benchmark harness |
| `spec/0612_recrawl_remote.md` | Remote optimization spec |
| `spec/0613_recrawl_v3.md` | V3 engine spec |
