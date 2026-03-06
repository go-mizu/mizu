# Spec 0679: Consistent WARC & Browse UI

## Problem Statement

Multiple UI inconsistencies across WARC, Browse, and Jobs pages degrade DX:

### WARC List Page (`#/warc`)
1. **Total shows 1000 instead of 100K** — metastore tracks ~1000 known records but manifest has 100,000 WARCs. Top stats pane should show manifest total prominently.
2. **Per-page selector is noisy** — remove `100/page` dropdown, use fixed page size.
3. **"Run next step" batch button is confusing** — remove it; users should use individual actions or the Jobs page for batch operations.
4. **WARC 00000/00001 state wrong** — `enrichWARCAPIRecord` checks disk for `.md.warc.gz` file but the metastore summary (`markdown_ready`) uses different criteria. The 3-phase pipeline display should be consistent with what the detail API reports.

### WARC Detail Page (`#/warc/00000`)
5. **System section is irrelevant** — remove system stats (Disk, Heap, Goroutines) from single-WARC view.
6. **Advanced actions are stale** — pack format selector, source selector, re-index are old pipeline concepts. Simplify to three pipeline actions: Download → Markdown → Index.
7. **Text copy is outdated** — modernize labels, descriptions, and button text.

### Browse Page (`#/browse`)
8. **"Pack" terminology** — Browse uses "packed" chips and "Pack Shard" button. Rename to "index" to match WARC pipeline (the user sees: Download → Markdown → Index).
9. **Status chips don't match WARC page** — Browse shows "downloaded", "packed", "indexed", "scanning". Should use the same 3-phase pipeline: Download → Markdown → Index.
10. **Browse uses stale `refreshDashboardContext()`** — should use `refreshCentralState()`.

### Jobs Page (`#/jobs`)
11. **Stat cards are plain** — need better visual hierarchy with colored borders matching job type icons.
12. **Summary line is raw text** — `jobs:4 · running:0` style is developer-facing, not user-friendly.

## Changes

### 1. WARC List Page — Stats Pane
- Show manifest total (100,000) as primary stat with label "Total WARCs"
- Show known/tracked count (1,000) as secondary with label "Tracked"
- Show downloaded, markdown, indexed counts as pipeline progress
- Remove per-page dropdown (fixed at 200)
- Remove "Run next step" batch button and `warcBatchNext()` function

### 2. WARC Detail Page — Single WARC Focus
- Remove System section entirely
- Remove pack format selector, source selector, re-index button
- Advanced Actions → just three buttons: Download, Extract Markdown, Build Index
- Add Re-index (delete + rebuild) as single button
- Add danger zone: delete specific artifacts (warc, markdown, index, all)
- Modern copy: "Pipeline" → "Processing Steps", cleaner descriptions

### 3. Browse Page — Rename Pack → Index
- Sidebar chips: "downloaded" → stays, "packed" → "markdown", "indexed" → "indexed"
- "Pack Shard" button → "Extract Markdown"
- `renderNotPackedState()` → update copy to say "not yet converted to markdown"
- Replace `refreshDashboardContext()` calls with `refreshCentralState()`
- Replace `refreshDashboardMeta()` calls with `refreshCentralState(true)`

### 4. Jobs Page — Better Stat Cards
- Stat cards get colored left border matching status color
- Remove raw summary line, replace with clean header
- Queued count added to cards

## File Changes

| File | Changes |
|------|---------|
| `js/warc.js` | Stats pane uses manifest total; remove page-size selector; remove batch button; simplify detail actions; remove system section |
| `js/browse.js` | Rename pack→index terminology; use `refreshCentralState()`; update chip labels |
| `js/jobs.js` | Better stat card styling; remove raw summary line |
