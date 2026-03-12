# pkg/dcrawler/x — Architecture

X/Twitter scraping package. Provides cookie-based authenticated access, guest
token anonymous access, and the syndication embed API — layered so guest token
is tried first on every public read, preserving session rate-limit budget.

---

## File Map

| File | Role |
|------|------|
| `client.go` | High-level API (`Client`), public method surface, request routing |
| `graphql.go` | Authenticated HTTP GraphQL client, rate-limit tracking |
| `guest.go` | Guest token activation, anonymous GraphQL calls |
| `syndication.go` | Twitter embed API (`cdn.syndication.twimg.com`) — zero auth |
| `parse.go` | JSON response parsers for every GraphQL shape |
| `types.go` | `Profile`, `Tweet`, `FollowUser`, `RateLimitError`, callbacks |
| `consts.go` | Bearer token, GraphQL query IDs, feature flags, user-agent |
| `config.go` | `Config` struct, all path helpers |
| `session.go` | Session JSON persistence, profile JSON persistence |
| `db.go` | DuckDB storage — tweets, users, articles tables |
| `export.go` | JSON / CSV / RSS / Markdown export |
| `download.go` | Concurrent media download (photos, videos, GIFs) |
| `tid.go` | `x-client-transaction-id` header generation |

---

## Authentication Layers

The package has three authentication layers, tried in priority order:

```
1. Syndication API     cdn.syndication.twimg.com/tweet-result
                       ↳ No auth. Token derived from tweet ID.
                       ↳ Used only for GetTweet (individual tweets).

2. Guest Token         POST /1.1/guest/activate.json → guest_token
                       ↳ Anonymous session. Rate limit pool separate from user sessions.
                       ↳ Cached 45 min, invalidated on 401/403.
                       ↳ Works for: profiles, single tweets, timelines, replies, search.

3. Cookie Auth         auth_token + ct0 (CSRF) cookies
                       ↳ Full authenticated session imported from browser.
                       ↳ Required only for: bookmarks, home/for-you timelines, private accounts.
                       ↳ Used as fallback when guest token fails.
```

### Guest-First Design

Every public-read method routes through `doGuestFirst`:

```
doGuestFirst(endpoint, vars, toggles):
  1. fetchGuestToken()               → cached 45 min
  2. doGuestGraphQL(token, ...)      → x-guest-token header
  3. on RateLimitError:
       invalidateGuestToken()
       fetchGuestToken()             → fresh token
       doGuestGraphQL(token2, ...)   → retry with new token
  4. on failure: c.gql.doGraphQL    → cookie auth (if session configured)
```

This means fetching 10 000 tweets with `--all` consumes almost zero of the
user's session rate-limit quota — guest tokens have their own per-IP pool.

Auth-only endpoints (bookmarks, home timeline) bypass `doGuestFirst` and call
`c.gql.doGraphQL` directly — they'll get a 401 from guest token so there's no
point trying.

---

## `graphql.go` — Authenticated HTTP Client

```go
type graphqlClient struct {
    http      *http.Client
    authToken string         // auth_token cookie value
    ct0       string         // CSRF token
    mu        sync.Mutex
    rlRemaining int          // from x-rate-limit-remaining header
    rlReset     time.Time    // from x-rate-limit-reset header (Unix epoch)
    rlLastUpdated time.Time
}
```

Every response updates `rlRemaining` / `rlReset`. `PacedDelay(minDelay)` spreads
the remaining quota evenly across the time window:

```
timeLeft  = rlReset - now
paced     = timeLeft / rlRemaining
returned  = max(paced, minDelay)
```

`doGraphQL` builds the request:
- URL: `https://x.com/i/api/graphql/{queryID}?variables=...&features=...&fieldToggles=...`
- Headers: `authorization: Bearer ...`, `x-twitter-auth-type: OAuth2Session`,
  `x-csrf-token: {ct0}`, `cookie: auth_token={t}; ct0={ct0}`
- Also sets: `x-client-transaction-id` (from `tid.go`), `sec-*` headers, gzip accept
- On 429 or API error code 88: returns `*RateLimitError{ResetAt, Wait}`
- On 89/239/326/37: returns hard error (expired/bad token, locked, suspended)
- Non-critical errors with data present: returns data (partial success)

---

## `guest.go` — Guest Token Client

```go
var (
    guestMu        sync.Mutex
    cachedToken    string
    cachedTokenExp time.Time   // 45 min TTL
)
```

**Token activation:**
```
POST https://api.twitter.com/1.1/guest/activate.json
Authorization: Bearer {publicBearerToken}
→ {"guest_token": "1234567890"}
```

`doGuestGraphQL` mirrors `doGraphQL` but uses `x-guest-token: {token}` instead
of cookie auth. On 401/403 it calls `invalidateGuestToken()` to force a fresh
activation on the next call.

Package-level functions:
- `GetProfileGuest(username)` — `UserByScreenName` via guest token
- `GetTweetGuest(id)` — `TweetDetail` via guest token, parses via `parseConversation`

---

## `syndication.go` — Embed API

Used exclusively for `GetTweet`. No auth at all — this is the endpoint Twitter
uses for its own embed widgets.

**Token formula (mirrors Twitter's embed JS):**
```go
token = Math.round(parseInt(id) / 1e15 * Math.PI).toString(36)
```

```
GET https://cdn.syndication.twimg.com/tweet-result
    ?id={tweetID}&lang=en&token={token}
    Referer: https://platform.twitter.com/
```

Response is ISO 8601 `created_at` (not Ruby date), includes photos/video/GIF
via `extended_entities.media`, hashtags/mentions/URLs in `entities`.

Limitations vs GraphQL: no view count, no reply thread, no quote tweet body.

---

## `client.go` — Public API Surface

### `Client` struct

```go
type Client struct {
    gql        *graphqlClient   // nil until SetAuthToken called
    cfg        Config
    authToken  string
    ct0        string
    searchMode string           // SearchTop/Latest/Photos/Videos/People
    userCache  map[string]string // username → rest_id (avoids repeated profile fetch)
}
```

### Public Methods by Category

**Auth setup:**
- `NewClient(cfg)` — creates unauthenticated client (works via guest token)
- `SetAuthToken(authToken, ct0)` — initialises `graphqlClient`
- `SetCookies(cookies)` — extracts `auth_token`/`ct0` from cookie slice
- `LoadSessionFile(path)` — loads JSON session, calls `SetAuthToken`
- `SaveSessionFile(path, username)` — persists session to disk
- `Activate()` — validates session via lightweight API call
- `HasAuth()` — true if cookie session is configured

**Profile:**
- `GetProfile(username)` — guest → auth; caches username→ID in `userCache`
- `GetProfileNoAuth(username)` — guest token only

**Single tweet:**
- `GetTweet(id)` — syndication → guest → auth; the main fetch path
- `GetTweetByRestID(id)` — `TweetResultByRestId` endpoint; needed for X Articles
- `GetTweetReplies(id)` — paginated `TweetDetail`; uses `doGuestFirst` per page
- `GetTweetNoAuth(id)` / package-level `GetTweetNoAuth(id)` — syndication → guest

**Timeline (paginated):**
- `GetTweets(ctx, username, maxTweets, cb)` — UserTweets, excludes replies
- `GetTweetsWithBatch(...)` — same but calls `batchCb` per page for incremental DB save
- `GetTweetsAndReplies(...)` — UserTweetsAndReplies endpoint
- `GetMediaTweets(...)` — UserMedia endpoint

**Search:**
- `SearchTweets(ctx, query, max, cb)` — SearchTimeline, respects `searchMode`
- `SearchTweetsWithBatch(...)` — with incremental save
- `SearchProfiles(ctx, query, max, cb)` — SearchTimeline with `product: People`

**Follow lists (paginated):**
- `GetFollowers(ctx, username, max, cb)`
- `GetFollowing(ctx, username, max, cb)`
- `GetRetweeters(tweetID, max)`
- `GetFavoriters(tweetID, max)`

**Lists & Spaces:**
- `GetListByID(id)` / `GetListBySlug(owner, slug)`
- `GetListTweets(ctx, listID, max, cb)`
- `GetListMembers(ctx, listID, max, cb)`
- `GetSpace(id)` — audio space metadata
- `GetTrends()` — trending topics via ExplorePage

**Auth-only (home/feed):**
- `GetBookmarks(ctx, max, cb)` — BookmarkSearchTimeline
- `GetHomeTweets(ctx, max, cb)` — HomeTimeline
- `GetForYouTweets(ctx, max, cb)` — HomeLatestTimeline

### Pagination Pattern

All list/timeline methods use the same cursor loop:

```
cursor = ""
for {
    vars["cursor"] = cursor      // only if cursor != ""
    data = doGraphQLRetry(...)   // guest → auth, with rate-limit retry
    result = parseTimeline(data)
    tweets += result.Tweets
    if batchCb != nil: batchCb(result.Tweets)   // incremental DB save
    if result.Cursor == "": break
    if emptyPages >= maxEmpty: break             // 3 for bounded, 10 for --all
    cursor = result.Cursor
    sleep(cfg.Delay)              // 500ms default, or PacedDelay from headers
}
```

### Rate Limit Retry (`doGraphQLRetry`)

```
doGuestFirst(...)               → try guest token (rotates on rate limit)
on RateLimitError:
  for retry in 0..2:
    wait = min(max(rle.Wait, 10s), 16min)
    log to cb: "rate limited, waiting Xs (resets HH:MM:SS)"
    sleep(wait)
    doGuestFirst(...)            → try again (may use fresh guest token)
```

---

## `parse.go` — Response Parsing

### Navigation helpers

```go
dig(m, "data", "user", "result")     // nil-safe nested map walk
asMap(v) map[string]any              // type assert or nil
asStr(v) string
asInt(v) int                         // handles float64, int, string
asBool(v) bool
asSlice(v) []any
```

### Timeline parsing

`parseTimeline(data)` handles both `UserTweets` and `UserTweetsAndReplies`:

```
data.data.user.result.timeline_v2.timeline.instructions[]
  ↳ type == "TimelineAddEntries":
      entries[] → each entry.content.itemContent or entry.content.items[]
        ↳ tweet_results.result → parseGraphTweet(node)
        ↳ type == "TimelineTimelineCursor" with cursorType == "Bottom" → cursor
```

### Tweet parsing (`parseGraphTweet`)

```
node (tweet_results.result):
  __typename                            → check for "TweetWithVisibilityResults" wrapper
  tweet_results.result.legacy:
    full_text, created_at, conversation_id_str
    in_reply_to_status_id_str, in_reply_to_screen_name
    retweeted_status_result.result.rest_id
    quoted_status_id_str
    favorite_count, retweet_count, reply_count, bookmark_count, quote_count
    lang, possibly_sensitive
    entities: hashtags, urls (expanded_url), user_mentions, media
    extended_entities.media[]
  tweet_results.result.views.count      → view count (string, needs ParseInt)
  tweet_results.result.core:
    user_results.result.legacy.screen_name, name → username/name
  tweet_results.result.note_tweet:
    note_tweet_results.result.text      → note tweet body
    note_tweet_results.result.entity_set → note tweet entities
  tweet_results.result.article:
    article_results.result.title
```

### Conversation parsing (`parseConversation`)

`TweetDetail` returns the focal tweet and replies interleaved. The parser:
1. Finds the focal tweet (matching `tweetID`) as the main tweet
2. Collects entries in `TimelineThreadedConversation` thread items as replies
3. Returns `(mainTweet, replies, nextCursor)`

### Profile parsing (`parseGraphUser`)

Handles two API shapes (X migrated formats ~2024):

```
node.user_results.result OR node.result:
  legacy.screen_name, name, description
  legacy.profile_image_url_https (strip "_normal" for full size)
  legacy.profile_banner_url
  legacy.followers_count, friends_count, statuses_count, favourites_count
  legacy.media_count, listed_count
  legacy.verified, is_blue_verified
  legacy.pinned_tweet_ids_str[]
  legacy.created_at                    → parseTwitterTime
  professional.professional_type, category[0].name
  can_dm, default_profile, default_profile_image
  legacy.entities.description.urls[]  → expand bio URLs
```

---

## `types.go` — Data Models

### `Tweet`

```go
type Tweet struct {
    ID, ConversationID string
    Title              string    // Note tweet title / X Article title
    Text, HTML         string
    Username, UserID, Name string
    PermanentURL       string    // https://x.com/{username}/status/{id}
    IsRetweet, IsReply, IsQuote, IsPin, IsEdited bool
    ReplyToID, ReplyToUser     string
    QuotedID, RetweetedID      string
    Likes, Retweets, Replies, Views, Bookmarks, Quotes int
    Photos, Videos, GIFs       []string  // media URLs
    Hashtags, Mentions, URLs   []string
    Sensitive bool
    Language, Source, Place    string
    PostedAt, FetchedAt        time.Time
}
```

### `Profile`

```go
type Profile struct {
    ID, Username, Name, Biography string
    Avatar, Banner, Location, Website, URL string
    Joined          time.Time
    Birthday        string
    FollowersCount, FollowingCount, TweetsCount int
    LikesCount, MediaCount, ListedCount         int
    IsPrivate, IsVerified, IsBlueVerified        bool
    PinnedTweetIDs                              []string
    ProfessionalType, ProfessionalCategory      string
    CanDM, DefaultProfile, DefaultAvatar         bool
    DescriptionURLs                             []string
    FetchedAt                                   time.Time
}
```

### `RateLimitError`

```go
type RateLimitError struct {
    ResetAt time.Time     // when limit resets (zero = unknown)
    Wait    time.Duration // how long to wait
}
```

Used throughout the retry machinery to distinguish rate limits from other errors.

---

## `db.go` — DuckDB Persistence

```go
type DB struct {
    db   *sql.DB
    path string
}
```

### Schema

**`tweets` table (36 columns):**
All `Tweet` fields. Arrays (`photos`, `videos`, `hashtags`, etc.) stored as
JSON strings. `NULL` used for empty strings to save space.

**`users` table (27 columns):**
All `Profile` fields. Arrays as JSON strings.

**`articles` table:**
`id` (tweet ID), `username`, `name`, `title`, `content_md`, engagement counts,
`posted_at`, `fetched_at`.

### Key operations

- `InsertTweets(tweets)` — batches of 500, wrapped in transactions.
  Uses `INSERT OR REPLACE` to handle duplicates gracefully.
- `InsertUser(p)` / `InsertFollowUsers(users)` — same batch pattern.
- `GetStats()` — tweet count, user count, file size (via `dbstat` pragma).
- `TweetCountInRange(since, until)` — used by timeline window deduplication.
- `TopTweets(limit)` — `ORDER BY likes DESC`.

### Schema migration

`OpenDB` runs `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` for every column on
open. This is safe for existing databases — adds missing columns with NULL
defaults. Columns are never removed, keeping backward compatibility.

---

## `export.go` — Output Formats

All formats share the `[]Tweet` input; path is the output file.

| Format | Function | Notes |
|--------|----------|-------|
| JSON | `ExportJSON` | Array of Tweet structs |
| CSV | `ExportCSV` | 29 columns, arrays as semicolon-joined |
| RSS 2.0 | `ExportRSS` | `<item>` per tweet; photos in `<description>` HTML |
| Markdown | `ExportMarkdown` | `TweetThreadToMarkdown` → file write |

### Thread Markdown rendering

`TweetThreadToMarkdown(thread []Tweet)` produces:

```markdown
# {Title}

**{Name}** (@{Username})

*2026-03-09 16:33 UTC*

👍 209 · 🔄 45 · 💬 12 · 👁 8.4K

---

{Tweet text}

![Image](https://pbs.twimg.com/...)

---

*Source: [https://x.com/...](https://x.com/...)*
```

`ExtractThread(root, replies)` filters replies to only those where
`Username == root.Username` (self-replies), returns them in posted_at order.

---

## `download.go` — Concurrent Media Download

```go
type MediaItem struct {
    TweetID, URL, Type string   // Type: photo/video/gif
    Index              int      // position in multi-photo tweet
}
```

`DownloadMedia(ctx, items, dir, workers, cb)`:
- Creates `dir` if missing
- Runs `workers` goroutines (default 8) consuming `items` channel
- Per item: checks file exists by non-zero size → skips; else downloads
- 120s per-file timeout via `context.WithTimeout`
- Atomic counters for `Downloaded`, `Skipped`, `Failed`, `Bytes`
- `cb` called after each item with a `DownloadProgress` snapshot

Filename format: `{tweetID}_{index}.{ext}` — extension inferred from URL
path or falls back to type hint (`.jpg` / `.mp4` / `.gif`).

---

## `tid.go` — Transaction ID Header

X requires `x-client-transaction-id` on all GraphQL requests. The algorithm
is ported from [Nitter's implementation](https://github.com/zedeus/nitter).

```
1. Fetch pair dict from GitHub (1h cache, stale-on-failure)
2. Pick random (animationKey, verification) pair
3. key   = base64decode(animationKey)
4. time  = (now.UnixMilli() / 1000).to_bytes(big-endian)
5. hash  = SHA256("GET!" + path + "!" + timeStr + tidKeyword + animationKey)
6. out   = key + time + hash[0:16] + [3]
7. XOR each byte with out[0] (a random byte seeded by first key byte)
8. return base64(out) without padding
```

Failure is non-fatal — `generateTID` returns `("", nil)` on error and the
request proceeds without the header (most endpoints still work).

---

## `config.go` — Paths

Default `DataDir`: `$HOME/data/x`

```
DataDir/
  .sessions/
    {username}.json         ← session file (0o600)
  {username}/
    profile.json            ← cached profile
    tweets.duckdb           ← tweet + user DB
    media/                  ← downloaded photos/videos
    {tweetID}.md            ← exported markdown threads
  search/
    {sanitized_query}/
      tweets.duckdb
  hashtag/
    {tag}/
      tweets.duckdb
```

All paths are constructed by `Config` helpers: `UserDir`, `UserDBPath`,
`UserMediaDir`, `ProfilePath`, `SearchDir`, `SearchDBPath`, `SessionPath`, etc.

---

## `session.go` — Session Persistence

```go
type Session struct {
    Username  string
    AuthToken string         // auth_token cookie value
    CT0       string         // ct0 (CSRF) cookie value
    Cookies   []*http.Cookie // full cookie list (fallback)
    SavedAt   time.Time
}
```

`SaveSession` writes JSON with `os.WriteFile(..., 0o600)`.
`LoadSession` reads JSON; `client.LoadSessionFile` prefers explicit
`AuthToken`/`CT0` fields, falls back to extracting from `Cookies` slice for
backward compatibility with older session files.

`SaveProfile(cfg, profile)` writes profile to `{UserDir}/profile.json` (0o644).

---

## `consts.go` — API Constants

### GraphQL query IDs

Query IDs (`{hash}/{OperationName}`) are extracted from X's main JS bundle.
They change periodically. Current values:

| Constant | Operation |
|----------|-----------|
| `gqlUserByScreenName` | `UserByScreenName` |
| `gqlUserById` | `TweetResultByRestId` |
| `gqlUserTweetsV2` | `UserTweets` |
| `gqlUserTweetsAndRepliesV2` | `UserTweetsAndReplies` |
| `gqlUserMedia` | `UserMedia` |
| `gqlConversationTimeline` | `TweetDetail` |
| `gqlSearchTimeline` | `SearchTimeline` |
| `gqlFollowers` / `gqlFollowing` | `Followers` / `Following` |
| `gqlBookmarks` | `BookmarkSearchTimeline` |
| `gqlHomeTimeline` / `gqlHomeLatestTimeline` | `HomeTimeline` / `HomeLatestTimeline` |
| `gqlListById` / `gqlListBySlug` | `ListByRestId` / `ListBySlug` |
| `gqlExplorePage` | `ExplorePage` (trends) |

### Feature flags (`gqlFeatures`)

80+ boolean feature flags passed on every GraphQL request. These tell the API
which response fields to include. Derived from Nitter's implementation.
Key flags: `longform_notetweets_consumption_enabled`, `articles_api_enabled`,
`responsive_web_twitter_article_tweet_consumption_enabled`.

### Field toggles

Per-endpoint JSON toggles (second level of feature control):
- `userFieldToggles` — `{"withPayments":false,"withAuxiliaryUserLabels":true}`
- `userTweetsFieldToggles` — `{"withArticlePlainText":false}`
- `tweetDetailFieldToggles` — `{"withArticleRichContentState":true,"withArticlePlainText":true,...}`

---

## End-to-End Data Flow

### `search x tweets karpathy --all`

```
initXClientOptional("default")
  LoadSessionFile → SetAuthToken (cookie auth available as fallback)

GetTweetsWithBatch(ctx, "karpathy", 0, progressCb, batchCb)
  resolveUserID("karpathy")
    GetProfile("karpathy")
      doGuestFirst(gqlUserByScreenName, ...)
        fetchGuestToken()          → POST /1.1/guest/activate.json
        doGuestGraphQL(token, ...) → GET UserByScreenName
        parseUserResult(data)      → Profile{ID: "33836629", ...}
      userCache["karpathy"] = "33836629"

  getUserTimeline loop (~250 pages for 10K tweets):
    doGraphQLRetry(ctx, gqlUserTweetsV2, {userId:"33836629", count:40}, ...)
      doGuestFirst(...)
        doGuestGraphQL(token, gqlUserTweetsV2, ...)
        on RateLimitError: rotate token, retry
        on persistent failure: c.gql.doGraphQL (cookie auth)
      parseTimeline(data) → []Tweet (up to 40 per page)
    batchCb(tweets)
      db.InsertTweets(batch)  → DuckDB INSERT OR REPLACE (transaction)
    sleep(500ms)
    cursor = result.Cursor → next page
```

### `search x tweet {id}`

```
initXClientOptional("default")

GetTweet(id)
  GetTweetSyndication(id)          → cdn.syndication.twimg.com/tweet-result
    syndicationToken(id)           → Math.round(id / 1e15 * π).toString(36)
    GET ?id={id}&token={tok}
    parseSyndicationTweet → Tweet  ← returned immediately on success

  [fallback if syndication fails]
  doGuestFirst(gqlConversationTimeline, {focalTweetId:id, ...})
    doGuestGraphQL(token, ...)
    parseConversation(data, id) → (mainTweet, _, _)

  [fallback if guest fails]
  c.gql.doGraphQL(gqlConversationTimeline, ...)
  parseConversation → mainTweet

GetTweetReplies(id)
  loop:
    doGuestFirst(gqlConversationTimeline, {cursor: ...})
    parseConversation → (_, replies, nextCursor)
```
