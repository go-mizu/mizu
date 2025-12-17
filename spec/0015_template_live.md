# Live Template Specification

**Template**: `live`
**Status**: Draft
**Version**: 1.0

## Overview

The `live` template provides a full-stack web application with real-time interactivity powered by Mizu Live. It extends the `web` template with:

- Mizu Live engine integration for WebSocket-based real-time updates
- Interactive counter page demonstrating state management
- Chat room example showing PubSub and multi-user interaction
- Live-specific template helpers and patterns
- Development mode with hot reload support

## Directory Structure

```
myapp/
├── cmd/web/
│   └── main.go              # Application entry point
├── app/web/
│   ├── app.go               # App struct with live engine
│   ├── config.go            # Configuration loading
│   └── routes.go            # Route definitions with live pages
├── page/
│   ├── counter.go           # Counter live page
│   └── chat.go              # Chat room live page
├── assets/
│   ├── embed.go             # Embeds views and static files
│   ├── views/
│   │   ├── layouts/
│   │   │   └── default.html # Default page layout with live script
│   │   ├── pages/
│   │   │   ├── home.html    # Home page template
│   │   │   ├── counter/
│   │   │   │   └── index.html # Counter page
│   │   │   └── chat/
│   │   │       └── index.html # Chat page
│   │   ├── partials/
│   │   │   ├── header.html  # Site header
│   │   │   ├── footer.html  # Site footer
│   │   │   ├── counter/
│   │   │   │   └── count.html   # Counter region
│   │   │   └── chat/
│   │   │       ├── messages.html # Messages region
│   │   │       └── users.html    # Users region
│   │   └── components/
│   │       ├── button.html  # Button component
│   │       └── card.html    # Card component
│   └── static/
│       ├── css/
│       │   └── app.css      # Custom styles with live states
│       └── js/
│           └── app.js       # Custom JavaScript (minimal)
├── go.mod
└── README.md
```

## Design Principles

### Zero JavaScript Philosophy

The live template emphasizes server-side interactivity:
- All state management happens on the server in Go
- DOM updates are automatic via WebSocket patches
- No custom client-side JavaScript required for interactions
- Progressive enhancement: works without JS, better with it

### Live Engine Integration

The template sets up Mizu Live with sensible defaults:
- In-memory PubSub for real-time messaging
- In-memory session store (suitable for development)
- Live template helpers registered with view engine
- Development mode with verbose logging

### Example-Driven Learning

Two working examples demonstrate key patterns:
1. **Counter**: Basic state, events, regions
2. **Chat**: PubSub, multi-user, presence

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `:8080` | Server listen address |
| `DEV` | `false` | Enable development mode |

### Development Mode

When `DEV=true`:
- Templates reload on every request
- Static files served from disk
- Live pages show detailed error pages
- Verbose logging for WebSocket events

## File Specifications

### cmd/web/main.go

Entry point that:
1. Loads configuration from environment
2. Creates app instance with live engine
3. Starts HTTP server with graceful shutdown

```go
package main

import (
    "log"

    "{{.Module}}/app/web"
)

func main() {
    cfg := web.LoadConfig()
    app := web.New(cfg)

    log.Printf("listening on %s", cfg.Addr)
    if cfg.Dev {
        log.Printf("development mode enabled")
    }

    if err := app.Listen(cfg.Addr); err != nil {
        log.Fatal(err)
    }
}
```

### app/web/app.go

App struct containing:
- Mizu app instance
- View engine reference
- Live engine reference
- Configuration

Key responsibilities:
- Initialize view engine with live template funcs
- Create live engine with PubSub
- Mount live infrastructure routes
- Setup development mode options

### app/web/routes.go

Route definitions:
- `GET /` - Home page (static)
- `GET /counter` - Counter live page
- `GET /chat` - Chat room live page
- `GET /static/*` - Static file serving
- `GET /_live/*` - Live infrastructure (auto-mounted)

### page/counter.go

Counter live page demonstrating:
- `Page[T]` interface implementation
- State struct with typed fields
- Event handling for inc/dec/reset
- Region-based partial updates
- `Mark()` for targeted re-renders

```go
type CounterState struct {
    Count int
}

type CounterPage struct{}

func (p *CounterPage) Mount(ctx *live.Ctx, s *live.Session[CounterState]) error {
    s.State = CounterState{Count: 0}
    s.MarkAll()
    return nil
}

func (p *CounterPage) Render(ctx *live.Ctx, s *live.Session[CounterState]) (live.View, error) {
    return live.View{
        Page: "counter/index",
        Regions: map[string]string{
            "count": "counter/count",
        },
    }, nil
}

func (p *CounterPage) Handle(ctx *live.Ctx, s *live.Session[CounterState], e live.Event) error {
    switch e.Name {
    case "inc":
        s.State.Count++
        s.Mark("count")
    case "dec":
        s.State.Count--
        s.Mark("count")
    case "reset":
        s.State.Count = 0
        s.Mark("count")
    }
    return nil
}

func (p *CounterPage) Info(ctx *live.Ctx, s *live.Session[CounterState], msg any) error {
    return nil
}
```

### page/chat.go

Chat room live page demonstrating:
- PubSub subscription in Mount
- Multi-user messaging via `Info()`
- Real-time updates across clients
- User presence tracking
- Form submission for sending messages

```go
type ChatState struct {
    Room     string
    Username string
    Messages []Message
    Users    []string
}

type Message struct {
    From string
    Text string
    Time time.Time
}

// Event types for PubSub
type ChatMessage struct {
    From string
    Text string
    Room string
}

type UserJoined struct {
    Room     string
    Username string
}

type UserLeft struct {
    Room     string
    Username string
}
```

### views/layouts/default.html

Default layout with live script inclusion:
- Standard HTML5 structure
- Tailwind CSS CDN
- shadcn-inspired design tokens
- Slot for title, content, scripts
- Live runtime script automatically injected

### views/pages/counter/index.html

Counter page template:
- Live root element with `data-lv`
- Event bindings via `data-lv-click`
- Region wrapper with matching ID
- Loading state indicators

```html
{{define "title"}}Counter{{end}}

{{define "content"}}
<div id="lv-root" data-lv class="max-w-md mx-auto">
    {{component "card" (dict "Title" "Live Counter")}}
        <div id="count" class="text-center">
            {{partial "counter/count" .}}
        </div>
        <div class="flex justify-center gap-4 mt-6">
            <button {{lvClick "dec"}} {{lvLoading "opacity-50"}}
                class="px-4 py-2 rounded-md bg-secondary text-secondary-foreground">
                -
            </button>
            <button {{lvClick "reset"}} {{lvLoading "opacity-50"}}
                class="px-4 py-2 rounded-md bg-muted text-muted-foreground">
                Reset
            </button>
            <button {{lvClick "inc"}} {{lvLoading "opacity-50"}}
                class="px-4 py-2 rounded-md bg-primary text-primary-foreground">
                +
            </button>
        </div>
    {{end}}
</div>
{{end}}
```

### views/partials/counter/count.html

Counter region partial:
- Renders just the count display
- Re-rendered independently on updates

```html
<p class="text-6xl font-bold tabular-nums">
    {{.State.Count}}
</p>
```

### views/pages/chat/index.html

Chat page template:
- Message list region
- User list region
- Message form with live submit
- Connection status indicator

### views/partials/chat/messages.html

Messages region partial:
- Scrollable message list
- Timestamps and usernames
- Auto-scroll on new messages

### views/partials/chat/users.html

Users region partial:
- Online user list
- Join/leave indicators

### static/css/app.css

Custom CSS extending base web template:
- Live loading state utilities
- Chat-specific styles
- Animation for new messages

```css
/* Live loading states */
[data-lv-loading] {
    transition: opacity 150ms ease-out;
}

[data-lv-loading].loading {
    opacity: 0.5;
    pointer-events: none;
}

/* Message animations */
@keyframes slide-in {
    from {
        opacity: 0;
        transform: translateY(10px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.message-enter {
    animation: slide-in 200ms ease-out;
}
```

### static/js/app.js

Minimal JavaScript for:
- Optional enhancements (e.g., sound on new message)
- No live functionality (handled by runtime)

## Template Helpers

The live template registers these helpers with the view engine:

| Helper | Usage | Output |
|--------|-------|--------|
| `lvClick` | `{{lvClick "save"}}` | `data-lv-click="save"` |
| `lvClick` | `{{lvClick "delete" (lvVal "id" .ID)}}` | `data-lv-click="delete" data-lv-value-id="123"` |
| `lvSubmit` | `{{lvSubmit "create"}}` | `data-lv-submit="create"` |
| `lvChange` | `{{lvChange "search"}}` | `data-lv-change="search"` |
| `lvKeydown` | `{{lvKeydown "submit" "Enter"}}` | `data-lv-keydown="submit" data-lv-key="Enter"` |
| `lvVal` | `{{lvVal "id" 123}}` | `map[string]any{"id": 123}` |
| `lvDebounce` | `{{lvDebounce 300}}` | `data-lv-debounce="300"` |
| `lvThrottle` | `{{lvThrottle 500}}` | `data-lv-throttle="500"` |
| `lvLoading` | `{{lvLoading "opacity-50"}}` | `data-lv-loading-class="opacity-50"` |
| `lvTarget` | `{{lvTarget "modal"}}` | `data-lv-target="modal"` |

## Usage

### Create New Project

```bash
mizu new myapp --template live
cd myapp
```

### Run Development Server

```bash
DEV=true go run ./cmd/web
```

Open http://localhost:8080 to see:
- Home page with navigation
- `/counter` - Interactive counter
- `/chat` - Multi-user chat room

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

After running `mizu new myapp --template live --module github.com/user/myapp`:

```
myapp/
├── cmd/web/main.go
├── app/web/
│   ├── app.go
│   ├── config.go
│   └── routes.go
├── page/
│   ├── counter.go
│   └── chat.go
├── assets/
│   ├── embed.go
│   ├── views/...
│   └── static/...
├── go.mod
└── README.md
```

## Key Patterns Demonstrated

### 1. State Management

```go
// Define typed state
type State struct {
    Count int
    Items []Item
}

// Modify state in handlers
s.State.Count++
s.State.Items = append(s.State.Items, item)
```

### 2. Event Handling

```go
func (p *Page) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    switch e.Name {
    case "save":
        // Handle click event
    case "submit":
        // Handle form submit
        title := e.Form.Get("title")
    }
    return nil
}
```

### 3. Region Updates

```go
// Mark specific regions dirty
s.Mark("count")           // Single region
s.Mark("items", "stats")  // Multiple regions
s.MarkAll()               // All regions
```

### 4. Real-time Messaging

```go
// Subscribe in Mount
ctx.Subscribe("room:" + roomID)

// Handle messages in Info
func (p *Page) Info(ctx *live.Ctx, s *live.Session[State], msg any) error {
    switch m := msg.(type) {
    case ChatMessage:
        s.State.Messages = append(s.State.Messages, m)
        s.Mark("messages")
    }
    return nil
}

// Publish from handlers
lv.PubSub().Publish("room:general", ChatMessage{...})
```

### 5. Client Commands

```go
// Redirect after action
s.Push(live.Redirect{To: "/dashboard"})

// Focus element
s.Push(live.Focus{Selector: "#input"})

// Scroll to element
s.Push(live.Scroll{Selector: "#bottom"})
```

## Developer Experience Goals

1. **Instant Feedback**: Changes appear immediately without page reload
2. **Type Safety**: Compile-time checks for state and events
3. **Clear Mental Model**: Server owns state, client shows UI
4. **Debugging**: Dev mode shows event flow and patches
5. **Progressive Enhancement**: Works without JS, better with it

## Implementation Notes

1. **Embed Strategy**: Views and static files embedded in `assets` package
2. **Live Script Injection**: Runtime automatically injected before `</body>`
3. **Region Matching**: Region IDs must match between Go and HTML
4. **Template Escaping**: Use `{{ "{{" }}` for view engine delimiters in CLI templates
5. **Error Handling**: Errors in handlers shown via flash messages or OnError callback
