# FTS Pipeline

Data flows through 4 stages. Each stage depends on the previous one.

```
.warc.gz ──1:download──► warc/
warc/    ──2:pack──────► warc_md/*.md.warc.gz + *.meta.duckdb
warc_md/ ──3:index─────► fts/{engine}/{shard}/
fts/     ──4:dashboard─► localhost:3457
```

## Storage layout

```
~/data/common-crawl/{crawl}/
├── warc/                          # Stage 1: raw WARC files (~1GB each)
│   └── CC-MAIN-...-00000.warc.gz
├── warc_md/                       # Stage 2: markdown WARC + per-shard metadata
│   ├── 00000.md.warc.gz           #   seekable concat-gzip (one gzip member per doc)
│   └── 00000.meta.duckdb          #   per-doc metadata for browse (built during pack)
└── fts/                           # Stage 3: FTS indexes
    └── dahlia/
        └── 00000/                 #   one shard per WARC file
```

---

## CLI reference

```bash
# Stage 1: Download
search cc warc download --file 0-3

# Stage 2: Pack (HTML → Markdown, builds meta.duckdb automatically)
search cc warc pack --file 0-3

# Stage 3: Index
search cc fts index --file 0-3

# Stage 4: Dashboard
search cc fts dashboard --port 3457 --open
```

---

## API reference (dashboard must be running on port 3457)

All jobs are created via `POST /api/jobs` and polled via `GET /api/jobs/{id}`.
The `files` field accepts `"0"`, `"0-3"`, or `"all"`.

### Stage 1: Download WARCs

```bash
curl -s -X POST http://localhost:3457/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"download","files":"0-3"}' | python3 -m json.tool
```

### Stage 2: Pack (WARC → Markdown + meta.duckdb)

```bash
curl -s -X POST http://localhost:3457/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"markdown","files":"0-3"}' | python3 -m json.tool
```

> Note: `type:"markdown"` is the pack step that produces `.md.warc.gz` files.

### Stage 3: Build FTS index

```bash
curl -s -X POST http://localhost:3457/api/jobs \
  -H 'Content-Type: application/json' \
  -d '{"type":"index","engine":"dahlia","source":"files","files":"0-3"}' | python3 -m json.tool
```

### Poll job status

```bash
curl -s http://localhost:3457/api/jobs/{id} | python3 -m json.tool
# status: queued | running | completed | failed | cancelled
# progress: 0.0–1.0
```

### List all jobs

```bash
curl -s http://localhost:3457/api/jobs | python3 -m json.tool
```

### Cancel a job

```bash
curl -s -X DELETE http://localhost:3457/api/jobs/{id}
```

### Search (engine must be indexed)

```bash
curl -s "http://localhost:3457/api/search?q=machine+learning&engine=dahlia" \
  | python3 -m json.tool
```

### Browse documents in a shard

```bash
# List shards
curl -s http://localhost:3457/api/browse | python3 -m json.tool

# List docs in shard 00000 (paginated)
curl -s "http://localhost:3457/api/browse?shard=00000&page=1&page_size=20" \
  | python3 -m json.tool

# Fetch a single document
curl -s http://localhost:3457/api/doc/00000/{doc_id}
```

### Overview / pipeline status

```bash
curl -s http://localhost:3457/api/overview | python3 -m json.tool
```

### Available engines

```bash
curl -s http://localhost:3457/api/engines
# {"engines":["dahlia","tantivy"]}
```

### WARC file list and per-file actions

```bash
# List all WARC files with stage flags
curl -s http://localhost:3457/api/warc | python3 -m json.tool

# Detail for one file
curl -s http://localhost:3457/api/warc/00000 | python3 -m json.tool

# Trigger a per-file action (download|pack|index|delete)
curl -s -X POST http://localhost:3457/api/warc/00000/action \
  -H 'Content-Type: application/json' \
  -d '{"action":"index","engine":"dahlia","source":"files"}' \
  | python3 -m json.tool
```

---

## CLI ↔ API equivalence

| CLI command | API call |
|---|---|
| `warc download --file 0-3` | `POST /api/jobs {"type":"download","files":"0-3"}` |
| `warc pack --file 0-3` | `POST /api/jobs {"type":"markdown","files":"0-3"}` |
| `fts index --file 0-3 --engine dahlia` | `POST /api/jobs {"type":"index","engine":"dahlia","source":"files","files":"0-3"}` |
| `fts search "query"` | `GET /api/search?q=query&engine=dahlia` |

---

## Notes

- `warc_md/*.meta.duckdb` is built automatically during `warc pack` (no separate step)
- Engines: `dahlia` (default, embedded), `tantivy` (requires `-tags tantivy` build)
- `--crawl` flag defaults to the latest crawl; override with `--crawl CC-MAIN-2026-08`
- The old `markdown/{shard}/` individual-file format is no longer supported
