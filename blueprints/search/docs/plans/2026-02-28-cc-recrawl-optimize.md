# CC Recrawl Optimize Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add error breakdown report to recrawl runs, disable domain-kill for NO FALSE NEGATIVE, raise server2 fd limit for ≥2000 rps, add Makefile targets for p:0/200/400/600 on both servers.

**Architecture:** 4 focused changes: (1) new `FailedURLTopDomains` in store, (2) `printErrorBreakdown` helper in recrawl.go, (3) Makefile fd+targets, (4) cc.go batch-size fix. All are independent; commit after each.

**Tech Stack:** Go 1.22+, DuckDB (database/sql), lipgloss (terminal styling), GNU Make

---

## Task 1: Add `FailedURLTopDomains` to store/failed.go

**Files:**
- Modify: `pkg/crawl/store/failed.go` (after `FailedURLSummary` at line ~503)
- Test: `pkg/crawl/store/failed_test.go`

**Context:** `FailedURLSummary` already exists and uses `sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")`. Follow the same pattern. The `sql` package is already imported.

**Step 1: Write the failing test**

Add to `pkg/crawl/store/failed_test.go` after the existing tests:

```go
func TestFailedDB_TopDomains(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "failed.duckdb")

	fdb, err := OpenFailedDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Add 3 failures for example.com, 1 for foo.net
	for range 3 {
		fdb.AddURL(crawl.FailedURL{URL: "http://example.com/p", Domain: "example.com", Reason: "http_timeout"})
	}
	fdb.AddURL(crawl.FailedURL{URL: "http://foo.net/p", Domain: "foo.net", Reason: "http_error"})
	if err := fdb.Close(); err != nil {
		t.Fatal(err)
	}

	top, err := FailedURLTopDomains(dbPath, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 {
		t.Fatalf("want 2 entries, got %d", len(top))
	}
	if top[0][0] != "example.com" || top[0][1] != "3" {
		t.Errorf("want example.com:3, got %v", top[0])
	}
	if top[1][0] != "foo.net" || top[1][1] != "1" {
		t.Errorf("want foo.net:1, got %v", top[1])
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./pkg/crawl/store/... -run TestFailedDB_TopDomains -v
```

Expected: FAIL with "undefined: FailedURLTopDomains"

**Step 3: Implement `FailedURLTopDomains`**

Add after `FailedURLSummary` in `pkg/crawl/store/failed.go`:

```go
// FailedURLTopDomains returns the top N domains by total failure count.
// Each entry is [domain, count_string] sorted by count descending.
// Opens the DB read-only; returns nil, nil when dbPath is empty.
func FailedURLTopDomains(dbPath string, n int) ([][2]string, error) {
	if dbPath == "" {
		return nil, nil
	}
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(
		`SELECT domain, COUNT(*) AS c FROM failed_urls
		 GROUP BY domain ORDER BY c DESC LIMIT ?`, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result [][2]string
	for rows.Next() {
		var domain string
		var count int
		rows.Scan(&domain, &count) //nolint:errcheck
		result = append(result, [2]string{domain, fmt.Sprintf("%d", count)})
	}
	return result, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./pkg/crawl/store/... -run TestFailedDB_TopDomains -v
```

Expected: PASS

**Step 5: Build check**

```bash
go build ./...
```

Expected: clean, no errors

**Step 6: Commit**

```bash
git add pkg/crawl/store/failed.go pkg/crawl/store/failed_test.go
git commit -m "feat(store): add FailedURLTopDomains for error breakdown report"
```

---

## Task 2: Add error breakdown report to cli/recrawl.go

**Files:**
- Modify: `cli/recrawl.go` (after line ~208, before `return err`)

**Context:**
- `sort` is already imported (line 10)
- `store` is already imported (line 17): `"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/store"`
- `labelStyle` is defined in `cli/ui.go`, accessible in same package
- `ccFmtInt64` is a function in `cli/recrawl.go` (used at line 194)
- `args.FailedDBPath` holds the path (empty string if no failed DB)
- Must NOT panic if DB is empty or doesn't exist (first run may have zero failures)

**Step 1: Add `printErrorBreakdown` function**

Find the line `return err` at line ~210 of `cli/recrawl.go` (the last line of `runRecrawlJob`). Insert before `return err`:

```go
	if args.FailedDBPath != "" {
		printErrorBreakdown(args.FailedDBPath)
	}
```

Then add a new private function after `runRecrawlJob`'s closing `}` and before the `// ── Display types` comment:

```go
// printErrorBreakdown prints a reason breakdown and top failing domains from the failed DB.
// Silent no-op if the DB is empty, missing, or unreadable.
func printErrorBreakdown(dbPath string) {
	urlSummary, urlTotal, err := store.FailedURLSummary(dbPath)
	if err != nil || urlTotal == 0 {
		return
	}

	type entry struct {
		reason string
		count  int
	}
	entries := make([]entry, 0, len(urlSummary))
	for r, c := range urlSummary {
		entries = append(entries, entry{r, c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].count > entries[j].count })

	fmt.Println(labelStyle.Render("Error breakdown:"))
	for _, e := range entries {
		pct := 100.0 * float64(e.count) / float64(urlTotal)
		fmt.Printf("  %-40s %s  (%5.1f%%)\n", e.reason, ccFmtInt64(int64(e.count)), pct)
	}

	topDomains, err := store.FailedURLTopDomains(dbPath, 10)
	if err != nil || len(topDomains) == 0 {
		return
	}
	fmt.Println(labelStyle.Render("Top failing domains:"))
	for _, pair := range topDomains {
		fmt.Printf("  %-45s %s\n", pair[0], pair[1])
	}
}
```

**Step 2: Verify build**

```bash
go build ./cli/...
```

Expected: clean

**Step 3: Quick integration test**

```bash
go test ./cli/... -run TestRecrawl -v 2>&1 | head -20
```

(No specific test to write — function is display-only. Build success is the gate.)

**Step 4: Commit**

```bash
git add cli/recrawl.go
git commit -m "feat(recrawl): print error breakdown + top failing domains after run"
```

---

## Task 3: Fix batch-size default in cli/cc.go

**Files:**
- Modify: `cli/cc.go` (one line change)

**Context:** The `--batch-size` flag defaults to `10`, causing 250K DuckDB INSERTs for a 2.5M-URL run. `hn recrawl` uses `100`. The `DefaultConfig().BatchSize` is `5000` in `pkg/crawl/engine.go`. Use `5000` to match store behavior and reduce overhead.

**Step 1: Find and change the line**

In `cli/cc.go`, find:
```go
cmd.Flags().IntVar(&batchSize, "batch-size", 10, "DB write batch size")
```

Change to:
```go
cmd.Flags().IntVar(&batchSize, "batch-size", 5000, "DB write batch size")
```

**Step 2: Build check**

```bash
go build ./cli/...
```

Expected: clean

**Step 3: Commit**

```bash
git add cli/cc.go
git commit -m "fix(cc): raise default batch-size from 10 to 5000 for full-parquet runs"
```

---

## Task 4: Raise server2 fd limit in deploy-linux-noble Makefile target

**Files:**
- Modify: `Makefile` (deploy-linux-noble wrapper script, line ~171)

**Context:** Server2 runs as root, so `ulimit -Hn 131072` can raise the hard limit. Server1 runs as non-root, so the hard limit set will fail silently; it falls through to `ulimit -n 65536`.

The wrapper script is a `printf` one-liner embedded in the Makefile. The current `ulimit` line is:
```
ulimit -n 65536 2>/dev/null || true
```

Change to (try hard+soft 131072, fall back to 65536):
```
ulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true
```

**Step 1: Find the exact line in Makefile**

Look at line ~171 of `Makefile`. The `deploy-linux-noble` target has a `printf` with the wrapper script. The relevant part within the string is:

```
ulimit -n 65536 2>/dev/null || true
```

**Step 2: Replace only in deploy-linux-noble (not deploy-linux)**

Only change the `deploy-linux-noble` target (server2 wrapper). The `deploy-linux` target (server1 wrapper) stays at `ulimit -n 65536`.

In the noble printf string, replace:
```
ulimit -n 65536 2>/dev/null || true
```
with:
```
ulimit -Hn 131072 2>/dev/null; ulimit -n 131072 2>/dev/null || ulimit -n 65536 2>/dev/null || true
```

**Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(deploy): raise server2 fd limit to 131072 for higher worker count"
```

---

## Task 5: Add Makefile targets for cc recrawl p:0/200/400/600

**Files:**
- Modify: `Makefile` (after the existing `remote-recrawl-bg` target at line ~209)

**Context:**
- Each parquet file writes to its own directory, so concurrent runs don't conflict
- `CC_RECRAWL_FLAGS` variable allows overriding per-invocation
- Server1 (5GB RAM): run one at a time. Server2 (12GB): all 4 in parallel via `remote-cc-recrawl-all`
- Log files: `~/cc-p0.log`, `~/cc-p200.log`, `~/cc-p400.log`, `~/cc-p600.log`
- `domain-fail-threshold 0` = NO FALSE NEGATIVE (no domain-kill)
- `domain-timeout 0` = disable per-domain context deadline (no abandonment)
- `batch-size 5000` = efficient DB writes (overrides the flag default which is now also 5000, but explicit is clearer)

**Step 1: Insert the new targets in Makefile**

After the existing `.PHONY: remote-tail` block (around line 211-213), insert:

```makefile
# ── CC Recrawl full-parquet runs (background, per file) ──────────────────────

# Flags applied to all cc recrawl background runs.
# domain-fail-threshold 0 = NO FALSE NEGATIVE (no domain-kill).
# domain-timeout 0        = disable per-domain context deadline.
# batch-size 5000         = efficient DuckDB writes.
CC_RECRAWL_FLAGS ?= --domain-fail-threshold 0 --domain-timeout 0 --batch-size 5000

.PHONY: remote-cc-recrawl-p0
remote-cc-recrawl-p0: ## CC recrawl p:0 in background, log ~/cc-p0.log (SERVER=1|2)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:0 $(CC_RECRAWL_FLAGS) >~/cc-p0.log 2>&1 & echo PID:$$!"'

.PHONY: remote-cc-recrawl-p200
remote-cc-recrawl-p200: ## CC recrawl p:200 in background, log ~/cc-p200.log (SERVER=1|2)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:200 $(CC_RECRAWL_FLAGS) >~/cc-p200.log 2>&1 & echo PID:$$!"'

.PHONY: remote-cc-recrawl-p400
remote-cc-recrawl-p400: ## CC recrawl p:400 in background, log ~/cc-p400.log (SERVER=1|2)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:400 $(CC_RECRAWL_FLAGS) >~/cc-p400.log 2>&1 & echo PID:$$!"'

.PHONY: remote-cc-recrawl-p600
remote-cc-recrawl-p600: ## CC recrawl p:600 in background, log ~/cc-p600.log (SERVER=1|2)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "nohup ~/bin/search cc recrawl --file p:600 $(CC_RECRAWL_FLAGS) >~/cc-p600.log 2>&1 & echo PID:$$!"'

.PHONY: remote-cc-recrawl-all
remote-cc-recrawl-all: remote-cc-recrawl-p0 remote-cc-recrawl-p200 remote-cc-recrawl-p400 remote-cc-recrawl-p600 ## CC recrawl all 4 parquet files in parallel background (SERVER=2 only — needs 12GB RAM)

.PHONY: remote-cc-tail-p0
remote-cc-tail-p0: ## Tail ~/cc-p0.log on remote (SERVER=1|2)
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p0.log'

.PHONY: remote-cc-tail-p200
remote-cc-tail-p200: ## Tail ~/cc-p200.log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p200.log'

.PHONY: remote-cc-tail-p400
remote-cc-tail-p400: ## Tail ~/cc-p400.log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p400.log'

.PHONY: remote-cc-tail-p600
remote-cc-tail-p600: ## Tail ~/cc-p600.log on remote
	@$(SSH) $(REMOTE_SSH) 'tail -f ~/cc-p600.log'

.PHONY: remote-cc-status
remote-cc-status: ## Show running cc recrawl PIDs and last log line for each file (SERVER=1|2)
	@$(SSH) $(REMOTE_SSH) 'bash -lc "pgrep -a search 2>/dev/null | grep recrawl || echo \"(no recrawl running)\"; for f in ~/cc-p0.log ~/cc-p200.log ~/cc-p400.log ~/cc-p600.log; do [ -f \"$$f\" ] && echo \"--- $$f:\" && tail -2 \"$$f\"; done"'
```

**Step 2: Verify Makefile syntax**

```bash
make -n remote-cc-recrawl-p0 SERVER=2 2>&1 | head -5
make -n remote-cc-recrawl-all SERVER=2 2>&1 | head -20
```

Expected: shows SSH commands without executing them

**Step 3: Commit**

```bash
git add Makefile
git commit -m "feat(makefile): add cc recrawl targets for p:0/200/400/600 + fd limit raise"
```

---

## Task 6: Build, deploy, and launch cc recrawl runs

**Step 1: Build both binaries**

```bash
make build-linux 2>&1 | tail -3
make build-linux-noble 2>&1 | tail -3
```

Expected:
```
Built Linux binary: ~/bin/search-linux (v0.5.24-...)
Built Noble binary: ~/bin/search-linux-noble (v0.5.24-...)
```

**Step 2: Deploy to both servers**

```bash
make deploy-linux SERVER=1 2>&1 | tail -2
make deploy-linux-noble SERVER=2 2>&1 | tail -2
```

**Step 3: Verify fd limit on server2**

```bash
ssh -i ~/.ssh/id_ed25519_deploy root@server2 'bash -lc "ulimit -n"'
```

Expected: `131072`

**Step 4: Smoke test error report on server1**

```bash
ssh -i ~/.ssh/id_ed25519_deploy tam@server \
  'bash -lc "~/bin/search hn recrawl --limit 200 --status-only --no-retry 2>&1 | grep -A20 \"Engine\|Error break\|Top fail\""'
```

Expected: shows "Error breakdown:" section with reason counts, "Top failing domains:" with domain list.

**Step 5: Launch cc recrawl p:0 on server1 (foreground, verify first)**

```bash
ssh -i ~/.ssh/id_ed25519_deploy tam@server \
  'bash -lc "~/bin/search cc recrawl --file p:0 --domain-fail-threshold 0 --domain-timeout 0 --batch-size 5000 --status-only --no-retry --limit 2000 2>&1 | tail -20"'
```

Expected: runs ~2000 seeds, shows error breakdown at end, EXIT:0.

**Step 6: Launch all 4 files in background on server2**

```bash
make remote-cc-recrawl-all SERVER=2
```

Expected: 4 PIDs printed, processes running in background.

**Step 7: Launch files sequentially on server1 (p:0 first)**

```bash
make remote-cc-recrawl-p0 SERVER=1
```

(Wait for completion before starting p:200. Monitor with `make remote-cc-tail-p0 SERVER=1`.)

**Step 8: Monitor server2 status**

```bash
make remote-cc-status SERVER=2
```

Expected: shows 4 running processes and last progress lines from each log.

**Step 9: Update MEMORY.md with lessons learned**

After first server2 run completes, check actual throughput from log:

```bash
make remote-cc-tail-p0 SERVER=2
```

Look for "Engine keepalive done: ... avg N rps | peak N rps" and record in MEMORY.md.
