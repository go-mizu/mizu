# Live Developer Experience Specification

**Package**: `view/live`
**Status**: Draft
**Version**: 1.0

## Overview

This document specifies the developer experience for building real-time interactive applications with Mizu Live. It covers template conventions, event binding, state management patterns, and common use cases from the developer's perspective.

## Design Goals

1. **Familiar to Go developers**: Standard interfaces, structs, methods. No magic metaprogramming.
2. **Minimal JavaScript**: Zero custom JS required for most interactions.
3. **Progressive enhancement**: Works without JS, upgrades to real-time with WebSocket.
4. **Type safety**: Generics ensure state is always typed.
5. **Clear mental model**: Server owns state, events flow up, patches flow down.

## Quick Start

### Hello World: Counter

**1. Define state**

```go
type CounterState struct {
    Count int
}
```

**2. Implement Page interface**

```go
type CounterPage struct{}

func (p *CounterPage) Mount(ctx *live.Ctx, s *live.Session[CounterState]) error {
    s.State = CounterState{Count: 0}
    s.MarkAll()
    return nil
}

func (p *CounterPage) Render(ctx *live.Ctx, s *live.Session[CounterState]) (live.View, error) {
    return live.View{Page: "counter/index"}, nil
}

func (p *CounterPage) Handle(ctx *live.Ctx, s *live.Session[CounterState], e live.Event) error {
    switch e.Name {
    case "inc":
        s.State.Count++
    case "dec":
        s.State.Count--
    }
    s.MarkAll()
    return nil
}

func (p *CounterPage) Info(ctx *live.Ctx, s *live.Session[CounterState], msg any) error {
    return nil
}
```

**3. Create template**

```html
<!-- views/pages/counter/index.html -->
{{define "content"}}
<div id="lv-root" data-lv>
    <h1>Counter: {{.Data.State.Count}}</h1>
    <button data-lv-click="dec">-</button>
    <button data-lv-click="inc">+</button>
</div>
{{end}}
```

**4. Mount the route**

```go
app.GET("/counter", lv.Page("/counter", &CounterPage{}))
```

That's it. Click the buttons, counter updates instantly.

## Template Conventions

### The Live Root

Every live page needs a root element with `data-lv`:

```html
<div id="lv-root" data-lv>
    <!-- Live content here -->
</div>
```

The `data-lv` attribute:
- Tells the client runtime where to apply patches
- Must wrap all interactive content
- Can be on any element (div, main, section, etc.)

### Template Data Structure

Templates receive this data:

```go
type TemplateData struct {
    Page    PageMeta        // Page metadata
    Data    struct {
        State T             // Your typed state
        Flash live.Flash    // Flash messages
    }
    Request RequestData     // Request info
    CSRF    string          // CSRF token
}
```

Access state in templates:

```html
<p>Count: {{.Data.State.Count}}</p>
<p>User: {{.Data.State.User.Name}}</p>
{{range .Data.State.Items}}
    <li>{{.Name}}</li>
{{end}}
```

## Event Binding

### Click Events

```html
<!-- Simple click -->
<button data-lv-click="save">Save</button>

<!-- Click with values -->
<button data-lv-click="delete" data-lv-value-id="123">Delete</button>

<!-- Multiple values -->
<button data-lv-click="move"
        data-lv-value-from="inbox"
        data-lv-value-to="archive">
    Archive
</button>
```

Handle in Go:

```go
func (p *MyPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    switch e.Name {
    case "save":
        // No values needed
    case "delete":
        id := e.Get("id") // "123"
    case "move":
        from := e.Get("from") // "inbox"
        to := e.Get("to")     // "archive"
    }
    return nil
}
```

### Form Submit

```html
<form data-lv-submit="create">
    <input name="title" type="text">
    <input name="description" type="text">
    <button type="submit">Create</button>
</form>
```

Handle in Go:

```go
case "create":
    title := e.Form.Get("title")
    desc := e.Form.Get("description")

    // Validate
    if title == "" {
        s.State.Error = "Title required"
        s.Mark("error")
        return nil
    }

    // Save
    item := Item{Title: title, Description: desc}
    s.State.Items = append(s.State.Items, item)
    s.Mark("items")
```

### Change Events

```html
<!-- Input change (debounced by default) -->
<input name="search"
       data-lv-change="search"
       data-lv-debounce="300">

<!-- Select change -->
<select data-lv-change="filter" name="status">
    <option value="all">All</option>
    <option value="active">Active</option>
    <option value="done">Done</option>
</select>

<!-- Checkbox change -->
<input type="checkbox"
       data-lv-change="toggle"
       data-lv-value-id="{{.ID}}"
       {{if .Done}}checked{{end}}>
```

### Key Events

```html
<!-- Keydown -->
<input data-lv-keydown="hotkey" data-lv-key="Enter">

<!-- Keyup -->
<input data-lv-keyup="release" data-lv-key="Escape">

<!-- Any key -->
<div data-lv-keydown="navigate" tabindex="0">
    <!-- Arrow key navigation -->
</div>
```

Handle key events:

```go
case "hotkey":
    if e.Key == "Enter" {
        // Submit
    }
case "navigate":
    switch e.Key {
    case "ArrowUp":
        s.State.Selected--
    case "ArrowDown":
        s.State.Selected++
    }
```

### Focus and Blur

```html
<input data-lv-focus="activate" data-lv-blur="deactivate">
```

### Mouse Events

```html
<div data-lv-mouseenter="preview" data-lv-mouseleave="hide">
    Hover me
</div>
```

## Event Modifiers

### Debounce

Delay event until user stops typing:

```html
<!-- Wait 300ms after last keystroke -->
<input data-lv-change="search" data-lv-debounce="300">

<!-- Default is 150ms for change events -->
<input data-lv-change="search" data-lv-debounce>
```

### Throttle

Limit event frequency:

```html
<!-- At most once per 500ms -->
<input data-lv-change="track" data-lv-throttle="500">
```

### Prevent Default

```html
<!-- Prevent form submission reload -->
<form data-lv-submit="save" data-lv-prevent>

<!-- Prevent link navigation -->
<a href="/old" data-lv-click="navigate" data-lv-prevent>Click</a>
```

### Stop Propagation

```html
<div data-lv-click="outer">
    <button data-lv-click="inner" data-lv-stop>
        Won't trigger outer
    </button>
</div>
```

### Capture Phase

```html
<div data-lv-click.capture="intercept">
    Captures before children
</div>
```

## Regions: Efficient Updates

Instead of re-rendering the entire page, define regions that update independently.

### Define Regions in Render

```go
func (p *DashboardPage) Render(ctx *live.Ctx, s *live.Session[State]) (live.View, error) {
    return live.View{
        Page: "dashboard/index",
        Regions: map[string]string{
            "stats":    "dashboard/stats",    // Partial template
            "activity": "dashboard/activity",
            "alerts":   "dashboard/alerts",
        },
    }, nil
}
```

### Mark Regions in Template

```html
<!-- views/pages/dashboard/index.html -->
{{define "content"}}
<div id="lv-root" data-lv>
    <header>Dashboard</header>

    <section id="stats">
        {{partial "dashboard/stats" .}}
    </section>

    <section id="activity">
        {{partial "dashboard/activity" .}}
    </section>

    <aside id="alerts">
        {{partial "dashboard/alerts" .}}
    </aside>
</div>
{{end}}
```

### Mark Dirty Regions

```go
func (p *DashboardPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    switch e.Name {
    case "refresh-stats":
        s.State.Stats = fetchStats()
        s.Mark("stats") // Only re-render stats section

    case "dismiss-alert":
        id := e.GetInt("id")
        s.State.Alerts = removeAlert(s.State.Alerts, id)
        s.Mark("alerts") // Only re-render alerts section

    case "new-activity":
        s.State.Activity = append(s.State.Activity, newItem)
        s.Mark("activity", "stats") // Re-render both
    }
    return nil
}
```

When regions are marked dirty:
1. Server re-renders only those partial templates
2. Server generates HTML patches
3. Client applies patches to matching DOM IDs
4. Unchanged regions are untouched (no flicker)

## Loading States

### Show Loading Indicator

```html
<button data-lv-click="save" data-lv-loading-class="opacity-50">
    <span data-lv-loading-hide>Save</span>
    <span data-lv-loading-show class="hidden">Saving...</span>
</button>
```

Attributes:
- `data-lv-loading-class`: Add class while processing
- `data-lv-loading-show`: Show element while processing
- `data-lv-loading-hide`: Hide element while processing
- `data-lv-loading-disable`: Disable element while processing

### Target Specific Regions

```html
<button data-lv-click="save" data-lv-loading-target="#form-section">
    Save
</button>

<div id="form-section">
    <!-- Shows loading state only for this section -->
</div>
```

## Client Commands (Push)

Push commands from server to client:

### Redirect

```go
func (p *AuthPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "login" {
        if authenticate(e.Form) {
            s.Push(live.Redirect{To: "/dashboard"})
        }
    }
    return nil
}
```

### Focus Element

```go
// Focus the error field
s.Push(live.Focus{Selector: "#email-input"})
```

### Scroll To

```go
// Scroll to element
s.Push(live.Scroll{Selector: "#new-message", Block: "end"})

// Scroll to top
s.Push(live.Scroll{})
```

### Download File

```go
s.Push(live.Download{
    URL:      "/api/export?format=csv",
    Filename: "report.csv",
})
```

### Execute JavaScript

For escape hatches when needed:

```go
s.Push(live.JS{
    Code: "window.confetti && confetti()",
})
```

## Server Push with PubSub

### Real-time Updates

```go
// Subscribe in Mount
func (p *ChatPage) Mount(ctx *live.Ctx, s *live.Session[ChatState]) error {
    roomID := ctx.Param("room")
    s.State.RoomID = roomID
    s.State.Messages = loadMessages(roomID)

    ctx.Subscribe("room:" + roomID)
    return nil
}

// Handle incoming messages
func (p *ChatPage) Info(ctx *live.Ctx, s *live.Session[ChatState], msg any) error {
    switch m := msg.(type) {
    case ChatMessage:
        s.State.Messages = append(s.State.Messages, m)
        s.Mark("messages")
        s.Push(live.Scroll{Selector: "#messages", Block: "end"})
    }
    return nil
}

// Publish from anywhere
lv.PubSub().Publish("room:general", ChatMessage{
    From: "alice",
    Text: "Hello everyone!",
    Time: time.Now(),
})
```

### Presence Tracking

```go
type Presence struct {
    Users map[string]UserInfo
}

func (p *ChatPage) Mount(ctx *live.Ctx, s *live.Session[ChatState]) error {
    user := auth.User(ctx.Ctx)

    // Subscribe to room and presence
    ctx.Subscribe(
        "room:"+s.State.RoomID,
        "presence:"+s.State.RoomID,
    )

    // Announce join
    lv.PubSub().Publish("presence:"+s.State.RoomID, UserJoined{
        User: user,
    })

    return nil
}
```

### Broadcast to All Sessions

```go
// System-wide notification
lv.PubSub().Broadcast(SystemNotice{
    Message: "Server restarting in 5 minutes",
})
```

## Timers and Intervals

### One-time Timer

```go
func (p *AlertPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "show-message" {
        s.State.Message = "Success!"
        s.Mark("message")

        // Clear after 3 seconds
        ctx.SendAfter(ClearMessage{}, 3*time.Second)
    }
    return nil
}

func (p *AlertPage) Info(ctx *live.Ctx, s *live.Session[State], msg any) error {
    if _, ok := msg.(ClearMessage); ok {
        s.State.Message = ""
        s.Mark("message")
    }
    return nil
}
```

### Recurring Updates

```go
type Tick struct{}

func (p *ClockPage) Mount(ctx *live.Ctx, s *live.Session[State]) error {
    s.State.Time = time.Now()
    ctx.SendAfter(Tick{}, time.Second)
    return nil
}

func (p *ClockPage) Info(ctx *live.Ctx, s *live.Session[State], msg any) error {
    if _, ok := msg.(Tick); ok {
        s.State.Time = time.Now()
        s.Mark("clock")
        ctx.SendAfter(Tick{}, time.Second) // Reschedule
    }
    return nil
}
```

### Cancellable Timers

```go
var autoSaveTimer *live.Timer

func (p *EditorPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "edit" {
        // Cancel existing timer
        if autoSaveTimer != nil {
            autoSaveTimer.Cancel()
        }

        // Schedule new auto-save
        autoSaveTimer = ctx.SendAfter(AutoSave{}, 5*time.Second)

        s.State.Content = e.Form.Get("content")
        s.State.Dirty = true
        s.Mark("status")
    }
    return nil
}
```

## Form Patterns

### Form Validation

```go
func (p *SignupPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "submit" {
        // Clear previous errors
        s.State.Errors = make(map[string]string)

        email := e.Form.Get("email")
        password := e.Form.Get("password")

        // Validate
        if email == "" {
            s.State.Errors["email"] = "Email is required"
        } else if !isValidEmail(email) {
            s.State.Errors["email"] = "Invalid email format"
        }

        if len(password) < 8 {
            s.State.Errors["password"] = "Password must be at least 8 characters"
        }

        // If errors, show them
        if len(s.State.Errors) > 0 {
            s.Mark("form")
            return nil
        }

        // Save and redirect
        createUser(email, password)
        s.Push(live.Redirect{To: "/welcome"})
    }
    return nil
}
```

Template:

```html
<form data-lv-submit="submit" data-lv-prevent>
    <div>
        <label>Email</label>
        <input name="email" type="email" value="{{.Data.State.Form.Email}}"
               class="{{if .Data.State.Errors.email}}border-red-500{{end}}">
        {{if .Data.State.Errors.email}}
            <p class="text-red-500">{{.Data.State.Errors.email}}</p>
        {{end}}
    </div>

    <div>
        <label>Password</label>
        <input name="password" type="password"
               class="{{if .Data.State.Errors.password}}border-red-500{{end}}">
        {{if .Data.State.Errors.password}}
            <p class="text-red-500">{{.Data.State.Errors.password}}</p>
        {{end}}
    </div>

    <button type="submit">Sign Up</button>
</form>
```

### Live Search

```go
type SearchState struct {
    Query   string
    Results []SearchResult
}

func (p *SearchPage) Handle(ctx *live.Ctx, s *live.Session[SearchState], e live.Event) error {
    if e.Name == "search" {
        s.State.Query = e.Form.Get("query")

        if len(s.State.Query) >= 2 {
            s.State.Results = search(s.State.Query)
        } else {
            s.State.Results = nil
        }

        s.Mark("results")
    }
    return nil
}
```

Template:

```html
<div id="lv-root" data-lv>
    <input name="query"
           type="search"
           value="{{.Data.State.Query}}"
           data-lv-change="search"
           data-lv-debounce="250"
           placeholder="Search...">

    <div id="results">
        {{if .Data.State.Results}}
            {{range .Data.State.Results}}
                <div class="result">
                    <h3>{{.Title}}</h3>
                    <p>{{.Snippet}}</p>
                </div>
            {{end}}
        {{else if .Data.State.Query}}
            <p>No results found</p>
        {{end}}
    </div>
</div>
```

### Multi-step Forms

```go
type WizardState struct {
    Step     int
    Step1    Step1Data
    Step2    Step2Data
    Step3    Step3Data
    Complete bool
}

func (p *WizardPage) Handle(ctx *live.Ctx, s *live.Session[WizardState], e live.Event) error {
    switch e.Name {
    case "next":
        // Validate current step
        if !validateStep(s.State.Step, e.Form) {
            s.Mark("errors")
            return nil
        }

        // Save step data
        saveStepData(s, e.Form)

        s.State.Step++
        s.MarkAll()

    case "back":
        s.State.Step--
        s.MarkAll()

    case "submit":
        if s.State.Step == 3 {
            completeWizard(s.State)
            s.State.Complete = true
            s.MarkAll()
        }
    }
    return nil
}
```

## Template Helpers

### Built-in Helpers

```html
<!-- Event attributes -->
<button {{lvClick "save"}}>Save</button>
<button {{lvClick "delete" (lvVal "id" .ID)}}>Delete</button>
<form {{lvSubmit "create"}}>

<!-- Loading states -->
<button {{lvClick "save"}} {{lvLoading "opacity-50"}}>
    Save
</button>
```

Helper functions:

```go
// Template functions provided by live package
func lvClick(event string, values ...any) template.HTMLAttr
func lvSubmit(event string) template.HTMLAttr
func lvChange(event string) template.HTMLAttr
func lvKeydown(event string, key string) template.HTMLAttr
func lvVal(key string, value any) map[string]any
func lvLoading(class string) template.HTMLAttr
func lvDebounce(ms int) template.HTMLAttr
```

Output:

```html
<button data-lv-click="save">Save</button>
<button data-lv-click="delete" data-lv-value-id="123">Delete</button>
<form data-lv-submit="create">
```

### Custom Helpers

Register custom template helpers:

```go
lv := live.New(live.Options{
    View: view.Must(view.New(view.Options{
        Funcs: template.FuncMap{
            "formatTime": func(t time.Time) string {
                return t.Format("3:04 PM")
            },
            "statusClass": func(status string) string {
                switch status {
                case "active": return "bg-green-100 text-green-800"
                case "pending": return "bg-yellow-100 text-yellow-800"
                default: return "bg-gray-100 text-gray-800"
                }
            },
        },
    })),
})
```

## Common Patterns

### Optimistic Updates

Show immediate feedback while processing:

```go
func (p *TodoPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "toggle" {
        id := e.GetInt("id")

        // Optimistic update - toggle immediately
        for i := range s.State.Todos {
            if s.State.Todos[i].ID == id {
                s.State.Todos[i].Done = !s.State.Todos[i].Done
                break
            }
        }
        s.Mark("todos")

        // Save in background (could fail)
        go func() {
            if err := saveTodo(id); err != nil {
                // Revert on error via pubsub
                lv.PubSub().Publish("session:"+ctx.SessionID, RevertToggle{ID: id})
            }
        }()
    }
    return nil
}
```

### Infinite Scroll

```html
<div id="items">
    {{range .Data.State.Items}}
        <div class="item">{{.Title}}</div>
    {{end}}
</div>

{{if .Data.State.HasMore}}
    <div data-lv-viewport="load-more" data-lv-viewport-margin="200px">
        Loading more...
    </div>
{{end}}
```

```go
func (p *ListPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "load-more" {
        more := loadMore(s.State.Page)
        s.State.Items = append(s.State.Items, more.Items...)
        s.State.Page++
        s.State.HasMore = more.HasMore
        s.Mark("items")
    }
    return nil
}
```

### Confirmation Dialogs

```go
type DeleteState struct {
    Items          []Item
    ConfirmDelete  *int // Item ID to confirm, nil if no dialog
}

func (p *ListPage) Handle(ctx *live.Ctx, s *live.Session[DeleteState], e live.Event) error {
    switch e.Name {
    case "request-delete":
        id := e.GetInt("id")
        s.State.ConfirmDelete = &id
        s.Mark("dialog")

    case "confirm-delete":
        if s.State.ConfirmDelete != nil {
            deleteItem(*s.State.ConfirmDelete)
            s.State.Items = removeItem(s.State.Items, *s.State.ConfirmDelete)
            s.State.ConfirmDelete = nil
            s.Mark("items", "dialog")
        }

    case "cancel-delete":
        s.State.ConfirmDelete = nil
        s.Mark("dialog")
    }
    return nil
}
```

### Tabs

```go
type TabsState struct {
    ActiveTab string
    TabData   map[string]any
}

func (p *TabbedPage) Handle(ctx *live.Ctx, s *live.Session[TabsState], e live.Event) error {
    if e.Name == "switch-tab" {
        tab := e.Get("tab")
        s.State.ActiveTab = tab

        // Load tab data if not cached
        if s.State.TabData[tab] == nil {
            s.State.TabData[tab] = loadTabData(tab)
        }

        s.Mark("tabs", "content")
    }
    return nil
}
```

## Testing Live Pages

### Unit Testing

```go
func TestCounterPage(t *testing.T) {
    page := &CounterPage{}

    // Create test session
    session := live.NewTestSession[CounterState]()
    ctx := live.NewTestCtx()

    // Test Mount
    err := page.Mount(ctx, session)
    require.NoError(t, err)
    assert.Equal(t, 0, session.State.Count)

    // Test Handle - increment
    err = page.Handle(ctx, session, live.Event{Name: "inc"})
    require.NoError(t, err)
    assert.Equal(t, 1, session.State.Count)
    assert.True(t, session.IsDirty())

    // Test Handle - decrement
    err = page.Handle(ctx, session, live.Event{Name: "dec"})
    require.NoError(t, err)
    assert.Equal(t, 0, session.State.Count)
}
```

### Integration Testing

```go
func TestCounterIntegration(t *testing.T) {
    // Setup
    eng := view.Must(view.New(view.Options{Dir: "testdata/views"}))
    lv := live.New(live.Options{View: eng})
    app := mizu.New()
    lv.Mount(app)
    app.GET("/counter", lv.Page("/counter", &CounterPage{}))

    // Create test client
    client := live.NewTestClient(t, app)
    defer client.Close()

    // Connect to page
    conn := client.Connect("/counter")

    // Verify initial render
    assert.Contains(t, conn.HTML(), "Counter: 0")

    // Send event
    conn.Click("inc")
    conn.Wait()

    // Verify update
    assert.Contains(t, conn.HTML(), "Counter: 1")
}
```

## Error Handling

### User-facing Errors

```go
func (p *FormPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "save" {
        if err := save(e.Form); err != nil {
            // Show error to user
            s.Flash.AddError("Failed to save: " + err.Error())
            s.Mark("flash")
            return nil // Not a system error
        }

        s.Flash.AddSuccess("Saved!")
        s.Mark("flash")
    }
    return nil
}
```

### System Errors

```go
func (p *DataPage) Handle(ctx *live.Ctx, s *live.Session[State], e live.Event) error {
    if e.Name == "load" {
        data, err := loadFromDatabase()
        if err != nil {
            // Return error for logging/monitoring
            return fmt.Errorf("database error: %w", err)
        }
        s.State.Data = data
        s.Mark("data")
    }
    return nil
}
```

Handle system errors globally:

```go
lv := live.New(live.Options{
    OnError: func(ctx *live.Ctx, err error) {
        log.Error("live error", "session", ctx.SessionID, "error", err)

        // Optionally notify user
        // ctx.SendInfo(ErrorNotice{Message: "An error occurred"})
    },
})
```

## Performance Tips

1. **Use Regions**: Mark specific regions dirty instead of `MarkAll()`
2. **Debounce Input**: Use `data-lv-debounce` on text inputs
3. **Lazy Load**: Load heavy data on demand, not in Mount
4. **Paginate**: Don't load thousands of items at once
5. **Cache in State**: Store computed values instead of recalculating

## CLI Integration

### Scaffold a Live Page

```bash
mizu generate live counter
```

Creates:
- `pages/counter.go` - Page implementation
- `views/pages/counter/index.html` - Template
- `views/partials/counter/...` - Region partials

### Start with Live Template

```bash
mizu new myapp --template live
```

Creates a full project with:
- Live engine setup
- Example counter page
- Chat room example
- All necessary templates
