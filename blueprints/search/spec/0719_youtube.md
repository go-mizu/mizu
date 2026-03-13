# 0719: YouTube Scraper

## Goal

Implement `pkg/scrape/youtube` and `cli/youtube.go` to crawl public YouTube data into local DuckDB databases using the same task-oriented pattern already used by `pkg/scrape/goodread` and `pkg/scrape/pinterest`.

The implementation should:

- use `pkg/core.Task[State, Metric]` for every top-level unit of work
- store durable data in `youtube.duckdb`
- store crawl queue / jobs / visited state in `state.duckdb`
- work without an API key by parsing public YouTube HTML and embedded JSON
- support single-entity fetches plus queued crawling

## External references

These repositories informed the design, especially around how much public data is realistically available and where YouTube tends to break:

- `yt-dlp`: robust extractor architecture built around YouTube page JSON and player responses  
  [https://github.com/yt-dlp/yt-dlp](https://github.com/yt-dlp/yt-dlp)
- `NewPipeExtractor`: browserless extraction of channels, playlists, search, and continuation-heavy pages  
  [https://github.com/TeamNewPipe/NewPipeExtractor](https://github.com/TeamNewPipe/NewPipeExtractor)
- `youtube-transcript-api`: transcript handling based on caption tracks exposed by the watch page  
  [https://github.com/jdepoix/youtube-transcript-api](https://github.com/jdepoix/youtube-transcript-api)

Live page verification on March 13, 2026 confirmed that public HTML still exposes:

- `ytInitialData`
- `ytInitialPlayerResponse`
- `ytcfg.set(...)`
- channel/video/playlist metadata in embedded JSON
- caption track URLs on watch pages when captions exist

## Scope

Initial implementation covers:

- videos
- channels
- playlists
- search result pages
- caption tracks + transcript text when available
- related videos discovered from watch pages
- uploads / playlist / related discovery into queue
- bulk crawl dispatcher

Out of scope for this iteration:

- authenticated/private data
- likes/subscriptions that require sign-in
- full comment tree crawling
- Shorts-specific mobile-only APIs
- live chat replay

## Package layout

```text
pkg/scrape/youtube/
├── client.go
├── config.go
├── db.go
├── display.go
├── parse.go
├── state.go
├── task_channel.go
├── task_crawl.go
├── task_playlist.go
├── task_search.go
├── task_video.go
└── types.go

cli/youtube.go
```

## Data model

### `youtube.duckdb`

Tables:

- `videos`
- `channels`
- `playlists`
- `playlist_videos`
- `related_videos`
- `caption_tracks`

Key fields:

- `videos`: ids, title, description, channel metadata, duration, view/comment/like counts, publish text/date, category, tags, transcript
- `channels`: channel id, handle, title, description, avatar/banner URLs, subscriber/video/view text, uploads playlist id
- `playlists`: playlist id, title, description, channel metadata, video count, view text
- `playlist_videos`: playlist-video edge with ordinal
- `related_videos`: source video -> related video edge with ordinal
- `caption_tracks`: per-language caption metadata

Arrays are stored as JSON text, matching existing scraper conventions.

### `state.duckdb`

Same queue/jobs/visited pattern as `pkg/scrape/goodread/state.go`.

Entity types:

- `video`
- `channel`
- `playlist`
- `search`

## Tasks

Every task implements `pkg/core.Task`.

### `VideoTask`

- fetch watch page
- parse `ytInitialPlayerResponse` and `ytInitialData`
- upsert video
- upsert caption tracks
- fetch transcript text from first caption track when available
- enqueue:
  - channel
  - playlist from watch URL if present
  - related videos

### `ChannelTask`

- fetch channel videos page
- parse channel metadata
- parse visible video list
- upsert channel
- upsert discovered videos
- enqueue discovered videos
- enqueue uploads playlist when known

### `PlaylistTask`

- fetch playlist page
- parse playlist metadata
- parse visible playlist items
- upsert playlist + `playlist_videos`
- upsert discovered videos
- enqueue discovered videos

### `SearchTask`

- fetch public search page
- parse video/channel/playlist results
- upsert returned entities
- optionally enqueue returned entities

### `CrawlTask`

- pop queue items
- dispatch by entity type
- emit crawl progress using the same model as `goodread`

## CLI

Register a new top-level command:

```bash
search youtube video <id|url>
search youtube channel <id|url|@handle>
search youtube playlist <id|url>
search youtube search <query> [--max-results 30] [--enqueue]
search youtube seed --file items.txt --entity video|channel|playlist|search
search youtube crawl [--workers 2]
search youtube info
search youtube jobs
search youtube queue [--status pending|failed|done] [--limit 20]
```

Defaults:

- `--db`: `$HOME/data/youtube/youtube.duckdb`
- `--state`: `$HOME/data/youtube/state.duckdb`

## Design notes

- Prefer public page extraction first; no external API credentials.
- Reuse `goodread` ergonomics: small task structs, small client, small CLI helpers.
- Keep parsers defensive: YouTube JSON changes frequently.
- Treat unavailable transcript / counts as non-fatal.
- Mark 404 / unavailable pages as done in state DB so the queue does not loop forever.

## Verification

Minimum verification for this change:

- `go test ./pkg/scrape/youtube/...` if tests exist
- `go test ./cli/...` or at least compile the root package
- `go test ./...` only if the workspace is stable enough
- manual smoke checks:
  - `search youtube video dQw4w9WgXcQ`
  - `search youtube channel @GoogleDevelopers`
  - `search youtube playlist PL590L5WQmH8fJ54F1L7aRQlQ-Qc8-ND8B`
  - `search youtube search golang --max-results 20 --enqueue`

