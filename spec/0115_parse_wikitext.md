# Parse WikiText for Internal Links

## Overview

This document describes the design for preserving internal wiki links when rendering pages. Currently, the `text` field in the Page struct contains pre-processed markdown that has already lost internal wiki links. The `wikitext` field contains the original MediaWiki markup with proper internal links.

## Problem

When viewing the Alan Turing page (viwiki/9992):

**Current `text` field** (links stripped):
```
Trong Chiến tranh thế giới thứ hai, Turing đã từng làm việc tại Bletchley Park...
```

**Original `wikitext` field** (has links):
```
Trong [[Chiến tranh thế giới thứ hai]], Turing đã từng làm việc tại [[Bletchley Park]]...
```

The `text` field is used for rendering, but it doesn't contain the `[[wiki links]]` syntax that the existing `convertWikiLinks()` function expects.

## Goals

1. **Preserve internal links** from WikiText when rendering pages
2. **Use WikiText as primary source** for content when available
3. **Fallback to text field** when WikiText is empty
4. **Render internal links as clickable links** pointing to `/page?wiki=xxx&title=PageName`

## Current Flow

```
Parquet (wikitext + text fields)
    ↓
DuckDB Store.GetByID/GetByTitle → Page struct
    ↓
handlers.page() calls view.RenderMarkdown(p.Text, p.WikiName)
    ↓
convertWikiLinks() looks for [[...]] in p.Text (but they're gone!)
    ↓
Rendered HTML (no internal links)
```

## Proposed Solution

### Option A: Use WikiText as Primary Source (Recommended)

Change the rendering to prefer `WikiText` over `Text` when available.

**Pros:**
- Simple change
- Preserves all wiki markup including links
- WikiText is the canonical source

**Cons:**
- WikiText contains MediaWiki syntax that isn't pure markdown
- Need to handle WikiText-specific markup (templates, refs, etc.)

### Option B: Parse Links from WikiText, Apply to Text

Extract `[[links]]` from WikiText and inject them into the text field.

**Pros:**
- Text field is cleaner markdown
- Only adds what's missing (links)

**Cons:**
- Complex matching between text and wikitext
- May miss links if text/wikitext diverge

## Design (Option A)

### Step 1: Add WikiText to Markdown Converter

Create `feature/view/wikitext.go`:

```go
// ConvertWikiTextToMarkdown converts MediaWiki markup to markdown.
// Handles common WikiText patterns:
// - [[Page]] → [Page](/page?wiki=xxx&title=Page)
// - [[Page|Display]] → [Display](/page?wiki=xxx&title=Page)
// - '''bold''' → **bold**
// - ''italic'' → *italic*
// - == Heading == → ## Heading
// - {{templates}} → stripped (for MVP)
// - <ref>...</ref> → stripped (for MVP)
func ConvertWikiTextToMarkdown(wikitext, wikiname string) string
```

### Step 2: Update RenderMarkdown

Modify `feature/view/markdown.go`:

```go
// RenderPage renders a page to HTML.
// Uses WikiText when available, falling back to Text.
func RenderPage(p *Page) (string, error) {
    content := p.Text
    if p.WikiText != "" {
        // Convert WikiText to markdown first
        content = ConvertWikiTextToMarkdown(p.WikiText, p.WikiName)
    }
    return RenderMarkdown(content, p.WikiName)
}
```

### Step 3: WikiText Conversion Rules

| WikiText | Markdown | Notes |
|----------|----------|-------|
| `[[Page]]` | `[Page](/page?wiki=x&title=Page)` | Internal link |
| `[[Page\|Text]]` | `[Text](/page?wiki=x&title=Page)` | Piped link |
| `'''bold'''` | `**bold**` | Bold text |
| `''italic''` | `*italic*` | Italic text |
| `== Heading ==` | `## Heading` | Level 2 heading |
| `=== Sub ===` | `### Sub` | Level 3 heading |
| `* item` | `* item` | Already markdown |
| `# item` | `1. item` | Numbered list |
| `{{template}}` | (removed) | Templates stripped |
| `<ref>...</ref>` | (removed) | References stripped |
| `[[File:...]]` | (removed) | File links stripped for MVP |
| `[[Category:...]]` | (removed) | Categories stripped |
| `{| table |}` | (removed) | Wiki tables stripped for MVP |

### Step 4: Update Handler

Modify `app/web/handlers.go`:

```go
func (h *handlers) page(c *mizu.Ctx) error {
    // ... fetch page ...

    // Use RenderPage instead of RenderMarkdown
    html, err := view.RenderPage(p)
    if err != nil {
        return err
    }
    // ...
}
```

## Implementation Plan

1. **Create `feature/view/wikitext.go`**
   - `ConvertWikiTextToMarkdown()` function
   - Regex patterns for WikiText conversion
   - Strip templates and refs for MVP

2. **Create `feature/view/wikitext_test.go`**
   - Test internal links `[[Page]]` and `[[Page|Text]]`
   - Test bold/italic conversion
   - Test heading conversion
   - Test template/ref stripping

3. **Update `feature/view/markdown.go`**
   - Add `RenderPage(*Page) (string, error)`
   - Use WikiText when available

4. **Update `app/web/handlers.go`**
   - Use `view.RenderPage(p)` instead of `view.RenderMarkdown(p.Text, ...)`

## File Changes

```
blueprints/finewiki/
├── feature/view/
│   ├── wikitext.go      (new) - WikiText to markdown converter
│   ├── wikitext_test.go (new) - Tests for converter
│   ├── markdown.go      (mod) - Add RenderPage function
│   └── api.go           (no change)
├── app/web/
│   └── handlers.go      (mod) - Use RenderPage
```

## Testing

### Unit Tests

Test with Alan Turing page (viwiki/9992) wikitext:

```go
func TestConvertWikiLinks(t *testing.T) {
    input := `Trong [[Chiến tranh thế giới thứ hai]], Turing làm việc tại [[Bletchley Park]]`
    want := `Trong [Chiến tranh thế giới thứ hai](/page?wiki=viwiki&title=...) ...`
    got := ConvertWikiTextToMarkdown(input, "viwiki")
    // assert contains expected links
}
```

### E2E Test

```bash
~/bin/finewiki serve vi &
curl "http://localhost:8080/page?id=viwiki/9992" | grep -o 'href="/page?wiki'
# Should show internal links
```

## Regex Patterns

```go
var (
    // Internal links: [[Page]] or [[Page|Display]]
    wikiLinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

    // Bold: '''text'''
    boldRe = regexp.MustCompile(`'''([^']+)'''`)

    // Italic: ''text''
    italicRe = regexp.MustCompile(`''([^']+)''`)

    // Headings: == Heading ==
    heading2Re = regexp.MustCompile(`(?m)^==\s*([^=]+)\s*==$`)
    heading3Re = regexp.MustCompile(`(?m)^===\s*([^=]+)\s*===$`)
    heading4Re = regexp.MustCompile(`(?m)^====\s*([^=]+)\s*====$`)

    // Templates: {{...}}
    templateRe = regexp.MustCompile(`\{\{[^}]+\}\}`)

    // References: <ref>...</ref>
    refRe = regexp.MustCompile(`<ref[^>]*>.*?</ref>`)
    refSelfRe = regexp.MustCompile(`<ref[^/]*/\s*>`)

    // File links: [[File:...]] or [[Image:...]]
    fileLinkRe = regexp.MustCompile(`\[\[(File|Image|Tập tin|Hình):[^\]]+\]\]`)

    // Category links: [[Category:...]]
    categoryRe = regexp.MustCompile(`\[\[(Category|Thể loại):[^\]]+\]\]`)
)
```

## Edge Cases

1. **Nested brackets**: `[[Page|[[nested]]]]` - handle outer only
2. **Escaped pipes**: `[[Page|text with \| pipe]]` - rare, ignore for MVP
3. **Section links**: `[[Page#Section]]` - strip section for MVP
4. **Interwiki links**: `[[en:Page]]` - strip for MVP
5. **Empty WikiText**: Fall back to Text field
