# Forum Theming System

## Overview

Implement a theming system for the forum blueprint that allows switching between different visual styles. Initial themes:

- **default** - Current modern design (Reddit-inspired with modern touches)
- **old** - Old Reddit style (classic old.reddit.com look)
- **hn** - Hacker News style (minimalist text-focused design)

## Architecture

### Directory Structure

```
assets/
├── static/
│   └── css/
│       ├── app.css           # Default theme CSS
│       ├── old.css           # Old Reddit theme CSS
│       └── hn.css            # Hacker News theme CSS
└── views/
    ├── default/              # Default theme templates
    │   ├── layouts/
    │   │   └── default.html
    │   ├── components/
    │   │   ├── nav.html
    │   │   ├── comment.html
    │   │   └── thread_card.html
    │   └── pages/
    │       └── *.html
    ├── old/                  # Old Reddit theme (inherits from default)
    │   ├── layouts/
    │   │   └── default.html  # Override layout for theme CSS
    │   └── components/
    │       ├── nav.html      # Classic reddit nav
    │       └── thread_card.html
    └── hn/                   # Hacker News theme (inherits from default)
        ├── layouts/
        │   └── default.html  # Override layout for theme CSS
        └── components/
            ├── nav.html      # Simple HN nav
            └── thread_card.html
```

### Theme Inheritance

Themes inherit from `default` theme. When loading templates:

1. Load all templates from `views/default/` as base
2. For non-default themes, overlay templates from `views/{theme}/`
3. Any file in the theme directory overrides the corresponding default file

### Implementation Changes

#### 1. `embed.go` Changes

```go
// TemplatesForTheme loads templates for a specific theme with inheritance from default.
func TemplatesForTheme(theme string) (map[string]*template.Template, error) {
    if theme == "" || theme == "default" {
        return Templates() // Use existing function
    }

    // Load default templates first, then overlay theme-specific ones
    // ...
}
```

#### 2. `ServerConfig` Changes

```go
type ServerConfig struct {
    Addr  string
    Dev   bool
    Theme string // "default", "old", "hn"
}
```

#### 3. `PageData` Changes

```go
type PageData struct {
    Title       string
    CurrentUser *accounts.Account
    UnreadCount int64
    Theme       string // Pass theme to templates for CSS selection
    Data        any
}
```

#### 4. Layout Template Changes

Each theme's `default.html` layout references its own CSS:

```html
<!-- default theme -->
<link rel="stylesheet" href="/static/css/app.css">

<!-- old theme -->
<link rel="stylesheet" href="/static/css/old.css">

<!-- hn theme -->
<link rel="stylesheet" href="/static/css/hn.css">
```

## Theme Specifications

### Old Reddit Theme (`old`)

Visual characteristics matching old.reddit.com:

**Colors:**
- Background: `#ffffff` (pure white)
- Secondary bg: `#f5f5f5`
- Text: `#1a1a1b`
- Links: `#0079d3` (Reddit blue)
- Upvote: `#ff4500` (Reddit orange)
- Downvote: `#7193ff` (periwinkle blue)
- Border: `#ccc`

**Typography:**
- Font: Verdana, Arial, sans-serif
- Size: 10px-14px (smaller than modern)

**Layout:**
- No rounded corners or minimal (2px)
- No shadows
- Compact spacing
- Vote arrows (▲ ▼) instead of filled buttons
- Score displayed between arrows
- Thread cards as simple rows
- Classic tabbed navigation

**Nav:**
- Horizontal tabs: hot | new | top | rising
- Classic Reddit logo styling
- User menu in top right

### Hacker News Theme (`hn`)

Visual characteristics matching news.ycombinator.com:

**Colors:**
- Background: `#f6f6ef` (warm off-white)
- Header: `#ff6600` (HN orange)
- Text: `#000000` (pure black)
- Links: `#000000` (black, underlined on hover)
- Comment links: `#828282` (gray)
- Visited: `#828282`

**Typography:**
- Font: Verdana, Geneva, sans-serif
- Size: 10pt (fixed, not rem)
- Line height: ~1.2

**Layout:**
- No rounded corners
- No shadows
- No cards - just text
- Extremely minimal padding
- Dense information display
- Simple table-based layout feel

**Nav:**
- Single orange header bar
- "Y" logo in corner
- Links: new | past | comments | ask | show | jobs | submit
- Login link far right

**Thread listing:**
- Numbered list (1. 2. 3.)
- Simple upvote triangle
- Title as plain link
- Domain in parentheses (example.com)
- "X points by user T hours ago | hide | N comments"

## Implementation Steps

### Phase 1: Core Infrastructure

1. **Modify `embed.go`:**
   - Add `TemplatesForTheme(theme string)` function
   - Implement theme inheritance (overlay theme files over default)
   - Keep `Templates()` for backward compatibility

2. **Update `ServerConfig`:**
   - Add `Theme` field

3. **Update `PageData`:**
   - Add `Theme` field for template access

4. **Update `server.go`:**
   - Pass theme to template loading
   - Pass theme to page rendering

### Phase 2: Old Reddit Theme

1. **Create `assets/static/css/old.css`:**
   - Classic Reddit color scheme
   - Compact typography
   - Minimal styling (no shadows, minimal radius)
   - Classic vote arrows

2. **Create `assets/views/old/layouts/default.html`:**
   - Reference old.css
   - Same structure as default

3. **Create `assets/views/old/components/nav.html`:**
   - Classic Reddit navigation
   - Tab-based sorting
   - User dropdown

4. **Create `assets/views/old/components/thread_card.html`:**
   - Compact row-based design
   - Vote arrows with score
   - Thumbnail support

### Phase 3: Hacker News Theme

1. **Create `assets/static/css/hn.css`:**
   - Minimalist design
   - Orange header
   - No cards, pure text
   - Dense layout

2. **Create `assets/views/hn/layouts/default.html`:**
   - Reference hn.css
   - Simpler structure

3. **Create `assets/views/hn/components/nav.html`:**
   - Simple orange header bar
   - Text links only

4. **Create `assets/views/hn/components/thread_card.html`:**
   - Numbered list item
   - Plain text design
   - HN-style metadata line

### Phase 4: Testing & Polish

1. Test all themes load correctly
2. Test all pages render in each theme
3. Verify JavaScript interactions work
4. Test responsive behavior

## API

No API changes needed. Theme is set at server startup via configuration.

Future enhancement: Allow users to select theme preference (stored in settings).

## CLI Usage

```bash
# Start with default theme
./forum serve

# Start with old reddit theme
./forum serve --theme old

# Start with hacker news theme
./forum serve --theme hn
```

## File Changes Summary

### New Files
- `assets/static/css/old.css`
- `assets/static/css/hn.css`
- `assets/views/old/layouts/default.html`
- `assets/views/old/components/nav.html`
- `assets/views/old/components/thread_card.html`
- `assets/views/hn/layouts/default.html`
- `assets/views/hn/components/nav.html`
- `assets/views/hn/components/thread_card.html`

### Modified Files
- `assets/embed.go` - Add theme-aware template loading
- `app/web/server.go` - Add theme config
- `app/web/handler/page.go` - Pass theme to templates
- `cmd/serve.go` - Add --theme flag
