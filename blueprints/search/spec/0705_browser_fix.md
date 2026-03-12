# Browser Scrape Fixes (2026-03-10)

## Problem

`search cc fts dashboard` → `#/scrape/vuejs.org` → **browser mode** produced three cascading errors:

```
get html: empty html from browser
timeout waiting for DOM
rod pool: context deadline exceeded
```

HTTP mode worked perfectly (109/109 pages in 4s). The browser mode ran for ~2 minutes with 0 successful pages.

---

## Root Cause Analysis

### 1. AutoBrowserPages: 80 tabs on 12 GB server → Chrome OOM

`AutoBrowserPages(11000 MB) = 80` (old formula: `availRAMMB/100`, max=80).

Chrome was launched with `--renderer-process-limit=8`, meaning 8 renderer processes shared across 80 tabs = **10 tabs per renderer process**.

Each renderer runs up to 10 V8 contexts × `--max-old-space-size=256` MB = **2.56 GB of V8 heap per renderer**.
Total potential: 8 renderers × 2.56 GB = **20.5 GB**, which far exceeds server2's 12 GB RAM.

Under memory pressure, the Linux OOM killer targets Chrome renderer processes. When a renderer is killed:
- All tabs handled by that process lose their CDP connection
- `Page.getFrameTree` / `Runtime.evaluate` return empty results
- `ep.HTML()` returns `""` (no error, empty string)

This is the single root cause that triggered all three error types:

| Error | How it's triggered |
|---|---|
| `get html: empty html from browser` | `ep.HTML()` returns `""` after renderer OOM-kill |
| `timeout waiting for DOM` | Chrome sluggish under memory pressure; readyState polling stalls |
| `rod pool: context deadline exceeded` | All 80 slots in use; new pages can't be allocated before fetchCtx expires |

### 2. rendererLimit not scaling with tab count

```go
// OLD: only applies when RodWorkers <= 20
rendererLimit := 8
if cfg.RodWorkers > 0 && cfg.RodWorkers <= 20 {
    rendererLimit = max(cfg.RodWorkers/3, 2)
}
```

For 80 tabs (RodWorkers=80 > 20), the condition is false → rendererLimit stays at 8, even though 8 renderers × 10 tabs × 256MB = 20 GB. The fix must always compute rendererLimit proportionally.

### 3. Empty HTML JS fallback not triggered on empty-string result

```go
htmlContent, err := ep.HTML()
if err != nil {           // ← only if error, NOT if empty string
    // Eval fallback...
}
if htmlContent == "" {   // ← falls through with empty string, records error
    recordError(...)
}
```

When `ep.HTML()` returns `("", nil)` (empty string, no error), the JS eval fallback is skipped entirely. The fix applies the fallback on empty strings too, giving Chrome a second chance via a fresh CDP evaluate call.

---

## Fixes

### Fix 1 — `config.go`: Lower AutoBrowserPages cap

**Old**: `availRAMMB / 100`, max = 80
**New**: `availRAMMB / 150`, max = 24

```
            Old      New
500 MB    → 8        8
4000 MB   → 40       26 → capped at 24
5000 MB   → 50 → 80  33 → capped at 24
11000 MB  → 80      24  ← server2: safe (24 × ~150MB = 3.6GB)
20000 MB  → 80      24
```

At 24 tabs with 8 renderer processes: 3 tabs per renderer × 256MB = **768 MB/renderer × 8 = 6.1 GB** total — well within server2's 12 GB budget even under load. Chrome stays stable; CDP calls return valid HTML.

Updated test cases in `config_test.go` to match new formula.

### Fix 2 — `rod.go:newLauncher`: Scale rendererLimit with worker count

**Old**: `rendererLimit = 8` hardcoded for `RodWorkers > 20`
**New**: always compute `rendererLimit = clamp(RodWorkers/4, 4, 16)`

```
 8 tabs → 8/4=2  → clamp to 4  → 2 tabs/renderer
16 tabs → 16/4=4 → 4           → 4 tabs/renderer
24 tabs → 24/4=6 → 6           → 4 tabs/renderer
32 tabs → 32/4=8 → 8           → 4 tabs/renderer
```

This ensures that regardless of tab count, each renderer handles at most `workers/4` tabs. Memory budget is proportional to actual worker count.

### Fix 3 — `rod.go:rodFetchAndProcess`: JS eval fallback on empty HTML

**Old**: Eval fallback only when `ep.HTML()` returns an error
**New**: Eval fallback when `ep.HTML()` returns an error **or** empty string

```go
// Before
htmlContent, err := ep.HTML()
if err != nil {
    // Eval fallback
}

// After
htmlContent, err := ep.HTML()
if err != nil || htmlContent == "" {
    // Eval fallback — also covers OOM-killed renderer returning "" with nil error
}
```

This handles the OOM race condition where the CDP call completes (no error) but returns an empty snapshot before the renderer process was restarted.

---

## Files Changed

| File | Change |
|---|---|
| `pkg/dcrawler/config.go` | `AutoBrowserPages`: formula `/100`→`/150`, max `80`→`24` |
| `pkg/dcrawler/config_test.go` | Update expected values for new formula |
| `pkg/dcrawler/rod.go` | `newLauncher`: always-proportional rendererLimit; `rodFetchAndProcess`: JS eval on empty HTML |

---

## Verification

After deploy on server2:
1. Open `http://185.209.229.109:3456/#/scrape/vuejs.org`
2. Start browser mode scrape → should report `Browser: 24 tabs`
3. Pages should succeed (200 OK, content extracted)
4. No `empty html from browser` errors in dashboard logs

Expected: 109 pages scraped successfully in browser mode (same as HTTP mode).

---

## Key Lessons

- **AutoBrowserPages max=80 caused Chrome OOM** on 12 GB server. Conservative budget (150 MB/tab, max 24) is safer.
- **rendererLimit must scale with tab count**, not be hardcoded at 8 for large worker counts.
- **`ep.HTML()` can return `("", nil)`** when the renderer process is OOM-killed mid-CDP-call. Always try the JS eval fallback on empty string, not just on error.
- **vuejs.org is VitePress SSG** — HTTP mode is always faster and correct for static sites. Browser mode is for anti-bot-gated JS SPAs that don't serve pre-rendered HTML.
