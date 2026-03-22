# 0773 — Enhanced File Preview

## Objective

Make the dashboard preview render rich content for as many file formats as
possible, using open-source client-side libraries lazy-loaded from CDN.

## Current State

| Format | fileType | Preview | Notes |
|--------|----------|---------|-------|
| PNG/JPG/GIF/SVG/WebP/BMP/ICO | image | `<img>` tag | Works well |
| MP4/WebM/MOV/AVI/MKV | video | Custom HTML5 player | Play/pause, scrub, fullscreen |
| MP3/WAV/OGG/FLAC/AAC/M4A | audio | Custom HTML5 player | Waveform viz, volume |
| PDF | doc | `<iframe>` | Browser-native viewer |
| Markdown (.md/.mdx) | markdown | Rendered HTML + KaTeX | Preview/source toggle |
| Code (40+ extensions) | code | Syntax-highlighted `<pre>` | 19 language grammars |
| CSV/TSV | sheet | HTML table via csvToTable() | Works well |
| Plain text (.txt/.log/.rst) | text | `<pre>` | HTML-escaped |
| **DOCX** | doc | **Generic download** | No rendering |
| **XLSX/XLS** | sheet | **Garbled text** | Binary fetched as text |
| **PPTX** | doc | **Generic download** | No rendering |
| **ODS** | sheet | **Garbled text** | Binary fetched as text |
| Archives (zip/tar/gz/rar/7z) | archive | Generic download | — |

**Key problems:**
1. DOCX files show only a download button — no inline rendering
2. XLSX/XLS/ODS are fetched as text (garbled binary) instead of rendered as tables
3. PPTX files show only a download button

## Target State

| Format | Library | CDN Size | Preview |
|--------|---------|----------|---------|
| DOCX | Mammoth.js 1.8 | ~55 KB | Rendered HTML (headings, lists, tables, images, bold/italic) |
| XLSX/XLS/ODS | SheetJS 0.20 | ~330 KB | Interactive table with sheet tab bar |
| PPTX | — | — | Enhanced info card with file metadata + download |
| PDF | (keep iframe) | 0 | Browser-native PDF viewer (no change) |
| CSV/TSV | (keep csvToTable) | 0 | Existing table rendering (no change) |
| All others | (no change) | — | Existing rich previews |

**Loading strategy:** Libraries are lazy-loaded on first use — zero cost to
initial page load. A `loadScript(url, globalName)` utility ensures each
library is loaded at most once.

## Libraries

### Mammoth.js 1.8 — DOCX rendering

- **CDN:** `https://cdn.jsdelivr.net/npm/mammoth@1.8.0/mammoth.browser.min.js`
- **License:** BSD-2-Clause (free for commercial use)
- **How it works:** Parses DOCX (Open XML) ArrayBuffer → semantic HTML
- **Supports:** headings, paragraphs, lists, tables, bold, italic, underline,
  strikethrough, links, images (embedded as data URIs), footnotes
- **Does NOT support:** .doc (old binary format), page layout, headers/footers,
  complex table merging, text boxes

### SheetJS 0.20 — XLSX/XLS/ODS rendering

- **CDN:** `https://cdn.sheetjs.com/xlsx-0.20.3/package/dist/xlsx.full.min.js`
- **License:** Apache 2.0
- **How it works:** Reads binary spreadsheet → in-memory workbook → HTML table
- **Supports:** XLSX, XLS, ODS, numbers, text, dates, formulas (computed values),
  multiple sheets, merged cells
- **API:** `XLSX.read(arrayBuffer, {type:'array'})` → `XLSX.utils.sheet_to_html(ws)`

### Why not PDF.js?

The native browser `<iframe>` PDF viewer works in all modern desktop browsers
and most mobile browsers. PDF.js would add ~1 MB (pdf.js + pdf.worker.js)
for marginal improvement. Not justified for this use case.

### Why not a PPTX renderer?

No production-quality client-side PPTX renderer exists on CDN. The `pptx-preview`
npm package is unmaintained and unreliable. Rendering DrawingML (the XML schema
used for PowerPoint slides) requires implementing a significant subset of the
OOXML specification. Instead, we show a clean info card with download.

## Implementation

### 1. Lazy Script Loader

```js
var _loadingScripts = {};
function loadScript(url, globalName) {
  if (window[globalName]) return Promise.resolve();
  if (_loadingScripts[url]) return _loadingScripts[url];
  _loadingScripts[url] = new Promise(function(resolve, reject) {
    var s = document.createElement('script');
    s.src = url;
    s.onload = function() { delete _loadingScripts[url]; resolve(); };
    s.onerror = function() { delete _loadingScripts[url]; reject(); };
    document.head.appendChild(s);
  });
  return _loadingScripts[url];
}
```

### 2. openPreview changes

Skip text-fetch for binary spreadsheet formats to avoid garbled content:

```js
// Before the text-fetch block:
var isBinarySheet = ft === 'sheet' && !/\.(csv|tsv)$/i.test(item.name);
if (isBinarySheet) { renderPreview(); return; }
```

### 3. renderPreview new branches

**DOCX** (checked before the generic `doc` fallback):

```
ft === 'doc' && /\.docx$/i.test(item.name)
```

1. Render a loading spinner inside `.preview-docx`
2. `resolveFileUrl()` → `fetch()` as ArrayBuffer
3. `loadScript(mammothUrl, 'mammoth')` (parallel with fetch)
4. `mammoth.convertToHtml({arrayBuffer})` → inject `.value` into container
5. Reuse `.preview-md` styling (headings, lists, tables, links)

**XLSX/XLS/ODS** (checked before text-based `sheet` rendering):

```
ft === 'sheet' && S.previewContent === null
```

1. Render a loading spinner inside `.preview-xlsx`
2. `resolveFileUrl()` → `fetch()` as ArrayBuffer
3. `loadScript(sheetjsUrl, 'XLSX')` (parallel with fetch)
4. `XLSX.read(data, {type:'array'})` → workbook
5. If multiple sheets, render a tab bar (`SheetNames`)
6. `XLSX.utils.sheet_to_html(activeSheet)` → inject into container

### 4. CSS additions

- `.preview-docx` — inherits from `.preview-md` typography
- `.preview-xlsx` — table styles with proper borders and alternating rows
- `.xlsx-tabs` — horizontal tab bar for sheet navigation
- `.xlsx-tab`, `.xlsx-tab--active` — individual tab buttons

### 5. Error handling

If library fails to load or file is corrupted:
- Show a clean error message inside the preview container
- Include a download button as fallback
- Never leave the UI in a broken state

## Test Files

Upload to `_preview_test/` folder in test@liteio.dev:

| File | Format | Content |
|------|--------|---------|
| document.docx | DOCX | Heading, paragraphs, list, table, bold/italic, link |
| spreadsheet.xlsx | XLSX | 2 sheets: "Sales" with numbers, "Notes" with text |
| presentation.pptx | PPTX | Title slide (tests graceful fallback) |
| data.csv | CSV | Already supported — verify still works |
| photo.jpg | JPEG | Sample image |
| notes.md | Markdown | Math formulas + headings (verify KaTeX) |
| sample.pdf | PDF | Multi-page doc (verify iframe) |

## Test Plan

- [ ] DOCX renders headings, paragraphs, lists, tables, bold/italic
- [ ] DOCX images render as inline data URIs
- [ ] XLSX renders first sheet as HTML table
- [ ] XLSX sheet tabs switch between sheets
- [ ] XLSX with single sheet shows no tab bar
- [ ] XLS (legacy format) also renders via SheetJS
- [ ] CSV/TSV still uses existing csvToTable (no regression)
- [ ] PPTX shows enhanced info card with download
- [ ] .doc (old Word) shows enhanced info card with download
- [ ] Libraries lazy-load only on first use (check Network tab)
- [ ] Second open of DOCX/XLSX is instant (library cached)
- [ ] Corrupted files show error message with download fallback
- [ ] Preview navigation (arrow keys) works across all formats
- [ ] Copy button hidden for binary formats (no raw text to copy)
- [ ] Dark theme looks correct for all new preview types
