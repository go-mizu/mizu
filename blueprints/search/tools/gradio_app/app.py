#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "gradio>=5.0",
#     "plotly>=5.18",
#     "pandas>=2.0",
#     "pyarrow>=14.0",
#     "duckdb>=1.0",
#     "kaleido>=0.2",
#     "fpdf2>=2.8",
# ]
# ///
"""
FineWeb-2 Vietnamese Dataset Analytics Dashboard

Interactive Gradio app with Plotly charts and DuckDB-powered queries.

Usage:
    uv run app.py                              # Launch dashboard
    uv run app.py --import-db                  # Import parquet → DuckDB first
    uv run app.py --export-pdf --split test    # Export PDF report
"""

import argparse
import os
import tempfile
import time
from datetime import datetime
from pathlib import Path

import duckdb
import gradio as gr
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots

# ── Chart color palette ─────────────────────────────────────────
BLUE = "#1a73e8"
RED = "#ea4335"
GREEN = "#34a853"
YELLOW = "#fbbc04"
ORANGE = "#ff6d01"
PALETTE = [BLUE, RED, GREEN, YELLOW, ORANGE, "#ab47bc", "#00acc1", "#5c6bc0"]

CHART_TEMPLATE = "plotly_white"
CHART_LAYOUT = dict(
    template=CHART_TEMPLATE,
    font=dict(family="Inter, system-ui, sans-serif", size=13),
    title_font_size=18,
    margin=dict(l=60, r=30, t=60, b=50),
    plot_bgcolor="white",
)


def db_path_for(data_dir: str, lang: str) -> str:
    """Return the conventional DuckDB database path."""
    return os.path.join(data_dir, lang, "analytics.duckdb")


def import_parquet_to_duckdb(data_dir: str, lang: str) -> str:
    """Import parquet files into a persistent DuckDB database for fast queries.

    Creates ~/data/fineweb-2/vie_Latn/analytics.duckdb with train/test tables.
    """
    dbpath = db_path_for(data_dir, lang)
    print(f"Importing parquet data into {dbpath}...")

    con = duckdb.connect(dbpath)

    for split in ["train", "test"]:
        parquet_dir = os.path.join(data_dir, lang, split)
        pattern = os.path.join(parquet_dir, "*.parquet")

        if not os.path.isdir(parquet_dir):
            print(f"  Skipping {split}: directory not found")
            continue

        t0 = time.time()
        print(f"  Importing {split}...", end=" ", flush=True)

        con.execute(f"DROP TABLE IF EXISTS {split}")
        con.execute(f"""
            CREATE TABLE {split} AS
            SELECT *,
                LENGTH(text) AS text_len,
                LENGTH(text) - LENGTH(REPLACE(text, ' ', '')) + 1 AS word_count,
                REGEXP_EXTRACT(url, '://([^/]+)', 1) AS host,
                CAST(STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y') AS VARCHAR) AS year,
                STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y-%m') AS month,
                STRFTIME(TRY_CAST(date AS TIMESTAMP), '%H') AS hour,
                DAYNAME(TRY_CAST(date AS TIMESTAMP)) AS day_of_week
            FROM read_parquet('{pattern}')
        """)

        count = con.execute(f"SELECT COUNT(*) FROM {split}").fetchone()[0]
        elapsed = time.time() - t0
        print(f"{count:,} rows in {elapsed:.1f}s")

        # Create indexes for common queries
        print(f"  Creating indexes for {split}...", end=" ", flush=True)
        t0 = time.time()
        for col in ["host", "year", "month"]:
            try:
                con.execute(f"CREATE INDEX idx_{split}_{col} ON {split}({col})")
            except Exception:
                pass  # index may already exist
        elapsed = time.time() - t0
        print(f"done ({elapsed:.1f}s)")

    con.close()
    print(f"Database ready: {dbpath}")
    return dbpath


class FineWebAnalytics:
    """Analytics engine backed by DuckDB for fast parquet queries."""

    def __init__(self, data_dir: str, lang: str = "vie_Latn"):
        self.data_dir = Path(data_dir)
        self.lang = lang

        # Use persistent DB if available, otherwise query parquet directly
        dbpath = db_path_for(data_dir, lang)
        if os.path.exists(dbpath):
            print(f"Using DuckDB database: {dbpath}")
            self.con = duckdb.connect(dbpath, read_only=True)
            self._from_db = True
        else:
            print("No DuckDB database found, querying parquet files directly.")
            print("  Tip: run with --import-db first for faster queries.")
            self.con = duckdb.connect()
            self._from_db = False
            self._setup_views()

    def _setup_views(self):
        """Create views over parquet files (fallback when no DB exists)."""
        for split in ["train", "test"]:
            parquet_path = self.data_dir / self.lang / split / "*.parquet"
            self.con.execute(f"""
                CREATE OR REPLACE VIEW {split} AS
                SELECT *,
                    LENGTH(text) AS text_len,
                    LENGTH(text) - LENGTH(REPLACE(text, ' ', '')) + 1 AS word_count,
                    REGEXP_EXTRACT(url, '://([^/]+)', 1) AS host,
                    CAST(STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y') AS VARCHAR) AS year,
                    STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y-%m') AS month,
                    STRFTIME(TRY_CAST(date AS TIMESTAMP), '%H') AS hour,
                    DAYNAME(TRY_CAST(date AS TIMESTAMP)) AS day_of_week
                FROM read_parquet('{parquet_path}')
            """)

    def query(self, sql: str) -> pd.DataFrame:
        return self.con.execute(sql).fetchdf()

    def close(self):
        self.con.close()

    # ── Overview ────────────────────────────────────────────────
    def overview(self, split: str) -> pd.DataFrame:
        return self.query(f"""
            SELECT
                COUNT(*) AS "Total Documents",
                SUM(text_len) AS "Total Characters",
                SUM(word_count) AS "Total Words",
                COUNT(DISTINCT host) AS "Unique Domains",
                MIN(date) AS "Earliest Date",
                MAX(date) AS "Latest Date",
                ROUND(AVG(language_score), 6) AS "Avg Language Score",
                ROUND(AVG(text_len), 0) AS "Avg Text Length",
                ROUND(MEDIAN(text_len), 0) AS "Median Text Length",
                ROUND(AVG(minhash_cluster_size), 1) AS "Avg Cluster Size"
            FROM {split}
        """)

    # ── Text Statistics ─────────────────────────────────────────
    def text_length_distribution(self, split: str, bins: int = 50) -> go.Figure:
        df = self.query(f"SELECT text_len FROM {split}")
        fig = px.histogram(df, x="text_len", nbins=bins,
                           title="Document Length Distribution (characters)",
                           labels={"text_len": "Text Length (chars)", "count": "Documents"},
                           color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT, bargap=0.05)
        return fig

    def word_count_distribution(self, split: str) -> go.Figure:
        df = self.query(f"SELECT word_count FROM {split}")
        fig = px.histogram(df, x="word_count", nbins=50,
                           title="Word Count Distribution",
                           labels={"word_count": "Words per Document"},
                           color_discrete_sequence=[GREEN])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    def text_percentiles(self, split: str) -> pd.DataFrame:
        return self.query(f"""
            SELECT
                PERCENTILE_CONT(0.01) WITHIN GROUP (ORDER BY text_len) AS P1,
                PERCENTILE_CONT(0.05) WITHIN GROUP (ORDER BY text_len) AS P5,
                PERCENTILE_CONT(0.10) WITHIN GROUP (ORDER BY text_len) AS P10,
                PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY text_len) AS P25,
                PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY text_len) AS P50,
                PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY text_len) AS P75,
                PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY text_len) AS P90,
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY text_len) AS P95,
                PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY text_len) AS P99
            FROM {split}
        """)

    def top_words(self, split: str, limit: int = 30) -> go.Figure:
        df = self.query(f"""
            WITH words AS (
                SELECT UNNEST(STRING_SPLIT(LOWER(text), ' ')) AS word
                FROM {split} USING SAMPLE 10000
            )
            SELECT word, COUNT(*) AS cnt
            FROM words
            WHERE LENGTH(word) BETWEEN 2 AND 20
            GROUP BY word ORDER BY cnt DESC LIMIT {limit}
        """)
        fig = px.bar(df, x="cnt", y="word", orientation="h",
                     title=f"Top {limit} Most Frequent Words (sampled)",
                     labels={"cnt": "Frequency", "word": ""},
                     color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT, yaxis=dict(autorange="reversed"), height=650)
        return fig

    def char_type_breakdown(self, split: str) -> go.Figure:
        df = self.query(f"""
            WITH chars AS (
                SELECT UNNEST(STRING_SPLIT(LEFT(text, 500), '')) AS ch
                FROM {split} USING SAMPLE 5000
            )
            SELECT
                CASE
                    WHEN ch ~ '[a-zA-Z]' THEN 'ASCII Letter'
                    WHEN ch ~ '[0-9]' THEN 'Digit'
                    WHEN ch ~ '\\s' THEN 'Whitespace'
                    WHEN UNICODE(ch) BETWEEN 192 AND 687
                      OR UNICODE(ch) BETWEEN 7680 AND 7935 THEN 'Vietnamese Diacritic'
                    ELSE 'Other / Punctuation'
                END AS char_type,
                COUNT(*) AS cnt
            FROM chars WHERE ch != ''
            GROUP BY char_type ORDER BY cnt DESC
        """)
        fig = px.pie(df, values="cnt", names="char_type",
                     title="Character Type Distribution (sampled)",
                     color_discrete_sequence=PALETTE)
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
        return fig

    # ── Temporal Analysis ───────────────────────────────────────
    def documents_per_year(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT year, COUNT(*) AS cnt
            FROM {split} WHERE year IS NOT NULL
            GROUP BY year ORDER BY year
        """)
        fig = px.bar(df, x="year", y="cnt",
                     title="Documents per Year",
                     labels={"year": "Year", "cnt": "Documents"},
                     color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    def monthly_trend(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT month, COUNT(*) AS cnt
            FROM {split} WHERE month IS NOT NULL
            GROUP BY month ORDER BY month
        """)
        fig = px.line(df, x="month", y="cnt",
                      title="Monthly Document Trend",
                      labels={"month": "Month", "cnt": "Documents"},
                      color_discrete_sequence=[RED])
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(line_width=2.5)
        return fig

    def top_dumps(self, split: str, limit: int = 25) -> go.Figure:
        df = self.query(f"""
            SELECT dump, COUNT(*) AS cnt
            FROM {split} GROUP BY dump
            ORDER BY cnt DESC LIMIT {limit}
        """)
        fig = px.bar(df, x="cnt", y="dump", orientation="h",
                     title=f"Top {limit} Common Crawl Dumps",
                     labels={"cnt": "Documents", "dump": ""},
                     color_discrete_sequence=[YELLOW])
        fig.update_layout(**CHART_LAYOUT, yaxis=dict(autorange="reversed"), height=600)
        return fig

    def hour_distribution(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT hour, COUNT(*) AS cnt
            FROM {split} WHERE hour IS NOT NULL
            GROUP BY hour ORDER BY hour
        """)
        fig = px.bar(df, x="hour", y="cnt",
                     title="Documents by Hour (UTC)",
                     labels={"hour": "Hour", "cnt": "Documents"},
                     color_discrete_sequence=[GREEN])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    def dow_distribution(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT day_of_week, COUNT(*) AS cnt
            FROM {split} WHERE day_of_week IS NOT NULL
            GROUP BY day_of_week
            ORDER BY CASE day_of_week
                WHEN 'Monday' THEN 1 WHEN 'Tuesday' THEN 2 WHEN 'Wednesday' THEN 3
                WHEN 'Thursday' THEN 4 WHEN 'Friday' THEN 5 WHEN 'Saturday' THEN 6
                WHEN 'Sunday' THEN 7 END
        """)
        fig = px.pie(df, values="cnt", names="day_of_week",
                     title="Day of Week Distribution",
                     color_discrete_sequence=PALETTE)
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
        return fig

    # ── Domain Analysis ─────────────────────────────────────────
    def top_domains(self, split: str, limit: int = 30) -> go.Figure:
        df = self.query(f"""
            SELECT REGEXP_REPLACE(host, '^www\\.', '') AS domain, COUNT(*) AS cnt
            FROM {split} WHERE host IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT {limit}
        """)
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title=f"Top {limit} Domains",
                     labels={"cnt": "Documents", "domain": ""},
                     color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT, yaxis=dict(autorange="reversed"), height=650)
        return fig

    def tld_distribution(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                CASE
                    WHEN host LIKE '%.com.vn' THEN '.com.vn'
                    WHEN host LIKE '%.edu.vn' THEN '.edu.vn'
                    WHEN host LIKE '%.gov.vn' THEN '.gov.vn'
                    WHEN host LIKE '%.org.vn' THEN '.org.vn'
                    WHEN host LIKE '%.net.vn' THEN '.net.vn'
                    ELSE '.' || REGEXP_EXTRACT(host, '\\.([a-z]+)$', 1)
                END AS tld,
                COUNT(*) AS cnt
            FROM {split} WHERE host IS NOT NULL
            GROUP BY tld ORDER BY cnt DESC LIMIT 10
        """)
        fig = px.pie(df, values="cnt", names="tld",
                     title="Top-Level Domain Distribution",
                     color_discrete_sequence=PALETTE)
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
        return fig

    def protocol_distribution(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                CASE WHEN url LIKE 'https%' THEN 'HTTPS' ELSE 'HTTP' END AS protocol,
                COUNT(*) AS cnt
            FROM {split} GROUP BY protocol
        """)
        fig = px.pie(df, values="cnt", names="protocol",
                     title="Protocol Distribution",
                     color_discrete_sequence=[GREEN, RED])
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
        return fig

    def url_depth_distribution(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                LENGTH(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1)) -
                LENGTH(REPLACE(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1), '/', '')) AS depth,
                COUNT(*) AS cnt
            FROM {split}
            GROUP BY depth ORDER BY depth LIMIT 15
        """)
        fig = px.bar(df, x="depth", y="cnt",
                     title="URL Path Depth Distribution",
                     labels={"depth": "Path Depth", "cnt": "Documents"},
                     color_discrete_sequence=[YELLOW])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    # ── Quality Metrics ─────────────────────────────────────────
    def language_score_distribution(self, split: str) -> go.Figure:
        df = self.query(f"SELECT language_score FROM {split}")
        fig = px.histogram(df, x="language_score", nbins=100,
                           title="Language Detection Score Distribution",
                           labels={"language_score": "Score"},
                           color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    def cluster_size_distribution(self, split: str) -> go.Figure:
        df = self.query(f"SELECT minhash_cluster_size FROM {split}")
        fig = px.histogram(df, x="minhash_cluster_size", nbins=50,
                           title="MinHash Cluster Size Distribution",
                           labels={"minhash_cluster_size": "Cluster Size"},
                           color_discrete_sequence=[RED])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    def cluster_categories(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                CASE
                    WHEN minhash_cluster_size = 1 THEN 'Unique (1)'
                    WHEN minhash_cluster_size <= 5 THEN 'Small (2-5)'
                    WHEN minhash_cluster_size <= 20 THEN 'Medium (6-20)'
                    WHEN minhash_cluster_size <= 100 THEN 'Large (21-100)'
                    ELSE 'Very Large (100+)'
                END AS category,
                COUNT(*) AS cnt
            FROM {split} GROUP BY category ORDER BY cnt DESC
        """)
        fig = px.pie(df, values="cnt", names="category",
                     title="Cluster Size Categories",
                     color_discrete_sequence=PALETTE)
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
        return fig

    def quality_bands(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                CASE
                    WHEN language_score < 0.8 THEN '<0.80'
                    WHEN language_score < 0.9 THEN '0.80-0.90'
                    WHEN language_score < 0.95 THEN '0.90-0.95'
                    WHEN language_score < 0.99 THEN '0.95-0.99'
                    ELSE '0.99-1.00'
                END AS band,
                COUNT(*) AS cnt
            FROM {split} GROUP BY band ORDER BY band
        """)
        fig = px.bar(df, x="band", y="cnt",
                     title="Quality Score Bands",
                     labels={"band": "Language Score Band", "cnt": "Documents"},
                     color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    def score_vs_length(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                CASE
                    WHEN language_score < 0.9 THEN '<0.9'
                    WHEN language_score < 0.95 THEN '0.9-0.95'
                    WHEN language_score < 0.99 THEN '0.95-0.99'
                    ELSE '>=0.99'
                END AS score_band,
                ROUND(AVG(text_len), 0) AS avg_len,
                ROUND(MEDIAN(text_len), 0) AS median_len,
                COUNT(*) AS cnt
            FROM {split} GROUP BY score_band ORDER BY score_band
        """)
        fig = go.Figure()
        fig.add_trace(go.Bar(x=df["score_band"], y=df["avg_len"],
                             name="Avg Length", marker_color=BLUE))
        fig.add_trace(go.Bar(x=df["score_band"], y=df["median_len"],
                             name="Median Length", marker_color=GREEN))
        fig.update_layout(**CHART_LAYOUT, title="Text Length by Quality Band",
                          barmode="group",
                          xaxis_title="Score Band", yaxis_title="Text Length (chars)")
        return fig

    # ── Vietnamese Content Analysis ─────────────────────────────
    def tone_distribution(self, split: str) -> go.Figure:
        df = self.query(f"""
            WITH chars AS (
                SELECT UNNEST(STRING_SPLIT(LEFT(text, 1000), '')) AS ch
                FROM {split} USING SAMPLE 10000
            ),
            tones AS (
                SELECT
                    CASE
                        WHEN ch IN ('á','ắ','ấ','é','ế','í','ó','ố','ớ','ú','ứ','ý')
                            THEN 'Sắc (rising)'
                        WHEN ch IN ('à','ằ','ầ','è','ề','ì','ò','ồ','ờ','ù','ừ','ỳ')
                            THEN 'Huyền (falling)'
                        WHEN ch IN ('ả','ẳ','ẩ','ẻ','ể','ỉ','ỏ','ổ','ở','ủ','ử','ỷ')
                            THEN 'Hỏi (questioning)'
                        WHEN ch IN ('ã','ẵ','ẫ','ẽ','ễ','ĩ','õ','ỗ','ỡ','ũ','ữ','ỹ')
                            THEN 'Ngã (tumbling)'
                        WHEN ch IN ('ạ','ặ','ậ','ẹ','ệ','ị','ọ','ộ','ợ','ụ','ự','ỵ')
                            THEN 'Nặng (heavy)'
                        ELSE NULL
                    END AS tone
                FROM chars
            )
            SELECT tone, COUNT(*) AS cnt
            FROM tones WHERE tone IS NOT NULL
            GROUP BY tone ORDER BY cnt DESC
        """)
        fig = px.pie(df, values="cnt", names="tone",
                     title="Vietnamese Tone Mark Distribution (sampled)",
                     color_discrete_sequence=[RED, YELLOW, GREEN, BLUE, ORANGE])
        fig.update_layout(**CHART_LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
        return fig

    def content_types(self, split: str) -> go.Figure:
        df = self.query(f"""
            SELECT
                CASE
                    WHEN url ILIKE '%news%' OR url ILIKE '%bao%'
                         OR url ILIKE '%tin-tuc%' THEN 'News'
                    WHEN url ILIKE '%forum%' OR url ILIKE '%dien-dan%'
                         OR url ILIKE '%thread%' THEN 'Forum'
                    WHEN url ILIKE '%blog%' THEN 'Blog'
                    WHEN url ILIKE '%shop%' OR url ILIKE '%product%'
                         OR url ILIKE '%san-pham%' THEN 'E-commerce'
                    WHEN url ILIKE '%.gov.vn%' THEN 'Government'
                    WHEN url ILIKE '%.edu.vn%' THEN 'Education'
                    WHEN url ILIKE '%wiki%' THEN 'Wiki / Reference'
                    ELSE 'Other'
                END AS content_type,
                COUNT(*) AS cnt
            FROM {split} GROUP BY content_type ORDER BY cnt DESC
        """)
        fig = px.bar(df, x="content_type", y="cnt",
                     title="Content Type Distribution (URL Heuristic)",
                     labels={"content_type": "Type", "cnt": "Documents"},
                     color_discrete_sequence=[BLUE])
        fig.update_layout(**CHART_LAYOUT)
        return fig

    # ── Comparison ──────────────────────────────────────────────
    def compare_splits(self) -> go.Figure:
        df = self.query("""
            SELECT 'train' AS split, COUNT(*) AS docs, AVG(text_len) AS avg_len,
                   AVG(language_score) AS avg_score, COUNT(DISTINCT host) AS unique_domains
            FROM train
            UNION ALL
            SELECT 'test' AS split, COUNT(*) AS docs, AVG(text_len) AS avg_len,
                   AVG(language_score) AS avg_score, COUNT(DISTINCT host) AS unique_domains
            FROM test
        """)
        fig = make_subplots(
            rows=1, cols=4,
            subplot_titles=["Documents", "Avg Text Length", "Avg Language Score", "Unique Domains"],
        )
        for i, col in enumerate(["docs", "avg_len", "avg_score", "unique_domains"], 1):
            fig.add_trace(go.Bar(
                x=df["split"], y=df[col],
                marker_color=[BLUE, RED], showlegend=False,
            ), row=1, col=i)
        fig.update_layout(**CHART_LAYOUT, title="Train vs Test Comparison", height=400)
        return fig

    # ── Collect all figures ──────────────────────────────────────
    def all_figures(self, split: str) -> list[tuple[str, go.Figure]]:
        """Generate all chart figures for a split."""
        return [
            ("Document Length Distribution", self.text_length_distribution(split)),
            ("Word Count Distribution", self.word_count_distribution(split)),
            ("Top 30 Words", self.top_words(split)),
            ("Character Type Breakdown", self.char_type_breakdown(split)),
            ("Documents per Year", self.documents_per_year(split)),
            ("Monthly Document Trend", self.monthly_trend(split)),
            ("Top Common Crawl Dumps", self.top_dumps(split)),
            ("Documents by Hour (UTC)", self.hour_distribution(split)),
            ("Day of Week Distribution", self.dow_distribution(split)),
            ("Top 30 Domains", self.top_domains(split)),
            ("TLD Distribution", self.tld_distribution(split)),
            ("Protocol Distribution", self.protocol_distribution(split)),
            ("URL Path Depth", self.url_depth_distribution(split)),
            ("Language Score Distribution", self.language_score_distribution(split)),
            ("Cluster Size Distribution", self.cluster_size_distribution(split)),
            ("Cluster Size Categories", self.cluster_categories(split)),
            ("Quality Score Bands", self.quality_bands(split)),
            ("Text Length by Quality Band", self.score_vs_length(split)),
            ("Vietnamese Tone Distribution", self.tone_distribution(split)),
            ("Content Type Distribution", self.content_types(split)),
            ("Train vs Test Comparison", self.compare_splits()),
        ]


# ── PDF Export ──────────────────────────────────────────────────
def export_pdf(data_dir: str, lang: str, split: str, output: str | None = None) -> str:
    """Export all charts to a multi-page PDF report."""
    from fpdf import FPDF
    from fpdf.enums import XPos, YPos

    analytics = FineWebAnalytics(data_dir, lang)

    if output is None:
        output = f"fineweb2_{lang}_{split}_{datetime.now():%Y%m%d_%H%M%S}.pdf"

    print(f"Generating PDF report for {split} split...")

    overview_df = analytics.overview(split)
    percentile_df = analytics.text_percentiles(split)
    figures = analytics.all_figures(split)

    # Render charts to temp PNGs
    tmpdir = tempfile.mkdtemp(prefix="fineweb_pdf_")
    chart_paths: list[tuple[str, str]] = []

    for i, (title, fig) in enumerate(figures):
        fig.update_layout(width=1000, height=550, template="plotly_white")
        path = os.path.join(tmpdir, f"chart_{i:02d}.png")
        fig.write_image(path, scale=2)
        chart_paths.append((title, path))
        print(f"  [{i + 1}/{len(figures)}] {title}")

    # Assemble PDF (landscape A4)
    pdf = FPDF(orientation="L", unit="mm", format="A4")
    pdf.set_auto_page_break(auto=True, margin=15)
    NL = {"new_x": XPos.LMARGIN, "new_y": YPos.NEXT}

    # ── Title page ──
    pdf.add_page()
    pdf.set_font("Helvetica", "B", 32)
    pdf.cell(0, 50, "", **NL)
    pdf.cell(0, 16, "FineWeb-2 Analytics Report", align="C", **NL)
    pdf.set_font("Helvetica", "", 18)
    pdf.cell(0, 14, f"Language: {lang}  |  Split: {split}", align="C", **NL)
    pdf.set_font("Helvetica", "", 14)
    pdf.cell(0, 10, f"Generated: {datetime.now():%Y-%m-%d %H:%M}", align="C", **NL)

    pdf.ln(12)
    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 10, "Dataset Overview", align="C", **NL)
    pdf.set_font("Helvetica", "", 12)

    for col in overview_df.columns:
        val = overview_df[col].iloc[0]
        if isinstance(val, (int, float)):
            # Format as integer if whole number, otherwise 2 decimal places
            if val == int(val):
                display = f"{int(val):,}"
            else:
                display = f"{val:,.6f}".rstrip("0").rstrip(".")
        else:
            display = str(val)
        pdf.cell(0, 8, f"{col}: {display}", align="C", **NL)

    # Percentiles
    pdf.ln(6)
    pdf.set_font("Helvetica", "B", 14)
    pdf.cell(0, 10, "Text Length Percentiles", align="C", **NL)
    pdf.set_font("Helvetica", "", 11)
    pct_vals = [f"P{p}: {int(percentile_df[f'P{p}'].iloc[0]):,}" for p in [1, 5, 25, 50, 75, 95, 99]]
    pdf.cell(0, 8, "   |   ".join(pct_vals), align="C", **NL)

    # ── Chart pages ──
    for title, path in chart_paths:
        pdf.add_page()
        pdf.set_font("Helvetica", "B", 18)
        pdf.cell(0, 12, title, align="C", **NL)
        pdf.ln(2)
        img_w = 255
        x = (297 - img_w) / 2
        pdf.image(path, x=x, w=img_w)

    pdf.output(output)

    # Cleanup
    for _, path in chart_paths:
        os.unlink(path)
    os.rmdir(tmpdir)

    size_mb = os.path.getsize(output) / (1024 * 1024)
    print(f"PDF saved: {output} ({size_mb:.1f} MB, {len(chart_paths) + 1} pages)")
    analytics.close()
    return output


# ── Gradio App ──────────────────────────────────────────────────
def create_app(data_dir: str, lang: str = "vie_Latn") -> gr.Blocks:
    """Create the Gradio dashboard app."""
    analytics = FineWebAnalytics(data_dir, lang)

    with gr.Blocks(
        title="FineWeb-2 Vietnamese Analytics",
        theme=gr.themes.Soft(primary_hue="blue"),
        css="""
        .main-title { text-align: center; margin-bottom: 8px; }
        .subtitle { text-align: center; color: #5f6368; margin-bottom: 16px; }
        """
    ) as app:
        gr.Markdown("# FineWeb-2 Vietnamese Dataset Analytics", elem_classes="main-title")
        gr.Markdown(
            f"Interactive exploration of **{lang}** web corpus  ·  "
            f"Powered by DuckDB {'(persistent DB)' if analytics._from_db else '(parquet)'}",
            elem_classes="subtitle",
        )

        with gr.Row():
            split = gr.Radio(["test", "train"], value="test",
                             label="Dataset Split", interactive=True)
            export_btn = gr.Button("Export PDF Report", variant="secondary",
                                   scale=0, min_width=150)
            pdf_output = gr.File(label="Download PDF", visible=False)

        def do_export(s):
            path = export_pdf(data_dir, lang, s)
            return gr.update(value=path, visible=True)

        export_btn.click(do_export, split, pdf_output)

        with gr.Tabs():
            # ── Overview ────────────────────────────────────────
            with gr.Tab("Overview"):
                overview_df = gr.Dataframe(label="Dataset Summary")
                compare_plot = gr.Plot(label="Train vs Test")

                def update_overview(s):
                    return analytics.overview(s), analytics.compare_splits()

                split.change(update_overview, split, [overview_df, compare_plot])
                app.load(update_overview, split, [overview_df, compare_plot])

            # ── Text Statistics ─────────────────────────────────
            with gr.Tab("Text Statistics"):
                bins_slider = gr.Slider(10, 200, value=50, step=10, label="Histogram Bins")
                with gr.Row():
                    text_len_plot = gr.Plot(label="Text Length")
                    word_count_plot = gr.Plot(label="Word Count")
                with gr.Row():
                    top_words_plot = gr.Plot(label="Top Words")
                    char_type_plot = gr.Plot(label="Character Types")
                percentile_df = gr.Dataframe(label="Text Length Percentiles")

                def update_text(s, b):
                    return (
                        analytics.text_length_distribution(s, b),
                        analytics.word_count_distribution(s),
                        analytics.top_words(s),
                        analytics.char_type_breakdown(s),
                        analytics.text_percentiles(s),
                    )

                for inp in [split, bins_slider]:
                    inp.change(update_text, [split, bins_slider],
                               [text_len_plot, word_count_plot, top_words_plot,
                                char_type_plot, percentile_df])
                app.load(update_text, [split, bins_slider],
                         [text_len_plot, word_count_plot, top_words_plot,
                          char_type_plot, percentile_df])

            # ── Temporal Analysis ───────────────────────────────
            with gr.Tab("Temporal Analysis"):
                with gr.Row():
                    year_plot = gr.Plot(label="Per Year")
                    monthly_plot = gr.Plot(label="Monthly Trend")
                with gr.Row():
                    hour_plot = gr.Plot(label="By Hour")
                    dow_plot = gr.Plot(label="By Day of Week")
                dumps_plot = gr.Plot(label="Top Dumps")

                def update_temporal(s):
                    return (
                        analytics.documents_per_year(s),
                        analytics.monthly_trend(s),
                        analytics.hour_distribution(s),
                        analytics.dow_distribution(s),
                        analytics.top_dumps(s),
                    )

                split.change(update_temporal, split,
                             [year_plot, monthly_plot, hour_plot, dow_plot, dumps_plot])
                app.load(update_temporal, split,
                         [year_plot, monthly_plot, hour_plot, dow_plot, dumps_plot])

            # ── Domains & URLs ──────────────────────────────────
            with gr.Tab("Domains & URLs"):
                with gr.Row():
                    domains_plot = gr.Plot(label="Top Domains")
                    tld_plot = gr.Plot(label="TLD Distribution")
                with gr.Row():
                    protocol_plot = gr.Plot(label="Protocol")
                    depth_plot = gr.Plot(label="URL Depth")

                def update_domains(s):
                    return (
                        analytics.top_domains(s),
                        analytics.tld_distribution(s),
                        analytics.protocol_distribution(s),
                        analytics.url_depth_distribution(s),
                    )

                split.change(update_domains, split,
                             [domains_plot, tld_plot, protocol_plot, depth_plot])
                app.load(update_domains, split,
                         [domains_plot, tld_plot, protocol_plot, depth_plot])

            # ── Quality Metrics ─────────────────────────────────
            with gr.Tab("Quality Metrics"):
                with gr.Row():
                    lang_score_plot = gr.Plot(label="Language Score")
                    cluster_plot = gr.Plot(label="Cluster Size")
                with gr.Row():
                    cluster_cat_plot = gr.Plot(label="Cluster Categories")
                    quality_band_plot = gr.Plot(label="Quality Bands")
                score_len_plot = gr.Plot(label="Score vs Length")

                def update_quality(s):
                    return (
                        analytics.language_score_distribution(s),
                        analytics.cluster_size_distribution(s),
                        analytics.cluster_categories(s),
                        analytics.quality_bands(s),
                        analytics.score_vs_length(s),
                    )

                split.change(update_quality, split,
                             [lang_score_plot, cluster_plot, cluster_cat_plot,
                              quality_band_plot, score_len_plot])
                app.load(update_quality, split,
                         [lang_score_plot, cluster_plot, cluster_cat_plot,
                          quality_band_plot, score_len_plot])

            # ── Vietnamese Content ──────────────────────────────
            with gr.Tab("Vietnamese Content"):
                with gr.Row():
                    tone_plot = gr.Plot(label="Tone Distribution")
                    content_type_plot = gr.Plot(label="Content Types")

                def update_vietnamese(s):
                    return (
                        analytics.tone_distribution(s),
                        analytics.content_types(s),
                    )

                split.change(update_vietnamese, split, [tone_plot, content_type_plot])
                app.load(update_vietnamese, split, [tone_plot, content_type_plot])

    return app


# ── CLI ─────────────────────────────────────────────────────────
if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="FineWeb-2 Analytics Dashboard",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  uv run app.py                            Launch dashboard (test split)
  uv run app.py --import-db                Import parquet → DuckDB database
  uv run app.py --export-pdf --split test  Export PDF report for test split
  uv run app.py --export-pdf --split train Export PDF report for train split
  uv run app.py --port 8080               Launch on custom port
        """,
    )
    parser.add_argument("--data-dir", default=os.path.expanduser("~/data/fineweb-2"),
                        help="Path to FineWeb-2 data directory (default: ~/data/fineweb-2)")
    parser.add_argument("--lang", default="vie_Latn", help="Language code (default: vie_Latn)")
    parser.add_argument("--port", type=int, default=7860, help="Port number (default: 7860)")
    parser.add_argument("--share", action="store_true", help="Create public Gradio link")
    parser.add_argument("--import-db", action="store_true",
                        help="Import parquet files into DuckDB database and exit")
    parser.add_argument("--export-pdf", action="store_true",
                        help="Export PDF report and exit (no server)")
    parser.add_argument("--split", default="test",
                        help="Split for PDF export (default: test)")
    args = parser.parse_args()

    if args.import_db:
        import_parquet_to_duckdb(args.data_dir, args.lang)
    elif args.export_pdf:
        export_pdf(args.data_dir, args.lang, args.split)
    else:
        app = create_app(args.data_dir, args.lang)
        app.launch(server_port=args.port, share=args.share)
