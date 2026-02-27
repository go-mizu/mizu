# RSS & Small Web Integration Plan

## 1. Objective

This plan outlines the implementation of a new `pkg/rss` for handling RSS/Atom/OPML feeds and enhancing the `search` CLI with an `rss` subcommand. This enables seeding, listing, and crawling of RSS feeds, with an initial seed list from Kagi's Small Web.

## 2. Core Components

### 2.1. `pkg/rss`: Feed Parsing

-   **Purpose:** Create a new package `pkg/rss` to abstract feed parsing logic.
-   **Dependencies:**
    -   `github.com/mmcdole/gofeed` for RSS and Atom feeds.
    -   `github.com/gilliek/go-opml` for OPML feed lists.
-   **Implementation:**
    -   `rss.go`: Define structs for `Feed`, `Item`, and `OPML`.
    -   `parser.go`: Implement `ParseURL(url string)` which fetches and parses a feed (RSS/Atom) or OPML file, automatically detecting the type. It returns a generic `Feed` or `OPML` struct.

### 2.2. `cli/rss.go`: CLI Command

-   **Purpose:** Create a new `rss` subcommand under `search`.
-   **File:** `cli/rss.go`
-   **Command Structure:** `search rss <subcommand>`
-   **Subcommands:**
    -   `seed`: Fetches feeds from Kagi Small Web (`https://kagi.com/api/v1/smallweb/feed/` and `https://kagi.com/smallweb/opml`) and saves them to the database.
    -   `list`: Lists all RSS feeds currently in the database.
    -   `items --feed-id <id>`: Lists items for a specific feed.
    -   `crawl [--feed-id <id>]`: Crawls a single feed by its ID, or all feeds if no ID is provided. This fetches the feed content and stores new items in the database.
    -   `recrawl [--feed-id <id>]`: Fetches the actual content of each post in the feed and indexes them in the main search index.

### 2.3. `store`: Database Interaction

-   **Purpose:** Extend the existing data store to manage RSS feeds and items.
-   **Schema (New Tables):**
    -   `rss_feeds`:
        -   `id` (PK)
        -   `url` (feed URL, unique)
        -   `title`
        -   `site_url`
        -   `description`
        -   `last_crawled_at`
    -   `rss_items`:
        -   `id` (PK)
        -   `feed_id` (FK to `rss_feeds`)
        -   `url` (item URL, unique)
        -   `title`
        -   `content`
        -   `published_at`
-   **Files:**
    -   `store/sqlite/rss.go`: Implementation of RSS-related database queries for SQLite.
    -   `store/sqlite/schema.go`: Updated schema with `rss_feeds` and `rss_items` tables.
    -   `store/store.go`: Update the `Store` interface with `RSS()` method.

## 3. Key Enhancements

### 3.1. `CrawlURLs` in `pkg/crawler`

The `Crawler` was enhanced with a `CrawlURLs(ctx context.Context, urls []string)` method to allow concurrent crawling of multiple start URLs. This is used by `search rss recrawl` to index all posts in a feed efficiently.

## 4. Testing Performed

1.  **Seeding:** Verified `search rss seed` adds thousands of feeds from Kagi.
2.  **Listing:** Verified `search rss list` displays feeds correctly.
3.  **Crawling:** Verified `search rss crawl --feed-id <id>` fetches and stores feed items.
4.  **Items Listing:** Verified `search rss items --feed-id <id>` shows stored items.
5.  **Indexing:** Verified `search rss recrawl --feed-id <id>` fetches and indexes post content into the main `documents` table.
