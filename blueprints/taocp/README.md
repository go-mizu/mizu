# taocp-converter

Convert a book PDF into one image per page, Markdown files, and LaTeX files.

For higher-quality OCR/text extraction, install optional engines:

```bash
uv sync --extra hq
```

Run:

```bash
uv run taocp-convert --input "$HOME/data/taocp/raw/vol_1.pdf" \
  --latex-dir "$HOME/data/taocp/latex" \
  --markdown-dir "$HOME/data/taocp/markdown" \
  --images-dir "$HOME/data/taocp/images" \
  --engine auto
```
