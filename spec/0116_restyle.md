# FineWiki Restyle - Medium-Inspired Reading Experience

## Overview

Complete redesign of FineWiki's styling inspired by Medium.com for optimal reading experience. Focus on typography, spacing, and clean visual hierarchy.

## Design Principles

1. **Reading First** - Typography optimized for long-form content
2. **Minimal Chrome** - Reduce UI noise, let content breathe
3. **High Contrast** - Clear text, distinct hierarchy
4. **Responsive** - Beautiful on all screen sizes
5. **Fast** - No external fonts, CSS-only styling

## Typography

### Font Stack

**Article Content (Serif)**:
```css
font-family: Georgia, Cambria, "Times New Roman", Times, serif;
```
- Serif fonts improve readability for long-form text
- Georgia is web-optimized and available on all systems

**UI Elements (System)**:
```css
font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
```
- System fonts for navigation, search, metadata
- Fast loading, native feel

### Font Sizes

| Element | Size | Line Height | Weight |
|---------|------|-------------|--------|
| Body text | 21px | 1.75 (36px) | 400 |
| H1 (title) | 42px | 1.25 | 700 |
| H2 | 30px | 1.35 | 700 |
| H3 | 24px | 1.4 | 600 |
| H4 | 21px | 1.5 | 600 |
| Small/meta | 14px | 1.5 | 400 |
| Caption | 15px | 1.6 | 400 |

### Paragraph Spacing

- First paragraph after heading: no top margin
- Between paragraphs: 1.5em (about 32px at 21px body)
- After headings: 0.5em
- Before headings: 2em

## Layout

### Content Width

- **Max width**: 680px (optimal reading line length ~65-75 characters)
- **Padding**: 24px mobile, 40px tablet+
- **Centered**: Always horizontally centered

### Topbar

- **Height**: 60px
- **Position**: Fixed at top
- **Background**: White with subtle bottom border
- **Logo**: Left-aligned, simple text
- **Search**: Center, minimal styling
- **Theme toggle**: Right-aligned

### Article Layout

```
┌────────────────────────────────────────────┐
│ [Logo]        [Search...]        [☀️/🌙] │
├────────────────────────────────────────────┤
│                                            │
│     ┌────────────────────────┐             │
│     │      Article Title      │             │
│     │      Meta: date • wiki  │             │
│     │                         │             │
│     │   Article content...    │             │
│     │   with proper spacing   │             │
│     │   and typography        │             │
│     │                         │             │
│     └────────────────────────┘             │
│                                            │
└────────────────────────────────────────────┘
```

## Color Palette

### Light Theme

| Variable | Value | Use |
|----------|-------|-----|
| `--bg` | `#ffffff` | Page background |
| `--fg` | `#242424` | Main text (high contrast) |
| `--fg-secondary` | `#6b6b6b` | Muted text, meta |
| `--accent` | `#1a8917` | Links (Medium green) |
| `--accent-hover` | `#0f730c` | Link hover |
| `--border` | `#e6e6e6` | Subtle borders |
| `--surface` | `#f9f9f9` | Card backgrounds |
| `--code-bg` | `#f4f4f4` | Code block background |

### Dark Theme

| Variable | Value | Use |
|----------|-------|-----|
| `--bg` | `#121212` | Page background |
| `--fg` | `#e6e6e6` | Main text |
| `--fg-secondary` | `#a0a0a0` | Muted text |
| `--accent` | `#1a8917` | Links (same green) |
| `--accent-hover` | `#2db82a` | Link hover |
| `--border` | `#333333` | Subtle borders |
| `--surface` | `#1a1a1a` | Card backgrounds |
| `--code-bg` | `#1e1e1e` | Code block background |

## Component Styling

### Links

```css
a {
    color: var(--accent);
    text-decoration: none;
}

a:hover {
    text-decoration: underline;
}

/* Internal wiki links - subtle underline */
.content a {
    text-decoration: underline;
    text-decoration-color: rgba(26, 137, 23, 0.4);
    text-underline-offset: 2px;
}

.content a:hover {
    text-decoration-color: var(--accent);
}
```

### Headings

```css
h1, h2, h3, h4, h5, h6 {
    font-family: var(--font-sans);
    font-weight: 700;
    color: var(--fg);
    letter-spacing: -0.02em;
    margin: 0;
}

h1 {
    font-size: 42px;
    line-height: 1.25;
    margin-bottom: 16px;
}

h2 {
    font-size: 30px;
    line-height: 1.35;
    margin-top: 48px;
    margin-bottom: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--border);
}
```

### Article Content

```css
.article-content {
    font-family: var(--font-serif);
    font-size: 21px;
    line-height: 1.75;
    color: var(--fg);
}

.article-content p {
    margin: 0 0 1.5em;
}

.article-content p:first-child {
    margin-top: 0;
}

/* Drop cap for first paragraph (optional) */
.article-content > p:first-of-type::first-letter {
    font-size: 3em;
    line-height: 1;
    float: left;
    margin: 0.1em 0.1em 0 0;
    font-weight: 700;
}
```

### Lists

```css
.article-content ul,
.article-content ol {
    margin: 1.5em 0;
    padding-left: 1.5em;
}

.article-content li {
    margin-bottom: 0.5em;
    padding-left: 0.5em;
}

.article-content li::marker {
    color: var(--fg-secondary);
}
```

### Blockquotes

```css
.article-content blockquote {
    margin: 2em 0;
    padding: 0 0 0 24px;
    border-left: 3px solid var(--fg);
    font-style: italic;
    color: var(--fg-secondary);
}
```

### Code

```css
.article-content code {
    font-family: var(--font-mono);
    font-size: 0.9em;
    background: var(--code-bg);
    padding: 0.15em 0.4em;
    border-radius: 3px;
}

.article-content pre {
    background: var(--code-bg);
    padding: 1.5em;
    border-radius: 4px;
    overflow-x: auto;
    font-size: 15px;
    line-height: 1.6;
}

.article-content pre code {
    background: none;
    padding: 0;
}
```

### Tables

```css
.article-content table {
    width: 100%;
    border-collapse: collapse;
    margin: 2em 0;
    font-size: 16px;
}

.article-content th,
.article-content td {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
    text-align: left;
}

.article-content th {
    font-weight: 600;
    color: var(--fg-secondary);
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
}
```

### Search

```css
.search-box {
    display: flex;
    align-items: center;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 24px;
    padding: 0 16px;
    max-width: 400px;
    margin: 0 auto;
    transition: border-color 0.2s, box-shadow 0.2s;
}

.search-box:focus-within {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(26, 137, 23, 0.1);
}

.search-input {
    flex: 1;
    border: none;
    background: transparent;
    padding: 12px 0;
    font-size: 15px;
    color: var(--fg);
    outline: none;
}
```

### Article Meta

```css
.article-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    font-family: var(--font-sans);
    font-size: 14px;
    color: var(--fg-secondary);
    margin-bottom: 32px;
}

.article-meta a {
    color: var(--fg-secondary);
}

.article-meta a:hover {
    color: var(--accent);
}
```

## Page Templates

### Home Page

- Large centered search box
- Simple tagline: "Fast Wikipedia Reader"
- Minimal, zen-like design
- Keyboard shortcut hint: "Press / to search"

### Search Results

- Clean list with title + wiki badge
- Subtle hover effect
- Direct click to page

### Article Page

- Title at top (not in topbar)
- Meta info (wiki, date) below title
- Content with proper typography
- Internal links styled subtly

## Responsive Breakpoints

| Breakpoint | Content Width | Font Size |
|------------|---------------|-----------|
| Mobile (<600px) | 100% - 48px | 18px |
| Tablet (600-900px) | 100% - 80px | 20px |
| Desktop (>900px) | 680px | 21px |

## File Changes

```
app/web/views/
├── layout/
│   └── app.html       (complete rewrite)
├── component/
│   ├── topbar.html    (simplified)
│   └── search.html    (new, for home)
└── page/
    ├── home.html      (redesigned)
    ├── search.html    (redesigned)
    └── view.html      (article styling)
```

## Implementation Order

1. **CSS Variables & Base Styles** - Set up new color palette and typography
2. **Layout Structure** - Topbar, content container
3. **Typography** - Headings, paragraphs, lists
4. **Article Content** - Links, code, tables, blockquotes
5. **Search** - Home page, results list
6. **Responsive** - Mobile/tablet adjustments
7. **Dark Mode** - Theme toggle and dark styles

## Testing

1. Test with Alan Turing page (viwiki/9992) - long content with various elements
2. Test internal links work correctly
3. Test dark mode toggle
4. Test on mobile viewport
5. Test search functionality
