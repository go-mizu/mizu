# 0725 — Publish Open Index to Hugging Face

**Date:** 2026-03-13
**Status:** In progress (66/900 shards published)
**Repo:** `open-index/cc-main-2026-08` (draft)

---

## Pipeline Summary

Each CC-MAIN shard goes through four stages:

```
download WARC (S3)
  → pack HTML → Markdown WARC (light engine)
  → export Parquet
  → commit to Hugging Face
  → delete raw WARC + md.warc.gz + local parquet (aggressive cleanup)
```

Command: `search cc publish --pipeline --cleanup --file <range>`

---

## Observed Stats (66 shards, CC-MAIN-2026-08)

| Metric | Per shard | Total (66 shards) |
|---|---|---|
| Rows (docs) | ~19,700 | 1,302,370 |
| Raw HTML | ~2.5 GB | 162 GB |
| Markdown (.md.warc.gz) | ~88 MB | 5.7 GB |
| Parquet (.parquet) | ~28 MB | 1.86 GB |
| HTML → Markdown compression | **96.5%** | |
| Markdown → Parquet compression | **67.9%** | |
| Overall HTML → Parquet | **98.9%** | |

### Timing per shard (avg)

| Stage | Time |
|---|---|
| Download WARC from S3 | 74s |
| Pack HTML → Markdown | 14s |
| Export Parquet | 18s |
| Publish to HF | 42s |
| **Total per shard** | **~149s (2.5 min)** |

---

## Projection: Full CC-MAIN-2026-08 (~900 shards)

| Metric | Projected total |
|---|---|
| Total documents | **~17.8 million** |
| Total raw HTML processed | ~2,200 GB (2.2 TB) |
| Total Markdown generated | ~77 GB |
| **Total Parquet on HF** | **~25 GB (900 × 28 MB)** |
| HF dataset size | ~25 GB |

### Time estimate

| Scenario | Estimate |
|---|---|
| 1 session (serial) | ~35h |
| 7 parallel sessions (current setup) | **~5h remaining** |
| Full 900 shards from scratch (7 sessions) | **~5–6h** |

Current sessions running in parallel on server2:
- `s37_100` (64 shards)
- `s101_250` (150 shards)
- `s251_400` (150 shards)
- `s401_550` (150 shards)
- `s551_700` (150 shards)
- `s701_850` (150 shards)
- `s851_1000` (150 shards)

As of 2026-03-13 ~08:00 UTC, 66 shards are committed. All 7 sessions are actively downloading/packing (shards 42, 255, 554, 705, 855 in progress).

---

## Disk Management (aggressive cleanup)

Each shard requires peak ~2.6 GB during processing:
- Raw WARC: ~800 MB → deleted after pack (`--cleanup`)
- md.warc.gz: ~88 MB → deleted after export
- Parquet: ~28 MB → deleted after successful HF commit

Steady-state disk per concurrent session: **~900 MB** (WARC download phase).
With 7 sessions: peak ~6 GB concurrent disk usage.

---

## Recovery

If a shard's local parquet was deleted after HF commit, reconstruct md.warc.gz via:

```bash
search cc pull --file <idx>              # download from HF + reconstruct
search cc pull --file <idx> --delete-local  # also clean up local parquet after
```

The parquet contains all 8 WARC headers needed for reconstruction:
`WARC-Target-URI`, `WARC-Date`, `WARC-Record-ID`, `WARC-Refers-To`,
`Content-Type`, `Content-Length`, `X-HTML-Length`, `WARC-Type`.

---

## HF Commit Strategy

- Each shard gets its own commit (parquet + README + stats.csv)
- Charts (PNG) only generated and committed on the **last shard** of a session to avoid conflicts
- `huggingface_hub` auto-retries HTTP 412 (concurrent commit conflicts) — parallel sessions are safe
- Commit message: `Publish shard CC-MAIN-2026-08/NNNNN`

---

## Notes

- CC-MAIN-2026-08 has exactly **900 WARC index files** (`00000`–`00899`)
- Shard 0 (file index 0): 19,498 docs, 39.9 MB parquet — representative
- `dur_download_s` = 0 for old-format rows (pre-v0.5.26); timing only captured from new binary
- Parquet schema: `doc_id, url, host, crawl_date, warc_record_id, warc_refers_to, html_length, markdown_length, markdown`
