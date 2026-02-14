# 0534 — Translate Web Pages

**Status**: Draft
**Date**: 2026-02-14
**Depends on**: 0533 (Translate App)

## Overview

Add web page translation to the translate app at `https://translate.go-mizu.workers.dev`. Users enter a URL, the worker fetches the page, translates all visible text, rewrites internal links to stay within the translation proxy, and returns the translated HTML. Translated URLs are permanent and shareable.

## How Google Translate Does It

### URL Format (Modern)

Google uses domain-encoded proxy URLs:
```
https://example-com.translate.goog/path?_x_tr_sl=auto&_x_tr_tl=vi&_x_tr_hl=en
```
Dots in the original domain become hyphens, appended with `.translate.goog`.

### Process

1. User submits URL to translate
2. Google fetches the original page server-side
3. Translates all visible text content (text nodes in the DOM)
4. Rewrites all `<a href>` links to stay within the `translate.goog` proxy
5. Injects `<base href>` so relative resources (CSS, images, JS) resolve to original domain
6. Adds `translated-ltr`/`translated-rtl` class to `<html>`, updates `lang` attribute
7. Serves translated HTML through proxy domain

### What Gets Translated

- Text nodes inside visible elements: `<p>`, `<h1>`-`<h6>`, `<li>`, `<td>`, `<th>`, `<span>`, `<div>`, `<a>`, `<label>`, `<button>`
- `<title>` element
- `alt` attributes on `<img>`
- `placeholder` attributes on `<input>`, `<textarea>`
- `content` attribute on `<meta name="description">`

### What Does NOT Get Translated

- `<script>`, `<style>`, `<code>`, `<pre>`, `<kbd>`, `<samp>`, `<var>` content
- `<noscript>`, `<canvas>`, `<audio>`, `<video>`, `<iframe>`, `<svg>`
- Elements with `class="notranslate"` or `translate="no"` attribute
- HTML tags, attributes, CSS, JavaScript code

### Link Rewriting

All `<a href>` links are rewritten to route through the translation proxy, preserving `tl` and `sl` parameters. Resource URLs (images, CSS, JS) are absolutized but served directly from the original domain (not proxied).

## Our Implementation

### URL Structure

**Permanent, shareable URLs** — path-based format:

```
https://translate.go-mizu.workers.dev/page/<tl>/<url>
```

Examples:
```
/page/vi/https://example.com
/page/ja/https://news.ycombinator.com
/page/fr/https://en.wikipedia.org/wiki/Hello
```

Source language is always auto-detected (no `sl` parameter needed — simpler URLs).

### Architecture

**Two-phase approach** in a single CF Worker request:

```
Phase 1: Fetch + Extract
  ┌─────────────────────────────────┐
  │ fetch(targetUrl)                │
  │ → get original HTML             │
  │ → buffer full response          │
  └─────────────────────────────────┘

Phase 2: Translate + Rewrite (HTMLRewriter streaming)
  ┌─────────────────────────────────┐
  │ HTMLRewriter                     │
  │  .on('head') → inject <base>    │
  │  .on('a[href]') → rewrite links │
  │  .on('form[action]') → rewrite  │
  │  .on('img,script,link') → abs   │
  │  .on('p,h1..h6,li,...') → xlate │
  │  .on('title') → translate       │
  │  .on('html') → set lang attr    │
  └─────────────────────────────────┘
```

### Text Translation Strategy

**Per-element async translation** using HTMLRewriter async text handlers:

1. HTMLRewriter walks the DOM streaming
2. For each translatable text node, concatenate chunks until `lastInTextNode === true`
3. If accumulated text has content (not just whitespace), call Google Translate API
4. Replace the text with the translated version

This is simpler than batch extraction, and since CF Workers allow async handlers in HTMLRewriter, each text node can independently `await` the translation API call.

**Google Translate preserves inline HTML** — we can send text with `<b>`, `<a>`, `<span>` tags and it will translate around them, keeping the tags intact.

### API Endpoint

```
GET /page/<tl>/<encoded-url>
```

**Parameters:**
- `tl` — target language code (required, in path)
- URL — the page to translate (required, in path after `tl/`)

**Response:** Translated HTML with:
- `<base href="https://original-domain.com/">` injected in `<head>`
- All `<a href>` links rewritten to `/page/<tl>/absolute-url`
- `<html lang="<tl>">` attribute set
- Text content translated to target language
- Resources (CSS, JS, images) loaded from original domain via `<base>`

**Error responses:**
- 400: Missing URL or invalid target language
- 502: Failed to fetch original page
- 504: Original page timeout

### Link Rewriting Rules

| Element | Attribute | Rewrite? | How |
|---------|-----------|----------|-----|
| `<a>` | `href` | Yes | `/page/<tl>/absoluteUrl` |
| `<form>` | `action` | Yes | `/page/<tl>/absoluteUrl` |
| `<img>` | `src` | No | Resolved by `<base>` |
| `<link>` | `href` | No | Resolved by `<base>` |
| `<script>` | `src` | No | Resolved by `<base>` |
| `<source>` | `src` | No | Resolved by `<base>` |

### Skip Rules

Do NOT translate content inside:
- `<script>`, `<style>`, `<code>`, `<pre>`, `<kbd>`, `<samp>`, `<var>`, `<svg>`
- `<noscript>`, `<canvas>`, `<audio>`, `<video>`, `<iframe>`
- Elements with `translate="no"` or `class` containing `notranslate`
- Text that is only whitespace, numbers, or punctuation

### Frontend Integration

Add a "Translate Page" tab/section to the existing translate app:

```
┌──────────────────────────────────────────────────┐
│  🌐 Translate                        [🌙 Theme]  │
├──────────────────────────────────────────────────┤
│  [Text]  [Page]                                   │  ← Tab switcher
│                                                    │
│  ┌──────────────────────────────────────────┐    │
│  │ 🔗 https://example.com          [▾ vi]   │    │
│  │                                    [Go →] │    │
│  └──────────────────────────────────────────┘    │
│                                                    │
│  Translated page opens in an iframe or new tab     │
│  URL: /page/vi/https://example.com                │
│  (permanent, shareable, bookmarkable)             │
│                                                    │
└──────────────────────────────────────────────────┘
```

The translated page URL `/page/vi/https://example.com` is permanent and shareable — anyone can visit it directly. All links within the translated page stay within the proxy.

### CF Worker Constraints

| Resource | Free Plan | Our Usage |
|----------|-----------|-----------|
| Subrequests | 50 | 1 (page fetch) + N (translate API calls) |
| CPU time | 10ms | Tight — use streaming HTMLRewriter |
| Memory | 128 MB | Buffering large pages could be an issue |

To stay within limits:
- Stream the response through HTMLRewriter (don't buffer entire HTML)
- Each text node triggers one translation API call (async, parallel-ish within stream)
- Skip empty/whitespace text nodes to reduce API calls
- Only translate text nodes inside block-level elements (skip inline-only elements to reduce calls)

### Translation Banner

Inject a small fixed banner at the top of translated pages:

```html
<div style="position:fixed;top:0;left:0;right:0;z-index:999999;background:#f0f0f0;padding:4px 12px;font:13px sans-serif;text-align:center;border-bottom:1px solid #ddd;">
  Translated from <b>English</b> to <b>Vietnamese</b> —
  <a href="ORIGINAL_URL">View original</a> |
  <a href="https://translate.go-mizu.workers.dev">Translate another page</a>
</div>
```

### Files

```
worker/
  routes/page.ts          — GET /page/:tl/* handler
  page-rewriter.ts        — HTMLRewriter element/text handlers
worker/index.ts           — mount page route
src/
  App.tsx                 — add Page tab alongside Text translation
  components/
    page-translator.tsx   — URL input + language select + iframe/link
```

## Non-Goals

- No JavaScript execution / dynamic content translation (server-side only)
- No caching of translated pages (can add KV cache later)
- No support for password-protected or cookie-authenticated pages
- No CSS/layout modification beyond `<base>` tag injection
- No translation of content loaded via JavaScript (AJAX, SPA)
