# 0670: WARC Markdown Pack

## Goal

`search cc warc pack` — single-pass pipeline that reads `.warc.gz` files, converts HTML→Markdown, and writes seekable `.md.warc.gz` files (one WARC record per gzip member, matching Common Crawl's format).

## Output Format

Each record is a WARC `conversion` record:

```
WARC/1.1
WARC-Type: conversion
WARC-Target-URI: https://example.com/page
WARC-Date: 2026-01-15T23:13:59Z
WARC-Record-ID: <urn:uuid:NEW-UUID>
WARC-Refers-To: <urn:uuid:ORIGINAL-ID>
Content-Type: text/markdown
Content-Length: 1234

# Page Title

Markdown content here...
```

### Seekable Gzip

Each WARC record is wrapped in its own gzip member (write gzip → close → next gzip → close). This produces a concatenated-gzip file where each member can be decompressed independently given its byte offset — identical to Common Crawl's `.warc.gz` format.

## Architecture

### Pipeline (single-pass, parallel conversion)

```
.warc.gz → [reader] → filter(response, 200, text/html) → [N converter workers] → [single writer] → .md.warc.gz
```

1. **Reader goroutine** (sequential, gzip constraint): reads WARC response records, parses HTTP, extracts HTML body + WARC headers. Sends `packItem` structs to a buffered channel.
2. **Converter pool** (N workers, CPU-bound): receives `packItem`, runs trafilatura/go-readability, sends `packResult` to output channel.
3. **Writer goroutine** (sequential, file I/O): receives `packResult`, writes WARC conversion record wrapped in individual gzip member.

Ordering: output records are NOT guaranteed to be in the same order as input (parallel conversion reorders). This is acceptable for WARC files.

### Output Path

```
$HOME/data/common-crawl/{crawl}/warc_md/{warcIdx}.md.warc.gz
```

One output file per input `.warc.gz` file. `warcIdx` is the 5-digit zero-padded file index.

## Files

- `pkg/warc_md/pack.go` — `RunPack()` + `PackConfig` + pipeline logic
- `cli/cc_warc_pack.go` — `search cc warc pack` command (flags, progress, summary)
- `cli/cc_warc.go` — register `pack` subcommand

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--crawl` | latest | Crawl ID |
| `--file` | `0` | File index, range, or `all` |
| `--from/--to` | -1 | Parallel range |
| `--workers` | NumCPU | Converter goroutines |
| `--force` | false | Re-process existing output |
| `--fast` | false | go-readability instead of trafilatura |
| `--status` | 200 | HTTP status filter |
| `--mime` | text/html | MIME type filter |
| `--max-body` | 512KB | Max HTML body bytes |

## Key Lessons from Prior Work

- **No temp files**: unlike the `markdown` command which writes individual .warc/.md files, pack streams directly to the output `.md.warc.gz`. This avoids millions of small file I/O operations.
- **Channel-based pipeline**: reader → converter pool → writer. Buffered channels (cap=workers×2) prevent backpressure stalls.
- **klauspost/compress gzip**: already a dependency, use for both reading and writing.
