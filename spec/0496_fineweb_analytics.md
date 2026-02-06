# Spec 0496: FineWeb-2 Dataset Analytics

## Overview

Comprehensive analytics package for FineWeb-2 Vietnamese web corpus data. Produces two outputs:
1. **Go CLI** — `search analytics` command that reads parquet files and generates rich Markdown reports with 55+ Mermaid/ASCII charts
2. **Python/Gradio** — Interactive dashboard app with real-time filtering, drill-down, and modern UI

## Data Source

| Split | File | Rows | Size |
|-------|------|------|------|
| train | `~/data/fineweb-2/vie_Latn/train/000_00000.parquet` | 2,319,000 | 4.51 GB |
| test | `~/data/fineweb-2/vie_Latn/test/000_00000.parquet` | 28,276 | 58.76 MB |

### Schema (11 columns)

| Column | Type | Description |
|--------|------|-------------|
| `text` | string | Document body (Vietnamese) |
| `id` | string | UUID identifier |
| `dump` | string | Common Crawl dump ID (e.g. `CC-MAIN-2013-48`) |
| `url` | string | Source URL |
| `date` | string | ISO 8601 crawl timestamp |
| `file_path` | string | S3 path to original WARC |
| `language` | string | Language code (`vie`) |
| `language_score` | float64 | Detection confidence (0-1) |
| `language_script` | string | Script (`Latn`) |
| `minhash_cluster_size` | int64 | Near-duplicate cluster size |
| `top_langs` | string | JSON with language scores |

---

## Architecture

### Package Layout

```
blueprints/search/
├── pkg/analytics/
│   ├── analyzer.go          # Main Analyzer struct, orchestrates all analysis
│   ├── report.go            # Markdown report writer
│   ├── charts.go            # Chart generation (Mermaid + ASCII)
│   ├── stats.go             # Statistical computation functions
│   ├── text_stats.go        # Text-specific analysis (length, tokens, chars)
│   ├── temporal.go          # Time-based analysis
│   ├── domain.go            # URL/domain analysis
│   ├── quality.go           # Language score & dedup analysis
│   ├── content.go           # Vietnamese content analysis
│   ├── reader.go            # Parquet reading (reuses engine/fineweb)
│   ├── types.go             # Analytics-specific types
│   └── report/
│       └── vie_Latn/
│           ├── train.md     # Generated train report
│           └── test.md      # Generated test report
├── cli/
│   └── analytics.go         # CLI command: `search analytics`
└── tools/
    └── gradio_app/
        ├── app.py           # Gradio interactive dashboard
        ├── requirements.txt # Python dependencies
        └── README.md        # Usage instructions
```

### Data Flow

```
Parquet File → ParquetReader → RawRecord → Analyzer.Collect() → Stats → ReportWriter → Markdown
                                                                                     → Gradio JSON
```

### Key Design Decisions

1. **Stream processing** — Never load all 2.3M records into memory. Single pass with accumulators.
2. **Reuse existing ParquetReader** — Import from `pkg/engine/fineweb` for reading.
3. **Extended parquet struct** — Read all 11 columns (existing reader only reads 7).
4. **Mermaid charts** — Render natively on GitHub, no image files needed.
5. **ASCII bar charts** — For distributions where Mermaid is too verbose.
6. **Progress reporting** — Show progress during long train-set analysis.

---

## CLI Command

### Usage

```bash
# Generate reports for both train and test
search analytics --lang vie_Latn

# Generate report for test only (faster, good for development)
search analytics --lang vie_Latn --split test

# Generate with custom output directory
search analytics --lang vie_Latn --output ./my-reports/

# Export raw stats as JSON (for Gradio app)
search analytics --lang vie_Latn --json > stats.json

# Limit rows processed (for quick preview)
search analytics --lang vie_Latn --limit 10000
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--lang` | `vie_Latn` | Language code |
| `--split` | `train,test` | Dataset splits to analyze |
| `--output` | `pkg/analytics/report/{lang}/` | Output directory |
| `--json` | false | Output raw statistics as JSON |
| `--limit` | 0 | Max rows to process (0=all) |
| `--workers` | runtime.NumCPU() | Parallel workers for analysis |
| `--data` | `~/data/fineweb-2` | Data directory |

---

## Chart Catalog: 55 Charts Across 5 Categories

### Category 1: Text Statistics (12 charts)

| # | Chart | Type | Description |
|---|-------|------|-------------|
| 1 | Document Length Distribution | Mermaid XY bar | Histogram of text length in characters (buckets: 0-100, 100-500, 500-1K, 1K-5K, 5K-10K, 10K-50K, 50K+) |
| 2 | Word Count Distribution | Mermaid XY bar | Histogram of word count per document |
| 3 | Sentence Count Distribution | Mermaid XY bar | Sentences per document distribution |
| 4 | Average Word Length | Mermaid XY bar | Distribution of average word length per document |
| 5 | Line Count Distribution | Mermaid XY bar | Lines per document histogram |
| 6 | Paragraph Count Distribution | Mermaid XY bar | Paragraphs per document histogram |
| 7 | Character Type Breakdown | Mermaid pie | Vietnamese diacritics vs ASCII vs digits vs punctuation vs whitespace |
| 8 | Top 30 Vietnamese Words | ASCII horizontal bar | Most frequent words (after basic tokenization) |
| 9 | Top 30 Bigrams | ASCII horizontal bar | Most frequent word pairs |
| 10 | Text Length Percentiles | Markdown table | P1, P5, P10, P25, P50, P75, P90, P95, P99 |
| 11 | Empty/Short Document Analysis | Mermaid pie | Documents <10 chars, <50 chars, <100 chars, ≥100 chars |
| 12 | Unicode Block Distribution | Mermaid pie | Latin Basic, Latin Extended, Vietnamese-specific diacritics, CJK, Other |

### Category 2: Temporal Analysis (11 charts)

| # | Chart | Type | Description |
|---|-------|------|-------------|
| 13 | Documents per Year | Mermaid XY bar | Yearly document counts from crawl dates |
| 14 | Documents per Month (all years) | Mermaid XY line | Monthly trend across all years |
| 15 | Documents per Common Crawl Dump | ASCII horizontal bar | Top 30 dumps by document count |
| 16 | Crawl Year vs Dump Distribution | Mermaid XY stacked bar | Years on X, dump counts stacked |
| 17 | Monthly Crawl Heatmap | ASCII grid | Month × Year grid with density indicators |
| 18 | Day-of-Week Distribution | Mermaid pie | Documents by day of week |
| 19 | Hour-of-Day Distribution | Mermaid XY bar | Documents by hour (UTC) |
| 20 | Earliest vs Latest Crawl Dates | Markdown table | Summary statistics |
| 21 | Year-over-Year Growth | Mermaid XY line | % growth per year |
| 22 | Dump Timeline | ASCII timeline | Chronological dump coverage |
| 23 | Quarterly Document Volume | Mermaid XY bar | Q1-Q4 aggregation per year |

### Category 3: URL & Domain Analysis (12 charts)

| # | Chart | Type | Description |
|---|-------|------|-------------|
| 24 | Top 30 Domains | ASCII horizontal bar | Most frequent domains by document count |
| 25 | TLD Distribution | Mermaid pie | .com, .vn, .org, .net, .edu, other |
| 26 | Domain Diversity per Dump | Mermaid XY line | Unique domains per crawl dump |
| 27 | URL Path Depth Distribution | Mermaid XY bar | Number of path segments histogram |
| 28 | Protocol Distribution | Mermaid pie | HTTP vs HTTPS |
| 29 | URL Length Distribution | Mermaid XY bar | URL character length histogram |
| 30 | Subdomain Analysis | Mermaid pie | www vs non-www vs other subdomains |
| 31 | Top 20 Vietnamese Domains (.vn) | ASCII horizontal bar | Vietnamese TLD breakdown |
| 32 | Domain Concentration (Lorenz) | ASCII line | Cumulative % of docs vs % of domains |
| 33 | Top Domains by Avg Text Length | ASCII horizontal bar | Which domains have longest documents |
| 34 | New Domains per Year | Mermaid XY bar | Domain first-appearance timeline |
| 35 | Query Parameter Prevalence | Mermaid pie | URLs with vs without query strings |

### Category 4: Quality & Deduplication Metrics (10 charts)

| # | Chart | Type | Description |
|---|-------|------|-------------|
| 36 | Language Score Distribution | Mermaid XY bar | Histogram of language_score values |
| 37 | Language Score Percentiles | Markdown table | P1 through P99 |
| 38 | MinHash Cluster Size Distribution | Mermaid XY bar | Histogram of cluster sizes |
| 39 | Cluster Size Categories | Mermaid pie | Unique (1), Small (2-5), Medium (6-20), Large (21-100), Very Large (100+) |
| 40 | Language Score vs Text Length | ASCII scatter (binned) | Correlation heatmap |
| 41 | Language Score vs Cluster Size | ASCII scatter (binned) | Correlation analysis |
| 42 | Low-Quality Document Analysis | Mermaid pie | Documents by score bands (<0.8, 0.8-0.9, 0.9-0.95, 0.95-0.99, 0.99-1.0) |
| 43 | Top Languages in top_langs | Mermaid XY bar | Parse JSON, show secondary language frequencies |
| 44 | Documents with Empty top_langs | Mermaid pie | Empty vs populated top_langs field |
| 45 | Cluster Size by Dump | Mermaid XY bar | Average cluster size per dump |

### Category 5: Vietnamese Content Analysis (10 charts)

| # | Chart | Type | Description |
|---|-------|------|-------------|
| 46 | Vietnamese Diacritic Frequency | ASCII horizontal bar | Frequency of each diacritic mark (á, à, ả, ã, ạ, etc.) |
| 47 | Tone Mark Distribution | Mermaid pie | 6 Vietnamese tones by frequency |
| 48 | Vietnamese vs Non-Vietnamese Characters | Mermaid pie | % of chars that are Vietnamese-specific |
| 49 | Common Vietnamese Stop Words | ASCII horizontal bar | Top 30 stop words (của, và, là, các, cho...) |
| 50 | Sentence-Ending Punctuation | Mermaid pie | Period vs question mark vs exclamation vs other |
| 51 | Digit & Number Prevalence | Mermaid XY bar | Documents by numeric content density |
| 52 | HTML/Boilerplate Residue | Mermaid pie | Documents containing HTML tags, JS code, CSS |
| 53 | Vietnamese Vowel Distribution | ASCII horizontal bar | Each of the 12 Vietnamese vowels (a, ă, â, e, ê, i, o, ô, ơ, u, ư, y) |
| 54 | Average Vietnamese Complexity Score | Mermaid XY bar | Per-dump average of diacritics/total chars ratio |
| 55 | Content Type Heuristic | Mermaid pie | News, Forum, Blog, E-commerce, Government, Other (based on URL+text patterns) |

---

## Implementation Details

### 1. Extended Parquet Record

```go
// FullRecord reads all 11 columns from FineWeb-2 parquet files
type FullRecord struct {
    Text             string  `parquet:"text"`
    ID               string  `parquet:"id"`
    Dump             string  `parquet:"dump"`
    URL              string  `parquet:"url"`
    Date             string  `parquet:"date"`
    FilePath         string  `parquet:"file_path"`
    Language         string  `parquet:"language"`
    LanguageScore    float64 `parquet:"language_score"`
    LanguageScript   string  `parquet:"language_script"`
    MinHashCluster   int64   `parquet:"minhash_cluster_size"`
    TopLangs         string  `parquet:"top_langs"`
}
```

### 2. Stream-Based Accumulator Pattern

```go
type Analyzer struct {
    // Text stats accumulators
    textLengths    *Histogram     // character length distribution
    wordCounts     *Histogram     // word count distribution
    sentenceCounts *Histogram     // sentence count distribution
    wordFreq       *TopK          // top-K word frequency
    bigramFreq     *TopK          // top-K bigrams
    charTypes      *Counter       // character type counts

    // Temporal accumulators
    yearCounts     *Counter       // documents per year
    monthCounts    *Counter       // documents per month
    dumpCounts     *Counter       // documents per dump
    hourCounts     *Counter       // documents per hour
    dowCounts      *Counter       // documents per day-of-week

    // Domain accumulators
    domainCounts   *Counter       // domain frequency
    tldCounts      *Counter       // TLD frequency
    pathDepths     *Histogram     // URL path depth distribution
    urlLengths     *Histogram     // URL length distribution

    // Quality accumulators
    langScores     *Histogram     // language score distribution
    clusterSizes   *Histogram     // minhash cluster size distribution
    topLangsCounts *Counter       // secondary language frequency

    // Vietnamese-specific
    diacriticFreq  *Counter       // diacritic mark frequency
    toneFreq       *Counter       // tone mark frequency
    vowelFreq      *Counter       // Vietnamese vowel frequency

    // Metadata
    totalDocs      int64
    processedDocs  int64
}
```

### 3. Efficient Data Structures

```go
// Histogram with configurable buckets for distributions
type Histogram struct {
    Buckets    []HistBucket  // sorted bucket boundaries
    Counts     []int64       // count per bucket
    Min, Max   float64       // running min/max
    Sum        float64       // running sum
    Count      int64         // total observations
    SumSquares float64       // for stddev calculation
    samples    []float64     // reservoir sample for percentiles (max 10000)
}

// TopK maintains top-K items by frequency using a min-heap
type TopK struct {
    K     int
    items map[string]int64
    // At flush time: extract top-K from map
}

// Counter is a simple frequency counter
type Counter struct {
    counts map[string]int64
}
```

### 4. Mermaid Chart Generation

```go
// MermaidPie generates a Mermaid pie chart
func MermaidPie(title string, data []PieSlice) string {
    var sb strings.Builder
    sb.WriteString("```mermaid\npie title " + title + "\n")
    for _, s := range data {
        sb.WriteString(fmt.Sprintf("    %q : %.1f\n", s.Label, s.Value))
    }
    sb.WriteString("```\n")
    return sb.String()
}

// MermaidXYBar generates a Mermaid XY bar chart
func MermaidXYBar(title, xLabel, yLabel string, categories []string, values []float64) string {
    var sb strings.Builder
    sb.WriteString("```mermaid\nxychart-beta\n")
    sb.WriteString(fmt.Sprintf("    title %q\n", title))
    sb.WriteString(fmt.Sprintf("    x-axis %q %s\n", xLabel, formatCategories(categories)))
    sb.WriteString(fmt.Sprintf("    y-axis %q\n", yLabel))
    sb.WriteString(fmt.Sprintf("    bar %s\n", formatValues(values)))
    sb.WriteString("```\n")
    return sb.String()
}

// MermaidXYLine generates a Mermaid XY line chart
func MermaidXYLine(title, xLabel, yLabel string, categories []string, values []float64) string {
    // Similar to bar but uses "line" keyword
}
```

### 5. ASCII Horizontal Bar Chart

```go
// ASCIIBar renders a horizontal bar chart using Unicode block chars
func ASCIIBar(title string, items []BarItem, maxWidth int) string {
    // Output:
    // Top 30 Domains
    // ──────────────────────────────────
    // vnexpress.net     ████████████████████████ 45,231
    // dantri.com.vn     ██████████████████       38,102
    // tuoitre.vn        ████████████████         31,445
    // ...
}
```

### 6. Processing Pipeline

```go
func (a *Analyzer) Run(ctx context.Context, parquetPath string, progress func(int64)) error {
    reader := NewFullReader(parquetPath, a.batchSize)

    for batch := range reader.ReadBatches(ctx) {
        for _, rec := range batch {
            a.processRecord(rec)
            a.processedDocs++
            if a.processedDocs % 10000 == 0 {
                progress(a.processedDocs)
            }
        }
    }
    return nil
}

func (a *Analyzer) processRecord(rec FullRecord) {
    // Text analysis
    a.analyzeText(rec.Text)

    // Temporal analysis
    a.analyzeDate(rec.Date, rec.Dump)

    // Domain analysis
    a.analyzeURL(rec.URL)

    // Quality analysis
    a.analyzeQuality(rec.LanguageScore, rec.MinHashCluster, rec.TopLangs)

    // Vietnamese content analysis
    a.analyzeVietnamese(rec.Text)

    a.totalDocs++
}
```

### 7. Memory Budget

For the train set (2.3M records):
- Histograms: ~1KB each × 10 = 10KB
- TopK (K=1000): ~100KB each × 4 = 400KB
- Counters: ~50KB each × 10 = 500KB
- Reservoir samples: 10K × 8 bytes × 5 = 400KB
- Word frequency map: ~50MB (capped at 100K entries)
- **Total: ~55MB** — well within limits

### 8. Text Tokenization (Vietnamese)

Vietnamese words are space-separated (like English), but each "word" may be a single syllable of a multi-syllable word. For basic analytics:

```go
func tokenize(text string) []string {
    return strings.Fields(text) // Vietnamese is space-delimited
}

func countSentences(text string) int {
    // Count sentence-ending punctuation: . ! ?
    return len(sentenceEndRe.FindAllString(text, -1))
}
```

### 9. Vietnamese Diacritic Analysis

Vietnamese uses 6 tones represented by diacritical marks:
- **Ngang** (level): no mark — a, e, o
- **Sắc** (rising): á, é, ó
- **Huyền** (falling): à, è, ò
- **Hỏi** (questioning): ả, ẻ, ỏ
- **Ngã** (tumbling): ã, ẽ, õ
- **Nặng** (heavy): ạ, ẹ, ọ

12 base vowels: a, ă, â, e, ê, i, o, ô, ơ, u, ư, y

```go
var toneMap = map[rune]string{
    'á': "sắc", 'à': "huyền", 'ả': "hỏi", 'ã': "ngã", 'ạ': "nặng",
    'ắ': "sắc", 'ằ': "huyền", 'ẳ': "hỏi", 'ẵ': "ngã", 'ặ': "nặng",
    'ấ': "sắc", 'ầ': "huyền", 'ẩ': "hỏi", 'ẫ': "ngã", 'ậ': "nặng",
    'é': "sắc", 'è': "huyền", 'ẻ': "hỏi", 'ẽ': "ngã", 'ẹ': "nặng",
    'ế': "sắc", 'ề': "huyền", 'ể': "hỏi", 'ễ': "ngã", 'ệ': "nặng",
    'í': "sắc", 'ì': "huyền", 'ỉ': "hỏi", 'ĩ': "ngã", 'ị': "nặng",
    'ó': "sắc", 'ò': "huyền", 'ỏ': "hỏi", 'õ': "ngã", 'ọ': "nặng",
    'ố': "sắc", 'ồ': "huyền", 'ổ': "hỏi", 'ỗ': "ngã", 'ộ': "nặng",
    'ớ': "sắc", 'ờ': "huyền", 'ở': "hỏi", 'ỡ': "ngã", 'ợ': "nặng",
    'ú': "sắc", 'ù': "huyền", 'ủ': "hỏi", 'ũ': "ngã", 'ụ': "nặng",
    'ứ': "sắc", 'ừ': "huyền", 'ử': "hỏi", 'ữ': "ngã", 'ự': "nặng",
    'ý': "sắc", 'ỳ': "huyền", 'ỷ': "hỏi", 'ỹ': "ngã", 'ỵ': "nặng",
}
```

---

## Markdown Report Structure

Each report (train.md, test.md) follows this structure:

```markdown
# FineWeb-2 Analytics: Vietnamese (vie_Latn) — {split}

> Generated: {timestamp} | Records: {count} | File: {path}

## Table of Contents
1. [Overview](#overview)
2. [Text Statistics](#text-statistics)
3. [Temporal Analysis](#temporal-analysis)
4. [URL & Domain Analysis](#url--domain-analysis)
5. [Quality & Deduplication](#quality--deduplication)
6. [Vietnamese Content Analysis](#vietnamese-content-analysis)

## Overview

### Dataset Summary
| Metric | Value |
|--------|-------|
| Total Documents | 2,319,000 |
| Total Characters | ... |
| Total Words | ... |
| Unique Domains | ... |
| Date Range | ... |
| Avg Language Score | ... |

## 1. Text Statistics
{charts 1-12}

## 2. Temporal Analysis
{charts 13-23}

## 3. URL & Domain Analysis
{charts 24-35}

## 4. Quality & Deduplication
{charts 36-45}

## 5. Vietnamese Content Analysis
{charts 46-55}
```

---

## Gradio Interactive App

### Features

1. **Dataset Overview Tab** — Summary statistics, key metrics cards
2. **Text Analysis Tab** — Interactive histograms with adjustable bins, word cloud
3. **Temporal Tab** — Time series with range selector, dump comparison
4. **Domain Tab** — Searchable domain table, TLD sunburst chart
5. **Quality Tab** — Score distribution with threshold slider, cluster analysis
6. **Vietnamese Tab** — Diacritic heatmap, tone distribution, vowel analysis
7. **Compare Tab** — Side-by-side train vs test comparison

### Tech Stack

- Python 3.11+
- Gradio 5.x — UI framework
- Plotly — Interactive charts
- Pandas + PyArrow — Data processing
- DuckDB (Python) — Efficient parquet querying

### Data Loading Strategy

For the Gradio app, use DuckDB's direct parquet reading for on-demand queries:

```python
import duckdb

con = duckdb.connect()
con.execute("""
    CREATE VIEW train AS
    SELECT * FROM read_parquet('~/data/fineweb-2/vie_Latn/train/*.parquet')
""")

# Then run analytics queries on-demand
result = con.execute("""
    SELECT LENGTH(text) as len, COUNT(*) as cnt
    FROM train
    GROUP BY len / 100 * 100
    ORDER BY len
""").fetchdf()
```

### App Structure

```python
# app.py
import gradio as gr
import duckdb
import plotly.express as px
import plotly.graph_objects as go

class FineWebAnalytics:
    def __init__(self, data_dir):
        self.con = duckdb.connect()
        self.setup_views(data_dir)

    def setup_views(self, data_dir):
        for split in ['train', 'test']:
            self.con.execute(f"""
                CREATE VIEW {split} AS
                SELECT *, LENGTH(text) as text_len
                FROM read_parquet('{data_dir}/vie_Latn/{split}/*.parquet')
            """)

    def text_length_histogram(self, split, bins=50):
        df = self.con.execute(f"""
            SELECT text_len FROM {split}
        """).fetchdf()
        fig = px.histogram(df, x='text_len', nbins=bins,
                          title='Document Length Distribution')
        return fig

    # ... 20+ interactive chart methods
```

---

## Implementation Plan

### Phase 1: Core Infrastructure
1. Create `pkg/analytics/types.go` — data structures (Histogram, TopK, Counter)
2. Create `pkg/analytics/reader.go` — extended parquet reader (all 11 fields)
3. Create `pkg/analytics/charts.go` — Mermaid + ASCII chart generators
4. Create `pkg/analytics/stats.go` — statistical helper functions

### Phase 2: Analysis Modules
5. Create `pkg/analytics/text_stats.go` — text analysis (charts 1-12)
6. Create `pkg/analytics/temporal.go` — temporal analysis (charts 13-23)
7. Create `pkg/analytics/domain.go` — domain analysis (charts 24-35)
8. Create `pkg/analytics/quality.go` — quality metrics (charts 36-45)
9. Create `pkg/analytics/content.go` — Vietnamese analysis (charts 46-55)

### Phase 3: Report Generation
10. Create `pkg/analytics/analyzer.go` — main orchestrator
11. Create `pkg/analytics/report.go` — markdown report writer

### Phase 4: CLI Integration
12. Create `cli/analytics.go` — CLI command with progress display
13. Register in `cli/root.go`

### Phase 5: Gradio App
14. Create `tools/gradio_app/app.py` — interactive dashboard
15. Create `tools/gradio_app/requirements.txt`

### Phase 6: Generate Reports
16. Run against test split (validation)
17. Run against train split (full generation)
18. Commit reports to `pkg/analytics/report/vie_Latn/`
