# Central State: Consistent Data Across All Dashboard Pages

## Problem

The dashboard has **5 pages** (Overview, Search, Browse, WARC, Jobs) that each fetch data independently, leading to inconsistent numbers, stale state, and confusing UX.

### Current Inconsistencies

1. **WARC totals disagree with Overview pipeline**
   - Overview: `buildOverviewResponse()` scans filesystem directly (`scanDownloadedStage` counts `.warc.gz` files in `warc/`)
   - WARC page: `handleWARCList()` uses metastore cache → `summarizeWARCRecords()`
   - Result: Overview may show 5 downloaded, WARC summary shows 4 (metastore hasn't refreshed yet)

2. **Manifest total missing from WARC page**
   - WARC summary shows `summary.total` = number of records the metastore knows about (e.g. 78,600)
   - Overview shows `manifest.total_warcs` = full manifest count (e.g. 100,000)
   - User sees "78,600 Total WARCs" on WARC page but "100,000 Total WARCs" on Overview → confusion

3. **Jobs fetched independently per page**
   - Overview: `apiJobs()` → `state.jobs`
   - Jobs page: `reloadJobs()` → `state.jobs` (overwrites)
   - WARC page: doesn't fetch jobs; only knows about jobs created during current session via `state.jobs.unshift()`
   - Navigating between pages causes `state.jobs` to be stale or overwritten

4. **Header status "syncing" always wrong**
   - `updateHeaderMetaChip()` reads `metaContext()` which merges `state.overview.meta` + `state.metaStatus`
   - On first load, both are null → shows stale dot/text
   - `refreshDashboardContext()` calls both `apiMetaStatus()` and `apiOverview()` every 20s — but overview is heavy (scans filesystem)
   - If meta is refreshing when page loads, header shows "syncing" until next 20s tick even after refresh completes

5. **No projection/estimation on WARC page**
   - Overview shows "Est. URLs", "Est. Size", "Projected Full" — none of this on WARC page
   - User can't see "if I download all 100K WARCs, how much disk will I need?" from the WARC page

6. **Active jobs not visible on WARC page**
   - WARC page only tracks jobs created via warcAction() in current session
   - Pre-existing running jobs (from another tab, server restart recovery) are invisible
   - No active job banner on WARC list page

## Solution: Central State Object

### Architecture

```
                    ┌─────────────────────┐
                    │  refreshCentralState │  (called on init, route change, 20s interval)
                    │                     │
                    │  Fetches:           │
                    │  1. /api/overview   │  (pipeline counts, storage, system)
                    │  2. /api/jobs       │  (all jobs)
                    │  3. /api/meta/status│  (meta backend status)
                    └─────────┬───────────┘
                              │
                              ▼
              ┌───────────────────────────────┐
              │         state.central         │
              │                               │
              │  .overview   (OverviewResponse)│
              │  .jobs       ([]*Job)          │
              │  .meta       (MetaStatus)      │
              │  .loadedAt   (timestamp)       │
              │  .loading    (bool)            │
              └───────────┬───────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
     Overview          WARC Console      Jobs
     (reads all)       (reads overview   (reads .jobs)
                        + .jobs)
```

### Key Changes

#### 1. New `state.central` object
Replace scattered state (`state.overview`, `state.jobs`, `state.metaStatus`) with a single `state.central` object populated by one function.

#### 2. Single refresh function: `refreshCentralState()`
- Fetches `/api/overview` + `/api/jobs` + `/api/meta/status` in parallel
- Updates `state.central` atomically
- Updates header status chip
- Called: on init, on every route change, on 20s interval, after job completion
- Debounced: skip if last refresh was <2s ago (unless forced)

#### 3. WARC page uses central state for totals
- `state.central.overview.manifest.total_warcs` → "Total in Manifest"
- WARC summary stats still from `/api/warc` (per-warc detail needs it)
- But top-level counts cross-referenced with overview for consistency
- Show projection: "If all indexed: est. X TB, Y billion URLs"

#### 4. Jobs visible everywhere
- Active jobs from `state.central.jobs` shown in:
  - Header: badge count of active jobs
  - Overview: active jobs section (already exists)
  - WARC: active jobs banner (new)
  - Jobs: full list (already exists)
- WebSocket updates modify `state.central.jobs` directly
- Job creation from any page updates `state.central.jobs`

#### 5. Header status fixed
- Header chip shows: meta backend + last updated time + active jobs count
- "syncing" only when `meta.refreshing === true`
- Active job count badge next to "Jobs" tab
- WebSocket connection status indicator

### Implementation Steps

1. **Add `state.central`** to state.js — replace `state.overview`, `state.jobs`, `state.metaStatus` with nested object
2. **Create `refreshCentralState()`** in utils.js — single function that populates central state
3. **Update init.js** — call `refreshCentralState()` on startup, set up 20s interval
4. **Update router.js** — call `refreshCentralState()` on route change (debounced)
5. **Update overview.js** — read from `state.central.overview` and `state.central.jobs`
6. **Update jobs.js** — read from `state.central.jobs`, write back to `state.central.jobs`
7. **Update warc.js** — show manifest total from `state.central.overview.manifest.total_warcs`, show active jobs banner
8. **Update header** — show active job count badge, fix syncing indicator
9. **Update `onJobUpdate()`** — modify `state.central.jobs` instead of `state.jobs`

### WARC Page Enhancement: Full Crawl Estimation

Show a "Full Crawl Estimate" card when we have enough data:
```
If all 100,000 WARCs indexed:
  Est. Download:  ~12.8 TB  (avg 128 MB/WARC × 100K)
  Est. Documents: ~3.0B     (avg 30K docs/WARC × 100K)
  Est. Index:     ~2.1 TB   (avg 21 MB/shard × 100K)
  Disk Required:  ~15 TB    (warc + md + index)
```

### Header Enhancement: Active Jobs Badge

```
[Overview] [Search] [Browse] [WARC] [Jobs (2)] [● syncing]
                                      ^^^^^^^^
                                      badge shows active count
```

### API Changes

**None required.** All data already available from existing endpoints. The change is purely frontend — fetching the same data once and sharing it across pages.

## Files to Change

| File | Change |
|------|--------|
| `static/js/state.js` | Add `state.central` object structure |
| `static/js/utils.js` | Add `refreshCentralState()`, update `updateHeaderMetaChip()` |
| `static/js/init.js` | Use `refreshCentralState()` for startup, interval |
| `static/js/router.js` | Replace `refreshDashboardContext()` with `refreshCentralState()` |
| `static/js/overview.js` | Read from `state.central.*` |
| `static/js/jobs.js` | Read/write `state.central.jobs`, add active badge |
| `static/js/warc.js` | Show manifest total, estimation card, active jobs |
| `static/index.html` | Add badge element to Jobs tab |

## Non-Goals

- No backend API changes
- No WebSocket protocol changes
- No new persistence layer
- No React/framework migration
