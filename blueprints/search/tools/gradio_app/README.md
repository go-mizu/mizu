# FineWeb-2 Analytics Dashboard

Interactive analytics dashboard for exploring FineWeb-2 Vietnamese web corpus data, powered by DuckDB and Plotly.

## Quick Start (uv)

```bash
# Install uv if you don't have it
curl -LsSf https://astral.sh/uv/install.sh | sh

# Run directly - uv reads inline script metadata and installs deps automatically
uv run app.py

# Or with options
uv run app.py --data-dir ~/data/fineweb-2 --lang vie_Latn --port 7860
```

No virtual environment or `pip install` needed. The PEP 723 inline metadata in `app.py` tells `uv` exactly which packages to install.

## Quick Start (pip)

```bash
pip install -r requirements.txt
python app.py
```

## DuckDB Database Import

For much faster queries (especially on the 2.3M-row train split), import parquet data into a persistent DuckDB database first:

```bash
# Import parquet files → ~/data/fineweb-2/vie_Latn/analytics.duckdb
uv run app.py --import-db

# Then launch the dashboard (auto-detects the .duckdb file)
uv run app.py
```

The import pre-computes derived columns (text_len, word_count, host, year, month, etc.) and creates indexes, making all subsequent queries significantly faster.

## Features

- **6 interactive tabs**: Overview, Text Statistics, Temporal Analysis, Domains & URLs, Quality Metrics, Vietnamese Content
- **21 interactive Plotly charts**: Zoom, pan, hover for details
- **Real-time filtering**: Switch between train/test splits
- **DuckDB backend**: Persistent database or direct parquet queries
- **PDF export**: 22-page report with all charts (from CLI or in-app button)
- **Google-style design**: Clean color palette, Inter font, white backgrounds

## Tabs

| Tab | Charts | Description |
|-----|--------|-------------|
| Overview | Summary metrics, train vs test comparison | Key dataset statistics at a glance |
| Text Statistics | Length histogram, word count, top words, char types | Document-level text analysis |
| Temporal Analysis | Yearly/monthly trends, top dumps, hour/DOW | Time-based crawl patterns |
| Domains & URLs | Top domains, TLD distribution, URL depth | Source website analysis |
| Quality Metrics | Language scores, cluster sizes, quality bands | Data quality assessment |
| Vietnamese Content | Tone distribution, content type classification | Vietnamese-specific analysis |

## CLI Options

```
usage: app.py [-h] [--data-dir DIR] [--lang LANG] [--port PORT]
              [--share] [--import-db] [--export-pdf] [--split SPLIT]

options:
  --data-dir DIR   Path to FineWeb-2 data directory (default: ~/data/fineweb-2)
  --lang LANG      Language code (default: vie_Latn)
  --port PORT      Port number (default: 7860)
  --share          Create public Gradio link
  --import-db      Import parquet files into DuckDB database and exit
  --export-pdf     Export PDF report and exit (no server)
  --split SPLIT    Split for PDF export: train or test (default: test)
```

## PDF Export

Generate a standalone PDF report with all 21 charts:

```bash
# Export PDF for test split
uv run app.py --export-pdf --split test

# Export PDF for train split
uv run app.py --export-pdf --split train

# Or click "Export PDF Report" button in the running dashboard
```

## Data Layout

```
~/data/fineweb-2/
  vie_Latn/
    analytics.duckdb           # Persistent database (created by --import-db)
    train/000_00000.parquet    # Training data (~2.3M rows, 4.5GB)
    test/000_00000.parquet     # Test data (~28K rows, 59MB)
```

## Architecture

```
app.py (single-file, PEP 723 inline deps)
  ├── import_parquet_to_duckdb()  # Parquet → DuckDB import with indexes
  ├── FineWebAnalytics            # DuckDB-backed analytics engine
  │   ├── __init__()              # Auto-detect .duckdb or fall back to parquet
  │   ├── overview()              # Summary statistics
  │   ├── text_*()                # Text analysis (4 charts)
  │   ├── documents_per_year()    # Temporal analysis (5 charts)
  │   ├── top_domains()           # Domain analysis (4 charts)
  │   ├── language_score_*()      # Quality metrics (5 charts)
  │   ├── tone_distribution()     # Vietnamese content (2 charts)
  │   ├── compare_splits()        # Train vs test comparison
  │   └── all_figures()           # Collect all charts for PDF
  ├── export_pdf()                # Multi-page PDF with fpdf2 + kaleido
  └── create_app()                # Gradio Blocks UI
```
