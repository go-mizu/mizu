# 0658: WARC Dashboard (Dedicated WARC Management UX + Metadata Cache)

## 1. Context and Goals

The dashboard currently provides high-level pipeline controls, but lacks a dedicated WARC-focused management experience.

This spec introduces:

- A dedicated `WARC` page in the dashboard for fleet-level WARC visibility.
- A dedicated per-WARC detail page for targeted operations.
- Metadata-first data serving for WARC stats/details (cached in sqlite/duckdb via metastore).
- Action controls for full lifecycle per WARC: download, markdown, pack, index, re-index, delete.
- Better observability: phase-level sizes and status, system disk/memory signals, and progress feedback in UI + logs.
- Dashboard UI refresh to feel like a modern CLI-style operations console (dense information, crisp status chips, practical charts).

## 2. Scope

### In scope

- New dashboard navigation tab: `WARC`.
- New routes in SPA:
  - `#/warc` (list page)
  - `#/warc/{warc_index}` (detail page)
- New backend APIs for WARC list/detail/actions.
- Metastore schema extension for per-WARC metadata.
- Meta refresh pipeline extension to populate per-WARC rows.
- Per-WARC delete/re-index behavior.
- UI charts for phase size distribution and pipeline coverage.
- Tests for metastore round-trip and server route behavior.

### Out of scope

- Replacing existing pipeline page; it remains as bulk workflow control.
- Hard real-time system metrics streaming; snapshot values are sufficient.
- Deep document-level analytics outside WARC lifecycle management.

## 3. Functional Requirements

### FR-1: WARC list page

Provide a dedicated page showing:

- Summary cards:
  - total WARCs
  - downloaded WARCs
  - markdown-ready WARCs
  - index-ready/indexed WARCs
  - total bytes by phase (warc/markdown/pack/index)
- Table listing WARC records (paginated or bounded list):
  - warc index (`00000`)
  - filename
  - remote path (if known)
  - phase statuses
  - bytes by phase
  - last updated timestamp
- Quick actions from row:
  - open detail
  - download
  - markdown
  - pack
  - index

### FR-2: WARC detail page

Provide per-WARC drill-down showing:

- Identity and linkage:
  - warc index
  - filename
  - remote path
  - manifest index
- Phase metrics:
  - warc file size
  - markdown docs count + size
  - pack sizes by format (`parquet/bin/duckdb/markdown`)
  - index sizes by engine
  - computed total on-disk bytes
- Action panel:
  - Start download
  - Convert markdown (`fast` toggle)
  - Pack (format selector)
  - Index (engine + source selector)
  - Re-index (delete target index shard then index)
  - Delete data by phase (warc/markdown/pack/index/all)

### FR-3: Metadata-first serving

WARC list/detail data must be served from metastore cache by default.

- Meta refresh populates both aggregate summary and per-WARC tables.
- API handlers read from cache; fallback to scan only when cache unavailable.
- Existing stale detection + background refresh remains active.

### FR-4: Action execution and progress

Actions are dashboard-callable and visible in progress channels:

- Action APIs produce/trigger existing jobs (`download`, `markdown`, `pack`, `index`) using single WARC file selector.
- Re-index action performs delete then index job creation.
- Delete action performs immediate filesystem operation and triggers metadata refresh.
- Job progress visible through existing job endpoints/WebSocket; action response includes job payload when applicable.

### FR-5: Enhanced dashboard style and charts

Refresh dashboard style to modern CLI-console direction:

- Dense, practical info layout.
- Status chips and compact monospace metrics.
- Simple inline charts:
  - phase-size bar chart
  - pipeline coverage chart (% WARCs at each phase)

## 4. Data Model Changes

## 4.1 Metastore type additions

Add per-WARC record type:

- `WARCRecord`
  - `crawl_id`
  - `warc_index` (string, zero-padded)
  - `manifest_index` (int64, optional)
  - `filename`
  - `remote_path`
  - `warc_bytes`
  - `markdown_docs`
  - `markdown_bytes`
  - `pack_bytes` map `format -> bytes`
  - `fts_bytes` map `engine -> bytes`
  - `total_bytes`
  - `updated_at`

Extend `SummaryRecord` to include optional `WARCs []WARCRecord` on write paths.

## 4.2 Metastore interface additions

Add methods:

- `ListWARCs(ctx, crawlID) ([]WARCRecord, error)`
- `GetWARC(ctx, crawlID, warcIndex) (WARCRecord, bool, error)`

Drivers (`sqlite`, `duckdb`) implement schema + methods.

## 4.3 SQL schema additions

Per-driver new tables:

- `warc_summary` (one row per crawl + warc_index)
- `warc_pack_summary` (pack bytes by format)
- `warc_fts_summary` (index bytes by engine)

Write strategy in `PutSummary`:

- existing summary tables upserted as today
- replace per-crawl WARC rows transactionally

## 5. Metadata Refresh Design

During refresh:

1. Build aggregate `DataSummary` (existing scan).
2. Build local per-WARC phase metadata from filesystem:
   - `warc/`
   - `markdown/{warc_idx}`
   - `pack/{format}/{warc_idx}.*`
   - `fts/{engine}/{warc_idx}/`
3. Optionally merge manifest-derived identity (`remote_path`, `filename`, `manifest_index`) when available.
4. Persist aggregate + per-WARC metadata in one metastore transaction.

Stale behavior:

- unchanged: stale reads trigger background refresh.
- WARC APIs call metadata accessor that may trigger refresh when stale.

## 6. API Design

### 6.1 Read APIs

- `GET /api/warc?crawl=<id>&q=<term>&offset=<n>&limit=<n>`
  - returns summary + list rows.
- `GET /api/warc/{index}?crawl=<id>`
  - returns detailed per-phase metrics for one WARC.

Response includes metadata cache headers/fields:

- backend
- generated_at
- stale
- refreshing
- last_error

### 6.2 Action APIs

- `POST /api/warc/{index}/action`

Request body:

- `action`: `download|markdown|pack|index|reindex|delete`
- optional: `fast`, `format`, `engine`, `source`, `target`

Behavior:

- `download|markdown|pack|index` -> create corresponding single-file job.
- `reindex` -> delete target index shard for chosen engine then create index job.
- `delete` -> immediate phase deletion.

Response:

- action status
- created job (if job action)
- refresh accepted flag (if metadata refresh requested)

## 7. UI / UX Design

## 7.1 Navigation

Add `WARC` tab in main nav.

## 7.2 WARC list page

Sections:

- Header: title + refresh metadata button + cache status line.
- Summary cards with compact metric chips.
- Coverage chart (`downloaded`, `markdown`, `packed`, `indexed`).
- Phase bytes chart.
- Search/filter bar (`index`, filename, path).
- Table with row actions and link to detail page.

## 7.3 WARC detail page

Sections:

- Breadcrumb back to list.
- Identity panel.
- Phase status cards (download/markdown/pack/index) with bytes/docs.
- Action panel with controls.
- Recent job stream related to this WARC.

## 7.4 Visual refresh

Dashboard-wide style adjustments:

- stronger monospace/utilitarian look
- subtle gradient background and panel layering
- denser spacing and clearer hierarchy
- color semantics for status chips (`ready/running/missing/error`)

## 8. Deletion Semantics

Delete targets:

- `warc` -> remove matching `*.warc.gz` for the WARC index
- `markdown` -> remove `markdown/{warc_idx}`
- `pack` -> remove one/all pack files for index
- `index` -> remove one/all `fts/{engine}/{warc_idx}`
- `all` -> remove all above

Post-delete:

- trigger metadata refresh (forced)
- return deleted path count

## 9. Logging and Observability

Add concise logs for:

- WARC metadata reads (list/detail)
- WARC action requests
- deletion operations and affected paths
- WARC-specific job creation

Reuse existing dashboard logger and avoid noisy per-request spam patterns.

## 10. Testing Plan

### Unit / integration tests

- Metastore roundtrip includes WARC rows/maps.
- MetaManager refresh persists WARC records and serves them.
- Server handlers:
  - list/detail/action endpoints
  - delete target handling
  - invalid action/params
- Existing dashboard lifecycle integration test extended for WARC routes.

### Runtime verification

Start dashboard:

- `search cc fts dashboard`

Verify:

- WARC tab renders and loads cached metadata.
- WARC detail page actions create jobs and reflect progress.
- Delete/re-index updates metadata after refresh.
- Overview/search/pipeline pages continue functioning.

## 11. Rollout / Compatibility

- Schema additions are additive.
- Existing caches remain readable; new tables auto-created.
- If metastore unavailable, handlers degrade to scan-fallback where practical.

## 12. Risks and Mitigations

- Risk: large crawl manifests / WARC cardinality.
  - Mitigation: bounded API responses and optional filtering.
- Risk: stale cache after direct filesystem deletes.
  - Mitigation: force refresh after destructive actions.
- Risk: UI complexity.
  - Mitigation: keep action flows explicit and status-forward.
