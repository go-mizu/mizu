# Web Template Specification

**Template**: `web`
**Status**: Draft
**Version**: 1.0

## Overview

The `web` template provides a full-stack web application starter with server-rendered HTML pages. It demonstrates the Mizu view package with:

- Layouts, pages, components, and partials
- Static file embedding with `embed.FS`
- Tailwind CSS via CDN for modern styling
- Clean, shadcn-inspired component design
- Development mode with hot reload support
- Production-ready with embedded templates

## Directory Structure

```
myapp/
├── cmd/web/
│   └── main.go              # Application entry point
├── app/web/
│   ├── app.go               # App struct with server lifecycle
│   ├── config.go            # Configuration loading
│   └── routes.go            # Route definitions
├── handler/
│   ├── home.go              # Home page handler
│   └── about.go             # About page handler
├── assets/
│   ├── embed.go             # Embeds views and static files
│   ├── views/
│   │   ├── layouts/
│   │   │   └── default.html # Default page layout
│   │   ├── pages/
│   │   │   ├── home.html    # Home page template
│   │   │   └── about.html   # About page template
│   │   ├── components/
│   │   │   ├── button.html  # Button component
│   │   │   └── card.html    # Card component
│   │   └── partials/
│   │       ├── header.html  # Site header
│   │       └── footer.html  # Site footer
│   └── static/
│       ├── css/
│       │   └── app.css      # Custom styles
│       └── js/
│           └── app.js       # Custom JavaScript
├── go.mod
└── README.md
```

## Design Principles

### Tailwind CSS Integration

Uses Tailwind CSS via CDN for zero-build styling. The template includes:
- Tailwind CSS Play CDN for rapid development
- Custom CSS file for app-specific overrides
- shadcn-inspired component styling patterns

### Component Library

Pre-built components following shadcn/ui patterns:

**Button Component**
```html
{{component "button" (dict "Label" "Click me" "Variant" "primary")}}
{{component "button" (dict "Label" "Cancel" "Variant" "outline")}}
```

Variants: `primary`, `secondary`, `outline`, `ghost`, `destructive`

**Card Component**
```html
{{component "card" (dict "Title" "Card Title")}}
    <p>Card content goes here</p>
{{end}}
```

### Layout System

The default layout provides:
- Responsive container with max-width
- Header with navigation
- Main content area with slot
- Footer
- Stack injection points for page-specific CSS/JS

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{slot "title" "My App"}}</title>
    {{stack "styles"}}
</head>
<body>
    {{partial "header"}}
    <main>{{slot "content"}}</main>
    {{partial "footer"}}
    {{stack "scripts"}}
</body>
</html>
```

### Static File Handling

Static files are embedded and served at `/static/`:
- `/static/css/app.css` - Custom styles
- `/static/js/app.js` - Custom JavaScript

In development mode, files are read from disk for hot reload.

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `:8080` | Server listen address |
| `DEV` | `false` | Enable development mode |

### Development Mode

When `DEV=true`:
- Templates reload on every request
- Static files served from disk (not embed)
- Detailed error pages with source context

### Production Mode

When `DEV=false`:
- All templates preloaded and cached
- Static files embedded in binary
- Optimized for performance

## File Specifications

### cmd/web/main.go

Entry point that:
1. Loads configuration from environment
2. Creates app instance
3. Starts HTTP server with graceful shutdown

### app/web/app.go

App struct containing:
- Mizu app instance
- View engine reference
- Configuration

### app/web/routes.go

Route definitions:
- `GET /` - Home page
- `GET /about` - About page
- `GET /static/*` - Static file serving

### handler/home.go

Home page handler demonstrating:
- Basic page rendering
- Passing data to templates

### handler/about.go

About page handler demonstrating:
- Simple static page rendering

### views/layouts/default.html

Default layout with:
- HTML5 doctype and structure
- Tailwind CSS CDN inclusion
- Meta viewport for responsive design
- Slot for title, head, content
- Stack for styles and scripts
- Header and footer partials

### views/pages/home.html

Home page demonstrating:
- Slot filling (title, content)
- Component usage (card, button)
- Data interpolation

### views/pages/about.html

About page demonstrating:
- Simple content page
- Layout inheritance

### views/components/button.html

Button component with:
- Multiple variants (primary, secondary, outline, ghost, destructive)
- Size options (sm, md, lg)
- Disabled state support

### views/components/card.html

Card component with:
- Title slot
- Children content support
- Optional footer

### views/partials/header.html

Site header with:
- Logo/brand
- Navigation links
- Responsive design

### views/partials/footer.html

Site footer with:
- Copyright notice
- Links

### static/css/app.css

Custom CSS for:
- CSS custom properties for theming
- Component refinements
- Animation utilities

### static/js/app.js

Minimal JavaScript for:
- Alpine.js-style interactivity (optional)
- Progressive enhancement

### assets/embed.go

Go file with embed directives in the `assets` package:
```go
package assets

import "embed"

//go:embed views
var ViewsFS embed.FS

//go:embed static
var StaticFS embed.FS
```

## Usage

### Create New Project

```bash
mizu new myapp --template web
cd myapp
```

### Run Development Server

```bash
DEV=true go run ./cmd/web
```

### Build for Production

```bash
go build -o myapp ./cmd/web
./myapp
```

## Template Variables

| Variable | Description |
|----------|-------------|
| `{{.Name}}` | Project name |
| `{{.Module}}` | Go module path |
| `{{.Year}}` | Current year |

## Example Output

After running `mizu new myapp --template web --module github.com/user/myapp`:

```
myapp/
├── cmd/web/main.go
├── app/web/
│   ├── app.go
│   ├── config.go
│   └── routes.go
├── handler/
│   ├── home.go
│   └── about.go
├── assets/
│   ├── embed.go
│   ├── views/...
│   └── static/...
├── go.mod
└── README.md
```

## Styling Guide

### Color Palette

Uses CSS custom properties for theming:

```css
:root {
    --background: 0 0% 100%;
    --foreground: 222.2 84% 4.9%;
    --primary: 222.2 47.4% 11.2%;
    --primary-foreground: 210 40% 98%;
    --secondary: 210 40% 96.1%;
    --secondary-foreground: 222.2 47.4% 11.2%;
    --muted: 210 40% 96.1%;
    --muted-foreground: 215.4 16.3% 46.9%;
    --accent: 210 40% 96.1%;
    --accent-foreground: 222.2 47.4% 11.2%;
    --destructive: 0 84.2% 60.2%;
    --destructive-foreground: 210 40% 98%;
    --border: 214.3 31.8% 91.4%;
    --ring: 222.2 84% 4.9%;
    --radius: 0.5rem;
}
```

### Button Variants

| Variant | Description |
|---------|-------------|
| `primary` | Solid background, primary color |
| `secondary` | Solid background, secondary color |
| `outline` | Border only, transparent background |
| `ghost` | No border, hover background |
| `destructive` | Red/danger styling |

### Responsive Breakpoints

Uses Tailwind's default breakpoints:
- `sm`: 640px
- `md`: 768px
- `lg`: 1024px
- `xl`: 1280px
- `2xl`: 1536px

## Implementation Notes

1. **Embed Strategy**: Both `views/` and `static/` are embedded in the `assets` package. The view engine uses `ViewsFS` directly while static files are served via file server.

2. **Development Detection**: Use environment variable `DEV` rather than build tags for easier local development. In dev mode, files are read from disk at `assets/views/` and `assets/static/`.

3. **Static Prefix**: Static files are served at `/static/` to avoid conflicts with page routes.

4. **Tailwind CDN**: Uses the Play CDN for simplicity. Production apps should consider a build step with the Tailwind CLI.

5. **No JavaScript Framework**: Vanilla JavaScript with optional progressive enhancement. Easy to add HTMX, Alpine.js, or others as needed.

6. **Template Escaping**: View templates use `{{ "{{" }}` escaping in the CLI template system to preserve the view engine's `{{` delimiters.
