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
  - size_chart.png      grouped bar: HTML vs Markdown vs Parquet bytes per shard
  - rows_chart.png      bar chart: document count per shard
  - timing_chart.png    stacked bar: download / convert / export / publish per shard
  - compression_pie.png donut chart: cumulative size breakdown
"""

import argparse
import sys
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import pandas as pd


# ── modern theme ─────────────────────────────────────────────────────────────

# Tailwind-inspired palette: indigo, emerald, amber, cyan, violet, rose
PALETTE = ["#6366f1", "#10b981", "#f59e0b", "#06b6d4", "#8b5cf6", "#f43f5e"]

def apply_theme():
    plt.rcParams.update({
        "figure.facecolor": "#ffffff",
        "figure.dpi": 150,
        "axes.facecolor": "#fafbfc",
        "axes.edgecolor": "#e5e7eb",
        "axes.linewidth": 0.8,
        "axes.grid": True,
        "axes.grid.axis": "y",
        "axes.spines.top": False,
        "axes.spines.right": False,
        "axes.titlesize": 13,
        "axes.titleweight": "bold",
        "axes.titlepad": 16,
        "axes.labelsize": 10,
        "axes.labelpad": 8,
        "grid.color": "#f0f0f0",
        "grid.linewidth": 0.6,
        "font.family": "sans-serif",
        "font.sans-serif": ["Inter", "Segoe UI", "Helvetica Neue", "Arial"],
        "font.size": 10,
        "xtick.labelsize": 8,
        "ytick.labelsize": 9,
        "xtick.color": "#6b7280",
        "ytick.color": "#6b7280",
        "legend.frameon": True,
        "legend.framealpha": 0.95,
        "legend.edgecolor": "#e5e7eb",
        "legend.fontsize": 9,
        "legend.borderpad": 0.6,
    })


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
    fig, ax1 = plt.subplots(figsize=(max(10, len(df) * 0.55), 5.5))
    x = range(len(df))
    w = 0.3
    ax1.bar([i - w / 2 for i in x], df["html_bytes"] / 1e9, width=w,
            label="HTML (GB)", color=PALETTE[0], alpha=0.88, edgecolor="white", linewidth=0.5)
    ax1.set_ylabel("HTML size (GB)", color=PALETTE[0])
    ax1.tick_params(axis="y", labelcolor=PALETTE[0])

    ax2 = ax1.twinx()
    ax2.bar([i + w / 2 for i in x], df["parquet_bytes"] / 1e6, width=w,
            label="Parquet (MB)", color=PALETTE[1], alpha=0.88, edgecolor="white", linewidth=0.5)
    ax2.set_ylabel("Parquet size (MB)", color=PALETTE[1])
    ax2.tick_params(axis="y", labelcolor=PALETTE[1])
    ax2.spines["right"].set_visible(True)
    ax2.spines["right"].set_color("#e5e7eb")

    ax1.set_xticks(list(x))
    ax1.set_xticklabels(df["shard"].tolist(), rotation=45, ha="right")
    ax1.set_title("Size per Shard: HTML vs Parquet")
    lines1, labels1 = ax1.get_legend_handles_labels()
    lines2, labels2 = ax2.get_legend_handles_labels()
    ax1.legend(lines1 + lines2, labels1 + labels2, loc="upper right")
    fig.tight_layout()
    fig.savefig(Path(out) / "size_chart.png")
    plt.close(fig)
    print(f"  Wrote {Path(out) / 'size_chart.png'}")

def chart_rows(df, out):
    fig, ax = plt.subplots(figsize=(max(10, len(df) * 0.55), 5))
    x = range(len(df))
    bars = ax.bar(x, df["rows"] / 1e3, color=PALETTE[0], alpha=0.88,
                  edgecolor="white", linewidth=0.5)
    peak = (df["rows"] / 1e3).max()
    for bar in bars:
        h = bar.get_height()
        if h > 0:
            ax.text(bar.get_x() + bar.get_width() / 2, h + peak * 0.012,
                    f"{h:.0f}K", ha="center", va="bottom", fontsize=7, color="#9ca3af")
    ax.set_xticks(list(x))
    ax.set_xticklabels(df["shard"].tolist(), rotation=45, ha="right")
    ax.set_ylabel("Documents (thousands)")
    ax.set_title("Document Count per Shard")
    fig.tight_layout()
    fig.savefig(Path(out) / "rows_chart.png")
    plt.close(fig)
    print(f"  Wrote {Path(out) / 'rows_chart.png'}")

def chart_timings(df, out):
    cols = ["dur_download_s", "dur_convert_s", "dur_export_s", "dur_publish_s"]
    if sum(df[c].sum() for c in cols) == 0:
        print("  Skipping timing chart (no timing data)")
        return
    fig, ax = plt.subplots(figsize=(max(10, len(df) * 0.55), 5))
    x = range(len(df))
    labels = ["Download", "Convert (HTML to MD)", "Export Parquet", "Publish HF"]
    bottom = pd.Series([0.0] * len(df))
    for i, (col, lbl) in enumerate(zip(cols, labels)):
        vals = df[col] / 60
        ax.bar(x, vals, bottom=bottom, label=lbl,
               color=PALETTE[i], alpha=0.88, edgecolor="white", linewidth=0.5)
        bottom = bottom + vals
    ax.set_xticks(list(x))
    ax.set_xticklabels(df["shard"].tolist(), rotation=45, ha="right")
    ax.set_ylabel("Time (minutes)")
    ax.set_title("Pipeline Time per Shard")
    ax.legend(loc="upper right")
    fig.tight_layout()
    fig.savefig(Path(out) / "timing_chart.png")
    plt.close(fig)
    print(f"  Wrote {Path(out) / 'timing_chart.png'}")

def chart_compression_pie(df, out):
    total_html = df["html_bytes"].sum()
    total_md = df["md_bytes"].sum()
    total_pq = df["parquet_bytes"].sum()
    stripped = total_html - total_md
    compressed = total_md - total_pq

    labels = [
        f"Stripped (HTML to MD)  {fmt_bytes(stripped)}",
        f"Compressed (Parquet)  {fmt_bytes(compressed)}",
        f"Final Parquet  {fmt_bytes(total_pq)}",
    ]
    sizes = [stripped, compressed, total_pq]

    fig, ax = plt.subplots(figsize=(8, 5.5))
    wedges, texts, autotexts = ax.pie(
        sizes, colors=PALETTE[:3],
        autopct="%1.1f%%", startangle=140, pctdistance=0.82,
        wedgeprops={"linewidth": 2.5, "edgecolor": "white", "width": 0.45},
    )
    for t in autotexts:
        t.set_fontsize(10)
        t.set_fontweight("bold")
        t.set_color("#374151")
    ax.legend(wedges, labels, loc="center left", bbox_to_anchor=(0.85, 0.5),
              fontsize=9, frameon=True, framealpha=0.95, edgecolor="#e5e7eb")
    ax.text(0, 0, f"{fmt_bytes(total_html)}\nto {fmt_bytes(total_pq)}",
            ha="center", va="center", fontsize=11, fontweight="bold", color="#374151")
    ax.set_title("Compression Breakdown", pad=16)
    fig.tight_layout()
    fig.savefig(Path(out) / "compression_pie.png")
    plt.close(fig)
    print(f"  Wrote {Path(out) / 'compression_pie.png'}")


# ── main ─────────────────────────────────────────────────────────────────────

def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("stats_csv", help="Path to stats.csv")
    ap.add_argument("--out", default="charts", help="Output directory (default: charts/)")
    ap.add_argument("--crawl", default="", help="Filter to a single crawl ID")
    args = ap.parse_args()

    apply_theme()
    df = load(args.stats_csv)
    if args.crawl:
        df = df[df["crawl_id"] == args.crawl].reset_index(drop=True)
        if df.empty:
            sys.exit(f"No rows for crawl {args.crawl!r}")

    Path(args.out).mkdir(parents=True, exist_ok=True)
    print(f"Generating charts for {len(df)} shards -> {args.out}/")

    chart_sizes(df, args.out)
    chart_rows(df, args.out)
    chart_timings(df, args.out)
    chart_compression_pie(df, args.out)

    print("Done.")

if __name__ == "__main__":
    main()
