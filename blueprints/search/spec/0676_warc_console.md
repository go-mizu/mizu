# 0676 WARC Console & Header Cleanup

## Header Issues

### H-1: Rename "FTS Dashboard" to "OpenIndex" with logo
- Title says "FTS Dashboard" in header and `<title>` — rename to "OpenIndex"
- Add a simple inline SVG logo (Lucide-inspired search/index icon)

### H-2: Header right side clutter — "Refresh Meta" button + meta chip
- `header-meta-refresh` button duplicates refresh functionality on every page
- `header-meta` chip shows raw technical detail (`meta:sqlite · updated:3m ago · stale:no`)
- These are noisy, duplicated across Overview, WARC, Browse page headers
- **Fix**: Remove both. Show a compact status dot (green/amber) + relative time only.

## WARC List Issues (`#/warc`)

### W-1: Pipeline funnel uses stacked bar for sequential phases
- Stacked bar misrepresents the data: these are nested subsets (Downloaded ⊃ Markdown ⊃ Packed ⊃ Indexed), not independent categories
- **Fix**: Use a horizontal funnel/waterfall with arrows, similar to overview pipeline

### W-2: Table has too many columns, poor readability
- "Filename" column is mostly truncated and wastes space
- ".warc.gz" and ".md.warc.gz" size columns are rarely useful in list view
- "Docs" column shows "—" for most rows
- **Fix**: Remove filename/sizes from list; show phases + progress + total size only.
  Move details to detail view.

### W-3: Action buttons (dl/md/pk/ix) are cryptic abbreviations
- `dl`, `md`, `pk`, `ix` are not intuitive
- **Fix**: Use full words or icon+tooltip. In list just show the primary "next step" button.

### W-4: "Refresh Metadata" + "Reload" buttons duplicate
- "Refresh Metadata" (forces meta cache refresh) and "Reload" (re-fetch list) are separate
- **Fix**: Single "Reload" button. Move metadata refresh to overview.

### W-5: "warc-meta" line shows raw technical details
- Same as H-2. Redundant with header.
- **Fix**: Remove. Pipeline funnel already shows all relevant status.

### W-6: Batch "Run next step (all)" is dangerous
- Fires N simultaneous jobs with no throttling or confirmation
- **Fix**: Show count + confirm dialog.

### W-7: No summary statistics beyond funnel
- Missing: total disk usage breakdown, avg WARC size, completion ETA
- **Fix**: Add stat cards row above funnel.

## WARC Detail Issues (`#/warc/00000`)

### D-1: "Refresh" button calls refreshDashboardMeta, not detail reload
- First button triggers meta cache refresh, second reloads detail. Confusing.
- **Fix**: Single "Reload" button.

### D-2: warcActionMessage persists across navigations
- Global `warcActionMessage` variable leaks stale messages into the detail view
- **Fix**: Clear on entry.

### D-3: Info cards show wrong/misleading doc count
- Uses `w.markdown_docs` (deprecated field) instead of `w.warc_md_docs`
- **Fix**: Use warc_md_docs.

### D-4: Actions section is cluttered with rarely-used options
- "Extract Markdown --fast" is niche
- Pack format dropdown (parquet/bin/duckdb/markdown) — most users only use one
- Re-index button is rarely needed
- Danger zone is too prominent
- **Fix**: Simplify to primary flow. Collapse advanced options.

### D-5: Donut chart too small, no hover interaction
- 80x80px donut is hard to read
- **Fix**: Larger donut (100px), add segment hover labels.

### D-6: Phase breakdown bars use generic renderBars
- Pack formats and FTS engines show as simple bar charts — no color coding
- **Fix**: Use colored bars matching phase colors.

### D-7: No disk size trend / history
- Would be useful to show how much disk each phase consumes relative to raw WARC
- **Fix**: Add compression ratio stat cards.

## Implementation Plan

### Phase 1: Header
1. Rename to OpenIndex + SVG logo
2. Remove header-meta chip + refresh button, add compact status indicator

### Phase 2: WARC List
3. Redesign pipeline funnel as waterfall
4. Simplify table: index, phases, docs, size, next action
5. Single reload button, remove meta line
6. Add stat cards
7. Confirm batch operations

### Phase 3: WARC Detail
8. Single reload button, fix doc count field
9. Clear stale action messages
10. Simplify actions into primary flow + collapsible advanced
11. Larger donut, compression ratios

## Files Changed

- `pkg/index/web/static/index.html` — header rename + logo + remove meta widgets
- `pkg/index/web/static/js/warc.js` — list + detail redesign
- `pkg/index/web/static/js/overview.js` — remove duplicate refresh button
- `pkg/index/web/static/js/init.js` — remove meta watchdog header updates
- `pkg/index/web/static/js/utils.js` — remove header meta functions
