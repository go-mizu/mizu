# 0720 — Facebook Scraper

## Overview

Implement `pkg/scrape/facebook` and `cli/facebook.go` for crawling the public Facebook surface into DuckDB, organized around `pkg/core.Task`.

This is intentionally scoped to what can be fetched with ordinary HTTP plus optional cookies:

- Public pages
- Public profiles
- Public groups
- Public post permalinks
- Comments visible on fetched post pages
- Media links discovered on posts
- Best-effort keyword search via Facebook's public/mobile search pages

It does **not** claim access to private timelines, private groups, friends-only posts, Messenger, ads manager, or internal authenticated APIs.

## Design Constraints

- Follow the established scraper pattern used by `pkg/scrape/goodread`:
  - Shared `Config`
  - Shared `Client`
  - DuckDB content DB
  - Separate DuckDB state DB for queue/jobs/visited
  - Entity-specific tasks implementing `core.Task[State, Metric]`
  - Thin Cobra CLI wrappers
- Use `mbasic.facebook.com` as the primary HTML surface because it is materially simpler and more stable for scraping than the modern JS application.
- Support optional cookies through config/env/file so users can fetch surfaces that require a logged-in but ordinary browser session.
- Favor resilient heuristics over brittle CSS selectors; Facebook markup changes frequently.

## Open-Source Inspiration

This design is informed by widely used public projects that converge on the same practical constraints:

- [`kevinzg/facebook-scraper`](https://github.com/kevinzg/facebook-scraper): public pages/profiles/groups/posts/comments with optional cookies, leaning on mobile/basic HTML.
- [`harismuneer/Ultimate-Facebook-Scraper`](https://github.com/harismuneer/Ultimate-Facebook-Scraper): browser-cookie based extraction for deeper authenticated surfaces.

What we borrow:

- Treat cookies as optional capability, not a hard dependency.
- Model scraping around concrete entities, not a monolithic “scrape everything” function.
- Expect permalink discovery and pagination to be heuristic and degrade gracefully.

What we do differently:

- Persist into DuckDB instead of JSON/CSV exports.
- Use `pkg/core.Task` for every crawl unit.
- Keep CLI + queue orchestration consistent with the rest of this repository.

## Package Layout

```text
pkg/scrape/facebook/
├── client.go
├── config.go
├── db.go
├── parse.go
├── state.go
├── task_crawl.go
├── task_group.go
├── task_page.go
├── task_post.go
├── task_profile.go
├── task_search.go
└── types.go
```

## Data Model

### `facebook.duckdb`

```sql
CREATE TABLE IF NOT EXISTS pages (
  page_id          VARCHAR PRIMARY KEY,
  slug             VARCHAR,
  name             VARCHAR,
  category         VARCHAR,
  about            VARCHAR,
  likes_count      BIGINT,
  followers_count  BIGINT,
  verified         BOOLEAN,
  website          VARCHAR,
  phone            VARCHAR,
  address          VARCHAR,
  url              VARCHAR,
  fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS profiles (
  profile_id       VARCHAR PRIMARY KEY,
  username         VARCHAR,
  name             VARCHAR,
  intro            VARCHAR,
  bio              VARCHAR,
  followers_count  BIGINT,
  friends_count    BIGINT,
  verified         BOOLEAN,
  hometown         VARCHAR,
  current_city     VARCHAR,
  work             VARCHAR,
  education        VARCHAR,
  url              VARCHAR,
  fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS groups (
  group_id         VARCHAR PRIMARY KEY,
  slug             VARCHAR,
  name             VARCHAR,
  description      VARCHAR,
  privacy          VARCHAR,
  members_count    BIGINT,
  url              VARCHAR,
  fetched_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS posts (
  post_id            VARCHAR PRIMARY KEY,
  owner_id           VARCHAR,
  owner_name         VARCHAR,
  owner_type         VARCHAR,
  text               VARCHAR,
  created_at_text    VARCHAR,
  like_count         BIGINT,
  comment_count      BIGINT,
  share_count        BIGINT,
  permalink          VARCHAR,
  media_urls         VARCHAR,
  external_links     VARCHAR,
  fetched_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS comments (
  comment_id         VARCHAR PRIMARY KEY,
  post_id            VARCHAR,
  author_id          VARCHAR,
  author_name        VARCHAR,
  text               VARCHAR,
  created_at_text    VARCHAR,
  like_count         BIGINT,
  permalink          VARCHAR,
  fetched_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS search_results (
  query              VARCHAR NOT NULL,
  result_url         VARCHAR NOT NULL,
  entity_type        VARCHAR,
  title              VARCHAR,
  snippet            VARCHAR,
  fetched_at         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (query, result_url)
);
```

JSON arrays are stored as `VARCHAR` JSON text for `media_urls` and `external_links`.

### `state.duckdb`

```sql
CREATE SEQUENCE IF NOT EXISTS queue_id_seq;

CREATE TABLE IF NOT EXISTS queue (
  id           BIGINT DEFAULT nextval('queue_id_seq') PRIMARY KEY,
  url          VARCHAR UNIQUE NOT NULL,
  entity_type  VARCHAR NOT NULL,
  priority     INTEGER DEFAULT 0,
  status       VARCHAR DEFAULT 'pending',
  attempts     INTEGER DEFAULT 0,
  last_attempt TIMESTAMP,
  error        VARCHAR,
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS jobs (
  job_id       VARCHAR PRIMARY KEY,
  name         VARCHAR,
  type         VARCHAR,
  status       VARCHAR,
  started_at   TIMESTAMP,
  completed_at TIMESTAMP,
  config       VARCHAR,
  stats        VARCHAR
);

CREATE TABLE IF NOT EXISTS visited (
  url          VARCHAR PRIMARY KEY,
  fetched_at   TIMESTAMP,
  status_code  INTEGER,
  entity_type  VARCHAR
);
```

## Entities

- `page`
- `profile`
- `group`
- `post`
- `search`

`crawl` is the queue runner, not a stored entity.

## Client Strategy

- Use plain `net/http` with:
  - rotating desktop/mobile user agents
  - request delay
  - retry on transient failures
  - optional `Cookie` header loaded from:
    - `FACEBOOK_COOKIE`
    - `FACEBOOK_COOKIE_FILE`
    - `Config.Cookies`
    - `Config.CookiesFile`
- Normalize Facebook URLs to `https://mbasic.facebook.com/...` for fetches.
- Preserve canonical public permalinks in stored rows when possible.

## Parsing Strategy

Facebook HTML changes often, so parsing uses layered heuristics:

- Identity:
  - extract page/profile/group identifiers from URL path first
  - fall back to `profile.php?id=...`, canonical links, or hidden tokens in anchors
- Metadata:
  - prefer `<title>`/`<h1>`
  - then fallback to first strong heading-like nodes
- Counts:
  - parse “1.2K followers”, “35K likes”, “12 comments”, “3 shares”, “18K members”
- Posts:
  - detect post permalinks from `story.php`, `/posts/`, `/permalink/`, `/groups/.../posts/`, `/watch/?v=`, `/photo.php`
  - extract nearby container text for the post body
  - collect external/media links from the same container
- Comments:
  - scan nearby anchors and blocks for comment author, text, likes, and permalink
- Pagination:
  - follow “See more”, “More stories”, “More results”, and `start=`/`cursor=`/`bacr=` style links when present

## Task Design

### `PageTask`

- Fetch a page URL
- Parse page metadata
- Parse visible feed posts from the page
- Upsert page + posts + comments
- Enqueue discovered post/profile/page/group URLs

### `ProfileTask`

- Fetch a public profile URL
- Parse profile metadata
- Parse visible feed posts
- Upsert profile + posts + comments
- Enqueue discovered permalinks

### `GroupTask`

- Fetch a public group URL
- Parse group metadata
- Parse visible feed posts
- Upsert group + posts + comments
- Enqueue discovered post URLs

### `PostTask`

- Fetch a single post permalink
- Parse the main post and visible comments
- Upsert rows
- Enqueue linked owner pages/profiles/groups

### `SearchTask`

- Build a best-effort search URL for one of:
  - `top`
  - `posts`
  - `pages`
  - `people`
  - `groups`
- Parse search results and store them in `search_results`
- Optionally enqueue matching URLs

### `CrawlTask`

- Pop queue items in batches
- Dispatch by `entity_type`
- Emit periodic progress
- Mark queue items done/failed in `state.duckdb`

## CLI

Add `search facebook` with subcommands:

- `post <url>`
- `page <url|slug>`
- `profile <url|username|id>`
- `group <url|slug|id>`
- `search <query>`
- `seed --file <path> --entity <type>`
- `crawl`
- `info`
- `jobs`
- `queue`

Common flags:

- `--db`
- `--state`
- `--delay`
- `--cookie`
- `--cookie-file`
- `--max-pages`
- `--max-comments`

## Known Limits

- Some public-looking pages still require a logged-in session.
- Search quality varies heavily by geography, cookies, and Facebook experiments.
- Reaction breakdowns are intentionally reduced to aggregate counts.
- Dynamic comments beyond what the HTML exposes are out of scope for the first version.
- We do not call private GraphQL endpoints or reverse-engineer authenticated AJAX contracts in this implementation.

## Implementation Notes

- Keep the first version robust and broad rather than “perfect” on a single selector set.
- Store raw-enough counts/text/permalinks for later enrichment.
- Favor queue growth from discovered permalinks and owners over trying to enumerate every internal tab immediately.
