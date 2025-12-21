# 0118: Random Page Feature

## Summary

Add a "random page" feature to FineWiki that allows users to discover random wiki articles. This includes:
1. A `/random` endpoint that redirects to a random page
2. A random button in the navbar (before the theme toggle)
3. Enhanced navbar styling with improved logo and borderless modern design

## Motivation

Users often want to explore content serendipitously. Wikipedia's "Random article" is one of its most beloved features. Adding this to FineWiki enhances the discovery experience.

## Design

### 1. Navbar Enhancements

#### Logo Redesign
- Add a subtle book/wiki icon before the "FineWiki" text
- Use a small SVG icon that represents knowledge/wiki

#### Remove Border Bottom
- Remove `border-bottom: 1px solid var(--border)` from `.topbar`
- Add subtle shadow or backdrop blur for modern floating effect

#### Random Button
- Add a dice/shuffle icon button before the theme toggle
- Clicking redirects to `/random`
- Tooltip: "Random article"

### 2. Random Page Endpoint

#### Route: `GET /random`

**Behavior:**
1. Query database for a random page ID from the `titles` table
2. Redirect (302) to `/page?id={random_id}`

**SQL Query:**
```sql
SELECT id FROM titles
ORDER BY RANDOM()
LIMIT 1
```

### 3. Implementation Layers

#### Store Layer (`store/duckdb/store.go`)
Add method:
```go
// GetRandomID returns a random page ID from the database.
func (s *Store) GetRandomID(ctx context.Context) (string, error)
```

#### View API (`feature/view/api.go`)
Extend the API interface:
```go
type Store interface {
    GetByID(ctx context.Context, id string) (*Page, error)
    GetByTitle(ctx context.Context, wikiname, title string) (*Page, error)
    GetRandomID(ctx context.Context) (string, error) // new
}

type API interface {
    ByID(ctx context.Context, id string) (*Page, error)
    ByTitle(ctx context.Context, wikiname, title string) (*Page, error)
    RandomID(ctx context.Context) (string, error) // new
}
```

#### Service Layer (`feature/view/service.go`)
```go
func (s *Service) RandomID(ctx context.Context) (string, error) {
    if s.store == nil {
        return "", errors.New("view: nil store")
    }
    return s.store.GetRandomID(ctx)
}
```

#### Web Handler (`app/web/handlers.go`)
```go
func (s *Server) randomPage(c *mizu.Ctx) error {
    ctx := c.Request().Context()
    id, err := s.view.RandomID(ctx)
    if err != nil {
        return c.Text(500, "No pages available")
    }
    return c.Redirect(302, fmt.Sprintf("/page?id=%s", id))
}
```

#### Routes (`app/web/server.go`)
```go
r.Get("/random", s.randomPage)
```

### 4. UI Changes

#### topbar.html
```html
<div class="topbar-actions">
    <a href="/random" class="random-btn" title="Random article">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M16 3h5v5M4 20L21 3M21 16v5h-5M15 15l6 6M4 4l5 5"/>
        </svg>
    </a>
    <button class="theme-toggle" data-theme-toggle aria-label="Toggle theme">
        ...
    </button>
</div>
```

#### CSS (layout/app.html)
```css
/* Random Button */
.random-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 8px;
    border-radius: 50%;
    color: var(--fg-secondary);
    transition: background 0.2s, color 0.2s;
}

.random-btn:hover {
    background: var(--surface);
    color: var(--fg);
    text-decoration: none;
}

.random-btn svg {
    width: 20px;
    height: 20px;
}
```

## File Changes

1. `spec/0118_random_page.md` - This spec
2. `store/duckdb/store.go` - Add `GetRandomID` method
3. `feature/view/api.go` - Extend `Store` and `API` interfaces
4. `feature/view/service.go` - Add `RandomID` method
5. `app/web/handlers.go` - Add `randomPage` handler
6. `app/web/server.go` - Add `/random` route
7. `app/web/views/component/topbar.html` - Add random button, enhance logo
8. `app/web/views/layout/app.html` - Remove border, add random button styles

## Testing

- Manual: Navigate to `/random`, verify redirect to random page
- Verify random button appears in navbar and works
- Verify navbar styling (no border, improved logo)
