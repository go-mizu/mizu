# 0770 Dashboard v2 Review

Deep dive review of every section, identifying missing features, DX inconsistencies, and bugs.

## Bugs (must fix)

### B1. Shares TTL modal offers 30d but backend caps at 7d
`MAX_TTL = 7 * 86400` in files-v2.ts:118. The share TTL modal offers "30 days" which silently gets capped to 7 days. User thinks they created a 30d link but it expires in 7d.
**Fix**: Either raise MAX_TTL to 30d or remove the 30d option from the modal.

### B2. Modal confirm button always styled as destructive (red)
Rename, create folder, create share all use `.modal-confirm` which is red with red border. Red = danger. Non-destructive actions should use primary style.
**Fix**: Add `.modal-confirm--primary` class, use it for non-destructive modals.

### B3. Shares "Create Share" requires typing path manually
No autocomplete or file picker. Users must know the exact path. Error-prone.
**Fix**: Add search-as-you-type to the share path input using `/files/search`.

### B4. Upload modal URL button has no label
Just shows a download icon (IC.download) with no text. Looks like a download button, not "upload from URL". Confusing.
**Fix**: Add text label "Fetch" or "Import".

### B5. Preview doesn't update URL hash
Opening a file preview keeps `#files/` in the URL. Can't share a direct link to a preview. Back button behavior also inconsistent.
**Fix**: Update hash to `#files/path/to/file` on preview open.

### B6. Keyboard shortcut `g h` for shares not in shortcuts modal
The shortcuts help modal lists g+o/f/e/a/k/s but not g+h for shares.
**Fix**: Add it.

## DX Inconsistencies

### D1. Overview "Recent Activity" uses table, Events uses compact rows
Overview renders events as a `d-table` with columns. Events section uses the compact `.ev-row` layout. Same data, different presentation.
**Fix**: Use compact ev-row layout in Overview too.

### D2. Share from file browser has no TTL selection
Right-click > Share or the share button always uses 24h TTL (`ttl: 86400` hardcoded in fbShare). The Shares section's create dialog lets you pick 1h/1d/7d/30d.
**Fix**: Show TTL picker when sharing from file browser too, or at minimum mention the TTL in the success modal.

### D3. No loading/disabled state on action buttons
Delete, revoke, create key - none disable the button or show loading while the API call is in flight. Double-clicking fires duplicate requests.
**Fix**: Disable button + show spinner during async operations.

### D4. Events section has no pagination
Hard-capped at 200 events with no "load more" button. Audit log does have load more.
**Fix**: Add load more to events, same pattern as audit.

### D5. Files section has no pagination
Limited to 500 files with no indication if truncated.
**Fix**: Show "load more" when truncated=true.

### D6. No clickable paths in Events
Event paths are display-only. Clicking should navigate to the file.
**Fix**: Make ev-path clickable, navigate to file preview.

## Missing Features

### M1. No PDF preview
PDF files detected as "doc" type fall through to generic download view. Should render in iframe.
**Fix**: Add iframe-based PDF preview.

### M2. No copy button for code/text preview
Can view code with syntax highlighting but can't copy content to clipboard.
**Fix**: Add copy button to preview toolbar.

### M3. No file sort
Can't sort file list by name, size, or modified date. Headers look clickable but aren't.
**Fix**: Make column headers clickable to toggle sort.

### M4. No bulk file operations
Can't select multiple files for delete or download.
**Defer**: Complex feature, not critical for v2.

### M5. No storage usage in Overview
Overview shows file count and size but doesn't show quota context.
**Defer**: No quota system yet.

### M6. Overview stat cards not clickable
"Files: 42" should navigate to Files section. "API Keys: 3" should navigate to Keys.
**Fix**: Make stat cards clickable.

## Priority for implementation

**Do now** (ship-blocking or low-effort high-impact):
1. B1 - Fix TTL mismatch (raise to 30d)
2. B2 - Non-destructive modal style
3. B4 - Upload URL button label
4. B5 - Preview hash update
5. B6 - Shortcuts modal missing shares
6. D1 - Overview use compact ev-row
7. D2 - TTL picker on file share
8. D4 - Events pagination
9. D5 - Files pagination
10. D6 - Clickable paths in events
11. M1 - PDF preview
12. M2 - Copy button for code preview
13. M3 - File sort
14. M6 - Clickable stat cards
15. B3 - Share path autocomplete
16. D3 - Button loading states
