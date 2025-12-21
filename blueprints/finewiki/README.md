```md
# FineWiki

FineWiki is a fast, read-only wiki viewer built on top of the FineWiki dataset. It serves wiki pages directly from Parquet files using DuckDB, with a clean server-side rendered web interface.

The project focuses on one thing only: extremely fast title-based search and page viewing, without importing data into a traditional database or running heavy background pipelines.

This repository is a blueprint-quality implementation that prioritizes clarity, performance, and minimal moving parts.

## What FineWiki Is

FineWiki is a wiki viewer, not an editor.  
It is a single-binary Go application.  
It is DuckDB-backed and reads Parquet directly.  
It provides title-first search for instant results.  
It renders HTML on the server.  
It requires no frontend framework and no client-side JavaScript for core functionality.  
It is designed to handle very large datasets efficiently.

## What FineWiki Is Not

FineWiki is not a Wikipedia replacement.  
It is not a collaborative editing platform.  
It is not a general-purpose full-text search engine.  
It is not an ETL-heavy data warehouse.  
It is not an API-first service.

Those are all valid systems, but they are intentionally out of scope here.

## Dataset

FineWiki is designed to work with the FineWiki dataset published on Hugging Face.

Dataset source  
https://huggingface.co/datasets/HuggingFaceFW/finewiki

The dataset is distributed as Parquet shards. FineWiki can operate on a single shard, multiple local Parquet files, or Parquet files downloaded from Hugging Face. For MVP and local usage, a single shard is sufficient.

## Architecture Overview

At a high level:

Parquet files remain the source of truth.  
DuckDB reads Parquet directly.  
A small derived titles table is created locally for fast search.  
Search queries never touch page text.  
Page rendering is fully server-side.

There are no background workers, no message queues, and no external services.

## Directory Structure

```

finewiki/
├── cmd/
│   └── finewiki/          # main entrypoint
├── app/
│   └── web/               # HTTP server and handlers (Mizu-based)
├── feature/
│   ├── search/            # title-only search logic
│   └── view/              # page view logic
├── store/
│   └── duckdb/            # DuckDB schema, seed, import, store
├── views/                 # HTML templates (SSR)
├── go.mod
└── README.md

````

Each layer has a single responsibility and no circular dependencies.

## Search Design

Search is deliberately constrained for speed.

Current behavior:

Title-only search  
Exact match first  
Prefix match second  
Optional FTS fallback for fuzzy word matching  
Hard limits to protect latency  
No scanning of page text during search

This keeps search predictable and extremely fast even on large datasets.

## UI Design

The UI is inspired by modern encyclopedic viewers such as Grokipedia.

It includes a sticky top bar with global search, keyboard shortcut support for search focus, a left sidebar with table of contents, a clean reading layout, and light and dark themes.

All pages are server-side rendered. No frontend framework is used.

## Requirements

Go 1.22 or newer  
DuckDB via go-duckdb  
Access to Parquet files, local or downloaded

Optional:

A Hugging Face token provided via the HF_TOKEN environment variable for private or rate-limited datasets.

## Getting Started

First, download a Parquet shard. You can do this manually or use the provided import helper.

Set a Hugging Face token if required:

```bash
export HF_TOKEN=your_token_if_needed
````

Then run FineWiki:

```bash
go run ./cmd/finewiki -parquet data/*.parquet
```

On the first run, DuckDB creates local tables, extracts and caches titles, and builds indexes and optional FTS structures. Subsequent runs start quickly.

## Configuration

FineWiki is configured via flags and environment variables.

Common options include:

FINEWIKI_PARQUET
Path or glob to Parquet files

FINEWIKI_DUCKDB
Path to the DuckDB database file

HF_TOKEN
Hugging Face access token, optional

Configuration is intentionally minimal.

## Performance Characteristics

The design goals are:

Cold start measured in seconds
Search latency dominated by index lookups, not I/O
No Parquet access on the hot search path
Stable performance as the dataset grows

For very large datasets, the architecture allows moving title search fully in memory without changing feature APIs.

## Extending FineWiki

The current codebase is intentionally conservative, but future extensions are straightforward.

Possible additions include random page browsing, language filters, HTML rendering instead of plain text, prebuilt table of contents, multi-dataset support, or a JSON API alongside the web UI.

All of these can be added without restructuring the core.

## Why DuckDB

DuckDB is used because it reads Parquet natively, requires no external service, handles analytical datasets efficiently, supports indexes and FTS when needed, and fits naturally into a single-binary Go deployment.

It acts as a fast embedded query engine rather than a traditional database server.

## Acknowledgements

FineWiki dataset contributors
DuckDB project
Hugging Face datasets infrastructure
Grokipedia for UI inspiration

```
```
