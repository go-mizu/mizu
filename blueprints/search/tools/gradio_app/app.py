#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "gradio>=6.5",
#     "plotly>=6.5",
#     "pandas>=2.2",
#     "pyarrow>=18.0",
#     "duckdb>=1.4",
#     "kaleido>=1.2",
#     "fpdf2>=2.8",
# ]
# ///
"""
FineWeb-2 Analytics Dashboard — 50 charts across 10 categories.

Usage:
    uv run app.py --setup                  # Import + compute cache
    uv run app.py                          # Launch dashboard (from cache)
    uv run app.py --export-pdf --split test

Download data first with the CLI:
    search download get --lang vie_Latn --split train --shards 2
    search download get --lang vie_Latn --split test
"""

import argparse
import json
import os
import tempfile
import time
from contextlib import contextmanager
from datetime import datetime
from pathlib import Path

import duckdb
import numpy as np
import pandas as pd
import plotly.express as px
import plotly.graph_objects as go
from plotly.subplots import make_subplots

# ── Constants ──────────────────────────────────────────────────
BLUE = "#1a73e8"
RED = "#ea4335"
GREEN = "#34a853"
YELLOW = "#fbbc04"
ORANGE = "#ff6d01"
PURPLE = "#ab47bc"
TEAL = "#00acc1"
INDIGO = "#5c6bc0"
PINK = "#e91e63"
BROWN = "#795548"
PALETTE = [BLUE, RED, GREEN, YELLOW, ORANGE, PURPLE, TEAL, INDIGO, PINK, BROWN]

LAYOUT = dict(
    template="plotly_white",
    font=dict(family="Inter, system-ui, sans-serif", size=12),
    title_font_size=15,
    margin=dict(l=50, r=20, t=50, b=40),
    plot_bgcolor="white",
    hoverlabel=dict(font_size=12),
)

CATEGORIES = [
    "Overview", "Text Length", "Vocabulary", "Temporal",
    "Domains", "URL & Web", "Quality", "Deduplication",
    "Vietnamese", "Content",
]


# ── Helpers ────────────────────────────────────────────────────
class Pipeline:
    """Track and report timing for multi-step operations."""
    def __init__(self, name="Pipeline"):
        self.name = name
        self.steps: list[tuple[str, float]] = []

    @contextmanager
    def step(self, label):
        t0 = time.time()
        n = len(self.steps) + 1
        print(f"  [{n}] {label}...", end=" ", flush=True)
        try:
            yield
        finally:
            elapsed = time.time() - t0
            self.steps.append((label, elapsed))
            print(f"done ({elapsed:.1f}s)")

    def summary(self):
        total = sum(t for _, t in self.steps)
        print(f"\n{'=' * 60}")
        print(f"  {self.name} completed in {total:.1f}s")
        print(f"{'=' * 60}")
        for label, t in self.steps:
            bar = "#" * max(1, int(t / max(total, 0.1) * 30))
            print(f"  {label:40s} {t:>7.1f}s  {bar}")
        print(f"{'=' * 60}\n")


def _sanitize(records: list[dict]) -> list[dict]:
    """Convert numpy/pandas types to JSON-safe Python types."""
    result = []
    for row in records:
        r = {}
        for k, v in row.items():
            if v is None:
                r[k] = None
            elif isinstance(v, float) and (v != v):  # NaN
                r[k] = None
            elif hasattr(v, "item"):  # numpy scalar
                val = v.item()
                r[k] = None if isinstance(val, float) and (val != val) else val
            else:
                r[k] = v
        result.append(r)
    return result


def _empty_fig(title: str) -> go.Figure:
    fig = go.Figure()
    fig.add_annotation(text="No data available", showarrow=False,
                       font=dict(size=16, color="#999"))
    fig.update_layout(**LAYOUT, title=title)
    return fig


def _fmt(v) -> str:
    if v is None:
        return "N/A"
    if isinstance(v, float) and v == int(v):
        return f"{int(v):,}"
    if isinstance(v, float):
        return f"{v:,.4f}"
    if isinstance(v, int):
        return f"{v:,}"
    return str(v)


def db_path(data_dir: str, lang: str, split: str) -> str:
    return os.path.join(data_dir, lang, f"{split}.duckdb")


def cache_path(data_dir: str, lang: str, split: str) -> str:
    return os.path.join(data_dir, lang, f".cache_{split}.json")


# ── Import to DuckDB ──────────────────────────────────────────
def import_to_duckdb(data_dir: str, lang: str, split: str) -> str | None:
    """Import parquet files into a split-specific DuckDB database."""
    import glob as _glob

    dbp = db_path(data_dir, lang, split)
    parquet_dir = os.path.join(data_dir, lang, split)
    pattern = os.path.join(parquet_dir, "*.parquet")

    files = sorted(_glob.glob(pattern))
    if not files:
        print(f"    No parquet files in {parquet_dir}")
        return None

    print(f"    Found {len(files)} parquet file(s)")

    if os.path.exists(dbp):
        os.unlink(dbp)

    con = duckdb.connect(dbp)
    t0 = time.time()
    con.execute(f"""
        CREATE TABLE docs AS
        SELECT *,
            LENGTH(text) AS text_len,
            LENGTH(text) - LENGTH(REPLACE(text, ' ', '')) + 1 AS word_count,
            REGEXP_EXTRACT(url, '://([^/]+)', 1) AS host,
            REGEXP_REPLACE(REGEXP_EXTRACT(url, '://([^/]+)', 1), '^www\\.', '') AS domain,
            CAST(STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y') AS VARCHAR) AS year,
            STRFTIME(TRY_CAST(date AS TIMESTAMP), '%Y-%m') AS month,
            STRFTIME(TRY_CAST(date AS TIMESTAMP), '%H') AS hour,
            DAYNAME(TRY_CAST(date AS TIMESTAMP)) AS day_of_week,
            LENGTH(url) AS url_len,
            CASE WHEN url LIKE 'https%' THEN 'HTTPS' ELSE 'HTTP' END AS protocol,
            CASE WHEN url LIKE '%?%' THEN 1 ELSE 0 END AS has_query,
            LENGTH(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1))
                - LENGTH(REPLACE(REGEXP_EXTRACT(url, '://[^/]+(.*)', 1), '/', '')) AS url_depth,
            CASE
                WHEN url ILIKE '%news%' OR url ILIKE '%bao%' OR url ILIKE '%tin-tuc%'
                    OR url ILIKE '%thoi-su%' THEN 'News'
                WHEN url ILIKE '%forum%' OR url ILIKE '%dien-dan%'
                    OR url ILIKE '%hoi-dap%' THEN 'Forum'
                WHEN url ILIKE '%blog%' THEN 'Blog'
                WHEN url ILIKE '%shop%' OR url ILIKE '%product%'
                    OR url ILIKE '%san-pham%' OR url ILIKE '%mua-ban%' THEN 'E-commerce'
                WHEN url ILIKE '%.gov.vn%' THEN 'Government'
                WHEN url ILIKE '%.edu.vn%' THEN 'Education'
                WHEN url ILIKE '%wiki%' THEN 'Wiki/Reference'
                WHEN url ILIKE '%video%' OR url ILIKE '%youtube%' THEN 'Video/Media'
                WHEN url ILIKE '%recipe%' OR url ILIKE '%nau-an%' THEN 'Recipe/Food'
                ELSE 'Other'
            END AS content_type,
            CASE
                WHEN REGEXP_EXTRACT(url, '://([^/]+)', 1) LIKE '%.com.vn' THEN '.com.vn'
                WHEN REGEXP_EXTRACT(url, '://([^/]+)', 1) LIKE '%.edu.vn' THEN '.edu.vn'
                WHEN REGEXP_EXTRACT(url, '://([^/]+)', 1) LIKE '%.gov.vn' THEN '.gov.vn'
                WHEN REGEXP_EXTRACT(url, '://([^/]+)', 1) LIKE '%.org.vn' THEN '.org.vn'
                WHEN REGEXP_EXTRACT(url, '://([^/]+)', 1) LIKE '%.net.vn' THEN '.net.vn'
                WHEN REGEXP_EXTRACT(url, '://([^/]+)', 1) LIKE '%.vn' THEN '.vn'
                ELSE '.' || REGEXP_EXTRACT(REGEXP_EXTRACT(url, '://([^/]+)', 1), '\\.([a-z]+)$', 1)
            END AS tld
        FROM read_parquet('{pattern}')
    """)

    count = con.execute("SELECT COUNT(*) FROM docs").fetchone()[0]
    print(f"    Imported {count:,} rows ({time.time() - t0:.1f}s)")

    t0 = time.time()
    for col in ["domain", "year", "month", "tld", "content_type"]:
        try:
            con.execute(f"CREATE INDEX idx_{col} ON docs({col})")
        except Exception:
            pass
    print(f"    Indexes created ({time.time() - t0:.1f}s)")

    con.close()
    return dbp


# ── Analytics Queries ──────────────────────────────────────────
def compute_cache(data_dir: str, lang: str, split: str) -> dict | None:
    """Run all analytics queries and save JSON cache."""
    dbp = db_path(data_dir, lang, split)
    if not os.path.exists(dbp):
        print(f"    No database: {dbp}")
        return None

    con = duckdb.connect(dbp, read_only=True)

    def q(sql: str) -> list[dict]:
        return _sanitize(con.execute(sql).fetchdf().to_dict(orient="records"))

    queries = _all_queries()
    data = {}
    total = len(queries)

    for i, (name, sql) in enumerate(queries):
        t0 = time.time()
        try:
            data[name] = q(sql)
        except Exception as e:
            print(f"\n    Warning: '{name}' failed: {e}")
            data[name] = []
        elapsed = time.time() - t0
        print(f"\r    [{i + 1}/{total}] {name:40s} ({elapsed:.1f}s)", end="", flush=True)

    print()
    con.close()

    cp = cache_path(data_dir, lang, split)
    with open(cp, "w") as f:
        json.dump(data, f, default=str)
    print(f"    Cached to {cp}")
    return data


def _all_queries() -> list[tuple[str, str]]:
    """Return all analytics queries as (name, sql) pairs."""
    return [
        # ── 1. Overview ──
        ("overview", """
            SELECT COUNT(*) AS total_docs, SUM(text_len) AS total_chars,
                SUM(word_count) AS total_words, COUNT(DISTINCT domain) AS unique_domains,
                MIN(date) AS earliest, MAX(date) AS latest,
                ROUND(AVG(language_score), 6) AS avg_score,
                ROUND(AVG(text_len), 0) AS avg_len,
                ROUND(MEDIAN(text_len), 0) AS median_len,
                ROUND(AVG(minhash_cluster_size), 1) AS avg_cluster
            FROM docs
        """),
        ("completeness", """
            SELECT COUNT(*) AS total,
                COUNT(text) AS has_text, COUNT(url) AS has_url,
                COUNT(date) AS has_date, COUNT(language_score) AS has_score,
                COUNT(dump) AS has_dump, COUNT(domain) AS has_domain
            FROM docs
        """),
        ("score_hist", """
            SELECT ROUND(language_score, 2) AS score_bin, COUNT(*) AS cnt
            FROM docs GROUP BY score_bin ORDER BY score_bin
        """),
        ("text_len_hist", """
            SELECT LEAST(FLOOR(text_len / 500) * 500, 50000) AS bin, COUNT(*) AS cnt
            FROM docs GROUP BY bin ORDER BY bin
        """),
        ("sample_5k", """
            SELECT text_len, word_count, language_score, minhash_cluster_size,
                domain, year, url_len, url_depth, content_type
            FROM docs USING SAMPLE 5000
        """),

        # ── 2. Text Length ──
        ("text_len_buckets", """
            SELECT CASE
                WHEN text_len < 500 THEN '< 500'
                WHEN text_len < 1000 THEN '500-1K'
                WHEN text_len < 2000 THEN '1K-2K'
                WHEN text_len < 5000 THEN '2K-5K'
                WHEN text_len < 10000 THEN '5K-10K'
                WHEN text_len < 20000 THEN '10K-20K'
                ELSE '20K+'
            END AS bucket, COUNT(*) AS cnt
            FROM docs GROUP BY bucket
            ORDER BY CASE bucket
                WHEN '< 500' THEN 1 WHEN '500-1K' THEN 2 WHEN '1K-2K' THEN 3
                WHEN '2K-5K' THEN 4 WHEN '5K-10K' THEN 5 WHEN '10K-20K' THEN 6
                ELSE 7 END
        """),
        ("percentiles", """
            SELECT
                PERCENTILE_CONT(0.05) WITHIN GROUP (ORDER BY text_len) AS P5,
                PERCENTILE_CONT(0.10) WITHIN GROUP (ORDER BY text_len) AS P10,
                PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY text_len) AS P25,
                PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY text_len) AS P50,
                PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY text_len) AS P75,
                PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY text_len) AS P90,
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY text_len) AS P95,
                PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY text_len) AS P99
            FROM docs
        """),
        ("short_docs_domains", """
            SELECT domain, COUNT(*) AS cnt
            FROM docs WHERE text_len < 500 AND domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 15
        """),
        ("long_docs_domains", """
            SELECT domain, COUNT(*) AS cnt, ROUND(AVG(text_len), 0) AS avg_len
            FROM docs WHERE text_len > 10000 AND domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 15
        """),

        # ── 3. Vocabulary ──
        ("top_words", """
            WITH words AS (
                SELECT UNNEST(STRING_SPLIT(LOWER(LEFT(text, 500)), ' ')) AS word
                FROM docs USING SAMPLE 10000
            )
            SELECT word, COUNT(*) AS cnt
            FROM words WHERE LENGTH(word) BETWEEN 2 AND 20
            GROUP BY word ORDER BY cnt DESC LIMIT 40
        """),
        ("char_types", """
            WITH chars AS (
                SELECT UNNEST(STRING_SPLIT(LEFT(text, 500), '')) AS ch
                FROM docs USING SAMPLE 5000
            )
            SELECT CASE
                WHEN ch ~ '[a-zA-Z]' THEN 'ASCII Letter'
                WHEN ch ~ '[0-9]' THEN 'Digit'
                WHEN ch ~ '\\s' THEN 'Whitespace'
                WHEN UNICODE(ch) BETWEEN 192 AND 687
                  OR UNICODE(ch) BETWEEN 7680 AND 7935 THEN 'Vietnamese'
                ELSE 'Punctuation'
            END AS char_type, COUNT(*) AS cnt
            FROM chars WHERE ch != ''
            GROUP BY char_type ORDER BY cnt DESC
        """),
        ("word_lengths", """
            WITH words AS (
                SELECT UNNEST(STRING_SPLIT(LOWER(LEFT(text, 500)), ' ')) AS word
                FROM docs USING SAMPLE 5000
            )
            SELECT LENGTH(word) AS wlen, COUNT(*) AS cnt
            FROM words WHERE LENGTH(word) BETWEEN 1 AND 20
            GROUP BY wlen ORDER BY wlen
        """),
        ("line_count_buckets", """
            WITH sampled AS (
                SELECT LENGTH(text) - LENGTH(REPLACE(text, CHR(10), '')) + 1 AS lc
                FROM docs USING SAMPLE 10000
            )
            SELECT CASE
                WHEN lc <= 5 THEN '1-5'
                WHEN lc <= 10 THEN '6-10'
                WHEN lc <= 20 THEN '11-20'
                WHEN lc <= 50 THEN '21-50'
                WHEN lc <= 100 THEN '51-100'
                ELSE '100+'
            END AS bucket, COUNT(*) AS cnt
            FROM sampled GROUP BY bucket
            ORDER BY CASE bucket
                WHEN '1-5' THEN 1 WHEN '6-10' THEN 2 WHEN '11-20' THEN 3
                WHEN '21-50' THEN 4 WHEN '51-100' THEN 5 ELSE 6 END
        """),
        ("viet_stop_words", """
            WITH words AS (
                SELECT UNNEST(STRING_SPLIT(LOWER(LEFT(text, 500)), ' ')) AS word
                FROM docs USING SAMPLE 10000
            )
            SELECT word, COUNT(*) AS cnt
            FROM words
            WHERE word IN ('cua','va','la','co','duoc','cho','trong','nay',
                           'voi','khong','cac','mot','tu','da','den','nguoi',
                           'nhu','khi','tai','theo','de','ve','tren','cung',
                           'con','hay','rat','nhat','neu','nhung',
                           'của','và','là','có','được','cho','trong','này',
                           'với','không','các','một','từ','đã','đến','người',
                           'như','khi','tại','theo','để','về','trên','cũng',
                           'còn','hay','rất','nhất','nếu','những')
            GROUP BY word ORDER BY cnt DESC LIMIT 25
        """),

        # ── 4. Temporal ──
        ("by_year", """
            SELECT year, COUNT(*) AS cnt FROM docs
            WHERE year IS NOT NULL GROUP BY year ORDER BY year
        """),
        ("by_month", """
            SELECT month, COUNT(*) AS cnt FROM docs
            WHERE month IS NOT NULL GROUP BY month ORDER BY month
        """),
        ("by_hour", """
            SELECT hour, COUNT(*) AS cnt FROM docs
            WHERE hour IS NOT NULL GROUP BY hour ORDER BY hour
        """),
        ("by_dow", """
            SELECT day_of_week AS dow, COUNT(*) AS cnt FROM docs
            WHERE day_of_week IS NOT NULL GROUP BY dow
            ORDER BY CASE dow
                WHEN 'Monday' THEN 1 WHEN 'Tuesday' THEN 2 WHEN 'Wednesday' THEN 3
                WHEN 'Thursday' THEN 4 WHEN 'Friday' THEN 5 WHEN 'Saturday' THEN 6
                WHEN 'Sunday' THEN 7 END
        """),
        ("hour_dow", """
            SELECT hour, day_of_week AS dow, COUNT(*) AS cnt
            FROM docs WHERE hour IS NOT NULL AND day_of_week IS NOT NULL
            GROUP BY hour, dow
        """),
        ("top_dumps", """
            SELECT dump, COUNT(*) AS cnt FROM docs
            GROUP BY dump ORDER BY cnt DESC LIMIT 20
        """),

        # ── 5. Domains ──
        ("top_domains", """
            SELECT domain, COUNT(*) AS cnt,
                ROUND(AVG(language_score), 4) AS avg_score,
                ROUND(AVG(text_len), 0) AS avg_len
            FROM docs WHERE domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 25
        """),
        ("domain_concentration", """
            WITH ranked AS (
                SELECT domain, COUNT(*) AS cnt,
                    SUM(COUNT(*)) OVER (ORDER BY COUNT(*) DESC) AS cumulative
                FROM docs WHERE domain IS NOT NULL
                GROUP BY domain
            )
            SELECT domain, cnt, cumulative,
                ROUND(100.0 * cumulative / (SELECT COUNT(*) FROM docs), 2) AS cum_pct
            FROM ranked ORDER BY cnt DESC LIMIT 50
        """),
        ("domains_per_year", """
            SELECT year, COUNT(DISTINCT domain) AS unique_domains, COUNT(*) AS total_docs
            FROM docs WHERE year IS NOT NULL AND domain IS NOT NULL
            GROUP BY year ORDER BY year
        """),
        ("domain_text_len", """
            SELECT domain, ROUND(AVG(text_len), 0) AS avg_len,
                ROUND(MEDIAN(text_len), 0) AS median_len, COUNT(*) AS cnt
            FROM docs WHERE domain IS NOT NULL
            GROUP BY domain HAVING cnt >= 50
            ORDER BY avg_len DESC LIMIT 20
        """),
        ("domain_size_dist", """
            SELECT CASE
                WHEN cnt <= 10 THEN '1-10'
                WHEN cnt <= 100 THEN '11-100'
                WHEN cnt <= 1000 THEN '101-1K'
                WHEN cnt <= 10000 THEN '1K-10K'
                ELSE '10K+'
            END AS bucket, COUNT(*) AS num_domains
            FROM (SELECT domain, COUNT(*) AS cnt FROM docs WHERE domain IS NOT NULL GROUP BY domain)
            GROUP BY bucket
            ORDER BY CASE bucket
                WHEN '1-10' THEN 1 WHEN '11-100' THEN 2 WHEN '101-1K' THEN 3
                WHEN '1K-10K' THEN 4 ELSE 5 END
        """),

        # ── 6. URL & Web ──
        ("tld", """
            SELECT tld, COUNT(*) AS cnt FROM docs
            WHERE tld IS NOT NULL AND tld != '.'
            GROUP BY tld ORDER BY cnt DESC LIMIT 12
        """),
        ("protocol_stats", """
            SELECT protocol, COUNT(*) AS cnt,
                ROUND(AVG(text_len), 0) AS avg_len,
                ROUND(AVG(language_score), 4) AS avg_score
            FROM docs GROUP BY protocol
        """),
        ("url_depth", """
            SELECT url_depth AS depth, COUNT(*) AS cnt
            FROM docs WHERE url_depth IS NOT NULL
            GROUP BY depth ORDER BY depth LIMIT 12
        """),
        ("url_length_hist", """
            SELECT LEAST(FLOOR(url_len / 20) * 20, 500) AS bin, COUNT(*) AS cnt
            FROM docs GROUP BY bin ORDER BY bin
        """),
        ("query_string", """
            SELECT CASE WHEN has_query = 1 THEN 'With Query Params'
                ELSE 'No Query Params' END AS qs, COUNT(*) AS cnt,
                ROUND(AVG(text_len), 0) AS avg_len
            FROM docs GROUP BY qs
        """),

        # ── 7. Quality ──
        ("quality_bands", """
            SELECT CASE
                WHEN language_score < 0.8 THEN '< 0.80'
                WHEN language_score < 0.9 THEN '0.80-0.90'
                WHEN language_score < 0.95 THEN '0.90-0.95'
                WHEN language_score < 0.99 THEN '0.95-0.99'
                ELSE '0.99-1.00'
            END AS band, COUNT(*) AS cnt,
                ROUND(AVG(text_len), 0) AS avg_len,
                ROUND(MEDIAN(text_len), 0) AS median_len
            FROM docs GROUP BY band ORDER BY band
        """),
        ("low_quality_domains", """
            SELECT domain, COUNT(*) AS cnt, ROUND(AVG(language_score), 4) AS avg_score,
                ROUND(AVG(text_len), 0) AS avg_len
            FROM docs WHERE language_score < 0.8 AND domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 15
        """),
        ("score_by_year", """
            SELECT year, ROUND(AVG(language_score), 4) AS avg_score,
                ROUND(MEDIAN(language_score), 4) AS median_score, COUNT(*) AS cnt
            FROM docs WHERE year IS NOT NULL
            GROUP BY year ORDER BY year
        """),
        ("score_percentiles", """
            SELECT
                PERCENTILE_CONT(0.05) WITHIN GROUP (ORDER BY language_score) AS P5,
                PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY language_score) AS P25,
                PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY language_score) AS P50,
                PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY language_score) AS P75,
                PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY language_score) AS P95
            FROM docs
        """),
        ("score_vs_text_len", """
            SELECT CASE
                WHEN text_len < 500 THEN '< 500'
                WHEN text_len < 2000 THEN '500-2K'
                WHEN text_len < 5000 THEN '2K-5K'
                WHEN text_len < 10000 THEN '5K-10K'
                ELSE '10K+'
            END AS len_bucket,
                ROUND(AVG(language_score), 4) AS avg_score,
                ROUND(MEDIAN(language_score), 4) AS median_score,
                COUNT(*) AS cnt
            FROM docs GROUP BY len_bucket
            ORDER BY CASE len_bucket
                WHEN '< 500' THEN 1 WHEN '500-2K' THEN 2 WHEN '2K-5K' THEN 3
                WHEN '5K-10K' THEN 4 ELSE 5 END
        """),

        # ── 8. Deduplication ──
        ("cluster_buckets", """
            SELECT CASE
                WHEN minhash_cluster_size = 1 THEN '1 (unique)'
                WHEN minhash_cluster_size = 2 THEN '2'
                WHEN minhash_cluster_size <= 5 THEN '3-5'
                WHEN minhash_cluster_size <= 10 THEN '6-10'
                WHEN minhash_cluster_size <= 50 THEN '11-50'
                WHEN minhash_cluster_size <= 100 THEN '51-100'
                ELSE '100+'
            END AS bucket, COUNT(*) AS cnt
            FROM docs GROUP BY bucket
            ORDER BY CASE bucket
                WHEN '1 (unique)' THEN 1 WHEN '2' THEN 2 WHEN '3-5' THEN 3
                WHEN '6-10' THEN 4 WHEN '11-50' THEN 5 WHEN '51-100' THEN 6
                ELSE 7 END
        """),
        ("cluster_cats", """
            SELECT CASE
                WHEN minhash_cluster_size = 1 THEN 'Unique (1)'
                WHEN minhash_cluster_size <= 5 THEN 'Small (2-5)'
                WHEN minhash_cluster_size <= 20 THEN 'Medium (6-20)'
                WHEN minhash_cluster_size <= 100 THEN 'Large (21-100)'
                ELSE 'Very Large (100+)'
            END AS category, COUNT(*) AS cnt
            FROM docs GROUP BY category ORDER BY cnt DESC
        """),
        ("dup_domains", """
            SELECT domain, COUNT(*) AS cnt,
                ROUND(AVG(minhash_cluster_size), 1) AS avg_cluster
            FROM docs WHERE minhash_cluster_size > 1 AND domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 15
        """),
        ("cluster_quality", """
            SELECT CASE
                WHEN minhash_cluster_size = 1 THEN 'Unique'
                WHEN minhash_cluster_size <= 5 THEN 'Small (2-5)'
                WHEN minhash_cluster_size <= 20 THEN 'Medium (6-20)'
                ELSE 'Large (20+)'
            END AS cluster_cat,
                ROUND(AVG(language_score), 4) AS avg_score,
                ROUND(AVG(text_len), 0) AS avg_len,
                COUNT(*) AS cnt
            FROM docs GROUP BY cluster_cat
            ORDER BY CASE cluster_cat
                WHEN 'Unique' THEN 1 WHEN 'Small (2-5)' THEN 2
                WHEN 'Medium (6-20)' THEN 3 ELSE 4 END
        """),
        ("unique_ratio", """
            SELECT
                SUM(CASE WHEN minhash_cluster_size = 1 THEN 1 ELSE 0 END) AS unique_docs,
                SUM(CASE WHEN minhash_cluster_size > 1 THEN 1 ELSE 0 END) AS dup_docs,
                COUNT(*) AS total
            FROM docs
        """),

        # ── 9. Vietnamese ──
        ("tones", """
            WITH chars AS (
                SELECT UNNEST(STRING_SPLIT(LEFT(text, 1000), '')) AS ch
                FROM docs USING SAMPLE 10000
            ), tones AS (
                SELECT CASE
                    WHEN ch IN ('á','ắ','ấ','é','ế','í','ó','ố','ớ','ú','ứ','ý') THEN 'Sắc (rising)'
                    WHEN ch IN ('à','ằ','ầ','è','ề','ì','ò','ồ','ờ','ù','ừ','ỳ') THEN 'Huyền (falling)'
                    WHEN ch IN ('ả','ẳ','ẩ','ẻ','ể','ỉ','ỏ','ổ','ở','ủ','ử','ỷ') THEN 'Hỏi (questioning)'
                    WHEN ch IN ('ã','ẵ','ẫ','ẽ','ễ','ĩ','õ','ỗ','ỡ','ũ','ữ','ỹ') THEN 'Ngã (tumbling)'
                    WHEN ch IN ('ạ','ặ','ậ','ẹ','ệ','ị','ọ','ộ','ợ','ụ','ự','ỵ') THEN 'Nặng (heavy)'
                    ELSE NULL
                END AS tone FROM chars
            )
            SELECT tone, COUNT(*) AS cnt FROM tones
            WHERE tone IS NOT NULL GROUP BY tone ORDER BY cnt DESC
        """),
        ("diacritics", """
            WITH chars AS (
                SELECT UNNEST(STRING_SPLIT(LEFT(text, 1000), '')) AS ch
                FROM docs USING SAMPLE 10000
            )
            SELECT CASE
                WHEN ch IN ('ă','ắ','ằ','ẳ','ẵ','ặ','Ă') THEN 'ă/Ă'
                WHEN ch IN ('â','ấ','ầ','ẩ','ẫ','ậ','Â') THEN 'â/Â'
                WHEN ch IN ('ê','ế','ề','ể','ễ','ệ','Ê') THEN 'ê/Ê'
                WHEN ch IN ('ô','ố','ồ','ổ','ỗ','ộ','Ô') THEN 'ô/Ô'
                WHEN ch IN ('ơ','ớ','ờ','ở','ỡ','ợ','Ơ') THEN 'ơ/Ơ'
                WHEN ch IN ('ư','ứ','ừ','ử','ữ','ự','Ư') THEN 'ư/Ư'
                WHEN ch IN ('đ','Đ') THEN 'đ/Đ'
                ELSE NULL
            END AS diacritic, COUNT(*) AS cnt
            FROM chars WHERE ch != ''
            GROUP BY diacritic HAVING diacritic IS NOT NULL
            ORDER BY cnt DESC
        """),
        ("viet_char_ratio", """
            WITH doc_stats AS (
                SELECT
                    CASE
                        WHEN language_score < 0.8 THEN 'Low (<0.8)'
                        WHEN language_score < 0.9 THEN 'Med (0.8-0.9)'
                        WHEN language_score < 0.95 THEN 'Good (0.9-0.95)'
                        ELSE 'High (>0.95)'
                    END AS quality_band,
                    LENGTH(REGEXP_REPLACE(LEFT(text, 500),
                        '[^àáảãạăắằẳẵặâấầẩẫậèéẻẽẹêếềểễệìíỉĩịòóỏõọôốồổỗộơớờởỡợùúủũụưứừửữựỳýỷỹỵđÀÁẢÃẠĂẮẰẲẴẶÂẤẦẨẪẬÈÉẺẼẸÊẾỀỂỄỆÌÍỈĨỊÒÓỎÕỌÔỐỒỔỖỘƠỚỜỞỠỢÙÚỦŨỤƯỨỪỬỮỰỲÝỶỸỴĐ]',
                        '', 'g')) AS viet_chars
                FROM docs USING SAMPLE 5000
            )
            SELECT quality_band,
                ROUND(AVG(100.0 * viet_chars / 500.0), 2) AS viet_pct,
                COUNT(*) AS cnt
            FROM doc_stats GROUP BY quality_band
            ORDER BY CASE quality_band
                WHEN 'Low (<0.8)' THEN 1 WHEN 'Med (0.8-0.9)' THEN 2
                WHEN 'Good (0.9-0.95)' THEN 3 ELSE 4 END
        """),
        ("vowel_patterns", """
            WITH chars AS (
                SELECT LOWER(UNNEST(STRING_SPLIT(LEFT(text, 1000), ''))) AS ch
                FROM docs USING SAMPLE 10000
            )
            SELECT ch AS vowel, COUNT(*) AS cnt
            FROM chars
            WHERE ch IN ('a','e','i','o','u','ă','â','ê','ô','ơ','ư','y')
            GROUP BY ch ORDER BY cnt DESC
        """),

        # ── 10. Content ──
        ("content_types", """
            SELECT content_type, COUNT(*) AS cnt,
                ROUND(AVG(text_len), 0) AS avg_len,
                ROUND(AVG(language_score), 4) AS avg_score
            FROM docs GROUP BY content_type ORDER BY cnt DESC
        """),
        ("news_domains", """
            SELECT domain, COUNT(*) AS cnt, ROUND(AVG(text_len), 0) AS avg_len
            FROM docs WHERE content_type = 'News' AND domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 15
        """),
        ("content_quality", """
            SELECT content_type,
                ROUND(AVG(language_score), 4) AS avg_score,
                ROUND(MEDIAN(language_score), 4) AS median_score,
                ROUND(AVG(text_len), 0) AS avg_len,
                COUNT(*) AS cnt
            FROM docs GROUP BY content_type
            HAVING cnt >= 10 ORDER BY avg_score DESC
        """),
        ("content_by_year", """
            SELECT year, content_type, COUNT(*) AS cnt
            FROM docs WHERE year IS NOT NULL
            GROUP BY year, content_type ORDER BY year, cnt DESC
        """),
        ("ecommerce_domains", """
            SELECT domain, COUNT(*) AS cnt, ROUND(AVG(text_len), 0) AS avg_len
            FROM docs WHERE content_type = 'E-commerce' AND domain IS NOT NULL
            GROUP BY domain ORDER BY cnt DESC LIMIT 15
        """),
    ]


# ── Chart Builders ─────────────────────────────────────────────

def _c1_overview(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 1: Overview — 5 charts."""
    charts = []
    ov = d.get("overview", [{}])[0]

    # 1.1 Key metrics bar
    metrics = [
        ("Documents", ov.get("total_docs", 0)),
        ("Characters", ov.get("total_chars", 0)),
        ("Words", ov.get("total_words", 0)),
        ("Domains", ov.get("unique_domains", 0)),
    ]
    fig = go.Figure(go.Bar(
        x=[m[0] for m in metrics], y=[m[1] for m in metrics],
        marker_color=PALETTE[:4], text=[_fmt(m[1]) for m in metrics],
        textposition="outside",
    ))
    fig.update_layout(**LAYOUT, title="Dataset Scale", yaxis_type="log",
                      yaxis_title="Count (log)", showlegend=False)
    charts.append(("Dataset Scale", fig))

    # 1.2 Field completeness
    comp = d.get("completeness", [{}])[0]
    total = comp.get("total", 1)
    fields = [(k.replace("has_", ""), v) for k, v in comp.items() if k.startswith("has_")]
    if fields:
        fig = go.Figure(go.Bar(
            y=[f[0] for f in fields],
            x=[100 * f[1] / total for f in fields],
            orientation="h", marker_color=GREEN,
            text=[f"{100 * f[1] / total:.1f}%" for f in fields],
            textposition="auto",
        ))
        fig.update_layout(**LAYOUT, title="Field Completeness (%)",
                          xaxis_range=[0, 105], xaxis_title="% Non-null")
    else:
        fig = _empty_fig("Field Completeness")
    charts.append(("Field Completeness", fig))

    # 1.3 Text length histogram
    rows = d.get("text_len_hist", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="bin", y="cnt", title="Text Length Distribution",
                     labels={"bin": "Characters", "cnt": "Documents"},
                     color_discrete_sequence=[BLUE])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("Text Length Distribution")
    charts.append(("Text Length Distribution", fig))

    # 1.4 Score histogram
    rows = d.get("score_hist", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="score_bin", y="cnt", title="Language Score Distribution",
                     labels={"score_bin": "Score", "cnt": "Documents"},
                     color_discrete_sequence=[TEAL])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("Language Score Distribution")
    charts.append(("Language Score Distribution", fig))

    # 1.5 Score vs Length scatter
    rows = d.get("sample_5k", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.scatter(df, x="text_len", y="language_score",
                         color="minhash_cluster_size", opacity=0.4,
                         title="Score vs Text Length (5K sample)",
                         labels={"text_len": "Length", "language_score": "Score"},
                         color_continuous_scale="Viridis")
        fig.update_layout(**LAYOUT, xaxis_type="log")
    else:
        fig = _empty_fig("Score vs Text Length")
    charts.append(("Score vs Text Length", fig))

    return charts


def _c2_text_length(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 2: Text Length — 5 charts."""
    charts = []

    # 2.1 Violin + box
    rows = d.get("sample_5k", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        fig.add_trace(go.Violin(
            y=df["text_len"], box_visible=True, meanline_visible=True,
            fillcolor=BLUE, opacity=0.6, line_color=BLUE, name="Length"))
        fig.update_layout(**LAYOUT, title="Text Length Distribution (Violin + Box)",
                          yaxis_title="Characters", showlegend=False)
    else:
        fig = _empty_fig("Text Length Violin")
    charts.append(("Text Length Violin", fig))

    # 2.2 Treemap buckets
    rows = d.get("text_len_buckets", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.treemap(df, path=["bucket"], values="cnt",
                         title="Document Length Buckets",
                         color="cnt", color_continuous_scale="Blues")
        fig.update_layout(**LAYOUT)
        fig.update_traces(textinfo="label+value+percent root")
    else:
        fig = _empty_fig("Length Buckets")
    charts.append(("Length Buckets", fig))

    # 2.3 Percentile ladder
    rows = d.get("percentiles", [])
    if rows:
        pct = rows[0]
        labels = [k for k in pct.keys()]
        values = [pct[k] for k in labels]
        fig = go.Figure(go.Bar(x=values, y=labels, orientation="h",
                               marker_color=INDIGO,
                               text=[_fmt(v) for v in values], textposition="auto"))
        fig.update_layout(**LAYOUT, title="Text Length Percentiles",
                          xaxis_title="Characters", xaxis_type="log")
    else:
        fig = _empty_fig("Percentiles")
    charts.append(("Percentiles", fig))

    # 2.4 Short docs domains
    rows = d.get("short_docs_domains", [])
    if rows:
        df = pd.DataFrame(rows[:12])
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title="Short Docs (< 500 chars) — Top Domains",
                     color="cnt", color_continuous_scale="Reds",
                     labels={"cnt": "Docs", "domain": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("Short Docs Domains")
    charts.append(("Short Docs Domains", fig))

    # 2.5 Long docs domains
    rows = d.get("long_docs_domains", [])
    if rows:
        df = pd.DataFrame(rows[:12])
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title="Long Docs (> 10K chars) — Top Domains",
                     color="avg_len" if "avg_len" in df.columns else "cnt",
                     color_continuous_scale="Greens",
                     labels={"cnt": "Docs", "domain": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("Long Docs Domains")
    charts.append(("Long Docs Domains", fig))

    return charts


def _c3_vocabulary(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 3: Vocabulary — 5 charts."""
    charts = []

    # 3.1 Top words
    rows = d.get("top_words", [])
    if rows:
        df = pd.DataFrame(rows[:30])
        fig = px.bar(df, x="cnt", y="word", orientation="h",
                     title="Top 30 Words (sampled)",
                     color="cnt", color_continuous_scale="Blues")
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          height=650, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Top Words")
    charts.append(("Top Words", fig))

    # 3.2 Char types donut
    rows = d.get("char_types", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.pie(df, values="cnt", names="char_type",
                     title="Character Type Distribution", hole=0.4,
                     color_discrete_sequence=PALETTE)
        fig.update_layout(**LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
    else:
        fig = _empty_fig("Character Types")
    charts.append(("Character Types", fig))

    # 3.3 Word length distribution
    rows = d.get("word_lengths", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="wlen", y="cnt", title="Word Length Distribution",
                     labels={"wlen": "Word Length (chars)", "cnt": "Count"},
                     color_discrete_sequence=[PURPLE])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("Word Length")
    charts.append(("Word Length", fig))

    # 3.4 Line count distribution
    rows = d.get("line_count_buckets", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="bucket", y="cnt", title="Lines per Document",
                     labels={"bucket": "Line Count", "cnt": "Documents"},
                     color_discrete_sequence=[TEAL])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("Lines per Document")
    charts.append(("Lines per Document", fig))

    # 3.5 Vietnamese stop words
    rows = d.get("viet_stop_words", [])
    if rows:
        df = pd.DataFrame(rows[:20])
        fig = px.bar(df, x="cnt", y="word", orientation="h",
                     title="Vietnamese Stop Word Frequency",
                     color="cnt", color_continuous_scale="Oranges")
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("Vietnamese Stop Words")
    charts.append(("Vietnamese Stop Words", fig))

    return charts


def _c4_temporal(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 4: Temporal — 5 charts."""
    charts = []

    # 4.1 By year
    rows = d.get("by_year", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="year", y="cnt", title="Documents per Year",
                     color="cnt", color_continuous_scale="Blues",
                     labels={"year": "Year", "cnt": "Documents"})
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Documents per Year")
    charts.append(("Documents per Year", fig))

    # 4.2 Monthly area
    rows = d.get("by_month", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.area(df, x="month", y="cnt", title="Monthly Document Trend",
                      labels={"month": "Month", "cnt": "Documents"},
                      color_discrete_sequence=[BLUE])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("Monthly Trend")
    charts.append(("Monthly Trend", fig))

    # 4.3 Hour x DOW heatmap
    rows = d.get("hour_dow", [])
    if rows:
        df = pd.DataFrame(rows)
        dow_order = ["Monday", "Tuesday", "Wednesday", "Thursday",
                     "Friday", "Saturday", "Sunday"]
        pivot = df.pivot_table(index="dow", columns="hour", values="cnt", fill_value=0)
        pivot = pivot.reindex(dow_order)
        fig = px.imshow(pivot, title="Crawl Activity Heatmap (Hour x Day)",
                        labels=dict(x="Hour (UTC)", y="Day", color="Docs"),
                        color_continuous_scale="YlOrRd", aspect="auto")
        fig.update_layout(**LAYOUT, height=350)
    else:
        fig = _empty_fig("Activity Heatmap")
    charts.append(("Activity Heatmap", fig))

    # 4.4 Top dumps
    rows = d.get("top_dumps", [])
    if rows:
        df = pd.DataFrame(rows[:15])
        fig = px.bar(df, x="cnt", y="dump", orientation="h",
                     title="Top 15 Common Crawl Dumps",
                     color="cnt", color_continuous_scale="Oranges",
                     labels={"cnt": "Documents", "dump": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("Top Dumps")
    charts.append(("Top Dumps", fig))

    # 4.5 Hour distribution
    rows = d.get("by_hour", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="hour", y="cnt", title="Crawl Hour Distribution (UTC)",
                     labels={"hour": "Hour", "cnt": "Documents"},
                     color_discrete_sequence=[INDIGO])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("Hour Distribution")
    charts.append(("Hour Distribution", fig))

    return charts


def _c5_domains(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 5: Domains — 5 charts."""
    charts = []

    # 5.1 Top domains treemap
    rows = d.get("top_domains", [])
    if rows:
        df = pd.DataFrame(rows[:20])
        fig = px.treemap(df, path=["domain"], values="cnt",
                         title="Top 20 Domains by Volume",
                         color="cnt", color_continuous_scale="Blues")
        fig.update_layout(**LAYOUT, height=500)
        fig.update_traces(textinfo="label+value+percent root")
    else:
        fig = _empty_fig("Top Domains")
    charts.append(("Top Domains", fig))

    # 5.2 Domain concentration (Lorenz curve)
    rows = d.get("domain_concentration", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        n = len(df)
        x_pct = [(i + 1) / n * 100 for i in range(n)]
        fig.add_trace(go.Scatter(x=x_pct, y=df["cum_pct"],
                                 mode="lines", line=dict(color=BLUE, width=2),
                                 name="Actual"))
        fig.add_trace(go.Scatter(x=[0, 100], y=[0, 100],
                                 mode="lines", line=dict(dash="dash", color="#ccc"),
                                 name="Perfect equality"))
        fig.update_layout(**LAYOUT, title="Domain Concentration (Lorenz Curve)",
                          xaxis_title="% of Domains (ranked)",
                          yaxis_title="Cumulative % of Documents")
    else:
        fig = _empty_fig("Domain Concentration")
    charts.append(("Domain Concentration", fig))

    # 5.3 Domain quality scatter
    rows = d.get("top_domains", [])
    if rows and "avg_score" in rows[0]:
        df = pd.DataFrame(rows[:20])
        fig = px.scatter(df, x="avg_len", y="avg_score", size="cnt",
                         text="domain", title="Top Domains: Quality vs Length",
                         labels={"avg_len": "Avg Length", "avg_score": "Avg Score"},
                         color_discrete_sequence=[BLUE], size_max=50)
        fig.update_traces(textposition="top center", textfont_size=9)
        fig.update_layout(**LAYOUT, showlegend=False)
    else:
        fig = _empty_fig("Domain Quality")
    charts.append(("Domain Quality", fig))

    # 5.4 Domain text length (top by length)
    rows = d.get("domain_text_len", [])
    if rows:
        df = pd.DataFrame(rows[:15])
        fig = px.bar(df, x="avg_len", y="domain", orientation="h",
                     title="Domains with Longest Avg Docs",
                     color="avg_len", color_continuous_scale="Greens",
                     labels={"avg_len": "Avg Length", "domain": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("Domain Text Length")
    charts.append(("Domain Text Length", fig))

    # 5.5 Domain size distribution
    rows = d.get("domain_size_dist", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="bucket", y="num_domains",
                     title="Domain Size Distribution (docs per domain)",
                     labels={"bucket": "Docs per Domain", "num_domains": "# Domains"},
                     color="num_domains", color_continuous_scale="Purples")
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Domain Size Distribution")
    charts.append(("Domain Size Distribution", fig))

    return charts


def _c6_url(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 6: URL & Web — 5 charts."""
    charts = []

    # 6.1 TLD donut
    rows = d.get("tld", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.pie(df, values="cnt", names="tld", title="TLD Distribution",
                     hole=0.35, color_discrete_sequence=PALETTE)
        fig.update_layout(**LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
    else:
        fig = _empty_fig("TLD Distribution")
    charts.append(("TLD Distribution", fig))

    # 6.2 Protocol
    rows = d.get("protocol_stats", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        fig.add_trace(go.Bar(x=df["protocol"], y=df["cnt"], name="Documents",
                             marker_color=BLUE, text=[_fmt(v) for v in df["cnt"]],
                             textposition="auto"))
        fig.update_layout(**LAYOUT, title="HTTP vs HTTPS", showlegend=False)
    else:
        fig = _empty_fig("Protocol")
    charts.append(("Protocol", fig))

    # 6.3 URL depth
    rows = d.get("url_depth", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="depth", y="cnt", title="URL Path Depth",
                     color="cnt", color_continuous_scale="Purples",
                     labels={"depth": "Path Depth", "cnt": "Documents"})
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("URL Depth")
    charts.append(("URL Depth", fig))

    # 6.4 URL length histogram
    rows = d.get("url_length_hist", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="bin", y="cnt", title="URL Length Distribution",
                     labels={"bin": "URL Length (chars)", "cnt": "Documents"},
                     color_discrete_sequence=[ORANGE])
        fig.update_layout(**LAYOUT)
    else:
        fig = _empty_fig("URL Length")
    charts.append(("URL Length", fig))

    # 6.5 Query string
    rows = d.get("query_string", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.pie(df, values="cnt", names="qs", title="Query Parameters Usage",
                     hole=0.4, color_discrete_sequence=[GREEN, RED])
        fig.update_layout(**LAYOUT)
        fig.update_traces(textinfo="percent+label+value")
    else:
        fig = _empty_fig("Query String")
    charts.append(("Query String", fig))

    return charts


def _c7_quality(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 7: Quality — 5 charts."""
    charts = []

    # 7.1 Quality bands dual-axis
    rows = d.get("quality_bands", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = make_subplots(specs=[[{"secondary_y": True}]])
        fig.add_trace(go.Bar(x=df["band"], y=df["cnt"], name="Documents",
                             marker_color=BLUE, opacity=0.8), secondary_y=False)
        fig.add_trace(go.Scatter(x=df["band"], y=df["avg_len"], name="Avg Length",
                                 mode="lines+markers", marker_color=RED,
                                 line=dict(width=3)), secondary_y=True)
        fig.update_layout(**LAYOUT, title_text="Quality Bands: Count & Avg Length")
        fig.update_yaxes(title_text="Documents", secondary_y=False)
        fig.update_yaxes(title_text="Avg Length", secondary_y=True)
    else:
        fig = _empty_fig("Quality Bands")
    charts.append(("Quality Bands", fig))

    # 7.2 Score vs text length bucketed
    rows = d.get("score_vs_text_len", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        fig.add_trace(go.Bar(x=df["len_bucket"], y=df["avg_score"], name="Avg Score",
                             marker_color=BLUE))
        fig.add_trace(go.Scatter(x=df["len_bucket"], y=df["median_score"],
                                 name="Median Score", mode="lines+markers",
                                 line=dict(color=RED, width=2)))
        fig.update_layout(**LAYOUT, title="Quality by Text Length Bucket",
                          yaxis_title="Language Score")
    else:
        fig = _empty_fig("Quality by Length")
    charts.append(("Quality by Length", fig))

    # 7.3 Low quality domains
    rows = d.get("low_quality_domains", [])
    if rows:
        df = pd.DataFrame(rows[:12])
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title="Low Quality (< 0.8 score) — Top Domains",
                     color="avg_score", color_continuous_scale="RdYlGn",
                     labels={"cnt": "Docs", "domain": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"))
    else:
        fig = _empty_fig("Low Quality Domains")
    charts.append(("Low Quality Domains", fig))

    # 7.4 Score by year
    rows = d.get("score_by_year", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        fig.add_trace(go.Scatter(x=df["year"], y=df["avg_score"], name="Avg Score",
                                 mode="lines+markers", line=dict(color=BLUE, width=2)))
        fig.add_trace(go.Scatter(x=df["year"], y=df["median_score"], name="Median",
                                 mode="lines+markers", line=dict(color=GREEN, width=2)))
        fig.update_layout(**LAYOUT, title="Language Score Trend by Year",
                          yaxis_title="Score")
    else:
        fig = _empty_fig("Score by Year")
    charts.append(("Score by Year", fig))

    # 7.5 Score percentiles indicator
    rows = d.get("score_percentiles", [])
    if rows:
        pct = rows[0]
        labels = list(pct.keys())
        values = [pct[k] for k in labels]
        fig = go.Figure(go.Bar(x=labels, y=values, marker_color=TEAL,
                               text=[f"{v:.4f}" for v in values], textposition="auto"))
        fig.update_layout(**LAYOUT, title="Language Score Percentiles",
                          yaxis_title="Score")
    else:
        fig = _empty_fig("Score Percentiles")
    charts.append(("Score Percentiles", fig))

    return charts


def _c8_dedup(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 8: Deduplication — 5 charts."""
    charts = []

    # 8.1 Cluster size buckets
    rows = d.get("cluster_buckets", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="bucket", y="cnt", title="MinHash Cluster Size Distribution",
                     color="cnt", color_continuous_scale="Blues",
                     labels={"bucket": "Cluster Size", "cnt": "Documents"})
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Cluster Size")
    charts.append(("Cluster Size", fig))

    # 8.2 Cluster categories donut
    rows = d.get("cluster_cats", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.pie(df, values="cnt", names="category",
                     title="Dedup Cluster Categories", hole=0.4,
                     color_discrete_sequence=PALETTE)
        fig.update_layout(**LAYOUT)
        fig.update_traces(textposition="inside", textinfo="percent+label")
    else:
        fig = _empty_fig("Cluster Categories")
    charts.append(("Cluster Categories", fig))

    # 8.3 Most duplicated domains
    rows = d.get("dup_domains", [])
    if rows:
        df = pd.DataFrame(rows[:12])
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title="Most Duplicated Domains",
                     color="avg_cluster", color_continuous_scale="Reds",
                     labels={"cnt": "Duplicate Docs", "domain": "",
                             "avg_cluster": "Avg Cluster"})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"))
    else:
        fig = _empty_fig("Duplicated Domains")
    charts.append(("Duplicated Domains", fig))

    # 8.4 Cluster quality relationship
    rows = d.get("cluster_quality", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        fig.add_trace(go.Bar(x=df["cluster_cat"], y=df["avg_score"], name="Avg Score",
                             marker_color=BLUE))
        fig.update_layout(**LAYOUT, title="Quality by Cluster Size",
                          yaxis_title="Avg Language Score")
    else:
        fig = _empty_fig("Cluster Quality")
    charts.append(("Cluster Quality", fig))

    # 8.5 Unique vs Duplicate ratio
    rows = d.get("unique_ratio", [])
    if rows:
        r = rows[0]
        fig = px.pie(
            values=[r.get("unique_docs", 0), r.get("dup_docs", 0)],
            names=["Unique", "Duplicated"],
            title="Unique vs Duplicated Documents", hole=0.4,
            color_discrete_sequence=[GREEN, RED])
        fig.update_layout(**LAYOUT)
        fig.update_traces(textinfo="percent+label+value")
    else:
        fig = _empty_fig("Unique vs Duplicate")
    charts.append(("Unique vs Duplicate", fig))

    return charts


def _c9_vietnamese(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 9: Vietnamese — 5 charts."""
    charts = []

    # 9.1 Tone radar
    rows = d.get("tones", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = go.Figure()
        fig.add_trace(go.Scatterpolar(
            r=df["cnt"], theta=df["tone"], fill="toself",
            fillcolor="rgba(26,115,232,0.2)", line_color=BLUE, name="Frequency"))
        fig.update_layout(**LAYOUT, title="Vietnamese Tone Radar",
                          polar=dict(radialaxis=dict(visible=True)))
    else:
        fig = _empty_fig("Tone Radar")
    charts.append(("Tone Radar", fig))

    # 9.2 Diacritics bar
    rows = d.get("diacritics", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="diacritic", y="cnt",
                     title="Vietnamese Diacritics Frequency",
                     color="cnt", color_continuous_scale="Blues",
                     labels={"diacritic": "Character", "cnt": "Count"})
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Diacritics")
    charts.append(("Diacritics", fig))

    # 9.3 Vowel patterns
    rows = d.get("vowel_patterns", [])
    if rows:
        df = pd.DataFrame(rows)
        colors = [BLUE if v in ("a", "e", "i", "o", "u", "y") else RED
                  for v in df["vowel"]]
        fig = go.Figure(go.Bar(x=df["vowel"], y=df["cnt"], marker_color=colors,
                               text=df["cnt"], textposition="auto"))
        fig.update_layout(**LAYOUT, title="Vowel Frequency (base + modified)",
                          yaxis_title="Count")
    else:
        fig = _empty_fig("Vowel Patterns")
    charts.append(("Vowel Patterns", fig))

    # 9.4 Vietnamese char ratio by quality
    rows = d.get("viet_char_ratio", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="quality_band", y="viet_pct",
                     title="Vietnamese Character % by Quality Band",
                     labels={"quality_band": "Quality", "viet_pct": "% Vietnamese Chars"},
                     color="viet_pct", color_continuous_scale="Greens")
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Vietnamese Char Ratio")
    charts.append(("Vietnamese Char Ratio", fig))

    # 9.5 Vietnamese stop words (from vocab data)
    rows = d.get("viet_stop_words", [])
    if rows:
        df = pd.DataFrame(rows[:15])
        fig = px.bar(df, x="cnt", y="word", orientation="h",
                     title="Vietnamese Function Word Frequency",
                     color="cnt", color_continuous_scale="Oranges")
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("Function Words")
    charts.append(("Function Words", fig))

    return charts


def _c10_content(d: dict) -> list[tuple[str, go.Figure]]:
    """Category 10: Content — 5 charts."""
    charts = []

    # 10.1 Content types bar
    rows = d.get("content_types", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="content_type", y="cnt", title="Content Type Distribution",
                     color="cnt", color_continuous_scale="Blues",
                     labels={"content_type": "Type", "cnt": "Documents"})
        fig.update_layout(**LAYOUT, coloraxis_showscale=False)
    else:
        fig = _empty_fig("Content Types")
    charts.append(("Content Types", fig))

    # 10.2 Content bubble (length vs quality vs volume)
    rows = d.get("content_types", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.scatter(df, x="avg_len", y="avg_score", size="cnt",
                         color="content_type", text="content_type",
                         title="Content Landscape: Length x Quality x Volume",
                         labels={"avg_len": "Avg Length", "avg_score": "Avg Score"},
                         color_discrete_sequence=PALETTE, size_max=60)
        fig.update_traces(textposition="top center", textfont_size=9)
        fig.update_layout(**LAYOUT, showlegend=False)
    else:
        fig = _empty_fig("Content Landscape")
    charts.append(("Content Landscape", fig))

    # 10.3 Content quality comparison
    rows = d.get("content_quality", [])
    if rows:
        df = pd.DataFrame(rows)
        fig = px.bar(df, x="avg_score", y="content_type", orientation="h",
                     title="Content Type Quality Ranking",
                     color="avg_score", color_continuous_scale="RdYlGn",
                     labels={"avg_score": "Avg Score", "content_type": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"))
    else:
        fig = _empty_fig("Content Quality")
    charts.append(("Content Quality", fig))

    # 10.4 News domains
    rows = d.get("news_domains", [])
    if rows:
        df = pd.DataFrame(rows[:12])
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title="Top News Domains",
                     color="avg_len" if "avg_len" in df.columns else "cnt",
                     color_continuous_scale="Oranges",
                     labels={"cnt": "Docs", "domain": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("News Domains")
    charts.append(("News Domains", fig))

    # 10.5 E-commerce domains
    rows = d.get("ecommerce_domains", [])
    if rows:
        df = pd.DataFrame(rows[:12])
        fig = px.bar(df, x="cnt", y="domain", orientation="h",
                     title="Top E-commerce Domains",
                     color="cnt", color_continuous_scale="Purples",
                     labels={"cnt": "Docs", "domain": ""})
        fig.update_layout(**LAYOUT, yaxis=dict(autorange="reversed"),
                          coloraxis_showscale=False)
    else:
        fig = _empty_fig("E-commerce Domains")
    charts.append(("E-commerce Domains", fig))

    return charts


CHART_BUILDERS = [
    _c1_overview, _c2_text_length, _c3_vocabulary, _c4_temporal,
    _c5_domains, _c6_url, _c7_quality, _c8_dedup,
    _c9_vietnamese, _c10_content,
]


def build_all_charts(data: dict) -> list[go.Figure]:
    """Build all 50 charts from cached data. Returns flat list of figures."""
    all_figs = []
    for builder in CHART_BUILDERS:
        pairs = builder(data)
        all_figs.extend([fig for _, fig in pairs])
    return all_figs


def build_all_chart_pairs(data: dict) -> list[tuple[str, go.Figure]]:
    """Build all charts with titles (for PDF export)."""
    all_pairs = []
    for builder in CHART_BUILDERS:
        all_pairs.extend(builder(data))
    return all_pairs


# ── PDF Export ──────────────────────────────────────────────────
def export_pdf(data_dir: str, lang: str, split: str, output: str | None = None) -> str:
    from fpdf import FPDF
    from fpdf.enums import XPos, YPos

    if output is None:
        output = os.path.join(
            data_dir, f"fineweb2_{lang}_{split}_{datetime.now():%Y%m%d_%H%M%S}.pdf"
        )

    cp = cache_path(data_dir, lang, split)
    if not os.path.exists(cp):
        print(f"No cache file: {cp}. Run --setup first.")
        return ""

    with open(cp) as f:
        data = json.load(f)

    ov = data.get("overview", [{}])[0]
    chart_pairs = build_all_chart_pairs(data)

    print(f"Rendering {len(chart_pairs)} charts...")
    tmpdir = tempfile.mkdtemp(prefix="fineweb_pdf_")
    chart_paths: list[tuple[str, str, int]] = []  # (title, path, category_idx)
    for i, (title, fig) in enumerate(chart_pairs):
        fig.update_layout(width=1000, height=550, template="plotly_white")
        path = os.path.join(tmpdir, f"chart_{i:02d}.png")
        fig.write_image(path, scale=2)
        cat_idx = i // 5  # 5 charts per category
        chart_paths.append((title, path, cat_idx))
        print(f"\r  [{i + 1}/{len(chart_pairs)}] {title:40s}", end="", flush=True)
    print()

    pdf = FPDF(orientation="L", unit="mm", format="A4")
    pdf.set_auto_page_break(auto=True, margin=15)
    NL = {"new_x": XPos.LMARGIN, "new_y": YPos.NEXT}

    # ── Title page ──────────────────────────────────────────
    pdf.add_page()
    # Blue header bar
    pdf.set_fill_color(26, 115, 232)  # #1a73e8
    pdf.rect(0, 0, 297, 50, "F")
    pdf.set_text_color(255, 255, 255)
    pdf.set_font("Helvetica", "B", 32)
    pdf.cell(0, 18, "", **NL)
    pdf.cell(0, 14, "FineWeb-2 Analytics Report", align="C", **NL)
    pdf.set_font("Helvetica", "", 14)
    pdf.cell(0, 10, f"{lang}  |  {split}  |  {datetime.now():%Y-%m-%d %H:%M}", align="C", **NL)

    # Reset colors
    pdf.set_text_color(32, 33, 36)  # #202124

    # Overview stats in a grid
    pdf.ln(8)
    pdf.set_font("Helvetica", "B", 16)
    pdf.cell(0, 10, "Dataset Overview", align="C", **NL)
    pdf.ln(3)

    stats = [
        ("Documents", ov.get("total_docs")),
        ("Characters", ov.get("total_chars")),
        ("Unique Domains", ov.get("unique_domains")),
        ("Avg Score", ov.get("avg_score")),
        ("Median Length", ov.get("median_len")),
    ]
    col_w = 297 / len(stats)
    # Stat values
    pdf.set_font("Helvetica", "B", 20)
    x_start = 0
    for _, val in stats:
        pdf.set_x(x_start)
        pdf.cell(col_w, 12, _fmt(val), align="C")
        x_start += col_w
    pdf.ln(12)
    # Stat labels
    pdf.set_font("Helvetica", "", 10)
    pdf.set_text_color(95, 99, 104)  # #5f6368
    x_start = 0
    for label, _ in stats:
        pdf.set_x(x_start)
        pdf.cell(col_w, 7, label, align="C")
        x_start += col_w
    pdf.ln(10)
    pdf.set_text_color(32, 33, 36)

    # Gray divider line
    pdf.set_draw_color(232, 234, 237)
    pdf.line(40, pdf.get_y(), 257, pdf.get_y())
    pdf.ln(6)

    # Table of contents — two columns
    pdf.set_font("Helvetica", "B", 14)
    pdf.cell(0, 9, "Categories", align="C", **NL)
    pdf.ln(2)
    pdf.set_font("Helvetica", "", 11)
    left_cats = CATEGORIES[:5]
    right_cats = CATEGORIES[5:]
    y_start = pdf.get_y()
    for idx, cat in enumerate(left_cats):
        chart_count = sum(1 for _, _, ci in chart_paths if ci == idx)
        pdf.set_x(40)
        pdf.cell(120, 7, f"{idx + 1}. {cat} ({chart_count} charts)")
        pdf.ln(7)
    pdf.set_y(y_start)
    for idx_off, cat in enumerate(right_cats):
        idx = idx_off + 5
        chart_count = sum(1 for _, _, ci in chart_paths if ci == idx)
        pdf.set_x(160)
        pdf.cell(120, 7, f"{idx + 1}. {cat} ({chart_count} charts)")
        pdf.ln(7)

    # ── Chart pages with category dividers ──────────────────
    prev_cat = -1
    for title, path, cat_idx in chart_paths:
        # Category divider page
        if cat_idx != prev_cat:
            prev_cat = cat_idx
            pdf.add_page()
            cat_name = CATEGORIES[cat_idx] if cat_idx < len(CATEGORIES) else f"Category {cat_idx + 1}"
            # Blue accent bar
            pdf.set_fill_color(26, 115, 232)
            pdf.rect(20, 60, 257, 4, "F")
            pdf.set_font("Helvetica", "B", 32)
            pdf.set_text_color(26, 115, 232)
            pdf.cell(0, 70, "", **NL)
            pdf.cell(0, 16, f"{cat_idx + 1}. {cat_name}", align="C", **NL)
            pdf.set_text_color(95, 99, 104)
            pdf.set_font("Helvetica", "", 14)
            chart_count = sum(1 for _, _, ci in chart_paths if ci == cat_idx)
            pdf.cell(0, 10, f"{chart_count} charts", align="C", **NL)
            pdf.set_text_color(32, 33, 36)

        # Chart page
        pdf.add_page()
        pdf.set_font("Helvetica", "B", 14)
        pdf.set_text_color(32, 33, 36)
        pdf.cell(0, 10, title, align="C", **NL)
        pdf.ln(1)
        pdf.image(path, x=(297 - 260) / 2, w=260)

    pdf.output(output)
    for _, path, _ in chart_paths:
        os.unlink(path)
    os.rmdir(tmpdir)

    size_mb = os.path.getsize(output) / (1024 * 1024)
    total_pages = 1 + len(CATEGORIES) + len(chart_paths)  # title + dividers + charts
    print(f"PDF saved: {output} ({size_mb:.1f} MB, {total_pages} pages)")
    return output


# ── Gradio App ─────────────────────────────────────────────────
def create_app(data_dir: str, lang: str = "vie_Latn"):
    import gradio as gr

    cache_store: dict[str, dict] = {}

    # Pre-load caches
    for split in ["test", "train"]:
        cp = cache_path(data_dir, lang, split)
        if os.path.exists(cp):
            with open(cp) as f:
                cache_store[split] = json.load(f)
            print(f"Loaded cache: {cp}")

    with gr.Blocks(title="FineWeb-2 Analytics") as app:
        gr.Markdown("# FineWeb-2 Vietnamese Analytics Dashboard")

        with gr.Row():
            split_radio = gr.Radio(["test", "train"], value="test",
                                   label="Dataset Split", interactive=True)
            status_md = gr.Markdown("Ready")
            export_btn = gr.Button("Export PDF", variant="secondary",
                                   scale=0, min_width=120)
            pdf_output = gr.File(label="PDF", visible=False)

        overview_md = gr.Markdown("")

        # Create 10 tabs with 5 plots each
        all_plots = []
        with gr.Tabs():
            for cat_name in CATEGORIES:
                with gr.Tab(cat_name):
                    with gr.Row():
                        all_plots.append(gr.Plot())
                        all_plots.append(gr.Plot())
                    with gr.Row():
                        all_plots.append(gr.Plot())
                        all_plots.append(gr.Plot())
                    all_plots.append(gr.Plot())

        def update_all(split_val):
            data = cache_store.get(split_val)
            if data is None:
                cp = cache_path(data_dir, lang, split_val)
                if os.path.exists(cp):
                    with open(cp) as f:
                        data = json.load(f)
                    cache_store[split_val] = data

            if data is None:
                msg = f"No data for {split_val}. Run: `uv run app.py --setup`"
                return [msg, msg] + [_empty_fig(f"No data ({split_val})")] * 50

            ov = data.get("overview", [{}])[0]
            overview = (
                f"**{_fmt(ov.get('total_docs'))}** documents  |  "
                f"**{_fmt(ov.get('total_chars'))}** characters  |  "
                f"**{_fmt(ov.get('unique_domains'))}** domains  |  "
                f"Avg score **{ov.get('avg_score', 'N/A')}**  |  "
                f"Median length **{_fmt(ov.get('median_len'))}**"
            )
            status = f"Loaded {split_val} split"

            figs = build_all_charts(data)
            return [status, overview] + figs

        def do_export(s):
            path = export_pdf(data_dir, lang, s)
            if path:
                return gr.File(value=path, visible=True)
            return gr.File(visible=False)

        outputs = [status_md, overview_md] + all_plots
        split_radio.change(update_all, split_radio, outputs)
        app.load(update_all, split_radio, outputs)
        export_btn.click(do_export, split_radio, pdf_output)

    launch_kwargs = dict(
        theme=gr.themes.Soft(primary_hue="blue"),
        css=".gradio-container { max-width: 1400px !important; }",
    )
    return app, launch_kwargs


# ── Setup Pipeline ─────────────────────────────────────────────
def run_setup(data_dir: str, lang: str):
    """Full pipeline: import parquet to DuckDB, compute analytics cache.

    Download data first with the CLI:
        search download get --lang vie_Latn --split train --shards 2
        search download get --lang vie_Latn --split test
    """
    pipe = Pipeline("FineWeb-2 Setup")
    print(f"\nFineWeb-2 Analytics Setup")
    print(f"  Data dir: {data_dir}")
    print(f"  Language: {lang}\n")

    # Check for old analytics.duckdb
    old_db = os.path.join(data_dir, lang, "analytics.duckdb")
    if os.path.exists(old_db):
        print(f"  Note: Old {old_db} found. Using split-specific DBs now.\n")

    # Check parquet files exist
    found_splits = []
    for split in ["test", "train"]:
        parquet_dir = os.path.join(data_dir, lang, split)
        if os.path.isdir(parquet_dir) and any(
            f.endswith(".parquet") for f in os.listdir(parquet_dir)
        ):
            found_splits.append(split)
        else:
            print(f"  Skipping {split}: no parquet files in {parquet_dir}")
            print(f"  Download with: search download get --lang {lang} --split {split}")

    if not found_splits:
        print("\n  No data found. Download first with:")
        print(f"    search download get --lang {lang} --split train --shards 2")
        print(f"    search download get --lang {lang} --split test")
        return

    # Import
    for split in found_splits:
        with pipe.step(f"Import {split} -> {split}.duckdb"):
            import_to_duckdb(data_dir, lang, split)

    # Compute cache
    for split in found_splits:
        dbp = db_path(data_dir, lang, split)
        if not os.path.exists(dbp):
            continue
        with pipe.step(f"Compute {split} analytics ({len(_all_queries())} queries)"):
            compute_cache(data_dir, lang, split)

    pipe.summary()


# ── CLI ────────────────────────────────────────────────────────
if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="FineWeb-2 Analytics Dashboard (50 charts, 10 categories)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  uv run app.py --setup                    Import parquet + compute cache
  uv run app.py                            Launch Gradio dashboard
  uv run app.py --export-pdf --split test  Export PDF report
  uv run app.py --import-db                Import parquet -> DuckDB only
  uv run app.py --compute-cache            Compute analytics cache only

Download data first with the CLI:
  search download get --lang vie_Latn --split train --shards 2
  search download get --lang vie_Latn --split test
        """,
    )
    parser.add_argument("--data-dir", default=os.path.expanduser("~/data/fineweb-2"))
    parser.add_argument("--lang", default="vie_Latn")
    parser.add_argument("--port", type=int, default=7860)
    parser.add_argument("--share", action="store_true")
    parser.add_argument("--setup", action="store_true", help="Import + compute cache")
    parser.add_argument("--import-db", action="store_true", help="Import parquet to DuckDB")
    parser.add_argument("--compute-cache", action="store_true", help="Compute JSON cache")
    parser.add_argument("--export-pdf", action="store_true", help="Export PDF report")
    parser.add_argument("--split", default="test")
    args = parser.parse_args()

    if args.setup:
        run_setup(args.data_dir, args.lang)
    elif args.import_db:
        for split in ["test", "train"]:
            print(f"Importing {split}...")
            import_to_duckdb(args.data_dir, args.lang, split)
    elif args.compute_cache:
        for split in ["test", "train"]:
            print(f"Computing cache for {split}...")
            compute_cache(args.data_dir, args.lang, split)
    elif args.export_pdf:
        export_pdf(args.data_dir, args.lang, args.split)
    else:
        import gradio as gr
        app, launch_kwargs = create_app(args.data_dir, args.lang)
        app.launch(server_port=args.port, share=args.share, **launch_kwargs)
