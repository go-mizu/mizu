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
  - size_chart.png      subplot: HTML (GB) and Parquet (MB) per shard
  - rows_chart.png      bar chart: document count per shard
  - timing_chart.png    stacked bar: download / convert / export / publish per shard
  - compression_pie.png donut chart: cumulative size breakdown
"""

import argparse
import sys
from pathlib import Path

import pandas as pd
import plotly.graph_objects as go
from plotly.subplots import make_subplots


# ── style ────────────────────────────────────────────────────────────────────

COLORS = ["#6366f1", "#10b981", "#f59e0b", "#06b6d4", "#8b5cf6", "#f43f5e"]

LAYOUT = dict(
    font_family="Inter, system-ui, -apple-system, Segoe UI, sans-serif",
    font_size=12,
    font_color="#374151",
    plot_bgcolor="#fafbfc",
    paper_bgcolor="#ffffff",
    margin=dict(l=56, r=24, t=56, b=72),
)

AXIS = dict(gridcolor="#f0f0f0", gridwidth=1, zeroline=False, linecolor="#e5e7eb")
XAXIS = dict(**AXIS, tickangle=-45, tickfont_size=9)


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
    """Two-panel chart: HTML (GB) on top, Parquet (MB) on bottom — separate scales."""
    fig = make_subplots(
        rows=2, cols=1, vertical_spacing=0.14,
        subplot_titles=("Raw HTML per Shard (GB)", "Final Parquet per Shard (MB)"),
    )
    fig.add_trace(go.Bar(
        x=df["shard"], y=df["html_bytes"] / 1e9,
        marker_color=COLORS[0], marker_line_width=0, showlegend=False,
    ), row=1, col=1)
    fig.add_trace(go.Bar(
        x=df["shard"], y=df["parquet_bytes"] / 1e6,
        marker_color=COLORS[1], marker_line_width=0, showlegend=False,
    ), row=2, col=1)
    fig.update_layout(
        **LAYOUT, height=560, width=max(900, len(df) * 26),
        title_text="Size per Shard",
    )
    fig.update_xaxes(**XAXIS)
    fig.update_yaxes(title_text="GB", row=1, col=1, **AXIS)
    fig.update_yaxes(title_text="MB", row=2, col=1, **AXIS)
    p = Path(out) / "size_chart.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


def chart_rows(df, out):
    fig = go.Figure(go.Bar(
        x=df["shard"], y=df["rows"] / 1e3,
        marker_color=COLORS[0], marker_line_width=0,
        text=[f"{v:.0f}K" for v in df["rows"] / 1e3],
        textposition="outside", textfont_size=8, textfont_color="#9ca3af",
    ))
    fig.update_layout(
        **LAYOUT, height=420, width=max(900, len(df) * 26),
        title_text="Document Count per Shard",
        yaxis_title="Documents (thousands)",
    )
    fig.update_xaxes(**XAXIS)
    fig.update_yaxes(**AXIS)
    p = Path(out) / "rows_chart.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


def chart_timings(df, out):
    cols = ["dur_download_s", "dur_convert_s", "dur_export_s", "dur_publish_s"]
    names = ["Download", "Convert (HTML to MD)", "Export Parquet", "Publish HF"]
    if sum(df[c].sum() for c in cols) == 0:
        print("  Skipping timing chart (no timing data)")
        return
    fig = go.Figure()
    for i, (col, name) in enumerate(zip(cols, names)):
        fig.add_trace(go.Bar(
            x=df["shard"], y=df[col] / 60, name=name,
            marker_color=COLORS[i], marker_line_width=0,
        ))
    fig.update_layout(
        **LAYOUT, barmode="stack",
        height=450, width=max(900, len(df) * 26),
        title_text="Pipeline Time per Shard",
        yaxis_title="Time (minutes)",
        legend=dict(orientation="h", yanchor="bottom", y=1.02, xanchor="right", x=1,
                    font_size=10, bgcolor="rgba(255,255,255,0.9)"),
    )
    fig.update_xaxes(**XAXIS)
    fig.update_yaxes(**AXIS)
    p = Path(out) / "timing_chart.png"
    fig.write_image(str(p), scale=2)
    print(f"  Wrote {p}")


def chart_compression(df, out):
    total_html = df["html_bytes"].sum()
    total_md = df["md_bytes"].sum()
    total_pq = df["parquet_bytes"].sum()
    stripped = total_html - total_md
    compressed = total_md - total_pq
    labels = [
        f"Stripped (HTML to MD) - {fmt_bytes(stripped)}",
        f"Compressed (Parquet) - {fmt_bytes(compressed)}",
        f"Final Parquet - {fmt_bytes(total_pq)}",
    ]
    fig = go.Figure(go.Pie(
        labels=labels, values=[stripped, compressed, total_pq],
        hole=0.55, marker_colors=COLORS[:3],
        marker_line=dict(color="white", width=2.5),
        textinfo="percent", textfont_size=13, textfont_color="#374151",
        sort=False,
    ))
    fig.update_layout(
        **LAYOUT, height=480, width=720,
        title_text="Compression Breakdown",
        legend=dict(orientation="v", yanchor="middle", y=0.5, xanchor="left", x=1.02,
                    font_size=11),
        annotations=[dict(
            text=f"{fmt_bytes(total_html)}<br>to {fmt_bytes(total_pq)}",
            x=0.44, y=0.5, font_size=14, font_color="#374151", showarrow=False,
        )],
    )
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
    chart_rows(df, args.out)
    chart_timings(df, args.out)
    chart_compression(df, args.out)

    print("Done.")


if __name__ == "__main__":
    main()
