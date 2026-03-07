# Browse Rewrite: Use WARC Console Data Model

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rewrite the browse feature to derive shard state from the WARC console's `has_markdown` / `has_fts` flags, drop legacy `markdown/` directory support, chain Extract & Index jobs properly, and auto-trigger doc scans so browsing works immediately after indexing completes.

**Architecture:** Remove the parallel shard-state logic in `handleBrowseShards` that duplicates what the WARC console already knows. Instead, query WARC records (via MetaManager) to determine which shards have markdown + FTS. The frontend `triggerPackShard` chains markdown→index jobs via WebSocket completion events (index starts only after markdown completes). `refreshAfterJobComplete` gains browse support. Auto doc-scan fires when a packed shard is first browsed without scan data.

**Tech Stack:** Go (server.go, doc_store.go, warc_api.go), vanilla JS (browse.js, jobs.js)

---

## Current State (Bugs)

1. **`handleBrowseShards`** uses `listWARCMdShards()` which only finds `warc_md/*.md.warc.gz` — misses legacy `markdown/{idx}/` dirs entirely
2. **`triggerPackShard`** fires markdown + index jobs simultaneously — index fails because it starts before markdown finishes
3. **`refreshAfterJobComplete`** ignores `currentPage === 'browse'` — browse never refreshes after job completion
4. **No auto doc-scan** — after Extract & Index, browse shows "Shard not yet indexed" until manual Scan Docs click

## Design

### Backend: `handleBrowseShards` rewrite

Replace `listWARCShards` + `listWARCMdShards` with a single call to MetaManager's WARC records. For each downloaded WARC with `has_markdown`:
- `has_pack = true` (shard has extractable content)
- Check DocStore for `has_scan`

This eliminates all duplicate disk scanning and aligns browse state with the WARC console.

### Backend: Auto doc-scan on first browse

When `handleBrowseDocs` finds `has_pack=true` but `has_scan=false` and the `.md.warc.gz` file exists, auto-trigger `ScanShard` in the background instead of just returning `not_scanned`. Return `scanning: true` so the frontend shows a scanning indicator.

### Frontend: Chain jobs in `triggerPackShard`

Subscribe to the markdown job's WebSocket events. Only create the index job when the markdown job completes. If markdown fails, show error without queuing index.

### Frontend: Browse refresh after job complete

Add `currentPage === 'browse'` branch in `refreshAfterJobComplete` to re-fetch shard list and reload current shard docs.

### Cleanup: Remove legacy `markdown/` support from browse

- Remove `listWARCShards()` (only used by browse)
- Remove `listWARCMdShards()` (only used by browse)
- Remove legacy `markdown/` fallback from `enrichWARCAPIRecord` (no — keep that, WARC console still uses it)

Actually: `listWARCShards` scans `warc/` for downloaded files — browse still needs to know which WARCs exist. But the rewrite replaces this with MetaManager data. `listWARCMdShards` scans `warc_md/` for packed files — replaced by MetaManager's `has_markdown` flag. Keep `enrichWARCAPIRecord` as-is since WARC console uses it.

---

## Implementation Plan

### Task 1: Rewrite `handleBrowseShards` to use WARC records from MetaManager

**Files:**
- Modify: `pkg/index/web/server.go:699-746` (`handleBrowseShards`)

**Step 1: Replace `handleBrowseShards` implementation**

Replace the entire function body. Instead of scanning `warc/` + `warc_md/` dirs, call MetaManager to get WARC records and build shard entries from them:

```go
func (s *Server) handleBrowseShards(c *mizu.Ctx) error {
	crawlID := s.CrawlID
	crawlDir := s.CrawlDir

	// Get WARC records from MetaManager (same source as WARC console).
	var recs []metastore.WARCRecord
	if s.Meta != nil {
		var err error
		recs, _, err = s.Meta.ListWARCs(c.Context(), crawlID, crawlDir)
		if err != nil {
			return c.JSON(500, errResp{err.Error()})
		}
	}

	// Only show shards that have been downloaded (warc_bytes > 0).
	var metas []DocShardMeta
	if s.Docs != nil {
		metas, _ = s.Docs.ListShardMetas(c.Context(), crawlID)
	}
	metaByName := make(map[string]DocShardMeta, len(metas))
	for _, m := range metas {
		metaByName[m.Shard] = m
	}

	var entries []shardEntry
	for _, rec := range recs {
		if rec.WARCBytes <= 0 {
			continue // not downloaded
		}
		// Derive local shard name from filename.
		localIdx := rec.WARCIndex
		if rec.Filename != "" {
			if s, ok := warcIndexFromPathStrict(rec.Filename); ok {
				localIdx = s
			}
		}

		hasMarkdown := rec.MarkdownDocs > 0 || rec.MarkdownBytes > 0
		// Also check warc_md file on disk (enrichment adds this).
		if !hasMarkdown {
			mdPath := filepath.Join(crawlDir, "warc_md", localIdx+".md.warc.gz")
			if _, err := os.Stat(mdPath); err == nil {
				hasMarkdown = true
			}
		}
		hasFTS := sumInt64Map(rec.FTSBytes) > 0

		e := shardEntry{
			Name:    localIdx,
			HasPack: hasMarkdown,
			HasFTS:  hasFTS,
		}

		if m, ok := metaByName[localIdx]; ok {
			e.HasScan = true
			e.FileCount = int(m.TotalDocs)
			e.TotalSize = m.TotalSizeBytes
			if !m.LastDocDate.IsZero() {
				e.LastDocDate = m.LastDocDate.UTC().Format(time.RFC3339)
			}
			e.MetaStale = time.Since(m.LastScannedAt) > time.Hour
			if !m.LastScannedAt.IsZero() {
				e.LastScannedAt = m.LastScannedAt.UTC().Format(time.RFC3339)
			}
			if e.HasPack && e.MetaStale && s.Docs != nil {
				e.Scanning = s.Docs.IsScanning(crawlID, localIdx)
			}
		} else if e.HasPack && s.Docs != nil {
			e.Scanning = s.Docs.IsScanning(crawlID, localIdx)
		}
		entries = append(entries, e)
	}

	return c.JSON(200, BrowseShardsResponse{
		Shards:     entries,
		HasDocMeta: s.Docs != nil,
	})
}
```

**Step 2: Add `HasFTS` to `shardEntry` struct**

In `server.go`, add `HasFTS` field to `shardEntry`:

```go
type shardEntry struct {
	Name          string `json:"name"`
	HasPack       bool   `json:"has_pack"`
	HasFTS        bool   `json:"has_fts"`
	HasScan       bool   `json:"has_scan"`
	Scanning      bool   `json:"scanning"`
	FileCount     int    `json:"file_count"`
	TotalSize     int64  `json:"total_size,omitempty"`
	LastDocDate   string `json:"last_doc_date,omitempty"`
	MetaStale     bool   `json:"meta_stale"`
	LastScannedAt string `json:"last_scanned_at,omitempty"`
}
```

**Step 3: Remove `listWARCShards` and `listWARCMdShards` from `server.go`**

These are now unused by browse. `listWARCMdShards` is only used in `server.go` (browse) and `doc_store.go`. Check `doc_store.go` — it's only used in `listWARCMdShards` calls within `server.go`. Safe to remove both from `server.go`. `listWARCMdShards` in `doc_store.go` is used by `ScanAll` — keep that one.

Actually, `listWARCMdShards` is defined in `doc_store.go:728` and only called from `server.go:705`. After the rewrite, it's unused. But `ScanAll` in `doc_store.go` uses `os.ReadDir` directly, not `listWARCMdShards`. So remove `listWARCShards` from `server.go:1086-1113` and remove the import of `listWARCMdShards` call.

Wait — `listWARCMdShards` is defined in `doc_store.go` but only called from `handleBrowseShards` in `server.go`. After rewrite, it's dead code. Remove it from `doc_store.go`.

**Step 4: Verify build**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build ./pkg/index/web/...
```

**Step 5: Commit**

```
feat(web/browse): rewrite handleBrowseShards to use WARC records from MetaManager
```

---

### Task 2: Auto doc-scan on first browse of packed shard

**Files:**
- Modify: `pkg/index/web/server.go:748-824` (`handleBrowseDocs`)

**Step 1: Add auto-scan trigger**

When `!hasMeta` and the `.md.warc.gz` file exists, trigger `ScanShard` in background and return `scanning: true`:

In `handleBrowseDocs`, replace the `!hasMeta` block (lines 764-778):

```go
	if !hasMeta {
		warcMdPath := filepath.Join(s.WARCMdBase, shard+".md.warc.gz")
		if _, err := os.Stat(warcMdPath); err != nil {
			return c.JSON(404, errResp{"shard not packed yet"})
		}
		// Auto-trigger scan in background.
		if s.Docs != nil && !scanning {
			scanning = true
			go func() {
				if _, err := s.Docs.ScanShard(context.Background(), s.CrawlID, shard, warcMdPath); err != nil {
					logErrorf("doc_store: auto-scan shard=%s err=%v", shard, err)
				}
			}()
		}
		return c.JSON(200, BrowseDocsResponse{
			Shard:      shard,
			Docs:       []docJSON{},
			Total:      0,
			Page:       1,
			Scanning:   scanning,
			NotScanned: !scanning,
		})
	}
```

Key change: `NotScanned` is only true when we can't scan (no DocStore). If we auto-triggered scan, return `scanning: true` instead so the frontend shows "scanning in background" rather than the manual "Scan Docs" button.

**Step 2: Verify build**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build ./pkg/index/web/...
```

**Step 3: Commit**

```
feat(web/browse): auto-trigger doc scan on first browse of packed shard
```

---

### Task 3: Chain Extract & Index jobs in frontend

**Files:**
- Modify: `pkg/index/web/static/js/browse.js:200-306` (`triggerPackShard`)

**Step 1: Rewrite `triggerPackShard` to chain jobs**

Replace the entire function. Key changes:
- Create markdown job first
- Subscribe to markdown job WebSocket events
- Only create index job when markdown completes
- If markdown fails, show error without queuing index

```javascript
async function triggerPackShard(shard) {
  const el = $('browse-content');
  if (!el) return;

  el.innerHTML = `
    <div class="mt-6 py-8">
      <div class="text-sm font-medium mb-1">Starting pipeline for ${esc(shard)}\u2026</div>
      <div id="pack-status-msg" class="text-xs ui-subtle mb-5">Queueing markdown extraction\u2026</div>
      <div id="pack-progress-area" class="max-w-sm space-y-4"></div>
    </div>`;

  const fileIdx = parseInt(shard, 10);
  const filesStr = String(fileIdx);

  const progress = {
    md:  { id: null, status: 'queued', pct: 0, msg: '' },
    idx: { id: null, status: 'queued', pct: 0, msg: '' },
  };

  function renderPackProgress() {
    const msgEl = $('pack-status-msg');
    const areaEl = $('pack-progress-area');
    if (!areaEl) return;

    const mdDone = progress.md.status === 'completed';
    const idxDone = progress.idx.status === 'completed';
    const mdFailed = progress.md.status === 'failed';
    const idxFailed = progress.idx.status === 'failed';

    if (msgEl) {
      if (mdFailed) msgEl.textContent = 'Markdown extraction failed.';
      else if (idxFailed) msgEl.textContent = 'Index build failed.';
      else if (mdDone && idxDone) msgEl.textContent = 'Pipeline complete \u2014 loading documents\u2026';
      else if (mdDone && progress.idx.id) msgEl.textContent = 'Markdown done \u2014 building index\u2026';
      else if (mdDone) msgEl.textContent = 'Markdown done \u2014 starting index\u2026';
      else msgEl.textContent = 'Extracting markdown\u2026';
    }

    areaEl.innerHTML = `
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-[11px] font-mono ui-subtle">1. Extract Markdown</span>
          <span class="text-[11px] font-mono ${mdDone ? 'status-completed' : mdFailed ? 'text-red-400' : ''}">${mdDone ? '\u2713 done' : mdFailed ? 'failed' : progress.md.pct + '%'}</span>
        </div>
        <div class="progress-track" style="height:4px">
          <div class="${mdDone ? 'ov-c2' : 'progress-fill'}" style="width:${mdDone ? 100 : progress.md.pct}%;height:100%"></div>
        </div>
        ${progress.md.msg ? `<div class="text-[10px] font-mono ui-subtle mt-1 truncate">${esc(progress.md.msg)}</div>` : ''}
      </div>
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-[11px] font-mono ui-subtle">2. Build Index</span>
          <span class="text-[11px] font-mono ${idxDone ? 'status-completed' : idxFailed ? 'text-red-400' : ''}">${!progress.idx.id && !mdDone ? 'waiting' : idxDone ? '\u2713 done' : idxFailed ? 'failed' : progress.idx.pct + '%'}</span>
        </div>
        <div class="progress-track" style="height:4px">
          <div class="${idxDone ? 'ov-c4' : 'progress-fill'}" style="width:${idxDone ? 100 : progress.idx.pct}%;height:100%"></div>
        </div>
        ${progress.idx.msg ? `<div class="text-[10px] font-mono ui-subtle mt-1 truncate">${esc(progress.idx.msg)}</div>` : ''}
      </div>`;

    if (mdDone && idxDone) {
      setTimeout(() => {
        apiBrowse().then(data => {
          state.browseShards = data.shards || [];
          renderShardList(state.browseShard);
          updateBrowseRefreshedAt();
          loadShardDocs(shard, 1);
        }).catch(() => loadShardDocs(shard, 1));
      }, 800);
    }
  }

  // Step 1: Create markdown job.
  let mdRes;
  try {
    mdRes = await apiPost('/api/jobs', { type: 'markdown', files: filesStr });
  } catch (e) {
    el.innerHTML = `<div class="text-xs text-red-400 py-8">${esc('Failed to queue markdown job: ' + e.message)}</div>`;
    return;
  }
  progress.md.id = mdRes && mdRes.job && mdRes.job.id;
  if (!progress.md.id) {
    el.innerHTML = `<div class="text-xs text-red-400 py-8">Failed to create markdown job.</div>`;
    return;
  }

  // Step 2: Subscribe to markdown job — chain index job on completion.
  wsClient.subscribe(progress.md.id, async (m) => {
    if (m.progress !== undefined) progress.md.pct = Math.round((m.progress || 0) * 100);
    if (m.status) progress.md.status = m.status;
    if (m.message) progress.md.msg = m.message;
    renderPackProgress();

    // When markdown completes, create index job.
    if (m.status === 'completed' && !progress.idx.id) {
      try {
        const idxRes = await apiPost('/api/jobs', { type: 'index', files: filesStr, source: 'files' });
        progress.idx.id = idxRes && idxRes.job && idxRes.job.id;
        if (progress.idx.id) {
          wsClient.subscribe(progress.idx.id, (m2) => {
            if (m2.progress !== undefined) progress.idx.pct = Math.round((m2.progress || 0) * 100);
            if (m2.status) progress.idx.status = m2.status;
            if (m2.message) progress.idx.msg = m2.message;
            renderPackProgress();
          });
        }
      } catch (e) {
        progress.idx.status = 'failed';
        progress.idx.msg = e.message;
      }
      renderPackProgress();
    }
  });

  renderPackProgress();
}
```

**Step 2: Verify frontend builds (no build step for vanilla JS, just review)**

Test by navigating to `http://localhost:3456/#/browse/00000` after restart.

**Step 3: Commit**

```
fix(web/browse): chain markdown→index jobs, index starts only after markdown completes
```

---

### Task 4: Add browse refresh after job complete

**Files:**
- Modify: `pkg/index/web/static/js/jobs.js:552-582` (`refreshAfterJobComplete`)

**Step 1: Add browse branch**

Add `else if (state.currentPage === 'browse')` before the final catch:

```javascript
async function refreshAfterJobComplete(completedJob) {
  try {
    await refreshCentralState(true);
    if (state.currentPage === 'warc' && state.warcDetail) {
      const detailIndex = (state.warcDetail.warc || {}).index;
      if (detailIndex) {
        const data = await apiWARCDetail(detailIndex);
        state.warcDetail = data;
        if ($('warc-detail-content')) renderWARCDetailContent(data, detailIndex);
      }
    } else if (state.currentPage === 'warc') {
      const data = await apiWARCList({
        offset: state.warcOffset || 0,
        limit: state.warcLimit || 200,
        q: state.warcQuery || '',
        phase: state.warcPhase || '',
      });
      state.warcSummary = data.summary || state.warcSummary;
      state.warcTotal = data.total || state.warcTotal;
      state.warcRows = data.warcs || state.warcRows;
      if ($('warc-summary')) renderWARCSummary(state.warcSummary);
      if ($('warc-tabs')) renderWARCTabs(state.warcSummary, state.warcTotal);
      if ($('warc-content')) renderWARCTable(data);
    } else if (state.currentPage === 'browse') {
      try {
        const data = await apiBrowse();
        state.browseShards = data.shards || [];
        renderShardList(state.browseShard);
        updateBrowseRefreshedAt();
        if (state.browseShard) loadShardDocs(state.browseShard, state.browsePage || 1);
      } catch (_) {}
    } else if (state.currentPage === 'overview') {
      if ($('overview-content')) renderOverviewContent(state.central.overview, state.central.jobs);
    }
  } catch (_) {}
}
```

**Step 2: Commit**

```
fix(web/browse): refresh shard list and docs after job completes on browse page
```

---

### Task 5: Update browse sidebar to show `has_fts` status

**Files:**
- Modify: `pkg/index/web/static/js/browse.js:153-198` (`renderShardList`)

**Step 1: Update chip logic to use `has_fts`**

Replace the chips logic to show clearer pipeline state:

```javascript
function renderShardList(active) {
  const el = $('shard-list');
  if (!el || !state.browseShards) return;
  if (state.browseShards.length === 0) {
    el.innerHTML = `<div class="ui-empty">No shards</div>`;
    return;
  }
  el.innerHTML = state.browseShards.map(s => {
    const isActive = s.name === active;
    const hasPack = !!s.has_pack;
    const hasFTS = !!s.has_fts;
    const hasScan = !!s.has_scan;
    const scanning = !!s.scanning;
    const ready = hasPack && hasScan;

    const chips = [];
    if (scanning) {
      chips.push(`<span class="ui-chip" style="border-color:rgba(96,165,250,0.6);color:#93c5fd">scanning</span>`);
    } else if (ready) {
      chips.push(`<span class="ui-chip ui-chip-ok">ready</span>`);
    } else if (hasFTS) {
      chips.push(`<span class="ui-chip" style="border-color:rgba(52,211,153,0.6);color:#6ee7b7">indexed</span>`);
    } else if (hasPack) {
      chips.push(`<span class="ui-chip" style="border-color:rgba(99,102,241,0.6);color:#a5b4fc">markdown</span>`);
    } else {
      chips.push(`<span class="ui-chip ui-chip-off">downloaded</span>`);
    }

    const countLabel = hasScan ? (s.file_count ?? 0).toLocaleString() : '';
    const sizeLabel = hasScan && s.total_size ? fmtBytes(s.total_size) : '';
    const packBtn = !hasPack && isDashboard
      ? `<button onclick="event.preventDefault();event.stopPropagation();triggerPackShard('${esc(s.name)}')" class="text-[9px] font-mono px-1.5 py-0.5 ui-btn">Extract &amp; Index</button>`
      : '';

    return `<a href="#/browse/${s.name}"
       class="block py-1.5 px-2 text-xs font-mono cursor-pointer transition-colors ${isActive ? 'shard-active' : 'shard-item'}" ${!hasPack ? 'style="opacity:0.55"' : ''}>
      <div class="flex items-center justify-between gap-1">
        <span class="truncate">${esc(s.name)}</span>
        ${countLabel ? `<span class="ui-subtle shrink-0">${countLabel}</span>` : ''}
      </div>
      <div class="flex items-center gap-1 mt-1 flex-wrap">
        ${chips.join('')}${packBtn}
        ${sizeLabel ? `<span class="text-[9px] font-mono ui-subtle ml-auto">${sizeLabel}</span>` : ''}
      </div>
    </a>`;
  }).join('');
  updateMobileSidebarLabel(active);
}
```

**Step 2: Update `loadShardDocs` not-packed check**

In `loadShardDocs` (line 342), the check `!shardInfo.has_pack` already works since we set `has_pack = hasMarkdown` in the backend. No change needed.

**Step 3: Commit**

```
feat(web/browse): show indexed/markdown/downloaded chips using has_fts field
```

---

### Task 6: Delete stale `listWARCShards` function

**Files:**
- Modify: `pkg/index/web/server.go` (remove `listWARCShards` function, lines 1086-1113)
- Modify: `pkg/index/web/doc_store.go` (remove `listWARCMdShards` function, lines 728-741)

**Step 1: Remove `listWARCShards` from server.go**

Delete the function at lines 1086-1113. Verify no other callers:

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search && grep -rn 'listWARCShards' pkg/index/web/
```

**Step 2: Remove `listWARCMdShards` from doc_store.go**

Delete the function at lines 728-741. Verify no other callers:

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search && grep -rn 'listWARCMdShards' pkg/index/web/
```

**Step 3: Verify build**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search && go build ./pkg/index/web/...
```

**Step 4: Commit**

```
refactor(web): remove unused listWARCShards and listWARCMdShards
```

---

### Task 7: Integration test — rebuild and verify end-to-end

**Step 1: Rebuild binary**

```bash
cd /Users/apple/github/go-mizu/mizu/blueprints/search && make install
```

**Step 2: Start dashboard**

```bash
search cc fts dashboard
```

**Step 3: Verify browse API returns correct state**

```bash
curl -s http://localhost:3456/api/browse | python3 -m json.tool
```

Expected: shards with `has_pack: true` for 00000/00001/00002 (those with markdown data), `has_fts: true` for 00000/00001 (those with rose index).

**Step 4: Verify browse UI**

1. Navigate to `http://localhost:3456/#/browse/00000`
2. Should auto-trigger scan (shown as "scanning in background")
3. After scan completes, refresh should show documents
4. Navigate to `http://localhost:3456/#/browse/00003`
5. Should show "Extract & Index" button (downloaded but no markdown)
6. Click "Extract & Index" — should show 2-step progress, index waits for markdown

**Step 5: Commit**

```
test(web/browse): verify browse rewrite works end-to-end
```
