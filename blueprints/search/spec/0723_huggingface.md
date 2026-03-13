# Spec 0723: Hugging Face Hub Scraper

## Goal

Implement `pkg/scrape/huggingface` and `cli/huggingface.go` to crawl public Hugging Face Hub data into local DuckDB databases, following the same project pattern already used by `pkg/scrape/goodread`, `pkg/scrape/spotify`, `pkg/scrape/amazon`, and `pkg/scrape/pinterest`.

The Hugging Face implementation should:

- use `pkg/core.Task[State, Metric]` for all long-running work
- discover entities through the public Hub API instead of HTML scraping
- persist detail records plus graph edges into a main DuckDB
- persist crawl queue, job state, and visited entities into a separate state DuckDB
- expose CLI subcommands for one-off fetches, bulk discovery, crawl execution, and inspection

## External Surfaces

Primary crawl surfaces:

- `GET /api/models`
- `GET /api/datasets`
- `GET /api/spaces`
- `GET /api/collections`
- `GET /api/papers`

Primary detail surfaces:

- `GET /api/models/{repo_id}`
- `GET /api/datasets/{repo_id}`
- `GET /api/spaces/{repo_id}`
- `GET /api/collections/{namespace}/{slug}`
- `GET /api/papers/{paper_id}`

Discovery pagination must follow the Hub API `Link: ... rel="next"` header and preserve the returned cursor query string. This matches the pagination strategy used by the official `huggingface_hub` client.

## Package Layout

```text
pkg/scrape/huggingface/
├── config.go
├── client.go
├── db.go
├── display.go
├── normalize.go
├── state.go
├── types.go
├── task_collection.go
├── task_crawl.go
├── task_dataset.go
├── task_discover.go
├── task_model.go
├── task_paper.go
└── task_space.go
```

## Entities

### Model

Stored from `/api/models/{repo_id}` with the important list/detail fields:

- `repo_id`, `author`, `sha`
- `created_at`, `last_modified`
- `private`, `gated`, `disabled`
- `likes`, `downloads`, `trending_score`
- `pipeline_tag`, `library_name`
- `tags`, `siblings`
- `card_data_json`, `config_json`, `transformers_info_json`, `widget_data_json`
- `spaces_json`
- `raw_json`

### Dataset

Stored from `/api/datasets/{repo_id}`:

- `repo_id`, `author`, `sha`
- `created_at`, `last_modified`
- `private`, `gated`, `disabled`
- `likes`, `downloads`, `trending_score`
- `description`, `tags`, `siblings`
- `card_data_json`
- `raw_json`

### Space

Stored from `/api/spaces/{repo_id}`:

- `repo_id`, `author`, `sha`
- `created_at`, `last_modified`
- `private`, `disabled`
- `likes`, `sdk`, `subdomain`
- `tags`, `siblings`
- `runtime_json`, `card_data_json`
- `raw_json`

### Collection

Stored from `/api/collections/{namespace}/{slug}`:

- `slug`, `namespace`, `title`
- `owner_json`, `theme`, `upvotes`, `private`
- `description`, `gating`
- `last_updated`
- `items_json`
- `raw_json`

Collection item edges are also normalized into a separate relation table to connect collections to repos and papers.

### Paper

Stored from `/api/papers/{paper_id}`:

- `paper_id`, `title`
- `summary`, `ai_summary`
- `published_at`
- `upvotes`
- `authors_json`
- `github_repo`, `project_page`, `thumbnail_url`
- `raw_json`

## Relational Schema

Main DB tables:

- `models`
- `datasets`
- `spaces`
- `collections`
- `papers`
- `repo_files`
- `repo_links`
- `collection_items`

Notes:

- `repo_files` stores flattened `siblings` entries for models, datasets, and spaces.
- `repo_links` stores discovered cross-entity edges such as model→space, space→model, space→dataset.
- `collection_items` stores normalized collection membership.
- JSON-heavy fields remain JSON strings to keep schema simple and resilient to API changes.

State DB tables:

- `queue`
- `jobs`
- `visited`

Queue key:

- unique on canonical entity URL

Queue entity types:

- `model`
- `dataset`
- `space`
- `collection`
- `paper`

## Task Model

Every top-level unit of work implements `pkg/core.Task`.

### `ModelTask`

- normalize `<repo_id|url>` to canonical `https://huggingface.co/{repo_id}`
- fetch `/api/models/{repo_id}`
- upsert the model row
- upsert `repo_files`
- enqueue discovered `spaces` from the detail payload
- mark queue row done/failed in state DB

### `DatasetTask`

- normalize to canonical dataset URL
- fetch `/api/datasets/{repo_id}`
- upsert dataset row
- upsert `repo_files`

### `SpaceTask`

- normalize to canonical space URL
- fetch `/api/spaces/{repo_id}`
- upsert space row
- upsert `repo_files`
- enqueue linked models and datasets from the detail payload

### `CollectionTask`

- normalize to canonical collection URL
- fetch collection detail
- upsert collection row
- upsert normalized collection items
- enqueue linked repos and papers from collection items

### `PaperTask`

- normalize paper id / URL
- fetch paper detail
- upsert paper row

### `DiscoverTask`

- iterate selected list endpoints using pagination
- enqueue every discovered entity URL with a configurable priority
- optionally fetch the first `N` pages only
- emit progress snapshots with per-entity discovered counts

### `CrawlTask`

- pop work from queue
- dispatch to the correct entity task
- emit periodic progress (`done`, `pending`, `failed`, `in_flight`, `rps`)

## Discovery Strategy

`discover` is the entry point for “crawl everything”.

Default behavior:

- include `models`, `datasets`, `spaces`, `collections`, `papers`
- request list pages with `limit=100`
- follow `rel="next"` cursor links until exhausted or `--max-pages` is reached
- enqueue canonical URLs only

Canonical URLs:

- model: `https://huggingface.co/{repo_id}`
- dataset: `https://huggingface.co/datasets/{repo_id}`
- space: `https://huggingface.co/spaces/{repo_id}`
- collection: `https://huggingface.co/collections/{namespace}/{slug}`
- paper: `https://huggingface.co/papers/{paper_id}`

## CLI

Add `search huggingface` with subcommands:

- `model <repo_id|url>`
- `dataset <repo_id|url>`
- `space <repo_id|url>`
- `collection <slug|url>`
- `paper <id|url>`
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

- Prefer the Hub API over parsing repo HTML.
- Keep the client generic: `FetchJSON`, `FetchJSONWithNext`, and typed helpers per entity.
- Preserve raw payloads to make the scraper forward-compatible with new Hub fields.
- Use canonical URL normalization so manual fetches and discovery share one queue key.
- Use best-effort edge extraction only; missing optional fields must not fail the scrape.

## GitHub Learnings Applied

The implementation should explicitly mirror the stronger patterns seen in popular Hugging Face-maintained repositories:

- official `huggingface_hub` pagination model: follow `Link` headers instead of inventing page arithmetic
- official Hub list/detail split: use lightweight discovery pages and fetch details only when needed
- JSON-forward compatibility: keep raw payloads and selected JSON subtrees instead of over-normalizing volatile fields
