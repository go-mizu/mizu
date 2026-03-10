# X Article Download & Rendering

**Date:** 2026-03-10
**Status:** Implemented

## Overview

X Articles are long-form content published on X (Twitter) using the "Articles" feature. They are rendered client-side via React/Draft.js, making them inaccessible to simple HTTP fetchers. This spec describes how we extract and render X Articles in both the Go CLI (`search x tweet --format markdown`) and the Cloudflare Worker web viewer (`tools/x-viewer`).

## Article URL Patterns

- **Embedded link (in tweets):** `https://x.com/i/article/{articleID}` — requires authentication
- **Public URL:** `https://x.com/{username}/article/{tweetID}` — publicly accessible with auth cookies

The tweet that links to an article contains only a `t.co` shortened URL in its text body. The `t.co` URL resolves to the `/i/article/{articleID}` form. The `articleID` is different from the `tweetID`.

## Architecture

### CLI Flow (`search x tweet <url> --format markdown`)

1. Fetch tweet via X GraphQL API (`TweetDetail` endpoint)
2. Detect article link: `isTweetJustALink()` + `extractXArticleID()` checks for `/i/article/` in URLs
3. Construct public article URL: `https://x.com/{username}/article/{tweetID}`
4. Launch headless Chrome via Rod with session cookies (`auth_token`, `ct0`)
5. Navigate to article URL, wait up to 20s for `[data-testid="twitterArticleRichTextView"]`
6. Extract structured content via JavaScript DOM walker (see DOM Structure below)
7. Render as markdown via `TweetThreadToMarkdown()`
8. Store in DuckDB `articles` table

### Worker Flow (`tools/x-viewer`)

1. Route: `GET /:username/article/:id`
2. Fetch tweet via X GraphQL API (same `TweetDetail` endpoint used for tweet pages)
3. Parse article content from API response:
   - `note_tweet.note_tweet_results.result` → title + rich text
   - `article.article_results.result` → article body (plain text or rich content)
4. Render as server-side HTML with article-specific styles
5. Cache in KV (1 hour TTL, same as tweets)

## X Article DOM Structure (Browser-Rendered)

The rendered article page uses Draft.js with these key elements:

| Element | data-testid / class | Content |
|---------|---------------------|---------|
| Title | `twitter-article-title` | Article H1 title |
| Body container | `twitterArticleReadView` | Wraps all content |
| Rich text wrapper | `twitterArticleRichTextView` | Inner content area |
| Text paragraph | `.longform-unstyled[data-block]` | Regular paragraph |
| Header H1 | `.longform-header-one` | Section header (rendered as `# `) |
| Header H2 | `<h2>.longform-header-two` | Sub-header (rendered as `## `) |
| Code block | `markdown-code-block` | Fenced code with language label |
| Image | `tweetPhoto` → `<img>` | Article image |
| List item | `.longform-ordered-list-item` / `.longform-unordered-list-item` | Bullet / numbered |
| Blockquote | `.longform-blockquote` | Block quote |
| Inline link | `<a href>` inside text blocks | Hyperlinks |
| Rich text block | `longformRichTextComponent` | Text content wrapper (NOT a code block) |

### Code Block Structure

```
data-testid="markdown-code-block"
  └─ <span>python</span>           ← language label
  └─ <button>Copy</button>         ← copy button (text to strip)
  └─ <div>...code spans...</div>   ← syntax-highlighted code
```

`innerText` of the `markdown-code-block` element gives: `"python\nCopy\n{code}"`. Strip the language prefix and "Copy" button text to get clean code.

### Image Structure

```
data-testid="tweetPhoto"
  └─ <img src="https://pbs.twimg.com/media/...?format=png&name=small">
```

Replace `name=small` → `name=large` for full resolution.

## Markdown Output Format

```markdown
# {Title}

**{Name}** (@{Username})

*{PostedAt UTC}*

👍 {Likes} · 🔄 {Retweets} · 💬 {Replies} · 👁 {Views}

---

{Article body with proper markdown formatting}

---

*Source: [{URL}]({URL})*
```

### Formatting Rules

- Paragraphs: separated by blank lines
- Headers: `#` / `##` / `###`
- Code blocks: ` ```{language}\n{code}\n``` `
- Images: `![Image]({url})`
- Links: `[text](url)` inline
- Lists: `- ` for unordered, `1. ` for ordered
- Blockquotes: `> text`

## DuckDB Schema

### `articles` Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR PK | Tweet ID |
| username | VARCHAR | Author username |
| name | VARCHAR | Author display name |
| title | VARCHAR | Article title |
| content_md | VARCHAR | Full markdown content |
| tweet_count | INTEGER | Thread length |
| likes | INTEGER | Like count |
| retweets | INTEGER | Retweet count |
| replies | INTEGER | Reply count |
| views | INTEGER | View count |
| posted_at | TIMESTAMP | Original post time |
| fetched_at | TIMESTAMP | Fetch time (default NOW()) |

## Key Lessons

- **`longformRichTextComponent` is NOT a code block** — it wraps all text content. Real code blocks use `data-testid="markdown-code-block"`.
- **`<noscript>` content is always in the DOM** — checking for "JavaScript is not available" in `page.HTML()` is unreliable. Check for `twitterArticleRichTextView` presence instead.
- **Auth cookies required** — X shows a login wall without cookies. Set cookies at browser level before navigation: `browser.SetCookies()`.
- **Public URL pattern** — `/{username}/article/{tweetID}` works with auth cookies; `/i/article/{articleID}` redirects and may not work.
- **Rate limiting is intermittent** — sometimes the article renders, sometimes X shows a login wall. The 20s wait loop with `twitterArticleRichTextView` detection handles this.

## Worker HTML Rendering

The x-viewer worker adds a `/:username/article/:id` route that:

1. Fetches the tweet to get metadata (author, stats, posted time)
2. Fetches the article page HTML via the same GraphQL API or uses the tweet's article data if available
3. Renders a reader-friendly HTML page with:
   - Article title as H1
   - Author info + avatar
   - Stats bar (likes, retweets, views)
   - Article body with proper HTML formatting (code blocks with syntax highlighting, images, headers)
   - Link back to original on X

## Files

| File | Purpose |
|------|---------|
| `cli/x.go` | `fetchXArticleWithRod()` — headless browser extraction |
| `cli/x.go` | `runXTweet()` — article detection + markdown export flow |
| `pkg/dcrawler/x/export.go` | `TweetThreadToMarkdown()` — markdown rendering |
| `pkg/dcrawler/x/db.go` | `Article` type, `InsertArticle()`, `articles` table schema |
| `pkg/dcrawler/x/parse.go` | Note tweet title + article body parsing from API |
| `tools/x-viewer/src/routes/article.ts` | Worker article route |
| `tools/x-viewer/src/html.ts` | `renderArticlePage()` — HTML rendering |

## Sample Articles

- `https://x.com/browser_use/status/2031045678411981115` → "How to Authenticate AI Web Agents"
- `https://x.com/LangChain/status/2031055593360990358` → "How we built LangChain's GTM Agent"
