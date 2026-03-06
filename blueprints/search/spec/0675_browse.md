# 0675 Browse Shards — Bug Fixes & UX Improvements

## Context

The Browse Shards page (`#/browse/{shard}`) in the FTS dashboard (`search cc fts dashboard`)
has several broken features, missing information, and poor responsive behavior.

## Issues Found

### BUG-1: Full page re-render on every shard click (navigation resets state)

**Problem:** Clicking a shard in the sidebar triggers `navigateTo('/browse/' + shard)` which
fires `hashchange` → `route()` → `renderBrowse(shard)`. `renderBrowse()` rebuilds the entire
page shell (sidebar + content), re-fetches the shard list from the API, and resets all browse
state (`browseQ`, `browseSort`, `browsePage`). This causes:

- Visible full-page flash on every shard click
- Loss of filter/sort state when switching shards
- Unnecessary `/api/browse` fetch to reload the sidebar (shards don't change between clicks)
- Animation replay on every click

**Fix:** Separate initial page render from shard switching. When browseShard changes but
currentPage is already 'browse', skip the full re-render — just update the sidebar active
state and load the new shard content. Only reset filter `q` (sort is a user preference).

### BUG-2: Empty state when docs are returned but `docs` array is null

**Problem:** If the server returns `{"docs": null}` instead of `{"docs": []}` (Go nil slice
serializes as `null`), `data.docs || []` works in `renderDocTable` but `total` is 0 and the
pagination shows "1–0 of 0" with an empty table. The `start-end of total` range becomes `1-0`.

**Fix:** Guard the range display: if `total === 0`, show "No documents found" message instead
of the broken range.

### BUG-3: Scanning status not shown in docs view

**Problem:** When `data.scanning` is true, the docs view shows documents normally but gives no
indication that a scan is in progress and more documents may appear. The `scanning` field in
`BrowseDocsResponse` is returned but never rendered in `renderDocTable()`.

**Fix:** Show a scanning indicator banner when `data.scanning` is true.

### BUG-4: Sidebar doesn't highlight active shard on initial load via URL

**Problem:** When navigating directly to `#/browse/00005`, the sidebar loads and renders all
shards. The `renderShardList(shard)` call uses the shard from the URL as the `active` param,
which correctly applies `shard-active` class. However, if the shard doesn't exist in the list
(typo, deleted), no error is shown — it just silently loads nothing.

**Fix:** Show a "Shard not found" error when the URL shard doesn't exist in the shard list.

### BUG-5: Doc table title/URL truncation breaks on mobile (responsive)

**Problem:** The doc table has fixed `max-w-xs` (20rem) on title column and `max-w-[240px]` on
URL column. On mobile these are too wide, causing horizontal scroll. The table doesn't adapt
to narrow viewports. The sidebar `w-56` (14rem) is also fixed and takes too much space on mobile.

**Fix:**
- Hide sidebar on mobile, show as a dropdown/collapsible
- Make table columns responsive with `hidden sm:table-cell` on less important columns
- Use fluid widths instead of fixed `max-w-*`

### BUG-6: Browse page title is generic "Browse" — should show shard name

**Problem:** The page title always shows "Browse" regardless of which shard is selected.
The user has no clear indication of which shard they're viewing except the sidebar highlight.

**Fix:** Show shard name in the content header area.

### BUG-7: Stats view DateFrom/DateTo shows raw ISO date strings

**Problem:** `ShardStats` returns `DateFrom` and `DateTo` as raw SQL strings from DuckDB
(e.g. `2024-01-15T00:00:00Z`). The `fmtDate()` function handles these correctly, but if
the DuckDB field contains just `2024-01-15` (from `LEFT(crawl_date, 10)`), the date range
card in stats could display inconsistently with the doc table dates.

**Fix:** Normalize date display in stats cards.

### BUG-8: Keyboard navigation missing

**Problem:** No keyboard shortcuts for browse-specific actions:
- No `j`/`k` for next/prev doc row
- No `n`/`p` for next/prev page
- No `Escape` to go back to shard list

**Fix:** Add browse-specific keyboard handlers.

### BUG-9: browseView state persists incorrectly

**Problem:** `state.browseView` persists when switching tabs. If the user views Stats for
shard 00000, navigates to Search, then back to Browse, a different shard auto-selected but
the Stats view is shown instead of Docs. Stats view should only persist within the same
browse session.

**Fix:** Reset `browseView` to 'docs' when entering browse from a different page.

### BUG-10: No loading skeleton for shard list

**Problem:** The shard list shows a plain "loading..." text in a dashed border box while
fetching. Should show skeleton loaders matching the shard item layout.

**Fix:** Add skeleton items to the shard list during loading.

### BUG-11: Mobile header nav overflow

**Problem:** On mobile, the header nav tabs (Overview, Search, Browse, WARC, Jobs) can
overflow and the horizontal scroll has hidden scrollbar (`scrollbar-width: none`). Users
don't know they can scroll to see more tabs.

**Fix:** Already has `overflow-x: auto` — this is acceptable. But ensure Browse tab is
visible without scrolling.

## Implementation Plan

### Phase 1: Critical Fixes
1. Fix shard navigation (BUG-1): avoid full re-render on shard switch
2. Fix empty docs state (BUG-2): show "No documents" message
3. Show scanning banner (BUG-3)
4. Show shard name in content area (BUG-6)
5. Reset browseView on page entry (BUG-9)

### Phase 2: Responsive
6. Mobile sidebar (BUG-5): collapsible sidebar, responsive table
7. Shard not found error (BUG-4)

### Phase 3: Polish
8. Loading skeletons (BUG-10)
9. Keyboard navigation (BUG-8)

## Files Changed

- `pkg/index/web/static/js/browse.js` — all frontend fixes
- `pkg/index/web/static/js/router.js` — skip full re-render on shard switch
- `pkg/index/web/static/index.html` — responsive CSS additions
