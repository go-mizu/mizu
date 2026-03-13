# 0718 SoundCloud Scraper

## Goal

Implement `pkg/scrape/soundcloud` and `cli/soundcloud.go` following the project scraper pattern used by Goodreads, Amazon, and Pinterest:

- per-entity tasks implementing `pkg/core.Task[State, Metric]`
- `soundcloud.duckdb` for durable entities
- `state.duckdb` for crawl queue / jobs / visited URLs
- CLI subcommands under `search soundcloud`

The implementation should prefer SoundCloud's public HTML and `window.__sc_hydration` payloads over deep API coupling. Public API calls are used narrowly for search, because maintained extractors such as `yt-dlp` and `node-soundcloud-downloader` also treat `client_id` acquisition as a volatile concern and keep fallback logic around it.

## Research Notes

### SoundCloud public pages

Observed on 2026-03-13:

- Track pages expose OpenGraph/Twitter metadata, noscript schema markup, and a `window.__sc_hydration` entry with `hydratable: "sound"`.
- Playlist pages expose a `hydratable: "playlist"` entry plus noscript tracklist markup.
- User pages expose a `hydratable: "user"` entry.
- The main page exposes `hydratable: "apiClient"` with a current public `client_id`.

This makes an HTML-first parser materially more stable than relying on undocumented `api-v2` pagination for all entities. Some `api-v2` endpoints intermittently trigger DataDome challenges, while the public entity pages remain fetchable.

### Reference repos

- `yt-dlp/yt_dlp/extractor/soundcloud.py`
  - extracts and refreshes `client_id`
  - treats SoundCloud auth/API access as unstable and retry-heavy
- `zackradisic/node-soundcloud-downloader`
  - also relies on a site-derived `client_id`
  - focuses on `resolve/search/info` more than full-site crawling

Design implication: use `client_id` only where it adds real value. For this implementation, that means search. Track/user/playlist ingestion should be page-driven.

## Scope

### Supported entity types

- `track`
- `user`
- `playlist`
- `search`

### CLI

```bash
search soundcloud track    <url|user/track>
search soundcloud user     <handle|url>
search soundcloud playlist <url|user/sets/name>
search soundcloud search   <query> [--type all|tracks|playlists|users] [--limit 25]
search soundcloud crawl    [--workers 2]
search soundcloud info
search soundcloud jobs
search soundcloud queue    [--status pending|in_progress|done|failed] [--limit 20]
```

## Package Layout

```text
pkg/scrape/soundcloud/
├── client.go
├── config.go
├── db.go
├── display.go
├── parse.go
├── state.go
├── task_crawl.go
├── task_playlist.go
├── task_search.go
├── task_track.go
├── task_user.go
└── types.go
```

## Storage

### `soundcloud.duckdb`

- `users`
- `tracks`
- `playlists`
- `playlist_tracks`
- `comments`
- `search_results`

### `state.duckdb`

Same queue / jobs / visited pattern as Goodreads:

- `queue`
- `jobs`
- `visited`

## Crawl behavior

- `track` task stores the track, nested user, and noscript comments.
- `playlist` task stores the playlist, owner, and playlist-track relationships, then enqueues discovered tracks.
- `user` task stores the profile and enqueues any discoverable SoundCloud entity links present on-page.
- `search` task uses public search with site-derived `client_id`, stores search result rows, and enqueues results.
- `crawl` drains the state queue and dispatches by inferred entity type.

## Parsing strategy

### Primary path

- extract `window.__sc_hydration`
- decode `sound`, `playlist`, `user`, and `apiClient` payloads

### Fallback path

- OpenGraph/Twitter meta tags
- noscript schema markup
- track comments from `section.comments`
- playlist track URLs from `section.tracklist`

## Tradeoffs

- No attempt is made to bypass DataDome or authenticated-only endpoints.
- The initial version does not paginate user-owned tracks/playlists through protected `api-v2` endpoints.
- Instead, crawl expansion comes from:
  - explicit search seeding
  - playlist track discovery
  - in-page SoundCloud links

This is intentionally conservative and aligned with what is reliably accessible without fragile anti-bot workarounds.
