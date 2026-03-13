#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = ["plotly", "kaleido", "pandas"]
# ///
"""
Generate charts from the Open Index stats.csv.

Usage:
    python chart_stats.py stats.csv [--out charts/]

Outputs PNG images suitable for embedding in README / HuggingFace model cards:
  - size_chart.png        grouped bar: HTML vs Markdown bytes per shard
  - totals_chart.png      horizontal bar: total HTML / Markdown / Parquet with reductions
  - timing_chart.png      stacked bar: download / convert / export / publish per shard
  - compression_pie.png   horizontal funnel bar: pipeline compression stages
"""

import argparse
import sys
from pathlib import Path

import pandas as pd
import plotly.express as px


# ── style ────────────────────────────────────────────────────────────────────

COLORS = ["#6366f1", "#10b981", "#f59e0b", "#06b6d4", "#8b5cf6", "#f43f5e"]

LAYOUT = dict(
    font_family="Inter, system-ui, -apple-system, Segoe UI, sans-serif",
    font_size=12,
    font_color="#374151",
    plot_bgcolor="#fafbfc",
    paper_bgcolor="#ffffff",
)


# ── helpers ──────────────────────────────────────────────────────────────────

def fmt_bytes(n):
    for unit in ("B", "KB", "MB", "GB", "TB"):
        if abs(n) < 1024:
            return f"{n:.1f} {unit}"
        n /= 1024
    return f"{n:.1f} PB"


def load(path):
    df = pd.read_csv(path)
    required = {"crawl_id", "file_idx", "rows", "html_bytes", "md_bytes", "parquet_bytes"}
    missing = required - set(df.columns)
    if missing:
        sys.exit(f"Missing columns: {missing}")
    for col in ("dur_download_s", "dur_convert_s", "dur_export_s", "dur_publish_s"):
        if col not in df.columns:
            df[col] = 0
    if "dur_pack_s" in df.columns and df["dur_convert_s"].sum() == 0:
        df["dur_convert_s"] = df["dur_pack_s"]
    df = df.sort_values(["crawl_id", "file_idx"]).reset_index(drop=True)
    df["shard"] = df["crawl_id"].str[-3:] + "/" + df["file_idx"].apply(lambda x: f"{x:05d}")
    return df


# ── charts ───────────────────────────────────────────────────────────────────

def chart_sizes(df, out):
    """Grouped bar: HTML vs Markdown bytes per shard."""
    plot_df = pd.DataFrame({
        "shard": list(df["shard"]) * 2,
        "bytes_mb": list(df["html_bytes"] / 1e6) + list(df["md_bytes"] / 1e6),
        "type": ["Raw HTML"] * len(df) + ["Markdown"] * len(df),
    })
    fig = px.bar(
        plot_df, x="shard", y="bytes_mb", color="type", barmode="group",
        color_discrete_sequence=[COLORS[0], COLORS[1]],
        title="Size per Shard: HTML vs Markdown (MB)",
        labels={"bytes_mb": "Size (MB)", "shard": "", "type": ""},
        height=450, width=max(900, len(df) * 26),
    )
    fig.update_layout(**LAYOUT,
        margin=dict(l=56, r=24, t=56, b=72),
        legend=dict(orientation="h", yanchor="bottom", y=1.02, xanchor="right", x=1,
                    font_size=10, bgcolor="rgba(255,255,255,0.9)"),
        xaxis=dict(tickangle=-45, tickfont_size=9, gridcolor="#f0f0f0", linecolor="#e5e7eb"),
        yaxis=dict(gridcolor="#f0f0f0", zeroline=False, linecolor="#e5e7eb"),
    )
    fig.update_traces(marker_line_width=0)
    p = Path(out) / "size_chart.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


def chart_totals(df, out):
    """Horizontal bar: total HTML / Markdown / Parquet with % reduction labels."""
    total_html = df["html_bytes"].sum()
    total_md = df["md_bytes"].sum()
    total_pq = df["parquet_bytes"].sum()

    pct_md = (1 - total_md / total_html) * 100 if total_html else 0
    pct_pq = (1 - total_pq / total_html) * 100 if total_html else 0
    pct_pq_from_md = (1 - total_pq / total_md) * 100 if total_md else 0

    plot_df = pd.DataFrame({
        "stage": ["Raw HTML", "Markdown", "Parquet"],
        "size_gb": [total_html / 1e9, total_md / 1e9, total_pq / 1e9],
        "label": [
            fmt_bytes(total_html),
            f"{fmt_bytes(total_md)}  (−{pct_md:.1f}%)",
            f"{fmt_bytes(total_pq)}  (−{pct_pq:.1f}% overall, −{pct_pq_from_md:.1f}% vs MD)",
        ],
    })
    fig = px.bar(
        plot_df, x="size_gb", y="stage", orientation="h", text="label",
        color="stage", color_discrete_sequence=[COLORS[0], COLORS[1], COLORS[2]],
        title=f"Total Size: HTML vs Markdown vs Parquet ({len(df)} shards)",
        labels={"size_gb": "Size (GB)", "stage": ""},
        category_orders={"stage": ["Parquet", "Markdown", "Raw HTML"]},
        height=300, width=820,
    )
    fig.update_layout(**LAYOUT,
        margin=dict(l=80, r=24, t=56, b=56),
        showlegend=False,
        xaxis=dict(range=[0, plot_df["size_gb"].max() * 1.45],
                   gridcolor="#f0f0f0", zeroline=False, linecolor="#e5e7eb"),
        yaxis=dict(gridcolor="#f0f0f0", linecolor="#e5e7eb"),
    )
    fig.update_traces(marker_line_width=0, textposition="outside",
                      textfont=dict(size=11, color="#374151"))
    p = Path(out) / "totals_chart.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


def chart_timings(df, out):
    """Stacked bar: pipeline time per shard."""
    cols = ["dur_download_s", "dur_convert_s", "dur_export_s", "dur_publish_s"]
    names = ["Download", "Convert (HTML→MD)", "Export Parquet", "Publish HF"]
    if sum(df[c].sum() for c in cols) == 0:
        print("  Skipping timing chart (no timing data)")
        return

    plot_df = pd.DataFrame({
        "shard": df["shard"].tolist() * len(cols),
        "minutes": [v for col in cols for v in (df[col] / 60).tolist()],
        "stage": [name for name in names for _ in range(len(df))],
    })
    fig = px.bar(
        plot_df, x="shard", y="minutes", color="stage", barmode="stack",
        color_discrete_sequence=COLORS[:len(cols)],
        title="Pipeline Time per Shard",
        labels={"minutes": "Time (minutes)", "shard": "", "stage": ""},
        height=450, width=max(900, len(df) * 26),
        category_orders={"stage": names},
    )
    fig.update_layout(**LAYOUT,
        margin=dict(l=56, r=24, t=56, b=72),
        legend=dict(orientation="h", yanchor="bottom", y=1.02, xanchor="right", x=1,
                    font_size=10, bgcolor="rgba(255,255,255,0.9)"),
        xaxis=dict(tickangle=-45, tickfont_size=9, gridcolor="#f0f0f0", linecolor="#e5e7eb"),
        yaxis=dict(gridcolor="#f0f0f0", zeroline=False, linecolor="#e5e7eb"),
    )
    fig.update_traces(marker_line_width=0)
    p = Path(out) / "timing_chart.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


def chart_compression(df, out):
    """Horizontal funnel bar: pipeline compression stages with reduction labels."""
    total_html = df["html_bytes"].sum()
    total_md = df["md_bytes"].sum()
    total_pq = df["parquet_bytes"].sum()

    if total_html == 0:
        print("  Skipping compression chart (no html_bytes data)")
        return

    # Estimate packed WARC as ~47% of uncompressed markdown
    total_packed = int(total_md * 0.47)

    pct_html_to_packed = (1 - total_packed / total_html) * 100
    pct_packed_to_pq = (1 - total_pq / total_packed) * 100 if total_packed else 0
    pct_end_to_end = (1 - total_pq / total_html) * 100

    plot_df = pd.DataFrame({
        "stage": ["HTML (uncompressed)", "Packed WARC (.md.warc.gz)", "Parquet (Zstd lv19)"],
        "size_gb": [total_html / 1e9, total_packed / 1e9, total_pq / 1e9],
        "label": [
            fmt_bytes(total_html),
            f"{fmt_bytes(total_packed)}  (−{pct_html_to_packed:.1f}% vs HTML)",
            f"{fmt_bytes(total_pq)}  (−{pct_packed_to_pq:.1f}% vs packed, −{pct_end_to_end:.1f}% overall)",
        ],
    })
    fig = px.bar(
        plot_df, x="size_gb", y="stage", orientation="h", text="label",
        color="stage", color_discrete_sequence=[COLORS[0], COLORS[1], COLORS[2]],
        title=f"Compression Pipeline ({len(df)} shards)",
        labels={"size_gb": "Size (GB)", "stage": ""},
        category_orders={"stage": ["Parquet (Zstd lv19)", "Packed WARC (.md.warc.gz)", "HTML (uncompressed)"]},
        height=320, width=900,
    )
    fig.update_layout(**LAYOUT,
        margin=dict(l=180, r=24, t=60, b=56),
        showlegend=False,
        xaxis=dict(range=[0, plot_df["size_gb"].max() * 1.55],
                   gridcolor="#f0f0f0", zeroline=False, linecolor="#e5e7eb"),
        yaxis=dict(gridcolor="#f0f0f0", linecolor="#e5e7eb"),
    )
    fig.update_traces(marker_line_width=0, textposition="outside",
                      textfont=dict(size=11, color="#374151"))
    p = Path(out) / "compression_pie.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


# ── main ─────────────────────────────────────────────────────────────────────

def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("stats_csv", help="Path to stats.csv")
    ap.add_argument("--out", default="charts", help="Output directory (default: charts/)")
    ap.add_argument("--crawl", default="", help="Filter to a single crawl ID")
    args = ap.parse_args()

    df = load(args.stats_csv)
    if args.crawl:
        df = df[df["crawl_id"] == args.crawl].reset_index(drop=True)
        if df.empty:
            sys.exit(f"No rows for crawl {args.crawl!r}")

    Path(args.out).mkdir(parents=True, exist_ok=True)
    print(f"Generating charts for {len(df)} shards -> {args.out}/")

    chart_sizes(df, args.out)
    chart_totals(df, args.out)
    chart_timings(df, args.out)
    chart_compression(df, args.out)

    print("Done.")


if __name__ == "__main__":
    main()
