# CC Recrawl Body Store Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire up the body CAS store in `search cc recrawl` so it matches the Rust `crawler cc recrawl` behavior (default store `~/data/common-crawl/bodies`, `--no-body-store` to disable).

**Architecture:** Three-file change: (1) add `BodyStore` field to `crawl.JobConfig` and wire it into the engine `Config` in `RunJob`; (2) replace the stub in `cli/recrawl.go` with real `bodystore.Open()` wiring; (3) add `--body-store`/`--no-body-store` flags to `cc recrawl` and thread the dir through `ccRecrawlOpts` → `runCCRecrawlV3` → `runRecrawlJob`.

**Tech Stack:** Go, `pkg/crawl/bodystore` (existing), Cobra flags

---

### Task 1: Add `BodyStore` to `crawl.JobConfig` and wire it in `RunJob`

**Files:**
- Modify: `pkg/crawl/job.go`

**Step 1: Add field to JobConfig struct**

In `job.go` around line 46 (after `Pass2Workers`), add:

```go
// BodyStore is optional. When non-nil, HTML bodies are stored in a CAS store
// and Result.BodyCID is populated; Result.Body is left empty.
BodyStore interface {
    Put(body []byte) (cid string, err error)
}
```

**Step 2: Wire it in RunJob**

After `engCfg.Notifier = cfg.Notifier` (around line 133), add:

```go
if cfg.BodyStore != nil {
    engCfg.BodyStore = cfg.BodyStore
}
```

**Step 3: Build to verify**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./pkg/crawl/...
```
Expected: no errors

**Step 4: Commit**

```bash
git add pkg/crawl/job.go
git commit -m "crawl: add BodyStore field to JobConfig, wire into engine Config"
```

---

### Task 2: Implement body store wiring in `cli/recrawl.go`

**Files:**
- Modify: `cli/recrawl.go`

**Step 1: Add import for bodystore**

In the import block, add:
```go
"github.com/go-mizu/mizu/blueprints/search/pkg/crawl/bodystore"
```

Also add `"os/user"` or use `os.UserHomeDir()` for home dir expansion.

**Step 2: Replace the stub**

Find (lines 99-102):
```go
if args.BodyStoreDir != "" {
    // body store setup if needed
    _ = args.BodyStoreDir
}
```

Replace with:
```go
if args.BodyStoreDir != "" {
    dir := args.BodyStoreDir
    if strings.HasPrefix(dir, "~/") {
        home, err := os.UserHomeDir()
        if err != nil {
            return fmt.Errorf("resolving home dir for body store: %w", err)
        }
        dir = filepath.Join(home, dir[2:])
    }
    bs, err := bodystore.Open(dir)
    if err != nil {
        return fmt.Errorf("opening body store %s: %w", dir, err)
    }
    args.JobCfg.BodyStore = bs
    fmt.Printf("Body store: %s\n", dir)
}
```

Note: `strings`, `os`, `filepath` are already imported. `bodystore` needs to be added.

**Step 3: Build to verify**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./cli/...
```
Expected: no errors

**Step 4: Commit**

```bash
git add cli/recrawl.go
git commit -m "cli: wire up body store in runRecrawlJob (was stub)"
```

---

### Task 3: Add `--body-store` / `--no-body-store` flags to `cc recrawl`

**Files:**
- Modify: `cli/cc.go`

**Step 1: Add vars to `newCCRecrawl()`**

In the `var (...)` block (around line 847), add:
```go
bodyStoreDir string
noBodyStore  bool
```

**Step 2: Add flags**

After existing flags in `newCCRecrawl()`, add:
```go
cmd.Flags().StringVar(&bodyStoreDir, "body-store", "~/data/common-crawl/bodies", "Body CAS store directory; HTML bodies stored as sha256:{hex}.gz (compatible with Rust crawler)")
cmd.Flags().BoolVar(&noBodyStore, "no-body-store", false, "Disable body CAS store (skip saving HTML bodies)")
```

**Step 3: Add field to `ccRecrawlOpts` struct**

In `ccRecrawlOpts` struct (around line 1012), add:
```go
bodyStoreDir string
```

**Step 4: Pass it in RunE**

In the `RunE` closure, compute the effective dir and pass it to `runCCRecrawl`:
```go
effectiveBodyStore := bodyStoreDir
if noBodyStore || opts.statusOnly || opts.headOnly {
    effectiveBodyStore = ""
}
```
Then add to `ccRecrawlOpts{...}`:
```go
bodyStoreDir: effectiveBodyStore,
```

Note: disable body store automatically for `--status-only` / `--head-only` since those modes don't fetch bodies.

**Step 5: Pass through `runCCRecrawlV3`**

In `runCCRecrawlV3` (around line 1460), add `BodyStoreDir` to `runRecrawlJob` call:
```go
return runRecrawlJob(ctx, recrawlJobArgs{
    ...
    BodyStoreDir: opts.bodyStoreDir,
    ...
})
```

**Step 6: Build to verify**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
go build ./...
```
Expected: no errors

**Step 7: Commit**

```bash
git add cli/cc.go
git commit -m "cli: add --body-store/--no-body-store to cc recrawl (matches Rust crawler defaults)"
```

---

### Task 4: Build Linux binary and deploy to server 2

**Step 1: Build noble Linux binary**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search
make build-linux-noble
```
Expected: binary at `build/search-linux-noble` (or similar path — check Makefile)

**Step 2: Deploy to server 2**

```bash
make deploy-linux SERVER=2
```

**Step 3: Verify on server 2**

```bash
ssh server2 "search cc recrawl --file p:0 --limit 100 --no-retry"
```

Expected: see `Body store: ~/data/common-crawl/bodies` in output, crawl runs successfully.

---

### Task 5: Update MEMORY.md

Document the body store wiring pattern for future reference
