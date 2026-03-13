# spec/0727 — `search amazon`: Full discovery + crawl plan

## Goal

Deliver a complete `search amazon` workflow that:

1. Discovers all crawl targets for a query (paged Amazon search URLs).
2. Crawls those targets concurrently with controlled request rate.
3. Extracts full per-result product-card data.
4. Persists results for resume, inspection, and downstream analysis.

## Command surface

- `search amazon <query>`
  - Full run: discover + crawl + persist + summary output.
- `search amazon discover <query>`
  - Discovery-only: prints all result page URLs that will be crawled.
- `search amazon info`
  - Prints DB-level crawl statistics.

## Discovery strategy

For a query `q`, generate URLs:

- `https://{market}/s?k={q}` for page 1
- `https://{market}/s?k={q}&page={n}` for page `n >= 2`

Inputs controlling discovery:

- `--pages` (max pages budget)
- `--resume` (start from next uncrawled page from DB)
- `--market` (domain: `.com`, `.co.uk`, etc.)
- `--sort` (optional Amazon sort key)

Stop conditions:

1. Hit `--pages` cap.
2. Crawled page has no enabled `a.s-pagination-next`.

## Crawl strategy

- Worker pool (`--workers`) consumes discovered page numbers.
- Each worker fetches one result page with Amazon UA + language headers.
- Optional request throttling (`--rate`) to reduce bot-pressure.
- Validate HTML (empty body / captcha guard).

## Extracted data per product card

From `div[data-component-type='s-search-result']`:

- ASIN
- Title
- Product URL
- Image URL
- Price text + numeric value + detected currency
- Rating
- Review count
- Prime marker
- Sponsored marker
- Badge text
- Position on page
- Result page number
- Scrape timestamp
- Raw card text snapshot

## Storage strategy

DuckDB table: `amazon_products`

Primary key:

- `(query, asin, result_page)`

Supports:

- idempotent upsert reruns (`INSERT OR REPLACE`)
- resumable crawling (`MAX(result_page)` per query)
- quick stats (`COUNT`, `COUNT DISTINCT`)

## Validation / checks

- Parser unit tests for:
  - product extraction correctness
  - pagination next detection
  - numeric field parsing
- CLI smoke checks:
  - command help
  - discover output shape

## Non-goals (this spec)

- Logged-in-only/private Amazon endpoints.
- Full product-detail page deep crawl.
- Anti-bot bypass escalation beyond conservative rate limiting.
