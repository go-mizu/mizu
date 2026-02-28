# spec/0627: CC Recrawl — Full Run p:0/200/400/600 + Error Report + Server2 Optimization

**Status:** implementation

**Goal:** Run `cc recrawl` for parquet files p:0, p:200, p:400, p:600 on both servers;
add error breakdown + top failing domains to the end-of-run report;
eliminate false negatives (domain-kill disabled); optimize server2 to ≥2000 req/s.

---

## Problems with current state

1. **False negatives** — `--domain-fail-threshold -1` (engine default 3) abandons domains
   after 3 timeout rounds, silently dropping URLs that were never attempted.
   `--domain-timeout 30000` (30s) cancels remaining URLs per domain after 30s.
   Both must be `0` for a correct run.

2. **No error breakdown** — end-of-run report shows only ok/total counts.
   Operators have no visibility into why URLs failed (timeout vs. hard error vs. domain-dead).

3. **Server2 fd cap** — `ulimit -n 65536` limits workers to 8192 (fd÷8).
   Server2 (root) can raise this to 131072 → auto-config picks workers=16384 (status-only).

4. **No Makefile targets** for p:0/200/400/600 background runs.

5. **Batch size = 10** in cc recrawl — very small; 250K DuckDB INSERTs for 2.5M URLs.
   Should be 5000 (matches hn recrawl intent).

---

## Changes

### 1. `pkg/crawl/store/failed.go` — add `FailedURLTopDomains`

```go
// FailedURLTopDomains returns the top N domains by total failure count.
// Each entry is (domain, count). Opens DB read-only.
func FailedURLTopDomains(dbPath string, n int) ([][2]string, error)
```

SQL: `SELECT domain, COUNT(*) AS c FROM failed_urls GROUP BY domain ORDER BY c DESC LIMIT ?`

Returns `[][2]string` — each element is `{domain, count_string}` for simple formatting.

### 2. `cli/recrawl.go` — print error breakdown after pass summaries

After the existing Pass 2 summary block, if `args.FailedDBPath != ""`:

```
Error breakdown (failed URLs):
  http_timeout               12,345  ( 45.2%)
  http_error                  8,901  ( 32.6%)
  domain_dead                 4,567  ( 16.7%)
  domain_deadline_exceeded    1,234  (  4.5%)
  domain_http_timeout_killed     98  (  0.4%)

Top failing domains:
  slow-example.com           1,234  timeouts
  broken.net                   567  errors
  (top 10)
```

- Call `store.FailedURLSummary` (already exists) for reason breakdown
- Call `store.FailedURLTopDomains` (new) for top 10 failing domains
- Print even when err != nil (skip gracefully)
- Do NOT print if failedDBPath is empty (devnull writer mode)

### 3. `Makefile` — raise server2 fd limit

In `deploy-linux-noble`, update the wrapper script written to `~/bin/search`:

```bash
ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true
```

(Try 131072 first; root on server2 can set above hard limit with `ulimit -Hn`.)

### 4. `Makefile` — new cc recrawl targets for p:0/200/400/600

```makefile
CC_RECRAWL_FLAGS ?= --domain-fail-threshold 0 --domain-timeout 0 --batch-size 5000

remote-cc-recrawl-p0:
    @$(SSH) $(REMOTE_SSH) 'nohup ... --file p:0 $(CC_RECRAWL_FLAGS) >~/cc-p0.log 2>&1 & echo PID:$$!'

remote-cc-recrawl-p200: ...
remote-cc-recrawl-p400: ...
remote-cc-recrawl-p600: ...

remote-cc-recrawl-all: remote-cc-recrawl-p0 remote-cc-recrawl-p200 \
                        remote-cc-recrawl-p400 remote-cc-recrawl-p600

remote-cc-tail-p0:
    @$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p0.log'
# ...
```

Runs are launched in background (nohup). All 4 start simultaneously — they write to
separate directories so there's no DuckDB conflict. Each uses ~3GB RAM, 4 together
use ~12GB (fits server2 12GB exactly; server1 5GB needs sequential runs).

Note: For server1 (5GB RAM), run sequentially — the Makefile targets still work,
just invoke one at a time.

### 5. `cli/cc.go` — fix batch-size default

Change `--batch-size` default from `10` → `5000` (consistent with hn recrawl intent
and store/result.go shard buffer).

---

## Expected throughput after optimization

**Server2 (12GB, 8 CPUs, noble, root):**
- fd limit: 131072 → workers=16384 (status-only, fd-capped: 131072÷8)
- Full body: memory-capped at ~9600 workers
- Previous benchmark: peak 10,832 rps (pass 1, 200K seeds, clean DB, workers=8192)
- With workers=16384: estimated peak 15–20K rps, avg ≥2000 rps sustained

**Server1 (5GB, 4 CPUs, focal):**
- fd limit: 65536 (unchanged; wrapper stays as-is for focal)
- workers=8192 (fd-capped, same as before)
- Run p:0/200/400/600 sequentially (not enough RAM for parallel)

---

## NOT changing

- Engine: keepalive (default, best for breadth-first CC data)
- Pass 2: enabled by default (`--retry-timeout 10000`), unless `--no-retry` used
- Server1 fd limit (not root, can't raise above hard limit)
