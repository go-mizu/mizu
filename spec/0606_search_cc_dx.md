# 0606 — `search cc` CLI DX Refresh

## Goal
Improve the developer experience of the `search cc` command family with a focus on:
- readable, scan-friendly terminal output (especially `cc parquet list`)
- consistent help text and examples across `cc` subcommands
- clearer selectors/defaults guidance (`--crawl`, `--subset`, `--file` selector forms)
- better visual hierarchy using Lipgloss table/cards/components

## Why this matters
Current output is functionally correct but hard to scan in large manifests and multi-step workflows:
- `cc parquet list` uses wide `fmt.Printf` rows with long paths, making columns hard to follow.
- Several `cc` subcommands mix styles and help patterns (some have examples in `Long`, others do not).
- Default behavior (latest crawl, default `subset=warc`) is not consistently surfaced in output/help.

## Design Principles (inspired by Lipgloss + common TUI patterns)
Use patterns common in popular terminal apps (Charmbracelet ecosystem, modern TUIs):
- **Progressive disclosure**: summary first, details below.
- **Visual grouping**: cards/sections for defaults, summary, results, hints.
- **Dense but legible tables**: zebra rows, clear headers, compact status chips.
- **Action-oriented hints**: put “what to run next” and selector cheat sheet near list output.
- **Stable columns**: predictable widths + truncation for long paths/file names.
- **Color as secondary signal**: output remains understandable without color.

## Scope
### Primary (must ship)
1. `cc parquet list` output redesign
   - Replace raw `fmt.Printf` table with Lipgloss table.
   - Add summary card (crawl, subset, entries, local cache count/size, elapsed).
   - Add subset breakdown table (count + percentage).
   - Improve row readability (selectors, local status chip, compact path/name formatting).
   - Add selector cheat sheet panel.
   - Keep `--names-only` for scripting/plain output.

2. Shared CC DX helpers
   - Create reusable helpers for:
     - section headers
     - key/value summary cards
     - styled tables
     - hint/cheat-sheet panels
     - status chips (`cached`, `missing`, `indexed`, etc.)
   - Reuse existing `cli/ui.go` styles to avoid a second theme system.

3. Review and improve `search cc` help DX (all core subcommands)
   - Standardize examples (`Example:` field) for major commands.
   - Clarify defaults and selector syntax in help text.
   - Remove stale examples/default crawl references where possible.

### Secondary (nice-to-have if low risk)
1. Improve `cc crawls` list rendering with the same table/card style.
2. Add compact summary banners to commands that list/search (`cc stats`, `cc query`).
3. Add `--wide` or width tuning for `cc parquet list` if terminal width becomes a limitation.

## Non-goals (this change)
- No changes to parquet download/import behavior.
- No changes to recrawl crawling semantics/performance.
- No interactive TUI mode (this remains line-oriented CLI output).

## Implementation Plan
### Phase 1: Shared rendering helpers
- Add a new `cli/cc_dx.go` module with:
  - `ccSection(title string) string`
  - `ccHintBox(title string, lines []string) string`
  - `ccKVCard(title string, rows [][2]string) string`
  - `ccTable(...) string` wrapper around `lipgloss/table`
  - `ccStatusChip(kind string, text string) string`
  - text truncation helpers (`trimMiddle`, path-aware trimming if not already available)
- Use bounded widths and avoid terminal width dependency for first iteration.

### Phase 2: `cc parquet list` redesign
- Refactor `runCCParquetList` rendering into logical blocks:
  1. banner/title/defaults
  2. manifest load summary
  3. summary card (entries/local cache stats)
  4. subset counts table
  5. main manifest table (limited rows if `--limit`)
  6. selector tips box + “showing X/Y” footer
- Preserve `--names-only` behavior for scripts.
- Ensure output remains readable with `subset=all` and long remote paths.

### Phase 3: Help/example DX pass across `cc`
- Review all major `cc` commands and update `Short`/`Long`/`Example` for consistency:
  - `cc`, `crawls`, `parquet`, `parquet list/download/import`, `index`, `stats`, `query`, `fetch`, `recrawl`, `verify`, `url`, `warc`
- Ensure examples reflect current selector forms and default behaviors.
- Highlight latest-crawl and subset defaults where relevant.

### Phase 4: Optional `cc crawls` list styling
- Replace plain table with Lipgloss table if low-risk after Phase 2.
- Add summary card (count shown, filter, cache source).

## Validation Plan
1. `gofmt` all touched files.
2. `go test ./cli ./pkg/cc`.
3. Smoke checks:
   - `search cc --help`
   - `search cc parquet list --limit 5`
   - `search cc parquet list --subset all --limit 10`
   - `search cc crawls --limit 10`
   - `search cc recrawl --help`
4. Verify `--names-only` output remains simple and script-friendly.

## Success Criteria
- `cc parquet list` can be scanned quickly without horizontal eye strain.
- Users can clearly identify which selector to use (`N`, `w:N`, `m:N`).
- Help output across `search cc` feels consistent and modern.
- No regressions in command behavior or scripting output modes.
