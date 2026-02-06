# FineWeb-2 Analytics Dashboard

Interactive analytics dashboard for exploring FineWeb-2 Vietnamese web corpus data, powered by DuckDB and Plotly.

## Quick Start

```bash
# 1. Download data with the CLI
search download get --lang vie_Latn --split test
search download get --lang vie_Latn --split train --shards 2

# 2. Import to DuckDB + compute analytics cache
uv run app.py --setup

# 3. Launch the dashboard
uv run app.py
```

No virtual environment or `pip install` needed. The PEP 723 inline metadata in `app.py` tells `uv` exactly which packages to install.

## CLI Download Commands

The `search download` CLI manages HuggingFace dataset downloads:

```bash
# List all 1300+ languages in FineWeb-2
search download langs
search download langs --search vie

# Show dataset size and statistics
search download info --lang vie_Latn

# List parquet files for a language
search download files --lang vie_Latn --split train

# Download with progress bar and speed
search download get --lang vie_Latn --split train --shards 2
search download get --lang vie_Latn --split test
```

## Features

- **10 category tabs** with **50 interactive charts**
- **Pre-computed analytics**: JSON cache for instant dashboard loading
- **Separate DuckDB per split**: `train.duckdb` and `test.duckdb`
- **PDF export**: 61-page report with category dividers (saves to `~/data/fineweb-2/`)
- **Pipeline timing**: Detailed step-by-step timing reports
- **Google-style design**: Clean color palette, Inter font, white backgrounds

## Categories

| # | Category | Charts | Description |
|---|----------|--------|-------------|
| 1 | Overview | 5 | Dataset scale, field completeness, distributions |
| 2 | Text Length | 5 | Length violin, buckets, percentiles, short/long docs |
| 3 | Vocabulary | 5 | Top words, character types, word length, stop words |
| 4 | Temporal | 5 | Yearly/monthly trends, heatmap, dumps, hours |
| 5 | Domains | 5 | Top domains, concentration, quality, size distribution |
| 6 | URL & Web | 5 | TLD, protocol, URL depth/length, query strings |
| 7 | Quality | 5 | Quality bands, score by length/year, percentiles |
| 8 | Deduplication | 5 | Cluster sizes, categories, duplicated domains |
| 9 | Vietnamese | 5 | Tones, diacritics, vowels, char ratio, function words |
| 10 | Content | 5 | Content types, landscape, quality, news, e-commerce |

## App CLI Options

```
usage: app.py [-h] [--data-dir DIR] [--lang LANG] [--port PORT]
              [--share] [--setup] [--import-db] [--compute-cache]
              [--export-pdf] [--split SPLIT]

options:
  --data-dir DIR      Path to data directory (default: ~/data/fineweb-2)
  --lang LANG         Language code (default: vie_Latn)
  --port PORT         Port number (default: 7860)
  --share             Create public Gradio link
  --setup             Import parquet to DuckDB + compute analytics cache
  --import-db         Import parquet files into DuckDB only
  --compute-cache     Compute JSON analytics cache only
  --export-pdf        Export PDF report
  --split SPLIT       Split for PDF export: train or test (default: test)
```

## PDF Export

Generate a standalone 61-page PDF report with all 50 charts:

```bash
# Export to ~/data/fineweb-2/
uv run app.py --export-pdf --split test
uv run app.py --export-pdf --split train
```

The PDF includes:
- Title page with overview stats and category index
- Category divider pages with blue accent bars
- All 50 charts at high resolution (2x scale)

## Data Layout

```
~/data/fineweb-2/
  vie_Latn/
    train.duckdb                  # Train database (from --import-db)
    test.duckdb                   # Test database
    .cache_train.json             # Pre-computed analytics (from --compute-cache)
    .cache_test.json
    train/
      000_00000.parquet           # Training data (~4.5 GB each)
      000_00001.parquet
    test/
      000_00000.parquet           # Test data (~59 MB)
  fineweb2_vie_Latn_test_*.pdf    # PDF reports (from --export-pdf)
```

## Architecture

```
app.py (single-file, PEP 723 inline deps)
  ├── Pipeline                     # Step timing and reporting
  ├── import_to_duckdb()           # Parquet → DuckDB with derived columns + indexes
  ├── compute_cache()              # Run 49 DuckDB queries → JSON cache
  ├── _all_queries()               # 49 SQL queries across 10 categories
  ├── build_all_charts()           # 50 Plotly figures from cached data
  ├── export_pdf()                 # 61-page PDF with fpdf2 + kaleido
  ├── create_app()                 # Gradio Blocks UI (reads JSON cache only)
  └── run_setup()                  # Import + cache pipeline
```

## Pip Alternative

```bash
pip install -r requirements.txt
python app.py --setup
python app.py
```
