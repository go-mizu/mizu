# 0683: Browse Page — Full Review & Fix

## Status: In Progress
**Date**: 2026-03-07
**Dashboard**: `search cc fts dashboard --port 3457`
**Crawl**: CC-MAIN-2026-08 (4 WARCs downloaded, markdown extracted, FTS indexed)

---

## Architecture Summary

### Data Flow
```
WARC (.warc.gz)
  → Phase 1: extract HTML → warc_single/{uuid}.warc (temp, removed)
  → Phase 2: convert HTML→MD → markdown/{shard}/{aa/bb/cc/uuid}.md (final)
  → Index: markdown/{shard}/ → fts/{engine}/{shard}/ (FTS index)
```

### Backend Components
- **DocStore** (`doc_store.go`): Per-shard DuckDB + in-memory cache for doc metadata
- **handleBrowseShards** (`server.go:701`): Lists shards with status chips
- **handleBrowseDocs** (`server.go:779`): Paginated doc listing for one shard
- **handleDoc** (`server.go:629`): Single doc viewer (markdown + HTML render)
- **WebSocket hub** (`ws.go`): Job progress broadcast
- **JobManager** (`jobs.go`): Pipeline job lifecycle + onComplete hook

### Frontend Components
- **renderBrowse** (`browse.js:4`): Main browse page with sidebar + content
- **renderShardList** (`browse.js:153`): Shard sidebar with status chips
- **renderDocTable** (`browse.js:380`): Paginated doc table
- **triggerPackShard** (`browse.js:202`): Pipeline progress UI (markdown → index)
- **WSClient** (`websocket.js`): Job subscription for real-time progress

---

## Bugs Found

### CRITICAL: Browse Page Completely Broken for New Markdown Format

The browse page was built for the **old format** (`.md.warc.gz` files in `warc_md/`),
but the current pipeline produces **individual `.md` files** in `markdown/{shard}/`.

| Component | Expected Path | Actual Path | Result |
|-----------|--------------|-------------|--------|
| DocStore.ScanShard | `warc_md/{shard}.md.warc.gz` | `markdown/{shard}/` | Can't scan: file doesn't exist |
| DocStore.ScanAll | `warc_md/*.md.warc.gz` | `markdown/*/` | Scans nothing: no `.md.warc.gz` files |
| handleBrowseDocs | `warc_md/{shard}.md.warc.gz` | `markdown/{shard}/` | **404 "shard not packed yet"** |
| handleDoc | `warc_md/{shard}.md.warc.gz` (WARC scan) | `markdown/{shard}/{prefix}/{uuid}.md` | **404 "document not found"** |
| onComplete hook | `ScanAll(crawlDir/warc_md)` | needs `crawlDir/markdown` | Scans empty dir |

**Impact**: Every shard shows `has_pack:true, has_fts:true` in the list but clicking
any shard returns "Shard not yet indexed" or 404.

### BUG-1: Shard list shows wrong state
- Shards show `has_pack:true` (correctly detected from `markdown/` dir by `warc_meta_scan.go`)
- But `has_scan:false`, `file_count:0` because DocStore can't find `.md.warc.gz`
- Chips show "ready" (has_pack && has_scan) — wrong since has_scan is false
- Should show file count from `markdown/{shard}/` walk

### BUG-2: "Not scanned" vs "Not packed" confusion
- `handleBrowseDocs` returns `not_scanned:true` when DuckDB metadata doesn't exist
- Frontend shows "Shard not yet indexed" — misleading when FTS index exists
- Auto-trigger scan fires but fails (no `.md.warc.gz` to scan)

### BUG-3: Doc viewer reads from wrong path
- `readDocFromWARCMd()` scans `.md.warc.gz` using WARC reader
- Individual `.md` files use UUID-sharded path: `markdown/{shard}/{aa/bb/cc/uuid}.md`
- Should use `warc_md.RecordIDToRelPath(docID)` to reconstruct path

### BUG-4: No WebSocket event for shard scan completion
- Background doc scan (`ScanShard`) finishes silently
- Browse page shows "Scanning shard..." but never knows when to refresh
- User must manually click Refresh to see results

### BUG-9: browseView resets on page entry
- `renderBrowse()` line 9: `state.browseView = 'docs'`
- Switching tabs and returning to Browse always resets to Docs view
- User loses Stats tab selection

### BUG-5: onComplete hook scans wrong directory
- `server.go:293`: `warcMdBase := filepath.Join(crawlDir, "warc_md")`
- Should also scan `markdown/` directory
- After markdown/index jobs complete, doc metadata is never populated

---

## Fixes Required

### Backend

#### Fix 1: DocStore.ScanMarkdownDir — scan individual .md files
New method that walks `markdown/{shard}/` and extracts metadata:
- `doc_id`: UUID from filename (strip `.md` extension)
- `title`: First H1/H2 from markdown content (first 256 bytes)
- `size_bytes`: From `os.Stat()`
- `word_count`: Estimate from `size_bytes / 5`
- `url`, `crawl_date`: Empty (not available in individual files)

#### Fix 2: DocStore.ScanShard — auto-detect format
- If path ends with `.md.warc.gz` → existing WARC scanning
- If path is a directory → new `ScanMarkdownDir` method
- Signature change: accept either file or directory path

#### Fix 3: DocStore.ScanAll — support both formats
- Walk `warc_md/` for `.md.warc.gz` files (backward compat)
- Walk `markdown/` for shard directories with `.md` files

#### Fix 4: handleBrowseDocs — detect markdown dir
- Check `markdown/{shard}/` directory in addition to `warc_md/{shard}.md.warc.gz`
- Auto-trigger scan for the correct path

#### Fix 5: handleDoc — read from markdown dir
- Try `warc_md/{shard}.md.warc.gz` first (old format, has URL/date)
- Fallback to `markdown/{shard}/{prefix}/{uuid}.md` (new format)
- Use `warc_md.RecordIDToRelPath(docID)` for path construction

#### Fix 6: onComplete hook — scan correct directory
- After job completes, scan `markdown/` (not just `warc_md/`)

#### Fix 7: WebSocket broadcast for shard scan completion
- Broadcast `shard_scan` event when `ScanShard` or `ScanMarkdownDir` completes
- Include shard name and doc count in the event

### Frontend

#### Fix 8: Subscribe to shard_scan events
- WSClient subscribes to `shard_scan` events (wildcard or special topic)
- On event: refresh shard list + reload current shard docs
- Only refresh affected components, not full page re-render

#### Fix 9: Fix browseView reset
- Don't reset `browseView` on page entry
- Preserve user's Docs/Stats tab selection

#### Fix 10: Correct empty/scanning state messages
- "Scanning documents..." when background scan is in progress
- "No documents in this shard" when scan completes with 0 docs
- Remove misleading "Shard not yet indexed" message

---

## Expected Workflows

### Workflow 1: First Visit (Empty State)
1. User navigates to `#/browse`
2. No shards → "No indexes found. Download and process WARCs first."
3. Link to WARC Console

### Workflow 2: WARCs Downloaded, No Markdown
1. Shards appear in sidebar with "downloaded" chip (dimmed)
2. Clicking shard shows "WARC downloaded — not yet extracted"
3. "Extract & Index" button starts pipeline
4. Pipeline progress shows via WebSocket

### Workflow 3: Markdown Extracted, FTS Indexed
1. Shards show "indexed" chip with doc count
2. Clicking shard triggers background doc scan if not scanned
3. "Scanning..." banner while scan runs
4. WebSocket `shard_scan` event triggers auto-refresh
5. Doc table shows title, size, word count
6. Stats tab shows domain distribution, size buckets, date histogram

### Workflow 4: Pipeline Completion
1. markdown job completes → auto-triggers index job
2. index job completes → `onComplete` hook fires
3. Hook triggers `ScanAll` for `markdown/` directory
4. `shard_scan` WS event broadcasts to connected clients
5. Browse page auto-refreshes shard list and current shard docs

### Workflow 5: Doc Viewer
1. Click doc title → navigate to `#/doc/{shard}/{docid}`
2. Read markdown from `markdown/{shard}/{prefix}/{uuid}.md`
3. Render HTML via goldmark
4. Show Rendered/Source tabs
5. "Back" navigates to browse page (preserves shard + page)
