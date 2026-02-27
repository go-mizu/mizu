# Spec 0612: Remote Server Optimization — `cc recrawl --file p:0`

## Status
- **Target:** `search cc recrawl --file p:0 --status-only` reaches 1000 pages/s, never crashes.
- **Baseline:** 400–427 pages/s with `--workers 500`, crashes on DuckDB lock conflict.
- **Server:** `doge-01` — 4-core AMD EPYC, 5.8 GB total (1.5 GB available), Ubuntu 20.04 kernel 5.4.

---

## Remote Server Profile (measured 2026-02-26)

| Parameter | Value | Notes |
|-----------|-------|-------|
| CPUs | 4 cores AMD EPYC | `nproc` |
| RAM total | 5.8 GB | `free -h` |
| RAM available | 1.5 GB | active Kubernetes cluster consuming memory |
| Swap | 0 B | none configured |
| `ulimit -n` | **1024** | **critical bottleneck** |
| `ip_local_port_range` | 32768–60999 | ~28 K ephemeral ports |
| `tcp_tw_reuse` | 2 (enabled) | TIME_WAIT sockets can be reused |
| `tcp_fin_timeout` | 60 s | long — affects port recycling |
| Ping 8.8.8.8 | 1.9 ms avg | datacenter network, low latency |
| Disk free | 192 GB | `/dev/sda3` |
| OS | Ubuntu 20.04 | kernel 5.4.0-105-generic |

---

## Root Cause Analysis

### 1. DuckDB exclusive-lock crash (immediate crash)

When a previous `search-linux` process (PID e.g. 1267518) is killed mid-run or times out, it
leaves an exclusive write-lock on `failed.duckdb`. The next run attempts `sql.Open("duckdb",
path)` which fails immediately with:

```
IO Error: Could not set lock on file "…/failed.duckdb": Conflicting lock is held in
/home/tam/bin/search-linux (PID 1267518) by user tam.
```

**Fix:** Before opening `failed.duckdb` and each result shard, detect stale locks held by
dead PIDs and remove the file. A dead PID is one where `/proc/<pid>/exe` no longer resolves
or `kill(pid, 0)` returns ESRCH.

### 2. fd limit = 1024 caps worker count (performance ceiling)

Each HTTP worker holds 1 socket fd while fetching. Additional fds consumed:

| Consumer | Count |
|----------|-------|
| DuckDB result shards (16) | 16 |
| DuckDB failed.duckdb | 1 |
| DuckDB dns.duckdb | 1 |
| stdin / stdout / stderr | 3 |
| Go runtime misc | ~15 |
| **Available for sockets** | **~988** |

With 1133 ms avg latency (status-only), achieving 1000 pages/s requires:

```
workers = 1000 req/s × 1.133 s/req ≈ 1133 concurrent workers
```

That exceeds the 988-fd socket budget → peak is capped at ~870 pages/s.

**Fix:** Add `ulimit -n 65536` to the `~/bin/search` wrapper script written by `deploy-linux`.

### 3. No `remote-recrawl` Makefile convenience target

Operators must SSH manually, remember the right flags, and manage background execution.

**Fix:** Add `remote-recrawl` and `remote-recrawl-bg` targets.

---

## Changes

### A. `Makefile` — deploy-linux wrapper adds `ulimit -n 65536`

In `deploy-linux`, the wrapper line:
```bash
printf '#!/usr/bin/env bash\nexport LD_LIBRARY_PATH=...\nexec ...\n'
```
becomes:
```bash
printf '#!/usr/bin/env bash\nulimit -n 65536 2>/dev/null || true\nexport LD_LIBRARY_PATH=...\nexec ...\n'
```

### B. `Makefile` — new `remote-recrawl` targets

```makefile
.PHONY: remote-recrawl
remote-recrawl: ## Run cc recrawl --file p:0 --status-only on remote (foreground)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "~/bin/search cc recrawl --file p:0 --status-only --workers 1500"'

.PHONY: remote-recrawl-bg
remote-recrawl-bg: ## Run cc recrawl --file p:0 --status-only on remote (background, log ~/recrawl.log)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:0 --status-only --workers 1500 >~/recrawl.log 2>&1 &; echo PID:$$!"'

.PHONY: remote-tail
remote-tail: ## Tail recrawl log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/recrawl.log'
```

### C. `pkg/recrawler/faileddb.go` — stale-lock detection before open

New exported function `OpenFailedDB(path string) (*FailedDB, error)` wraps `NewFailedDB`
with stale-lock cleanup:

```go
func OpenFailedDB(path string) (*FailedDB, error) {
    if err := removeIfStaleLocked(path); err != nil {
        return nil, err
    }
    return NewFailedDB(path)
}

// removeIfStaleLocked checks whether path is locked by a dead process.
// On Linux, DuckDB writes a .lock file alongside the database.
// If the lock file exists and the recorded PID is dead, remove both.
func removeIfStaleLocked(dbPath string) error {
    lockPath := dbPath + ".lock"
    data, err := os.ReadFile(lockPath)
    if errors.Is(err, os.ErrNotExist) {
        return nil // no lock file → clean
    }
    if err != nil {
        return nil // can't read → let DuckDB handle it
    }
    pid, err := parseLockPID(data)
    if err != nil || pid <= 0 {
        return nil // can't parse → let DuckDB handle it
    }
    if isProcessAlive(pid) {
        return nil // genuinely locked by live process
    }
    // dead PID → stale lock, remove
    os.Remove(lockPath)
    os.Remove(dbPath)
    return nil
}
```

### D. `cli/cc.go` — use `recrawler.OpenFailedDB` instead of `recrawler.NewFailedDB`

Replace the single call site at line ~1403.

Also use `recrawler.OpenResultDB` (same pattern) for result shards.

---

## Expected Result

After these changes:

| Metric | Before | After |
|--------|--------|-------|
| Crash on retry | Yes (DuckDB lock) | No (stale lock cleaned) |
| Max workers | ~900 (fd=1024) | ~3000 (fd=65536) |
| Peak pages/s (`--workers 1500`) | 427 | ~1000+ |
| Operator UX | Manual SSH + flags | `make remote-recrawl` |

---

## Verification

```bash
# 1. Deploy updated binary + wrapper
make deploy-linux

# 2. Verify ulimit in wrapper
make remote-search  # should show "ulimit -n 65536" effect indirectly
ssh -i ~/.ssh/id_ed25519_deploy tam@server 'bash -c "source ~/bin/search; ulimit -n"'
# Expected: 65536

# 3. Run full recrawl (no limit = full 2.5M URL parquet)
make remote-recrawl
# Expected: Peak >= 1000 pages/s after warmup phase

# 4. Kill and re-run (stale lock test)
ssh tam@server 'pkill -f search-linux; sleep 2; ~/bin/search cc recrawl --file p:0 --status-only --workers 1500 --limit 1000'
# Expected: runs without "Conflicting lock" error
```
