# 0717 — Spotify Scraper

## Overview

Scrape public Spotify entity pages into a local DuckDB database and crawl the public entity graph from a seed set. The implementation follows the dedicated scraper pattern already used by `pkg/scrape/goodread`, `pkg/scrape/amazon`, and `pkg/scrape/pinterest`: entity-specific task structs implementing `pkg/core.Task[State, Metric]`, a shared HTTP client, a separate state DB for the crawl queue, and CLI subcommands under `cli/spotify.go`.

Primary sources used for the design:

- Spotify public entity pages on `open.spotify.com` expose stable SEO metadata (`<title>`, Open Graph tags, JSON-LD, canonical URL, oEmbed link).
- Spotify public entity pages also expose a richer base64-encoded `initialState` bootstrap payload in `<script id="initialState" type="text/plain">...`.
- Spotify’s official oEmbed endpoint returns anonymous embed metadata for public entity pages.
- Popular Spotify ecosystem repositories consistently normalize Spotify URIs/URLs aggressively and traverse entity graphs from seed entities rather than depending on a single brittle endpoint.

This implementation intentionally avoids authenticated Web API dependencies. Public entity-page scraping is the baseline because it works anonymously and exposes richer bootstrap data than the current SSR search page.

## Scope

Implemented entity types:

- `track`
- `album`
- `artist`
- `playlist`

Implemented crawl capabilities:

- Fetch a single entity page by ID, URI, or URL
- Persist normalized entity records plus relations into `spotify.duckdb`
- Enqueue discovered related entities into `state.duckdb`
- Bulk-crawl the queue with worker concurrency using `pkg/core.Task`
- Inspect queue/jobs/stats from CLI
- Seed the queue from a file

Explicitly out of scope for this iteration:

- `search` subcommand: current `open.spotify.com/search/...` page is not SSR-backed with query results, so a robust implementation would require private/internal API coupling
- authenticated/private user libraries
- podcast/show/episode/audiobook coverage
- playlist pagination beyond what is surfaced in the page bootstrap payload

## Package Layout

```text
blueprints/search/
├── pkg/scrape/spotify/
│   ├── types.go          # Track, Album, Artist, Playlist, edge rows, queue/job types
│   ├── config.go         # Config + DefaultConfig()
│   ├── client.go         # HTTP client + HTML/bootstrap parsing helpers
│   ├── db.go             # spotify.duckdb schema + upsert methods
│   ├── state.go          # state.duckdb queue/jobs/visited
│   ├── display.go        # stats + crawl progress printing
│   ├── task_track.go     # TrackTask
│   ├── task_album.go     # AlbumTask
│   ├── task_artist.go    # ArtistTask
│   ├── task_playlist.go  # PlaylistTask
│   └── task_crawl.go     # CrawlTask dispatcher
└── cli/spotify.go        # CLI subcommands
```

## Source Strategy

### Public HTML metadata

Each public entity page contains:

- canonical URL
- Open Graph title/description/image
- type-specific Open Graph tags:
  - track: `music:album`, `music:musician`, `music:duration`, `music:release_date`, `og:audio`
  - artist: profile metadata and monthly listeners in `og:description`
  - album/playlist: cover art and descriptive text
- JSON-LD for track and artist pages
- official oEmbed link

These fields are enough for a resilient fallback record even if bootstrap parsing changes.

### `initialState` bootstrap payload

The entity pages also include:

```html
<script id="initialState" type="text/plain">BASE64_JSON</script>
```

Decoding yields a JSON object with:

- `entities.items["spotify:track:<id>"]`
- `entities.items["spotify:album:<id>"]`
- `entities.items["spotify:artist:<id>"]`
- `entities.items["spotify:playlist:<id>"]`

Useful fields found in practice:

- track:
  - `name`
  - `duration.totalMilliseconds`
  - `trackNumber`
  - `albumOfTrack.{name,uri,coverArt,date,type}`
  - `firstArtist.items[]`
  - `otherArtists.items[]`
  - `previews.audioPreviews.items[].url`
  - `playability.playable`
- album:
  - `name`, `type`, `date.year|month|day`, `copyright`
  - `coverArt.sources[]`
  - `artists.items[]`
  - `tracksV2.items[].track`
- artist:
  - `profile.name`
  - `profile.biography.text`
  - `profile.externalLinks.items[]`
  - `stats.followers`, `stats.monthlyListeners`
  - `visuals.avatarImage.sources[]`
  - `discography.latest`
  - `discography.popularReleasesAlbums.items[]`
  - `discography.albums.items[].releases.items[]`
  - `discography.singles.items[].releases.items[]`
  - `discography.topTracks.items[].track`
  - `relatedContent.relatedArtists.items[]`
- playlist:
  - `name`, `description`, `followers`
  - `images.items[].sources[]`
  - `ownerV2.data.{name,username,uri}`
  - `content.items[].itemV2.data`
  - `content.totalCount`
  - `content.pagingInfo.nextOffset`

## Data Model

Driver: `github.com/duckdb/duckdb-go/v2`

Lists are stored as JSON text (`VARCHAR`) when necessary.

### spotify.duckdb

```sql
CREATE TABLE IF NOT EXISTS tracks (
  track_id            VARCHAR PRIMARY KEY,
  name                VARCHAR,
  duration_ms         BIGINT,
  track_number        INTEGER,
  disc_number         INTEGER,
  playable            BOOLEAN,
  preview_url         VARCHAR,
  playcount           BIGINT,
  album_id            VARCHAR,
  album_name          VARCHAR,
  cover_url           VARCHAR,
  release_date        VARCHAR,
  url                 VARCHAR,
  spotify_uri         VARCHAR,
  source_title        VARCHAR,
  source_description  VARCHAR,
  fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS albums (
  album_id            VARCHAR PRIMARY KEY,
  name                VARCHAR,
  album_type          VARCHAR,
  release_date        VARCHAR,
  total_tracks        INTEGER,
  cover_url           VARCHAR,
  copyright_text      VARCHAR,
  url                 VARCHAR,
  spotify_uri         VARCHAR,
  source_title        VARCHAR,
  source_description  VARCHAR,
  fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS artists (
  artist_id           VARCHAR PRIMARY KEY,
  name                VARCHAR,
  biography           VARCHAR,
  followers           BIGINT,
  monthly_listeners   BIGINT,
  avatar_url          VARCHAR,
  external_links_json VARCHAR,
  url                 VARCHAR,
  spotify_uri         VARCHAR,
  source_title        VARCHAR,
  source_description  VARCHAR,
  fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlists (
  playlist_id         VARCHAR PRIMARY KEY,
  name                VARCHAR,
  description         VARCHAR,
  followers           BIGINT,
  owner_name          VARCHAR,
  owner_username      VARCHAR,
  owner_uri           VARCHAR,
  image_url           VARCHAR,
  total_items         INTEGER,
  next_offset         INTEGER,
  url                 VARCHAR,
  spotify_uri         VARCHAR,
  source_title        VARCHAR,
  source_description  VARCHAR,
  fetched_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS track_artists (
  track_id            VARCHAR NOT NULL,
  artist_id           VARCHAR NOT NULL,
  artist_name         VARCHAR,
  ord                 INTEGER,
  PRIMARY KEY (track_id, artist_id)
);

CREATE TABLE IF NOT EXISTS album_artists (
  album_id            VARCHAR NOT NULL,
  artist_id           VARCHAR NOT NULL,
  artist_name         VARCHAR,
  ord                 INTEGER,
  PRIMARY KEY (album_id, artist_id)
);

CREATE TABLE IF NOT EXISTS album_tracks (
  album_id            VARCHAR NOT NULL,
  track_id            VARCHAR NOT NULL,
  ord                 INTEGER,
  PRIMARY KEY (album_id, track_id)
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
  playlist_id         VARCHAR NOT NULL,
  track_id            VARCHAR NOT NULL,
  ord                 INTEGER,
  added_by            VARCHAR,
  PRIMARY KEY (playlist_id, track_id)
);

CREATE TABLE IF NOT EXISTS artist_related (
  artist_id           VARCHAR NOT NULL,
  related_artist_id   VARCHAR NOT NULL,
  related_name        VARCHAR,
  ord                 INTEGER,
  PRIMARY KEY (artist_id, related_artist_id)
);
```

### state.duckdb

Same queue/jobs/visited pattern as `pkg/scrape/goodread/state.go` and `pkg/scrape/pinterest/state.go`.

`entity_type` values:

- `track`
- `album`
- `artist`
- `playlist`

Suggested crawl priorities:

- `artist`: 10
- `album`: 8
- `track`: 6
- `playlist`: 5

## URL and URI Normalization

Accepted input forms:

- raw ID: `11dFghVXANMlKmJXsNCbNl`
- Spotify URI: `spotify:track:11dFghVXANMlKmJXsNCbNl`
- full URL: `https://open.spotify.com/track/11dFghVXANMlKmJXsNCbNl`
- locale URL: `https://open.spotify.com/intl-ja/track/...`

Normalization rules:

- strip query string and locale prefixes
- normalize to canonical `https://open.spotify.com/<entity>/<id>`
- parse and preserve entity type from URI/URL
- reject unsupported entity kinds early

## Task Behavior

### `TrackTask`

- normalize input to canonical track URL
- fetch and parse page
- extract fallback SEO metadata
- decode bootstrap entity
- store track row
- upsert `track_artists`
- enqueue:
  - album
  - all artists

### `AlbumTask`

- fetch and parse album page
- store album row
- upsert `album_artists`
- upsert `album_tracks`
- store minimal track rows for embedded tracks
- enqueue:
  - album artists
  - album tracks

### `ArtistTask`

- fetch and parse artist page
- store artist row
- store minimal album rows from latest/popular/discography
- store minimal track rows from top tracks
- upsert `artist_related`
- enqueue:
  - discography albums/singles
  - top tracks
  - related artists

### `PlaylistTask`

- fetch and parse playlist page
- store playlist row
- store minimal track rows for visible playlist entries
- upsert `playlist_tracks`
- enqueue:
  - playlist tracks
  - track artists
  - track albums

### `CrawlTask`

Queue-driven worker pool:

- pop pending rows from `state.duckdb`
- dispatch to entity-specific tasks
- mark `visited` on success / `pending|failed` on retry exhaustion
- emit periodic progress snapshots using `pkg/core.Task`

## CLI

`search spotify`

Subcommands:

- `track <id|uri|url>`
- `album <id|uri|url>`
- `artist <id|uri|url>`
- `playlist <id|uri|url>`
- `seed --file <path> --entity <track|album|artist|playlist> [--priority N]`
- `crawl [--workers N] [--delay MS]`
- `info`
- `jobs [--limit N]`
- `queue [--status pending] [--limit N]`

Examples:

```bash
search spotify track 11dFghVXANMlKmJXsNCbNl
search spotify album spotify:album:0tGPJ0bkWOUmH7MEOR77qc
search spotify artist https://open.spotify.com/artist/6sFIWsNpZYqfjUpaCgueju
search spotify playlist 37i9dQZF1DXcBWIGoYBM5M
search spotify seed --file artists.txt --entity artist --priority 10
search spotify crawl --workers 3 --delay 250
search spotify info
```

## External Learnings Applied

From Spotify public pages and the broader GitHub ecosystem:

- Normalize every ID/URI/URL to a canonical URL before queueing or deduping.
- Prefer extracting stable bootstrap/state payloads over brittle DOM selectors.
- Keep entity parsing tolerant: store partial records when only subset fields are present.
- Treat crawl expansion as graph traversal from seed entities, not as a single monolithic “crawl all” endpoint.
- Keep HTML metadata as a fallback path so parser regressions degrade gracefully instead of fully failing.

## Validation

Implementation acceptance criteria:

- `go build ./cli/...` and package build succeed
- direct commands fetch and store at least one row for each entity type
- `seed` + `crawl` traverses discovered relations without duplicate queue explosions
- `info`, `jobs`, and `queue` produce usable output
