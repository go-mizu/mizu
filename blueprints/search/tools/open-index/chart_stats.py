#!/usr/bin/env python3
# /// script
# requires-python = ">=3.10"
# dependencies = ["matplotlib", "pandas"]
# ///
"""
Generate charts from the Open Index stats.csv.

Usage:
    python chart_stats.py stats.csv [--out charts/]

Outputs PNG images suitable for embedding in README / HuggingFace model cards:
  - size_chart.png      bar chart: HTML vs Markdown vs Parquet bytes per shard
  - rows_chart.png      bar chart: document count per shard
  - timing_chart.png    stacked bar: pack / export / publish seconds per shard
  - compression_pie.png pie chart: cumulative size breakdown
"""

import argparse
import csv
import os
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import pandas as pd


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
    # Timing columns are optional (added in later pipeline versions)
    for col in ("dur_pack_s", "dur_export_s", "dur_publish_s"):
        if col not in df.columns:
            df[col] = 0
    df = df.sort_values(["crawl_id", "file_idx"]).reset_index(drop=True)
    df["shard"] = df["crawl_id"].str[-3:] + "/" + df["file_idx"].apply(lambda x: f"{x:05d}")
    return df


# ── charts ───────────────────────────────────────────────────────────────────

PALETTE = ["#4e8ef7", "#34c38f", "#f46a6a", "#f1b44c", "#74788d"]

def chart_sizes(df, out):
    fig, ax = plt.subplots(figsize=(max(8, len(df) * 0.45), 5))
    x = range(len(df))
    w = 0.25
    ax.bar([i - w for i in x], df["html_bytes"] / 1e9,   width=w, label="HTML (GB)", color=PALETTE[0])
    ax.bar([i      for i in x], df["md_bytes"] / 1e9,    width=w, label="Markdown (GB)", color=PALETTE[1])
    ax.bar([i + w for i in x], df["parquet_bytes"] / 1e6, width=w, label="Parquet (MB×0.001)", color=PALETTE[2])
    ax.set_xticks(list(x))
    ax.set_xticklabels(df["shard"].tolist(), rotation=45, ha="right", fontsize=8)
    ax.set_ylabel("Size")
    ax.set_title("Size per shard: HTML vs Markdown vs Parquet")
    ax.legend()
    ax.yaxis.set_major_formatter(mticker.FormatStrFormatter("%.1f"))
    fig.tight_layout()
    p = Path(out) / "size_chart.png"
    fig.savefig(p, dpi=150)
    plt.close(fig)
    print(f"  Wrote {p}")

def chart_rows(df, out):
    fig, ax = plt.subplots(figsize=(max(8, len(df) * 0.45), 4))
    ax.bar(range(len(df)), df["rows"] / 1e3, color=PALETTE[0])
    ax.set_xticks(range(len(df)))
    ax.set_xticklabels(df["shard"].tolist(), rotation=45, ha="right", fontsize=8)
    ax.set_ylabel("Documents (thousands)")
    ax.set_title("Document count per shard")
    fig.tight_layout()
    p = Path(out) / "rows_chart.png"
    fig.savefig(p, dpi=150)
    plt.close(fig)
    print(f"  Wrote {p}")

def chart_timings(df, out):
    has_timing = (df["dur_pack_s"] + df["dur_export_s"] + df["dur_publish_s"]).sum() > 0
    if not has_timing:
        print("  Skipping timing chart (no timing data in CSV)")
        return
    fig, ax = plt.subplots(figsize=(max(8, len(df) * 0.45), 4))
    x = range(len(df))
    ax.bar(x, df["dur_pack_s"] / 60,    label="Pack (min)", color=PALETTE[0])
    ax.bar(x, df["dur_export_s"] / 60,  bottom=df["dur_pack_s"] / 60, label="Export (min)", color=PALETTE[1])
    ax.bar(x, df["dur_publish_s"] / 60, bottom=(df["dur_pack_s"] + df["dur_export_s"]) / 60, label="Publish (min)", color=PALETTE[2])
    ax.set_xticks(list(x))
    ax.set_xticklabels(df["shard"].tolist(), rotation=45, ha="right", fontsize=8)
    ax.set_ylabel("Time (minutes)")
    ax.set_title("Pipeline time per shard: Pack / Export / Publish")
    ax.legend()
    fig.tight_layout()
    p = Path(out) / "timing_chart.png"
    fig.savefig(p, dpi=150)
    plt.close(fig)
    print(f"  Wrote {p}")

def chart_compression_pie(df, out):
    total_html = df["html_bytes"].sum()
    total_md = df["md_bytes"].sum()
    total_pq = df["parquet_bytes"].sum()
    stripped = total_html - total_md
    compressed = total_md - total_pq

    labels = [
        f"Stripped by HTML→MD\n({fmt_bytes(stripped)})",
        f"Compressed by Parquet\n({fmt_bytes(compressed)})",
        f"Final Parquet\n({fmt_bytes(total_pq)})",
    ]
    sizes = [stripped, compressed, total_pq]
    colors = [PALETTE[0], PALETTE[1], PALETTE[2]]

    fig, ax = plt.subplots(figsize=(7, 5))
    wedges, texts, autotexts = ax.pie(
        sizes, labels=labels, colors=colors,
        autopct="%1.1f%%", startangle=140,
        wedgeprops={"linewidth": 1, "edgecolor": "white"},
    )
    for t in autotexts:
        t.set_fontsize(9)
    ax.set_title(f"Cumulative size breakdown\n(from {fmt_bytes(total_html)} HTML → {fmt_bytes(total_pq)} Parquet)")
    fig.tight_layout()
    p = Path(out) / "compression_pie.png"
    fig.savefig(p, dpi=150)
    plt.close(fig)
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
    print(f"Generating charts for {len(df)} shards → {args.out}/")

    chart_sizes(df, args.out)
    chart_rows(df, args.out)
    chart_timings(df, args.out)
    chart_compression_pie(df, args.out)

    print("Done.")

if __name__ == "__main__":
    main()
