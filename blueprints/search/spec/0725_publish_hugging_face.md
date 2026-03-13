# 0725 — Publish Open Index to Hugging Face

**Date:** 2026-03-13
**Status:** In progress — ~270 shards on HF, 7 sessions running on server2
**Repo:** `open-index/draft`
**Scale:** CC-MAIN-2026-08 = 100,000 WARC files × ~19,300 pages = ~1.93 billion pages total

---

## Dataset Scale (measured from 270 shards)

| Metric | Per shard (measured) | × 100,000 shards |
|---|---|---|
| Raw WARC (.warc.gz) | ~800 MB compressed | ~80 TB |
| HTML (uncompressed) | ~2.56 GB | ~256 TB |
| Markdown (.md.warc.gz) | ~93 MB | ~9.3 TB |
| **Final parquet (Zstd lv19)** | **~29.6 MB** | **~2.96 TB** |
| Rows (documents) | ~19,298 | ~1.93 billion |
| HTML → Markdown | **−96.4%** | |
| Markdown → Parquet | **−68.1%** | |
| HTML → Parquet overall | **−98.8%** | |

CC-MAIN-2026-08: **100 segments × 1,000 WARC files** = 100,000 total. File indices 0–99,999 map directly into `warc.paths.gz`.

---

## Measured Pipeline Timings (stats.csv, 270 shards)

| Stage | Avg | Median | n | Notes |
|---|---|---|---|---|
| Download WARC from S3 | **147s** | 136s | 220 | bottleneck — ~5 MB/s per session |
| Pack HTML → Markdown | **67s** | 72s | 17 | light engine (~290 docs/s) |
| Export to Parquet | **28s** | 23s | 238 | I/O-bound |
| Publish to HF | **81s avg / 34s median** | 34s | 212 | avg inflated by old batch=1 runs; median=34s reflects batch=10 |
| **Total per shard** | **252s avg** | **214s** | 220 | |

> At 7 sessions × 252s/shard → **~100 shards/h** current throughput on server2.

---

## Current Status (2026-03-13 ~12:30 UTC)

| Metric | Value |
|---|---|
| Shards on HF | ~270 (stats.csv has 270, max idx=890) |
| Sessions running | **7** on server2 (started 11:34 UTC) |
| Server1 | **⚠ Unreachable** — host key changed (`ssh-keyscan` needed) |
| Server2 binary | `v0.5.26-382-g87c17f9a` (UUID5 doc_id ✓) |
| New deploy pending | `btc2wklpj` — adds stats.csv HF merge; kill/restart sessions when done |

### Active sessions (server2, as of 12:30 UTC)

| Session | Range | Current file | Remaining |
|---|---|---|---|
| g48_100 | 0048–0100 | 00050 | ~51 |
| g149_250 | 0149–0250 | 00150 | ~102 |
| g292_400 | 0292–0400 | 00294 | ~108 |
| g425_550 | 0425–0550 | 00427 | ~125 |
| g563_700 | 0563–0700 | 00565 (exporting) | ~136 |
| g721_850 | 0721–0850 | 00723 (exporting) | ~129 |
| g889_999 | 0889–0999 | 00891 | ~110 |
| **Total remaining 0–999** | | | **~761** |

- ETA to complete 0–999: ~761/100 ≈ **7.6 hours** (~20:00 UTC tonight)
- Files 1000–99,999: not yet started (need server1 fix + more sessions)

---

## Time Estimates (updated with measured data)

| Scenario | Per shard | Shards/h | ETA 100k shards |
|---|---|---|---|
| **Current** (7 sessions, server2) | 252s | ~100/h | **~41 days** |
| + Fix server1 (14 sessions) | 252s | ~200/h | **~21 days** |
| + Prefetch (hidden download) | 181s¹ | ~140/h per server | **~30 days** (1 server) |
| 2 servers + prefetch | 181s | ~280/h | **~15 days** |
| 4 servers + prefetch | 181s | ~560/h | **~7.4 days** |
| 10 servers + prefetch | 181s | ~1,400/h | **~3 days** |
| **<1 day target** | 181s | ~4,170/h | requires **~30 servers** |

¹ With prefetch: effective = max(147s download, 67+28s pack+export) + 34s publish = 147+34 = **181s/shard**. Download still dominates — need more bandwidth per server to improve further.

> **Key bottleneck: download at 147s (58% of total).** Each server shares bandwidth across sessions (~5 MB/s per session). Prefetch hides pack+export behind download but can't reduce download itself. More servers = more bandwidth = faster.

---

## Bottleneck Analysis

### #1 — Download (147s, 58% of wall time)
- Server2 has ~35 MB/s total; 7 sessions × ~5 MB/s → 800 MB / 5 = 160s (matches measured 147s)
- **Fix A: More servers** — each gets own bandwidth (30+ MB/s) → download drops to ~27s
- **Fix B: Prefetch** — overlap download of shard N+1 with pack+export of N (saves ~67+28=95s overlap)

### #2 — Pack + Export (67+28 = 95s, 38% of wall time)
- Light engine ~290 docs/s is CPU-bound; export ~28s is I/O-bound
- **Fix: Prefetch** hides this completely behind download

### #3 — Publish to HF (34s amortized at batch=10, 13%)
- batch=10 already implemented and working
- Further improvement possible with batch=20+, but diminishing returns

---

## Optimization Roadmap

### ✅ Implemented
- `--commit-batch N` — batch N parquets per HF commit; median publish 34s at batch=10 (was 81s avg at batch=1)
- `doc_id = UUID5(NamespaceURL, url)` — deterministic, stable across crawls (2026-03-13)
- `search cc pull` — reverse pipeline: HF parquet → md.warc.gz
- Cleanup: raw WARC + md.warc.gz + local parquet deleted after commit
- Charts: `size_chart.png` (HTML vs Markdown), `totals_chart.png`, `timing_chart.png`
- `stats.csv` HF sync (pending deploy `btc2wklpj`) — each session pulls from HF before commit, merges remote rows; HF is single source of truth across all servers

### 🔴 Immediate: Fix server1 access
```bash
ssh-keyscan <server1-ip> >> ~/.ssh/known_hosts
```
Then start 9 sessions for files 1001–1900 (iteration 1–9 of the 90-iteration strategy).
**Impact: 2× throughput immediately.**

### 🔴 Implement prefetch download
In `ccRunPipelineWithCommits`: start downloading shard N+1 in a background goroutine while packing shard N.
- Implementation: buffered channel `prefetchCh chan string` (cap=1), goroutine pre-downloads next WARC
- Impact: 252s → 181s per shard = **28% faster**

### 🟡 More bandwidth per session
- With 7 sessions sharing 35 MB/s → 5 MB/s each; 800 MB / 30 MB/s = 27s download
- Reduce sessions per server to 2–3 to give each more bandwidth
- Or use servers in different datacenters with separate uplinks

---

## 90-Iteration Strategy (files 1001–10,000)

Each iteration = 100 files. Measure real timing, update spec, then proceed.

### Iteration schedule

| Iterations | Files | Focus | Expected gain |
|---|---|---|---|
| 1–9 | 1001–1900 | Baseline measurement (batch=10, current binary) | 0% |
| 10–18 | 1901–2800 | Implement & measure prefetch download | ~28% |
| 19–27 | 2801–3700 | Fix server1, add 2nd server in parallel | ~2× cumulative |
| 28–36 | 3701–4600 | Tune sessions-per-server vs bandwidth tradeoff | ~10% |
| 37–45 | 4601–5500 | Increase batch size to 20–50 | ~5% |
| 46–54 | 5501–6400 | Add 3rd server | ~3× cumulative |
| 55–63 | 6401–7300 | Add 4th server | ~4× cumulative |
| 64–72 | 7301–8200 | Tune workers / light engine parameters | ~5% |
| 73–81 | 8201–9100 | Stable config validation | — |
| 82–90 | 9101–10,000 | Production config, ready to scale to 100k | — |

### Iteration commands (server2, after completing 0–999)
```bash
export HF_TOKEN=<your-hf-token>

# Iterations 1–3 (batch=10 baseline, measure)
screen -dmS iter01 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 1001-1100 >> /tmp/iter01.log 2>&1"
screen -dmS iter02 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 1101-1200 >> /tmp/iter02.log 2>&1"
# ... continue for 1201-1900 with 7 sessions
```

---

## Session Management

### Restart all sessions (server2) after binary deploy
```bash
export HF_TOKEN=<your-hf-token>
# Kill existing sessions first:
screen -ls | grep -E 'g[0-9]' | awk '{print $1}' | xargs -I{} screen -S {} -X quit

# Restart covering all gaps in 0–999:
screen -dmS g48_100  bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 48-100   >> /tmp/g48_100.log  2>&1"
screen -dmS g149_250 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 149-250  >> /tmp/g149_250.log 2>&1"
screen -dmS g292_400 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 292-400  >> /tmp/g292_400.log 2>&1"
screen -dmS g425_550 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 425-550  >> /tmp/g425_550.log 2>&1"
screen -dmS g563_700 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 563-700  >> /tmp/g563_700.log 2>&1"
screen -dmS g721_850 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 721-850  >> /tmp/g721_850.log 2>&1"
screen -dmS g889_999 bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file 889-999  >> /tmp/g889_999.log 2>&1"
```

### Fix server1 and start iteration sessions
```bash
# On local machine:
ssh-keyscan <server1-ip> >> ~/.ssh/known_hosts

# On server1, start 9 sessions for iterations 1–9:
export HF_TOKEN=<your-hf-token>
for range in "1001-1100" "1101-1200" "1201-1300" "1301-1400" "1401-1500" "1501-1600" "1601-1700" "1701-1800" "1801-1900"; do
  name="iter_${range/-/_}"
  screen -dmS $name bash -c "export HF_TOKEN=$HF_TOKEN; ~/bin/search cc publish --pipeline --cleanup --commit-batch 10 --file $range >> /tmp/${name}.log 2>&1"
done
```

---

## Migration Notes

See `MIGRATION_NOTES.md` in the HF repo for the doc_id formula change (2026-03-13).

- **Shards 00000–00888**: `doc_id` = UUID from WARC-Record-ID (old formula)
- **Shards 00889+**: `doc_id` = `UUID5(NamespaceURL, url)` (current formula)
- **Workaround**: recompute from `url` column: `uuid.uuid5(uuid.NAMESPACE_URL, row["url"])`
