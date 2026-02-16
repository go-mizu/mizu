# 0546 TAOCP PDF -> LaTeX/Markdown Conversion Spec

## Objective
Create a reproducible `uv`-based Python pipeline that converts the TAOCP source PDF into:

- one image per page in `$HOME/data/taocp/images/*`
- one Markdown file per page plus `main.md` in `$HOME/data/taocp/markdown/*`
- one LaTeX file per page plus `main.tex` in `$HOME/data/taocp/latex/*`

Input PDF:

- `$HOME/data/taocp/raw/vol_1.pdf`

## User Requirements
1. Pipeline must be implemented in Python and runnable with `uv`.
2. First step must extract pages as images (one page => one image).
3. Must produce both Markdown and LaTeX outputs.
4. LaTeX output must include a root `main.tex` that includes all per-page `.tex` files.
5. Add a detailed technical spec at `spec/0546_taocp.md`.
6. Implement and run tests to validate output correctness.
7. Improve quality by integrating `marker` and `surya` libraries.

## Non-Goals
1. Perfect semantic reconstruction of all equations/tables from TAOCP.
2. A single monolithic `.tex` containing full book-level structure reconstruction.
3. Reflowed chapter-level authoring quality edits.

## Solution Overview
A CLI tool `taocp-convert` is provided with a conversion pipeline:

1. Validate input and page-range settings.
2. Extract text using configurable engine strategy:
   - `surya` (default high-quality OCR)
   - `marker` (optional high-quality document conversion)
   - local text layer extraction (PyMuPDF)
   - local OCR fallback on macOS (`ocrmac`) when text is missing or forced
3. Render each selected page to PNG (one image per page).
4. Write per-page Markdown and per-page LaTeX files.
5. Generate `main.md` and `main.tex` index/entry files.

## Engine Strategy
Supported `--engine` values:

1. `surya` (default): use Surya OCR pipeline first.
2. `marker`: use Marker only for high-quality text extraction; fail if unavailable.
3. `auto`: `surya` -> `marker` -> text layer (+ optional local OCR)
4. `text`: skip marker/surya and use local extraction only.

### Marker Integration Details
- Command used: `marker_single`
- Mode: markdown output, paginated output enabled
- Page filtering: uses Marker `--page_range` with 0-based page indices
- OCR quality option: `--marker-force-ocr/--no-marker-force-ocr`
- Parsing: split paginated markers like `{page_id}-----` into per-page text
- Resilience: on accelerator/MPS failure, automatically retries with `TORCH_DEVICE=cpu`

### Surya Integration Details
- Command used: `surya_ocr`
- Output parsed from `results.json`
- Page filtering: `--page_range` with 0-based page indices
- Text extraction: joins `text_lines[].text` per page
- Optional `--surya-disable-math` exposed

### Local Fallback
- Text layer extraction: `page.get_text("text")`
- OCR fallback: `ocrmac` backend on macOS (`--force-ocr` or empty page text)

## Output Contract
For selected page `N` (1-based):

1. Image: `images/page_NNNN.png`
2. Markdown: `markdown/page_NNNN.md`
3. LaTeX: `latex/page_NNNN.tex`

Aggregate files:

1. `markdown/main.md` links all page Markdown files.
2. `latex/main.tex` includes all page `.tex` files via `\input{...}`.

Each page markdown includes:

- page heading
- image reference
- extraction-method metadata comment
- extracted text body
- math normalization for markdown (`<math>...</math>` to `$...$`, `<sub>/<sup>` to `_{} / ^{}`)

Each page LaTeX includes:

- section heading (`Page N`)
- embedded image (`\includegraphics`)
- escaped/normalized extracted text
- page break (`\clearpage`)

Text normalization removes non-printable control characters so corrupted PDF text-layer bytes do not leak into Markdown/LaTeX output.

## CLI Specification
Command:

```bash
uv run taocp-convert \
  --input "$HOME/data/taocp/raw/vol_1.pdf" \
  --latex-dir "$HOME/data/taocp/latex" \
  --markdown-dir "$HOME/data/taocp/markdown" \
  --images-dir "$HOME/data/taocp/images" \
  --engine surya
```

Important flags:

1. `--start-page`, `--end-page` (1-based)
2. `--engine {auto,marker,surya,text}`
3. `--marker-force-ocr/--no-marker-force-ocr`
4. `--surya-disable-math`
5. `--force-ocr` (local OCR fallback)
6. `--no-ocr` (disable local OCR backend)
7. `--silent` (reduce output; default mode is verbose and streams progress)

## Verbose Progress Design
Default mode is verbose. Use `--silent` to reduce output.

When verbose mode is active:

1. The tool logs conversion stages, page range, and per-page progress (`[i/N]`).
2. Every external command is printed before execution (`RUN[...]` with full command line).
3. `marker_single` and `surya_ocr` stdout/stderr are streamed live line-by-line.
4. Command completion is logged with exit code and duration (`DONE[...]`).
5. Per-page output summary includes extraction method and character counts.
6. Marker/Surya are run with their own `--debug` flag for library-level diagnostics.

## Project Layout

- `pyproject.toml`: package metadata, script entrypoint, dependencies
- `src/taocp_converter/converter.py`: conversion pipeline and engine integrations
- `src/taocp_converter/cli.py`: CLI interface
- `tests/test_converter.py`: unit/integration tests
- `spec/0546_taocp.md`: this spec

## Dependencies
Core:

1. `pymupdf`
2. `ocrmac` (macOS local OCR fallback)

Optional high-quality engines:

1. `marker-pdf`
2. `surya-ocr`

Install optional engines:

```bash
uv sync --extra hq
```

## Testing Strategy
### Automated tests (`pytest`)
1. Page-range compaction utility correctness.
2. Marker paginated markdown parser correctness.
3. Surya `results.json` parser correctness.
4. LaTeX escaping correctness for special characters.
5. End-to-end conversion on synthetic text PDF:
   - writes page images
   - writes per-page Markdown/LaTeX
   - writes `main.md` and `main.tex`
6. OCR fallback path correctness on empty text-layer PDF.
7. Control-character stripping in normalization.
8. Marker CPU-retry error classifier correctness.

### Smoke test on TAOCP file
Run conversion command using real TAOCP PDF and inspect:

1. expected output directories created
2. `main.tex` exists and includes page files
3. Markdown/LaTeX pages are non-empty for sampled pages
4. image-per-page files exist

## Acceptance Criteria
1. Command runs via `uv run taocp-convert`.
2. Generates image, markdown, latex outputs in requested directories.
3. Includes `latex/main.tex` that includes all page `.tex` files.
4. Tests pass.
5. Conversion summary reports extraction modes used.

## Known Tradeoffs
1. Marker and Surya are heavyweight ML dependencies; first run can be slow due model downloads.
2. `marker` paginated output format is parsed heuristically from page separators.
3. LaTeX output is page-oriented and escaped, not yet chapter-structured.

## Future Improvements
1. Add chapter/section reconstruction based on layout metadata.
2. Preserve table/equation semantics in dedicated LaTeX environments.
3. Add retry/fallback diagnostics persisted as JSON manifest.
4. Add parallel rendering and OCR for large PDFs.
