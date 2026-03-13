# Spec 0722: Kaggle Scraper

## Goal

Implement `pkg/scrape/kaggle` and `cli/kaggle.go` to crawl public Kaggle content into local DuckDB databases, following the same task-oriented pattern already used by `pkg/scrape/goodread`, `pkg/scrape/ebay`, `pkg/scrape/youtube`, and `pkg/scrape/soundcloud`.

The implementation must:

- use `pkg/core.Task[State, Metric]` for all long-running units of work
- persist entity data into a main DuckDB
- persist queue, jobs, and visited state into a separate DuckDB
- expose CLI subcommands for one-off fetches, discovery, crawl execution, and inspection
- prefer Kaggle’s public JSON endpoints where they are available
- fall back to public HTML metadata parsing where Kaggle does not expose equivalent unauthenticated JSON

## Public Surfaces

Verified public unauthenticated JSON:

- `GET https://www.kaggle.com/api/v1/datasets/list`
- `GET https://www.kaggle.com/api/v1/datasets/view/{owner}/{slug}`
- `GET https://www.kaggle.com/api/v1/models/list`

Verified public HTML detail pages:

- `https://www.kaggle.com/datasets/{owner}/{slug}`
- `https://www.kaggle.com/models/{owner}/{slug}`
- `https://www.kaggle.com/competitions/{slug}`
- `https://www.kaggle.com/{handle}`
- `https://www.kaggle.com/code/{owner}/{slug}`

Observed constraints:

- datasets list pagination uses `?page=N`
- models list pagination uses `nextPageToken`
- competitions list and kernels list return `401 Unauthenticated` on `/api/v1/*` for anonymous access
- profile pages are client-rendered; reliable anonymous extraction is limited to `<title>`, meta description, OpenGraph/Twitter metadata, and URL-derived identity

## GitHub References

Implementation direction is informed by:

- `Kaggle/kaggle-api` for API-first interaction with Kaggle public resources
- established repo-local scraper patterns in `pkg/scrape/goodread`, `pkg/scrape/ebay`, and `pkg/scrape/youtube`

The practical takeaway is:

- use public JSON where Kaggle exposes stable list/detail payloads
- do not depend on authenticated/private Kaggle APIs
- keep HTML parsing shallow and metadata-oriented for the surfaces that are mostly client-rendered

## Scope

### Fully supported

- datasets
- models
- bulk discovery for datasets and models

### Supported from direct URLs / seeds

- competitions
- notebooks
- profiles

These HTML-only entities are fetched and stored from their public pages, but they do not currently have reliable anonymous bulk-discovery endpoints comparable to datasets/models.

## Package Layout

```text
pkg/scrape/kaggle/
├── client.go
├── config.go
├── db.go
├── display.go
├── normalize.go
├── state.go
├── task_competition.go
├── task_crawl.go
├── task_dataset.go
├── task_discover.go
├── task_model.go
├── task_notebook.go
├── task_profile.go
└── types.go
```

## Entities

### Dataset

Stored from `/api/v1/datasets/view/{owner}/{slug}` with list/detail fields:

- `id`, `ref`, `owner_ref`, `owner_name`
- `creator_name`, `creator_url`
- `title`, `subtitle`, `description`
- `url`, `license_name`, `thumbnail_image_url`
- `download_count`, `view_count`, `vote_count`
- `kernel_count`, `topic_count`
- `current_version_number`
- `usability_rating`
- `total_bytes`
- `is_private`, `is_featured`
- `last_updated`
- `tags_json`, `versions_json`, `raw_json`

Child table:

- `dataset_files`

### Model

Stored from `/api/v1/models/list` filtered to the requested ref:

- `id`, `ref`, `owner_ref`
- `title`, `subtitle`, `description`
- `author`, `author_image_url`
- `url`
- `vote_count`
- `update_time`
- `is_private`
- `tags_json`, `raw_json`

Child table:

- `model_instances`

### Competition

Stored from public HTML metadata:

- `slug`
- `title`
- `description`
- `url`
- `image_url`
- `raw_meta_json`

### Notebook

Stored from public HTML metadata:

- `ref`
- `owner_ref`
- `slug`
- `title`
- `description`
- `url`
- `image_url`
- `raw_meta_json`

### Profile

Stored from public HTML metadata:

- `handle`
- `display_name`
- `bio`
- `url`
- `image_url`
- `raw_meta_json`

## Relational Schema

Main DB tables:

- `datasets`
- `dataset_files`
- `models`
- `model_instances`
- `competitions`
- `notebooks`
- `profiles`

State DB tables:

- `queue`
- `jobs`
- `visited`

Queue entity types:

- `dataset`
- `model`
- `competition`
- `notebook`
- `profile`

Queue uniqueness key:

- canonical entity URL

## Task Model

Every top-level unit of work implements `pkg/core.Task`.

### `DatasetTask`

- normalize `<owner/slug|url>` to canonical dataset URL
- fetch `/api/v1/datasets/view/{owner}/{slug}`
- upsert the dataset row and `dataset_files`
- enqueue the owner profile URL
- mark queue row done/failed in the state DB

### `ModelTask`

- normalize `<owner/slug|url>` to canonical model URL
- locate the exact model via `/api/v1/models/list?search=...`
- upsert the model row and `model_instances`
- enqueue the owner profile URL
- mark queue row done/failed

### `CompetitionTask`

- normalize `<slug|url>` to canonical competition URL
- fetch public HTML
- parse title/description/OpenGraph image
- upsert the competition row

### `NotebookTask`

- normalize `<owner/slug|url>` to canonical notebook URL
- fetch public HTML
- parse title/description/OpenGraph image
- upsert the notebook row
- enqueue owner profile URL

### `ProfileTask`

- normalize `<handle|url>` to canonical profile URL
- fetch public HTML
- parse title/description/OpenGraph image
- upsert the profile row

### `DiscoverTask`

- iterate datasets pages using `page=N`
- iterate models pages using `nextPageToken`
- optionally filter entity kinds with `--types`
- enqueue canonical URLs only
- optionally enqueue discovered owner profiles
- emit progress snapshots with per-type counts

### `CrawlTask`

- pop queued work
- dispatch to the correct entity task
- emit periodic progress (`done`, `pending`, `failed`, `in_flight`, `rps`)

## Discovery Strategy

`discover` is the entry point for “crawl everything public”.

Default behavior:

- include `dataset,model`
- request datasets with `page=1..N`
- request models using `nextPageToken` until exhausted or `--max-pages` reached
- enqueue canonical URLs for each discovered entity
- also enqueue owner profiles at lower priority

This is intentionally limited to the surfaces Kaggle exposes anonymously in a stable way.

## CLI

Add `search kaggle` with subcommands:

- `dataset <owner/slug|url>`
- `model <owner/slug|url>`
- `competition <slug|url>`
- `profile <handle|url>`
- `notebook <owner/slug|url>`
- `discover`
- `seed --file <path> --type <entity>`
- `crawl`
- `info`
- `jobs`
- `queue`

Shared flags:

- `--db`
- `--state`
- `--delay`
- `--workers`
- `--max-pages`
- `--types`

## Implementation Notes

- Keep the client split between:
  - JSON helpers for datasets/models
  - HTML metadata helpers for competitions/notebooks/profiles
- Persist JSON-heavy payloads as JSON strings to stay resilient to Kaggle schema drift.
- Keep canonical URL normalization strict so queue deduplication works.
- Do not attempt authenticated endpoints or browser automation in this initial implementation.
